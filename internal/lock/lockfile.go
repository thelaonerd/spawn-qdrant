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

	// Use O_CREATE | O_EXCL to ensure atomic creation.
	// This fails if the file already exists.
	// Permissions 0600 (read/write only by owner) for security.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("lock file %s exists. Application is likely running. Use 'stop' or 'clean' first", path)
		}
		return fmt.Errorf("failed to create lock file: %w", err)
	}
	f.Close()
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
