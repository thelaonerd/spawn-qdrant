package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thelaonerd/spawn-qdrant/internal/system"
)

type CheckResult struct {
	AvailableRAMMB uint64 `json:"available_ram_mb"`
	MaxStartup     uint64 `json:"max_startup"`
	MaxEfficient   uint64 `json:"max_efficient"`
}

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

		if viper.GetString("output") == "json" {
			res := CheckResult{
				AvailableRAMMB: ramMB,
				MaxStartup:     maxStartup,
				MaxEfficient:   maxEfficient,
			}
			b, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Available RAM: %d MB\n", ramMB)
		fmt.Fprintf(cmd.OutOrStdout(), "Max instances (startup only, 256MB/each): %d\n", maxStartup)
		fmt.Fprintf(cmd.OutOrStdout(), "Max efficient instances (vector ops, 512MB/each): %d\n", maxEfficient)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
