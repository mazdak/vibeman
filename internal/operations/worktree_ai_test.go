package operations

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"vibeman/internal/config"
	"vibeman/internal/constants"
	"vibeman/internal/container"
	"vibeman/internal/db"
	"vibeman/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestStartWorktree_WithAIContainer tests that AI container is created when worktree starts
func TestStartWorktree_WithAIContainer(t *testing.T) {
	// Setup test database
	database := testutil.SetupTestDB(t)
	defer database.Close()

	// Create test directory structure
	tempDir, err := os.MkdirTemp("", "vibeman-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	worktreePath := filepath.Join(tempDir, "worktrees", "feature-test")
	err = os.MkdirAll(worktreePath, constants.DirPermissions)
	require.NoError(t, err)

	// Create test repository and worktree in database
	repo := &db.Repository{
		ID:   "repo-123",
		Name: "test-repo",
		Path: filepath.Join(tempDir, "test-repo"),
	}
	repoRepo := db.NewRepositoryRepository(database)
	err = repoRepo.Create(context.Background(), repo)
	require.NoError(t, err)

	worktree := &db.Worktree{
		ID:           "wt-123",
		RepositoryID: repo.ID,
		Name:         "feature-test",
		Branch:       "feature/test",
		Path:         worktreePath,
		Status:       db.StatusStopped,
	}
	worktreeRepo := db.NewWorktreeRepository(database)
	err = worktreeRepo.Create(context.Background(), worktree)
	require.NoError(t, err)

	// Create vibeman.toml config with AI enabled (default)
	configContent := `
[repository]
name = "test-repo"

[repository.ai]
# enabled = true is the default
`
	configPath := filepath.Join(worktreePath, "vibeman.toml")
	err = os.WriteFile(configPath, []byte(configContent), constants.FilePermissions)
	require.NoError(t, err)

	// Create mocks
	mockGitMgr := new(testutil.MockGitManager)
	mockContainerMgr := new(testutil.MockContainerManager)
	mockServiceMgr := new(testutil.MockServiceManager)
	cfg := &config.Manager{}

	// Mock AI container creation
	mockAIContainer := &container.Container{
		ID:   "ai-container-123",
		Name: "test-repo-feature-test-ai",
		Type: "ai",
	}

	// Expect AI container creation with proper config
	mockContainerMgr.On("CreateWithConfig", mock.Anything, mock.Anything).Return(mockAIContainer, nil)

	// Expect AI container start (use StartContainer which is what Start calls)
	mockContainerMgr.On("StartContainer", mock.Anything, "ai-container-123").Return(nil)

	// Expect log aggregation to list containers
	mockContainerMgr.On("List", mock.Anything).Return([]*container.Container{mockAIContainer}, nil)

	// Expect log aggregation to get logs
	mockContainerMgr.On("Logs", mock.Anything, "ai-container-123", false).Return([]byte("AI container logs"), nil)

	// Create operations instance
	ops := NewWorktreeOperations(database, mockGitMgr, mockContainerMgr, mockServiceMgr, cfg)

	// Execute
	err = ops.StartWorktree(context.Background(), worktree.ID)

	// Verify
	assert.NoError(t, err)

	// Verify mocks were called
	mockContainerMgr.AssertExpectations(t)

	// Verify worktree status was updated
	updatedWorktree, err := worktreeRepo.Get(context.Background(), worktree.ID)
	require.NoError(t, err)
	assert.Equal(t, db.StatusRunning, updatedWorktree.Status)
}

// TestStartWorktree_AIContainerDisabled tests that AI container is not created when disabled
func TestStartWorktree_AIContainerDisabled(t *testing.T) {
	// Setup test database
	database := testutil.SetupTestDB(t)
	defer database.Close()

	// Create test directory structure
	tempDir, err := os.MkdirTemp("", "vibeman-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	worktreePath := filepath.Join(tempDir, "worktrees", "feature-test")
	err = os.MkdirAll(worktreePath, constants.DirPermissions)
	require.NoError(t, err)

	// Create test repository and worktree in database
	repo := &db.Repository{
		ID:   "repo-123",
		Name: "test-repo",
		Path: filepath.Join(tempDir, "test-repo"),
	}
	repoRepo := db.NewRepositoryRepository(database)
	err = repoRepo.Create(context.Background(), repo)
	require.NoError(t, err)

	worktree := &db.Worktree{
		ID:           "wt-123",
		RepositoryID: repo.ID,
		Name:         "feature-test",
		Branch:       "feature/test",
		Path:         worktreePath,
		Status:       db.StatusStopped,
	}
	worktreeRepo := db.NewWorktreeRepository(database)
	err = worktreeRepo.Create(context.Background(), worktree)
	require.NoError(t, err)

	// Create vibeman.toml config with AI disabled
	configContent := `
[repository]
name = "test-repo"

[repository.ai]
enabled = false
`
	configPath := filepath.Join(worktreePath, "vibeman.toml")
	err = os.WriteFile(configPath, []byte(configContent), constants.FilePermissions)
	require.NoError(t, err)

	// Create mocks
	mockGitMgr := new(testutil.MockGitManager)
	mockContainerMgr := new(testutil.MockContainerManager)
	mockServiceMgr := new(testutil.MockServiceManager)
	cfg := &config.Manager{}

	// Should NOT create AI container when disabled

	// Create operations instance
	ops := NewWorktreeOperations(database, mockGitMgr, mockContainerMgr, mockServiceMgr, cfg)

	// Execute
	err = ops.StartWorktree(context.Background(), worktree.ID)

	// Verify
	assert.NoError(t, err)

	// Verify AI container was NOT created
	mockContainerMgr.AssertNotCalled(t, "CreateWithConfig", mock.Anything, mock.Anything)
	mockContainerMgr.AssertNotCalled(t, "Start", mock.Anything, mock.Anything)

	// Verify worktree status was updated
	updatedWorktree, err := worktreeRepo.Get(context.Background(), worktree.ID)
	require.NoError(t, err)
	assert.Equal(t, db.StatusRunning, updatedWorktree.Status)
}

// TestStartWorktree_CustomAIImage tests custom AI image configuration
func TestStartWorktree_CustomAIImage(t *testing.T) {
	// Setup test database
	database := testutil.SetupTestDB(t)
	defer database.Close()

	// Create test directory structure
	tempDir, err := os.MkdirTemp("", "vibeman-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	worktreePath := filepath.Join(tempDir, "worktrees", "feature-test")
	err = os.MkdirAll(worktreePath, constants.DirPermissions)
	require.NoError(t, err)

	// Create test repository and worktree in database
	repo := &db.Repository{
		ID:   "repo-123",
		Name: "test-repo",
		Path: filepath.Join(tempDir, "test-repo"),
	}
	repoRepo := db.NewRepositoryRepository(database)
	err = repoRepo.Create(context.Background(), repo)
	require.NoError(t, err)

	worktree := &db.Worktree{
		ID:           "wt-123",
		RepositoryID: repo.ID,
		Name:         "feature-test",
		Branch:       "feature/test",
		Path:         worktreePath,
		Status:       db.StatusStopped,
	}
	worktreeRepo := db.NewWorktreeRepository(database)
	err = worktreeRepo.Create(context.Background(), worktree)
	require.NoError(t, err)

	// Create vibeman.toml config with custom AI image
	configContent := `
[repository]
name = "test-repo"

[repository.ai]
enabled = true
image = "mycompany/custom-ai:v2"

[repository.ai.env]
CUSTOM_VAR = "custom_value"
API_KEY = "test-key"

[repository.ai.volumes]
"/host/custom" = "/container/custom"
`
	configPath := filepath.Join(worktreePath, "vibeman.toml")
	err = os.WriteFile(configPath, []byte(configContent), constants.FilePermissions)
	require.NoError(t, err)

	// Create mocks
	mockGitMgr := new(testutil.MockGitManager)
	mockContainerMgr := new(testutil.MockContainerManager)
	mockServiceMgr := new(testutil.MockServiceManager)
	cfg := &config.Manager{}

	// Mock AI container creation
	mockAIContainer := &container.Container{
		ID:   "ai-container-123",
		Name: "test-repo-feature-test-ai",
		Type: "ai",
	}

	// Expect AI container creation with custom config
	mockContainerMgr.On("CreateWithConfig", mock.Anything, mock.MatchedBy(func(config *container.CreateConfig) bool {
		// Check custom image
		if config.Image != "mycompany/custom-ai:v2" {
			return false
		}

		// Check custom env vars are included
		hasCustomVar := false
		hasAPIKey := false
		for _, env := range config.EnvVars {
			if env == "CUSTOM_VAR=custom_value" {
				hasCustomVar = true
			}
			if env == "API_KEY=test-key" {
				hasAPIKey = true
			}
		}

		// Check custom volume is included
		hasCustomVolume := false
		for _, vol := range config.Volumes {
			if vol == "/host/custom:/container/custom" {
				hasCustomVolume = true
			}
		}

		return hasCustomVar && hasAPIKey && hasCustomVolume
	})).Return(mockAIContainer, nil)

	// Expect AI container start (use StartContainer which is what Start calls)
	mockContainerMgr.On("StartContainer", mock.Anything, "ai-container-123").Return(nil)

	// Expect log aggregation to list containers
	mockContainerMgr.On("List", mock.Anything).Return([]*container.Container{mockAIContainer}, nil)

	// Expect log aggregation to get logs
	mockContainerMgr.On("Logs", mock.Anything, "ai-container-123", false).Return([]byte("AI container logs"), nil)

	// Create operations instance
	ops := NewWorktreeOperations(database, mockGitMgr, mockContainerMgr, mockServiceMgr, cfg)

	// Execute
	err = ops.StartWorktree(context.Background(), worktree.ID)

	// Verify
	assert.NoError(t, err)

	// Verify mocks were called
	mockContainerMgr.AssertExpectations(t)
}

// TestStopWorktree_WithAIContainer tests that AI container is stopped when worktree stops
func TestStopWorktree_WithAIContainer(t *testing.T) {
	// Setup test database
	database := testutil.SetupTestDB(t)
	defer database.Close()

	// Create test repository and worktree in database
	repo := &db.Repository{
		ID:   "repo-123",
		Name: "test-repo",
		Path: "/test/repo",
	}
	repoRepo := db.NewRepositoryRepository(database)
	err := repoRepo.Create(context.Background(), repo)
	require.NoError(t, err)

	worktree := &db.Worktree{
		ID:           "wt-123",
		RepositoryID: repo.ID,
		Name:         "feature-test",
		Branch:       "feature/test",
		Path:         "/test/worktree",
		Status:       db.StatusRunning,
	}
	worktreeRepo := db.NewWorktreeRepository(database)
	err = worktreeRepo.Create(context.Background(), worktree)
	require.NoError(t, err)

	// Create mocks
	mockGitMgr := new(testutil.MockGitManager)
	mockContainerMgr := new(testutil.MockContainerManager)
	mockServiceMgr := new(testutil.MockServiceManager)
	cfg := &config.Manager{}

	// Mock AI container lookup
	mockAIContainer := &container.Container{
		ID:   "ai-container-123",
		Name: "test-repo-feature-test-ai",
		Type: "ai",
	}
	mockContainerMgr.On("GetByName", mock.Anything, "test-repo-feature-test-ai").Return(mockAIContainer, nil)

	// Expect AI container stop (use StopContainer which is what Stop calls)
	mockContainerMgr.On("StopContainer", mock.Anything, "ai-container-123").Return(nil)

	// Create operations instance
	ops := NewWorktreeOperations(database, mockGitMgr, mockContainerMgr, mockServiceMgr, cfg)

	// Execute
	err = ops.StopWorktree(context.Background(), worktree.ID)

	// Verify
	assert.NoError(t, err)

	// Verify mocks were called
	mockContainerMgr.AssertExpectations(t)

	// Verify worktree status was updated
	updatedWorktree, err := worktreeRepo.Get(context.Background(), worktree.ID)
	require.NoError(t, err)
	assert.Equal(t, db.StatusStopped, updatedWorktree.Status)
}

// TestRemoveWorktree_WithAIContainer tests that AI container is removed when worktree is removed
func TestRemoveWorktree_WithAIContainer(t *testing.T) {
	// Setup test database
	database := testutil.SetupTestDB(t)
	defer database.Close()

	// Create test directory structure
	tempDir, err := os.MkdirTemp("", "vibeman-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	worktreePath := filepath.Join(tempDir, "worktrees", "feature-test")
	err = os.MkdirAll(worktreePath, constants.DirPermissions)
	require.NoError(t, err)

	// Create test repository and worktree in database
	repo := &db.Repository{
		ID:   "repo-123",
		Name: "test-repo",
		Path: filepath.Join(tempDir, "test-repo"),
	}
	repoRepo := db.NewRepositoryRepository(database)
	err = repoRepo.Create(context.Background(), repo)
	require.NoError(t, err)

	worktree := &db.Worktree{
		ID:           "wt-123",
		RepositoryID: repo.ID,
		Name:         "feature-test",
		Branch:       "feature/test",
		Path:         worktreePath,
		Status:       db.StatusStopped,
	}
	worktreeRepo := db.NewWorktreeRepository(database)
	err = worktreeRepo.Create(context.Background(), worktree)
	require.NoError(t, err)

	// Create mocks
	mockGitMgr := new(testutil.MockGitManager)
	mockContainerMgr := new(testutil.MockContainerManager)
	mockServiceMgr := new(testutil.MockServiceManager)
	cfg := &config.Manager{}

	// Mock AI container lookup
	mockAIContainer := &container.Container{
		ID:   "ai-container-123",
		Name: "test-repo-feature-test-ai",
		Type: "ai",
	}
	mockContainerMgr.On("GetByName", mock.Anything, "test-repo-feature-test-ai").Return(mockAIContainer, nil)

	// Expect AI container stop and remove (use underlying methods)
	mockContainerMgr.On("StopContainer", mock.Anything, "ai-container-123").Return(nil)
	mockContainerMgr.On("RemoveContainer", mock.Anything, "ai-container-123").Return(nil)

	// Mock git operations
	mockGitMgr.On("HasUncommittedChanges", mock.Anything, worktreePath).Return(false, nil)
	mockGitMgr.On("HasUnpushedCommits", mock.Anything, worktreePath).Return(false, nil)
	mockGitMgr.On("RemoveWorktree", mock.Anything, worktreePath).Return(nil)

	// Create operations instance
	ops := NewWorktreeOperations(database, mockGitMgr, mockContainerMgr, mockServiceMgr, cfg)

	// Execute
	err = ops.RemoveWorktree(context.Background(), worktree.ID, false)

	// Verify
	assert.NoError(t, err)

	// Verify mocks were called
	mockContainerMgr.AssertExpectations(t)
	mockGitMgr.AssertExpectations(t)

	// Verify worktree was removed from database
	_, err = worktreeRepo.Get(context.Background(), worktree.ID)
	assert.Error(t, err)
}

// TestStartWorktree_AIContainerCreationFailure tests graceful handling of AI container creation failure
func TestStartWorktree_AIContainerCreationFailure(t *testing.T) {
	// Setup test database
	database := testutil.SetupTestDB(t)
	defer database.Close()

	// Create test directory structure
	tempDir, err := os.MkdirTemp("", "vibeman-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	worktreePath := filepath.Join(tempDir, "worktrees", "feature-test")
	err = os.MkdirAll(worktreePath, constants.DirPermissions)
	require.NoError(t, err)

	// Create test repository and worktree in database
	repo := &db.Repository{
		ID:   "repo-123",
		Name: "test-repo",
		Path: filepath.Join(tempDir, "test-repo"),
	}
	repoRepo := db.NewRepositoryRepository(database)
	err = repoRepo.Create(context.Background(), repo)
	require.NoError(t, err)

	worktree := &db.Worktree{
		ID:           "wt-123",
		RepositoryID: repo.ID,
		Name:         "feature-test",
		Branch:       "feature/test",
		Path:         worktreePath,
		Status:       db.StatusStopped,
	}
	worktreeRepo := db.NewWorktreeRepository(database)
	err = worktreeRepo.Create(context.Background(), worktree)
	require.NoError(t, err)

	// Create vibeman.toml config with AI enabled
	configContent := `
[repository]
name = "test-repo"

[repository.ai]
enabled = true
`
	configPath := filepath.Join(worktreePath, "vibeman.toml")
	err = os.WriteFile(configPath, []byte(configContent), constants.FilePermissions)
	require.NoError(t, err)

	// Create mocks
	mockGitMgr := new(testutil.MockGitManager)
	mockContainerMgr := new(testutil.MockContainerManager)
	mockServiceMgr := new(testutil.MockServiceManager)
	cfg := &config.Manager{}

	// Mock AI container creation failure
	mockContainerMgr.On("CreateWithConfig", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	// Create operations instance
	ops := NewWorktreeOperations(database, mockGitMgr, mockContainerMgr, mockServiceMgr, cfg)

	// Execute - should not fail even if AI container creation fails
	err = ops.StartWorktree(context.Background(), worktree.ID)

	// Verify - operation should succeed
	assert.NoError(t, err)

	// Verify mocks were called
	mockContainerMgr.AssertExpectations(t)

	// Verify worktree status was still updated to running
	updatedWorktree, err := worktreeRepo.Get(context.Background(), worktree.ID)
	require.NoError(t, err)
	assert.Equal(t, db.StatusRunning, updatedWorktree.Status)
}