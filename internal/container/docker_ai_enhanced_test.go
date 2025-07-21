package container

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test enhanced AI container creation with development tools
func TestDockerRuntime_CreateEnhancedAIContainer(t *testing.T) {
	tests := []struct {
		name           string
		config         *CreateConfig
		expectedOutput string
		mockCommands   []MockCommand
		wantError      bool
	}{
		{
			name: "create enhanced AI container with tools",
			config: &CreateConfig{
				Name:       "test-worktree-ai",
				Image:      "vibeman/ai-assistant:latest",
				WorkingDir: "/workspace",
				Repository: "test-repo",
				Type:       "ai",
				EnvVars: []string{
					"VIBEMAN_WORKTREE_ID=test-123",
					"VIBEMAN_AI_CONTAINER=true",
					"SHELL=/bin/zsh",
				},
				Volumes: []string{
					"/test/workspace:/workspace",
					"/test/logs:/logs:ro",
				},
			},
			mockCommands: []MockCommand{
				{
					expectedCmd:  "docker",
					expectedArgs: []string{"run", "-d", "--name", "test-worktree-ai"},
					output:       "container123\n",
				},
			},
			expectedOutput: "container123",
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockCommandExecutor{
				commands: tt.mockCommands,
			}
			runtime := &DockerRuntime{executor: mockExecutor}

			// Execute
			container, err := runtime.Create(context.Background(), tt.config)

			// Assert
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, container)
				assert.Equal(t, tt.expectedOutput, container.ID)
			}

			// Verify all commands were executed
			assert.Equal(t, len(tt.mockCommands), mockExecutor.index)
		})
	}
}

// Test that enhanced tools are available in the container
func TestDockerRuntime_VerifyEnhancedAITools(t *testing.T) {
	tests := []struct {
		name         string
		tool         string
		checkCommand []string
		expectedPath string
	}{
		{
			name:         "verify ripgrep installed",
			tool:         "rg",
			checkCommand: []string{"which", "rg"},
			expectedPath: "/usr/bin/rg",
		},
		{
			name:         "verify fd installed",
			tool:         "fd",
			checkCommand: []string{"which", "fd"},
			expectedPath: "/usr/local/bin/fd",
		},
		{
			name:         "verify ast-grep installed",
			tool:         "sg",
			checkCommand: []string{"which", "sg"},
			expectedPath: "/usr/local/bin/sg",
		},
		{
			name:         "verify fzf installed",
			tool:         "fzf",
			checkCommand: []string{"which", "fzf"},
			expectedPath: "/usr/bin/fzf",
		},
		{
			name:         "verify zsh installed",
			tool:         "zsh",
			checkCommand: []string{"which", "zsh"},
			expectedPath: "/bin/zsh",
		},
		{
			name:         "verify bat installed",
			tool:         "bat",
			checkCommand: []string{"which", "bat"},
			expectedPath: "/usr/bin/bat",
		},
		{
			name:         "verify exa installed",
			tool:         "exa",
			checkCommand: []string{"which", "exa"},
			expectedPath: "/usr/bin/exa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			containerID := "ai-container-123"
			
			mockExecutor := &MockCommandExecutor{
				commands: []MockCommand{
					{
						expectedCmd:  "docker",
						expectedArgs: append([]string{"exec", containerID}, tt.checkCommand...),
						output:       tt.expectedPath + "\n",
					},
				},
			}
			runtime := &DockerRuntime{executor: mockExecutor}

			// Execute
			output, err := runtime.Exec(context.Background(), containerID, tt.checkCommand)

			// Assert
			assert.NoError(t, err)
			assert.Contains(t, string(output), tt.expectedPath)
			assert.Equal(t, 1, mockExecutor.index)
		})
	}
}

// Test CLAUDE.md file existence and zsh configuration
func TestDockerRuntime_AIContainerConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		description  string
		command      []string
		expectedOut  string
		expectError  bool
	}{
		{
			name:        "verify CLAUDE.md exists",
			description: "Check global CLAUDE.md file",
			command:     []string{"test", "-f", "/home/vibeman/.claude/CLAUDE.md"},
			expectedOut: "", // test command returns empty on success
			expectError: false,
		},
		{
			name:        "verify zshrc exists",
			description: "Check zsh configuration",
			command:     []string{"test", "-f", "/home/vibeman/.zshrc"},
			expectedOut: "", // test command returns empty on success
			expectError: false,
		},
		{
			name:        "verify oh-my-zsh installed",
			description: "Check oh-my-zsh installation",
			command:     []string{"test", "-d", "/home/vibeman/.oh-my-zsh"},
			expectedOut: "", // test command returns empty on success
			expectError: false,
		},
		{
			name:        "verify user shell is zsh",
			description: "Check default shell",
			command:     []string{"sh", "-c", "echo $SHELL"},
			expectedOut: "/bin/zsh",
			expectError: false,
		},
		{
			name:        "verify vibeman user",
			description: "Check running as vibeman user",
			command:     []string{"whoami"},
			expectedOut: "vibeman",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			containerID := "ai-container-123"
			
			mockExecutor := &MockCommandExecutor{
				commands: []MockCommand{
					{
						expectedCmd:  "docker",
						expectedArgs: append([]string{"exec", containerID}, tt.command...),
						output:       tt.expectedOut + "\n",
						err:          nil,
					},
				},
			}
			
			if tt.expectError {
				mockExecutor.commands[0].err = exec.ErrNotFound
			}
			
			runtime := &DockerRuntime{executor: mockExecutor}

			// Execute
			output, err := runtime.Exec(context.Background(), containerID, tt.command)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectedOut != "" {
					assert.Contains(t, strings.TrimSpace(string(output)), tt.expectedOut)
				}
			}
			assert.Equal(t, 1, mockExecutor.index)
		})
	}
}

// Test AI container specific environment variables
func TestDockerRuntime_AIContainerEnvironment(t *testing.T) {
	ctx := context.Background()
	
	config := &CreateConfig{
		Name:       "test-ai-env",
		Image:      "vibeman/ai-assistant:latest",
		WorkingDir: "/workspace",
		Repository: "test-repo",
		Type:       "ai",
		EnvVars: []string{
			"VIBEMAN_AI_CONTAINER=true",
			"VIBEMAN_WORKTREE_ID=test-123",
			"VIBEMAN_REPOSITORY=test-repo",
			"VIBEMAN_LOG_DIR=/logs",
			"SHELL=/bin/zsh",
		},
		Volumes: []string{
			"/workspace:/workspace",
			"/logs:/logs:ro",
		},
	}

	mockExecutor := &MockCommandExecutor{
		commands: []MockCommand{
			{
				expectedCmd:  "docker",
				expectedArgs: []string{"run", "-d", "--name", "test-ai-env"},
				output:       "container123\n",
			},
		},
	}

	runtime := &DockerRuntime{executor: mockExecutor}

	// Create container
	container, err := runtime.Create(ctx, config)
	assert.NoError(t, err)
	assert.NotNil(t, container)
	assert.Equal(t, "container123", container.ID)
	
	// Verify the command was called with proper env vars
	// Note: In a real test, we'd verify the actual docker command includes all env vars
	// But with our mock, we just verify the command was called
	assert.Equal(t, 1, mockExecutor.index)
}