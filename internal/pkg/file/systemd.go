package file

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
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
	startCmd := exec.Command("systemctl", "start", "pi-app-deployer-agent")
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
	filename := fmt.Sprintf("/etc/systemd/system/%s.service", unitName)
	_, err := os.ReadFile(filename)
	if err != nil {
		if err.Error() == fmt.Sprintf("open %s: no such file or directory", filename) {
			// unit is not installed, it is considered stopped
			return nil
		}
		return err
	}

	_, err = exec.Command("systemctl", "stop", unitName).Output()
	if err != nil {
		return fmt.Errorf("stopping systemd unit: %s", err)
	}
	return nil
}

func (s SystemdTool) SystemdUnitEnabled(unitName string) (bool, error) {
	cmd := exec.Command("systemctl", "is-enabled", unitName)
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return false, err
	}

	notInstalledErr := fmt.Sprintf("Failed to get unit file state for %s: No such file or directory", unitName)

	if strings.Contains(getStdErrText(stderr), notInstalledErr) {
		return false, nil
	}
	return false, nil
}

func (s SystemdTool) TailSystemdLogs(systemdUnit string, ch chan string) error {
	cmd := exec.Command("journalctl", "-u", systemdUnit, "-f", "-n 0")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	buf := make([]byte, 1024)
	for {
		n, err := stdout.Read(buf)
		if err != nil {
			break
		}

		ch <- string(buf[0:n])
	}
	close(ch)
	if err := cmd.Wait(); err != nil {
		return err
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
