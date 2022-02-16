package github

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/google/go-github/github"
)

const (
	ghReleaseURL = "https://api.github.com/repos/%s/releases/latest"
)

type AppInfo struct {
	TagName string `json:"tag_name"`
}

type GithubClient struct {
	Config      config.Config
	GithubToken string
}

type Latest struct {
	Version          string
	AssetDownloadURL string
}

func NewClient(cfg config.Config) GithubClient {
	ghToken := os.Getenv("GITHUB_TOKEN")
	return GithubClient{
		Config:      cfg,
		GithubToken: ghToken,
	}
}

func (c *GithubClient) GetLatestVersion(cfg config.Config) (Latest, error) {
	var latest Latest
	req, err := http.NewRequest("GET", fmt.Sprintf(ghReleaseURL, cfg.RepoName), nil)
	if err != nil {
		return latest, err
	}
	// TODO: if no rate limiting risk exists then remove this comment
	if c.GithubToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.GithubToken))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return latest, err
	}
	var info AppInfo
	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		return latest, fmt.Errorf("parsing version from api response: %s", err)
	}
	if info.TagName == "" {
		return latest, fmt.Errorf("empty tag name from api response: %s", info)
	}
	latest.Version = info.TagName

	url, err := c.getAssetDownloadURL(info.TagName)
	if err != nil {
		return latest, err
	}
	latest.AssetDownloadURL = url
	return latest, nil
}

func (c *GithubClient) getAssetDownloadURL(latestVersion string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", c.Config.RepoName), nil)
	if err != nil {
		return "", err
	}
	if c.GithubToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.GithubToken))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting latest release: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var release github.RepositoryRelease
	err = json.Unmarshal(body, &release)

	if err != nil {
		return "", fmt.Errorf("unmarshalling json: %s", err)
	}
	for _, a := range release.Assets {
		expectedName := fmt.Sprintf("%s-%s-linux-arm.tar.gz", c.Config.PackageName, latestVersion)
		if expectedName == *a.Name {
			return *a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf(fmt.Sprintf("download URL not found, no assets matched expected name containing package name %s and version %s", c.Config.PackageName, latestVersion))
}
