package status

type UpdateCondition struct {
	Status       string `json:"status"`
	RepoName     string `json:"repoName"`
	ManifestName string `json:"manifestName"`
	Error        string `json:"error"`
}
