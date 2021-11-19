package main

import (
	"encoding/json"
	"flag"
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

func main() {

	fullRepoName := flag.String("full-repo-name", "", "Name of the Github repo including the owner")
	updateScript := flag.String("update-script", "", "Absolute path of the script or executable that is responsible for running the update actions ")
	installScript := flag.String("install-script", "", "Absolute path of the script or executable that is responsible for running the installing actions")
	pollPeriodMin := flag.Int64("poll-period-min", 5, "Number of minutes between polling for new version")
	flag.Parse()

	var args = map[string]string{
		"full-repo-name": *fullRepoName,
		"update-script":  *updateScript,
		"install-script": *installScript,
	}
	for k, v := range args {
		if v == "" {
			log.Fatalln(fmt.Sprintf("--%s is required", k))
		}
	}

	var cronSpec string
	if os.Getenv("TEST_MODE") != "" {
		cronSpec = fmt.Sprintf("@every 2s")
	} else {
		cronSpec = fmt.Sprintf("@every %dm", pollPeriodMin)
	}

	var cronLib *cron.Cron
	cronLib = cron.New()
	cronLib.AddFunc(cronSpec, func() {
		err := checkForUpdates(*fullRepoName, *updateScript, *installScript)
		if err != nil {
			log.Println("Error checking for updates:", err)
		}
	})
	cronLib.Start()

	go forever()
	select {} // block forever
}

func updateApp(fullRepoName string, updateScript string, latestVersion string) error {
	out, err := exec.Command(updateScript, fullRepoName, string(latestVersion)).Output()
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

func installApp(fullRepoName string, installScript string, latestVersion string) error {
	out, err := exec.Command(installScript, fullRepoName, string(latestVersion)).Output()
	if err != nil {
		return fmt.Errorf("initiating install command with latest version: %s", err)
	}
	// err = ioutil.WriteFile("./.version", []byte(latestVersion), 0644)
	// if err != nil {
	// 	return fmt.Errorf("writing latest version to file: %s", err)
	// }
	log.Println(string(out))
	return nil
}

func checkForUpdates(fullRepoName string, updateScript string, installScript string) error {
	log.Println("Checking for updates")
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", fullRepoName))
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
			log.Println(fmt.Sprintf("Error reading current version from file: %s. Installing app now using install script %s", err, installScript))
			err := installApp(fullRepoName, installScript, string(latestVersion))
			if err != nil {
				return fmt.Errorf("installing app: %s", err)
			}
			log.Println("Successfully installed app")
		} else {
			v := strings.TrimSuffix(string(version), "\n")
			if info.TagName != v {
				log.Println(fmt.Sprintf("New version available. Current version: %s, latest version: %s", v, string(latestVersion)))
				err := updateApp(fullRepoName, updateScript, string(latestVersion))
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
