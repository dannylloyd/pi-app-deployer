package config

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

type AgentPayload struct {
	Artifact    Artifact    `json:"artifact"`
	ConfigFiles ConfigFiles `json:"configFiles"`
}
