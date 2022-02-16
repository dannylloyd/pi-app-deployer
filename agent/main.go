package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/andrewmarklloyd/pi-app-updater-server/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-updater-server/internal/pkg/mqtt"
)

const pushTopic = "repo/push"

var logger = log.New(os.Stdout, "[Pi-App-Updater-Server] ", log.LstdFlags)

func main() {
	repoName := flag.String("repo-name", "", "Name of the Github repo including the owner")
	packageName := flag.String("package-name", "", "Package name to install")
	flag.Parse()

	user := os.Getenv("CLOUDMQTT_AGENT_USER")
	password := os.Getenv("CLOUDMQTT_AGENT_PASSWORD")
	mqttURL := os.Getenv("CLOUDMQTT_URL")
	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", user, password, mqttURL)

	cfg := config.Config{
		RepoName:    *repoName,
		PackageName: *packageName,
	}

	client := mqtt.NewMQTTClient(mqttAddr, *logger)
	client.Subscribe(pushTopic, func(message string) {
		var payload config.UpdaterPayload
		err := json.Unmarshal([]byte(message), &payload)
		if err != nil {
			logger.Println(fmt.Sprintf("unmarshalling payload from topic %s: %s", pushTopic, err))
		} else {
			if payload.Repository == cfg.RepoName {
				handleRepoUpdate(payload)
			}
		}
	})

	go forever()
	select {} // block forever
}

func handleRepoUpdate(payload config.UpdaterPayload) {
	logger.Println(fmt.Sprintf("Received message on topic %s: %s", pushTopic, payload))
}

func forever() {
	for {
		time.Sleep(5 * time.Minute)
	}
}
