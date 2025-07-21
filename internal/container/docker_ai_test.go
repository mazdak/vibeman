package container

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCommandExecutor for testing
type MockCommandExecutor struct {
	commands []MockCommand
	index    int
}

type MockCommand struct {
	expectedCmd  string
	expectedArgs []string
	output       string
	err          error
}

func (m *MockCommandExecutor) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	if m.index >= len(m.commands) {
		panic(fmt.Sprintf("unexpected command: %s %v", name, args))
	}

	expected := m.commands[m.index]
	m.index++

	// Create a command that will return our mocked output
	cmd := exec.Command("echo", expected.output)
	if expected.err != nil {
		// For error cases, use a command that will fail
		cmd = exec.Command("false")
	}

	return cmd
}

// TestDockerRuntime_CreateWithType tests container creation with Type field
func TestDockerRuntime_CreateWithType(t *testing.T) {
	tests := []struct {
		name           string
		config         *CreateConfig
		expectedType   string
		expectedLabels []string
	}{
		{
			name: "AI container",
			config: &CreateConfig{
				Name:        "test-repo-feature-ai",
				Image:       "vibeman/ai-assistant:latest",
				Repository:  "test-repo",
				Environment: "feature",
				Type:        "ai",
			},
			expectedType: "ai",
			expectedLabels: []string{
				"--label", "vibeman.type=ai",
			},
		},
		{
			name: "Worktree container",
			config: &CreateConfig{
				Name:        "test-repo-feature",
				Image:       "node:18",
				Repository:  "test-repo",
				Environment: "feature",
				Type:        "worktree",
			},
			expectedType: "worktree",
			expectedLabels: []string{
				"--label", "vibeman.type=worktree",
			},
		},
		{
			name: "Service container",
			config: &CreateConfig{
				Name:        "postgres",
				Image:       "postgres:15",
				Repository:  "",
				Environment: "",
				Type:        "service",
			},
			expectedType: "service",
			expectedLabels: []string{
				"--label", "vibeman.type=service",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockCommandExecutor{
				commands: []MockCommand{
					{
						expectedCmd:  "docker",
						expectedArgs: []string{"run", "-d", "--name", tt.config.Name},
						output:       "container-123",
						err:          nil,
					},
				},
			}

			runtime := NewDockerRuntime(mockExecutor)
			container, err := runtime.Create(context.Background(), tt.config)

			require.NoError(t, err)
			assert.NotNil(t, container)
			assert.Equal(t, "container-123", container.ID)
			assert.Equal(t, tt.config.Name, container.Name)
			assert.Equal(t, tt.config.Type, container.Type)
		})
	}
}

// TestDockerRuntime_ListWithType tests container listing with Type field
func TestDockerRuntime_ListWithType(t *testing.T) {
	// Mock docker ps output
	dockerPsOutput := `{"ID":"container1","Names":"test-repo-feature-ai","Image":"vibeman/ai-assistant:latest","Status":"Up 2 hours"}
{"ID":"container2","Names":"test-repo-feature","Image":"node:18","Status":"Up 1 hour"}
{"ID":"container3","Names":"postgres","Image":"postgres:15","Status":"Up 3 hours"}`

	// Mock docker inspect outputs for labels
	labelsContainer1 := map[string]string{
		"vibeman.type":       "ai",
		"vibeman.repository": "test-repo",
		"vibeman.environment": "feature",
		"vibeman.managed":    "true",
	}

	labelsContainer2 := map[string]string{
		"vibeman.type":       "worktree",
		"vibeman.repository": "test-repo",
		"vibeman.environment": "feature",
		"vibeman.managed":    "true",
	}

	labelsContainer3 := map[string]string{
		"vibeman.type":    "service",
		"vibeman.managed": "true",
	}

	mockExecutor := &MockCommandExecutor{
		commands: []MockCommand{
			// docker ps command
			{
				expectedCmd:  "docker",
				expectedArgs: []string{"ps", "-a", "--format", "json"},
				output:       dockerPsOutput,
				err:          nil,
			},
			// docker inspect for env vars - container1
			{
				expectedCmd:  "docker",
				expectedArgs: []string{"inspect", "container1", "--format", "{{json .Config.Env}}"},
				output:       `["VIBEMAN_REPOSITORY=test-repo","VIBEMAN_ENV=feature"]`,
				err:          nil,
			},
			// docker inspect for labels - container1
			{
				expectedCmd:  "docker",
				expectedArgs: []string{"inspect", "container1", "--format", "{{json .Config.Labels}}"},
				output:       mustMarshalJSON(labelsContainer1),
				err:          nil,
			},
			// docker inspect for env vars - container2
			{
				expectedCmd:  "docker",
				expectedArgs: []string{"inspect", "container2", "--format", "{{json .Config.Env}}"},
				output:       `["VIBEMAN_REPOSITORY=test-repo","VIBEMAN_ENV=feature"]`,
				err:          nil,
			},
			// docker inspect for labels - container2
			{
				expectedCmd:  "docker",
				expectedArgs: []string{"inspect", "container2", "--format", "{{json .Config.Labels}}"},
				output:       mustMarshalJSON(labelsContainer2),
				err:          nil,
			},
			// docker inspect for env vars - container3
			{
				expectedCmd:  "docker",
				expectedArgs: []string{"inspect", "container3", "--format", "{{json .Config.Env}}"},
				output:       `[]`,
				err:          nil,
			},
			// docker inspect for labels - container3
			{
				expectedCmd:  "docker",
				expectedArgs: []string{"inspect", "container3", "--format", "{{json .Config.Labels}}"},
				output:       mustMarshalJSON(labelsContainer3),
				err:          nil,
			},
		},
	}

	runtime := NewDockerRuntime(mockExecutor)
	containers, err := runtime.List(context.Background())

	require.NoError(t, err)
	require.Len(t, containers, 3)

	// Verify container types
	assert.Equal(t, "ai", containers[0].Type)
	assert.Equal(t, "test-repo-feature-ai", containers[0].Name)

	assert.Equal(t, "worktree", containers[1].Type)
	assert.Equal(t, "test-repo-feature", containers[1].Name)

	assert.Equal(t, "service", containers[2].Type)
	assert.Equal(t, "postgres", containers[2].Name)
}

// TestDockerRuntime_ListWithoutTypeLabel tests backward compatibility for containers without type label
func TestDockerRuntime_ListWithoutTypeLabel(t *testing.T) {
	dockerPsOutput := `{"ID":"container1","Names":"old-container","Image":"node:16","Status":"Up 1 day"}`

	// Container without vibeman.type label
	labelsContainer1 := map[string]string{
		"vibeman.repository":  "old-repo",
		"vibeman.environment": "main",
		"vibeman.managed":     "true",
	}

	mockExecutor := &MockCommandExecutor{
		commands: []MockCommand{
			// docker ps command
			{
				expectedCmd:  "docker",
				expectedArgs: []string{"ps", "-a", "--format", "json"},
				output:       dockerPsOutput,
				err:          nil,
			},
			// docker inspect for env vars
			{
				expectedCmd:  "docker",
				expectedArgs: []string{"inspect", "container1", "--format", "{{json .Config.Env}}"},
				output:       `["VIBEMAN_REPOSITORY=old-repo","VIBEMAN_ENV=main"]`,
				err:          nil,
			},
			// docker inspect for labels
			{
				expectedCmd:  "docker",
				expectedArgs: []string{"inspect", "container1", "--format", "{{json .Config.Labels}}"},
				output:       mustMarshalJSON(labelsContainer1),
				err:          nil,
			},
		},
	}

	runtime := NewDockerRuntime(mockExecutor)
	containers, err := runtime.List(context.Background())

	require.NoError(t, err)
	require.Len(t, containers, 1)

	// Should default to "worktree" for backward compatibility
	assert.Equal(t, "worktree", containers[0].Type)
}

// Helper function to marshal JSON
func mustMarshalJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}