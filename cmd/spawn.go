package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thelaonerd/spawn-qdrant/internal/container"
	"github.com/thelaonerd/spawn-qdrant/internal/lock"
	"github.com/thelaonerd/spawn-qdrant/internal/system"
)

type SpawnResult struct {
	AvailableRAMMB   uint64 `json:"available_ram_mb"`
	MaxStartup       uint64 `json:"max_startup"`
	MaxEfficient     uint64 `json:"max_efficient"`
	SpawnedInstances int    `json:"spawned_instances"`
	Status           string `json:"status"`
}

var spawnCmd = &cobra.Command{
	Use:   "spawn [instance_count]",
	Short: "Spawn qdrant instances",
	Long: `Spawn n instances of Qdrant.
If instance_count is not provided, it estimates the maximum instances based on available RAM.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Set up signal handling for graceful exit
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		// Acquire lock only if we actually intend to spawn (not just estimate)
		if len(args) > 0 {
			if err := lock.Create(); err != nil {
				return err
			}
		}

		// Pre-flight check: RAM
		ramMB, err := system.GetAvailableRAM()
		if err != nil {
			if len(args) > 0 {
				lock.Remove()
			}
			return fmt.Errorf("failed to get available RAM: %w", err)
		}
		maxStartup, maxEfficient := system.EstimateInstances(ramMB)

		if len(args) == 0 {
			if viper.GetString("output") == "json" {
				res := SpawnResult{
					AvailableRAMMB:   ramMB,
					MaxStartup:       maxStartup,
					MaxEfficient:     maxEfficient,
					SpawnedInstances: 0,
					Status:           "estimation",
				}
				b, _ := json.MarshalIndent(res, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Available RAM: %d MB\n", ramMB)
			fmt.Fprintf(cmd.OutOrStdout(), "Max instances (startup only, 256MB/each): %d\n", maxStartup)
			fmt.Fprintf(cmd.OutOrStdout(), "Max efficient instances (vector ops, 512MB/each): %d\n", maxEfficient)
			return nil
		}

		n, err := strconv.Atoi(args[0])
		if err != nil || n <= 0 {
			lock.Remove()
			return fmt.Errorf("instance_count must be a positive integer")
		}
		count := uint64(n)

		if count > maxStartup {
			lock.Remove()
			return fmt.Errorf("insufficient RAM. Requested %d, but max possible for startup is %d (Available: %d MB)", count, maxStartup, ramMB)
		}

		if count > maxEfficient {
			logInfo(cmd, "WARNING: Requested %d instances, but max efficient count is %d. Performance may be degraded.", count, maxEfficient)
		}

		// Pre-flight check: Image
		logInfo(cmd, "Checking for qdrant/qdrant image...")
		if err := container.EnsureImage("qdrant/qdrant"); err != nil {
			lock.Remove()
			return fmt.Errorf("failed to ensure qdrant image: %w", err)
		}

		networkName := "qdrant_network"
		// Create network if not exists
		_ = container.CreateNetwork(networkName)

		currentUser, err := user.Current()
		if err != nil {
			lock.Remove()
			return err
		}
		homeDir := currentUser.HomeDir

		startRest := viper.GetInt("rest-port")
		startGrpc := viper.GetInt("grpc-port")

		for i := 0; i < n; i++ {
			select {
			case <-ctx.Done():
				logInfo(cmd, "\nInterrupted by user. Stopping further spawns...")
				// If we spawned at least one, we keep the lock because there are running instances
				if i == 0 {
					lock.Remove()
				}
				return ctx.Err()
			default:
				// Continue spawning
			}

			instanceNum := i + 1
			suffix := fmt.Sprintf("%02d", instanceNum)
			containerName := fmt.Sprintf("qdrant-%s", suffix)
			storageDir := filepath.Join(homeDir, fmt.Sprintf(".qdrant_storage%s", suffix))

			if err := os.MkdirAll(storageDir, 0755); err != nil {
				if i == 0 {
					lock.Remove()
				}
				return fmt.Errorf("failed to create storage dir %s: %w", storageDir, err)
			}

			restPort := startRest + (2 * i)
			grpcPort := startGrpc + (2 * i)

			logInfo(cmd, "Spawning %s on ports %d(REST), %d(GRPC)...", containerName, restPort, grpcPort)

			err = container.RunQdrant(container.QdrantConfig{
				Name:       containerName,
				Network:    networkName,
				RestPort:   restPort,
				GrpcPort:   grpcPort,
				StorageDir: storageDir,
			})
			if err != nil {
				logInfo(cmd, "Failed to spawn %s: %v", containerName, err)
				if i == 0 {
					lock.Remove()
				}
				return err
			}

			if i < n-1 {
				logInfo(cmd, "Waiting 30 seconds before spawning next instance...")
				// Wait but allow cancellation
				select {
				case <-time.After(30 * time.Second):
				case <-ctx.Done():
					logInfo(cmd, "\nInterrupted during wait. Stopping further spawns...")
					return ctx.Err()
				}
			}
		}

		if viper.GetString("output") == "json" {
			res := SpawnResult{
				AvailableRAMMB:   ramMB,
				MaxStartup:       maxStartup,
				MaxEfficient:     maxEfficient,
				SpawnedInstances: n,
				Status:           "success",
			}
			b, _ := json.MarshalIndent(res, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully spawned %d instances.\n", n)
		}
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(spawnCmd)
}
