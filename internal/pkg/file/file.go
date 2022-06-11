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
		return fmt.Errorf("opening source file: %s", err)
	}
	defer source.Close()

	destination, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("opening destination file: %s", err)
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	if err != nil {
		return fmt.Errorf("copying source to destination file: %s", err)
	}

	err = os.Chown(dest, defaultUserID, defaultUserID)
	if err != nil {
		return fmt.Errorf("changing ownership of file: %s", err)
	}
	return nil
}

func CopyWithOwnership(srcDestMap map[string]string) error {
	for s, d := range srcDestMap {
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

func MoveFile(src, dest string) error {
	if err := os.Rename(src, dest); err != nil {
		return fmt.Errorf("renaming file: %s", err)
	}
	return nil
}
