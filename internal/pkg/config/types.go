package config

import (
	"fmt"
	"strings"

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

	return toOnelineErr(result)
}

func (p ServiceActionPayload) Validate() error {
	var result error

	if p.RepoName == "" {
		result = multierror.Append(result, fmt.Errorf("repoName field is required"))
	}

	if p.ManifestName == "" {
		result = multierror.Append(result, fmt.Errorf("manifestName field is required"))
	}

	if p.Action == "" {
		result = multierror.Append(result, fmt.Errorf("action field is required"))
	} else {
		if p.Action != ServiceActionStart && p.Action != ServiceActionStop && p.Action != ServiceActionRestart {
			result = multierror.Append(result, fmt.Errorf("action must be one of: %s, %s, or %s, but was %s", ServiceActionStart, ServiceActionStop, ServiceActionRestart, p.Action))
		}
	}

	return toOnelineErr(result)
}

func toOnelineErr(err error) error {
	if err != nil {
		errString := strings.ReplaceAll(err.Error(), "\t", `\t`)
		return fmt.Errorf("%s", strings.ReplaceAll(errString, "\n", `\n`))
	}
	return nil
}
