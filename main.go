package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

type AppInfo struct {
	TagName string `json:"tag_name"`
}

const (
	interval = 2
)

func main() {
	// args should be:
	// url to check for new releases
	// absolute path for script to download new release and re-install
	// absolute path for script to install if not installed already
	args := os.Args[1:]
	releaseURL := args[0]
	updateScript := args[1]
	installScript := args[2]

	var cronLib *cron.Cron
	cronLib = cron.New()
	cronLib.AddFunc(fmt.Sprintf("@every %ds", interval), func() {
		err := checkForUpdates(releaseURL, updateScript, installScript)
		if err != nil {
			log.Println("Error checking for updates:", err)
		}
	})
	cronLib.Start()

	go forever()
	select {} // block forever
}

func updateApp(updateScript string, latestVersion string) error {
	out, err := exec.Command(updateScript, string(latestVersion)).Output()
	if err != nil {
		return fmt.Errorf("initiating update command: %s", err)
	}
	err = ioutil.WriteFile("./.version", []byte(latestVersion), 0644)
	if err != nil {
		return fmt.Errorf("writing latest version to file: %s", err)
	}
	log.Println(string(out))
	return nil
}

func installApp(installScript string, latestVersion string) error {
	out, err := exec.Command(installScript, string(latestVersion)).Output()
	if err != nil {
		return fmt.Errorf("initiating install command with latest version: %s", err)
	}
	err = ioutil.WriteFile("./.version", []byte(latestVersion), 0644)
	if err != nil {
		return fmt.Errorf("writing latest version to file: %s", err)
	}
	log.Println(string(out))
	return nil
}

func checkForUpdates(releaseURL string, updateScript string, installScript string) error {
	log.Println("Checking for updates")
	resp, err := http.Get(releaseURL)
	if err != nil {
		return err
	}
	var info AppInfo
	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		return fmt.Errorf("parsing version from api response: %s", err)
	} else {
		latestVersion := []byte(info.TagName)
		version, err := ioutil.ReadFile("./.version")
		if err != nil {
			log.Println(fmt.Sprintf("Error reading current version from file: %s. Installing app now using install script", err))
			err := installApp(installScript, string(latestVersion))
			if err != nil {
				return fmt.Errorf("installing app: %s", err)
			}
			log.Println("Successfully installed app")
		} else {
			v := strings.TrimSuffix(string(version), "\n")
			if info.TagName != v {
				log.Println(fmt.Sprintf("New version available. Current version: %s, latest version: %s", v, string(latestVersion)))
				err := updateApp(updateScript, string(latestVersion))
				if err != nil {
					return fmt.Errorf("updating app: %s", err)
				}
				log.Println("Successfully installed app")
			} else {
				log.Println("App already up to date")
			}
		}

	}
	return nil
}

func forever() {
	for {
		time.Sleep(5 * time.Minute)
	}
}
