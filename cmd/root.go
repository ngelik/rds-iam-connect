package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"

	"rds-iam-connect/config"
	"rds-iam-connect/internal/aws"
	"rds-iam-connect/internal/rds"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

var (
	configPath string
	rdsService rds.DatabaseService
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rds-iam-connect",
	Short: "Connect to AWS RDS clusters using IAM authentication",
	RunE:  run, // Using RunE for error handling
}

func run(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("loading config: %w", err)
	}

	// Initialize AWS session
	awsCfg, err := aws.CheckAWSCredentials()
	if err != nil {
		return fmt.Errorf("checking AWS credentials: %w", err)
	}

	rdsService = rds.NewService(*awsCfg.Config)
	clusters, err := rdsService.GetClusters(ctx, cfg.AWS.TagName, cfg.AWS.TagValue)
	if err != nil {
		return err
	}

	if len(clusters) == 0 {
		return fmt.Errorf("no RDS clusters found with specified tags and IAM authentication enabled")
	}

	cluster, user, err := promptUserSelections(clusters, cfg.AllowedIAMUsers)
	if err != nil {
		return err
	}

	// Generate IAM Auth Token
	token, err := rds.GenerateAuthToken(*awsCfg.Config, cluster, user)
	if err != nil {
		return fmt.Errorf("generating IAM auth token: %w", err)
	}

	return connectToRDS(cluster, user, token)
}

// promptUserSelections handles user interaction to select cluster and IAM user
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
		return rds.Cluster{}, "", fmt.Errorf("cluster selection failed: %w", err)
	}

	var selectedUser string
	if err := survey.AskOne(&survey.Select{
		Message:  "Choose an IAM user:",
		Options:  allowedUsers,
		PageSize: 10,
	}, &selectedUser); err != nil {
		return rds.Cluster{}, "", fmt.Errorf("user selection failed: %w", err)
	}

	return clusterMap[selectedCluster], selectedUser, nil
}

// connectToRDS establishes a connection to the RDS instance using the mysql client
func connectToRDS(cluster rds.Cluster, user, token string) error {
	cmd := exec.Command("mysql",
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
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil // Normal exit from MySQL client
		}
		return fmt.Errorf("connecting to RDS: %w", err)
	}
	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yaml", "path to config file")
}