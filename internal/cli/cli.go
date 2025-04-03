// Package cli provides command-line interface components for the RDS IAM Connect tool.
// It handles user interaction and input validation.
package cli

import (
	"fmt"
	"rds-iam-connect/internal/rds"

	"github.com/AlecAivazis/survey/v2"
)

// Prompter defines the interface for user interaction prompts.
type Prompter interface {
	SelectCluster(clusters []rds.Cluster) (rds.Cluster, error)
	SelectUser(users []string) (string, error)
}

// SurveyPrompter implements the Prompter interface using the survey package.
type SurveyPrompter struct{}

// NewPrompter creates a new instance of SurveyPrompter.
func NewPrompter() Prompter {
	return &SurveyPrompter{}
}

// SelectCluster presents an interactive prompt for selecting an RDS cluster.
// Returns the selected cluster or an error if the selection fails.
func (p *SurveyPrompter) SelectCluster(clusters []rds.Cluster) (rds.Cluster, error) {
	clusterNames := make([]string, 0, len(clusters))
	clusterMap := make(map[string]rds.Cluster, len(clusters))

	for _, cluster := range clusters {
		display := fmt.Sprintf("%s (%s:%d)", cluster.Identifier, cluster.Endpoint, cluster.Port)
		clusterNames = append(clusterNames, display)
		clusterMap[display] = cluster
	}

	var selected string
	if err := survey.AskOne(&survey.Select{
		Message:  "Choose an RDS cluster:",
		Options:  clusterNames,
		PageSize: 10,
	}, &selected); err != nil {
		return rds.Cluster{}, err
	}

	return clusterMap[selected], nil
}

// SelectUser presents an interactive prompt for selecting an IAM user.
// Returns the selected user or an error if the selection fails.
func (p *SurveyPrompter) SelectUser(users []string) (string, error) {
	var selected string
	if err := survey.AskOne(&survey.Select{
		Message:  "Choose an IAM user:",
		Options:  users,
		PageSize: 10,
	}, &selected); err != nil {
		return "", err
	}
	return selected, nil
}

// CLI represents the command-line interface for user interaction.
type CLI struct {
	prompter Prompter
}

// NewCLI creates a new CLI instance with the given prompter.
func NewCLI(prompter Prompter) *CLI {
	return &CLI{
		prompter: prompter,
	}
}

// SelectCluster prompts the user to select a cluster from the given list.
func (c *CLI) SelectCluster(clusters []rds.Cluster) (rds.Cluster, error) {
	return c.prompter.SelectCluster(clusters)
}

// SelectUser prompts the user to select a user from the given list.
func (c *CLI) SelectUser(users []string) (string, error) {
	return c.prompter.SelectUser(users)
}
