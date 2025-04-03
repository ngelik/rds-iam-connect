// Package aws provides AWS-specific functionality for the RDS IAM Connect tool.
// It handles AWS credential management, IAM role verification, and RDS access checks.
package aws

import (
	"context"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// STSClient is an interface for AWS STS operations.
type STSClient interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

// IAMClient is an interface for AWS IAM operations.
type IAMClient interface {
	SimulatePrincipalPolicy(ctx context.Context, params *iam.SimulatePrincipalPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulatePrincipalPolicyOutput, error)
}

// Config wraps the AWS SDK config and provides additional functionality.
type Config struct {
	*aws.Config
	stsClient STSClient
	iamClient IAMClient
}

// CheckAWSCredentials validates and loads AWS credentials for the specified region.
// It returns a Config instance if successful, or an error if the credentials are invalid.
func CheckAWSCredentials(region string) (*Config, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return &Config{
		Config:    &cfg,
		stsClient: sts.NewFromConfig(cfg),
		iamClient: iam.NewFromConfig(cfg),
	}, nil
}

// GetCurrentIAMRole retrieves the IAM role ARN of the current AWS identity.
// It parses the STS caller identity to extract the IAM role information.
// Returns the IAM role ARN or an error if the operation fails.
func (c *Config) GetCurrentIAMRole(ctx context.Context) (string, error) {
	identity, err := c.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("failed to get caller identity: %w", err)
	}

	// Regular expression to match and extract components from STS ARN
	re := regexp.MustCompile(`arn:aws:sts::(\d+):assumed-role/([^/]+)/.*`)
	matches := re.FindStringSubmatch(*identity.Arn)

	if len(matches) == 3 {
		// matches[1] is the account ID
		// matches[2] is the role name
		return fmt.Sprintf("arn:aws:iam::%s:role/%s", matches[1], matches[2]), nil
	}

	return *identity.Arn, nil
}

// CheckIAMUserAccess verifies if the specified IAM role has permission to connect to the RDS cluster.
// It uses the IAM policy simulator to check the rds-db:connect permission.
// Returns an error if the access check fails or if the operation encounters an error.
func (c *Config) CheckIAMUserAccess(ctx context.Context, iamRole, resourceID, dbUserID string) error {
	resourceArn := fmt.Sprintf("arn:aws:rds-db:*:*:dbuser:%s/%s", resourceID, dbUserID)
	fmt.Printf("Checking IAM access for role %s to resource %s\n", iamRole, resourceArn)

	input := &iam.SimulatePrincipalPolicyInput{
		PolicySourceArn: aws.String(iamRole),
		ActionNames:     []string{"rds-db:connect"},
		ResourceArns:    []string{resourceArn},
	}

	output, err := c.iamClient.SimulatePrincipalPolicy(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to simulate IAM policy: %w", err)
	}

	if len(output.EvaluationResults) == 0 {
		return fmt.Errorf("no evaluation results found")
	}

	lastResult := output.EvaluationResults[len(output.EvaluationResults)-1]
	if lastResult.EvalDecision != "allowed" {
		return fmt.Errorf("IAM access denied: %s", lastResult.EvalDecision)
	}

	return nil
}

// WithSTSClient sets a custom STS client for testing.
func (c *Config) WithSTSClient(client STSClient) *Config {
	c.stsClient = client
	return c
}

// WithIAMClient sets a custom IAM client for testing.
func (c *Config) WithIAMClient(client IAMClient) *Config {
	c.iamClient = client
	return c
}
