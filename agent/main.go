package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/mqtt"
)

var logger = log.New(os.Stdout, "[Pi-App-Updater-Agent] ", log.LstdFlags)

func main() {
	// todo: support multiple repos and packages
	repoName := flag.String("repo-name", "", "Name of the Github repo including the owner")
	packageName := flag.String("package-name", "", "Package name to install")
	flag.Parse()

	if *repoName == "" {
		logger.Fatalln("repo-name is required")
	}
	if *packageName == "" {
		logger.Fatalln("package-name is required")
	}

	user := os.Getenv("CLOUDMQTT_AGENT_USER")
	password := os.Getenv("CLOUDMQTT_AGENT_PASSWORD")
	mqttURL := os.Getenv("CLOUDMQTT_URL")
	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", user, password, mqttURL)

	cfg := config.Config{
		RepoName:    *repoName,
		PackageName: *packageName,
	}

	client := mqtt.NewMQTTClient(mqttAddr, *logger)
	client.Subscribe(config.RepoPushTopic, func(message string) {
		var artifact config.Artifact
		err := json.Unmarshal([]byte(message), &artifact)
		if err != nil {
			logger.Println(fmt.Sprintf("unmarshalling payload from topic %s: %s", config.RepoPushTopic, err))
		} else {
			if artifact.Repository == cfg.RepoName {
				handleRepoUpdate(artifact)
			}
		}
	})

	go forever()
	select {} // block forever
}

func handleRepoUpdate(artifact config.Artifact) {
	logger.Println(fmt.Sprintf("Received message on topic %s:", config.RepoPushTopic))

	postBody, _ := json.Marshal(map[string]string{
		"name":  "Toby",
		"email": "Toby@example.com",
	})

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, "https://pi-app-updater.herokuapp.com/templates/render", bytes.NewBuffer(postBody))
	if err != nil {
		logger.Println(err)
		return
	}
	req.Header.Add("api-key", os.Getenv("PI_APP_UPDATER_API_KEY"))
	resp, err := client.Do(req)
	if err != nil {
		logger.Println(err)
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(data))
}

func forever() {
	for {
		time.Sleep(5 * time.Minute)
	}
}
