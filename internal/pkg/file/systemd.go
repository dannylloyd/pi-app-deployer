package file

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
)

const (
	systemDPath = "/etc/systemd/system"
)

type SystemdTool struct {
}

func NewSystemdTool(cfg config.Config) SystemdTool {
	s := SystemdTool{}
	return s
}

func (s SystemdTool) SetupSystemdUnits(unitName string) error {
	_, err := exec.Command("systemctl", "daemon-reload").Output()
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	_, err = exec.Command("systemctl", "start", unitName).Output()
	if err != nil {
		return fmt.Errorf("starting %s systemd unit: %s", unitName, err)
	}

	// todo: better error handling
	startCmd := exec.Command("systemctl", "start", "pi-app-updater-agent")
	stderr, pipeErr := startCmd.StderrPipe()
	if pipeErr != nil {
		return pipeErr
	}
	if err := startCmd.Start(); err != nil {
		return fmt.Errorf("err: %s, stderr text: %s", err, getStdErrText(stderr))
	}

	return nil
}

func (s SystemdTool) StopSystemdUnit(unitName string) error {
	cmd := exec.Command("systemctl", "is-enabled", unitName)
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return err
	}

	notInstalledErr := fmt.Sprintf("Failed to get unit file state for %s: No such file or directory", unitName)

	// unit not intalled, it can be considered stopped
	if strings.Contains(getStdErrText(stderr), notInstalledErr) {
		return nil
	}

	_, err := exec.Command("systemctl", "stop", unitName).Output()
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
