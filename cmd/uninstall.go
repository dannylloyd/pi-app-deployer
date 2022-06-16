package cmd

import (
	"os"

	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
	"github.com/spf13/cobra"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "TODO",
	Long:  `TODO`,
	Run: func(cmd *cobra.Command, args []string) {
		runUninstall(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)

	uninstallCmd.PersistentFlags().Bool("all", false, "Uninstall all apps")
	uninstallCmd.PersistentFlags().String("repoName", "", "Name of the Github repo including the owner")
	uninstallCmd.PersistentFlags().String("manifestName", "", "Name of the pi-app-deployer manifest")
}

func runUninstall(cmd *cobra.Command, args []string) {
	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		logger.Fatalf("error getting all flag: %s", err)
	}

	repoName, err := cmd.Flags().GetString("repoName")
	if err != nil {
		logger.Fatalf("error getting repoName flag: %s", err)
	}

	manifestName, err := cmd.Flags().GetString("manifestName")
	if err != nil {
		logger.Fatalf("error getting manifestName flag: %s", err)
	}

	herokuApp, err := cmd.Flags().GetString("herokuApp")
	if err != nil {
		logger.Fatalf("error getting herokuApp flag: %s", err)
	}

	if herokuApp == "" {
		logger.Fatal("herokuApp flag cannot be empty")
	}

	deployerConfig, err := config.NewDeployerConfig(config.DeployerConfigFile, herokuApp)
	if err != nil {
		logger.Fatalf("error getting deployer config: %s", err)
	}

	if all {
		logger.Info("Uninstalling all apps")
		err := unInstallAll(deployerConfig.AppConfigs)
		if err != nil {
			logger.Fatalf("Error uninstalling all apps: %s", err)
		}
		logger.Info("Successfully uninstalled all apps")
		os.Exit(0)
	}

	if repoName == "" || manifestName == "" {
		logger.Fatal("repoName and manifestName cannot be empty if not using the --all flag")
	}

	logger.Infof("Uninstalling %s/%s", repoName, manifestName)
	err = unInstall(deployerConfig.AppConfigs, repoName, manifestName)
	if err != nil {
		logger.Fatalf("Error uninstalling %s/%s: %s", repoName, manifestName, err)
	}
	logger.Infof("Successfully uninstalled %s/%s", repoName, manifestName)
}
