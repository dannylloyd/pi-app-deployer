package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/andrewmarklloyd/pi-app-updater/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/file"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/github"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/heroku"
	"github.com/robfig/cron/v3"
)

const (
	defaultPollPeriodMin     = 5
	defaultTestPollPeriodSec = 5
	piUserHomeDir            = "/home/pi"
)

var testMode bool

func main() {
	file.SetUpdateInProgress(false)
	repoName := flag.String("repo-name", "", "Name of the Github repo including the owner")
	packageName := flag.String("package-name", "", "Package name to install")
	pollPeriodMin := flag.Int64("poll-period-min", defaultPollPeriodMin, "Number of minutes between polling for new version")
	install := flag.Bool("install", false, "First time install of the application. Will not trigger checking for updates")
	flag.Parse()

	testMode = os.Getenv("TEST_MODE") == "true"

	if testMode {
		fmt.Println("Running in test mode")
	}

	var stringArgs = map[string]string{
		"repo-name":    *repoName,
		"package-name": *packageName,
	}
	for k, v := range stringArgs {
		if v == "" {
			log.Fatalln(fmt.Sprintf("--%s is required", k))
		}
	}

	cfg := config.Config{
		RepoName:    *repoName,
		PackageName: *packageName,
	}

	vTool := file.NewVersionTool(testMode, *packageName)
	ghClient := github.NewClient(cfg)
	sdTool := file.NewSystemdTool(testMode, cfg)

	if *install {
		installed, err := vTool.AppInstalled()
		if err != nil {
			log.Fatalln(fmt.Errorf("checking if app is installed already: %s", err))
		}
		if installed {
			log.Fatalln("App already installed, remove '--install' flag to check for updates")
		}

		latest, err := ghClient.GetLatestVersion(cfg)
		if err != nil {
			log.Fatalln(fmt.Sprintf("error getting latest version from github: %s", err))
		}

		err = installRelease(cfg, latest.AssetDownloadURL, sdTool)
		if err != nil {
			log.Fatalln(fmt.Errorf("error installing app: %s", err))
		}
		err = vTool.WriteCurrentVersion(latest.Version)
		if err != nil {
			log.Fatalln(fmt.Errorf("writing latest version to file: %s", err))
		}
		log.Println("Successfully installed app")

	} else {
		var cronSpec string
		if testMode {
			cronSpec = fmt.Sprintf("@every %ds", defaultTestPollPeriodSec)
		} else {
			cronSpec = fmt.Sprintf("@every %dm", *pollPeriodMin)
		}

		var cronLib *cron.Cron
		cronLib = cron.New()
		cronLib.AddFunc(cronSpec, func() {
			if !file.UpdateInProgress() {
				file.SetUpdateInProgress(true)
				err := checkForUpdates(ghClient, vTool, sdTool, cfg)
				if err != nil {
					log.Println("Error checking for updates:", err)
				}
				file.SetUpdateInProgress(false)
			}
		})
		cronLib.Start()

		go forever()
		select {} // block forever
	}
}

func installRelease(cfg config.Config, url string, sdTool file.SystemdTool) error {
	dlDir := file.DownloadDirectory(cfg.PackageName)
	err := file.DownloadExtract(url, dlDir)
	if err != nil {
		return fmt.Errorf("downloading and extracting release: %s", err)
	}

	m, err := manifest.GetManifest(fmt.Sprintf("%s/.pi-app-updater.yaml", dlDir))
	if err != nil {
		return fmt.Errorf("getting manifest: %s", err)
	}

	apiKeyEnv := os.Getenv("HEROKU_API_KEY")
	var apiKey string
	if apiKeyEnv == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter value for Heroku API key: ")
		apiKey, err = reader.ReadString('\n')
		if err != nil {
			return err
		}
		apiKey = strings.TrimSuffix(apiKey, "\n")
	} else {
		apiKey = apiKeyEnv
	}

	serviceFile := fmt.Sprintf("%s.service", cfg.PackageName)
	serviceFileOutputPath := fmt.Sprintf("%s/%s", dlDir, serviceFile)
	serviceFileString, err := file.EvalServiceTemplate(m, apiKey)
	if err != nil {
		return fmt.Errorf("evaluating service template: %s", err)
	}
	err = os.WriteFile(serviceFileOutputPath, []byte(serviceFileString), 0644)
	if err != nil {
		return fmt.Errorf("writing service file: %s", err)
	}

	hClient, err := heroku.NewClient(m.Heroku.App, apiKey)
	if err != nil {
		return err
	}

	runScriptFile := fmt.Sprintf("run-%s.sh", cfg.PackageName)
	runScriptOutputPath := fmt.Sprintf("%s/%s", dlDir, runScriptFile)
	runScriptFileString, err := file.EvalRunScriptTemplate(m, hClient)
	if err != nil {
		return err
	}
	err = os.WriteFile(runScriptOutputPath, []byte(runScriptFileString), 0644)
	if err != nil {
		return fmt.Errorf("writing run script: %s", err)
	}

	updaterServiceFileOutputPath := fmt.Sprintf("%s/%s", dlDir, "pi-app-updater.service")
	updaterServiceFileString, err := file.EvalUpdaterTemplate(cfg)
	if err != nil {
		return err
	}
	err = os.WriteFile(updaterServiceFileOutputPath, []byte(updaterServiceFileString), 0644)
	if err != nil {
		return fmt.Errorf("writing updater service file: %s", err)
	}

	if testMode {
		fmt.Println("Test mode, not moving files")
		return nil
	}

	packageBinaryOutputPath := fmt.Sprintf("%s/%s", piUserHomeDir, cfg.PackageName)

	var srdDestMap = map[string]string{
		serviceFileOutputPath:                        sdTool.UnitPath,
		runScriptOutputPath:                          fmt.Sprintf("%s/%s", piUserHomeDir, runScriptFile),
		fmt.Sprintf("%s/%s", dlDir, cfg.PackageName): packageBinaryOutputPath,
	}

	err = file.CopyWithOwnership(srdDestMap)
	if err != nil {
		return err
	}

	err = file.MakeExecutable([]string{runScriptOutputPath, packageBinaryOutputPath})
	if err != nil {
		return err
	}

	err = sdTool.SetupSystemdUnits()
	if err != nil {
		return err
	}

	err = os.RemoveAll(dlDir)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	return nil
}

func checkForUpdates(ghClient github.GithubClient, vTool file.VersionTool, sdTool file.SystemdTool, cfg config.Config) error {
	log.Println("Checking for updates")
	latest, err := ghClient.GetLatestVersion(cfg)
	if err != nil {
		return fmt.Errorf("getting latest version: %s", err)
	}
	currentVersion, err := vTool.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("getting current version: %s", err)
	}
	if latest.Version != currentVersion {
		log.Println(fmt.Sprintf("New version available. Current version: %s, latest version: %s", currentVersion, latest.Version))
		sdTool.StopSystemdUnit()

		apiKey, err := sdTool.FindApiKeyFromSystemd()
		if err != nil {
			return err
		}
		os.Setenv("HEROKU_API_KEY", apiKey)

		err = installRelease(cfg, latest.AssetDownloadURL, sdTool)
		if err != nil {
			return fmt.Errorf("updating app: %s", err)
		}
		err = vTool.WriteCurrentVersion(latest.Version)
		if err != nil {
			return fmt.Errorf("writing latest version to file: %s", err)
		}
		log.Println("Successfully updated app")
	} else {
		log.Println(fmt.Sprintf("App already up to date. Current version: %s, latest version: %s", currentVersion, latest.Version))
	}

	return nil
}

func forever() {
	for {
		time.Sleep(5 * time.Minute)
	}
}
