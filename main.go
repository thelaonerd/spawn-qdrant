package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/thelaonerd/spawn-qdrant/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(getExitCode(err))
	}
}

func getExitCode(err error) int {
	if err == nil {
		return 0
	}
	
	// Check for cancellation/interrupts
	if errors.Is(err, os.ErrDeadlineExceeded) || strings.Contains(err.Error(), "context canceled") {
		return 130
	}

	errStr := err.Error()

	// Usage Error
	if strings.Contains(errStr, "invalid argument") || strings.Contains(errStr, "required flag") || strings.Contains(errStr, "unknown command") || strings.Contains(errStr, "positive integer") {
		return 64
	}

	// Data Error (RAM, ports)
	if strings.Contains(errStr, "insufficient RAM") || strings.Contains(errStr, "port") {
		return 65
	}

	// System Error (sudo, missing tools, locks)
	if strings.Contains(errStr, "sudo") || strings.Contains(errStr, "installed") || strings.Contains(errStr, "lock file") || strings.Contains(errStr, "system error") {
		return 71
	}

	// Default generic failure
	return 1
}
