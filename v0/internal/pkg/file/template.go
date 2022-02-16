package file

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/andrewmarklloyd/pi-app-updater/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/heroku"
	"github.com/hashicorp/go-multierror"
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

func EvalServiceTemplate(m manifest.Manifest, herokuAPIKey string) (string, error) {
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
	return evalTemplate(serviceTemplate, d)
}

func EvalRunScriptTemplate(m manifest.Manifest, h heroku.HerokuClient) (string, error) {
	d := RunScriptTemplateData{}
	envVars, err := h.GetEnv()
	if err != nil {
		return "", fmt.Errorf("getting env from heroku: %s", err)
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
	return evalTemplate(runScriptTemplate, d)
}

func EvalUpdaterTemplate(cfg config.Config) (string, error) {
	var result error

	if cfg.PackageName == "" {
		result = multierror.Append(result, fmt.Errorf("config package name is required"))
	}
	if cfg.RepoName == "" {
		result = multierror.Append(result, fmt.Errorf("config repo name is required"))

	}
	if result != nil {
		return "", result
	}

	return evalTemplate(updaterTemplate, cfg)
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

func getExecStartName(m manifest.Manifest) string {
	return fmt.Sprintf("/home/pi/run-%s.sh", m.Name)
}
