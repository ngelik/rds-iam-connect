// Package config provides configuration management for the RDS IAM Connect tool.
// It handles loading and parsing of configuration files, with support for YAML format.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rds-iam-connect/internal/utils"

	"github.com/spf13/viper"
)

// Config represents the application configuration structure.
// It contains settings for RDS tags, IAM users, environment tags, caching, and IAM permission checks.
type Config struct {
	// RdsTags contains the tag name and value used to identify RDS clusters.
	RdsTags struct {
		TagName  string // The name of the tag used to identify RDS clusters.
		TagValue string // The value of the tag used to identify RDS clusters.
	}
	// AllowedIAMUsers lists the IAM users permitted to connect to RDS clusters.
	AllowedIAMUsers []string
	// EnvTag maps environment names to their release state and region.
	EnvTag map[string]struct {
		ReleaseState string // The release state of the environment (e.g., "prod", "staging").
		Region       string // The AWS region where the environment is located.
	}
	// Caching controls the caching behavior for RDS cluster data.
	Caching struct {
		Enabled  bool   // Whether caching is enabled.
		Duration string // The duration for which cached data is valid.
	}
	// CheckIAMPermissions determines whether to verify IAM permissions before connecting.
	CheckIAMPermissions bool
	// Debug enables detailed logging when set to true.
	Debug bool
}

// LoadConfig loads the application configuration from a YAML file.
// If configPath is not provided, it uses the default path in the user's home directory.
// If the config file doesn't exist, it copies the example config.
// Returns a Config instance or an error if the operation fails.
func LoadConfig(configPath string) (*Config, error) {
	if configPath != "config.yaml" {
		return loadConfigFromPath(configPath)
	}

	return loadDefaultConfig()
}

// loadConfigFromPath loads configuration from the specified path.
func loadConfigFromPath(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config into struct: %w", err)
	}

	return &config, nil
}

// loadDefaultConfig loads the default configuration from the user's home directory.
func loadDefaultConfig() (*Config, error) {
	cacheDir, err := utils.GetCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	configPath := filepath.Join(cacheDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultConfig(configPath); err != nil {
			return nil, err
		}
	}

	return loadConfigFromPath(configPath)
}

// createDefaultConfig creates a new default configuration file.
func createDefaultConfig(configPath string) error {
	exampleConfig := filepath.Clean(filepath.Join(".", "config.yaml"))
	if !strings.HasPrefix(exampleConfig, ".") {
		return fmt.Errorf("invalid example config path")
	}

	data, err := os.ReadFile(exampleConfig)
	if err != nil {
		return fmt.Errorf("failed to read example config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to create default config: %w", err)
	}

	fmt.Printf("Created default config at %s\n", configPath)
	return nil
}
