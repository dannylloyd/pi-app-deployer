package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/file"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/github"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/mqtt"
)

type Agent struct {
	Config            config.Config
	MqttClient        mqtt.MqttClient
	GHApiToken        string
	HerokuAPIKey      string
	ServerApiKey      string
	DownloadDirectory string
}

func newAgent(cfg config.Config, client mqtt.MqttClient, ghApiToken, herokuAPIKey, serverApiKey string) Agent {
	dlDir := strings.ReplaceAll(cfg.RepoName, "/", "_")
	return Agent{
		Config:            cfg,
		MqttClient:        client,
		GHApiToken:        ghApiToken,
		HerokuAPIKey:      herokuAPIKey,
		ServerApiKey:      serverApiKey,
		DownloadDirectory: fmt.Sprintf("/tmp/%s", dlDir),
	}
}

func (a *Agent) handleRepoUpdate(artifact config.Artifact) error {
	logger.Println(fmt.Sprintf("updating app for repository %s", artifact.Repository))

	url, err := github.GetDownloadURLWithRetries(artifact, false)
	if err != nil {
		return err
	}
	artifact.ArchiveDownloadURL = url
	err = a.installOrUdpdateApp(artifact)
	if err != nil {
		return err
	}

	return nil
}

func (a *Agent) handleInstall(artifact config.Artifact) error {
	url, err := github.GetDownloadURLWithRetries(artifact, true)
	if err != nil {
		logger.Fatalln(fmt.Errorf("getting download url for latest release: %s", err))
	}

	artifact.SHA = "HEAD"
	artifact.ArchiveDownloadURL = url

	err = a.installOrUdpdateApp(artifact)
	if err != nil {
		return err
	}

	return nil
}

func (a *Agent) installOrUdpdateApp(artifact config.Artifact) error {

	err := file.DownloadExtract(artifact.ArchiveDownloadURL, a.DownloadDirectory, a.GHApiToken)
	if err != nil {
		return fmt.Errorf("downloading and extracting artifact: %s", err)
	}

	m, err := manifest.GetManifest(fmt.Sprintf("%s/.pi-app-deployer.yaml", a.DownloadDirectory), a.Config.ManifestName)
	if err != nil {
		return fmt.Errorf("getting manifest from directory %s: %s", a.DownloadDirectory, err)
	}

	err = file.WriteServiceEnvFile(m, a.HerokuAPIKey, artifact.SHA, a.Config.HomeDir)
	if err != nil {
		return fmt.Errorf("writing service file environment file: %s", err)
	}

	serviceUnit, err := file.EvalServiceTemplate(m, a.Config.HomeDir, a.Config.AppUser)
	if err != nil {
		return fmt.Errorf("rendering service template: %s", err)
	}

	runScript, err := file.EvalRunScriptTemplate(m, artifact.SHA, a.Config.HomeDir)
	if err != nil {
		return fmt.Errorf("rendering runscript template: %s", err)
	}

	deployerFile, err := file.EvalDeployerTemplate(a.Config)
	if err != nil {
		return fmt.Errorf("rendering deployer template: %s", err)
	}

	for _, t := range []string{serviceUnit, runScript, deployerFile} {
		if t == "" {
			return fmt.Errorf("one of the templates rendered was empty")
		}
	}

	serviceFile := fmt.Sprintf("%s.service", m.Name)
	serviceFileOutputPath := fmt.Sprintf("%s/%s", a.DownloadDirectory, serviceFile)
	err = os.WriteFile(serviceFileOutputPath, []byte(serviceUnit), 0644)
	if err != nil {
		return fmt.Errorf("writing service file: %s", err)
	}

	runScriptFile := fmt.Sprintf("run-%s.sh", m.Name)
	runScriptOutputPath := fmt.Sprintf("%s/%s", a.DownloadDirectory, runScriptFile)
	err = os.WriteFile(runScriptOutputPath, []byte(runScript), 0644)
	if err != nil {
		return fmt.Errorf("writing run script: %s", err)
	}

	deployerServiceFileOutputPath := fmt.Sprintf("%s/%s", a.DownloadDirectory, "pi-app-deployer-agent.service")
	err = os.WriteFile(deployerServiceFileOutputPath, []byte(deployerFile), 0644)
	if err != nil {
		return fmt.Errorf("writing deployer service file: %s", err)
	}

	err = file.StopSystemdUnit(m.Name)
	if err != nil {
		return err
	}

	tmpBinarypath := fmt.Sprintf("%s/%s", a.DownloadDirectory, m.Executable)
	packageBinaryOutputPath := fmt.Sprintf("%s/%s", a.Config.HomeDir, m.Executable)

	var srcDestMap = map[string]string{
		serviceFileOutputPath:         fmt.Sprintf("/etc/systemd/system/%s.service", m.Name),
		runScriptOutputPath:           fmt.Sprintf("%s/%s", a.Config.HomeDir, runScriptFile),
		tmpBinarypath:                 packageBinaryOutputPath,
		deployerServiceFileOutputPath: "/etc/systemd/system/pi-app-deployer-agent.service",
	}

	err = file.CopyWithOwnership(srcDestMap)
	if err != nil {
		return err
	}

	err = file.MakeExecutable([]string{fmt.Sprintf("%s/%s", a.Config.HomeDir, runScriptFile), packageBinaryOutputPath})
	if err != nil {
		return err
	}

	err = file.SetupSystemdUnits(m.Name)
	if err != nil {
		return err
	}

	err = os.RemoveAll(a.DownloadDirectory)
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	return nil
}

func (a *Agent) startLogForwarder(unitName string, f func(string)) {
	ch := make(chan string)
	go file.TailSystemdLogs(unitName, ch)
	for logs := range ch {
		logLines := strings.Split(strings.Replace(logs, "\n", `\n`, -1), `\n`)
		for _, line := range logLines {
			if line != "" {
				f(line)
			}
		}
	}
}
