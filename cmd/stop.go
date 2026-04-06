package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/thelaonerd/spawn-qdrant/internal/container"
	"github.com/thelaonerd/spawn-qdrant/internal/lock"
)

var stopCmd = &cobra.Command{
	Use:   "stop [all|n]",
	Short: "Stop qdrant instances",
	Long:  `Stop a specific qdrant instance by number or all instances.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := args[0]
		if arg == "all" {
			return stopAll(cmd)
		}

		n, err := strconv.Atoi(arg)
		if err != nil {
			return fmt.Errorf("argument must be 'all' or a valid instance number")
		}
		return stopInstance(cmd, n)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func stopAll(cmd *cobra.Command) error {
	// Find all containers starting with qdrant-
	targets, err := container.ListContainerNames("qdrant-")
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(targets) == 0 {
		logInfo(cmd, "No qdrant instances found to stop.")
		return nil
	}

	for _, name := range targets {
		if err := stopAndRemove(cmd, name); err != nil {
			logInfo(cmd, "Failed to stop/remove %s: %v", name, err)
		}
	}

	// Try to remove network
	_ = container.RemoveNetwork("qdrant_network")

	return lock.Remove()
}

func stopInstance(cmd *cobra.Command, n int) error {
	name := fmt.Sprintf("qdrant-%02d", n)
	if err := stopAndRemove(cmd, name); err != nil {
		return err
	}

	// Check if any qdrant instances remain
	anyRemaining, err := container.HasRunningContainers("name=qdrant-")
	if err == nil && !anyRemaining {
		_ = container.RemoveNetwork("qdrant_network")
		_ = lock.Remove()
	}
	return nil
}

func stopAndRemove(cmd *cobra.Command, name string) error {
	logInfo(cmd, "Stopping and removing %s...", name)
	return container.StopAndRemoveContainer(name)
}
