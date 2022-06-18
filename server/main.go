package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	mqttC "github.com/eclipse/paho.mqtt.golang"
	"gopkg.in/yaml.v2"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/status"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/logging"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/mqtt"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/redis"
	"go.uber.org/zap"

	gmux "github.com/gorilla/mux"
)

var logger *zap.SugaredLogger
var forwarderLogger *zap.SugaredLogger

var messageClient mqtt.MqttClient
var redisClient redis.Redis

var version string

func main() {
	l, err := zap.NewProduction()
	if err != nil {
		log.Fatalln("Error creating logger:", err)
	}
	logger = l.Sugar().Named("pi-app-deployer")
	defer logger.Sync()

	forwarderLogger = l.Sugar().Named("pi-app-deployer-forwarder")
	defer forwarderLogger.Sync()

	srvAddr := fmt.Sprintf("0.0.0.0:%s", os.Getenv("PORT"))

	user := os.Getenv("CLOUDMQTT_USER")
	pw := os.Getenv("CLOUDMQTT_PASSWORD")
	url := os.Getenv("CLOUDMQTT_URL")

	if user == "" || pw == "" || url == "" {
		logger.Fatal("CLOUDMQTT_USER CLOUDMQTT_PASSWORD CLOUDMQTT_URL env vars must be set")
	}

	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", user, pw, strings.Split(url, "@")[1])

	messageClient = mqtt.NewMQTTClient(mqttAddr, func(client mqttC.Client) {
		logger.Info("Connected to MQTT server")
	}, func(client mqttC.Client, err error) {
		logger.Fatalf("Connection to MQTT server lost: %s", err)
	})
	err = messageClient.Connect()
	if err != nil {
		logger.Fatalf("connecting to mqtt: %s", err)
	}

	redisClient, err = redis.NewRedisClient(os.Getenv("REDIS_TLS_URL"))
	if err != nil {
		logger.Fatalf("creating redis client: %s", err)
	}

	fConfig := os.Getenv("FORWARDER_CONFIG")
	var logCM logging.ConfigMap
	err = yaml.Unmarshal([]byte(fConfig), &logCM)
	if err != nil {
		logger.Fatalf("unmarshalling log forwarder config %s", err)
	}

	messageClient.Subscribe(config.LogForwarderTopic, func(message string) {
		var log config.Log
		err := json.Unmarshal([]byte(message), &log)
		if err != nil {
			logger.Errorf("unmarshalling log forwarder message: %s", err)
			return
		}

		c, ok := logCM[log.Config.RepoName]
		if !ok {
			logger.Errorw("Received forwarded log but no forwarder config exists. Unable to send logs to app endpoint",
				"repoName", log.Config.RepoName,
				"manifestName", log.Config.ManifestName,
			)
			return
		}

		if err = logging.SendLogs(c, log); err != nil {
			logger.Errorw(fmt.Sprintf("forwarding app logs: %s", err),
				"repoName", log.Config.RepoName,
				"manifestName", log.Config.ManifestName,
			)
		}

	})

	messageClient.Subscribe(config.RepoPushStatusTopic, func(message string) {
		var c status.UpdateCondition
		err := json.Unmarshal([]byte(message), &c)
		if err != nil {
			logger.Errorf("unmarshalling update condition message: %s, raw message string: %s", err, message)
			return
		}

		logger.Infow("repo push status received",
			"condition", c.Status,
			"repoName", c.RepoName,
			"manifestName", c.ManifestName,
			"host", c.Host,
			"error", c.Error)

		err = redisClient.WriteCondition(context.Background(), c)
		if err != nil {
			logger.Errorf("writing condition message to redis: %s", err)
			return
		}
	})

	var inventoryTimerMap map[string]*time.Timer = make(map[string]*time.Timer)
	messageClient.Subscribe(config.AgentInventoryTopic, func(message string) {
		p := config.AgentInventoryPayload{}
		unmarshErr := json.Unmarshal([]byte(message), &p)
		if unmarshErr != nil {
			logger.Errorf("unmarshalling agent inventory payload: %s", unmarshErr)
			return
		}

		expiration := 0 * time.Minute
		if p.Transient {
			expiration = 1 * time.Minute
		}
		err = redisClient.WriteAgentInventory(context.Background(), p, expiration)
		if err != nil {
			logger.Errorf("writing agent inventory to redis: %s", err)
			return
		}

		// there can be multiple manifest/repo per host. For
		// timeout we're only interested in host, so last one wins.
		if !p.Transient {
			currentTimer := inventoryTimerMap[p.Host]
			if currentTimer != nil {
				currentTimer.Stop()
			}
			timer := time.AfterFunc(config.InventoryTickerTimeout, func() {
				logger.Errorf("Agent inventory timeout occurred for host: %s", p.Host)
			})
			inventoryTimerMap[p.Host] = timer
		}
	})

	router := gmux.NewRouter().StrictSlash(true)
	router.Handle("/push", requireLogin(http.HandlerFunc(handleRepoPush))).Methods("POST")
	router.Handle("/deploy/status", requireLogin(http.HandlerFunc(handleDeployStatus))).Methods("GET")
	router.Handle("/service", requireLogin(http.HandlerFunc(handleServicePost))).Methods("POST")
	router.Handle("/health", requireLogin(http.HandlerFunc(handleHealthCheck))).Methods("GET")

	srv := &http.Server{
		Handler: router,
		Addr:    srvAddr,
	}

	logger.Infof("server started on %s", srvAddr)
	logger.Fatalf("error running web server: %s", srv.ListenAndServe())
}

func requireLogin(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		if !isAuthenticated(req) {
			logger.Warnf("Unauthenticated request, host: %s, headers: %s", req.Host, req.Header)
			http.Error(w, `{"status":"error","error":"unauthenticated"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, req)
	}
	return http.HandlerFunc(fn)
}

func isAuthenticated(req *http.Request) bool {
	allowedApiKey := os.Getenv("PI_APP_DEPLOYER_API_KEY")
	apiKey := req.Header.Get("api-key")
	if apiKey == "" {
		return false
	}
	if apiKey != allowedApiKey {
		return false
	}
	return true
}
