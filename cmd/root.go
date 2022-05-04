package cmd

import (
	"log"
	"os"
	"os/user"

	"github.com/spf13/cobra"
)

var logger = log.New(os.Stdout, "[pi-app-deployer-Agent] ", log.LstdFlags)

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
	u, err := user.Current()
	if err != nil {
		logger.Fatalln("error getting current user:", err)
	}
	if u.Username != "root" {
		logger.Fatalln("agent must be run as root, user found was", u.Username)
	}

	logger.Println("Version:", version)

	rootCmd.PersistentFlags().String("herokuApp", "", "Name of the Heroku app")
}
