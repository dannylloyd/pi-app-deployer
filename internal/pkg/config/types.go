package config

import (
	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/manifest"
)

const (
	RepoPushTopic = "repo/push"
)

type Config struct {
	RepoName     string
	ManifestName string
}

type Artifact struct {
	SHA                string `json:"sha"`
	Repository         string `json:"repository"`
	Name               string `json:"name"`
	ArchiveDownloadURL string `json:"download_url"`
	ManifestName       string `json:"manifest_name"`
}

type RenderTemplatesPayload struct {
	Config   Config            `json:"config"`
	Manifest manifest.Manifest `json:"manifest"`
}
