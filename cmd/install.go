package cmd

import (
	"fmt"
	"os"

	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
	"github.com/spf13/cobra"
)

var varFlags config.EnvVarFlags

func NewInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Use the install command installs an application.",
		Long: `The pi-app-deployer-agent install command downloads
artifacts and a Manifest from Github, creates a Systemd unit,
and orchestrates updates as well as log forwarding to the
pi-app-deployer-agent.`,
		Run: func(cmd *cobra.Command, args []string) {
			runInstall(cmd, args)
		},
	}
}

func init() {
	installCmd := NewInstallCmd()
	rootCmd.AddCommand(installCmd)

	installCmd.PersistentFlags().String("repoName", "", "Name of the Github repo including the owner")
	installCmd.PersistentFlags().String("manifestName", "", "Name of the pi-app-deployer manifest")
	installCmd.PersistentFlags().Bool("logForwarding", false, "Send application logs to server")
	installCmd.PersistentFlags().String("appUser", "pi", "Name of user that will run the app service")

	installCmd.PersistentFlags().Var(&varFlags, "envVar", "List of non-secret environment variable configuration, separated by =, can pass multiple values. Example: --env-var foo=bar --env-var hello=world")
}

func runInstall(cmd *cobra.Command, args []string) {
	cfg := getConfig(cmd)
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

	deployerConfig, err := config.NewDeployerConfig(config.DeployerConfigFile, herokuApp)
	if err != nil {
		logger.Fatalln("error getting deployer config:", err)
	}

	// TODO: support updating a config?
	if deployerConfig.ConfigExists(cfg) {
		logger.Fatalln("App already exists in app configs file", config.DeployerConfigFile)
	}

	logger.Println("Installing application")
	deployerConfig.SetAppConfig(cfg)
	deployerConfig.WriteDeployerConfig()

	a := config.Artifact{
		RepoName:     cfg.RepoName,
		ManifestName: cfg.ManifestName,
	}
	err = agent.handleInstall(a, cfg)
	if err != nil {
		logger.Fatalln(fmt.Errorf("failed installation: %s", err))
	}

	logger.Println("Successfully installed app")
}

func getConfig(cmd *cobra.Command) config.Config {
	repoName, err := cmd.Flags().GetString("repoName")
	if err != nil {
		fmt.Println("error getting repoName flag", err)
		os.Exit(1)
	}
	if repoName == "" {
		fmt.Println("repoName flag is required")
		os.Exit(1)
	}

	manifestName, err := cmd.Flags().GetString("manifestName")
	if err != nil {
		fmt.Println("error getting manifestName flag", err)
		os.Exit(1)
	}
	if manifestName == "" {
		fmt.Println("manifestName flag is required")
		os.Exit(1)
	}

	appUser, err := cmd.Flags().GetString("appUser")
	logForwarding, err := cmd.Flags().GetBool("logForwarding")

	return config.Config{
		RepoName:      repoName,
		ManifestName:  manifestName,
		AppUser:       appUser,
		LogForwarding: logForwarding,
		EnvVars:       varFlags.Map,
	}
}
