package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/status"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/file"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Use the update command to update previously installed applications.",
	Long: `The update command opens an MQTT connection to the
server, receives update command on new commits to Github, and
orchestrates updating the Systemd unit.`,
	Run: func(cmd *cobra.Command, args []string) {
		runUpdate(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) {
	host, err := os.Hostname()
	if err != nil {
		logger.Fatalln("error getting hostname:", err)
	}

	herokuAPIKey := os.Getenv("HEROKU_API_KEY")
	if herokuAPIKey == "" {
		logger.Fatalln("HEROKU_API_TOKEN environment variable is required")
	}

	herokuApp, err := cmd.Flags().GetString("herokuApp")
	if err != nil {
		fmt.Println("error getting herokuApp flag", err)
		os.Exit(1)
	}
	if herokuApp == "" {
		fmt.Println("herokuApp flag is required")
		os.Exit(1)
	}

	agent, err := newAgent(herokuAPIKey, herokuApp)
	if err != nil {
		logger.Fatalln(fmt.Errorf("error creating agent: %s", err))
	}

	updateProgressFile := fmt.Sprintf("%s/%s", config.PiAppDeployerDir, ".update-in-progress")
	// TODO: need to clean this up instead of hard coding
	if _, err := os.Stat(updateProgressFile); err == nil {
		updateCondition := status.UpdateCondition{
			RepoName:     "andrewmarklloyd/pi-app-deployer",
			ManifestName: "pi-app-deployer-agent",
			Status:       config.StatusSuccess,
			Host:         host,
		}

		err = agent.publishUpdateCondition(updateCondition)
		if err != nil {
			logger.Println("Error publishing success of previously running version update. This will cause problems attempting to further update the agent:", err)
		}

		err = os.Remove(updateProgressFile)
		if err != nil {
			logger.Println("removing update progress file:", err)
		}
	}

	deployerConfig, err := config.NewDeployerConfig(config.DeployerConfigFile, herokuApp)
	if err != nil {
		logger.Fatalln("error getting app configs:", err)
	}

	if deployerConfig.FeatureAutoUpdateAgent {
		logger.Println("Auto update agent feature flag set to true")
	}

	err = agent.MqttClient.Connect()
	if err != nil {
		logger.Fatalln("connecting to mqtt: ", err)
	}

	inventoryTicker := time.NewTicker(config.InventoryTickerSchedule)
	go func() {
		for t := range inventoryTicker.C {
			err := agent.publishAgentInventory(deployerConfig.AppConfigs, host, t.Unix())
			if err != nil {
				logger.Println("error publishing agent inventory:", err)
			}
		}
	}()

	agent.startLogForwarder(deployerConfig, host, func(l config.Log) {
		json, err := json.Marshal(l)
		if err != nil {
			logger.Println(fmt.Sprintf("marshalling log forwarder message: %s", err))
			return
		}
		err = agent.MqttClient.Publish(config.LogForwarderTopic, string(json))
		if err != nil {
			logger.Println(fmt.Sprintf("error publishing log forwarding message: %s", err))
		}
	})

	agent.MqttClient.Subscribe(config.RepoPushTopic, func(message string) {
		var artifact config.Artifact
		err := json.Unmarshal([]byte(message), &artifact)
		if err != nil {
			logger.Println(fmt.Sprintf("unmarshalling payload from topic %s: %s", config.RepoPushTopic, err))
			return
		}

		if deployerConfig.FeatureAutoUpdateAgent {
			if artifact.RepoName == "andrewmarklloyd/pi-app-deployer" && artifact.ManifestName == "pi-app-deployer-agent" {
				logger.Println("New pi-app-deployer-agent version published, updating now", artifact)
				updateCondition := status.UpdateCondition{
					RepoName:     artifact.RepoName,
					ManifestName: artifact.ManifestName,
					Status:       config.StatusInProgress,
					Host:         host,
				}

				err = agent.publishUpdateCondition(updateCondition)
				if err != nil {
					// log but don't block update from proceeding
					logger.Println(err)
				}

				// note the last step of this function is
				// to restart the systemd unit.
				err = agent.handleDeployerAgentUpdate(artifact)
				if err != nil {
					logger.Println("error updating agent version:", err)
					updateCondition.Error = err.Error()
					updateCondition.Status = config.StatusErr
					err = agent.publishUpdateCondition(updateCondition)
					if err != nil {
						logger.Println(err)
					}
					return
				}
			}
		}

		for _, cfg := range deployerConfig.AppConfigs {
			if artifact.RepoName == cfg.RepoName && artifact.ManifestName == cfg.ManifestName {
				logger.Println(fmt.Sprintf("updating repo %s with manifest name %s", cfg.RepoName, cfg.ManifestName))
				updateCondition := status.UpdateCondition{
					RepoName:     cfg.RepoName,
					ManifestName: cfg.ManifestName,
					Status:       config.StatusInProgress,
					Host:         host,
				}

				err = agent.publishUpdateCondition(updateCondition)
				if err != nil {
					// log but don't block update from proceeding
					logger.Println(err)
				}
				err := agent.handleRepoUpdate(artifact, cfg)
				if err != nil {
					logger.Println(err)
					updateCondition.Error = err.Error()
					updateCondition.Status = config.StatusErr
					err = agent.publishUpdateCondition(updateCondition)
					if err != nil {
						logger.Println(err)
					}
					return
				}
				// TODO: should check systemctl status before sending success?
				updateCondition.Status = config.StatusSuccess
				err = agent.publishUpdateCondition(updateCondition)
				if err != nil {
					logger.Println(err)
				}
			}
		}
	})

	agent.MqttClient.Subscribe(config.ServiceActionTopic, func(message string) {
		var payload config.ServiceActionPayload
		err := json.Unmarshal([]byte(message), &payload)
		if err != nil {
			logger.Println(fmt.Sprintf("unmarshalling payload from topic %s: %s", config.ServiceActionTopic, err))
			return
		}
		for _, cfg := range deployerConfig.AppConfigs {
			if payload.RepoName == cfg.RepoName && payload.ManifestName == cfg.ManifestName {
				logger.Println(fmt.Sprintf("Running service action %s on %s/%s", payload.Action, payload.RepoName, payload.ManifestName))
				var err error
				switch payload.Action {
				case config.ServiceActionStart:
					err = file.StartSystemdUnit(payload.ManifestName)
					break
				case config.ServiceActionStop:
					err = file.StopSystemdUnit(payload.ManifestName)
					break
				case config.ServiceActionRestart:
					err = file.RestartSystemdUnit(payload.ManifestName)
					break
				default:
					err = fmt.Errorf("Action %s is not valid", payload.Action)
					break
				}
				if err != nil {
					logger.Println(err)
				}
			}
		}
	})

	go forever()
	select {} // block forever

}

func forever() {
	for {
		time.Sleep(5 * time.Minute)
	}
}
