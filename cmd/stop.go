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
			return stopAll()
		}

		n, err := strconv.Atoi(arg)
		if err != nil {
			return fmt.Errorf("argument must be 'all' or a valid instance number")
		}
		return stopInstance(n)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func stopAll() error {
	// Find all containers starting with qdrant-
	targets, err := container.ListContainerNames("qdrant-")
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(targets) == 0 {
		fmt.Println("No qdrant instances found to stop.")
		return nil
	}

	for _, name := range targets {
		if err := stopAndRemove(name); err != nil {
			fmt.Printf("Failed to stop/remove %s: %v\n", name, err)
		}
	}

	// Try to remove network
	_ = container.RemoveNetwork("qdrant_network")

	return lock.Remove()
}

func stopInstance(n int) error {
	name := fmt.Sprintf("qdrant-%02d", n)
	if err := stopAndRemove(name); err != nil {
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

func stopAndRemove(name string) error {
	fmt.Printf("Stopping and removing %s...\n", name)
	return container.StopAndRemoveContainer(name)
}
