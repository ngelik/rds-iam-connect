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

// Config wraps the AWS SDK config
type Config struct {
	*aws.Config
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

// CheckIAMUserAccess checks if the current IAM role has access to the RDS cluster
func (c *Config) CheckIAMUserAccess(ctx context.Context, iamRole, resourceId, dbUserId string) error {
	resourceArn := fmt.Sprintf("arn:aws:rds-db:*:*:dbuser:%s/%s", resourceId, dbUserId)
	fmt.Printf("Checking IAM access for role %s to resource %s\n", iamRole, resourceArn)

	input := &iam.SimulatePrincipalPolicyInput{
		PolicySourceArn: aws.String(iamRole),
		ActionNames:     []string{"rds-db:connect"},
		ResourceArns:    []string{fmt.Sprintf("arn:aws:rds-db:*:*:dbuser:%s/%s", resourceId, dbUserId)},
	}

	iamClient := iam.NewFromConfig(*c.Config)
	output, err := iamClient.SimulatePrincipalPolicy(ctx, input)
	if err != nil {
		fmt.Printf("Error generating IAM auth token: %v\n", err)
		return err
	}

	if len(output.EvaluationResults) > 0 {
		lastResult := output.EvaluationResults[len(output.EvaluationResults)-1]
		fmt.Printf("Action: %s, Decision: %s\n", *lastResult.EvalActionName, lastResult.EvalDecision)
		return fmt.Errorf("IAM access check: %s", lastResult.EvalDecision)
	}
	return fmt.Errorf("no evaluation results")
}
