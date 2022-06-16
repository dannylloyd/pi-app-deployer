package cmd

import (
	"log"
	"os/user"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var logger *zap.SugaredLogger
var rootCmd = &cobra.Command{
	Use:   "pi-app-deployer-agent",
	Short: "",
	Long:  ``,
}

var version string

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	l, err := zap.NewProduction()
	if err != nil {
		log.Fatalln("Error creating logger:", err)
	}
	logger = l.Sugar().Named("pi-app-deployer-agent")
	defer logger.Sync()

	u, err := user.Current()
	if err != nil {
		logger.Fatalf("error getting current user: %s", err)
	}
	if u.Username != "root" {
		logger.Fatalf("agent must be run as root, user found was %s", u.Username)
	}

	logger.Infof("Version: %s", version)

	rootCmd.PersistentFlags().String("herokuApp", "", "Name of the Heroku app")
}
