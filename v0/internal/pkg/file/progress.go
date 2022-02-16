package file

import "os"

const (
	progressFile = "/tmp/.pi-app-updater.inprogress"
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

func UpdateInProgress() bool {
	_, err := os.Stat(progressFile)
	if os.IsNotExist(err) {
		return false
	}
	return true
}
