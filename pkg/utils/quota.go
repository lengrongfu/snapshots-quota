package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
)

// GetMountOptions returns the mount options for the filesystem containing the given path
func GetMountOptions(path string) (string, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return "", fmt.Errorf("failed to get filesystem stats: %v", err)
	}

	// Open /proc/mounts
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return "", fmt.Errorf("failed to open /proc/mounts: %v", err)
	}
	defer file.Close()

	// Use bufio.Scanner to read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		// Check if this is the mount point we're looking for
		mountPoint := fields[1]
		filesystemType := fields[2]
		if strings.EqualFold(path, mountPoint) && filesystemType == "xfs" {
			// Return the mount options
			return fields[3], nil
		}

	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading mount file: %v", err)
	}

	return "", fmt.Errorf("mount point not found for path: %s", path)
}

// IsPrjQuotaEnabled checks if the filesystem has prjquota mount option enabled
func IsPrjQuotaEnabled(path string) (bool, error) {
	options, err := GetMountOptions(path)
	if err != nil {
		return false, err
	}
	// Check if prjquota is in the mount options
	return strings.Contains(options, "prjquota"), nil
}
