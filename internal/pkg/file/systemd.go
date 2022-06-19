package file

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const (
	systemDPath = "/etc/systemd/system"
)

type Syslog struct {
	Identifier string `json:"SYSLOG_IDENTIFIER"`
	Message    string `json:"MESSAGE"`
	Error      error
}

func SetupSystemdUnits(unitName string) error {
	output, err := runSystemctlCommand("daemon-reload")
	if err != nil {
		return fmt.Errorf("running daemon-reload: %s, %s", err, output)
	}

	output, err = runSystemctlCommand("start", unitName)
	if err != nil {
		return fmt.Errorf("starting %s systemd unit: %s, %s", unitName, err, output)
	}

	output, err = runSystemctlCommand("enable", unitName)
	if err != nil {
		return fmt.Errorf("enabling %s systemd unit: %s, %s", unitName, err, output)
	}

	output, err = runSystemctlCommand("start", "pi-app-deployer-agent")
	if err != nil {
		return fmt.Errorf("starting pi-app-deployer-agent systemd unit: %s, %s", err, output)
	}

	output, err = runSystemctlCommand("enable", "pi-app-deployer-agent")
	if err != nil {
		return fmt.Errorf("enabling pi-app-deployer-agent systemd unit: %s, %s", err, output)
	}

	return nil
}

func StopSystemdUnit(unitName string) error {
	output, err := runSystemctlCommand("stop", unitName)
	if err != nil {
		notLoadedErr := fmt.Sprintf("Failed to stop %s.service: Unit %s.service not loaded.\n", unitName, unitName)
		if output == notLoadedErr {
			return nil
		}
		return fmt.Errorf("stopping systemd unit: %s: %s", err, output)
	}
	return nil
}

func StartSystemdUnit(unitName string) error {
	output, err := runSystemctlCommand("start", unitName)
	if err != nil {
		return fmt.Errorf("stopping systemd unit: %s: %s", err, output)
	}
	return nil
}

func RestartSystemdUnit(unitName string) error {
	output, err := runSystemctlCommand("restart", unitName)
	if err != nil {
		return fmt.Errorf("stopping systemd unit: %s: %s", err, output)
	}
	return nil
}

func SystemdUnitEnabled(unitName string) (bool, error) {
	output, err := runSystemctlCommand("is-enabled", unitName)
	if err != nil {
		notInstalledErr := fmt.Sprintf("Failed to get unit file state for %s.service: No such file or directory\n", unitName)
		if output == notInstalledErr {
			return false, nil
		}
	}

	if output == "enabled\n" {
		return true, nil
	}

	return false, nil
}

func DaemonReload() error {
	output, err := runSystemctlCommand("daemon-reload")
	if err != nil {
		return fmt.Errorf("running daemon-reload: %s, %s", err, output)
	}
	return nil
}

func runSystemctlCommand(args ...string) (string, error) {
	cmd := exec.Command("systemctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

func TailSystemdLogs(systemdUnit string, ch chan Syslog) error {
	cmd := exec.Command("journalctl", "-u", systemdUnit, "-f", "-n 0", "--output", "json")
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating command stdout pipe: %s", err)
	}

	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			var s Syslog
			if err := json.Unmarshal([]byte(scanner.Text()), &s); err != nil {
				s.Error = fmt.Errorf("unmarshalling log: %s, original log text: %s", err, scanner.Text())
				ch <- s
				break
			}
			if s.Message != "" && s.Identifier != "systemd" && !strings.Contains(s.Message, "Logs begin at") {
				ch <- s
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting command: %s", err)
	}

	if err := cmd.Wait(); err != nil {
		close(ch)
		return fmt.Errorf("waiting for command: %s", err)
	}

	return nil
}
