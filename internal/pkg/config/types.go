package config

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
)

// todo: standardize repoName vs repository. also reuse some structs?

const (
	RepoPushTopic       = "repo/push"
	LogForwarderTopic   = "logs"
	RepoPushStatusTopic = "repo/push/status"
	ServiceActionTopic  = "service"

	StatusUnknown    = "UNKNOWN"
	StatusInProgress = "IN_PROGRESS"
	StatusErr        = "ERROR"
	StatusSuccess    = "SUCCESS"

	ServiceActionStart   = "START"
	ServiceActionStop    = "STOP"
	ServiceActionRestart = "RESTART"
)

type Log struct {
	Message string `json:"message"`
	Config  Config `json:"config"`
}

type UpdateCondition struct {
	Status       string `json:"status"`
	RepoName     string `json:"repoName"`
	ManifestName string `json:"manifestName"`
}

type ServiceActionPayload struct {
	RepoName     string `json:"repoName"`
	ManifestName string `json:"manifestName"`
	Action       string `json:"action"`
}

type Config struct {
	RepoName      string
	ManifestName  string
	HomeDir       string
	AppUser       string
	LogForwarding bool
}

type Artifact struct {
	SHA                string `json:"sha"`
	Repository         string `json:"repository"`
	Name               string `json:"name"`
	ArchiveDownloadURL string `json:"download_url"`
	ManifestName       string `json:"manifest_name"`
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
