package cli

import (
	"fmt"
	"rds-iam-connect/internal/rds"

	"github.com/AlecAivazis/survey/v2"
)

type Prompter interface {
	SelectCluster(clusters []rds.Cluster) (rds.Cluster, error)
	SelectUser(users []string) (string, error)
}

type SurveyPrompter struct{}

func NewPrompter() Prompter {
	return &SurveyPrompter{}
}

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
