package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/google/go-github/github"
	"github.com/robfig/cron/v3"
)

const (
	defaultPollPeriodMin     = 5
	defaultTestPollPeriodSec = 5
	systemDPath              = "/etc/systemd/system"
	piUserHomeDir            = "/home/pi"
	progressFile             = "/tmp/.pi-app-updater.inprogress"
	versionFile              = "/home/pi/.version"
)

//go:embed templates/run.tmpl
var runScriptTemplate string

//go:embed templates/service.tmpl
var serviceTemplate string

var testMode string

type AppInfo struct {
	TagName string `json:"tag_name"`
}

type Manifest struct {
	Name string   `yaml:"name"`
	Env  []string `yaml:"env"`
}

type Config struct {
	RepoName     string
	PackageNames []string
}

func main() {
	setUpdateInProgress(false)
	repoName := flag.String("repo-name", "", "Name of the Github repo including the owner")
	packageNames := flag.String("package-names", "", "Comma separated with no spaces list of package names to install")
	pollPeriodMin := flag.Int64("poll-period-min", defaultPollPeriodMin, "Number of minutes between polling for new version")
	install := flag.Bool("install", false, "First time install of the application. Will not trigger checking for updates")
	flag.Parse()

	testMode = os.Getenv("TEST_MODE")

	var stringArgs = map[string]string{
		"repo-name":    *repoName,
		"binary-names": *packageNames,
	}
	for k, v := range stringArgs {
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

		latestVersion, err := getLatestVersion(config)
		if err != nil {
			log.Println(fmt.Sprintf("error getting latest version: %s", err))
			os.Exit(1)
		}
		err = installApp(config, latestVersion)
		if err != nil {
			log.Println(fmt.Errorf("error installing app: %s", err))
			os.Exit(1)
		}
		err = writeCurrentVersion(latestVersion)
		if err != nil {
			log.Println(fmt.Errorf("writing latest version to file: %s", err))
			os.Exit(1)
		}
		log.Println("Successfully installed app")

	} else {
		var cronSpec string
		if testMode != "" {
			cronSpec = fmt.Sprintf("@every %ds", defaultTestPollPeriodSec)
		} else {
			cronSpec = fmt.Sprintf("@every %dm", pollPeriodMin)
		}

		var cronLib *cron.Cron
		cronLib = cron.New()
		cronLib.AddFunc(cronSpec, func() {
			if !updateInProgress() {
				setUpdateInProgress(true)
				err := checkForUpdates(config)
				if err != nil {
					log.Println("Error checking for updates:", err)
				}
				setUpdateInProgress(false)
			}
		})
		cronLib.Start()

		go forever()
		select {} // block forever
	}
}

func updateInProgress() bool {
	_, err := os.Stat(progressFile)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func setUpdateInProgress(inProgress bool) error {
	if inProgress {
		f, err := os.Create(progressFile)
		if err != nil {
			return err
		}
		defer f.Close()
	} else {
		err := os.Remove(progressFile)
		if err != nil {
			return err
		}
	}
	return nil
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

func updateApp(config Config, latestVersion string) error {
	// TODO: new releases might have new env vars, how do we handle this but be automated? Need some kind of secrets management?
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
		return fmt.Errorf("no packages found")
	}
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
		Name        string
		Description string
		Keys        []string
		Map         map[string]string
		NewLine     string
	}

	s := srvData{
		Name:        m.Name,
		Description: m.Name,
		Keys:        make([]string, 0),
		Map:         make(map[string]string, 0),
		NewLine:     "\n",
	}

	for _, v := range m.Env {
		s.Keys = append(s.Keys, v)
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(fmt.Sprintf("Enter value for %s: ", v))
		text, _ := reader.ReadString('\n')
		s.Map[v] = strings.TrimSuffix(string(text), "\n")
	}

	serviceFile := fmt.Sprintf("%s.service", m.Name)
	serviceFileOutputPath := fmt.Sprintf("%s/%s", syncDir, serviceFile)
	err = evalTemplate(serviceTemplate, serviceFileOutputPath, s)
	if err != nil {
		return fmt.Errorf("evaluating service template: %s", err)
	}

	runScriptFile := fmt.Sprintf("run-%s.sh", m.Name)
	runScriptOutputPath := fmt.Sprintf("%s/%s", syncDir, runScriptFile)
	err = evalTemplate(runScriptTemplate, runScriptOutputPath, s)
	if err != nil {
		return fmt.Errorf("evaluating run script template: %s", err)
	}

	err = os.Chmod(runScriptOutputPath, 755)
	if err != nil {
		return fmt.Errorf("changing file mode for %s: %s", runScriptOutputPath, err)
	}

	err = copyWithOwnership(serviceFileOutputPath, fmt.Sprintf("%s/%s", systemDPath, serviceFile))
	if err != nil {
		return err
	}

	err = copyWithOwnership(runScriptOutputPath, fmt.Sprintf("%s/%s", piUserHomeDir, runScriptFile))
	if err != nil {
		return err
	}

	err = copyWithOwnership(fmt.Sprintf("%s/%s", syncDir, s.Name), fmt.Sprintf("%s/%s", piUserHomeDir, s.Name))
	if err != nil {
		return err
	}

	_, err = exec.Command("systemctl", "daemon-reload").Output()
	if err != nil {
		return err
	}

	_, err = exec.Command("systemctl", "start", serviceFile).Output()
	if err != nil {
		return err
	}

	err = os.RemoveAll(syncDir)
	if err != nil {
		return err
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

func getLatestVersion(config Config) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", config.RepoName), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", os.Getenv("GITHUB_TOKEN")))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	var info AppInfo
	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		return "", fmt.Errorf("parsing version from api response: %s", err)
	}
	if info.TagName == "" {
		return "", fmt.Errorf("empty tag name from api response: %s", info)
	}
	latestVersion := []byte(info.TagName)
	return string(latestVersion), nil
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
	err = os.Chown(versionFile, 1000, 1000)
	if err != nil {
		return err
	}
	return nil
}

func checkForUpdates(config Config) error {
	log.Println("Checking for updates")
	latestVersion, err := getLatestVersion(config)
	if err != nil {
		return fmt.Errorf("getting latest version: %s", err)
	}
	currentVersion, err := getCurrentVersion()
	if err != nil {
		return fmt.Errorf("getting current version: %s", err)
	}
	if latestVersion != currentVersion {
		log.Println(fmt.Sprintf("New version available. Current version: %s, latest version: %s", currentVersion, latestVersion))
		err := updateApp(config, latestVersion)
		if err != nil {
			return fmt.Errorf("updating app: %s", err)
		}
		log.Println("Successfully updated app")
	} else {
		log.Println("App already up to date")
	}

	return nil
}

func forever() {
	for {
		time.Sleep(5 * time.Minute)
	}
}
