package file

import (
	"fmt"
	"io"
	"os"
)

const (
	defaultUserID = 1000
)

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
