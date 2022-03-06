package file

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
	"github.com/hashicorp/go-multierror"
)

//go:embed templates/run.tmpl
var runScriptTemplate string

//go:embed templates/service.tmpl
var serviceTemplate string

//go:embed templates/pi-app-deployer-agent.tmpl
var deployerTemplate string

type ServiceTemplateData struct {
	Description     string
	After           string
	Requires        string
	ExecStart       string
	TimeoutStartSec int
	Restart         string
	RestartSec      int
	EnvironmentFile string
	HomeDir         string
	AppUser         string
}

type DeployerTemplateData struct {
	HomeDir         string
	EnvironmentFile string
	RepoName        string
	ManifestName    string
}

type RunScriptTemplateData struct {
	EnvVarKeys    []string
	AppVersion    string
	ExecStart     string
	HerokuAppName string
	BinaryPath    string
	NewLine       string
}

func EvalServiceTemplate(m manifest.Manifest, homeDir, user string) (string, error) {
	d := ServiceTemplateData{
		Description:     m.Systemd.Unit.Description,
		ExecStart:       getExecStartName(m, homeDir),
		TimeoutStartSec: m.Systemd.Service.TimeoutStartSec,
		Restart:         m.Systemd.Service.Restart,
		RestartSec:      m.Systemd.Service.RestartSec,
		EnvironmentFile: getServiceEnvFileName(m, homeDir),
		HomeDir:         homeDir,
		AppUser:         user,
	}

	for _, a := range m.Systemd.Unit.After {
		d.After += fmt.Sprintf("%s ", a)
	}
	d.After = strings.Trim(d.After, " ")

	for _, a := range m.Systemd.Unit.Requires {
		d.Requires += fmt.Sprintf("%s ", a)
	}
	d.Requires = strings.Trim(d.Requires, " ")
	return evalTemplate(serviceTemplate, d)
}

func EvalRunScriptTemplate(m manifest.Manifest, version, homeDir string) (string, error) {
	d := RunScriptTemplateData{}
	d.EnvVarKeys = m.Heroku.Env
	d.AppVersion = version
	d.ExecStart = getExecStartName(m, homeDir)
	d.HerokuAppName = m.Heroku.App
	d.NewLine = "\n"
	d.BinaryPath = getBinaryPath(m, homeDir)
	return evalTemplate(runScriptTemplate, d)
}

func EvalDeployerTemplate(cfg config.Config) (string, error) {
	var result error

	if cfg.RepoName == "" {
		result = multierror.Append(result, fmt.Errorf("config repo name is required"))
	}

	if cfg.ManifestName == "" {
		result = multierror.Append(result, fmt.Errorf("config manifest name is required"))
	}

	d := DeployerTemplateData{
		EnvironmentFile: getDeployerEnvFileName(cfg.HomeDir),
		RepoName:        cfg.RepoName,
		ManifestName:    cfg.ManifestName,
		HomeDir:         cfg.HomeDir,
	}

	if result != nil {
		return "", result
	}

	return evalTemplate(deployerTemplate, d)
}

func evalTemplate(templateFile string, d interface{}) (string, error) {
	t, err := template.New("").Delims("<<", ">>").Parse(templateFile)
	if err != nil {
		return "", fmt.Errorf("opening service file: %s", err)
	}

	var doc bytes.Buffer
	err = t.Execute(&doc, d)
	if err != nil {
		return "", fmt.Errorf("executing template: %s", err)
	}

	return doc.String(), nil
}

func WriteServiceEnvFile(m manifest.Manifest, herokuAPIKey, version, homeDir string) error {
	envTemplate := `HEROKU_API_KEY=%s
APP_VERSION=%s`
	err := os.WriteFile(getServiceEnvFileName(m, homeDir), []byte(fmt.Sprintf(envTemplate, herokuAPIKey, version)), 0644)
	if err != nil {
		return fmt.Errorf("writing service env file: %s", err)
	}
	return nil
}

func getExecStartName(m manifest.Manifest, homeDir string) string {
	return fmt.Sprintf("%s/run-%s.sh", homeDir, m.Name)
}

func getBinaryPath(m manifest.Manifest, homeDir string) string {
	return fmt.Sprintf("%s/%s", homeDir, m.Executable)
}

func getServiceEnvFileName(m manifest.Manifest, homeDir string) string {
	return fmt.Sprintf("%s/.%s.env", homeDir, m.Name)
}

func getDeployerEnvFileName(homeDir string) string {
	return fmt.Sprintf("%s/.pi-app-deployer-agent.env", homeDir)
}

func FromJSONCompliant(fileWithNewlines string) string {
	return strings.Replace(fileWithNewlines, `\n`, "\n", -1)
}

func ToJSONCompliant(fileRendered string) string {
	return strings.Replace(fileRendered, "\n", `\n`, -1)
}
