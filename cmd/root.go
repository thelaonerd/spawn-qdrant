package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thelaonerd/spawn-qdrant/internal/container"
)

var rootCmd = &cobra.Command{
	Use:   "spawn-qdrant",
	Short: "A tool to spawn Qdrant instances using Docker/Podman",
	Long: `spawn-qdrant is a CLI library for spawning and managing 
multiple Qdrant instances in Docker containers.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := container.InitRuntime(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
