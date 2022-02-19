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
		return
	}
	defer r.Body.Close()

	logger.Println(string(data))
}

func renderTemplates(a config.Artifact) (config.ConfigFiles, error) {
	c := config.ConfigFiles{}
	// download manifest from repo, render templates
	// where can I get heroku api key? do we really want to send this to agents??
	// manifest.GetManifest()
	m := manifest.Manifest{
		Name: "abc",
		Heroku: manifest.Heroku{
			App: "abc",
			Env: []string{"HELLO", "WORLD"},
		},
		Systemd: manifest.SystemdConfig{
			Unit: manifest.SystemdUnit{
				Description: "this is description",
			},
		},
	}
	serviceUnit, err := file.EvalServiceTemplate(m, "abc")
	if err != nil {
		return config.ConfigFiles{}, err
	}

	c.Systemd = file.ToJSONCompliant(serviceUnit)

	runScript, err := file.EvalRunScriptTemplate(m)
	if err != nil {
		return config.ConfigFiles{}, err
	}
	c.RunScript = file.ToJSONCompliant(runScript)

	return c, nil
}
