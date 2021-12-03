package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/file"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/github"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/heroku"
	"github.com/robfig/cron/v3"
)

const (
	defaultPollPeriodMin     = 5
	defaultTestPollPeriodSec = 5
	systemDPath              = "/etc/systemd/system"
	piUserHomeDir            = "/home/pi"
	progressFile             = "/tmp/.pi-app-updater.inprogress"
	defaultVersionFile       = "/home/pi/.version"
)

//go:embed templates/run.tmpl
var runScriptTemplate string

//go:embed templates/service.tmpl
var serviceTemplate string

var testMode bool

var versionFile string

type AppInfo struct {
	TagName string `json:"tag_name"`
}

type Manifest struct {
	Name   string `yaml:"name"`
	Heroku struct {
		App string   `yaml:"app"`
		Env []string `yaml:"env"`
	} `yaml:"heroku"`
}

func main() {
	file.SetUpdateInProgress(false)
	repoName := flag.String("repo-name", "", "Name of the Github repo including the owner")
	packageName := flag.String("package-name", "", "Package name to install")
	pollPeriodMin := flag.Int64("poll-period-min", defaultPollPeriodMin, "Number of minutes between polling for new version")
	install := flag.Bool("install", false, "First time install of the application. Will not trigger checking for updates")
	flag.Parse()

	testMode = os.Getenv("TEST_MODE") == "true"
	versionFile = defaultVersionFile
	if testMode {
		fmt.Println("Running in test mode")
		versionFile = "./.version"
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

	ghClient := github.NewClient(cfg)

	if *install {
		currentVersion, err := getCurrentVersion()
		if err == nil && currentVersion != "" {
			log.Println(fmt.Errorf("App already installed at version %s, remove '--install' flag to check for updates", currentVersion))
			os.Exit(0)
		}
		if err != nil && fmt.Sprintf("reading current version from file: open %s: no such file or directory", versionFile) != err.Error() {
			log.Println(fmt.Errorf("getting current version: %s", err))
			os.Exit(1)
		}

		latest, err := ghClient.GetLatestVersion(cfg)
		if err != nil {
			log.Println(fmt.Sprintf("error getting latest version from github: %s", err))
			os.Exit(1)
		}

		err = installRelease(cfg.PackageName, latest.AssetDownloadURL)
		if err != nil {
			log.Println(fmt.Errorf("error installing app: %s", err))
			os.Exit(1)
		}
		err = writeCurrentVersion(latest.Version)
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
			cronSpec = fmt.Sprintf("@every %dm", pollPeriodMin)
		}

		var cronLib *cron.Cron
		cronLib = cron.New()
		cronLib.AddFunc(cronSpec, func() {
			if !file.UpdateInProgress() {
				file.SetUpdateInProgress(true)
				err := checkForUpdates(ghClient, cfg)
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

func getManifest(path string) (Manifest, error) {
	var m Manifest
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return m, fmt.Errorf("reading manifest yaml file: %s ", err)
	}
	err = yaml.Unmarshal(yamlFile, &m)
	if err != nil {
		return m, fmt.Errorf("unmarshalling manifest yaml file: %s ", err)
	}
	return m, nil
}

func findApiKeyFromSystemd(cfg config.Config) (string, error) {
	var path string
	if testMode {
		path = fmt.Sprintf("/tmp/%s/%s.service", cfg.PackageName, cfg.PackageName)
	} else {
		path = fmt.Sprintf("%s/%s.service", systemDPath, cfg.PackageName)
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var keyLineString string
	scanner := bufio.NewScanner(f)
	line := 1
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "HEROKU_API_KEY") {
			keyLineString = scanner.Text()
			break
		}
		line++
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	split := strings.Split(keyLineString, "=")
	if len(split) != 3 {
		return "", fmt.Errorf("expected systemd file heroku api key line to have length 3")
	}

	return split[2], nil
}

func installRelease(packageName string, url string) error {
	fmt.Println(fmt.Sprintf("Downloading latest release of %s", packageName))
	syncDir := fmt.Sprintf("/tmp/%s", packageName)
	err := os.RemoveAll(syncDir)
	if err != nil {
		return fmt.Errorf("removing download directory: %s", err)
	}
	err = os.Mkdir(syncDir, 0755)
	if err != nil {
		return fmt.Errorf("creating download directory: %s", err)
	}

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

	m, err := getManifest(fmt.Sprintf("%s/.pi-app-updater.yaml", syncDir))
	if err != nil {
		return fmt.Errorf("getting manifest: %s", err)
	}

	type srvData struct {
		Name          string
		Description   string
		Keys          []string
		Map           map[string]string
		NewLine       string
		HerokuAPIKey  string
		HerokuAppName string
		BinaryName    string
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

	s := srvData{
		Name:          packageName,
		Description:   packageName,
		Keys:          make([]string, 0),
		Map:           make(map[string]string, 0),
		NewLine:       "\n",
		HerokuAPIKey:  apiKey,
		HerokuAppName: m.Heroku.App,
		BinaryName:    packageName,
	}

	for _, v := range m.Heroku.Env {
		if envVars[v] != "" {
			s.Keys = append(s.Keys, v)
		}
	}

	serviceFile := fmt.Sprintf("%s.service", packageName)
	serviceFileOutputPath := fmt.Sprintf("%s/%s", syncDir, serviceFile)
	err = evalTemplate(serviceTemplate, serviceFileOutputPath, s)
	if err != nil {
		return fmt.Errorf("evaluating service template: %s", err)
	}

	runScriptFile := fmt.Sprintf("run-%s.sh", packageName)
	runScriptOutputPath := fmt.Sprintf("%s/%s", syncDir, runScriptFile)
	err = evalTemplate(runScriptTemplate, runScriptOutputPath, s)
	if err != nil {
		return fmt.Errorf("evaluating run script template: %s", err)
	}

	if testMode {
		fmt.Println("Test mode, not moving files")
		return nil
	}

	err = os.Chmod(runScriptOutputPath, 0755)
	if err != nil {
		return fmt.Errorf("changing file mode for %s: %s", runScriptOutputPath, err)
	}

	err = copyWithOwnership(serviceFileOutputPath, fmt.Sprintf("%s/%s", systemDPath, serviceFile))
	if err != nil {
		return fmt.Errorf("copying service file to systemd path: %s", err)
	}

	err = copyWithOwnership(runScriptOutputPath, fmt.Sprintf("%s/%s", piUserHomeDir, runScriptFile))
	if err != nil {
		return fmt.Errorf("copying run script file to home dir: %s", err)
	}

	packageBinaryOutputPath := fmt.Sprintf("%s/%s", piUserHomeDir, packageName)
	err = copyWithOwnership(fmt.Sprintf("%s/%s", syncDir, packageName), packageBinaryOutputPath)
	if err != nil {
		return fmt.Errorf("copying binary package file to home dir: %s", err)
	}
	err = os.Chmod(packageBinaryOutputPath, 0755)
	if err != nil {
		return fmt.Errorf("changing file mode for %s: %s", packageBinaryOutputPath, err)
	}

	// sudo systemctl enable pi-sensor.service
	_, err = exec.Command("systemctl", "daemon-reload").Output()
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	_, err = exec.Command("systemctl", "start", serviceFile).Output()
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	err = os.RemoveAll(syncDir)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	return nil
}

func copyWithOwnership(src, dest string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}

	err = os.Chown(dest, 1000, 1000)
	if err != nil {
		return err
	}
	return nil
}

func evalTemplate(templateFile string, outputPath string, i interface{}) error {
	t, err := template.New("").Delims("<<", ">>").Parse(templateFile)
	if err != nil {
		return fmt.Errorf("opening service file: %s", err)
	}

	fi, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("opening service file: %s", err)
	}
	err = t.Execute(fi, i)
	if err != nil {
		return fmt.Errorf("executing template: %s", err)
	}
	return nil
}

func getCurrentVersion() (string, error) {
	// TODO use unique name of .version file, like .pi-test.version
	currentVersionBytes, err := ioutil.ReadFile(versionFile)
	if err != nil {
		return "", fmt.Errorf("reading current version from file: %s", err)
	}
	return strings.TrimSuffix(string(currentVersionBytes), "\n"), nil
}

func writeCurrentVersion(version string) error {
	err := ioutil.WriteFile(versionFile, []byte(version), 0644)
	if err != nil {
		return err
	}
	if !testMode {
		err = os.Chown(versionFile, 1000, 1000)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkForUpdates(ghClient github.GithubClient, cfg config.Config) error {
	log.Println("Checking for updates")
	latest, err := ghClient.GetLatestVersion(cfg)
	if err != nil {
		return fmt.Errorf("getting latest version: %s", err)
	}
	currentVersion, err := getCurrentVersion()
	if err != nil {
		return fmt.Errorf("getting current version: %s", err)
	}
	if latest.Version != currentVersion {
		log.Println(fmt.Sprintf("New version available. Current version: %s, latest version: %s", currentVersion, latest.Version))

		apiKey, err := findApiKeyFromSystemd(cfg)
		if err != nil {
			return err
		}
		os.Setenv("HEROKU_API_KEY", apiKey)

		err = installRelease(cfg.PackageName, latest.AssetDownloadURL)
		if err != nil {
			return fmt.Errorf("updating app: %s", err)
		}
		err = writeCurrentVersion(latest.Version)
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
