package rds

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type CacheData struct {
	Timestamp time.Time `json:"timestamp"`
	Clusters  []Cluster `json:"clusters"`
}

func getCacheFilePath() string {
	// Create .tmp directory if it doesn't exist
	if err := os.MkdirAll(".tmp", 0755); err != nil {
		// If we can't create the directory, fall back to current directory
		return "rds-clusters-cache.json"
	}
	return filepath.Join(".tmp", "rds-clusters-cache.json")
}

func (svc *DatabaseService) loadFromCache() ([]Cluster, bool) {
	if !svc.cacheConfig.Enabled {
		return nil, false
	}

	cacheFile := getCacheFilePath()
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, false
	}

	var cache CacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, false
	}

	// Parse duration (e.g., "1d" = 1 day)
	duration, err := parseDuration(svc.cacheConfig.Duration)
	if err != nil {
		return nil, false
	}

	// Check if cache is expired
	if time.Since(cache.Timestamp) > duration {
		return nil, false
	}

	fmt.Println("Loaded list of RDS clusters from cache")
	return cache.Clusters, true
}

func (svc *DatabaseService) saveToCache(clusters []Cluster) error {
	if !svc.cacheConfig.Enabled {
		return nil
	}

	cache := CacheData{
		Timestamp: time.Now(),
		Clusters:  clusters,
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	cacheFile := getCacheFilePath()
	return os.WriteFile(cacheFile, data, 0644)
}

func parseDuration(s string) (time.Duration, error) {
	// Handle day format (e.g., "1d")
	if len(s) > 0 && s[len(s)-1] == 'd' {
		days, err := time.ParseDuration(s[:len(s)-1] + "h")
		if err != nil {
			return 0, err
		}
		return days * 24, nil
	}
	return time.ParseDuration(s)
}
