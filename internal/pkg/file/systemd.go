package file

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
)

const (
	systemDPath = "/etc/systemd/system"
)

type SystemdTool struct {
	UnitPath string
	UnitName string
}

func NewSystemdTool(cfg config.Config) SystemdTool {
	s := SystemdTool{}
	s.UnitPath = fmt.Sprintf("%s/%s.service", systemDPath, cfg.PackageName)
	s.UnitName = fmt.Sprintf("%s.service", cfg.PackageName)
	return s
}

func (s SystemdTool) FindApiKeyFromSystemd() (string, error) {
	f, err := os.Open(s.UnitPath)
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

	// Expected Systemd env var pattern: Environment=HEROKU_API_KEY=<api-key>
	split := strings.Split(keyLineString, "=")
	if len(split) != 3 {
		return "", fmt.Errorf("expected systemd file heroku api key line to have length 3")
	}

	return split[2], nil
}

func (s SystemdTool) SetupSystemdUnits() error {
	_, err := exec.Command("systemctl", "daemon-reload").Output()
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	_, err = exec.Command("systemctl", "start", s.UnitName).Output()
	if err != nil {
		return fmt.Errorf("starting %s systemd unit: %s", s.UnitName, err)
	}

	// todo: better error handling
	startCmd := exec.Command("systemctl", "start", "pi-app-updater")
	stderr, pipeErr := startCmd.StderrPipe()
	if pipeErr != nil {
		return pipeErr
	}
	if err := startCmd.Start(); err != nil {
		return fmt.Errorf("err: %s, stderr text: %s", err, getStdErrText(stderr))
	}

	return nil
}

func (s SystemdTool) StopSystemdUnit() error {
	cmd := exec.Command("systemctl", "is-enabled", s.UnitName)
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return err
	}

	notInstalledErr := fmt.Sprintf("Failed to get unit file state for %s: No such file or directory", s.UnitName)

	// unit not intalled, it can be considered stopped
	if strings.Contains(getStdErrText(stderr), notInstalledErr) {
		return nil
	}

	_, err := exec.Command("systemctl", "stop", s.UnitName).Output()
	if err != nil {
		return fmt.Errorf("stopping systemd unit: %s", err)
	}
	return nil
}

func getStdErrText(stderr io.ReadCloser) string {
	stdErrText := ""
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		stdErrText += scanner.Text()
	}
	return stdErrText
}
