// +build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vibeman/internal/config"
	"vibeman/internal/container"
	"vibeman/internal/db"
	"vibeman/internal/git"
	"vibeman/internal/operations"
	"vibeman/internal/service"
	"vibeman/internal/xdg"

	"github.com/stretchr/testify/suite"
)

type AIContainerIntegrationTestSuite struct {
	suite.Suite
	testDir      string
	db           *db.DB
	configMgr    *config.Manager
	gitMgr       *git.Manager
	containerMgr *container.Manager
	serviceMgr   *service.Manager
	repoOps      *operations.RepositoryOperations
	worktreeOps  *operations.WorktreeOperations
}

func (s *AIContainerIntegrationTestSuite) SetupSuite() {
	// Create test directory
	testDir, err := os.MkdirTemp("", "vibeman-ai-integration-*")
	s.Require().NoError(err)
	s.testDir = testDir

	// Initialize database
	dbConfig := &db.Config{
		Driver: "sqlite3",
		DSN:    filepath.Join(testDir, "test.db"),
	}
	s.db, err = db.New(dbConfig)
	s.Require().NoError(err)
	s.Require().NoError(s.db.Migrate())

	// Initialize managers
	s.configMgr = config.New()
	s.gitMgr = git.New(s.configMgr)
	s.containerMgr = container.New(s.configMgr)
	s.serviceMgr = service.New(s.configMgr)

	// Initialize operations with adapter
	serviceAdapter := &aiContainerServiceAdapter{mgr: s.serviceMgr}
	s.repoOps = operations.NewRepositoryOperations(s.configMgr, s.gitMgr, s.db)
	s.worktreeOps = operations.NewWorktreeOperations(s.db, s.gitMgr, s.containerMgr, serviceAdapter, s.configMgr)
}

func (s *AIContainerIntegrationTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
	
	// Clean up test directory and any worktree directories
	if s.testDir != "" {
		// Remove the main test directory
		os.RemoveAll(s.testDir)
		
		// Also clean up any worktree directories that might be siblings
		parentDir := filepath.Dir(s.testDir)
		if entries, err := os.ReadDir(parentDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && (
					strings.HasSuffix(entry.Name(), "-worktrees") ||
					strings.Contains(entry.Name(), "test-repo-worktrees")) {
					worktreePath := filepath.Join(parentDir, entry.Name())
					os.RemoveAll(worktreePath)
				}
			}
		}
	}
}

func (s *AIContainerIntegrationTestSuite) TestAIContainerLifecycle() {
	if !s.isDockerAvailable() {
		s.T().Skip("Docker not available, skipping AI container integration test")
	}

	ctx := context.Background()

	// Create test repository with AI enabled (default)
	repoPath := filepath.Join(s.testDir, "test-repo")
	s.createTestRepository(repoPath)

	// Add repository to Vibeman
	repo, err := s.repoOps.AddRepository(ctx, operations.AddRepositoryRequest{
		Path: repoPath,
		Name: "test-repo",
	})
	s.Require().NoError(err)

	// Create worktree with AI container
	worktreeResp, err := s.worktreeOps.CreateWorktree(ctx, operations.CreateWorktreeRequest{
		RepositoryID: repo.ID,
		Name:         "ai-integration-test",
		Branch:       "feature/ai-test",
		BaseBranch:   "main",
		SkipSetup:    true,
		AutoStart:    true, // This should trigger AI container creation
	})
	s.Require().NoError(err)
	worktree := worktreeResp.Worktree

	// Wait for containers to start
	time.Sleep(5 * time.Second)

	// List containers and verify AI container exists
	containers, err := s.containerMgr.List(ctx)
	s.Require().NoError(err)

	var aiContainer *container.Container
	for _, c := range containers {
		if strings.Contains(c.Name, "ai-integration-test-ai") {
			aiContainer = c
			break
		}
	}

	s.NotNil(aiContainer, "AI container should be created")
	if aiContainer != nil {
		s.Equal("ai", aiContainer.Type, "Container type should be 'ai'")
		// Check if container is running (Docker may return different status formats)
		status := strings.ToLower(aiContainer.Status)
		s.True(strings.Contains(status, "running") || strings.Contains(status, "up"), "AI container should be running, got status: %s", aiContainer.Status)
		
		// Note: Volume verification would require Docker inspect API
		// For now, we just verify the container was created with the right type
	}

	// Stop worktree (should stop AI container)
	err = s.worktreeOps.StopWorktree(ctx, worktree.ID)
	s.Require().NoError(err)

	// Wait for containers to stop
	time.Sleep(2 * time.Second)

	// Verify AI container is stopped
	if aiContainer != nil {
		// List containers again to check status
		containers, err = s.containerMgr.List(ctx)
		if err == nil {
			for _, c := range containers {
				if c.ID == aiContainer.ID {
					// Check if container is stopped (not running/up)
					status := strings.ToLower(c.Status)
					s.False(strings.Contains(status, "running") || strings.Contains(status, "up"), "AI container should be stopped, got status: %s", c.Status)
					break
				}
			}
		}
	}

	// Remove worktree (should remove AI container)
	err = s.worktreeOps.RemoveWorktree(ctx, worktree.ID, true)
	s.Require().NoError(err)

	// Wait for cleanup
	time.Sleep(2 * time.Second)

	// Verify AI container is removed
	containers, err = s.containerMgr.List(ctx)
	s.Require().NoError(err)

	found := false
	for _, c := range containers {
		if strings.Contains(c.Name, "ai-integration-test-ai") {
			found = true
			break
		}
	}
	s.False(found, "AI container should be removed")

	// Clean up repository
	err = s.repoOps.RemoveRepository(ctx, repo.ID)
	s.Require().NoError(err)
}

func (s *AIContainerIntegrationTestSuite) TestAIContainerConfiguration() {
	if !s.isDockerAvailable() {
		s.T().Skip("Docker not available, skipping AI container configuration test")
	}

	ctx := context.Background()

	// Test 1: AI container disabled
	repoPath1 := filepath.Join(s.testDir, "test-repo-disabled")
	s.createTestRepository(repoPath1)

	// Create vibeman.toml with AI disabled
	configContent := `[repository]
name = "test-repo-disabled"

[repository.ai]
enabled = false
`
	err := os.WriteFile(filepath.Join(repoPath1, "vibeman.toml"), []byte(configContent), 0644)
	s.Require().NoError(err)

	// Add repository
	repo1, err := s.repoOps.AddRepository(ctx, operations.AddRepositoryRequest{
		Path: repoPath1,
		Name: "test-repo-disabled",
	})
	s.Require().NoError(err)

	// Create worktree with AutoStart
	worktreeResp1, err := s.worktreeOps.CreateWorktree(ctx, operations.CreateWorktreeRequest{
		RepositoryID: repo1.ID,
		Name:         "no-ai-test",
		Branch:       "feature/no-ai",
		BaseBranch:   "main",
		SkipSetup:    true,
		AutoStart:    true,
	})
	s.Require().NoError(err)
	worktree1 := worktreeResp1.Worktree

	// Wait and verify no AI container
	time.Sleep(3 * time.Second)
	containers, err := s.containerMgr.List(ctx)
	s.Require().NoError(err)

	aiFound := false
	for _, c := range containers {
		if strings.Contains(c.Name, "no-ai-test-ai") {
			aiFound = true
			break
		}
	}
	s.False(aiFound, "AI container should not be created when disabled")

	// Clean up
	s.worktreeOps.RemoveWorktree(ctx, worktree1.ID, true)
	s.repoOps.RemoveRepository(ctx, repo1.ID)

	// Test 2: Custom AI image
	repoPath2 := filepath.Join(s.testDir, "test-repo-custom")
	s.createTestRepository(repoPath2)

	// Create vibeman.toml with custom image
	configContent2 := `[repository]
name = "test-repo-custom"

[repository.ai]
enabled = true
image = "alpine:latest"

[repository.ai.env]
CUSTOM_VAR = "test_value"

[repository.ai.volumes]
"/tmp" = "/host-tmp"
`
	err = os.WriteFile(filepath.Join(repoPath2, "vibeman.toml"), []byte(configContent2), 0644)
	s.Require().NoError(err)

	// Add repository
	repo2, err := s.repoOps.AddRepository(ctx, operations.AddRepositoryRequest{
		Path: repoPath2,
		Name: "test-repo-custom",
	})
	s.Require().NoError(err)

	// Create worktree
	worktreeResp2, err := s.worktreeOps.CreateWorktree(ctx, operations.CreateWorktreeRequest{
		RepositoryID: repo2.ID,
		Name:         "custom-ai-test",
		Branch:       "feature/custom-ai",
		BaseBranch:   "main",
		SkipSetup:    true,
		AutoStart:    true,
	})
	s.Require().NoError(err)
	worktree2 := worktreeResp2.Worktree

	// Wait and verify custom AI container
	time.Sleep(3 * time.Second)
	containers, err = s.containerMgr.List(ctx)
	s.Require().NoError(err)

	var customAI *container.Container
	for _, c := range containers {
		if strings.Contains(c.Name, "custom-ai-test-ai") {
			customAI = c
			break
		}
	}

	s.NotNil(customAI, "Custom AI container should be created")
	if customAI != nil {
		s.Equal("alpine:latest", customAI.Image, "Should use custom image")
		// Note: Environment variables and volumes would need to be verified through docker inspect
		// which is beyond the scope of this basic integration test
	}

	// Clean up
	s.worktreeOps.StopWorktree(ctx, worktree2.ID)
	s.worktreeOps.RemoveWorktree(ctx, worktree2.ID, true)
	s.repoOps.RemoveRepository(ctx, repo2.ID)
}

func (s *AIContainerIntegrationTestSuite) TestLogAggregation() {
	if !s.isDockerAvailable() {
		s.T().Skip("Docker not available, skipping log aggregation test")
	}

	ctx := context.Background()

	// Create test repository
	repoPath := filepath.Join(s.testDir, "test-repo-logs")
	s.createTestRepository(repoPath)

	// Add repository
	repo, err := s.repoOps.AddRepository(ctx, operations.AddRepositoryRequest{
		Path: repoPath,
		Name: "test-repo-logs",
	})
	s.Require().NoError(err)

	// Create worktree with multiple containers
	worktreeResp, err := s.worktreeOps.CreateWorktree(ctx, operations.CreateWorktreeRequest{
		RepositoryID: repo.ID,
		Name:         "logs-test",
		Branch:       "feature/logs",
		BaseBranch:   "main",
		SkipSetup:    true,
		AutoStart:    true,
	})
	s.Require().NoError(err)
	worktree := worktreeResp.Worktree

	// Wait for containers to start
	time.Sleep(5 * time.Second)

	// Get logs directory
	logsDir := xdg.LogsDir()
	worktreeLogsDir := filepath.Join(logsDir, repo.Name, worktree.Name)

	// Check if log files are being created
	entries, err := os.ReadDir(worktreeLogsDir)
	if err == nil {
		s.Greater(len(entries), 0, "Log files should be created")
		
		// Check for aggregated directory
		aggregatedDir := filepath.Join(worktreeLogsDir, "aggregated")
		if _, err := os.Stat(aggregatedDir); err == nil {
			// Check for README
			readmePath := filepath.Join(aggregatedDir, "README.md")
			s.FileExists(readmePath, "README should exist in aggregated directory")
		}
	}

	// Clean up
	s.worktreeOps.StopWorktree(ctx, worktree.ID)
	s.worktreeOps.RemoveWorktree(ctx, worktree.ID, true)
	s.repoOps.RemoveRepository(ctx, repo.ID)
}

func (s *AIContainerIntegrationTestSuite) TestAIContainerFailureHandling() {
	if !s.isDockerAvailable() {
		s.T().Skip("Docker not available, skipping failure handling test")
	}

	ctx := context.Background()

	// Create test repository with invalid AI image
	repoPath := filepath.Join(s.testDir, "test-repo-failure")
	s.createTestRepository(repoPath)

	// Create vibeman.toml with non-existent image
	configContent := `[repository]
name = "test-repo-failure"

[repository.ai]
enabled = true
image = "this-image-definitely-does-not-exist:latest"
`
	err := os.WriteFile(filepath.Join(repoPath, "vibeman.toml"), []byte(configContent), 0644)
	s.Require().NoError(err)

	// Add repository
	repo, err := s.repoOps.AddRepository(ctx, operations.AddRepositoryRequest{
		Path: repoPath,
		Name: "test-repo-failure",
	})
	s.Require().NoError(err)

	// Create worktree - should succeed despite AI container failure
	worktreeResp, err := s.worktreeOps.CreateWorktree(ctx, operations.CreateWorktreeRequest{
		RepositoryID: repo.ID,
		Name:         "failure-test",
		Branch:       "feature/failure",
		BaseBranch:   "main",
		SkipSetup:    true,
		AutoStart:    true,
	})
	s.Require().NoError(err, "Worktree creation should succeed even if AI container fails")
	worktree := worktreeResp.Worktree

	// Verify worktree is running
	worktreeRepo := db.NewWorktreeRepository(s.db)
	updatedWorktree, err := worktreeRepo.Get(ctx, worktree.ID)
	s.Require().NoError(err)
	s.Equal(db.StatusRunning, updatedWorktree.Status, "Worktree should be running")

	// Clean up
	s.worktreeOps.StopWorktree(ctx, worktree.ID)
	s.worktreeOps.RemoveWorktree(ctx, worktree.ID, true)
	s.repoOps.RemoveRepository(ctx, repo.ID)
}

func (s *AIContainerIntegrationTestSuite) TestEnhancedAIContainerTools() {
	if !s.isDockerAvailable() {
		s.T().Skip("Docker not available, skipping enhanced AI container test")
	}

	ctx := context.Background()

	// Create test repository
	repoPath := filepath.Join(s.testDir, "test-repo-enhanced")
	s.createTestRepository(repoPath)

	// Add repository
	repo, err := s.repoOps.AddRepository(ctx, operations.AddRepositoryRequest{
		Path: repoPath,
		Name: "test-repo-enhanced",
	})
	s.Require().NoError(err)

	// Create worktree with AI container
	worktreeResp, err := s.worktreeOps.CreateWorktree(ctx, operations.CreateWorktreeRequest{
		RepositoryID: repo.ID,
		Name:         "enhanced-ai-test",
		Branch:       "feature/enhanced",
		BaseBranch:   "main",
		SkipSetup:    true,
		AutoStart:    true,
	})
	s.Require().NoError(err)
	worktree := worktreeResp.Worktree

	// Wait for container to start
	time.Sleep(5 * time.Second)

	// Find AI container
	containers, err := s.containerMgr.List(ctx)
	s.Require().NoError(err)

	var aiContainer *container.Container
	for _, c := range containers {
		if strings.Contains(c.Name, "enhanced-ai-test-ai") {
			aiContainer = c
			break
		}
	}

	s.Require().NotNil(aiContainer, "AI container should be created")

	// Test tool availability (basic check - full tool testing would require docker exec)
	// In a real integration test, we would execute commands in the container
	// For now, we just verify the container is running with the correct image
	s.Equal("ai", aiContainer.Type)
	// Check if container is running (Docker may return different status formats)
	status := strings.ToLower(aiContainer.Status)
	s.True(strings.Contains(status, "running") || strings.Contains(status, "up"), "AI container should be running, got status: %s", aiContainer.Status)
	s.Contains(aiContainer.Image, "vibeman/ai-assistant")

	// Clean up
	s.worktreeOps.StopWorktree(ctx, worktree.ID)
	s.worktreeOps.RemoveWorktree(ctx, worktree.ID, true)
	s.repoOps.RemoveRepository(ctx, repo.ID)
}

// TestAIContainerIntegration runs the AI container integration test suite
func TestAIContainerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	suite.Run(t, new(AIContainerIntegrationTestSuite))
}

// Note: worktreeServiceAdapter is already defined in worktree_lifecycle_test.go

// Helper methods

func (s *AIContainerIntegrationTestSuite) isDockerAvailable() bool {
	// Try to list containers - if it works, Docker is available
	ctx := context.Background()
	_, err := s.containerMgr.List(ctx)
	return err == nil
}

func (s *AIContainerIntegrationTestSuite) createTestRepository(path string) {
	// Create directory
	err := os.MkdirAll(path, 0755)
	s.Require().NoError(err)

	// Initialize git repository
	ctx := context.Background()
	err = s.gitMgr.InitRepository(ctx, path)
	s.Require().NoError(err)

	// Create README
	readmePath := filepath.Join(path, "README.md")
	err = os.WriteFile(readmePath, []byte("# Test Repository\n"), 0644)
	s.Require().NoError(err)

	// Create unique worktree directory name based on the repository path
	// This ensures each test gets its own worktree directory
	repoName := filepath.Base(path)
	worktreeDir := filepath.Join(filepath.Dir(path), repoName+"-worktrees")

	// Create vibeman.toml with unique worktree directory
	vibemanConfig := fmt.Sprintf(`[repository]
name = "test-repo"
description = "Test repository for integration tests"

[repository.worktrees]
directory = "%s"

[repository.git]
default_branch = "main"
`, worktreeDir)
	configPath := filepath.Join(path, "vibeman.toml")
	err = os.WriteFile(configPath, []byte(vibemanConfig), 0644)
	s.Require().NoError(err)

	// Commit initial files
	err = s.gitMgr.AddAndCommit(ctx, path, "Initial commit")
	s.Require().NoError(err)
}

// aiContainerServiceAdapter adapts service.Manager to operations.ServiceManager interface
type aiContainerServiceAdapter struct {
	mgr *service.Manager
}

func (a *aiContainerServiceAdapter) StartService(ctx context.Context, name string) error {
	return a.mgr.StartService(ctx, name)
}

func (a *aiContainerServiceAdapter) StopService(ctx context.Context, name string) error {
	return a.mgr.StopService(ctx, name)
}

func (a *aiContainerServiceAdapter) GetService(name string) (interface{}, error) {
	return a.mgr.GetService(name)
}

func (a *aiContainerServiceAdapter) HealthCheck(ctx context.Context, name string) error {
	return a.mgr.HealthCheck(ctx, name)
}

func (a *aiContainerServiceAdapter) AddReference(serviceName, repoName string) error {
	return a.mgr.AddReference(serviceName, repoName)
}

func (a *aiContainerServiceAdapter) RemoveReference(serviceName, repoName string) error {
	return a.mgr.RemoveReference(serviceName, repoName)
}