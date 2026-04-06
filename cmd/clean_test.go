package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilterStorageDirs(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "spawn-qdrant-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", tmpDir)
	}
	defer os.RemoveAll(tmpDir)

	// Setup files
	validDir := filepath.Join(tmpDir, "valid_dir")
	if err := os.Mkdir(validDir, 0755); err != nil {
		t.Fatalf("failed to create valid dir: %v", err)
	}

	regularFile := filepath.Join(tmpDir, "regular_file")
	if err := os.WriteFile(regularFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create regular file: %v", err)
	}

	symlink := filepath.Join(tmpDir, "symlink")
	if err := os.Symlink(validDir, symlink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	inputs := []string{validDir, regularFile, symlink, filepath.Join(tmpDir, "non_existent")}
	
	validated := filterStorageDirs(cleanCmd, inputs)

	if len(validated) != 1 {
		t.Errorf("expected 1 validated match, got %d: %v", len(validated), validated)
	} else if validated[0] != validDir {
		t.Errorf("expected %s to be validated, got %s", validDir, validated[0])
	}
}
