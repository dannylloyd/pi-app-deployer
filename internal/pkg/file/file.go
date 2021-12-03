package file

import (
	"os"
)

const (
	systemDPath        = "/etc/systemd/system"
	piUserHomeDir      = "/home/pi"
	progressFile       = "/tmp/.pi-app-updater.inprogress"
	defaultVersionFile = "/home/pi/.version"
)

func SetUpdateInProgress(inProgress bool) error {
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
