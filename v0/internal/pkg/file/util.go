package file

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

const (
	defaultUserID = 1000
)

func DownloadDirectory(packageName string) string {
	return fmt.Sprintf("/tmp/%s", packageName)
}

func DownloadExtract(url string, dlDir string) error {
	err := os.RemoveAll(dlDir)
	if err != nil {
		return fmt.Errorf("removing download directory: %s", err)
	}
	err = os.Mkdir(dlDir, 0755)
	if err != nil {
		return fmt.Errorf("creating download directory: %s", err)
	}

	var tarOut bytes.Buffer
	curl := exec.Command("curl", "-sL", url)
	tar := exec.Command("tar", "xz", "-C", dlDir)
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
	return nil
}

func doCopyWithOwnership(src, dest string) error {
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

	err = os.Chown(dest, defaultUserID, defaultUserID)
	if err != nil {
		return err
	}
	return nil
}

func CopyWithOwnership(srdDestMap map[string]string) error {
	for s, d := range srdDestMap {
		err := doCopyWithOwnership(s, d)
		if err != nil {
			return err
		}
	}
	return nil
}

func MakeExecutable(paths []string) error {
	for _, p := range paths {
		err := os.Chmod(p, 0755)
		if err != nil {
			return fmt.Errorf("changing file mode for %s: %s", p, err)
		}
	}
	return nil
}
