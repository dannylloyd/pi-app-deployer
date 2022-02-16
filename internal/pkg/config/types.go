package config

type Config struct {
	RepoName    string
	PackageName string
}

type UpdaterPayload struct {
	SHA                string `json:"sha"`
	Repository         string `json:"repository"`
	ArtifactName       string `json:"artifact_name"`
	ArchiveDownloadURL string `json:"archive_download_url"`
}
