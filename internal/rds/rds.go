package rds

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

// Cluster represents an RDS database cluster with its connection details
type Cluster struct {
	Identifier string // The unique identifier of the RDS cluster
	Endpoint   string // The endpoint URL to connect to the cluster
	Port       int32  // The port number the cluster is listening on
}

// GetFilteredRDSClusters retrieves RDS clusters filtered by tag name and value
func GetFilteredRDSClusters(ctx context.Context, cfg aws.Config, tagName, tagValue string) ([]Cluster, error) {
	if tagName == "" || tagValue == "" {
		return nil, fmt.Errorf("tagName and tagValue cannot be empty")
	}

	client := rds.NewFromConfig(cfg)
	input := &rds.DescribeDBClustersInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:" + tagName),
				Values: []string{tagValue},
			},
		},
	}

	clusters := make([]Cluster, 0)
	paginator := rds.NewDescribeDBClustersPaginator(client, input)

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

			clusters = append(clusters, Cluster{
				Identifier: *dbCluster.DBClusterIdentifier,
				Endpoint:   *dbCluster.Endpoint,
				Port:       *dbCluster.Port,
			})
		}
	}

	return clusters, nil
}

// GenerateAuthToken generates an authentication token for connecting to an RDS cluster
func GenerateAuthToken(cfg aws.Config, cluster Cluster, user string) (string, error) {
	if user == "" {
		return "", fmt.Errorf("user cannot be empty")
	}

	return auth.BuildAuthToken(
		context.Background(),
		cluster.Endpoint,
		cfg.Region,
		user,
		cfg.Credentials,
	)
}

type DatabaseService interface {
	GetClusters(ctx context.Context, tagName, tagValue string) ([]Cluster, error)
	GenerateToken(cluster Cluster, user string) (string, error)
}

type Service struct {
	cfg aws.Config
}

func NewService(cfg aws.Config) DatabaseService {
	return &Service{cfg: cfg}
}

func (s *Service) GetClusters(ctx context.Context, tagName, tagValue string) ([]Cluster, error) {
	return GetFilteredRDSClusters(ctx, s.cfg, tagName, tagValue)
}

func (s *Service) GenerateToken(cluster Cluster, user string) (string, error) {
	return GenerateAuthToken(s.cfg, cluster, user)
}
