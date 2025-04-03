// Package rds provides functionality for interacting with AWS RDS clusters.
// It includes cluster discovery, authentication, and caching capabilities.
package rds

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rds-iam-connect/internal/utils"
)

// Constants for cache operations.
const (
	// 0600 is more secure as it only allows the owner to read/write.
	cacheFileMode = 0600
)

// GetCacheFileName returns the name of the cache file for a specific environment.
func GetCacheFileName(env string) string {
	return fmt.Sprintf("rds-clusters-cache-%s.json", env)
}

// validateCacheFile checks if the cache file exists and is valid.
func (svc *DatabaseService) validateCacheFile(cacheFile string) (os.FileInfo, error) {
	info, err := os.Stat(cacheFile)
	if err != nil {
		svc.logger.Debugf("Cache file not found or inaccessible: %v", err)
		return nil, err
	}
	if !info.Mode().IsRegular() {
		svc.logger.Debugf("Cache file is not a regular file: %s", cacheFile)
		return nil, fmt.Errorf("cache file is not a regular file")
	}
	svc.logger.Debugf("Cache file validated: %s", cacheFile)
	return info, nil
}

// parseCacheData reads and parses the cache file.
func (svc *DatabaseService) parseCacheData(cacheFile string, cacheDir string) (*CacheData, error) {
	// Validate the cache file path
	if !strings.HasPrefix(cacheFile, cacheDir) {
		svc.logger.Debugf("Invalid cache file path: %s", cacheFile)
		return nil, fmt.Errorf("invalid cache file path")
	}

	//nolint:gosec // False positive: path is validated above
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		svc.logger.Debugf("Failed to read cache file: %v", err)
		return nil, err
	}

	var cache CacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		svc.logger.Debugf("Failed to parse cache data: %v", err)
		return nil, err
	}
	svc.logger.Debugf("Successfully parsed cache data from: %s", cacheFile)
	return &cache, nil
}

// isCacheExpired checks if the cache is expired based on duration and current time.
// Duration should be a valid Go duration string (e.g., "24h", "30m", "1h30m").
// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
func (svc *DatabaseService) isCacheExpired(cache *CacheData, duration time.Duration) bool {
	now := time.Now()
	expired := now.Sub(cache.Timestamp) > duration || cache.Timestamp.After(now)
	if expired {
		svc.logger.Debugf("Cache is expired. Cache timestamp: %v, Current time: %v, Duration: %v",
			cache.Timestamp, now, duration)
	} else {
		svc.logger.Debugf("Cache is valid. Cache timestamp: %v, Current time: %v, Duration: %v",
			cache.Timestamp, now, duration)
	}
	return expired
}

// loadFromCache attempts to load RDS clusters from the cache file.
// Returns the clusters and a boolean indicating if the cache was valid and loaded successfully.
// The cache duration should be a valid Go duration string (e.g., "24h", "30m", "1h30m").
func (svc *DatabaseService) loadFromCache(env string) ([]Cluster, bool) {
	if !svc.cacheConfig.Enabled {
		svc.logger.Debugln("Cache is disabled")
		return nil, false
	}

	cacheDir, err := utils.GetCacheDir()
	if err != nil {
		svc.logger.Debugf("Failed to get cache directory: %v", err)
		return nil, false
	}

	cacheFile := filepath.Join(cacheDir, GetCacheFileName(env))
	if _, err := svc.validateCacheFile(cacheFile); err != nil {
		return nil, false
	}

	cache, err := svc.parseCacheData(cacheFile, cacheDir)
	if err != nil {
		return nil, false
	}

	duration, err := time.ParseDuration(svc.cacheConfig.Duration)
	if err != nil {
		svc.logger.Debugf("Invalid cache duration format '%s'. Use a valid Go duration (e.g., '24h', '30m'): %v",
			svc.cacheConfig.Duration, err)
		return nil, false
	}

	if svc.isCacheExpired(cache, duration) {
		return nil, false
	}

	svc.logger.Debugf("Successfully loaded %d clusters from cache for environment %s", len(cache.Clusters), env)
	return cache.Clusters, true
}

// saveToCache saves the RDS clusters to the cache file.
// Returns an error if the operation fails.
func (svc *DatabaseService) saveToCache(clusters []Cluster, env string) error {
	if !svc.cacheConfig.Enabled {
		svc.logger.Debugln("Cache is disabled, skipping save")
		return nil
	}

	cacheDir, err := utils.GetCacheDir()
	if err != nil {
		svc.logger.Debugf("Failed to get cache directory: %v", err)
		return fmt.Errorf("failed to get cache directory: %w", err)
	}

	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		svc.logger.Debugf("Failed to create cache directory: %v", err)
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := CacheData{
		Clusters:  clusters,
		Timestamp: time.Now().UTC(),
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		svc.logger.Debugf("Failed to marshal cache data: %v", err)
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	cacheFile := filepath.Join(cacheDir, GetCacheFileName(env))
	if err := os.WriteFile(cacheFile, data, cacheFileMode); err != nil {
		svc.logger.Debugf("Failed to write cache file: %v", err)
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	svc.logger.Debugf("Successfully saved %d clusters to cache for environment %s: %s", len(clusters), env, cacheFile)
	return nil
}
