package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/thelaonerd/spawn-qdrant/internal/config"
	"github.com/thelaonerd/spawn-qdrant/internal/container"
	"github.com/thelaonerd/spawn-qdrant/internal/lock"
	"github.com/thelaonerd/spawn-qdrant/internal/system"
)

var spawnCmd = &cobra.Command{
	Use:   "spawn [instance_count]",
	Short: "Spawn qdrant instances",
	Long: `Spawn n instances of Qdrant.
If instance_count is not provided, it estimates the maximum instances based on available RAM.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Acquire lock
		if err := lock.Create(); err != nil {
			return err
		}

		// Pre-flight check: Image
		fmt.Println("Checking for qdrant/qdrant image...")
		if err := container.EnsureImage("qdrant/qdrant"); err != nil {
			// Clean up lock if we fail here?
			// Ideally yes, but maybe simple error return is fine for now,
			// user can run 'stop all' or manually remove.
			// Better: defer lock removal if error, but wait, we want to KEEP lock if success.
			// So only remove on error.
			lock.Remove()
			return fmt.Errorf("failed to ensure qdrant image: %w", err)
		}

		// Pre-flight check: RAM
		ramMB, err := system.GetAvailableRAM()
		if err != nil {
			lock.Remove()
			return fmt.Errorf("failed to get available RAM: %w", err)
		}
		maxStartup, maxEfficient := system.EstimateInstances(ramMB)

		if len(args) == 0 {
			lock.Remove() // Estimation only, no spawn
			fmt.Printf("Available RAM: %d MB\n", ramMB)
			fmt.Printf("Max instances (startup only, 256MB/each): %d\n", maxStartup)
			fmt.Printf("Max efficient instances (vector ops, 512MB/each): %d\n", maxEfficient)
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
			fmt.Printf("WARNING: Requested %d instances, but max efficient count is %d. Performance may be degraded.\n", count, maxEfficient)
		}

		cfg := config.LoadConfig()
		networkName := "qdrant_network"

		// Create network if not exists
		_ = container.CreateNetwork(networkName)

		currentUser, err := user.Current()
		if err != nil {
			lock.Remove()
			return err
		}
		homeDir := currentUser.HomeDir

		startRest := cfg.RestPort
		startGrpc := cfg.GrpcPort

		for i := 0; i < n; i++ {
			instanceNum := i + 1
			suffix := fmt.Sprintf("%02d", instanceNum)
			containerName := fmt.Sprintf("qdrant-%s", suffix)
			storageDir := filepath.Join(homeDir, fmt.Sprintf(".qdrant_storage%s", suffix))

			if err := os.MkdirAll(storageDir, 0755); err != nil {
				lock.Remove()
				return fmt.Errorf("failed to create storage dir %s: %w", storageDir, err)
			}

			restPort := startRest + (2 * i)
			grpcPort := startGrpc + (2 * i)

			fmt.Printf("Spawning %s on ports %d(REST), %d(GRPC)...\n", containerName, restPort, grpcPort)

			err := container.RunQdrant(container.QdrantConfig{
				Name:       containerName,
				Network:    networkName,
				RestPort:   restPort,
				GrpcPort:   grpcPort,
				StorageDir: storageDir,
			})
			if err != nil {
				fmt.Printf("Failed to spawn %s: %v\n", containerName, err)
				lock.Remove()
				return err
			}

			if i < n-1 {
				fmt.Println("Waiting 30 seconds before spawning next instance...")
				time.Sleep(30 * time.Second)
			}
		}

		fmt.Printf("Successfully spawned %d instances.\n", n)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(spawnCmd)
}
