package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/andrewmarklloyd/pi-app-updater/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/file"
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

	url, err := getDownloadURLWithRetries(a)
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

func handleTemplatesRender(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Printf("error reading request body: err=%s\n", err)
		http.Error(w, `{"error":"reading request body"}`, http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	m := manifest.Manifest{}
	err = json.Unmarshal([]byte(data), &m)
	if err != nil {
		logger.Println(err)
		http.Error(w, `{"error":"unmarshalling json"}`, http.StatusInternalServerError)
		return
	}

	c := config.ConfigFiles{}

	serviceUnit, err := file.EvalServiceTemplate(m)
	if err != nil {
		logger.Println(err)
		http.Error(w, `{"error":"evaluating service template"}`, http.StatusInternalServerError)
		return
	}

	c.Systemd = file.ToJSONCompliant(serviceUnit)

	runScript, err := file.EvalRunScriptTemplate(m)
	if err != nil {
		logger.Println(err)
		http.Error(w, `{"error":"evaluating run script template"}`, http.StatusInternalServerError)
		return
	}
	c.RunScript = file.ToJSONCompliant(runScript)
	body, err := json.Marshal(c)
	if err != nil {
		logger.Println(err)
		http.Error(w, `{"error":"marshalling config files json"}`, http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, string(body))
}

func handleVersionMain(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Printf("error reading request body: err=%s\n", err)
		http.Error(w, `{"error":"reading request body"}`, http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// m := manifest.Manifest{}
	// err = json.Unmarshal([]byte(data), &m)
	// if err != nil {
	// 	logger.Println(err)
	// 	http.Error(w, `{"error":"unmarshalling json"}`, http.StatusInternalServerError)
	// 	return
	// }
	logger.Println(string(data))

	fmt.Fprintf(w, "{}")
}
