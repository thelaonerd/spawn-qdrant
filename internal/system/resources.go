package system

import (
	"bufio"
	"fmt"
	"os"
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
				kb, err := strconvToUint64(parts[1])
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

func strconvToUint64(s string) (uint64, error) {
	var res uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid digit")
		}
		res = res*10 + uint64(c-'0')
	}
	return res, nil
}
