package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/github"
)

func handleRepoPush(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading request body: err=%s\n", err)
		return
	}
	defer r.Body.Close()

	var a config.Artifact
	err = json.Unmarshal(data, &a)
	if err != nil {
		http.Error(w, "Error parsing request", http.StatusBadRequest)
		return
	}

	if a.Repository == "" || a.Name == "" || a.SHA == "" {
		// todo: better error reporting to user
		logger.Println("empty field(s) found in artifact:", a)
		http.Error(w, "Error parsing request", http.StatusBadRequest)
		return
	}

	logger.Println(fmt.Sprintf("Received new artifact published event for repository %s", a.Repository))

	url, err := github.GetDownloadURLWithRetries(a, false)
	if err != nil {
		logger.Println(err)
		http.Error(w, "Error parsing request", http.StatusBadRequest)
		return
	}
	a.ArchiveDownloadURL = url

	json, err := json.Marshal(a)

	if err != nil {
		logger.Println(err)
		http.Error(w, "an error occurred", http.StatusInternalServerError)
		return
	}
	err = messageClient.Publish(config.RepoPushTopic, string(json))
	if err != nil {
		logger.Println(err)
		http.Error(w, "Error publishing event", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "{\"status\":\"success\"}")
}
