package main

import (
	"bytes"
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

	"github.com/google/go-github/github"
	"github.com/robfig/cron/v3"
)

const (
	defaultPollPeriodMin     = 5
	defaultTestPollPeriodSec = 5
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
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", config.RepoName))
	if err != nil {
		return fmt.Errorf("requesting latest release: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var release github.RepositoryRelease
	err = json.Unmarshal(body, &release)

	if err != nil {
		return fmt.Errorf("unmarshalling json: %s", err)
	}
	found := true
	for _, pName := range config.PackageNames {
		for _, a := range release.Assets {
			expectedName := fmt.Sprintf("%s-%s-linux-arm.tar.gz", pName, latestVersion)
			if expectedName == *a.Name {
				log.Println(fmt.Sprintf("Installing release %s", *a.Name))
				err := installRelease(pName, *a.Name, *a.BrowserDownloadURL)
				if err != nil {
					return err
				}
			}
		}
	}
	if !found {
		log.Println("No packages matched")
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

func installRelease(packageName string, releaseName string, url string) error {
	syncDir := fmt.Sprintf("/tmp/%s", packageName)
	err := os.RemoveAll(syncDir)
	if err != nil {
		return fmt.Errorf("removing download directory: %s", err)
	}
	err = os.Mkdir(syncDir, 0755)
	if err != nil {
		return fmt.Errorf("creating download directory: %s", err)
	}

	var curlOut bytes.Buffer
	var tarOut bytes.Buffer
	curl := exec.Command("curl", "-sL", url)
	tar := exec.Command("tar", "xz", "-C", syncDir)
	tar.Stdout = &tarOut
	tar.Stdin, err = curl.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating curl stdout pipe: %s", err)
	}
	err = tar.Start()
	if err != nil {
		return fmt.Errorf("starting tar command: %s", err)
	}
	err = curl.Run()
	if err != nil {
		return fmt.Errorf("running curl command: %s", err)
	}
	err = tar.Wait()
	if err != nil {
		return fmt.Errorf("waiting on tar command: %s", err)
	}

	fmt.Println(curlOut.String())

	return nil
}

func checkForUpdates(config Config) error {
	log.Println("Checking for updates")

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", config.RepoName), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", os.Getenv("GITHUB_TOKEN")))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	var info AppInfo
	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		return fmt.Errorf("parsing version from api response: %s", err)
	}
	if info.TagName == "" {
		return fmt.Errorf("empty tag name from api response: %s", info)
	}
	latestVersion := []byte(info.TagName)
	version, err := ioutil.ReadFile("./.version")
	if err != nil {
		log.Println(fmt.Sprintf("Error reading current version from file: %s. Installing app now", err))
		err := installApp(config, string(latestVersion))
		if err != nil {
			return fmt.Errorf("error installing app: %s", err)
		}
		log.Println("Successfully installed app")
		err = ioutil.WriteFile("./.version", []byte(latestVersion), 0644)
		if err != nil {
			return fmt.Errorf("writing latest version to file: %s", err)
		}
	} else {
		v := strings.TrimSuffix(string(version), "\n")
		if info.TagName != v {
			log.Println(fmt.Sprintf("New version available. Current version: %s, latest version: %s", v, string(latestVersion)))
			err := updateApp(config)
			if err != nil {
				return fmt.Errorf("updating app: %s", err)
			}
			log.Println("Successfully updated app")
		} else {
			log.Println("App already up to date")
		}
	}
	return nil
}

func forever() {
	for {
		time.Sleep(5 * time.Minute)
	}
}
