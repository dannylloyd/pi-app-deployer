package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
)

func handleRepoPush(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Println("error reading request body:", err)
		handleError(w, "error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var a config.Artifact
	err = json.Unmarshal(data, &a)
	if err != nil {
		handleError(w, "Error parsing request", http.StatusInternalServerError)
		return
	}

	if a.Validate() != nil {
		errs := fmt.Sprintf("error validating artifact: %s", a.Validate().Error())
		logger.Println(errs)
		handleError(w, errs, http.StatusBadRequest)
		return
	}

	logger.Println(fmt.Sprintf("Received new artifact published event for repository %s", a.Repository))

	json, err := json.Marshal(a)
	if err != nil {
		logger.Println(err)
		handleError(w, "error occurred marshalling json", http.StatusInternalServerError)
		return
	}
	err = messageClient.Publish(config.RepoPushTopic, string(json))
	if err != nil {
		logger.Println(err)
		handleError(w, "Error publishing event", http.StatusInternalServerError)
		return
	}

	key := fmt.Sprintf("%s/%s", a.Repository, a.ManifestName)
	err = redisClient.WriteCondition(r.Context(), key, config.StatusInProgress)
	if err != nil {
		handleError(w, "Error setting deploy status", http.StatusBadRequest)
	}

	fmt.Fprintf(w, `{"status":"success"}`)
}

func handleDeployStatus(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Println("error reading request body:", err)
		handleError(w, "error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var a config.Artifact
	err = json.Unmarshal(data, &a)
	if err != nil {
		handleError(w, "Error parsing request", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("%s/%s", a.Repository, a.ManifestName)
	c, err := redisClient.ReadCondition(r.Context(), key)
	if err != nil {
		handleError(w, "Error getting deploy status", http.StatusBadRequest)
	}

	if c == "" {
		c = config.StatusUnknown
	}
	fmt.Fprintf(w, fmt.Sprintf(`{"status":"%s"}`, c))
}

func handleError(w http.ResponseWriter, err string, statusCode int) {
	http.Error(w, fmt.Sprintf(`{"status":"error","error","%s"}`, err), statusCode)
}
