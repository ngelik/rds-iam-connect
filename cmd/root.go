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

	"log"

	"github.com/eiannone/keyboard"
	"github.com/spf13/cobra"
)

var (
	configPath string
	rdsService *rds.DatabaseService
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

	// Add environment selection
	releaseState, err := promptEnvironmentSelection(cfg.EnvTag)
	if err == ErrUserCanceled {
		fmt.Println("\nOperation canceled by user")
		return nil
	} else if err != nil {
		return err
	}

	// Initialize AWS session
	awsCfg, err := aws.CheckAWSCredentials()
	if err != nil {
		return fmt.Errorf("checking AWS credentials: %w", err)
	}

	rdsService = rds.NewService(*awsCfg.Config)
	clusters, err := rdsService.GetClusters(ctx, cfg.RdsTags.TagName, cfg.RdsTags.TagValue, "ReleaseState", releaseState)
	if err != nil {
		return err
	}

	if len(clusters) == 0 {
		return fmt.Errorf("no RDS clusters found with specified tags and IAM authentication enabled")
	}

	cluster, user, err := promptUserSelections(clusters, cfg.AllowedIAMUsers)
	if err == ErrUserCanceled {
		fmt.Println("\nOperation canceled by user")
		return nil
	} else if err != nil {
		return err
	}

	// Generate IAM Auth Token
	token, err := rds.GenerateAuthToken(*awsCfg.Config, cluster, user, log.Default())
	if err != nil {
		return fmt.Errorf("generating IAM auth token: %w", err)
	}

	return connectToRDS(cluster, user, token)
}

// promptUserSelections handles user interaction to select cluster and IAM user
func promptUserSelections(clusters []rds.Cluster, users []string) (rds.Cluster, string, error) {
	if err := keyboard.Open(); err != nil {
		return rds.Cluster{}, "", fmt.Errorf("failed to open keyboard: %w", err)
	}
	defer keyboard.Close()

	// First, select cluster
	for {
		fmt.Println("\nSelect RDS cluster:")
		for i, cluster := range clusters {
			fmt.Printf("%d) %s\n", i+1, cluster.Identifier)
		}
		fmt.Println("\nPress Backspace to return to previous menu")

		char, key, err := keyboard.GetSingleKey()
		if err != nil {
			return rds.Cluster{}, "", err
		}

		if key == keyboard.KeyBackspace || key == keyboard.KeyDelete || key == keyboard.KeyEsc {
			return rds.Cluster{}, "", ErrUserCanceled
		}

		num := int(char - '0')
		if num > 0 && num <= len(clusters) {
			selectedCluster := clusters[num-1]

			// Then select user
			for {
				fmt.Println("\nSelect user:")
				for i, user := range users {
					fmt.Printf("%d) %s\n", i+1, user)
				}
				fmt.Println("\nPress Backspace to return to cluster selection")

				char, key, err := keyboard.GetSingleKey()
				if err != nil {
					return rds.Cluster{}, "", err
				}

				if key == keyboard.KeyBackspace || key == keyboard.KeyDelete || key == keyboard.KeyEsc {
					break // Return to cluster selection
				}

				num := int(char - '0')
				if num > 0 && num <= len(users) {
					return selectedCluster, users[num-1], nil
				}
			}
		}
	}
}

// connectToRDS establishes a connection to the RDS instance using the mysql client
func connectToRDS(cluster rds.Cluster, user, token string) error {
	fmt.Printf("Generated IAM Auth Token: %s\n", token) // Debug statement for token
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

func promptEnvironmentSelection(envTags []string) (string, error) {
	if err := keyboard.Open(); err != nil {
		return "", fmt.Errorf("failed to open keyboard: %w", err)
	}
	defer keyboard.Close()

	for {
		fmt.Println("\nSelect environment:")
		for i, env := range envTags {
			fmt.Printf("%d) %s\n", i+1, env)
		}
		fmt.Println("\nPress Backspace to return to previous menu")

		char, key, err := keyboard.GetSingleKey()
		if err != nil {
			return "", err
		}

		// Check for backspace/delete to return
		if key == keyboard.KeyBackspace || key == keyboard.KeyDelete || key == keyboard.KeyEsc {
			return "", ErrUserCanceled // Define this error type in your package
		}

		// Convert char to number and validate
		num := int(char - '0')
		if num > 0 && num <= len(envTags) {
			return envTags[num-1], nil
		}
	}
}

// Add this error type at the package level
var ErrUserCanceled = fmt.Errorf("user canceled operation")
