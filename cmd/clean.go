package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Stop instances, backup storage, and clean up",
	Long: `Stops all qdrant instances, creates a backup of storage in ~/qdrant_backup, 
and deletes the storage directories using sudo.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Stop all instances
		fmt.Println("Stopping all instances...")
		if err := stopAll(); err != nil { // reusing stopAll from stop.go
			return fmt.Errorf("failed to stop instances: %w", err)
		}

		currentUser, err := user.Current()
		if err != nil {
			return err
		}
		homeDir := currentUser.HomeDir
		backupBaseDir := filepath.Join(homeDir, "qdrant_backup")

		// 2. Create backup
		if err := os.MkdirAll(backupBaseDir, 0755); err != nil {
			return fmt.Errorf("failed to create backup dir: %w", err)
		}

		timestamp := time.Now().Format("20060102_150405")
		backupFile := filepath.Join(backupBaseDir, fmt.Sprintf("backup_%s.tar.gz", timestamp))

		storagePattern := filepath.Join(homeDir, ".qdrant_storage*")
		matches, err := filepath.Glob(storagePattern)
		if err != nil {
			return fmt.Errorf("failed to glob storage dirs: %w", err)
		}

		validatedMatches := filterStorageDirs(matches)

		if len(validatedMatches) == 0 {
			fmt.Println("No valid storage directories found to clean.")
			return nil
		}

		fmt.Printf("Backing up storage to %s...\n", backupFile)
		fmt.Println("Note: This operation uses 'sudo' to access files owned by root; you may be prompted for your password.")

		// Construct tar args
		tarArgs := []string{"tar", "-czf", backupFile, "--"}
		tarArgs = append(tarArgs, validatedMatches...)

		// IMPORTANT: tar backup usually requires sudo if files are owned by root
		// We use a timeout to prevent hanging in non-interactive environments
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		tarCmd := exec.CommandContext(ctx, "sudo", tarArgs...)
		tarCmd.Stdout = os.Stdout
		tarCmd.Stderr = os.Stderr
		// Only attach Stdin if we are in an interactive terminal
		if isatty(os.Stdin) {
			tarCmd.Stdin = os.Stdin
		}

		if err := tarCmd.Run(); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("backup timed out (sudo might be waiting for a password in a non-interactive environment)")
			}
			return fmt.Errorf("failed to backup storage: %w", err)
		}

		// 3. Delete storage
		fmt.Println("Deleting storage directories with sudo...")
		rmArgs := []string{"rm", "-rf", "--"}
		rmArgs = append(rmArgs, validatedMatches...)

		rmCtx, rmCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer rmCancel()

		rmCmd := exec.CommandContext(rmCtx, "sudo", rmArgs...)
		rmCmd.Stdout = os.Stdout
		rmCmd.Stderr = os.Stderr
		if isatty(os.Stdin) {
			rmCmd.Stdin = os.Stdin
		}

		if err := rmCmd.Run(); err != nil {
			if rmCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("cleanup timed out (sudo might be waiting for a password in a non-interactive environment)")
			}
			return fmt.Errorf("failed to delete storage: %w", err)
		}

		fmt.Println("Clean up completed successfully.")
		return nil
	},
}

// isatty performs a basic check to see if the file is a terminal
func isatty(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func filterStorageDirs(matches []string) []string {
	var validated []string
	for _, match := range matches {
		info, err := os.Lstat(match)
		if err != nil {
			fmt.Printf("Warning: could not stat %s, skipping: %v\n", match, err)
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			fmt.Printf("Warning: %s is a symbolic link, skipping for security reasons\n", match)
			continue
		}
		if !info.IsDir() {
			fmt.Printf("Warning: %s is not a directory, skipping: %s\n", match, match)
			continue
		}
		validated = append(validated, match)
	}
	return validated
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
