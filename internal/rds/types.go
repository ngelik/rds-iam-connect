package rds

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"

	"rds-iam-connect/internal/logger"
)

// Client defines the interface for AWS RDS operations.
type Client interface {
	DescribeDBClusters(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error)
	ListTagsForResource(ctx context.Context, params *rds.ListTagsForResourceInput, optFns ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error)
}

// Cluster represents an RDS database cluster with its connection details.
type Cluster struct {
	Identifier string // The unique identifier of the RDS cluster.
	Endpoint   string // The endpoint URL to connect to the cluster.
	Port       int32  // The port number the cluster is listening on.
	Arn        string // The Amazon Resource Name of the cluster.
	Region     string // The AWS region where the cluster is located.
}

// DatabaseService provides functionality for interacting with AWS RDS clusters.
type DatabaseService struct {
	client      *rds.Client
	config      aws.Config
	cacheConfig struct {
		Enabled  bool
		Duration string
	}
	logger *logger.Logger
}

// CacheData represents the structure of cached RDS cluster data.
type CacheData struct {
	Timestamp time.Time `json:"timestamp"`
	Clusters  []Cluster `json:"clusters"`
}

// ErrClusterSkipped is returned when a cluster is skipped due to not meeting criteria.
var ErrClusterSkipped = errors.New("cluster skipped")
