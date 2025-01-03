package rds

import (
	"context"
	"fmt"

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
	client *rds.Client
}

// NewService creates a new instance of DatabaseService
func NewService(cfg aws.Config) *DatabaseService {
	return &DatabaseService{
		client: rds.NewFromConfig(cfg),
	}
}

// GetClusters retrieves RDS clusters filtered by tag name and value
func (svc *DatabaseService) GetClusters(ctx context.Context, tagName, tagValue string) ([]Cluster, error) {
	if tagName == "" || tagValue == "" {
		return nil, fmt.Errorf("tagName and tagValue cannot be empty")
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

			// Check if the cluster has the specified tag
			hasTag := false
			for _, tag := range tagsOutput.TagList {
				if *tag.Key == tagName && *tag.Value == tagValue {
					hasTag = true
					break
				}
			}

			if hasTag {
				clusters = append(clusters, Cluster{
					Identifier: *dbCluster.DBClusterIdentifier,
					Endpoint:   *dbCluster.Endpoint,
					Port:       *dbCluster.Port,
				})
			}
		}
	}

	return clusters, nil
}

// GenerateAuthToken generates an authentication token for connecting to an RDS cluster
func GenerateAuthToken(cfg aws.Config, cluster Cluster, user string) (string, error) {
	if user == "" {
		return "", fmt.Errorf("user cannot be empty")
	}

	fmt.Printf("Generating auth token with the following parameters:\n")
	fmt.Printf("Endpoint: %s:%d", cluster.Endpoint, cluster.Port)
	fmt.Printf("Port: %d\n", cluster.Port)
	fmt.Printf("User: %s\n", user)

	return auth.BuildAuthToken(
		context.Background(),
		fmt.Sprintf("%s:%d", cluster.Endpoint, cluster.Port),
		cfg.Region,
		user,
		cfg.Credentials,
	)
}
