package commands

import (
	"context"
	"testing"

	"vibeman/internal/config"
	"vibeman/internal/db"
	"vibeman/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockContainerManager for testing AI commands
type MockContainerManager struct {
	mock.Mock
}

func (m *MockContainerManager) Create(ctx context.Context, repositoryName, environment, image string) (*types.Container, error) {
	args := m.Called(ctx, repositoryName, environment, image)
	return args.Get(0).(*types.Container), args.Error(1)
}

func (m *MockContainerManager) Start(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockContainerManager) Stop(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockContainerManager) Remove(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockContainerManager) List(ctx context.Context) ([]*types.Container, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*types.Container), args.Error(1)
}

func (m *MockContainerManager) GetByName(ctx context.Context, name string) (*types.Container, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*types.Container), args.Error(1)
}

func (m *MockContainerManager) GetByRepository(ctx context.Context, repository string) ([]*types.Container, error) {
	args := m.Called(ctx, repository)
	return args.Get(0).([]*types.Container), args.Error(1)
}

func (m *MockContainerManager) Exec(ctx context.Context, containerID string, command []string) ([]byte, error) {
	args := m.Called(ctx, containerID, command)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockContainerManager) Logs(ctx context.Context, containerID string, follow bool) ([]byte, error) {
	args := m.Called(ctx, containerID, follow)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockContainerManager) Shell(ctx context.Context, containerID string, shell string) error {
	args := m.Called(ctx, containerID, shell)
	return args.Error(0)
}

func (m *MockContainerManager) SSH(ctx context.Context, containerID string, user string) error {
	args := m.Called(ctx, containerID, user)
	return args.Error(0)
}

func (m *MockContainerManager) Attach(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockContainerManager) CopyToContainer(ctx context.Context, containerID, srcPath, dstPath string) error {
	args := m.Called(ctx, containerID, srcPath, dstPath)
	return args.Error(0)
}

func (m *MockContainerManager) CopyFromContainer(ctx context.Context, containerID, srcPath, dstPath string) error {
	args := m.Called(ctx, containerID, srcPath, dstPath)
	return args.Error(0)
}

func (m *MockContainerManager) RunSetup(ctx context.Context, containerID string, repositoryPath string) error {
	args := m.Called(ctx, containerID, repositoryPath)
	return args.Error(0)
}

func (m *MockContainerManager) RunLifecycleHook(ctx context.Context, containerID string, hook string) error {
	args := m.Called(ctx, containerID, hook)
	return args.Error(0)
}

// MockServiceManager for testing
type MockServiceManager struct {
	mock.Mock
}

func (m *MockServiceManager) StartService(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockServiceManager) StopService(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockServiceManager) AddReference(serviceName, repositoryName string) error {
	args := m.Called(serviceName, repositoryName)
	return args.Error(0)
}

func (m *MockServiceManager) RemoveReference(serviceName, repositoryName string) error {
	args := m.Called(serviceName, repositoryName)
	return args.Error(0)
}

func (m *MockServiceManager) HealthCheck(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockServiceManager) GetService(name string) (interface{}, error) {
	args := m.Called(name)
	return args.Get(0), args.Error(1)
}

// Test CreateAICommand
func TestCreateAICommand(t *testing.T) {
	cfg := &config.Manager{}
	containerMgr := &MockContainerManager{}
	serviceMgr := &MockServiceManager{}
	database := &db.DB{} // Mock database

	cmd := CreateAICommand(cfg, containerMgr, nil, serviceMgr, database)

	assert.NotNil(t, cmd)
	assert.Equal(t, "ai [worktree]", cmd.Use)
	assert.Equal(t, "Start Claude CLI in AI container", cmd.Short)

	// Check that subcommands are added
	subcommands := cmd.Commands()
	assert.Len(t, subcommands, 4) // attach, claude, list, logs

	commandNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		commandNames[i] = subcmd.Use
	}

	assert.Contains(t, commandNames, "attach [worktree-name]")
	assert.Contains(t, commandNames, "claude [worktree-name]")
	assert.Contains(t, commandNames, "list")
	assert.Contains(t, commandNames, "logs [worktree-name]")
}

// Test default behavior of ai command (should start Claude)
func TestAICommandDefaultBehavior(t *testing.T) {
	cfg := &config.Manager{}
	containerMgr := &MockContainerManager{}
	serviceMgr := &MockServiceManager{}
	database := &db.DB{} // Mock database

	// Mock containers - one running AI container
	containers := []*types.Container{
		{
			ID:     "ai-container-123",
			Name:   "repo-test-branch-ai",
			Status: "Up 5 minutes",
			Type:   "ai",
		},
	}

	containerMgr.On("List", mock.Anything).Return(containers, nil)

	cmd := CreateAICommand(cfg, containerMgr, nil, serviceMgr, database)
	
	// Verify the command has a RunE function (default behavior)
	assert.NotNil(t, cmd.RunE)
	assert.Equal(t, "Start Claude CLI in AI container", cmd.Short)
	
	// Note: We can't easily test the actual execution without mocking os.Getwd() and docker exec
	// The command structure and setup is what we're primarily testing here
}

// Test createAIListCommand
func TestCreateAIListCommand(t *testing.T) {
	containerMgr := &MockContainerManager{}

	// Mock containers
	containers := []*types.Container{
		{
			ID:     "container1",
			Name:   "repo-feature1-ai",
			Status: "running",
			Image:  "vibeman/ai-assistant:latest",
			Type:   "ai",
		},
		{
			ID:     "container2",
			Name:   "repo-main",
			Status: "running",
			Image:  "postgres:13",
			Type:   "worktree",
		},
		{
			ID:     "container3",
			Name:   "repo-feature2-ai",
			Status: "stopped",
			Image:  "vibeman/ai-assistant:latest",
			Type:   "ai",
		},
	}

	containerMgr.On("List", mock.Anything).Return(containers, nil)

	cmd := createAIListCommand(containerMgr)
	assert.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List AI containers", cmd.Short)

	// Test execution by calling RunE directly
	err := cmd.RunE(cmd, []string{})
	assert.NoError(t, err)

	containerMgr.AssertExpectations(t)
}

// Test createAIListCommand with no AI containers
func TestCreateAIListCommand_NoContainers(t *testing.T) {
	containerMgr := &MockContainerManager{}

	// Mock no containers
	containers := []*types.Container{}
	containerMgr.On("List", mock.Anything).Return(containers, nil)

	cmd := createAIListCommand(containerMgr)
	err := cmd.RunE(cmd, []string{})
	assert.NoError(t, err)

	containerMgr.AssertExpectations(t)
}

// Test createAILogsCommand
func TestCreateAILogsCommand(t *testing.T) {
	containerMgr := &MockContainerManager{}

	// Mock worktree getter
	getWorktrees := func(ctx context.Context) ([]*db.Worktree, error) {
		return []*db.Worktree{
			{Name: "feature1", Path: "/test/workspace/feature1"},
			{Name: "feature2", Path: "/test/workspace/feature2"},
		}, nil
	}

	// Mock containers
	containers := []*types.Container{
		{
			ID:   "ai-container-123",
			Name: "repo-feature1-ai",
			Type: "ai",
		},
	}

	// Mock logs
	logOutput := []byte("2024-01-25 10:30:00 [INFO] AI container started\n2024-01-25 10:30:01 [INFO] Claude CLI ready\n")

	containerMgr.On("List", mock.Anything).Return(containers, nil)
	containerMgr.On("Logs", mock.Anything, "ai-container-123", false).Return(logOutput, nil)

	cmd := createAILogsCommand(containerMgr, getWorktrees)
	assert.NotNil(t, cmd)
	assert.Equal(t, "logs [worktree-name]", cmd.Use)

	// Test execution with explicit worktree name
	err := cmd.RunE(cmd, []string{"feature1"})
	assert.NoError(t, err)

	containerMgr.AssertExpectations(t)
}

// Test helper functions
func TestExtractWorktreeFromAIContainer(t *testing.T) {
	tests := []struct {
		name          string
		containerName string
		expected      string
	}{
		{
			name:          "standard AI container name",
			containerName: "repo-feature1-ai",
			expected:      "feature1",
		},
		{
			name:          "multi-part worktree name",
			containerName: "myrepo-feature-branch-ai",
			expected:      "feature-branch",
		},
		{
			name:          "single part worktree",
			containerName: "repo-main-ai",
			expected:      "main",
		},
		{
			name:          "non-AI container",
			containerName: "repo-feature1",
			expected:      "unknown",
		},
		{
			name:          "malformed name",
			containerName: "ai",
			expected:      "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractWorktreeFromAIContainer(tt.containerName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		length   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			length:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			length:   5,
			expected: "hello",
		},
		{
			name:     "long string",
			input:    "this is a very long string",
			length:   10,
			expected: "this is...",
		},
		{
			name:     "very short length",
			input:    "hello",
			length:   3,
			expected: "hel",
		},
		{
			name:     "length of 1",
			input:    "hello",
			length:   1,
			expected: "h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.length)
			assert.Equal(t, tt.expected, result)
		})
	}
}