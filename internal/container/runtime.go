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

func RunCommand(args ...string) error {
	cmd := exec.Command(string(containerRuntime), args...)
	cmd.Stdout = nil // We might want to stream stdout/stderr later
	cmd.Stderr = nil
	return cmd.Run()
}

func RunCommandOutput(args ...string) (string, error) {
	cmd := exec.Command(string(containerRuntime), args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out // Capture stderr too for debugging
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func EnsureImage(imageName string) error {
	// Check if image exists locally
	_, err := RunCommandOutput("inspect", "--type=image", imageName)
	if err == nil {
		return nil // Image exists
	}

	fmt.Printf("Image %s not found locally. Pulling...\n", imageName)
	return RunCommand("pull", imageName)
}
