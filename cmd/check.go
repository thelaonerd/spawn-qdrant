package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/thelaonerd/spawn-qdrant/internal/system"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check how many Qdrant instances can be run based on available RAM",
	Long:  `Validates the system RAM and reports the maximum number of Qdrant instances that can be run for both startup (256MB) and efficient operation (512MB).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ramMB, err := system.GetAvailableRAM()
		if err != nil {
			return fmt.Errorf("failed to get available RAM: %w", err)
		}

		maxStartup, maxEfficient := system.EstimateInstances(ramMB)

		fmt.Printf("Available RAM: %d MB\n", ramMB)
		fmt.Printf("Max instances (startup only, 256MB/each): %d\n", maxStartup)
		fmt.Printf("Max efficient instances (vector ops, 512MB/each): %d\n", maxEfficient)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
