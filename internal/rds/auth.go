package rds

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
)

// GenerateAuthToken generates an authentication token for connecting to an RDS cluster.
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
