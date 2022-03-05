package file

import (
	"fmt"
	"os/exec"
)

const (
	systemDPath = "/etc/systemd/system"
)

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
		notLoadedErr := fmt.Sprintf("Failed to stop %s.service: Unit pi-tes.service not loaded.\n", unitName)
		if output == notLoadedErr {
			return nil
		}
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

func runSystemctlCommand(args ...string) (string, error) {
	cmd := exec.Command("systemctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

func TailSystemdLogs(systemdUnit string, ch chan string) error {
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
