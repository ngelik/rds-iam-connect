// Package utils provides utility functions for the RDS IAM Connect tool.
// It includes functions for directory and file management.
package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetCacheDir returns the path to the cache directory for the RDS IAM Connect tool.
// It creates the directory if it doesn't exist, with secure permissions (0700).
// Returns the absolute path to the cache directory or an error if the operation fails.
func GetCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".rds-iam-connect")
	// Use 0700 permissions to ensure only the owner has access
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	return cacheDir, nil
}
