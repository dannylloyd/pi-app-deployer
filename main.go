package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
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
	args := os.Args[1:]
	releaseURL := args[0]
	updateScript := args[1]

	var cronLib *cron.Cron
	cronLib = cron.New()
	cronLib.AddFunc(fmt.Sprintf("@every %ds", interval), func() {
		err := checkForUpdates(releaseURL, updateScript)
		if err != nil {
			fmt.Println("Error checking for updates:", err)
		}
	})
	cronLib.Start()

	go forever()
	select {} // block forever
}

func checkForUpdates(releaseURL string, updateScript string) error {
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
		err = ioutil.WriteFile("./.latestVersion", latestVersion, 0644)
		if err != nil {
			return fmt.Errorf("writing latest version to file: %s", err)
		}

		version, err := ioutil.ReadFile("./.version")
		if err != nil {
			return fmt.Errorf("reading current version from file: %s", err)
		}

		if info.TagName != string(version) {
			fmt.Println("New version available, updating now")
			out, err := exec.Command(updateScript).Output()
			if err != nil {
				return fmt.Errorf("initiating command: %s", err)
			}
			fmt.Println(string(out))
		}
	}
	return nil
}

func forever() {
	for {
		time.Sleep(5 * time.Minute)
	}
}
