package cli

import (
	"testing"

	"rds-iam-connect/internal/rds"

	"github.com/stretchr/testify/assert"
)

// MockPrompter is a mock implementation of the Prompter interface.
type MockPrompter struct {
	selectedCluster rds.Cluster
	selectedUser    string
}

func (m *MockPrompter) SelectCluster(_ []rds.Cluster) (rds.Cluster, error) {
	return m.selectedCluster, nil
}

func (m *MockPrompter) SelectUser(_ []string) (string, error) {
	return m.selectedUser, nil
}

func TestSelectCluster(t *testing.T) {
	mockPrompter := &MockPrompter{
		selectedCluster: rds.Cluster{
			Identifier: "test-cluster-1",
			Endpoint:   "test-cluster-1.xxxxx.us-west-2.rds.amazonaws.com",
			Port:       3306,
			Region:     "us-west-2",
		},
	}

	cli := NewCLI(mockPrompter)

	clusters := []rds.Cluster{
		{
			Identifier: "test-cluster-1",
			Endpoint:   "test-cluster-1.xxxxx.us-west-2.rds.amazonaws.com",
			Port:       3306,
			Region:     "us-west-2",
		},
		{
			Identifier: "test-cluster-2",
			Endpoint:   "test-cluster-2.xxxxx.us-west-2.rds.amazonaws.com",
			Port:       3306,
			Region:     "us-west-2",
		},
	}
	selected, err := cli.SelectCluster(clusters)

	assert.NoError(t, err)
	assert.Equal(t, "test-cluster-1", selected.Identifier)
}

func TestSelectUser(t *testing.T) {
	mockPrompter := &MockPrompter{
		selectedUser: "test-user",
	}

	cli := NewCLI(mockPrompter)

	users := []string{"test-user", "admin"}
	selected, err := cli.SelectUser(users)

	assert.NoError(t, err)
	assert.Equal(t, "test-user", selected)
}
