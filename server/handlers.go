package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/status"
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

	if err := a.Validate(); err != nil {
		errs := fmt.Sprintf("error validating artifact: %s", err.Error())
		logger.Println(errs)
		handleError(w, errs, http.StatusBadRequest)
		return
	}

	logger.Println(fmt.Sprintf("Received new artifact published event for repository %s", a.RepoName))

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

	uc := status.UpdateCondition{
		Status:       config.StatusUnknown,
		RepoName:     a.RepoName,
		ManifestName: a.ManifestName,
	}

	err = redisClient.WriteCondition(r.Context(), uc)
	if err != nil {
		handleError(w, "Error setting deploy status", http.StatusBadRequest)
	}

	fmt.Fprintf(w, `{"request":"success"}`)
}

func handleDeployStatus(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Println("error reading request body:", err)
		handleError(w, "error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var p config.DeployStatusPayload
	err = json.Unmarshal(data, &p)
	if err != nil {
		handleError(w, "Error parsing request", http.StatusBadRequest)
		return
	}

	if err = p.Validate(); err != nil {
		errs := fmt.Sprintf("error validating payload: %s", err.Error())
		logger.Println(errs)
		handleError(w, errs, http.StatusBadRequest)
		return
	}

	c, err := redisClient.ReadCondition(r.Context(), p.RepoName, p.ManifestName)

	if err != nil {
		logger.Println(fmt.Sprintf("Error getting deploy status from redis: %s. RepoName: %s, ManifestName: %s", err, p.RepoName, p.ManifestName))
		if err.Error() == "redis: nil" {
			handleError(w, fmt.Sprintf("Could not find deploy status for RepoName: %s, ManifestName: %s", p.RepoName, p.ManifestName), http.StatusBadRequest)
			return
		}
		handleError(w, "Error getting deploy status", http.StatusBadRequest)
		return
	}

	var cond status.UpdateCondition
	err = json.Unmarshal([]byte(c), &cond)
	if err != nil {
		logger.Println(fmt.Sprintf("unmarshalling update condition from redis: %s", err))
		handleError(w, "Error getting deploy status", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, fmt.Sprintf(`{"request":"success","updateCondition":%s}`, c))
}

func handleServicePost(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Println("error reading request body:", err)
		handleError(w, "error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var payload config.ServiceActionPayload
	err = json.Unmarshal(data, &payload)
	if err != nil {
		handleError(w, "Error parsing request", http.StatusInternalServerError)
		return
	}

	if payload.Validate() != nil {
		err := fmt.Sprintf("error validating payload: %s", payload.Validate().Error())
		logger.Println(err)
		handleError(w, err, http.StatusBadRequest)
		return
	}

	json, err := json.Marshal(payload)
	if err != nil {
		logger.Println(err)
		handleError(w, "error occurred marshalling json", http.StatusInternalServerError)
		return
	}

	err = messageClient.Publish(config.ServiceActionTopic, string(json))
	if err != nil {
		logger.Println(err)
		handleError(w, "Error publishing event", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, fmt.Sprintf(`{"request":"success"}`))
}

func handleError(w http.ResponseWriter, err string, statusCode int) {
	http.Error(w, fmt.Sprintf(`{"request":"error","error":"%s"}`, err), statusCode)
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, fmt.Sprintf(`{"version":"%s"}`, version))
}
