package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var force bool

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Stop instances, backup storage, and clean up",
	Long: `Stops all qdrant instances, creates a backup of storage in ~/qdrant_backup, 
and deletes the storage directories using sudo.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Interactive prompt for destructive action
		if !viper.GetBool("force") && isatty(os.Stdin) {
			fmt.Fprintf(cmd.OutOrStdout(), "WARNING: This will stop all instances and delete storage directories (with backup).\nAre you sure you want to continue? [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			response = strings.ToLower(strings.TrimSpace(response))
			if response != "y" && response != "yes" {
				logInfo(cmd, "Clean operation cancelled.")
				return nil
			}
		}

		// 1. Stop all instances
		logInfo(cmd, "Stopping all instances...")
		if err := stopAll(cmd); err != nil { // reusing stopAll from stop.go
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

		validatedMatches := filterStorageDirs(cmd, matches)

		if len(validatedMatches) == 0 {
			logInfo(cmd, "No valid storage directories found to clean.")
			return nil
		}

		logInfo(cmd, "Backing up storage to %s...", backupFile)
		logInfo(cmd, "Note: This operation uses 'sudo' to access files owned by root; you may be prompted for your password.")

		// Construct tar args
		tarArgs := []string{"tar", "-czf", backupFile, "--"}
		tarArgs = append(tarArgs, validatedMatches...)

		// IMPORTANT: tar backup usually requires sudo if files are owned by root
		// We use a timeout to prevent hanging in non-interactive environments
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		tarCmd := exec.CommandContext(ctx, "sudo", tarArgs...)
		tarCmd.Stdout = cmd.OutOrStdout()
		tarCmd.Stderr = cmd.ErrOrStderr()
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
		logInfo(cmd, "Deleting storage directories with sudo...")
		rmArgs := []string{"rm", "-rf", "--"}
		rmArgs = append(rmArgs, validatedMatches...)

		rmCtx, rmCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer rmCancel()

		rmCmd := exec.CommandContext(rmCtx, "sudo", rmArgs...)
		rmCmd.Stdout = cmd.OutOrStdout()
		rmCmd.Stderr = cmd.ErrOrStderr()
		if isatty(os.Stdin) {
			rmCmd.Stdin = os.Stdin
		}

		if err := rmCmd.Run(); err != nil {
			if rmCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("cleanup timed out (sudo might be waiting for a password in a non-interactive environment)")
			}
			return fmt.Errorf("failed to delete storage: %w", err)
		}

		logInfo(cmd, "Clean up completed successfully.")
		return nil
	},
}

// isatty performs a robust check to see if the file is a terminal using ioctl
func isatty(f *os.File) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}

func filterStorageDirs(cmd *cobra.Command, matches []string) []string {
	var validated []string
	for _, match := range matches {
		info, err := os.Lstat(match)
		if err != nil {
			logInfo(cmd, "Warning: could not stat %s, skipping: %v", match, err)
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			logInfo(cmd, "Warning: %s is a symbolic link, skipping for security reasons", match)
			continue
		}
		if !info.IsDir() {
			logInfo(cmd, "Warning: %s is not a directory, skipping", match)
			continue
		}
		validated = append(validated, match)
	}
	return validated
}

func init() {
	cleanCmd.Flags().BoolVarP(&force, "force", "f", false, "Force clean without prompting")
	viper.BindPFlag("force", cleanCmd.Flags().Lookup("force"))
	rootCmd.AddCommand(cleanCmd)
}
