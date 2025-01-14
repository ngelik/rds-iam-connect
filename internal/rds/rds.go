package rds

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

// Cluster represents an RDS database cluster with its connection details
type Cluster struct {
	Identifier string // The unique identifier of the RDS cluster
	Endpoint   string // The endpoint URL to connect to the cluster
	Port       int32  // The port number the cluster is listening on
}

// DatabaseService represents the RDS service
type DatabaseService struct {
	client      *rds.Client
	cacheConfig struct {
		Enabled  bool
		Duration string
	}
}

// NewService creates a new instance of DatabaseService
func NewService(cfg aws.Config, cacheEnabled bool, cacheDuration string) *DatabaseService {
	return &DatabaseService{
		client: rds.NewFromConfig(cfg),
		cacheConfig: struct {
			Enabled  bool
			Duration string
		}{
			Enabled:  cacheEnabled,
			Duration: cacheDuration,
		},
	}
}

// GetClusters retrieves RDS clusters filtered by tags
func (svc *DatabaseService) GetClusters(ctx context.Context, tagName, tagValue, envTagName, envTagValue string) ([]Cluster, error) {
	// Try to load from cache first
	if clusters, ok := svc.loadFromCache(); ok {
		return clusters, nil
	}

	// Original cluster fetching logic
	if tagName == "" || tagValue == "" || envTagName == "" || envTagValue == "" {
		return nil, fmt.Errorf("tag parameters cannot be empty")
	}

	input := &rds.DescribeDBClustersInput{}
	clusters := make([]Cluster, 0)
	paginator := rds.NewDescribeDBClustersPaginator(svc.client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing RDS clusters: %w", err)
		}

		for _, dbCluster := range page.DBClusters {
			if dbCluster.IAMDatabaseAuthenticationEnabled == nil || !*dbCluster.IAMDatabaseAuthenticationEnabled {
				continue
			}

			if dbCluster.DBClusterIdentifier == nil || dbCluster.Endpoint == nil || dbCluster.Port == nil {
				continue // Skip clusters with missing required fields
			}

			// Retrieve tags for the cluster
			tagsInput := &rds.ListTagsForResourceInput{
				ResourceName: dbCluster.DBClusterArn,
			}
			tagsOutput, err := svc.client.ListTagsForResource(ctx, tagsInput)
			if err != nil {
				return nil, fmt.Errorf("listing tags for resource: %w", err)
			}

			// Check if the cluster has both specified tags
			hasTagName := false
			hasEnvTag := false
			for _, tag := range tagsOutput.TagList {
				if *tag.Key == tagName && *tag.Value == tagValue {
					hasTagName = true
				}
				if *tag.Key == envTagName && *tag.Value == envTagValue {
					hasEnvTag = true
				}
			}

			if hasTagName && hasEnvTag {
				clusters = append(clusters, Cluster{
					Identifier: *dbCluster.DBClusterIdentifier,
					Endpoint:   *dbCluster.Endpoint,
					Port:       *dbCluster.Port,
				})
			}
		}
	}

	// Save to cache before returning
	if err := svc.saveToCache(clusters); err != nil {
		// Log the error but don't fail the operation
		fmt.Printf("Failed to save clusters to cache: %v\n", err)
	}

	return clusters, nil
}

// GenerateAuthToken generates an authentication token for connecting to an RDS cluster
func GenerateAuthToken(cfg aws.Config, cluster Cluster, user string, logger *log.Logger) (string, error) {
	if user == "" {
		return "", fmt.Errorf("user cannot be empty")
	}

	logger.Printf("generating auth token for endpoint: %s:%d, user: %s",
		cluster.Endpoint, cluster.Port, user)

	return auth.BuildAuthToken(
		context.Background(),
		fmt.Sprintf("%s:%d", cluster.Endpoint, cluster.Port),
		cfg.Region,
		user,
		cfg.Credentials,
	)
}
