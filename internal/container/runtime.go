package container

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
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
	output, err := executeCommand(args...)
	if err == nil || !shouldUseSudo(output, err) {
		return err
	}

	_, sudoErr := executeCommandWithSudo(args...)
	return sudoErr
}

func runCommandOutput(args ...string) (string, error) {
	output, err := executeCommand(args...)
	if err == nil || !shouldUseSudo(output, err) {
		return strings.TrimSpace(output), err
	}

	output, err = executeCommandWithSudo(args...)
	return strings.TrimSpace(output), err
}

// executeCommand runs the container runtime as the current user. The output
// is retained so a permission failure can trigger the sudo fallback without
// treating unrelated runtime errors as a request for privilege escalation.
func executeCommand(args ...string) (string, error) {
	return execute(exec.Command(string(containerRuntime), args...))
}

func executeCommandWithSudo(args ...string) (string, error) {
	sudoArgs := append([]string{string(containerRuntime)}, args...)
	return execute(exec.Command("sudo", sudoArgs...))
}

func execute(cmd *exec.Cmd) (string, error) {
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if isInteractiveTerminal(os.Stdin) {
		cmd.Stdin = os.Stdin
	}
	err := cmd.Run()
	return output.String(), err
}

func shouldUseSudo(output string, err error) bool {
	if err == nil || !isInteractiveTerminal(os.Stdin) {
		return false
	}
	text := strings.ToLower(output + " " + err.Error())
	return strings.Contains(text, "permission denied") ||
		strings.Contains(text, "access denied") ||
		strings.Contains(text, "must be root")
}

// isInteractiveTerminal prevents a non-interactive invocation from hanging
// while sudo waits for a password that cannot be entered.
func isInteractiveTerminal(file *os.File) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, file.Fd(), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}

func EnsureImage(imageName string) error {
	_, err := runCommandOutput("inspect", "--type=image", "--", imageName)
	if err == nil {
		return nil
	}

	fmt.Printf("Image %s not found locally. Pulling...\n", imageName)
	return runCommand("pull", "--", imageName)
}

func CreateNetwork(name string) error {
	_, err := runCommandOutput("network", "create", "--", name)
	return err
}

func RemoveNetwork(name string) error {
	return runCommand("network", "rm", "--", name)
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
	_ = runCommand("stop", "--", name)
	return runCommand("rm", "--", name)
}

func RunQdrant(cfg QdrantConfig) error {
	// Security check for storage directory to prevent arbitrary volume mounting
	if strings.Contains(cfg.StorageDir, ":") {
		return fmt.Errorf("invalid storage directory: path cannot contain ':'")
	}

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
