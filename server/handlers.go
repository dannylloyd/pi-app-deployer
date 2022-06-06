package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

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

	logger.Println(fmt.Sprintf("Received new artifact published event for repository %s, manifest %s, SHA %s", a.RepoName, a.ManifestName, a.SHA))

	j, err := json.Marshal(a)
	if err != nil {
		logger.Println(err)
		handleError(w, "error occurred marshalling json", http.StatusInternalServerError)
		return
	}

	err = redisClient.DeleteConditions(r.Context(), a.RepoName, a.ManifestName)
	if err != nil {
		handleError(w, "Error clearing previous deploy status", http.StatusBadRequest)
		return
	}

	err = messageClient.Publish(config.RepoPushTopic, string(j))
	if err != nil {
		logger.Println(err)
		handleError(w, "Error publishing event", http.StatusInternalServerError)
		return
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

	conditions, err := redisClient.ReadConditions(r.Context(), p.RepoName, p.ManifestName)

	if err != nil {
		logger.Println(fmt.Sprintf("Error getting deploy status from redis: %s. RepoName: %s, ManifestName: %s", err, p.RepoName, p.ManifestName))
		if err.Error() == "redis: nil" {
			handleError(w, fmt.Sprintf("Could not find deploy status for RepoName: %s, ManifestName: %s", p.RepoName, p.ManifestName), http.StatusBadRequest)
			return
		}
		handleError(w, "Error getting deploy status", http.StatusBadRequest)
		return
	}

	agents, err := redisClient.ReadAgentInventory(r.Context(), p.RepoName, p.ManifestName)
	if err != nil {
		handleError(w, "error listing agents configured to update app", http.StatusBadRequest)
		return
	}

	if len(agents) == 0 {
		logger.Println("length of agents is 0 indicating a request for deploy status occurred but no hosts are configured for this app")
		handleError(w, "no agents are configured for this app", http.StatusBadRequest)
		return
	}

	successfulHosts := map[string]status.UpdateCondition{}
	unsuccessfulHosts := map[string]status.UpdateCondition{}
	now := time.Now()
	for host, timestamp := range agents {
		cond, ok := conditions[host]
		if !ok {
			unsuccessfulHosts[host] = status.UpdateCondition{
				Host:   host,
				Status: config.StatusUnknown,
				Error:  "agent not reporting health check for app",
			}
			continue
		}

		diff := now.Sub(timestamp)
		if diff.Minutes() > 5 {
			unsuccessfulHosts[host] = status.UpdateCondition{
				Host:   host,
				Status: config.StatusUnknown,
				Error:  "agent health check for app greater than 5 minutes and considered stale",
			}
			continue
		}

		if cond.Status != config.StatusSuccess {
			unsuccessfulHosts[host] = cond
			continue
		}

		successfulHosts[host] = cond
	}

	successJson, err := json.Marshal(successfulHosts)
	if err != nil {
		handleError(w, "Error marshalling successful hosts", http.StatusBadRequest)
		return
	}

	unsuccessJson, err := json.Marshal(unsuccessfulHosts)
	if err != nil {
		handleError(w, "Error marshalling unsuccessful hosts", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, fmt.Sprintf(`{"request":"success","successfulHosts":%s,"unsuccessfulHosts":%s}`, successJson, unsuccessJson))
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
