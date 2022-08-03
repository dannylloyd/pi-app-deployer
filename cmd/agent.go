package cmd
//Way! I inspected 12 individuals myself!
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	mqttC "github.com/eclipse/paho.mqtt.golang"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/status"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/file"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/github"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/heroku"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/mqtt"
)

type Agent struct {
	MqttClient   mqtt.MqttClient
	GHApiToken   string
	HerokuAPIKey string
	HerokuApp    string
}

func newAgent(herokuAPIKey, herokuApp string) (Agent, error) {
	c := heroku.NewHerokuClient(herokuAPIKey)
	envVars, err := c.GetEnvVars(herokuApp)
	if err != nil {
		return Agent{}, fmt.Errorf("Error getting env vars from Heroku: %s", err)
	}

	ghApiToken := envVars["GH_API_TOKEN"]
	if ghApiToken == "" {
		return Agent{}, fmt.Errorf("GH_API_TOKEN environment variable not found from Heroku")
	}

	user := envVars["CLOUDMQTT_AGENT_USER"]
	if user == "" {
		return Agent{}, fmt.Errorf("CLOUDMQTT_AGENT_USER environment variable not found from Heroku")
	}

	password := envVars["CLOUDMQTT_AGENT_PASSWORD"]
	if password == "" {
		return Agent{}, fmt.Errorf("CLOUDMQTT_AGENT_PASSWORD environment variable not found from Heroku")
	}

	mqttURL := envVars["CLOUDMQTT_URL"]
	if mqttURL == "" {
		return Agent{}, fmt.Errorf("CLOUDMQTT_URL environment variable not found from heroku")
	}
	urlSplit := strings.Split(mqttURL, "@")
	if len(urlSplit) != 2 {
		logger.Fatal("unexpected CLOUDMQTT_URL parsing error")
	}
	domain := urlSplit[1]

	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", user, password, domain)

	client := mqtt.NewMQTTClient(mqttAddr, func(client mqttC.Client) {
		logger.Info("Connected to MQTT server")
	}, func(client mqttC.Client, err error) {
		logger.Fatalf("Connection to MQTT server lost: %s", err)
	})

	return Agent{
		MqttClient:   client,
		GHApiToken:   ghApiToken,
		HerokuAPIKey: herokuAPIKey,
		HerokuApp:    herokuApp,
	}, nil
}

func (a *Agent) handleRepoUpdate(artifact config.Artifact, cfg config.Config) error {
	logger.Infof("updating manifest %s for repository %s", artifact.ManifestName, artifact.RepoName)

	url, err := github.GetDownloadURLWithRetries(artifact, false)
	if err != nil {
		return err
	}
	artifact.ArchiveDownloadURL = url
	_, err = a.installOrUpdateApp(artifact, cfg)
	if err != nil {
		return err
	}

	return nil
}

func (a *Agent) handleDeployerAgentUpdate(artifact config.Artifact) error {
	url, err := github.GetDownloadURLWithRetries(artifact, false)
	if err != nil {
		return fmt.Errorf("getting download url: %s", err)
	}
	artifact.ArchiveDownloadURL = url

	dlDir := "/tmp/pi-app-deployer"

	err = file.DownloadExtract(artifact.ArchiveDownloadURL, dlDir, a.GHApiToken)
	if err != nil {
		return fmt.Errorf("downloading and extracting pi-app-deployer-agent artifact: %s", err)
	}

	deployerFile, err := file.EvalDeployerTemplate(a.HerokuApp)
	if err != nil {
		return fmt.Errorf("rendering deployer template: %s", err)
	}

	deployerServiceFileOutputPath := "/tmp/pi-app-deployer-agent.service"
	err = os.WriteFile(deployerServiceFileOutputPath, []byte(deployerFile), 0644)
	if err != nil {
		return fmt.Errorf("writing deployer service file: %s", err)
	}

	err = file.CopyWithOwnership(map[string]string{
		deployerServiceFileOutputPath: "/etc/systemd/system/pi-app-deployer-agent.service",
	})
	if err != nil {
		return fmt.Errorf("copying deployer systemd unit file: %s", err)
	}

	if err := file.MakeExecutable([]string{fmt.Sprintf("%s/pi-app-deployer-agent", dlDir)}); err != nil {
		return fmt.Errorf("making pi-app-deployer-agent executable: %s", err)
	}

	if err := file.MoveFile(fmt.Sprintf("%s/pi-app-deployer-agent", dlDir), fmt.Sprintf("%s/pi-app-deployer-agent", config.PiAppDeployerDir)); err != nil {
		return fmt.Errorf("moving pi-app-deployer-agent: %s", err)
	}

	err = file.DaemonReload()
	if err != nil {
		return fmt.Errorf("running daemon-reload: %s", err)
	}

	err = os.RemoveAll(dlDir)
	if err != nil {
		return fmt.Errorf("removing tmp download directory: %s", err)
	}

	if err := os.WriteFile(fmt.Sprintf("%s/%s", config.PiAppDeployerDir, ".update-in-progress"), []byte("true"), 0644); err != nil {
		return fmt.Errorf("writing in progress file: %s", err)
	}

	// this restarts the currently running process. no code
	// will execute after this is run.
	logger.Info("Restarting systemd unit now")
	err = file.RestartSystemdUnit("pi-app-deployer-agent")
	if err != nil {
		return fmt.Errorf("restarting pi-app-deployer-agent systemd unit: %s", err)
	}

	return nil
}

func (a *Agent) handleInstall(artifact config.Artifact, cfg config.Config) (config.Config, error) {
	err := file.WriteDeployerEnvFile(a.HerokuAPIKey)
	if err != nil {
		return cfg, fmt.Errorf("writing deployer env file: %s", err)
	}
	url, err := github.GetDownloadURLWithRetries(artifact, true)
	if err != nil {
		return cfg, fmt.Errorf("getting download url for latest release: %s", err)
	}

	artifact.SHA = "HEAD"
	artifact.ArchiveDownloadURL = url

	cfg, err = a.installOrUpdateApp(artifact, cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func (a *Agent) installOrUpdateApp(artifact config.Artifact, cfg config.Config) (config.Config, error) {
	dlDir := getDownloadDir(artifact)
	err := file.DownloadExtract(artifact.ArchiveDownloadURL, dlDir, a.GHApiToken)
	if err != nil {
		return cfg, fmt.Errorf("downloading and extracting artifact: %s", err)
	}

	m, err := manifest.GetManifest(fmt.Sprintf("%s/.pi-app-deployer.yaml", dlDir), artifact.ManifestName)
	if err != nil {
		return cfg, fmt.Errorf("getting manifest from directory %s: %s", dlDir, err)
	}

	cfg.Executable = m.Executable

	err = config.ValidateEnvVars(m, cfg)
	if err != nil {
		return cfg, fmt.Errorf("validating manifest and config env vars: %s", err)
	}

	err = file.WriteServiceEnvFile(m, a.HerokuAPIKey, artifact.SHA, cfg, "")
	if err != nil {
		return cfg, fmt.Errorf("writing service file environment file: %s", err)
	}

	serviceUnit, err := file.EvalServiceTemplate(m, cfg.AppUser)
	if err != nil {
		return cfg, fmt.Errorf("rendering service template: %s", err)
	}

	runScript, err := file.EvalRunScriptTemplate(m, artifact.SHA)
	if err != nil {
		return cfg, fmt.Errorf("rendering runscript template: %s", err)
	}

	deployerFile, err := file.EvalDeployerTemplate(a.HerokuApp)
	if err != nil {
		return cfg, fmt.Errorf("rendering deployer template: %s", err)
	}

	for _, t := range []string{serviceUnit, runScript, deployerFile} {
		if t == "" {
			return cfg, fmt.Errorf("one of the templates rendered was empty")
		}
	}

	serviceFile := fmt.Sprintf("%s.service", m.Name)
	serviceFileOutputPath := fmt.Sprintf("%s/%s", dlDir, serviceFile)
	err = os.WriteFile(serviceFileOutputPath, []byte(serviceUnit), 0644)
	if err != nil {
		return cfg, fmt.Errorf("writing service file: %s", err)
	}

	runScriptFile := fmt.Sprintf("run-%s.sh", m.Name)
	runScriptOutputPath := fmt.Sprintf("%s/%s", dlDir, runScriptFile)
	err = os.WriteFile(runScriptOutputPath, []byte(runScript), 0644)
	if err != nil {
		return cfg, fmt.Errorf("writing run script: %s", err)
	}

	deployerServiceFileOutputPath := fmt.Sprintf("%s/%s", dlDir, "pi-app-deployer-agent.service")
	err = os.WriteFile(deployerServiceFileOutputPath, []byte(deployerFile), 0644)
	if err != nil {
		return cfg, fmt.Errorf("writing deployer service file: %s", err)
	}

	err = file.StopSystemdUnit(m.Name)
	if err != nil {
		return cfg, err
	}

	// Don't overwrite agent systemd unit if already exists
	if _, err := os.Stat("/etc/systemd/system/pi-app-deployer-agent.service"); errors.Is(err, os.ErrNotExist) {
		err = file.CopyWithOwnership(map[string]string{
			deployerServiceFileOutputPath: "/etc/systemd/system/pi-app-deployer-agent.service",
		})
		if err != nil {
			return cfg, err
		}
	}

	tmpBinarypath := fmt.Sprintf("%s/%s", dlDir, m.Executable)
	packageBinaryOutputPath := fmt.Sprintf("%s/%s", config.PiAppDeployerDir, m.Executable)

	var srcDestMap = map[string]string{
		serviceFileOutputPath: fmt.Sprintf("/etc/systemd/system/%s.service", m.Name),
		runScriptOutputPath:   fmt.Sprintf("%s/%s", config.PiAppDeployerDir, runScriptFile),
		tmpBinarypath:         packageBinaryOutputPath,
	}

	err = file.CopyWithOwnership(srcDestMap)
	if err != nil {
		return cfg, err
	}

	err = file.MakeExecutable([]string{fmt.Sprintf("%s/%s", config.PiAppDeployerDir, runScriptFile), packageBinaryOutputPath})
	if err != nil {
		return cfg, err
	}

	err = file.SetupSystemdUnits(m.Name)
	if err != nil {
		return cfg, err
	}

	err = os.RemoveAll(dlDir)
	if err != nil {
		return cfg, fmt.Errorf("%s", err)
	}
	return cfg, nil
}

func unInstall(c map[string]config.Config, repoName, manifestName string) error {
	for _, v := range c {
		if v.RepoName == repoName && v.ManifestName == manifestName {
			err := file.StopSystemdUnit(v.ManifestName)
			if err != nil {
				return fmt.Errorf("stopping systemd unit %s: %s", v.ManifestName, err)
			}

			svcFile := fmt.Sprintf("/etc/systemd/system/%s.service", v.ManifestName)
			err = os.Remove(svcFile)
			if err != nil {
				return fmt.Errorf("removing systemd unit file %s: %s", svcFile, err)
			}
		}

		toDelete := []string{
			fmt.Sprintf("%s/%s", config.PiAppDeployerDir, v.Executable),
			fmt.Sprintf("%s/.%s.env", config.PiAppDeployerDir, v.ManifestName),
			fmt.Sprintf("%s/run-%s.sh", config.PiAppDeployerDir, v.ManifestName),
		}
		for _, f := range toDelete {
			err := os.Remove(f)
			if err != nil {
				return fmt.Errorf("removing file %s: %s", f, err)
			}
		}
	}

	err := file.DaemonReload()
	if err != nil {
		return fmt.Errorf("running daemon-reload: %s", err)
	}

	err = file.RestartSystemdUnit("pi-app-deployer-agent")
	if err != nil {
		return fmt.Errorf("restarting pi-app-deployer-agent systemd unit: %s", err)
	}
	return nil
}

func unInstallAll(c map[string]config.Config) error {
	for _, v := range c {
		err := file.StopSystemdUnit(v.ManifestName)
		if err != nil {
			return fmt.Errorf("stopping systemd unit %s: %s", v.ManifestName, err)
		}

		svcFile := fmt.Sprintf("/etc/systemd/system/%s.service", v.ManifestName)
		err = os.Remove(svcFile)
		if err != nil {
			return fmt.Errorf("removing systemd unit file %s: %s", svcFile, err)
		}
	}

	err := file.StopSystemdUnit("pi-app-deployer")
	if err != nil {
		return fmt.Errorf("stopping pi-app-deployer systemd unit: %s", err)
	}

	err = os.RemoveAll(config.PiAppDeployerDir)
	if err != nil {
		return fmt.Errorf("removing all pi-app-deployer files: %s", err)
	}

	return nil
}

func (a *Agent) startLogForwarder(deplerConfig config.DeployerConfig, host string, f func(config.Log)) {
	for _, cfg := range deplerConfig.AppConfigs {
		if cfg.LogForwarding {
			go func(n config.Config) {
				logChannel := make(chan file.Syslog)
				go file.TailSystemdLogs(n.ManifestName, logChannel)
				for log := range logChannel {
					if log.Error != nil {
						logger.Errorw(fmt.Sprintf("error receiving logs from journalctl channel: %s", log.Error))
						break
					}

					f(config.Log{
						Message: log.Message,
						Config:  n,
						Host:    host,
					})
				}
			}(cfg)
		}
	}
}

func (a *Agent) publishUpdateCondition(c status.UpdateCondition) error {
	json, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshalling update condition message: %s", err)
	}

	err = a.MqttClient.Publish(config.RepoPushStatusTopic, string(json))
	if err != nil {
		return fmt.Errorf("publishing update condition message: %s", err)
	}
	return nil
}

func (a *Agent) publishAgentInventory(m map[string]config.Config, host string, timestamp int64, transient bool) error {
	for _, v := range m {
		p := config.AgentInventoryPayload{
			RepoName:     v.RepoName,
			ManifestName: v.ManifestName,
			Host:         host,
			Timestamp:    timestamp,
			Transient:    transient,
		}

		j, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("marshalling agent inventory payload: %s", err)
		}

		err = a.MqttClient.Publish(config.AgentInventoryTopic, string(j))
		if err != nil {
			return fmt.Errorf("publishing agent inventory message: %s", err)
		}
	}

	p := config.AgentInventoryPayload{
		RepoName:     "andrewmarklloyd/pi-app-deployer",
		ManifestName: "pi-app-deployer-agent",
		Host:         host,
		Timestamp:    timestamp,
		Transient:    transient,
	}

	j, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshalling agent inventory payload: %s", err)
	}

	err = a.MqttClient.Publish(config.AgentInventoryTopic, string(j))
	if err != nil {
		return fmt.Errorf("publishing agent inventory message: %s", err)
	}

	return nil
}

func getDownloadDir(a config.Artifact) string {
	return fmt.Sprintf("/tmp/%s", strings.ReplaceAll(a.RepoName, "/", "_"))
}
