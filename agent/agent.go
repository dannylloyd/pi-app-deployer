package main

import (
	"fmt"
	"strings"

	"github.com/andrewmarklloyd/pi-app-updater/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/file"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/mqtt"
)

type Agent struct {
	Config       config.Config
	MqttClient   mqtt.MqttClient
	GHApiToken   string
	HerokuAPIKey string
}

func newAgent(cfg config.Config, client mqtt.MqttClient, ghApiToken, herokuAPIKey string) Agent {
	return Agent{
		Config:       cfg,
		MqttClient:   client,
		GHApiToken:   ghApiToken,
		HerokuAPIKey: herokuAPIKey,
	}
}

func (a *Agent) handleRepoUpdate(artifact config.Artifact) error {
	logger.Println(fmt.Sprintf("Received message on topic %s:", config.RepoPushTopic), artifact.Repository)
	dlDir := file.DownloadDirectory(a.Config.PackageName)
	err := file.DownloadExtract(artifact.ArchiveDownloadURL, dlDir, a.GHApiToken)
	if err != nil {
		return fmt.Errorf("downloading and extracting artifact: %s", err)
	}

	m, err := manifest.GetManifest(fmt.Sprintf("%s/.pi-app-updater.yaml", dlDir))
	if err != nil {
		return fmt.Errorf("getting manifest from directory %s: %s", dlDir, err)
	}

	c, err := file.RenderTemplates(m)
	if err != nil {
		return fmt.Errorf("rendering templates: %s", err)
	}

	// updating heroku api key is required so we don't send it
	// to the server unnecessarily
	c.Systemd = strings.ReplaceAll(c.Systemd, "{{.HerokuAPIKey}}", a.HerokuAPIKey)

	return nil
}
