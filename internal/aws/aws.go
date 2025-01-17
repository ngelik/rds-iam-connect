package aws

import (
	"context"
	"fmt"
	"regexp"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
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

func (c *Config) GetCurrentIAMRole(ctx context.Context) (string, error) {
	stsClient := sts.NewFromConfig(*c.Config)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("getting caller identity: %w", err)
	}

	// Construct IAM Role ARN from STS ARN
	// Regular expression to match and extract components from STS ARN
	re := regexp.MustCompile(`arn:aws:sts::(\d+):assumed-role/([^/]+)/.*`)
	matches := re.FindStringSubmatch(*identity.Arn)

	if len(matches) == 3 {
		// matches[0] is the full match
		// matches[1] is the account ID
		// matches[2] is the role name
		return fmt.Sprintf("arn:aws:iam::%s:role/%s", matches[1], matches[2]), nil
	}

	return *identity.Arn, nil
}
