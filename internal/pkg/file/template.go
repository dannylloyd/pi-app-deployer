package file

import (
	_ "embed"
	"fmt"
	"os"
	"text/template"
)

//go:embed templates/run.tmpl
var runScriptTemplate string

//go:embed templates/service.tmpl
var serviceTemplate string

type TemplateData struct {
	Name          string
	Keys          []string
	NewLine       string
	HerokuAPIKey  string
	HerokuAppName string
}

func EvalRunScriptTemplate(outputPath string, d TemplateData) error {
	return evalTemplate(runScriptTemplate, outputPath, d)
}

func EvalServiceTemplate(outputPath string, d TemplateData) error {
	return evalTemplate(serviceTemplate, outputPath, d)
}

func evalTemplate(templateFile string, outputPath string, d TemplateData) error {
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
