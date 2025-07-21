// +build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vibeman/internal/config"
	"vibeman/internal/container"
	"vibeman/internal/db"
	"vibeman/internal/git"
	"vibeman/internal/operations"
	"vibeman/internal/service"

	"github.com/stretchr/testify/suite"
)

// WorktreeLifecycleTestSuite tests the complete lifecycle of worktree operations
type WorktreeLifecycleTestSuite struct {
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

func (s *WorktreeLifecycleTestSuite) SetupSuite() {
	// Create test directory
	testDir, err := os.MkdirTemp("", "vibeman-integration-*")
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
	serviceAdapter := &worktreeServiceAdapter{mgr: s.serviceMgr}
	s.repoOps = operations.NewRepositoryOperations(s.configMgr, s.gitMgr, s.db)
	s.worktreeOps = operations.NewWorktreeOperations(s.db, s.gitMgr, s.containerMgr, serviceAdapter, s.configMgr)
}

func (s *WorktreeLifecycleTestSuite) TearDownSuite() {
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

func (s *WorktreeLifecycleTestSuite) SetupTest() {
	// Clean up any existing data from previous tests
	ctx := context.Background()
	
	// Remove all worktrees from database
	worktreeRepo := db.NewWorktreeRepository(s.db)
	worktrees, _ := worktreeRepo.List(ctx, "", "")
	for _, wt := range worktrees {
		worktreeRepo.Delete(ctx, wt.ID)
	}
	
	// Remove all repositories from database
	repoRepo := db.NewRepositoryRepository(s.db)
	repos, _ := repoRepo.List(ctx)
	for _, repo := range repos {
		repoRepo.Delete(ctx, repo.ID)
	}
	
	// Clean up any leftover worktree directories in temp space
	parentDir := filepath.Dir(s.testDir)
	if entries, err := os.ReadDir(parentDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && (
				strings.Contains(entry.Name(), "worktrees") ||
				strings.HasPrefix(entry.Name(), "test-repo")) {
				leftoverPath := filepath.Join(parentDir, entry.Name())
				// Only remove if it's not our current test directory
				if leftoverPath != s.testDir {
					os.RemoveAll(leftoverPath)
				}
			}
		}
	}
}

func (s *WorktreeLifecycleTestSuite) TestCompleteWorktreeLifecycle() {
	ctx := context.Background()
	
	// Step 1: Create a test repository
	testRepoPath := filepath.Join(s.testDir, "test-repo")
	s.createTestRepository(testRepoPath)
	
	// Step 2: Add repository to vibeman
	repo, err := s.repoOps.AddRepository(ctx, operations.AddRepositoryRequest{
		Path: testRepoPath,
		Name: "test-repo",
	})
	s.Require().NoError(err)
	s.NotNil(repo)
	s.Equal("test-repo", repo.Name)
	
	// Step 3: Create a worktree
	worktreeResp, err := s.worktreeOps.CreateWorktree(ctx, operations.CreateWorktreeRequest{
		RepositoryID: repo.ID,
		Name:        "feature-test",
		Branch:      "feature/test-branch",
		BaseBranch:  "main",
		SkipSetup:   true,
		AutoStart:   false,
	})
	s.Require().NoError(err)
	s.NotNil(worktreeResp)
	s.NotNil(worktreeResp.Worktree)
	
	// Verify worktree was created
	s.Equal("feature-test", worktreeResp.Worktree.Name)
	s.Equal("feature/test-branch", worktreeResp.Worktree.Branch)
	s.Equal(db.StatusStopped, worktreeResp.Worktree.Status)
	
	// Verify physical worktree exists
	s.DirExists(worktreeResp.Path)
	s.FileExists(filepath.Join(worktreeResp.Path, "README.md"))
	s.FileExists(filepath.Join(worktreeResp.Path, "CLAUDE.md"))
	s.FileExists(filepath.Join(worktreeResp.Path, "vibeman.toml"))
	
	// Step 4: Start the worktree (if Docker is available)
	if s.isDockerAvailable() {
		err = s.worktreeOps.StartWorktree(ctx, worktreeResp.Worktree.ID)
		s.NoError(err)
		
		// Verify status changed
		worktreeRepo := db.NewWorktreeRepository(s.db)
		worktree, err := worktreeRepo.Get(ctx, worktreeResp.Worktree.ID)
		s.NoError(err)
		s.Equal(db.StatusRunning, worktree.Status)
		
		// Step 5: Stop the worktree
		err = s.worktreeOps.StopWorktree(ctx, worktreeResp.Worktree.ID)
		s.NoError(err)
		
		// Verify status changed back
		worktree, err = worktreeRepo.Get(ctx, worktreeResp.Worktree.ID)
		s.NoError(err)
		s.Equal(db.StatusStopped, worktree.Status)
	}
	
	// Step 6: Remove the worktree
	err = s.worktreeOps.RemoveWorktree(ctx, worktreeResp.Worktree.ID, true)
	s.NoError(err)
	
	// Verify worktree was removed from database
	worktreeRepo := db.NewWorktreeRepository(s.db)
	_, err = worktreeRepo.Get(ctx, worktreeResp.Worktree.ID)
	s.Error(err)
	
	// Verify physical worktree was removed
	s.NoDirExists(worktreeResp.Path)
	
	// Step 7: Remove the repository
	err = s.repoOps.RemoveRepository(ctx, repo.ID)
	s.NoError(err)
	
	// Verify repository was removed
	repoRepo := db.NewRepositoryRepository(s.db)
	_, err = repoRepo.GetByID(ctx, repo.ID)
	s.Error(err)
}

func (s *WorktreeLifecycleTestSuite) TestWorktreeWithUncommittedChanges() {
	ctx := context.Background()
	
	// Create repository and worktree
	testRepoPath := filepath.Join(s.testDir, "test-repo-2")
	s.createTestRepository(testRepoPath)
	
	repo, err := s.repoOps.AddRepository(ctx, operations.AddRepositoryRequest{
		Path: testRepoPath,
		Name: "test-repo-2",
	})
	s.Require().NoError(err)
	
	worktreeResp, err := s.worktreeOps.CreateWorktree(ctx, operations.CreateWorktreeRequest{
		RepositoryID: repo.ID,
		Name:        "feature-changes",
		Branch:      "feature/changes",
		BaseBranch:  "main",
		SkipSetup:   true,
	})
	s.Require().NoError(err)
	
	// Make changes to the worktree
	testFile := filepath.Join(worktreeResp.Path, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	s.Require().NoError(err)
	
	// Try to remove without force - should fail
	err = s.worktreeOps.RemoveWorktree(ctx, worktreeResp.Worktree.ID, false)
	s.Error(err)
	// Should fail due to uncommitted changes or unpushed commits
	s.True(
		strings.Contains(err.Error(), "uncommitted") || 
		strings.Contains(err.Error(), "unpushed") ||
		strings.Contains(err.Error(), "reference not found"),
		"Expected error about uncommitted changes, unpushed commits, or git reference issues, got: %s", err.Error())
	
	// Remove with force - should succeed
	err = s.worktreeOps.RemoveWorktree(ctx, worktreeResp.Worktree.ID, true)
	s.NoError(err)
}

func (s *WorktreeLifecycleTestSuite) TestMultipleWorktrees() {
	ctx := context.Background()
	
	// Create repository
	testRepoPath := filepath.Join(s.testDir, "test-repo-3")
	s.createTestRepository(testRepoPath)
	
	repo, err := s.repoOps.AddRepository(ctx, operations.AddRepositoryRequest{
		Path: testRepoPath,
		Name: "test-repo-3",
	})
	s.Require().NoError(err)
	
	// Create multiple worktrees
	worktrees := []string{"feature-1", "feature-2", "feature-3"}
	createdWorktrees := make([]*db.Worktree, 0, len(worktrees))
	
	for _, name := range worktrees {
		resp, err := s.worktreeOps.CreateWorktree(ctx, operations.CreateWorktreeRequest{
			RepositoryID: repo.ID,
			Name:        name,
			Branch:      "feature/" + name,
			BaseBranch:  "main",
			SkipSetup:   true,
		})
		s.Require().NoError(err)
		createdWorktrees = append(createdWorktrees, resp.Worktree)
	}
	
	// Verify all worktrees exist
	worktreeRepo := db.NewWorktreeRepository(s.db)
	listedWorktrees, err := worktreeRepo.List(ctx, repo.ID, "")
	s.NoError(err)
	s.Len(listedWorktrees, len(worktrees))
	
	// Try to remove repository with active worktrees - should fail
	err = s.repoOps.RemoveRepository(ctx, repo.ID)
	s.Error(err)
	s.Contains(err.Error(), "active worktrees")
	
	// Remove all worktrees
	for _, wt := range createdWorktrees {
		err = s.worktreeOps.RemoveWorktree(ctx, wt.ID, true)
		s.NoError(err)
	}
	
	// Now repository removal should succeed
	err = s.repoOps.RemoveRepository(ctx, repo.ID)
	s.NoError(err)
}

// Helper methods

func (s *WorktreeLifecycleTestSuite) createTestRepository(path string) {
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
	
	// Commit files
	err = s.gitMgr.AddAndCommit(ctx, path, "Initial commit")
	s.Require().NoError(err)
}

func (s *WorktreeLifecycleTestSuite) isDockerAvailable() bool {
	// Try to list containers - if it works, Docker is available
	ctx := context.Background()
	_, err := s.containerMgr.List(ctx)
	return err == nil
}

// worktreeServiceAdapter adapts service.Manager to operations.ServiceManager interface
type worktreeServiceAdapter struct {
	mgr *service.Manager
}

func (a *worktreeServiceAdapter) StartService(ctx context.Context, name string) error {
	return a.mgr.StartService(ctx, name)
}

func (a *worktreeServiceAdapter) StopService(ctx context.Context, name string) error {
	return a.mgr.StopService(ctx, name)
}

func (a *worktreeServiceAdapter) GetService(name string) (interface{}, error) {
	return a.mgr.GetService(name)
}

func (a *worktreeServiceAdapter) HealthCheck(ctx context.Context, name string) error {
	return a.mgr.HealthCheck(ctx, name)
}

func (a *worktreeServiceAdapter) AddReference(serviceName, repoName string) error {
	return a.mgr.AddReference(serviceName, repoName)
}

func (a *worktreeServiceAdapter) RemoveReference(serviceName, repoName string) error {
	return a.mgr.RemoveReference(serviceName, repoName)
}

// TestWorkLifecycleIntegration runs the integration test suite
func TestWorktreeLifecycleIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	suite.Run(t, new(WorktreeLifecycleTestSuite))
}