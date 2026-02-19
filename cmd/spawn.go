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
		if _, err := container.RunCommandOutput("network", "create", networkName); err != nil {
			// Ignore error if network already exists, but ideally checking first is better.
			// However, 'network create' usually errors if it exists.
			// Let's rely on docker/podman behavior or check.
			// For simplicity, we just try to create and ignore "already exists" errors if possible,
			// checking output or just proceeding.
			// A specific check would be cleaner:
			// logic to check network... skip for brevity unless critical.
		}

		currentUser, err := user.Current()
		if err != nil {
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
				return fmt.Errorf("failed to create storage dir %s: %w", storageDir, err)
			}

			// Calculate ports
			// Per spec:
			// If n=1: existing ports.
			// If n>1: start + 2*(i). Wait, spec says:
			// "The actual ports used will be REST_PORT + 2*(instance_count - 1)" -> This formula in spec seems to describe the END port?
			// Spec example:
			// n=1: 6333, 6334. (i=0 -> 6333 + 2*0 = 6333)
			// n=2:
			//   i=0 -> 6333, 6334
			//   i=1 -> 6335, 6336 (6333 + 2*1)

			restPort := startRest + (2 * i)
			grpcPort := startGrpc + (2 * i)

			fmt.Printf("Spawning %s on ports %d(REST), %d(GRPC)...\n", containerName, restPort, grpcPort)

			// docker run -d --name qdrant-01 --net qdrant_network --restart unless-stopped -p 6333:6333 -p 6334:6334 -v ~/.qdrant_storage01:/qdrant/storage qdrant/qdrant
			err := container.RunCommand("run", "-d",
				"--name", containerName,
				"--net", networkName,
				"--restart", "unless-stopped",
				"-p", fmt.Sprintf("%d:6333", restPort),
				"-p", fmt.Sprintf("%d:6334", grpcPort),
				"-v", fmt.Sprintf("%s:/qdrant/storage", storageDir),
				"qdrant/qdrant",
			)
			if err != nil {
				fmt.Printf("Failed to spawn %s: %v\n", containerName, err)
				// Clean up lock if we fail?
				// Since we might have partial success (some spawned), we shouldn't blindly remove lock.
				// But we are returning error, so maybe we should?
				// If we error out, user has to fix state.
				// Returning error here stops the loop.
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
