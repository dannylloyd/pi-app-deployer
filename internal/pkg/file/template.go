package file

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/andrewmarklloyd/pi-app-updater/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/heroku"
)

//go:embed templates/run.tmpl
var runScriptTemplate string

//go:embed templates/service.tmpl
var serviceTemplate string

//go:embed templates/pi-app-updater.tmpl
var updaterTemplate string

type ServiceTemplateData struct {
	Description     string
	After           string
	Requires        string
	ExecStart       string
	TimeoutStartSec int
	Restart         string
	RestartSec      int
	HerokuAPIKey    string
}

type RunScriptTemplateData struct {
	EnvVarKeys    []string
	ExecStart     string
	HerokuAppName string
	NewLine       string
}

type UpdaterTemplateData struct {
	RepoName    string
	PackageName string
}

func EvalServiceTemplate(outputPath string, m manifest.Manifest, herokuAPIKey string) error {
	d := ServiceTemplateData{
		Description:     m.Systemd.Unit.Description,
		ExecStart:       getExecStartName(m),
		TimeoutStartSec: m.Systemd.Service.TimeoutStartSec,
		Restart:         m.Systemd.Service.Restart,
		RestartSec:      m.Systemd.Service.RestartSec,
		HerokuAPIKey:    herokuAPIKey,
	}

	for _, a := range m.Systemd.Unit.After {
		d.After += fmt.Sprintf("%s ", a)
	}
	d.After = strings.Trim(d.After, " ")

	for _, a := range m.Systemd.Unit.Requires {
		d.Requires += fmt.Sprintf("%s ", a)
	}
	d.Requires = strings.Trim(d.Requires, " ")
	return evalTemplate(serviceTemplate, outputPath, d)
}

func EvalRunScriptTemplate(outputPath string, m manifest.Manifest, h heroku.HerokuClient) error {
	d := RunScriptTemplateData{}
	envVars, err := h.GetEnv()
	if err != nil {
		return fmt.Errorf("getting env from heroku: %s", err)
	}

	envVarKeys := []string{}
	for _, v := range m.Heroku.Env {
		if envVars[v] == "" {
			fmt.Println(fmt.Sprintf("Env var '%s' declared in manifest, but is not set in Heroku config vars", v))
		} else {
			envVarKeys = append(envVarKeys, v)
		}
	}
	d.EnvVarKeys = envVarKeys
	d.ExecStart = getExecStartName(m)
	d.HerokuAppName = m.Heroku.App
	d.NewLine = "\n"
	return evalTemplate(runScriptTemplate, outputPath, d)
}

func EvalUpdaterTemplate(outputPath string, cfg config.Config) error {
	d := UpdaterTemplateData{
		PackageName: cfg.PackageName,
		RepoName:    cfg.RepoName,
	}
	return evalTemplate(updaterTemplate, outputPath, d)
}

func evalTemplate(templateFile string, outputPath string, d interface{}) error {
	t, err := template.New("").Delims("<<", ">>").Parse(templateFile)
	if err != nil {
		return fmt.Errorf("opening service file: %s", err)
	}

	fi, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("opening service file: %s", err)
	}
	err = t.Execute(fi, d)
	if err != nil {
		return fmt.Errorf("executing template: %s", err)
	}
	return nil
}

func getExecStartName(m manifest.Manifest) string {
	return fmt.Sprintf("/home/pi/run-%s.sh", m.Name)
}
