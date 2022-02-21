package config

import (
	"github.com/andrewmarklloyd/pi-app-updater/api/v1/manifest"
)

const (
	RepoPushTopic = "repo/push"
)

type Config struct {
	RepoName    string
	PackageName string
}

type Artifact struct {
	SHA                string `json:"sha"`
	Repository         string `json:"repository"`
	Name               string `json:"name"`
	ArchiveDownloadURL string `json:"download_url"`
}

type ConfigFiles struct {
	Systemd      string `json:"systemd"`
	RunScript    string `json:"runScript"`
	PiAppUpdater string `json:"piAppUpdater"`
}

type RenderTemplatesPayload struct {
	Config   Config            `json:"config"`
	Manifest manifest.Manifest `json:"manifest"`
}
