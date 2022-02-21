package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/andrewmarklloyd/pi-app-updater/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/file"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/github"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/mqtt"
)

type Agent struct {
	Config       config.Config
	MqttClient   mqtt.MqttClient
	GHApiToken   string
	HerokuAPIKey string
	TestMode     bool
	VersionTool  file.VersionTool
}

func newAgent(cfg config.Config, client mqtt.MqttClient, ghApiToken, herokuAPIKey string, versionTool file.VersionTool, testMode bool) Agent {
	return Agent{
		Config:       cfg,
		MqttClient:   client,
		GHApiToken:   ghApiToken,
		HerokuAPIKey: herokuAPIKey,
		VersionTool:  versionTool,
		TestMode:     testMode,
	}
}

func (a *Agent) handleRepoUpdate(artifact config.Artifact) error {
	logger.Println(fmt.Sprintf("Received message on topic %s:", config.RepoPushTopic), artifact.Repository)

	err := a.gatherDependencies(artifact)
	if err != nil {
		return err
	}
	// stop systemd unit. Replace unit file and run file. Reload systemd daemon. Restart systemd unit.

	return nil
}

func (a *Agent) handleInstall(artifact config.Artifact) error {
	installed, version, err := a.VersionTool.AppInstalled()
	if err != nil {
		return fmt.Errorf("checking if app is installed already: %s", err)
	}
	if installed {
		return fmt.Errorf(fmt.Sprintf("App already installed at version '%s', remove '--install' flag to check for updates", version))
	}

	url, err := github.GetDownloadURLWithRetries(artifact, true)
	if err != nil {
		logger.Fatalln(fmt.Errorf("getting download url for latest release: %s", err))
	}

	artifact.ArchiveDownloadURL = url
	err = a.gatherDependencies(artifact)
	if err != nil {
		return err
	}
	// agent.VersionTool.WriteCurrentVersion("hello-world")
	return nil
}

func (a *Agent) gatherDependencies(artifact config.Artifact) error {
	dlDir := file.DownloadDirectory(a.Config.PackageName)
	err := file.DownloadExtract(artifact.ArchiveDownloadURL, dlDir, a.GHApiToken)
	if err != nil {
		return fmt.Errorf("downloading and extracting artifact: %s", err)
	}

	m, err := manifest.GetManifest(fmt.Sprintf("%s/.pi-app-updater.yaml", dlDir))
	if err != nil {
		return fmt.Errorf("getting manifest from directory %s: %s", dlDir, err)
	}

	c, err := file.RenderTemplates(m, a.Config)
	if err != nil {
		return fmt.Errorf("rendering templates: %s", err)
	}

	for _, t := range []string{c.PiAppUpdater, c.RunScript, c.Systemd} {
		if t == "" {
			return fmt.Errorf("one of the templates returned was empty")
		}
	}

	// updating heroku api key is required so we don't send it
	// to the server unnecessarily
	c.Systemd = strings.ReplaceAll(c.Systemd, "{{.HerokuAPIKey}}", a.HerokuAPIKey)

	serviceFile := fmt.Sprintf("%s.service", a.Config.PackageName)
	serviceFileOutputPath := fmt.Sprintf("%s/%s", dlDir, serviceFile)
	err = os.WriteFile(serviceFileOutputPath, []byte(file.FromJSONCompliant(c.Systemd)), 0644)
	if err != nil {
		return fmt.Errorf("writing service file: %s", err)
	}

	runScriptFile := fmt.Sprintf("run-%s.sh", a.Config.PackageName)
	runScriptOutputPath := fmt.Sprintf("%s/%s", dlDir, runScriptFile)
	err = os.WriteFile(runScriptOutputPath, []byte(file.FromJSONCompliant(c.RunScript)), 0644)
	if err != nil {
		return fmt.Errorf("writing run script: %s", err)
	}

	updaterServiceFileOutputPath := fmt.Sprintf("%s/%s", dlDir, "pi-app-updater.service")
	err = os.WriteFile(updaterServiceFileOutputPath, []byte(c.PiAppUpdater), 0644)
	if err != nil {
		return fmt.Errorf("writing updater service file: %s", err)
	}

	return nil
}
