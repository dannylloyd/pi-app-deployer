package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/file"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/github"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/mqtt"
)

var logger = log.New(os.Stdout, "[Pi-App-Updater-Agent] ", log.LstdFlags)

func main() {
	// todo: support multiple repos and packages
	repoName := flag.String("repo-name", "", "Name of the Github repo including the owner")
	packageName := flag.String("package-name", "", "Package name to install or update")
	install := flag.Bool("install", false, "First time install of the application")
	flag.Parse()

	var testMode bool
	if os.Getenv("TEST_MODE") == "true" {
		testMode = true
		logger.Println("*** Running in test mode ***")
	}

	if *repoName == "" {
		logger.Fatalln("repo-name is required")
	}
	if *packageName == "" {
		logger.Fatalln("package-name is required")
	}

	cfg := config.Config{
		RepoName:    *repoName,
		PackageName: *packageName,
	}

	ghApiToken := os.Getenv("GH_API_TOKEN")
	if ghApiToken == "" {
		logger.Fatalln("GH_API_TOKEN environment variable is required")
	}

	herokuAPIKey := os.Getenv("HEROKU_API_KEY")
	if herokuAPIKey == "" {
		logger.Fatalln("HEROKU_API_TOKEN environment variable is required")
	}

	user := os.Getenv("CLOUDMQTT_AGENT_USER")
	password := os.Getenv("CLOUDMQTT_AGENT_PASSWORD")
	mqttURL := os.Getenv("CLOUDMQTT_URL")
	if user == "" || password == "" || mqttURL == "" {
		logger.Fatalln("CLOUDMQTT_AGENT_USER, CLOUDMQTT_AGENT_PASSWORD, and CLOUDMQTT_URL environment variables are required")
	}
	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", user, password, mqttURL)
	client := mqtt.NewMQTTClient(mqttAddr, *logger)

	vTool := file.NewVersionTool(testMode, *packageName)
	agent := newAgent(cfg, client, ghApiToken, herokuAPIKey, vTool, testMode)

	if *install {
		logger.Println("Installing application")
		installed, version, err := agent.VersionTool.AppInstalled()
		if err != nil {
			logger.Fatalln(fmt.Errorf("checking if app is installed already: %s", err))
		}
		if installed {
			logger.Fatalln(fmt.Sprintf("App already installed at version '%s', remove '--install' flag to check for updates", version))
		}

		c := config.Artifact{
			Repository: cfg.RepoName,
		}
		url, err := github.GetDownloadURLWithRetries(c, true)
		if err != nil {
			logger.Fatalln(fmt.Errorf("getting download url for latest release: %s", err))
		}
		logger.Println(url)

		// latest, err := ghClient.GetLatestVersion(cfg)
		// if err != nil {
		// 	log.Fatalln(fmt.Sprintf("error getting latest version from github: %s", err))
		// }

		// err = installRelease(cfg, latest.AssetDownloadURL, sdTool)
		// if err != nil {
		// 	log.Fatalln(fmt.Errorf("error installing app: %s", err))
		// }
		// err = vTool.WriteCurrentVersion(latest.Version)
		// if err != nil {
		// 	log.Fatalln(fmt.Errorf("writing latest version to file: %s", err))
		// }

		// agent.VersionTool.WriteCurrentVersion("hello-world")
		logger.Println("Successfully installed app")
		os.Exit(0)
	}

	agent.MqttClient.Connect()

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

	go forever()
	select {} // block forever
}

func forever() {
	for {
		time.Sleep(5 * time.Minute)
	}
}
