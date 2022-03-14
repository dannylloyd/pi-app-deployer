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
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/heroku"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/mqtt"
)

var logger = log.New(os.Stdout, "[pi-app-deployer-Agent] ", log.LstdFlags)

func main() {
	u := os.Getenv("USER")
	if u != "root" {
		logger.Fatalln("agent must be run as root, user found was", u)
	}
	// todo: support multiple repos and packages
	repoName := flag.String("repo-name", "", "Name of the Github repo including the owner")
	manifestName := flag.String("manifest-name", "", "Name of the pi-app-deployer manifest")
	appUser := flag.String("app-user", "pi", "Name of user that will run the app service")
	homeDir := flag.String("home-dir", "/home/pi", "Name of app user's home directory")
	install := flag.Bool("install", false, "First time install of the application")
	logForwarding := flag.Bool("log-forwarding", false, "Send application logs to server")
	flag.Parse()

	if *repoName == "" {
		logger.Fatalln("repo-name is required")
	}

	if *manifestName == "" {
		logger.Fatalln("manifest-name is required")
	}

	cfg := config.Config{
		RepoName:      *repoName,
		ManifestName:  *manifestName,
		HomeDir:       *homeDir,
		AppUser:       *appUser,
		LogForwarding: *logForwarding,
	}

	herokuAPIKey := os.Getenv("HEROKU_API_KEY")
	if herokuAPIKey == "" {
		logger.Fatalln("HEROKU_API_TOKEN environment variable is required")
	}

	c := heroku.NewHerokuClient(herokuAPIKey)
	envVars, err := c.GetEnvVars()

	ghApiToken := envVars["GH_API_TOKEN"]
	if ghApiToken == "" {
		logger.Fatalln("GH_API_TOKEN environment variable not found from heroku")
	}

	serverApiKey := envVars["PI_APP_DEPLOYER_API_KEY"]
	if serverApiKey == "" {
		logger.Fatalln("PI_APP_DEPLOYER_API_KEY environment variable not found from heroku")
	}

	user := envVars["CLOUDMQTT_AGENT_USER"]
	if user == "" {
		logger.Fatalln("CLOUDMQTT_AGENT_USER environment variable not found from heroku")
	}

	password := envVars["CLOUDMQTT_AGENT_PASSWORD"]
	if password == "" {
		logger.Fatalln("CLOUDMQTT_AGENT_PASSWORD environment variable not found from heroku")
	}
	mqttURL := envVars["CLOUDMQTT_URL"]
	if mqttURL == "" {
		logger.Fatalln("CLOUDMQTT_URL environment variable not found from heroku")
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

	err = agent.MqttClient.Connect()
	if err != nil {
		logger.Fatalln("connecting to mqtt: ", err)
	}

	updateCondition := config.UpdateCondition{
		RepoName:     cfg.RepoName,
		ManifestName: cfg.ManifestName,
	}

	agent.MqttClient.Subscribe(config.RepoPushTopic, func(message string) {
		var artifact config.Artifact
		err := json.Unmarshal([]byte(message), &artifact)
		if err != nil {
			logger.Println(fmt.Sprintf("unmarshalling payload from topic %s: %s", config.RepoPushTopic, err))
			return
		}
		if artifact.Repository == cfg.RepoName && artifact.ManifestName == cfg.ManifestName {
			logger.Println(fmt.Sprintf("updating repo %s with manifest name %s", cfg.RepoName, cfg.ManifestName))
			updateCondition.Status = config.StatusInProgress
			err = agent.publishUpdateCondition(updateCondition)
			if err != nil {
				// log but don't block update from proceeding
				logger.Println(err)
			}
			err := agent.handleRepoUpdate(artifact)
			if err != nil {
				logger.Println(err)
				updateCondition.Status = config.StatusErr
				err = agent.publishUpdateCondition(updateCondition)
				if err != nil {
					logger.Println(err)
				}
				return
			}
			// TODO: should check systemctl status before sending success?
			updateCondition.Status = config.StatusSuccess
			err = agent.publishUpdateCondition(updateCondition)
			if err != nil {
				logger.Println(err)
			}
		}
	})

	agent.MqttClient.Subscribe(config.ServiceActionTopic, func(message string) {
		var payload config.ServiceActionPayload
		err := json.Unmarshal([]byte(message), &payload)
		if err != nil {
			logger.Println(fmt.Sprintf("unmarshalling payload from topic %s: %s", config.ServiceActionTopic, err))
			return
		}
		if payload.RepoName == cfg.RepoName && payload.ManifestName == cfg.ManifestName {
			logger.Println(fmt.Sprintf("Running service action %s on %s/%s", payload.Action, payload.RepoName, payload.ManifestName))
			var err error
			switch payload.Action {
			case config.ServiceActionStart:
				err = file.StartSystemdUnit(payload.ManifestName)
				break
			case config.ServiceActionStop:
				err = file.StopSystemdUnit(payload.ManifestName)
				break
			case config.ServiceActionRestart:
				err = file.RestartSystemdUnit(payload.ManifestName)
				break
			default:
				err = fmt.Errorf("Action %s is not valid", payload.Action)
				break
			}
			if err != nil {
				logger.Println(err)
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
			err = agent.MqttClient.Publish(config.LogForwarderTopic, string(json))
			if err != nil {
				logger.Println(fmt.Sprintf("error publishing log forwarding message: %s", err))
			}
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
