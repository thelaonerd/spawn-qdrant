package cmd

import (
	"fmt"
	"strconv"
	"strings"

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
	// We can use docker/podman ps -a --format "{{.Names}}" and filter
	output, err := container.RunCommandOutput("ps", "-a", "--format", "{{.Names}}")
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	lines := strings.Split(output, "\n")
	var targets []string
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if strings.HasPrefix(name, "qdrant-") {
			targets = append(targets, name)
		}
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
	// Ignore error if other containers are using it
	_ = container.RunCommand("network", "rm", "qdrant_network")

	return lock.Remove()
}

func stopInstance(n int) error {
	name := fmt.Sprintf("qdrant-%02d", n)
	if err := stopAndRemove(name); err != nil {
		return err
	}

	// Check if any qdrant instances remain
	output, err := container.RunCommandOutput("ps", "-q", "-f", "name=qdrant-")
	if err == nil && len(strings.TrimSpace(output)) == 0 {
		_ = lock.Remove()
	}
	return nil
}

func stopAndRemove(name string) error {
	fmt.Printf("Stopping and removing %s...\n", name)
	// Stop
	_ = container.RunCommand("stop", name)
	// Remove
	return container.RunCommand("rm", name)
}
