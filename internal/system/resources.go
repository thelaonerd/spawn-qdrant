package system

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// GetAvailableRAM returns the available RAM in MB
func GetAvailableRAM() (uint64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemAvailable:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				kb, err := strconv.ParseUint(parts[1], 10, 64)
				if err != nil {
					return 0, err
				}
				return kb / 1024, nil // Convert KB to MB
			}
		}
	}
	return 0, fmt.Errorf("MemAvailable not found in /proc/meminfo")
}

// EstimateInstances returns max instances based on startup (256MB) and efficiency (512MB)
func EstimateInstances(availableRAMMB uint64) (maxStartup uint64, maxEfficient uint64) {
	maxStartup = availableRAMMB / 256
	maxEfficient = availableRAMMB / 512
	return
}
