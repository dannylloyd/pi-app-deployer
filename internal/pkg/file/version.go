package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type VersionTool struct {
	TestMode    bool
	VersionFile string
}

func NewVersionTool(testMode bool, packageName string) VersionTool {
	v := VersionTool{
		TestMode: testMode,
	}
	if testMode {
		v.VersionFile = fmt.Sprintf("./.%s.version", packageName)
	} else {
		v.VersionFile = fmt.Sprintf("/home/pi/.%s.version", packageName)
	}
	return v
}

func (v VersionTool) AppInstalled() (bool, error) {
	currentVersion, err := v.GetCurrentVersion()
	if err == nil && currentVersion != "" {
		return true, nil
	}
	if err != nil && fmt.Sprintf("reading current version from file: open %s: no such file or directory", v.VersionFile) != err.Error() {
		return false, nil
	}
	return false, nil
}

func (v VersionTool) GetCurrentVersion() (string, error) {
	currentVersionBytes, err := ioutil.ReadFile(v.VersionFile)
	if err != nil {
		return "", fmt.Errorf("reading current version from file: %s", err)
	}
	return strings.TrimSuffix(string(currentVersionBytes), "\n"), nil
}

func (v VersionTool) WriteCurrentVersion(version string) error {
	err := ioutil.WriteFile(v.VersionFile, []byte(version), 0644)
	if err != nil {
		return err
	}
	if !v.TestMode {
		err = os.Chown(v.VersionFile, 1000, 1000)
		if err != nil {
			return err
		}
	}
	return nil
}
