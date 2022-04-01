package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/manifest"
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
	ServerApiKey string
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

	serverApiKey := envVars["PI_APP_DEPLOYER_API_KEY"]
	if serverApiKey == "" {
		return Agent{}, fmt.Errorf("PI_APP_DEPLOYER_API_KEY environment variable not found from Heroku")
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
	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", user, password, strings.Split(mqttURL, "@")[1])
	client := mqtt.NewMQTTClient(mqttAddr, *logger)

	return Agent{
		MqttClient:   client,
		GHApiToken:   ghApiToken,
		HerokuAPIKey: herokuAPIKey,
		ServerApiKey: serverApiKey,
	}, nil
}

func (a *Agent) handleRepoUpdate(artifact config.Artifact, cfg config.Config) error {
	logger.Println(fmt.Sprintf("updating manifest %s for repository %s", artifact.ManifestName, artifact.RepoName))

	url, err := github.GetDownloadURLWithRetries(artifact, false)
	if err != nil {
		return err
	}
	artifact.ArchiveDownloadURL = url
	err = a.installOrUpdateApp(artifact, cfg)
	if err != nil {
		return err
	}

	return nil
}

func (a *Agent) handleInstall(artifact config.Artifact, cfg config.Config) error {
	url, err := github.GetDownloadURLWithRetries(artifact, true)
	if err != nil {
		logger.Fatalln(fmt.Errorf("getting download url for latest release: %s", err))
	}

	artifact.SHA = "HEAD"
	artifact.ArchiveDownloadURL = url

	err = a.installOrUpdateApp(artifact, cfg)
	if err != nil {
		return err
	}

	return nil
}

func (a *Agent) installOrUpdateApp(artifact config.Artifact, cfg config.Config) error {
	dlDir := getDownloadDir(artifact)
	err := file.DownloadExtract(artifact.ArchiveDownloadURL, dlDir, a.GHApiToken)
	if err != nil {
		return fmt.Errorf("downloading and extracting artifact: %s", err)
	}

	m, err := manifest.GetManifest(fmt.Sprintf("%s/.pi-app-deployer.yaml", dlDir), artifact.ManifestName)
	if err != nil {
		return fmt.Errorf("getting manifest from directory %s: %s", dlDir, err)
	}

	err = config.ValidateEnvVars(m, cfg)
	if err != nil {
		return fmt.Errorf("validating manifest and config env vars: %s", err)
	}

	err = file.WriteServiceEnvFile(m, a.HerokuAPIKey, artifact.SHA, cfg, "")
	if err != nil {
		return fmt.Errorf("writing service file environment file: %s", err)
	}

	serviceUnit, err := file.EvalServiceTemplate(m, cfg.AppUser)
	if err != nil {
		return fmt.Errorf("rendering service template: %s", err)
	}

	runScript, err := file.EvalRunScriptTemplate(m, artifact.SHA)
	if err != nil {
		return fmt.Errorf("rendering runscript template: %s", err)
	}

	deployerFile, err := file.EvalDeployerTemplate(cfg)
	if err != nil {
		return fmt.Errorf("rendering deployer template: %s", err)
	}

	for _, t := range []string{serviceUnit, runScript, deployerFile} {
		if t == "" {
			return fmt.Errorf("one of the templates rendered was empty")
		}
	}

	serviceFile := fmt.Sprintf("%s.service", m.Name)
	serviceFileOutputPath := fmt.Sprintf("%s/%s", dlDir, serviceFile)
	err = os.WriteFile(serviceFileOutputPath, []byte(serviceUnit), 0644)
	if err != nil {
		return fmt.Errorf("writing service file: %s", err)
	}

	runScriptFile := fmt.Sprintf("run-%s.sh", m.Name)
	runScriptOutputPath := fmt.Sprintf("%s/%s", dlDir, runScriptFile)
	err = os.WriteFile(runScriptOutputPath, []byte(runScript), 0644)
	if err != nil {
		return fmt.Errorf("writing run script: %s", err)
	}

	deployerServiceFileOutputPath := fmt.Sprintf("%s/%s", dlDir, "pi-app-deployer-agent.service")
	err = os.WriteFile(deployerServiceFileOutputPath, []byte(deployerFile), 0644)
	if err != nil {
		return fmt.Errorf("writing deployer service file: %s", err)
	}

	err = file.StopSystemdUnit(m.Name)
	if err != nil {
		return err
	}

	// Don't overwrite agent systemd unit if already exists
	if _, err := os.Stat("/etc/systemd/system/pi-app-deployer-agent.service"); errors.Is(err, os.ErrNotExist) {
		err = file.CopyWithOwnership(map[string]string{
			deployerServiceFileOutputPath: "/etc/systemd/system/pi-app-deployer-agent.service",
		})
		if err != nil {
			return err
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
		return err
	}

	err = file.MakeExecutable([]string{fmt.Sprintf("%s/%s", config.PiAppDeployerDir, runScriptFile), packageBinaryOutputPath})
	if err != nil {
		return err
	}

	err = file.SetupSystemdUnits(m.Name)
	if err != nil {
		return err
	}

	err = os.RemoveAll(dlDir)
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	return nil
}

// TODO: is there a better way to capture closures without so much nesting?
func (a *Agent) startLogForwarder(deplerConfig config.DeployerConfig, f func(config.Log)) {
	for _, cfg := range deplerConfig.AppConfigs {
		if cfg.LogForwarding {
			go func(n config.Config) {
				ch := make(chan string)
				go file.TailSystemdLogs(n.ManifestName, ch)
				for logs := range ch {
					logLines := strings.Split(strings.Replace(logs, "\n", `\n`, -1), `\n`)
					for _, line := range logLines {
						if line != "" {
							l := config.Log{
								Message: line,
								Config:  n,
							}
							f(l)
						}
					}
				}
			}(cfg)
		}
	}
}

func (a *Agent) publishUpdateCondition(c config.UpdateCondition) error {
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

func getDownloadDir(a config.Artifact) string {
	return fmt.Sprintf("/tmp/%s", strings.ReplaceAll(a.RepoName, "/", "_"))
}
