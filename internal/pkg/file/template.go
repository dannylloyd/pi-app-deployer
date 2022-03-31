package file

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
)

//go:embed templates/run.tmpl
var runScriptTemplate string

//go:embed templates/service.tmpl
var serviceTemplate string

//go:embed templates/pi-app-deployer-agent.tmpl
var deployerTemplate string

type ServiceTemplateData struct {
	Description      string
	After            string
	Requires         string
	ExecStart        string
	TimeoutStartSec  int
	Restart          string
	RestartSec       int
	EnvironmentFile  string
	AppUser          string
	WorkingDirectory string
}

type DeployerTemplateData struct {
	WorkingDirectory string
	EnvironmentFile  string
	RepoName         string
	ManifestName     string
	ExecStart        string
}

type RunScriptTemplateData struct {
	EnvVarKeys    []string
	AppVersion    string
	ExecStart     string
	HerokuAppName string
	BinaryPath    string
	NewLine       string
}

func EvalServiceTemplate(m manifest.Manifest, user string) (string, error) {
	d := ServiceTemplateData{
		Description:      m.Systemd.Unit.Description,
		ExecStart:        getExecStartName(m, config.PiAppDeployerDir),
		TimeoutStartSec:  m.Systemd.Service.TimeoutStartSec,
		Restart:          m.Systemd.Service.Restart,
		RestartSec:       m.Systemd.Service.RestartSec,
		EnvironmentFile:  getServiceEnvFileName(m, config.PiAppDeployerDir),
		WorkingDirectory: config.PiAppDeployerDir,
		AppUser:          user,
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

func EvalRunScriptTemplate(m manifest.Manifest, version string) (string, error) {
	d := RunScriptTemplateData{}
	d.EnvVarKeys = m.Heroku.Env
	d.AppVersion = version
	d.ExecStart = getExecStartName(m, config.PiAppDeployerDir)
	d.HerokuAppName = m.Heroku.App
	d.NewLine = "\n"
	d.BinaryPath = getBinaryPath(m, config.PiAppDeployerDir)
	return evalTemplate(runScriptTemplate, d)
}

func EvalDeployerTemplate(cfg config.Config) (string, error) {
	d := DeployerTemplateData{
		EnvironmentFile:  getDeployerEnvFileName(config.PiAppDeployerDir),
		WorkingDirectory: config.PiAppDeployerDir,
		ExecStart:        getDeployerExecStart(cfg),
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

func WriteServiceEnvFile(m manifest.Manifest, herokuAPIKey, version string, cfg config.Config, outpath string) error {
	if outpath == "" {
		outpath = config.PiAppDeployerDir
	}
	envTemplate := `HEROKU_API_KEY=%s
APP_VERSION=%s`
	sorted := sortMap(cfg.EnvVars)
	for k, v := range sorted {
		envTemplate += fmt.Sprintf("\n%s=%s", k, v)
	}
	err := os.WriteFile(getServiceEnvFileName(m, outpath), []byte(fmt.Sprintf(envTemplate, herokuAPIKey, version)), 0644)
	if err != nil {
		return fmt.Errorf("writing service env file: %s", err)
	}
	return nil
}

func sortMap(envVars map[string]string) map[string]string {
	var keys []string
	var newMap = make(map[string]string, len(envVars))
	for k := range envVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		newMap[k] = envVars[k]
	}
	return newMap
}

func getExecStartName(m manifest.Manifest, homeDir string) string {
	return fmt.Sprintf("%s/run-%s.sh", homeDir, m.Name)
}

func getDeployerExecStart(cfg config.Config) string {
	execStart := fmt.Sprintf("%s/pi-app-deployer-agent update", config.PiAppDeployerDir)
	return execStart
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
