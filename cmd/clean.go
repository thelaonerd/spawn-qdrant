package cmd

import (
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
		if err := stopAll(); err != nil { // reusing stopAll from stop.go (needs to be exported or same package)
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

		// Pattern for storage directories: ~/.qdrant_storage*
		// We need to pass the actual paths to tar.
		// Since wildcard expansion happens in shell, we should probably just use `sh -c` for tar too
		// or glob manually.

		storagePattern := filepath.Join(homeDir, ".qdrant_storage*")
		matches, err := filepath.Glob(storagePattern)
		if err != nil {
			return fmt.Errorf("failed to glob storage dirs: %w", err)
		}

		if len(matches) == 0 {
			fmt.Println("No storage directories found to clean.")
			return nil
		}

		fmt.Printf("Backing up storage to %s...\n", backupFile)
		// usage: tar -czf <archive> -C <basedir> <dirs...>
		// But here dirs are absolute paths in home.
		// Simple tar command: tar -czf backup.tar.gz -C /home/user .qdrant_storage01 .qdrant_storage02 ...

		// Construct tar args
		tarArgs := []string{"-czf", backupFile}
		tarArgs = append(tarArgs, matches...)

		// IMPORTANT: tar backup usually requires sudo if files are owned by root (which they are if created by docker -v)
		// So we must use sudo for tar as well.
		tarCmd := exec.Command("sudo", append([]string{"tar"}, tarArgs...)...)
		tarCmd.Stdout = os.Stdout
		tarCmd.Stderr = os.Stderr
		tarCmd.Stdin = os.Stdin // Allow sudo password input
		if err := tarCmd.Run(); err != nil {
			return fmt.Errorf("failed to backup storage: %w", err)
		}

		// 3. Delete storage
		fmt.Println("Deleting storage directories with sudo...")
		rmArgs := append([]string{"rm", "-rf"}, matches...)
		rmCmd := exec.Command("sudo", rmArgs...)
		rmCmd.Stdout = os.Stdout
		rmCmd.Stderr = os.Stderr
		rmCmd.Stdin = os.Stdin
		if err := rmCmd.Run(); err != nil {
			return fmt.Errorf("failed to delete storage: %w", err)
		}

		fmt.Println("Clean up completed successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
