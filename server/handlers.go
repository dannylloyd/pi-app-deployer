package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
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

	if a.Validate() != nil {
		errs := fmt.Sprintf("error validating artifact: %s", a.Validate().Error())
		logger.Println(errs)
		http.Error(w, errs, http.StatusBadRequest)
		return
	}

	logger.Println(fmt.Sprintf("Received new artifact published event for repository %s", a.Repository))

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

func handleDeployStatus(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading request body: err=%s\n", err)
		return
	}
	defer r.Body.Close()

	logger.Println(string(data))
	fmt.Fprintf(w, "{\"status\":\"IN_PROGRESS\"}")
}
