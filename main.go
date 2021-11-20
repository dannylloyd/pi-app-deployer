package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/robfig/cron/v3"
)

const (
	defaultPollPeriodMin     = 5
	defaultTestPollPeriodSec = 2
)

type AppInfo struct {
	TagName string `json:"tag_name"`
}

type Config struct {
	RepoName     string
	PackageNames []string
}

func main() {
	repoName := flag.String("full-repo-name", "", "Name of the Github repo including the owner")
	packageNames := flag.String("package-names", "Comma separated with no spaces list of package names to install", "Binary names")
	pollPeriodMin := flag.Int64("poll-period-min", defaultPollPeriodMin, "Number of minutes between polling for new version")
	flag.Parse()

	var args = map[string]string{
		"full-repo-name": *repoName,
		"binary-names":   *packageNames,
	}
	for k, v := range args {
		if v == "" {
			log.Fatalln(fmt.Sprintf("--%s is required", k))
		}
	}

	config := Config{
		RepoName:     *repoName,
		PackageNames: []string{},
	}

	for _, v := range strings.Split(*packageNames, ",") {
		config.PackageNames = append(config.PackageNames, v)
	}

	var cronSpec string
	if os.Getenv("TEST_MODE") != "" {
		cronSpec = fmt.Sprintf("@every %ds", defaultTestPollPeriodSec)
	} else {
		cronSpec = fmt.Sprintf("@every %dm", pollPeriodMin)
	}

	var cronLib *cron.Cron
	cronLib = cron.New()
	cronLib.AddFunc(cronSpec, func() {
		err := checkForUpdates(config)
		if err != nil {
			log.Println("Error checking for updates:", err)
		}
	})
	cronLib.Start()

	go forever()
	select {} // block forever
}

func updateApp(config Config) error {
	// out, err := exec.Command(updateScript, repoName, string(latestVersion)).Output()
	// if err != nil {
	// 	return fmt.Errorf("initiating update command: %s", err)
	// }
	// err = ioutil.WriteFile("./.version", []byte(latestVersion), 0644)
	// if err != nil {
	// 	return fmt.Errorf("writing latest version to file: %s", err)
	// }
	// log.Println(string(out))
	return nil
}

func installApp(config Config, latestVersion string) error {
	fmt.Println("App install called", config.RepoName)
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", config.RepoName))
	if err != nil {
		return fmt.Errorf("requesting latest release: %s", err)
	}
	fmt.Println("*")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("**")
	var release github.RepositoryRelease
	err = json.Unmarshal(body, &release)
	fmt.Println("***")
	if err != nil {
		return fmt.Errorf("unmarshalling json: %s", err)
	}
	fmt.Println("checking packages", config.PackageNames)
	found := true
	for _, bName := range config.PackageNames {
		fmt.Println("bName", bName)
		for _, a := range release.Assets {
			fmt.Println("a.Name", *a.Name)
			expectedName := fmt.Sprintf("%s-%s-linux-arm.tar.gz", bName, latestVersion)
			fmt.Println("expectedName", expectedName)
			if expectedName == *a.Name {
				fmt.Println(fmt.Sprintf("Installing release %s", *a.Name))
				err := installRelease(*a.Name, *a.BrowserDownloadURL)
				if err != nil {
					fmt.Println("wtf", err)
					return err
				}
				fmt.Println("((((((((((((((((((((((((((((((")
				os.Exit(0)
			}
		}
	}
	if !found {
		fmt.Println("No packages matched")
	}

	// out, err := exec.Command(installScript, repoName, string(latestVersion)).Output()
	// if err != nil {
	// 	return fmt.Errorf("initiating install command with latest version: %s", err)
	// }
	// err = ioutil.WriteFile("./.version", []byte(latestVersion), 0644)
	// if err != nil {
	// 	return fmt.Errorf("writing latest version to file: %s", err)
	// }
	// log.Println(string(out))
	return nil
}

func installRelease(name string, url string) error {
	fmt.Println("Release url:", url)

	// Create the file
	out, err := os.Create(name)
	if err != nil {
		fmt.Println("creating file", err)
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("getting url", err)
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		fmt.Println("status code", err)
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println("copying", err)
		return err
	}

	return nil
}

func checkForUpdates(config Config) error {
	log.Println("Checking for updates")
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", config.RepoName))
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
			log.Println(fmt.Sprintf("Error reading current version from file: %s. Installing app now", err))
			err := installApp(config, string(latestVersion))
			if err != nil {
				return fmt.Errorf("error installing app: %s", err)
			}
			log.Println("Successfully installed app")
		} else {
			v := strings.TrimSuffix(string(version), "\n")
			if info.TagName != v {
				log.Println(fmt.Sprintf("New version available. Current version: %s, latest version: %s", v, string(latestVersion)))
				err := updateApp(config)
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
