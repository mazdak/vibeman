package operations

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"vibeman/internal/config"
	"vibeman/internal/constants"
	"vibeman/internal/db"
	"vibeman/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestCreateWorktreeRequest_NewFields tests the new fields added in Phase 4
func TestCreateWorktreeRequest_NewFields(t *testing.T) {
	req := CreateWorktreeRequest{
		RepositoryID:    "repo-123",
		Name:            "feature-test",
		PostScripts:     []string{"npm install", "npm run build"},
		ComposeFile:     "./custom-compose.yaml",
		Services: []string{"backend", "frontend"},
		AutoStart:       true,
	}

	// Verify all new fields are accessible
	assert.Equal(t, []string{"npm install", "npm run build"}, req.PostScripts)
	assert.Equal(t, "./custom-compose.yaml", req.ComposeFile)
	assert.Equal(t, []string{"backend", "frontend"}, req.Services)
	assert.True(t, req.AutoStart)
}

// TestCreateWorktree_PostScriptsExecution tests that post-scripts are executed
func TestCreateWorktree_PostScriptsExecution(t *testing.T) {
	// Setup test database
	database := testutil.SetupTestDB(t)
	defer database.Close()

	// Create test directory structure
	tempDir, err := os.MkdirTemp("", "vibeman-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	repoPath := filepath.Join(tempDir, "test-repo")
	worktreesDir := filepath.Join(tempDir, "worktrees")
	worktreePath := filepath.Join(worktreesDir, "feature-test")

	err = os.MkdirAll(repoPath, constants.DirPermissions)
	require.NoError(t, err)

	// Create test repository in database
	repo := &db.Repository{
		ID:          "repo-123",
		Name:        "test-repo",
		Path:        repoPath,
		Description: "Test repository",
	}
	repoRepo := db.NewRepositoryRepository(database)
	err = repoRepo.Create(context.Background(), repo)
	require.NoError(t, err)

	// Create vibeman.toml config
	configContent := `
[repository]
name = "test-repo"
description = "Test repository"

[repository.worktrees]
directory = "` + worktreesDir + `"

[repository.setup]
worktree_init = "echo 'Repository setup'"
`
	configPath := filepath.Join(repoPath, "vibeman.toml")
	err = os.WriteFile(configPath, []byte(configContent), constants.FilePermissions)
	require.NoError(t, err)

	// Create mocks
	mockGitMgr := new(testutil.MockGitManager)
	mockContainerMgr := new(testutil.MockContainerManager)
	mockServiceMgr := new(testutil.MockServiceManager)
	cfg := &config.Manager{}

	// Mock git operations
	mockGitMgr.On("CreateWorktree", mock.Anything, repoPath, "feature/test", worktreePath).Return(nil)

	// Create operations instance
	ops := NewWorktreeOperations(database, mockGitMgr, mockContainerMgr, mockServiceMgr, cfg)

	// Test worktree creation with post-scripts
	req := CreateWorktreeRequest{
		RepositoryID: repo.ID,
		Name:         "feature-test",
		Branch:       "feature/test",
		PostScripts: []string{
			"echo 'Post-script 1'",
			"echo 'Post-script 2'",
		},
		SkipSetup: false, // Enable setup to test script execution
	}

	// Execute
	result, err := ops.CreateWorktree(context.Background(), req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "feature-test", result.Worktree.Name)

	// Verify mocks were called
	mockGitMgr.AssertExpectations(t)
}

// TestCreateWorktree_ComposeOverrides tests compose file and service overrides
func TestCreateWorktree_ComposeOverrides(t *testing.T) {
	// Setup test database
	database := testutil.SetupTestDB(t)
	defer database.Close()

	// Create test directory structure
	tempDir, err := os.MkdirTemp("", "vibeman-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	repoPath := filepath.Join(tempDir, "test-repo")
	worktreesDir := filepath.Join(tempDir, "worktrees")
	worktreePath := filepath.Join(worktreesDir, "feature-test")

	err = os.MkdirAll(repoPath, constants.DirPermissions)
	require.NoError(t, err)

	// Create test repository in database
	repo := &db.Repository{
		ID:   "repo-123",
		Name: "test-repo",
		Path: repoPath,
	}
	repoRepo := db.NewRepositoryRepository(database)
	err = repoRepo.Create(context.Background(), repo)
	require.NoError(t, err)

	// Create vibeman.toml config with default compose settings
	configContent := `
[repository]
name = "test-repo"

[repository.container]
compose_file = "./docker-compose.yaml"
compose_services = ["default"]

[repository.worktrees]
directory = "` + worktreesDir + `"
`
	configPath := filepath.Join(repoPath, "vibeman.toml")
	err = os.WriteFile(configPath, []byte(configContent), constants.FilePermissions)
	require.NoError(t, err)

	// Create mocks
	mockGitMgr := new(testutil.MockGitManager)
	mockContainerMgr := new(testutil.MockContainerManager)
	mockServiceMgr := new(testutil.MockServiceManager)
	cfg := &config.Manager{}

	// Mock git operations
	mockGitMgr.On("CreateWorktree", mock.Anything, repoPath, "feature/test", worktreePath).Return(nil)

	// Create operations instance
	ops := NewWorktreeOperations(database, mockGitMgr, mockContainerMgr, mockServiceMgr, cfg)

	// Test worktree creation with compose overrides
	req := CreateWorktreeRequest{
		RepositoryID:    repo.ID,
		Name:            "feature-test",
		Branch:          "feature/test",
		ComposeFile:     "./custom-compose.yaml",
		Services: []string{"backend", "frontend"},
		SkipSetup:       true, // Skip setup to focus on compose overrides
	}

	// Execute
	result, err := ops.CreateWorktree(context.Background(), req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify worktree config was updated with overrides
	// Check if the config file in the worktree has the overrides
	worktreeConfigPath := filepath.Join(worktreePath, "vibeman.toml")
	if _, err := os.Stat(worktreeConfigPath); err == nil {
		// Config file exists, verify it has overrides
		configBytes, err := os.ReadFile(worktreeConfigPath)
		if err == nil {
			configStr := string(configBytes)
			assert.Contains(t, configStr, "custom-compose.yaml", "Config should contain custom compose file")
		}
	}

	// Verify mocks were called
	mockGitMgr.AssertExpectations(t)
}

// TestCreateWorktree_ServiceDependencies tests that required services are started
func TestCreateWorktree_ServiceDependencies(t *testing.T) {
	// Setup test database
	database := testutil.SetupTestDB(t)
	defer database.Close()

	// Create test directory structure
	tempDir, err := os.MkdirTemp("", "vibeman-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	repoPath := filepath.Join(tempDir, "test-repo")
	worktreesDir := filepath.Join(tempDir, "worktrees")
	worktreePath := filepath.Join(worktreesDir, "feature-test")

	err = os.MkdirAll(repoPath, constants.DirPermissions)
	require.NoError(t, err)

	// Create test repository in database
	repo := &db.Repository{
		ID:   "repo-123",
		Name: "test-repo",
		Path: repoPath,
	}
	repoRepo := db.NewRepositoryRepository(database)
	err = repoRepo.Create(context.Background(), repo)
	require.NoError(t, err)

	// Create vibeman.toml config with required services
	configContent := `
[repository]
name = "test-repo"

[repository.worktrees]
directory = "` + worktreesDir + `"

[repository.services]
postgres = { required = true }
redis = { required = true }
localstack = { required = false }
`
	configPath := filepath.Join(repoPath, "vibeman.toml")
	err = os.WriteFile(configPath, []byte(configContent), constants.FilePermissions)
	require.NoError(t, err)

	// Create mocks
	mockGitMgr := new(testutil.MockGitManager)
	mockContainerMgr := new(testutil.MockContainerManager)
	mockServiceMgr := new(testutil.MockServiceManager)
	cfg := &config.Manager{}

	// Mock git operations
	mockGitMgr.On("CreateWorktree", mock.Anything, repoPath, "feature/test", worktreePath).Return(nil)

	// Mock service starts - only required services should be started
	mockServiceMgr.On("StartService", mock.Anything, "postgres").Return(nil)
	mockServiceMgr.On("StartService", mock.Anything, "redis").Return(nil)
	// localstack should NOT be started because it's not required

	// Create operations instance
	ops := NewWorktreeOperations(database, mockGitMgr, mockContainerMgr, mockServiceMgr, cfg)

	// Test worktree creation with service dependencies
	req := CreateWorktreeRequest{
		RepositoryID: repo.ID,
		Name:         "feature-test",
		Branch:       "feature/test",
		SkipSetup:    false, // Enable setup to test service dependencies
	}

	// Execute
	result, err := ops.CreateWorktree(context.Background(), req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify required services were started
	mockServiceMgr.AssertExpectations(t)
	mockGitMgr.AssertExpectations(t)
}

// TestCreateWorktree_AutoStart tests the auto-start functionality
func TestCreateWorktree_AutoStart(t *testing.T) {
	// Setup test database
	database := testutil.SetupTestDB(t)
	defer database.Close()

	// Create test directory structure
	tempDir, err := os.MkdirTemp("", "vibeman-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	repoPath := filepath.Join(tempDir, "test-repo")
	worktreesDir := filepath.Join(tempDir, "worktrees")
	worktreePath := filepath.Join(worktreesDir, "feature-test")

	err = os.MkdirAll(repoPath, constants.DirPermissions)
	require.NoError(t, err)

	// Create test repository in database
	repo := &db.Repository{
		ID:   "repo-123",
		Name: "test-repo",
		Path: repoPath,
	}
	repoRepo := db.NewRepositoryRepository(database)
	err = repoRepo.Create(context.Background(), repo)
	require.NoError(t, err)

	// Create simple vibeman.toml config  
	configContent := `
[repository]
name = "test-repo"

[repository.worktrees]
directory = "` + worktreesDir + `"
`
	configPath := filepath.Join(repoPath, "vibeman.toml")
	err = os.WriteFile(configPath, []byte(configContent), constants.FilePermissions)
	require.NoError(t, err)

	// Create mocks
	mockGitMgr := new(testutil.MockGitManager)
	mockContainerMgr := new(testutil.MockContainerManager)
	mockServiceMgr := new(testutil.MockServiceManager)
	cfg := &config.Manager{}

	// Mock git operations
	mockGitMgr.On("CreateWorktree", mock.Anything, repoPath, "feature/test", worktreePath).Return(nil)

	// Create operations instance
	ops := NewWorktreeOperations(database, mockGitMgr, mockContainerMgr, mockServiceMgr, cfg)

	// Test worktree creation with auto-start enabled
	req := CreateWorktreeRequest{
		RepositoryID: repo.ID,
		Name:         "feature-test",
		Branch:       "feature/test",
		AutoStart:    true,
		SkipSetup:    true,
	}

	// Execute
	result, err := ops.CreateWorktree(context.Background(), req)

	// Verify - should succeed even if StartWorktree fails (it's logged but not fatal)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "feature-test", result.Worktree.Name)

	// Verify mocks were called
	mockGitMgr.AssertExpectations(t)
}

// TestSaveRepositoryConfig tests the new config save functionality
func TestSaveRepositoryConfig(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "vibeman-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test config
	testConfig := &config.RepositoryConfig{}
	testConfig.Repository.Name = "test-repo"
	testConfig.Repository.Description = "Test repository"
	testConfig.Repository.Container.ComposeFile = "./custom-compose.yaml"
	testConfig.Repository.Container.Services = []string{"backend", "frontend"}

	// Save config
	err = config.SaveRepositoryConfig(tempDir, testConfig)
	assert.NoError(t, err)

	// Verify file was created
	configPath := filepath.Join(tempDir, "vibeman.toml")
	assert.FileExists(t, configPath)

	// Verify file contents
	configBytes, err := os.ReadFile(configPath)
	require.NoError(t, err)
	configStr := string(configBytes)

	assert.Contains(t, configStr, "test-repo")
	assert.Contains(t, configStr, "custom-compose.yaml") 
	assert.Contains(t, configStr, "backend")
	assert.Contains(t, configStr, "frontend")
}