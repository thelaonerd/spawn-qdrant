package container

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Runtime string

const (
	Docker Runtime = "docker"
	Podman Runtime = "podman"
)

var containerRuntime Runtime

func InitRuntime() error {
	if isCommandAvailable("docker") {
		containerRuntime = Docker
		return nil
	}
	if isCommandAvailable("podman") {
		containerRuntime = Podman
		return nil
	}
	return fmt.Errorf("neither docker nor podman is installed")
}

func GetRuntime() Runtime {
	return containerRuntime
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

type QdrantConfig struct {
	Name       string
	Network    string
	RestPort   int
	GrpcPort   int
	StorageDir string
}

func runCommand(args ...string) error {
	cmd := exec.Command(string(containerRuntime), args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func runCommandOutput(args ...string) (string, error) {
	cmd := exec.Command(string(containerRuntime), args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func EnsureImage(imageName string) error {
	_, err := runCommandOutput("inspect", "--type=image", imageName)
	if err == nil {
		return nil
	}

	fmt.Printf("Image %s not found locally. Pulling...\n", imageName)
	return runCommand("pull", imageName)
}

func CreateNetwork(name string) error {
	_, err := runCommandOutput("network", "create", name)
	return err
}

func RemoveNetwork(name string) error {
	return runCommand("network", "rm", name)
}

func ListContainerNames(prefix string) ([]string, error) {
	output, err := runCommandOutput("ps", "-a", "--format", "{{.Names}}")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(output, "\n") {
		name := strings.TrimSpace(line)
		if strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}
	return names, nil
}

func HasRunningContainers(filter string) (bool, error) {
	output, err := runCommandOutput("ps", "-q", "-f", filter)
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(output)) > 0, nil
}

func StopAndRemoveContainer(name string) error {
	_ = runCommand("stop", name)
	return runCommand("rm", name)
}

func RunQdrant(cfg QdrantConfig) error {
	return runCommand("run", "-d",
		"--name", cfg.Name,
		"--net", cfg.Network,
		"--restart", "unless-stopped",
		"-p", fmt.Sprintf("%d:6333", cfg.RestPort),
		"-p", fmt.Sprintf("%d:6334", cfg.GrpcPort),
		"-v", fmt.Sprintf("%s:/qdrant/storage", cfg.StorageDir),
		"qdrant/qdrant",
	)
}
