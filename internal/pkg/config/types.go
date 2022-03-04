package config

import (
	"fmt"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/manifest"
	"github.com/hashicorp/go-multierror"
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

func (a Artifact) Validate() error {
	var result error

	if a.Repository == "" {
		result = multierror.Append(result, fmt.Errorf("repository field is required"))
	}

	if a.Name == "" {
		result = multierror.Append(result, fmt.Errorf("name field is required"))
	}

	if a.SHA == "" {
		result = multierror.Append(result, fmt.Errorf("sha field is required"))
	}

	if a.ManifestName == "" {
		result = multierror.Append(result, fmt.Errorf("manifest_name field is required"))
	}

	return result
}
