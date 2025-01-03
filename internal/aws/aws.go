package aws

import (
	"context"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// Config wraps the AWS SDK config
type Config struct {
	*awssdk.Config
}

func CheckAWSCredentials() (*Config, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	return &Config{&cfg}, nil
}
