package main

import (
	"fmt"
	"os"

	"github.com/andrewmarklloyd/pi-app-updater/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/file"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/github"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/mqtt"
)

const (
	piUserHomeDir = "/home/pi"
)

type tmpOutputPaths struct {
	ServiceFile        string
	UpdaterServiceFile string
	RunScript          string
	PackageBinary      string
	DownloadDirectory  string
	SrcDestMap         map[string]string
}

type Agent struct {
	Config       config.Config
	MqttClient   mqtt.MqttClient
	GHApiToken   string
	HerokuAPIKey string
	ServerApiKey string
	TestMode     bool
	VersionTool  file.VersionTool
	SystemdTool  file.SystemdTool
}

func newAgent(cfg config.Config, client mqtt.MqttClient, ghApiToken, herokuAPIKey, serverApiKey string, versionTool file.VersionTool, systemdTool file.SystemdTool, testMode bool) Agent {

	return Agent{
		Config:       cfg,
		MqttClient:   client,
		GHApiToken:   ghApiToken,
		HerokuAPIKey: herokuAPIKey,
		ServerApiKey: serverApiKey,
		VersionTool:  versionTool,
		SystemdTool:  systemdTool,
		TestMode:     testMode,
	}
}

func (a *Agent) handleRepoUpdate(artifact config.Artifact) error {
	logger.Println(fmt.Sprintf("Received message on topic %s:", config.RepoPushTopic), artifact.Repository)

	err := a.gatherDependencies(artifact)
	if err != nil {
		return err
	}

	err = a.installDependencies(artifact)
	if err != nil {
		return err
	}

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

	// if a.TestMode {
	// 	logger.Println("*** Test mode, not installing files ***")
	// 	return nil
	// }

	err = a.installDependencies(artifact)
	if err != nil {
		return err
	}

	// agent.VersionTool.WriteCurrentVersion("hello-world")
	return nil
}

func (a *Agent) gatherDependencies(artifact config.Artifact) error {
	paths := a.calcTmpOutputPaths()

	err := file.DownloadExtract(artifact.ArchiveDownloadURL, paths.DownloadDirectory, a.GHApiToken)
	if err != nil {
		return fmt.Errorf("downloading and extracting artifact: %s", err)
	}

	m, err := manifest.GetManifest(fmt.Sprintf("%s/.pi-app-updater.yaml", paths.DownloadDirectory))
	if err != nil {
		return fmt.Errorf("getting manifest from directory %s: %s", paths.DownloadDirectory, err)
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

	err = os.WriteFile(paths.ServiceFile, []byte(serviceUnit), 0644)
	if err != nil {
		return fmt.Errorf("writing service file: %s", err)
	}

	err = os.WriteFile(paths.RunScript, []byte(runScript), 0644)
	if err != nil {
		return fmt.Errorf("writing run script: %s", err)
	}

	err = os.WriteFile(paths.UpdaterServiceFile, []byte(updaterFile), 0644)
	if err != nil {
		return fmt.Errorf("writing updater service file: %s", err)
	}

	return nil
}

func (a *Agent) installDependencies(artifact config.Artifact) error {
	paths := a.calcTmpOutputPaths()

	err := file.MakeExecutable([]string{paths.RunScript, paths.PackageBinary})
	if err != nil {
		return err
	}

	err = a.SystemdTool.StopSystemdUnit()
	if err != nil {
		return err
	}

	err = file.CopyWithOwnership(paths.SrcDestMap)
	if err != nil {
		return err
	}

	err = a.SystemdTool.SetupSystemdUnits()
	if err != nil {
		return err
	}

	err = os.RemoveAll(paths.DownloadDirectory)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	return nil
}

func (a *Agent) calcTmpOutputPaths() tmpOutputPaths {
	dlDir := file.DownloadDirectory(a.Config.PackageName)
	runScriptFile := fmt.Sprintf("run-%s.sh", a.Config.PackageName)
	runScriptOutputPath := fmt.Sprintf("%s/%s", dlDir, runScriptFile)

	serviceFile := fmt.Sprintf("%s.service", a.Config.PackageName)
	serviceFileOutputPath := fmt.Sprintf("%s/%s", dlDir, serviceFile)

	updaterServiceFileOutputPath := fmt.Sprintf("%s/%s", dlDir, "pi-app-updater.service")

	tmpBinarypath := fmt.Sprintf("%s/%s", dlDir, a.Config.PackageName)
	packageBinaryOutputPath := fmt.Sprintf("%s/%s", piUserHomeDir, a.Config.PackageName)

	var srcDestMap = map[string]string{
		serviceFileOutputPath: a.SystemdTool.UnitPath,
		runScriptOutputPath:   fmt.Sprintf("%s/%s", piUserHomeDir, runScriptFile),
		tmpBinarypath:         packageBinaryOutputPath,
	}

	return tmpOutputPaths{
		RunScript:          runScriptOutputPath,
		ServiceFile:        serviceFileOutputPath,
		UpdaterServiceFile: updaterServiceFileOutputPath,
		PackageBinary:      tmpBinarypath,
		DownloadDirectory:  dlDir,
		SrcDestMap:         srcDestMap,
	}
}
