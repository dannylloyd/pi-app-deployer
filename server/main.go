package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/mqtt"
	gmux "github.com/gorilla/mux"
)

var logger = log.New(os.Stdout, "[Pi-App-Updater-Server] ", log.LstdFlags)

var messageClient mqtt.MqttClient

func main() {
	srvAddr := fmt.Sprintf("0.0.0.0:%s", os.Getenv("PORT"))

	// TODO: reusing another app's mqtt instance to save cost. Once viable MVP finished I can provision a dedicated instance
	// TODO: read/write user is fine for this app, but clients will need read only
	user := os.Getenv("CLOUDMQTT_USER")
	pw := os.Getenv("CLOUDMQTT_PASSWORD")
	url := os.Getenv("CLOUDMQTT_URL")
	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", user, pw, url)

	messageClient = mqtt.NewMQTTClient(mqttAddr, *logger)
	messageClient.Connect()

	router := gmux.NewRouter().StrictSlash(true)
	router.Handle("/push", requireLogin(http.HandlerFunc(handleRepoPush))).Methods("POST")
	router.Handle("/templates/render", requireLogin(http.HandlerFunc(handleTemplatesRender))).Methods("POST")

	srv := &http.Server{
		Handler: router,
		Addr:    srvAddr,
	}

	log.Println("server started on ", srvAddr)
	logger.Fatal(srv.ListenAndServe())
}

func requireLogin(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		if !isAuthenticated(req) {
			logger.Println(fmt.Sprintf("Unauthenticated request, host: %s, headers: %s", req.Host, req.Header))
			http.Error(w, "", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, req)
	}
	return http.HandlerFunc(fn)
}

func isAuthenticated(req *http.Request) bool {
	allowedApiKey := os.Getenv("PI_APP_UPDATER_API_KEY")
	apiKey := req.Header.Get("api-key")
	if apiKey == "" {
		return false
	}
	if apiKey != allowedApiKey {
		return false
	}
	return true
}
