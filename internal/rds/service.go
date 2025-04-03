package rds

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"

	"rds-iam-connect/internal/logger"
)

// NewService creates a new instance of DatabaseService.
func NewService(cfg aws.Config, cacheEnabled bool, cacheDuration string, debug bool) *DatabaseService {
	return &DatabaseService{
		client: rds.NewFromConfig(cfg),
		config: cfg,
		cacheConfig: struct {
			Enabled  bool
			Duration string
		}{
			Enabled:  cacheEnabled,
			Duration: cacheDuration,
		},
		logger: logger.New(debug),
	}
}

// validateTags checks if the required tags are provided.
func validateTags(tagName, tagValue, envTagName, envTagValue string) error {
	if tagName == "" || tagValue == "" || envTagName == "" || envTagValue == "" {
		return fmt.Errorf("tag parameters cannot be empty")
	}
	return nil
}

// hasRequiredTags checks if a cluster has both specified tags.
func hasRequiredTags(tags []types.Tag, tagName, tagValue, envTagName, envTagValue string) bool {
	hasTagName := false
	hasEnvTag := false

	for _, tag := range tags {
		if *tag.Key == tagName && *tag.Value == tagValue {
			hasTagName = true
		}
		if *tag.Key == envTagName && *tag.Value == envTagValue {
			hasEnvTag = true
		}
	}

	return hasTagName && hasEnvTag
}

// extractRegionFromARN extracts the region from an ARN.
func extractRegionFromARN(arn string) string {
	if arnParts := strings.Split(arn, ":"); len(arnParts) >= 4 {
		return arnParts[3]
	}
	return ""
}

// processDBCluster processes a single DB cluster and returns a Cluster if it matches the criteria.
// Returns ErrClusterSkipped if the cluster doesn't meet the criteria.
func (svc *DatabaseService) processDBCluster(ctx context.Context, dbCluster types.DBCluster, tagName, tagValue, envTagName, envTagValue string) (*Cluster, error) {
	if dbCluster.IAMDatabaseAuthenticationEnabled == nil || !*dbCluster.IAMDatabaseAuthenticationEnabled {
		return nil, ErrClusterSkipped
	}

	if dbCluster.DBClusterIdentifier == nil || dbCluster.Endpoint == nil || dbCluster.Port == nil {
		return nil, ErrClusterSkipped
	}

	tagsInput := &rds.ListTagsForResourceInput{
		ResourceName: dbCluster.DBClusterArn,
	}
	tagsOutput, err := svc.client.ListTagsForResource(ctx, tagsInput)
	if err != nil {
		return nil, fmt.Errorf("listing tags for resource: %w", err)
	}

	if !hasRequiredTags(tagsOutput.TagList, tagName, tagValue, envTagName, envTagValue) {
		return nil, ErrClusterSkipped
	}

	region := extractRegionFromARN(*dbCluster.DBClusterArn)
	if region != svc.config.Region {
		return nil, ErrClusterSkipped
	}

	return &Cluster{
		Identifier: *dbCluster.DBClusterIdentifier,
		Endpoint:   *dbCluster.Endpoint,
		Port:       *dbCluster.Port,
		Arn:        *dbCluster.DBClusterArn,
		Region:     region,
	}, nil
}

// fetchClustersFromAWS retrieves clusters from AWS RDS and processes them.
func (svc *DatabaseService) fetchClustersFromAWS(ctx context.Context, tagName, tagValue, envTagName, envTagValue string) ([]Cluster, error) {
	svc.logger.Debugf("Fetching RDS clusters from AWS (region: %s)", svc.config.Region)
	clusters := make([]Cluster, 0)
	input := &rds.DescribeDBClustersInput{}
	paginator := rds.NewDescribeDBClustersPaginator(svc.client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			svc.logger.Debugf("Error describing RDS clusters: %v", err)
			return nil, fmt.Errorf("describing RDS clusters: %w", err)
		}

		svc.logger.Debugf("Processing %d clusters from AWS", len(page.DBClusters))
		for _, dbCluster := range page.DBClusters {
			cluster, err := svc.processDBCluster(ctx, dbCluster, tagName, tagValue, envTagName, envTagValue)
			if err != nil {
				if errors.Is(err, ErrClusterSkipped) {
					svc.logger.Debugf("Skipping cluster %s: %v", *dbCluster.DBClusterIdentifier, err)
					continue
				}
				svc.logger.Debugf("Error processing cluster %s: %v", *dbCluster.DBClusterIdentifier, err)
				return nil, err
			}
			if cluster != nil {
				svc.logger.Debugf("Found matching cluster: %s", cluster.Identifier)
				clusters = append(clusters, *cluster)
			}
		}
	}
	svc.logger.Debugf("Found %d matching RDS clusters in AWS", len(clusters))
	return clusters, nil
}

// GetClusters retrieves RDS clusters based on the provided tags and environment.
func (svc *DatabaseService) GetClusters(ctx context.Context, tagName, tagValue, envTagName, envTagValue, env string) ([]Cluster, error) {
	if err := validateTags(tagName, tagValue, envTagName, envTagValue); err != nil {
		svc.logger.Debugf("Invalid tags provided: %v", err)
		return nil, err
	}

	// Try to load from cache first
	svc.logger.Debugln("Attempting to load clusters from cache")
	if clusters, ok := svc.loadFromCache(env); ok {
		svc.logger.Debugf("Successfully loaded %d clusters from cache", len(clusters))
		return clusters, nil
	}
	svc.logger.Debugln("Cache miss or invalid, fetching from AWS")

	// Fetch clusters from AWS
	clusters, err := svc.fetchClustersFromAWS(ctx, tagName, tagValue, envTagName, envTagValue)
	if err != nil {
		return nil, err
	}

	// Save to cache before returning
	if err := svc.saveToCache(clusters, env); err != nil {
		svc.logger.Debugf("Warning: Failed to save clusters to cache: %v", err)
	}

	return clusters, nil
}

// GetRDSInstanceIdentifier gets the RDS instance identifier.
func (svc *DatabaseService) GetRDSInstanceIdentifier(cluster Cluster) string {
	input := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(cluster.Identifier),
	}

	output, err := svc.client.DescribeDBClusters(context.Background(), input)
	if err != nil {
		return ""
	}

	return *output.DBClusters[0].DbClusterResourceId
}
