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

const (
	piUserHomeDir = "/home/pi"
)

type Agent struct {
	Config            config.Config
	MqttClient        mqtt.MqttClient
	GHApiToken        string
	HerokuAPIKey      string
	ServerApiKey      string
	TestMode          bool
	VersionTool       file.VersionTool
	SystemdTool       file.SystemdTool
	DownloadDirectory string
}

func newAgent(cfg config.Config, client mqtt.MqttClient, ghApiToken, herokuAPIKey, serverApiKey string, systemdTool file.SystemdTool, testMode bool) Agent {
	return Agent{
		Config:            cfg,
		MqttClient:        client,
		GHApiToken:        ghApiToken,
		HerokuAPIKey:      herokuAPIKey,
		ServerApiKey:      serverApiKey,
		SystemdTool:       systemdTool,
		TestMode:          testMode,
		DownloadDirectory: strings.ReplaceAll(fmt.Sprintf("/tmp/%s", cfg.RepoName), "/", "_"),
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
	// todo: find a better way to check version
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

	artifact.SHA = "HEAD"
	artifact.ArchiveDownloadURL = url

	err = a.installOrUdpdateApp(artifact)
	if err != nil {
		return err
	}

	// a.VersionTool.WriteCurrentVersion(artifact.SHA)
	return nil
}

func (a *Agent) installOrUdpdateApp(artifact config.Artifact) error {

	err := file.DownloadExtract(artifact.ArchiveDownloadURL, a.DownloadDirectory, a.GHApiToken)
	if err != nil {
		return fmt.Errorf("downloading and extracting artifact: %s", err)
	}

	m, err := manifest.GetManifest(fmt.Sprintf("%s/.pi-app-updater.yaml", a.DownloadDirectory))
	if err != nil {
		return fmt.Errorf("getting manifest from directory %s: %s", a.DownloadDirectory, err)
	}

	serviceUnit, err := file.EvalServiceTemplate(m, a.HerokuAPIKey)
	if err != nil {
		return fmt.Errorf("rendering service template: %s", err)
	}

	runScript, err := file.EvalRunScriptTemplate(m)
	if err != nil {
		return fmt.Errorf("rendering runscript template: %s", err)
	}

	updaterFile, err := file.EvalUpdaterTemplate(a.Config)
	if err != nil {
		return fmt.Errorf("rendering updater template: %s", err)
	}

	for _, t := range []string{serviceUnit, runScript, updaterFile} {
		if t == "" {
			return fmt.Errorf("one of the templates rendered was empty")
		}
	}

	serviceFile := fmt.Sprintf("%s.service", a.Config.PackageName)
	serviceFileOutputPath := fmt.Sprintf("%s/%s", a.DownloadDirectory, serviceFile)
	err = os.WriteFile(serviceFileOutputPath, []byte(serviceUnit), 0644)
	if err != nil {
		return fmt.Errorf("writing service file: %s", err)
	}

	runScriptFile := fmt.Sprintf("run-%s.sh", a.Config.PackageName)
	runScriptOutputPath := fmt.Sprintf("%s/%s", a.DownloadDirectory, runScriptFile)
	err = os.WriteFile(runScriptOutputPath, []byte(runScript), 0644)
	if err != nil {
		return fmt.Errorf("writing run script: %s", err)
	}

	updaterServiceFileOutputPath := fmt.Sprintf("%s/%s", a.DownloadDirectory, "pi-app-updater-agent.service")
	err = os.WriteFile(updaterServiceFileOutputPath, []byte(updaterFile), 0644)
	if err != nil {
		return fmt.Errorf("writing updater service file: %s", err)
	}

	err = a.SystemdTool.StopSystemdUnit(m.Name)
	if err != nil {
		return err
	}

	tmpBinarypath := fmt.Sprintf("%s/%s", a.DownloadDirectory, m.Executable)
	packageBinaryOutputPath := fmt.Sprintf("%s/%s", piUserHomeDir, m.Executable)

	var srcDestMap = map[string]string{
		serviceFileOutputPath:        fmt.Sprintf("/etc/systemd/system/%s.service", m.Name),
		runScriptOutputPath:          fmt.Sprintf("%s/%s", piUserHomeDir, runScriptFile),
		tmpBinarypath:                packageBinaryOutputPath,
		updaterServiceFileOutputPath: "/etc/systemd/system/pi-app-updater-agent.service",
	}

	err = file.CopyWithOwnership(srcDestMap)
	if err != nil {
		return err
	}

	err = file.MakeExecutable([]string{fmt.Sprintf("%s/%s", piUserHomeDir, runScriptFile), packageBinaryOutputPath})
	if err != nil {
		return err
	}

	err = a.SystemdTool.SetupSystemdUnits(m.Name)
	if err != nil {
		return err
	}

	err = os.RemoveAll(a.DownloadDirectory)
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	return nil
}
