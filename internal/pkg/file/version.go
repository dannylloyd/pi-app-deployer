package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type VersionTool struct {
}

func NewVersionTool() VersionTool {
	v := VersionTool{}

	return v
}

func (v VersionTool) AppInstalled(serviceName string) (bool, string, error) {
	currentVersion, err := v.GetCurrentVersion(serviceName)
	if err == nil && currentVersion != "" {
		return true, currentVersion, nil
	}
	if err != nil && fmt.Sprintf("reading current version from file: open %s: no such file or directory", getServiceFilePath(serviceName)) != err.Error() {
		return false, "", nil
	}
	return false, "", nil
}

func (v VersionTool) GetCurrentVersion(serviceName string) (string, error) {
	currentVersionBytes, err := ioutil.ReadFile(getServiceFilePath(serviceName))
	if err != nil {
		return "", fmt.Errorf("reading current version from file: %s", err)
	}
	return strings.TrimSuffix(string(currentVersionBytes), "\n"), nil
}

func (v VersionTool) WriteCurrentVersion(serviceName, version string) error {
	err := ioutil.WriteFile(getServiceFilePath(serviceName), []byte(version), 0644)
	if err != nil {
		return err
	}
	err = os.Chown(getServiceFilePath(serviceName), 1000, 1000)
	if err != nil {
		return err
	}

	return nil
}

func (v VersionTool) Cleanup(serviceName string) error {
	err := os.Remove(getServiceFilePath(serviceName))
	if err != nil && fmt.Sprintf("remove %s: no such file or directory", getServiceFilePath(serviceName)) != err.Error() {
		return err
	}
	return nil
}

func getServiceFilePath(serviceName string) string {
	return fmt.Sprintf("/home/pi/.%s.version", serviceName)
}
