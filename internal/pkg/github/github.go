package github

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/google/go-github/v42/github"
)

var backoffSchedule = []time.Duration{
	10 * time.Second,
	15 * time.Second,
	20 * time.Second,
	30 * time.Second,
	60 * time.Second,
}

func GetDownloadURLWithRetries(artifact config.Artifact, latest bool) (string, error) {
	var err error
	var url string
	for _, backoff := range backoffSchedule {
		url, err = getDownloadURL(artifact, latest)
		if url != "" {
			return url, nil
		}

		time.Sleep(backoff)
	}
	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("an unexpected event occurred, no url found and no error returned")
}

func getDownloadURL(artifact config.Artifact, latest bool) (string, error) {
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
