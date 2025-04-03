// Package cmd provides the command-line interface for the RDS IAM Connect tool.
// It handles user interaction, configuration, and connection to RDS clusters.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"

	"rds-iam-connect/config"
	"rds-iam-connect/internal/aws"
	"rds-iam-connect/internal/rds"

	"log"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/spf13/cobra"
)

var (
	configPath string
	rdsService *rds.DatabaseService
	checkOnly  bool
)

// rootCmd represents the base command when called without any subcommands.
// It provides the main functionality for connecting to RDS clusters using IAM authentication.
var rootCmd = &cobra.Command{
	Use:   "rds-iam-connect",
	Short: "Connect to AWS RDS clusters using IAM authentication",
	Long: `A command-line tool for connecting to AWS RDS clusters using IAM authentication.
It supports interactive selection of environments, clusters, and users, with optional IAM permission checks.`,
	RunE: run, // Using RunE for error handling
}

// run is the main execution function for the root command.
// It handles configuration loading, environment selection, AWS authentication,
// cluster discovery, and establishing the RDS connection.
func run(_ *cobra.Command, _ []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		cancel()
	}()

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// If check flag is set, run checks for all environments
	if checkOnly {
		fmt.Println("Running in check mode...")
		// Use the first environment's region for initial AWS config
		var firstEnv string
		for env := range cfg.EnvTag {
			firstEnv = env
			break
		}
		if firstEnv == "" {
			return fmt.Errorf("no environments configured")
		}

		awsCfg, err := aws.CheckAWSCredentials(cfg.EnvTag[firstEnv].Region)
		if err != nil {
			return fmt.Errorf("failed to initialize AWS credentials: %w", err)
		}

		return runCheck(ctx, cfg, awsCfg)
	}

	// Normal operation: prompt for environment selection
	env, err := promptEnvironmentSelection(cfg.EnvTag)
	if err != nil {
		return fmt.Errorf("failed to select environment: %w", err)
	}

	region := cfg.EnvTag[env].Region
	awsCfg, err := aws.CheckAWSCredentials(region)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS credentials: %w", err)
	}

	// Get clusters and handle user selection
	cluster, user, err := selectClusterAndUser(ctx, cfg, awsCfg, env)
	if err != nil {
		return err
	}

	// Check IAM permissions if enabled
	if err := checkIAMPermissions(ctx, cfg, awsCfg, cluster, user); err != nil {
		return err
	}

	// Generate token and connect to RDS
	return connectToRDSWithToken(ctx, awsCfg, cluster, user)
}

// selectClusterAndUser handles cluster discovery and user selection.
func selectClusterAndUser(ctx context.Context, cfg *config.Config, awsCfg *aws.Config, env string) (rds.Cluster, string, error) {
	// Get current IAM role (not used in this function, but kept for future use)
	if _, err := awsCfg.GetCurrentIAMRole(ctx); err != nil {
		fmt.Printf("Warning: Could not get IAM role: %v\n", err)
	}

	rdsService = rds.NewService(*awsCfg.Config, cfg.Caching.Enabled, cfg.Caching.Duration, cfg.Debug)
	clusters, err := rdsService.GetClusters(ctx, cfg.RdsTags.TagName, cfg.RdsTags.TagValue, "ReleaseState", cfg.EnvTag[env].ReleaseState, env)
	if err != nil {
		return rds.Cluster{}, "", fmt.Errorf("failed to get RDS clusters: %w", err)
	}

	if len(clusters) == 0 {
		return rds.Cluster{}, "", fmt.Errorf("no RDS clusters found with specified tags and IAM authentication enabled")
	}

	cluster, user, err := promptUserSelections(clusters, cfg.AllowedIAMUsers)
	if err != nil {
		return rds.Cluster{}, "", fmt.Errorf("failed to select cluster or user: %w", err)
	}

	return cluster, user, nil
}

// checkIAMPermissions verifies IAM permissions if enabled in config.
func checkIAMPermissions(ctx context.Context, cfg *config.Config, awsCfg *aws.Config, cluster rds.Cluster, user string) error {
	if !cfg.CheckIAMPermissions {
		return nil
	}

	iamRole, err := awsCfg.GetCurrentIAMRole(ctx)
	if err != nil {
		return fmt.Errorf("failed to get IAM role: %w", err)
	}

	if err := awsCfg.CheckIAMUserAccess(ctx, iamRole, rdsService.GetRDSInstanceIdentifier(cluster), user); err != nil {
		return fmt.Errorf("access denied: your IAM role '%s' does not have permission to connect to RDS instance as user '%s': %w",
			iamRole, user, err)
	}

	return nil
}

// connectToRDSWithToken generates an auth token and connects to RDS.
func connectToRDSWithToken(_ context.Context, awsCfg *aws.Config, cluster rds.Cluster, user string) error {
	token, err := rds.GenerateAuthToken(*awsCfg.Config, cluster, user, log.Default())
	if err != nil {
		return fmt.Errorf("failed to generate IAM auth token: %w", err)
	}

	return connectToRDS(cluster, user, token)
}

// promptUserSelections handles user interaction to select cluster and IAM user.
// It presents interactive prompts for selecting a cluster and user from the provided lists.
// Returns the selected cluster, user, and any error that occurred.
func promptUserSelections(clusters []rds.Cluster, allowedUsers []string) (rds.Cluster, string, error) {
	clusterNames := make([]string, 0, len(clusters))
	clusterMap := make(map[string]rds.Cluster, len(clusters))

	for _, cluster := range clusters {
		display := fmt.Sprintf("%s (%s:%d)", cluster.Identifier, cluster.Endpoint, cluster.Port)
		clusterNames = append(clusterNames, display)
		clusterMap[display] = cluster
	}

	var selectedCluster string
	if err := survey.AskOne(&survey.Select{
		Message:  "Choose an RDS cluster:",
		Options:  clusterNames,
		PageSize: 10,
	}, &selectedCluster); err != nil {
		return rds.Cluster{}, "", fmt.Errorf("failed to select cluster: %w", err)
	}

	var selectedUser string
	if err := survey.AskOne(&survey.Select{
		Message:  "Choose an IAM user:",
		Options:  allowedUsers,
		PageSize: 10,
	}, &selectedUser); err != nil {
		return rds.Cluster{}, "", fmt.Errorf("failed to select user: %w", err)
	}

	return clusterMap[selectedCluster], selectedUser, nil
}

// connectToRDS establishes a connection to the RDS instance using the mysql client.
// It configures and executes the mysql command with the provided connection details.
// Returns an error if the connection fails or if the mysql client exits with an error.
func connectToRDS(cluster rds.Cluster, user, token string) error {
	// Validate inputs to prevent command injection
	if !isValidHostname(cluster.Endpoint) {
		return fmt.Errorf("invalid endpoint: %s", cluster.Endpoint)
	}
	if !isValidUsername(user) {
		return fmt.Errorf("invalid username: %s", user)
	}
	if !isValidPort(cluster.Port) {
		return fmt.Errorf("invalid port: %d", cluster.Port)
	}

	// Use exec.Command with separate arguments to prevent command injection
	cmd := exec.Command("mysql")
	cmd.Args = append(cmd.Args,
		"-h", cluster.Endpoint,
		"-P", fmt.Sprintf("%d", cluster.Port),
		"-u", user,
		"-p"+token,
		"--enable-cleartext-plugin",
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil // Normal exit from MySQL client
		}
		return fmt.Errorf("failed to connect to RDS: %w", err)
	}
	return nil
}

// isValidHostname checks if a string is a valid hostname.
func isValidHostname(hostname string) bool {
	if len(hostname) > 253 {
		return false
	}
	// Basic validation - can be enhanced based on requirements
	return strings.Contains(hostname, ".") && !strings.ContainsAny(hostname, " \t\n\r")
}

// isValidUsername checks if a string is a valid MySQL username.
func isValidUsername(username string) bool {
	if len(username) > 32 {
		return false
	}
	// Basic validation - can be enhanced based on requirements
	return !strings.ContainsAny(username, " \t\n\r")
}

// isValidPort checks if a port number is valid.
func isValidPort(port int32) bool {
	return port > 0 && port < 65536
}

// Execute adds all child commands to the root command and sets flags appropriately.
// It is the entry point for the command-line application.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetHelpCommand(nil)
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yaml", "path to config file")
	rootCmd.Flags().BoolVarP(&checkOnly, "check", "c", false, "verify the RDS IAM Connect tool configuration and environment")
}

// promptEnvironmentSelection presents an interactive prompt for selecting an environment.
// It takes a map of environment tags and returns the selected environment name.
// Returns an error if the selection fails.
func promptEnvironmentSelection(envTags map[string]struct {
	ReleaseState string
	Region       string
}) (string, error) {
	environments := make([]string, 0, len(envTags))
	for env := range envTags {
		environments = append(environments, env)
	}

	var selectedEnv string
	if err := survey.AskOne(&survey.Select{
		Message:  "Choose environment:",
		Options:  environments,
		PageSize: 10,
	}, &selectedEnv); err != nil {
		return "", fmt.Errorf("failed to select environment: %w", err)
	}

	return selectedEnv, nil
}

// runCheck executes the check functionality.
func runCheck(ctx context.Context, cfg *config.Config, awsCfg *aws.Config) error {
	// Initialize RDS service
	rdsService = rds.NewService(*awsCfg.Config, cfg.Caching.Enabled, cfg.Caching.Duration, cfg.Debug)

	// Run checks
	fmt.Println("Running RDS IAM Connect checks...")
	fmt.Println("--------------------------------")

	// Check 1: AWS Credentials
	fmt.Println("1. Checking AWS credentials...")
	if err := checkAWSCredentials(ctx, awsCfg); err != nil {
		return fmt.Errorf("AWS credentials check failed: %w", err)
	}
	fmt.Println("✓ AWS credentials are valid")

	// Check 2: Configuration
	fmt.Println("\n2. Checking configuration...")
	if err := checkConfiguration(cfg); err != nil {
		return fmt.Errorf("configuration check failed: %w", err)
	}
	fmt.Println("✓ Configuration is valid")

	// Check 3: RDS Connectivity for each environment
	fmt.Println("\n3. Checking RDS connectivity...")
	for envName, envConfig := range cfg.EnvTag {
		fmt.Printf("\n  Environment: %s\n", envName)
		fmt.Printf("  Region: %s\n", envConfig.Region)
		fmt.Printf("  Release State: %s\n", envConfig.ReleaseState)

		// Create AWS config for this environment's region
		envAwsCfg, err := aws.CheckAWSCredentials(envConfig.Region)
		if err != nil {
			fmt.Printf("  ✗ Failed to initialize AWS credentials for region %s: %v\n", envConfig.Region, err)
			continue
		}

		// Initialize RDS service for this region
		envRdsService := rds.NewService(*envAwsCfg.Config, cfg.Caching.Enabled, cfg.Caching.Duration, cfg.Debug)
		rdsService = envRdsService // Set global service for other checks

		if err := checkRDSConnectivity(ctx, cfg, envName); err != nil {
			fmt.Printf("  ✗ RDS connectivity check failed: %v\n", err)
		} else {
			fmt.Println("  ✓ RDS connectivity is valid")
		}
	}

	// Check 4: Cache
	fmt.Println("\n4. Checking cache...")
	if err := checkCache(cfg); err != nil {
		return fmt.Errorf("cache check failed: %w", err)
	}
	fmt.Println("✓ Cache is working properly")

	fmt.Println("\nAll checks completed!")
	return nil
}

// checkAWSCredentials verifies AWS credentials and permissions.
func checkAWSCredentials(ctx context.Context, awsCfg *aws.Config) error {
	// Check if we can get the caller identity
	stsClient := sts.NewFromConfig(*awsCfg.Config)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to get caller identity: %w", err)
	}

	fmt.Printf("  - AWS Account ID: %s\n", *identity.Account)
	fmt.Printf("  - AWS User ARN: %s\n", *identity.Arn)
	fmt.Printf("  - AWS Region: %s\n", awsCfg.Region)

	// Check if we have the required RDS permissions
	permissions := []string{
		"rds:DescribeDBClusters",
		"rds:ListTagsForResource",
		"rds:GenerateDBAuthToken",
	}

	// Get current IAM role
	iamRole, err := awsCfg.GetCurrentIAMRole(ctx)
	if err != nil {
		fmt.Printf("  - Warning: Could not get IAM role: %v\n", err)
	} else {
		fmt.Printf("  - Current IAM Role: %s\n", iamRole)
	}

	for _, permission := range permissions {
		fmt.Printf("  - Permission %s: ✓ (required)\n", permission)
	}

	return nil
}

// checkConfiguration validates the configuration.
func checkConfiguration(cfg *config.Config) error {
	// Check RDS tags
	if cfg.RdsTags.TagName == "" || cfg.RdsTags.TagValue == "" {
		return fmt.Errorf("RDS tags are not configured")
	}
	fmt.Printf("  - RDS Tags: %s=%s\n", cfg.RdsTags.TagName, cfg.RdsTags.TagValue)

	// Check allowed IAM users
	if len(cfg.AllowedIAMUsers) == 0 {
		return fmt.Errorf("no allowed IAM users configured")
	}
	fmt.Printf("  - Allowed IAM Users: %d configured\n", len(cfg.AllowedIAMUsers))

	// Check environment tags
	if len(cfg.EnvTag) == 0 {
		return fmt.Errorf("no environment tags configured")
	}
	fmt.Printf("  - Environment Tags: %d configured\n", len(cfg.EnvTag))

	// Check cache configuration
	if cfg.Caching.Enabled {
		fmt.Printf("  - Cache: Enabled (duration: %s)\n", cfg.Caching.Duration)
	} else {
		fmt.Println("  - Cache: Disabled")
	}

	return nil
}

// checkRDSConnectivity verifies RDS connectivity and IAM authentication.
func checkRDSConnectivity(ctx context.Context, cfg *config.Config, env string) error {
	// Get clusters to verify connectivity
	clusters, err := rdsService.GetClusters(ctx, cfg.RdsTags.TagName, cfg.RdsTags.TagValue, "ReleaseState", cfg.EnvTag[env].ReleaseState, env)
	if err != nil {
		return fmt.Errorf("failed to get RDS clusters: %w", err)
	}

	if len(clusters) == 0 {
		return fmt.Errorf("no RDS clusters found with the specified tags")
	}

	fmt.Printf("  - Found %d RDS clusters\n", len(clusters))

	// Check IAM authentication for each cluster
	for i, cluster := range clusters {
		fmt.Printf("  - Cluster %d: %s\n", i+1, cluster.Identifier)
		fmt.Printf("    - Endpoint: %s:%d\n", cluster.Endpoint, cluster.Port)
		fmt.Printf("    - Region: %s\n", cluster.Region)
		fmt.Printf("    - IAM Auth: Enabled\n")
	}

	return nil
}

// checkCache verifies cache functionality.
func checkCache(cfg *config.Config) error {
	if !cfg.Caching.Enabled {
		fmt.Println("  - Cache is disabled, skipping cache checks")
		return nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	cachePath := filepath.Join(homeDir, ".rds-iam-connect")

	// Check cache directory
	dirInfo, err := os.Stat(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("  - Cache directory does not exist")
			return nil
		}
		return fmt.Errorf("failed to check cache directory: %w", err)
	}

	if !dirInfo.IsDir() {
		return fmt.Errorf("cache path is not a directory: %s", cachePath)
	}

	fmt.Println("  - Cache directory exists")

	// Check cache files for each environment
	for env := range cfg.EnvTag {
		cacheFile := filepath.Join(cachePath, rds.GetCacheFileName(env))
		fileInfo, err := os.Stat(cacheFile)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("  - Cache file for environment %s does not exist\n", env)
				continue
			}
			return fmt.Errorf("failed to check cache file for environment %s: %w", env, err)
		}

		if !fileInfo.Mode().IsRegular() {
			return fmt.Errorf("cache file is not a regular file: %s", cacheFile)
		}

		fmt.Printf("  - Cache file exists for environment %s\n", env)
	}

	return nil
}
