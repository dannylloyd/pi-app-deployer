package file

import (
	"bytes"
	_ "embed"
	"errors"
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

func EvalDeployerTemplate(herokuApp string) (string, error) {
	d := DeployerTemplateData{
		EnvironmentFile:  getDeployerEnvFileName(config.PiAppDeployerDir),
		WorkingDirectory: config.PiAppDeployerDir,
		ExecStart:        getDeployerExecStart(herokuApp),
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
	keys := mapToSortedKeys(cfg.EnvVars)
	for _, k := range keys {
		envTemplate += fmt.Sprintf("\n%s=%s", k, cfg.EnvVars[k])
	}

	err := os.WriteFile(getServiceEnvFileName(m, outpath), []byte(fmt.Sprintf(envTemplate, herokuAPIKey, version)), 0644)
	if err != nil {
		return fmt.Errorf("writing service env file: %s", err)
	}
	return nil
}

func WriteDeployerEnvFile(herokuAPIKey string) error {
	if herokuAPIKey == "" {
		return fmt.Errorf("heroku api key must not be empty")
	}

	envFileName := getDeployerEnvFileName(config.PiAppDeployerDir)

	if _, err := os.Stat(envFileName); errors.Is(err, os.ErrNotExist) {
		content := fmt.Sprintf(`HEROKU_API_KEY=%s`, herokuAPIKey)
		// this is only used for CI testing
		envVar := os.Getenv("INVENTORY_TRANSIENT")
		if envVar != "" {
			content += fmt.Sprintf("\nINVENTORY_TRANSIENT=%s", "true")
		}
		err := os.WriteFile(envFileName, []byte(content), 0644)
		if err != nil {
			return fmt.Errorf("writing deployer env file: %s", err)
		}
	}

	return nil
}

func mapToSortedKeys(envVars map[string]string) []string {
	var keys []string
	for k := range envVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func getExecStartName(m manifest.Manifest, dir string) string {
	return fmt.Sprintf("%s/run-%s.sh", dir, m.Name)
}

func getDeployerExecStart(herokuApp string) string {
	execStart := fmt.Sprintf("%s/pi-app-deployer-agent update --herokuApp %s", config.PiAppDeployerDir, herokuApp)
	return execStart
}

func getBinaryPath(m manifest.Manifest, dir string) string {
	return fmt.Sprintf("%s/%s", dir, m.Executable)
}

func getServiceEnvFileName(m manifest.Manifest, dir string) string {
	return fmt.Sprintf("%s/.%s.env", dir, m.Name)
}

func getDeployerEnvFileName(dir string) string {
	return fmt.Sprintf("%s/.pi-app-deployer-agent.env", dir)
}

func FromJSONCompliant(fileWithNewlines string) string {
	return strings.Replace(fileWithNewlines, `\n`, "\n", -1)
}

func ToJSONCompliant(fileRendered string) string {
	return strings.Replace(fileRendered, "\n", `\n`, -1)
}
