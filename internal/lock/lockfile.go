package lock

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

func getLockFilePath() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(currentUser.HomeDir, ".spawn-qdrant.lock"), nil
}

// Create checks if lock exists, and if not creates it.
// Returns error if lock exists.
func Create() error {
	path, err := getLockFilePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("lock file %s exists. Application is likely running. Use 'stop' or 'clean' first", path)
	}

	// Create file
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

// Remove removes the lock file.
func Remove() error {
	path, err := getLockFilePath()
	if err != nil {
		return err
	}
	// Ignore if file doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

// Exists checks if lock file exists
func Exists() (bool, error) {
	path, err := getLockFilePath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
