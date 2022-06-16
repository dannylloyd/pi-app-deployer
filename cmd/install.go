package cmd

import (
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
		logger.Fatal("HEROKU_API_TOKEN environment variable is required")
	}

	herokuApp, err := cmd.Flags().GetString("herokuApp")
	if err != nil {
		logger.Fatalf("error getting herokuApp flag: %s", err)
	}
	if herokuApp == "" {
		logger.Fatal("herokuApp flag is required")
	}

	agent, err := newAgent(herokuAPIKey, herokuApp)
	if err != nil {
		logger.Fatalf("error creating agent: %s", err)
	}

	deployerConfig, err := config.NewDeployerConfig(config.DeployerConfigFile, herokuApp)
	if err != nil {
		logger.Fatalf("error getting deployer config: %s", err)
	}

	// TODO: support updating a config?
	if deployerConfig.ConfigExists(cfg) {
		logger.Fatalf("App already exists in app configs file %s", config.DeployerConfigFile)
	}

	logger.Info("Installing application")
	// writing deployer config here is required since the install
	// starts the pi-app-deployer-agent systemd unit
	deployerConfig.SetAppConfig(cfg)
	deployerConfig.WriteDeployerConfig()

	a := config.Artifact{
		RepoName:     cfg.RepoName,
		ManifestName: cfg.ManifestName,
	}
	cfg, err = agent.handleInstall(a, cfg)
	if err != nil {
		logger.Fatalf("failed installation: %s", err)
	}

	// writing deployer config here is required to
	// get executable field written which is only found
	// during the install via the manifest
	deployerConfig.SetAppConfig(cfg)
	deployerConfig.WriteDeployerConfig()

	logger.Info("Successfully installed app")
}

func getConfig(cmd *cobra.Command) config.Config {
	repoName, err := cmd.Flags().GetString("repoName")
	if err != nil {
		logger.Fatalf("error getting repoName flag: %s", err)
	}
	if repoName == "" {
		logger.Fatal("repoName flag is required")
	}

	manifestName, err := cmd.Flags().GetString("manifestName")
	if err != nil {
		logger.Fatalf("error getting manifestName flag: %s", err)
	}
	if manifestName == "" {
		logger.Fatal("manifestName flag is required")
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
