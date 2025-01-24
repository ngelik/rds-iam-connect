package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	RdsTags struct {
		TagName  string
		TagValue string
	}
	AllowedIAMUsers []string
	EnvTag          map[string]struct {
		ReleaseState string
	}
	Caching struct {
		Enabled  bool
		Duration string
	}
	CheckIAMPermissions bool
}

func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	return &config, nil
}
