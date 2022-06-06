package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
)

// todo: standardize repoName vs repository. also reuse some structs?

const (
	PiAppDeployerDir = "/usr/local/src/pi-app-deployer"

	RepoPushTopic       = "repo/push"
	LogForwarderTopic   = "logs"
	RepoPushStatusTopic = "repo/push/status"
	AgentInventoryTopic = "agent/inventory"
	ServiceActionTopic  = "service"

	StatusUnknown    = "UNKNOWN"
	StatusInProgress = "IN_PROGRESS"
	StatusErr        = "ERROR"
	StatusSuccess    = "SUCCESS"

	ServiceActionStart   = "START"
	ServiceActionStop    = "STOP"
	ServiceActionRestart = "RESTART"

	InventoryTickerSchedule = 30 * time.Second
	InventoryTickerTimeout  = 5 * time.Minute
)

type Log struct {
	Message string `json:"message"`
	Config  Config `json:"config"`
	Host    string `json:"host"`
}

type AgentInventoryPayload struct {
	RepoName     string `json:"repoName"`
	ManifestName string `json:"manifestName"`
	Host         string `json:"host"`
	Timestamp    int64  `json:"timestamp"`
}

type ServiceActionPayload struct {
	RepoName     string `json:"repoName"`
	ManifestName string `json:"manifestName"`
	Action       string `json:"action"`
}

type Config struct {
	RepoName      string            `yaml:"repoName"`
	ManifestName  string            `yaml:"manifestName"`
	AppUser       string            `yaml:"appUser"`
	LogForwarding bool              `yaml:"logForwarding"`
	EnvVars       map[string]string `yaml:"envVars"`
	Executable    string            `yaml:"executable"`
}

type DeployStatusPayload struct {
	RepoName     string `json:"repoName"`
	ManifestName string `json:"manifestName"`
}

type Artifact struct {
	SHA                string `json:"sha"`
	RepoName           string `json:"repoName"`
	Name               string `json:"name"`
	ArchiveDownloadURL string `json:"downloadURL"`
	ManifestName       string `json:"manifestName"`
}

func (a Artifact) Validate() error {
	var result error

	if a.RepoName == "" {
		result = multierror.Append(result, fmt.Errorf("repoName field is required"))
	}

	if a.Name == "" {
		result = multierror.Append(result, fmt.Errorf("name field is required"))
	}

	if a.SHA == "" {
		result = multierror.Append(result, fmt.Errorf("sha field is required"))
	}

	if a.ManifestName == "" {
		result = multierror.Append(result, fmt.Errorf("manifestName field is required"))
	}

	return toOnelineErr(result)
}

func (p DeployStatusPayload) Validate() error {
	var result error

	if p.RepoName == "" {
		result = multierror.Append(result, fmt.Errorf("repoName field is required"))
	}

	if p.ManifestName == "" {
		result = multierror.Append(result, fmt.Errorf("manifestName field is required"))
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
