package operations

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vibeman/internal/container"
	"vibeman/internal/db"
	"vibeman/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLogAggregator_StartLogAggregation(t *testing.T) {
	// Setup test database
	database := testutil.SetupTestDB(t)
	defer database.Close()

	// Create temp directory for logs
	tempDir, err := os.MkdirTemp("", "vibeman-logs-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Override XDG logs dir for testing
	oldXDGStateHome := os.Getenv("XDG_STATE_HOME")
	os.Setenv("XDG_STATE_HOME", tempDir)
	defer os.Setenv("XDG_STATE_HOME", oldXDGStateHome)

	// Create test repository and worktree
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
		Path:         filepath.Join(tempDir, "worktree"),
		Status:       db.StatusRunning,
	}
	worktreeRepo := db.NewWorktreeRepository(database)
	err = worktreeRepo.Create(context.Background(), worktree)
	require.NoError(t, err)

	// Create mock container manager
	mockContainerMgr := new(testutil.MockContainerManager)

	// Mock containers for the worktree
	containers := []*container.Container{
		{
			ID:          "container-1",
			Name:        "test-repo-feature-test",
			Type:        "worktree",
			Repository:  repo.Name,
			Environment: worktree.Name,
		},
		{
			ID:          "container-2",
			Name:        "test-repo-feature-test-ai",
			Type:        "ai",
			Repository:  repo.Name,
			Environment: worktree.Name,
		},
		{
			ID:          "container-3",
			Name:        "postgres",
			Type:        "service",
			Repository:  "",
			Environment: "",
		},
	}
	mockContainerMgr.On("List", mock.Anything).Return(containers, nil)

	// Mock logs for each container
	mockContainerMgr.On("Logs", mock.Anything, "container-1", false).Return([]byte("Worktree container logs\n"), nil)
	mockContainerMgr.On("Logs", mock.Anything, "container-2", false).Return([]byte("AI container logs\n"), nil)

	// Create log aggregator
	logAggregator := NewLogAggregator(database, mockContainerMgr)

	// Start log aggregation
	err = logAggregator.StartLogAggregation(context.Background(), worktree.ID)
	assert.NoError(t, err)

	// Give some time for goroutines to start
	time.Sleep(100 * time.Millisecond)

	// Check that log files were created
	logsDir := filepath.Join(tempDir, "vibeman", "logs", repo.Name, worktree.Name)
	
	// Verify log files exist - filename is prefixed with container type
	worktreeLogFile := filepath.Join(logsDir, "worktree-test-repo-feature-test.log")
	aiLogFile := filepath.Join(logsDir, "ai-test-repo-feature-test-ai.log")
	
	// Wait a bit more for files to be created
	time.Sleep(200 * time.Millisecond)
	
	assert.FileExists(t, worktreeLogFile)
	assert.FileExists(t, aiLogFile)

	// Verify log content
	worktreeLogContent, err := os.ReadFile(worktreeLogFile)
	require.NoError(t, err)
	assert.Contains(t, string(worktreeLogContent), "Worktree container logs")

	aiLogContent, err := os.ReadFile(aiLogFile)
	require.NoError(t, err)
	assert.Contains(t, string(aiLogContent), "AI container logs")

	// Stop log aggregation
	logAggregator.StopLogAggregation(worktree.ID)

	// Give goroutines time to clean up
	time.Sleep(100 * time.Millisecond)
}

func TestLogAggregator_AggregateLogsForAIContainer(t *testing.T) {
	// Setup test database
	database := testutil.SetupTestDB(t)
	defer database.Close()

	// Create temp directory for logs
	tempDir, err := os.MkdirTemp("", "vibeman-logs-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Override XDG logs dir for testing
	oldXDGStateHome := os.Getenv("XDG_STATE_HOME")
	os.Setenv("XDG_STATE_HOME", tempDir)
	defer os.Setenv("XDG_STATE_HOME", oldXDGStateHome)

	// Create test repository and worktree
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
		Path:         filepath.Join(tempDir, "worktree"),
		Status:       db.StatusRunning,
	}
	worktreeRepo := db.NewWorktreeRepository(database)
	err = worktreeRepo.Create(context.Background(), worktree)
	require.NoError(t, err)

	// Create log directory and sample log files
	logsDir := filepath.Join(tempDir, "vibeman", "logs", repo.Name, worktree.Name)
	err = os.MkdirAll(logsDir, 0755)
	require.NoError(t, err)

	// Create sample log files
	logFiles := []string{
		"worktree-test.log",
		"ai-assistant.log",
		"service-postgres.log",
	}
	for _, logFile := range logFiles {
		logPath := filepath.Join(logsDir, logFile)
		err = os.WriteFile(logPath, []byte("Sample log content"), 0644)
		require.NoError(t, err)
	}

	// Create mock container manager
	mockContainerMgr := new(testutil.MockContainerManager)

	// Create log aggregator
	logAggregator := NewLogAggregator(database, mockContainerMgr)

	// Aggregate logs for AI container
	err = logAggregator.AggregateLogsForAIContainer(context.Background(), worktree.ID)
	assert.NoError(t, err)

	// Check aggregated directory
	aggregatedDir := filepath.Join(logsDir, "aggregated")
	assert.DirExists(t, aggregatedDir)

	// Check README exists
	readmePath := filepath.Join(aggregatedDir, "README.md")
	assert.FileExists(t, readmePath)

	// Verify README content
	readmeContent, err := os.ReadFile(readmePath)
	require.NoError(t, err)
	assert.Contains(t, string(readmeContent), "Aggregated Logs for test-repo - feature-test")
	assert.Contains(t, string(readmeContent), "tail -f *.log")

	// Check symlinks exist
	for _, logFile := range logFiles {
		symlinkPath := filepath.Join(aggregatedDir, logFile)
		info, err := os.Lstat(symlinkPath)
		require.NoError(t, err)
		assert.True(t, info.Mode()&os.ModeSymlink != 0, "Expected %s to be a symlink", logFile)
		
		// Verify symlink points to correct file
		target, err := os.Readlink(symlinkPath)
		require.NoError(t, err)
		expectedTarget := filepath.Join(logsDir, logFile)
		assert.Equal(t, expectedTarget, target)
	}
}

func TestLogAggregator_StopLogAggregation(t *testing.T) {
	// Setup test database
	database := testutil.SetupTestDB(t)
	defer database.Close()

	// Create mock container manager
	mockContainerMgr := new(testutil.MockContainerManager)

	// Create log aggregator
	logAggregator := NewLogAggregator(database, mockContainerMgr)

	// Add some active streams
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	ctx3, cancel3 := context.WithCancel(context.Background())

	logAggregator.activeStreams["wt-123-container1"] = cancel1
	logAggregator.activeStreams["wt-123-container2"] = cancel2
	logAggregator.activeStreams["wt-456-container3"] = cancel3

	// Stop log aggregation for wt-123
	logAggregator.StopLogAggregation("wt-123")

	// Verify only wt-123 streams were cancelled
	assert.Len(t, logAggregator.activeStreams, 1)
	assert.Contains(t, logAggregator.activeStreams, "wt-456-container3")

	// Verify contexts were cancelled
	select {
	case <-ctx1.Done():
		// Good, context was cancelled
	default:
		t.Error("Expected ctx1 to be cancelled")
	}

	select {
	case <-ctx2.Done():
		// Good, context was cancelled
	default:
		t.Error("Expected ctx2 to be cancelled")
	}

	select {
	case <-ctx3.Done():
		t.Error("Expected ctx3 to NOT be cancelled")
	default:
		// Good, context was not cancelled
	}
}

func TestIsWorktreeContainer(t *testing.T) {
	tests := []struct {
		name          string
		container     *container.Container
		repoName      string
		worktreeName  string
		expected      bool
	}{
		{
			name: "matching worktree container",
			container: &container.Container{
				Name: "test-repo-feature-branch",
			},
			repoName:     "test-repo",
			worktreeName: "feature-branch",
			expected:     true,
		},
		{
			name: "matching AI container",
			container: &container.Container{
				Name: "test-repo-feature-branch-ai",
			},
			repoName:     "test-repo",
			worktreeName: "feature-branch",
			expected:     true,
		},
		{
			name: "different repo",
			container: &container.Container{
				Name: "other-repo-feature-branch",
			},
			repoName:     "test-repo",
			worktreeName: "feature-branch",
			expected:     false,
		},
		{
			name: "different worktree",
			container: &container.Container{
				Name: "test-repo-main",
			},
			repoName:     "test-repo",
			worktreeName: "feature-branch",
			expected:     false,
		},
		{
			name: "service container",
			container: &container.Container{
				Name: "postgres",
			},
			repoName:     "test-repo",
			worktreeName: "feature-branch",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWorktreeContainer(tt.container, tt.repoName, tt.worktreeName)
			assert.Equal(t, tt.expected, result)
		})
	}
}