package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/file"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/mqtt"
)

var logger = log.New(os.Stdout, "[pi-app-deployer-Agent] ", log.LstdFlags)

func main() {
	// todo: support multiple repos and packages
	repoName := flag.String("repo-name", "", "Name of the Github repo including the owner")
	manifestName := flag.String("manifest-name", "", "Name of the pi-app-deployer manifest")
	install := flag.Bool("install", false, "First time install of the application")
	logForwarding := flag.Bool("log-forwarding", false, "Send application logs to server")
	flag.Parse()

	if *repoName == "" {
		logger.Fatalln("repo-name is required")
	}

	if *manifestName == "" {
		logger.Fatalln("manifest-name is required")
	}

	homeDir := os.Getenv("HOME")

	cfg := config.Config{
		RepoName:     *repoName,
		ManifestName: *manifestName,
		HomeDir:      homeDir,
	}

	ghApiToken := os.Getenv("GH_API_TOKEN")
	if ghApiToken == "" {
		logger.Fatalln("GH_API_TOKEN environment variable is required")
	}

	herokuAPIKey := os.Getenv("HEROKU_API_KEY")
	if herokuAPIKey == "" {
		logger.Fatalln("HEROKU_API_TOKEN environment variable is required")
	}

	serverApiKey := os.Getenv("PI_APP_DEPLOYER_API_KEY")
	if serverApiKey == "" {
		logger.Fatalln("PI_APP_DEPLOYER_API_KEY environment variable is required")
	}

	user := os.Getenv("CLOUDMQTT_AGENT_USER")
	password := os.Getenv("CLOUDMQTT_AGENT_PASSWORD")
	mqttURL := os.Getenv("CLOUDMQTT_URL")
	if user == "" || password == "" || mqttURL == "" {
		logger.Fatalln("CLOUDMQTT_AGENT_USER, CLOUDMQTT_AGENT_PASSWORD, and CLOUDMQTT_URL environment variables are required")
	}
	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", user, password, mqttURL)
	client := mqtt.NewMQTTClient(mqttAddr, *logger)

	agent := newAgent(cfg, client, ghApiToken, herokuAPIKey, serverApiKey)

	if *install {
		enabled, err := file.SystemdUnitEnabled(cfg.ManifestName)
		if err != nil {
			logger.Fatalln("error checking if app is installed already: ", err)
		}

		if enabled {
			logger.Fatalln("App already installed, remove '--install' flag to check for updates")
		}

		logger.Println("Installing application")
		a := config.Artifact{
			Repository:   cfg.RepoName,
			ManifestName: cfg.ManifestName,
		}
		err = agent.handleInstall(a)
		if err != nil {
			logger.Fatalln(fmt.Errorf("failed installation: %s", err))
		}
		logger.Println("Successfully installed app")
		os.Exit(0)
	}

	err := agent.MqttClient.Connect()
	if err != nil {
		logger.Fatalln("connecting to mqtt: ", err)
	}

	agent.MqttClient.Subscribe(config.RepoPushTopic, func(message string) {
		var artifact config.Artifact
		err := json.Unmarshal([]byte(message), &artifact)
		if err != nil {
			logger.Println(fmt.Sprintf("unmarshalling payload from topic %s: %s", config.RepoPushTopic, err))
		} else {
			if artifact.Repository == cfg.RepoName {
				err := agent.handleRepoUpdate(artifact)
				if err != nil {
					logger.Println(err)
				}
			}
		}
	})

	if *logForwarding {
		logger.Println(fmt.Sprintf("Log forwarding is enabled for %s", cfg.ManifestName))
		agent.startLogForwarder(cfg.ManifestName, func(log string) {
			l := config.Log{
				Message: log,
				Config:  cfg,
			}
			json, err := json.Marshal(l)
			if err != nil {
				logger.Println(fmt.Sprintf("marshalling log forwarder message: %s", err))
				return
			}
			agent.MqttClient.Publish(config.LogForwarderTopic, string(json))
		})
	}

	go forever()
	select {} // block forever
}

func forever() {
	for {
		time.Sleep(5 * time.Minute)
	}
}
