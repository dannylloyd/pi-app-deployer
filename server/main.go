package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/andrewmarklloyd/pi-app-updater/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/file"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/mqtt"
	gmux "github.com/gorilla/mux"

	"github.com/google/go-github/v42/github"
)

var backoffSchedule = []time.Duration{
	10 * time.Second,
	15 * time.Second,
	20 * time.Second,
	30 * time.Second,
	60 * time.Second,
}

var logger = log.New(os.Stdout, "[Pi-App-Updater-Server] ", log.LstdFlags)

var messageClient mqtt.MqttClient

func handleWebhook(w http.ResponseWriter, r *http.Request) {
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

func getDownloadURLWithRetries(artifact config.Artifact) (string, error) {
	var err error
	var url string
	for _, backoff := range backoffSchedule {
		url, err = getDownloadURL(artifact)
		if url != "" {
			return url, nil
		}

		logger.Println(fmt.Sprintf("Retrying in %v", backoff))
		time.Sleep(backoff)
	}
	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("an unexpected event occurred, no url found and no error returned")
}

func getDownloadURL(artifact config.Artifact) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/actions/artifacts", artifact.Repository), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var artifacts github.ArtifactList
	err = json.Unmarshal(body, &artifacts)
	if err != nil {
		return "", err
	}

	for _, a := range artifacts.Artifacts {
		if artifact.Name == a.GetName() {
			return a.GetArchiveDownloadURL(), nil
		}
	}

	return "", fmt.Errorf("no artifact found for %s", artifact.Name)
}

func main() {
	srvAddr := fmt.Sprintf("0.0.0.0:%s", os.Getenv("PORT"))

	// TODO: reusing another app's mqtt instance to save cost. Once viable MVP finished I can provision a dedicated instance
	// TODO: read/write user is fine for this app, but clients will need read only
	user := os.Getenv("CLOUDMQTT_USER")
	pw := os.Getenv("CLOUDMQTT_PASSWORD")
	url := os.Getenv("CLOUDMQTT_URL")
	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", user, pw, url)

	messageClient = mqtt.NewMQTTClient(mqttAddr, *logger)

	router := gmux.NewRouter().StrictSlash(true)
	router.Handle("/push", requireLogin(http.HandlerFunc(handleWebhook))).Methods("POST")

	srv := &http.Server{
		Handler: router,
		Addr:    srvAddr,
	}

	log.Println("server started on ", srvAddr)
	logger.Fatal(srv.ListenAndServe())
}

func requireLogin(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		if !isAuthenticated(req) {
			logger.Println(fmt.Sprintf("Unauthenticated request, host: %s, headers: %s", req.Host, req.Header))
			http.Error(w, "", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, req)
	}
	return http.HandlerFunc(fn)
}

func isAuthenticated(req *http.Request) bool {
	allowedApiKey := os.Getenv("PI_APP_UPDATER_API_KEY")
	apiKey := req.Header.Get("api-key")
	if apiKey == "" {
		return false
	}
	if apiKey != allowedApiKey {
		return false
	}
	return true
}
