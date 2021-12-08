package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

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
			log.Println(fmt.Errorf("checking if app is installed already: %s", err))
			os.Exit(1)
		}
		if installed {
			log.Println("App already installed, remove '--install' flag to check for updates")
			os.Exit(1)
		}

		latest, err := ghClient.GetLatestVersion(cfg)
		if err != nil {
			log.Println(fmt.Sprintf("error getting latest version from github: %s", err))
			os.Exit(1)
		}

		err = installRelease(cfg, latest.AssetDownloadURL, sdTool)
		if err != nil {
			log.Println(fmt.Errorf("error installing app: %s", err))
			os.Exit(1)
		}
		err = vTool.WriteCurrentVersion(latest.Version)
		if err != nil {
			log.Println(fmt.Errorf("writing latest version to file: %s", err))
			os.Exit(1)
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

	m, err := file.GetManifest(fmt.Sprintf("%s/.pi-app-updater.yaml", dlDir))
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

	hClient, err := heroku.NewClient(m.Heroku.App, apiKey)
	if err != nil {
		return err
	}
	envVars, err := hClient.GetEnv()
	if err != nil {
		return fmt.Errorf("getting env from heroku: %s", err)
	}

	data := file.TemplateData{
		Name:          cfg.PackageName,
		Keys:          make([]string, 0),
		NewLine:       "\n",
		HerokuAPIKey:  apiKey,
		HerokuAppName: m.Heroku.App,
		RepoName:      cfg.RepoName,
		PackageName:   cfg.PackageName,
	}

	for _, v := range m.Heroku.Env {
		if envVars[v] != "" {
			data.Keys = append(data.Keys, v)
		}
	}

	runScriptFile := fmt.Sprintf("run-%s.sh", cfg.PackageName)
	runScriptOutputPath := fmt.Sprintf("%s/%s", dlDir, runScriptFile)
	err = file.EvalRunScriptTemplate(runScriptOutputPath, data)
	if err != nil {
		return err
	}

	serviceFile := fmt.Sprintf("%s.service", cfg.PackageName)
	serviceFileOutputPath := fmt.Sprintf("%s/%s", dlDir, serviceFile)
	err = file.EvalServiceTemplate(serviceFileOutputPath, data)
	if err != nil {
		return err
	}

	updaterServiceFileOutputPath := fmt.Sprintf("%s/%s", dlDir, "pi-app-updater.service")
	err = file.EvalUpdaterTemplate(updaterServiceFileOutputPath, data)
	if err != nil {
		return err
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
		updaterServiceFileOutputPath:                 "/etc/systemd/system/pi-app-updater.service",
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
