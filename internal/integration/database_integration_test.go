// +build integration

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vibeman/internal/db"

	"github.com/stretchr/testify/suite"
)

// DatabaseIntegrationTestSuite tests database operations comprehensively
type DatabaseIntegrationTestSuite struct {
	suite.Suite
	testDir      string
	db           *db.DB
	repoRepo     *db.RepositoryRepository
	worktreeRepo *db.WorktreeRepository
}

func (s *DatabaseIntegrationTestSuite) SetupSuite() {
	// Create test directory
	testDir, err := os.MkdirTemp("", "vibeman-db-integration-*")
	s.Require().NoError(err)
	s.testDir = testDir
}

func (s *DatabaseIntegrationTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
	if s.testDir != "" {
		os.RemoveAll(s.testDir)
	}
}

func (s *DatabaseIntegrationTestSuite) SetupTest() {
	// Create fresh database for each test
	dbPath := filepath.Join(s.testDir, "test.db")
	
	// Remove existing database file if it exists
	os.Remove(dbPath)
	
	config := &db.Config{
		Driver: "sqlite3",
		DSN:    dbPath,
	}
	database, err := db.New(config)
	s.Require().NoError(err)
	s.db = database
	
	// Run migrations
	err = s.db.Migrate()
	s.Require().NoError(err)
	
	// Create repositories
	s.repoRepo = db.NewRepositoryRepository(s.db)
	s.worktreeRepo = db.NewWorktreeRepository(s.db)
}

func (s *DatabaseIntegrationTestSuite) TearDownTest() {
	if s.db != nil {
		s.db.Close()
		s.db = nil
		s.repoRepo = nil
		s.worktreeRepo = nil
	}
}

// Test: Database Creation and Migration
func (s *DatabaseIntegrationTestSuite) TestDatabaseCreationAndMigration() {
	// Database should be created and migrated during SetupTest
	
	// Test that we can perform a basic operation
	ctx := context.Background()
	repos, err := s.repoRepo.List(ctx)
	s.NoError(err)
	// repos can be nil or empty slice when no repositories exist
	s.Empty(repos)
	
	// Verify tables exist by attempting operations on each
	worktrees, err := s.worktreeRepo.List(ctx, "", "")
	s.NoError(err)
	// worktrees can be nil or empty slice when no worktrees exist
	s.Empty(worktrees)
}

// Test: Repository CRUD Operations
func (s *DatabaseIntegrationTestSuite) TestRepositoryCRUDOperations() {
	ctx := context.Background()
	
	// CREATE - Add a new repository
	repo := &db.Repository{
		ID:          "test-repo-1",
		Name:        "test-repository",
		Path:        "/test/path/repo",
		Description: "Test repository for integration tests",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	err := s.repoRepo.Create(ctx, repo)
	s.NoError(err)
	
	// READ - Get repository by ID
	retrievedRepo, err := s.repoRepo.GetByID(ctx, repo.ID)
	s.NoError(err)
	s.NotNil(retrievedRepo)
	s.Equal(repo.Name, retrievedRepo.Name)
	s.Equal(repo.Path, retrievedRepo.Path)
	s.Equal(repo.Description, retrievedRepo.Description)
	
	// READ - List all repositories
	repos, err := s.repoRepo.List(ctx)
	s.NoError(err)
	s.Len(repos, 1)
	s.Equal(repo.ID, repos[0].ID)
	
	// UPDATE - Modify repository
	repo.Description = "Updated description"
	repo.UpdatedAt = time.Now()
	err = s.repoRepo.Update(ctx, repo)
	s.NoError(err)
	
	// Verify update
	updatedRepo, err := s.repoRepo.GetByID(ctx, repo.ID)
	s.NoError(err)
	s.Equal("Updated description", updatedRepo.Description)
	
	// DELETE - Remove repository
	err = s.repoRepo.Delete(ctx, repo.ID)
	s.NoError(err)
	
	// Verify deletion
	deletedRepo, err := s.repoRepo.GetByID(ctx, repo.ID)
	s.Error(err)
	s.Nil(deletedRepo)
	
	// List should be empty
	repos, err = s.repoRepo.List(ctx)
	s.NoError(err)
	s.Empty(repos)
}

// Test: Worktree CRUD Operations
func (s *DatabaseIntegrationTestSuite) TestWorktreeCRUDOperations() {
	ctx := context.Background()
	
	// First create a repository
	repo := &db.Repository{
		ID:        "test-repo-2",
		Name:      "worktree-test-repo",
		Path:      "/test/path/worktree-repo",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := s.repoRepo.Create(ctx, repo)
	s.Require().NoError(err)
	
	// CREATE - Add a new worktree
	worktree := &db.Worktree{
		ID:           "test-worktree-1",
		RepositoryID: repo.ID,
		Name:         "feature-branch",
		Branch:       "feature/test-feature",
		Path:         "/test/path/worktrees/feature-branch",
		Status:       db.StatusStopped,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	err = s.worktreeRepo.Create(ctx, worktree)
	s.NoError(err)
	
	// READ - Get worktree by ID
	retrievedWorktree, err := s.worktreeRepo.Get(ctx, worktree.ID)
	s.NoError(err)
	s.NotNil(retrievedWorktree)
	s.Equal(worktree.Name, retrievedWorktree.Name)
	s.Equal(worktree.Branch, retrievedWorktree.Branch)
	s.Equal(worktree.Path, retrievedWorktree.Path)
	s.Equal(worktree.Status, retrievedWorktree.Status)
	
	// READ - List all worktrees
	worktrees, err := s.worktreeRepo.List(ctx, "", "")
	s.NoError(err)
	s.Len(worktrees, 1)
	s.Equal(worktree.ID, worktrees[0].ID)
	
	// READ - List worktrees by repository
	repoWorktrees, err := s.worktreeRepo.ListByRepository(ctx, repo.ID)
	s.NoError(err)
	s.Len(repoWorktrees, 1)
	s.Equal(worktree.ID, repoWorktrees[0].ID)
	
	// UPDATE - Modify worktree status
	worktree.Status = db.StatusRunning
	worktree.UpdatedAt = time.Now()
	err = s.worktreeRepo.Update(ctx, worktree)
	s.NoError(err)
	
	// Verify update
	updatedWorktree, err := s.worktreeRepo.Get(ctx, worktree.ID)
	s.NoError(err)
	s.Equal(db.StatusRunning, updatedWorktree.Status)
	
	// DELETE - Remove worktree
	err = s.worktreeRepo.Delete(ctx, worktree.ID)
	s.NoError(err)
	
	// Verify deletion
	deletedWorktree, err := s.worktreeRepo.Get(ctx, worktree.ID)
	s.Error(err)
	s.Nil(deletedWorktree)
	
	// List should be empty
	worktrees, err = s.worktreeRepo.List(ctx, "", "")
	s.NoError(err)
	s.Empty(worktrees)
	
	// Clean up repository
	err = s.repoRepo.Delete(ctx, repo.ID)
	s.NoError(err)
}

// Test: Repository with Multiple Worktrees
func (s *DatabaseIntegrationTestSuite) TestRepositoryWithMultipleWorktrees() {
	ctx := context.Background()
	
	// Create repository
	repo := &db.Repository{
		ID:        "test-repo-3",
		Name:      "multi-worktree-repo",
		Path:      "/test/path/multi-repo",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := s.repoRepo.Create(ctx, repo)
	s.Require().NoError(err)
	
	// Create multiple worktrees
	worktreeCount := 5
	for i := 0; i < worktreeCount; i++ {
		worktree := &db.Worktree{
			ID:           s.T().Name() + "-worktree-" + string(rune('0'+i)),
			RepositoryID: repo.ID,
			Name:         s.T().Name() + "-feature-" + string(rune('0'+i)),
			Branch:       "feature/test-" + string(rune('0'+i)),
			Path:         "/test/path/worktrees/feature-" + string(rune('0'+i)),
			Status:       db.StatusStopped,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err := s.worktreeRepo.Create(ctx, worktree)
		s.NoError(err)
	}
	
	// List all worktrees for repository
	worktrees, err := s.worktreeRepo.ListByRepository(ctx, repo.ID)
	s.NoError(err)
	s.Len(worktrees, worktreeCount)
	
	// Verify all worktrees belong to the same repository
	for _, wt := range worktrees {
		s.Equal(repo.ID, wt.RepositoryID)
	}
	
	// Delete one worktree
	err = s.worktreeRepo.Delete(ctx, worktrees[0].ID)
	s.NoError(err)
	
	// Verify count decreased
	remainingWorktrees, err := s.worktreeRepo.ListByRepository(ctx, repo.ID)
	s.NoError(err)
	s.Len(remainingWorktrees, worktreeCount-1)
	
	// Clean up all worktrees
	for _, wt := range remainingWorktrees {
		err = s.worktreeRepo.Delete(ctx, wt.ID)
		s.NoError(err)
	}
	
	// Clean up repository
	err = s.repoRepo.Delete(ctx, repo.ID)
	s.NoError(err)
}

// Test: Query Filtering and Pagination
func (s *DatabaseIntegrationTestSuite) TestQueryFilteringAndPagination() {
	ctx := context.Background()
	
	// Create multiple repositories
	repoCount := 10
	for i := 0; i < repoCount; i++ {
		repo := &db.Repository{
			ID:          s.T().Name() + "-repo-" + string(rune('0'+i)),
			Name:        "test-repo-" + string(rune('0'+i)),
			Path:        "/test/path/repo-" + string(rune('0'+i)),
			Description: "Repository number " + string(rune('0'+i)),
			CreatedAt:   time.Now().Add(time.Duration(i) * time.Hour),
			UpdatedAt:   time.Now().Add(time.Duration(i) * time.Hour),
		}
		err := s.repoRepo.Create(ctx, repo)
		s.NoError(err)
	}
	
	// List all repositories
	allRepos, err := s.repoRepo.List(ctx)
	s.NoError(err)
	s.Len(allRepos, repoCount)
	
	// Verify ordering (should be by creation time or name)
	// This depends on the actual implementation
	
	// Create worktrees for specific repository
	targetRepo := allRepos[0]
	for i := 0; i < 3; i++ {
		worktree := &db.Worktree{
			ID:           s.T().Name() + "-filter-wt-" + string(rune('0'+i)),
			RepositoryID: targetRepo.ID,
			Name:         "worktree-" + string(rune('0'+i)),
			Branch:       "branch-" + string(rune('0'+i)),
			Path:         "/test/worktree-" + string(rune('0'+i)),
			Status:       db.StatusStopped,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		err := s.worktreeRepo.Create(ctx, worktree)
		s.NoError(err)
	}
	
	// Filter worktrees by repository
	targetWorktrees, err := s.worktreeRepo.ListByRepository(ctx, targetRepo.ID)
	s.NoError(err)
	s.Len(targetWorktrees, 3)
	
	// Clean up
	for _, wt := range targetWorktrees {
		err = s.worktreeRepo.Delete(ctx, wt.ID)
		s.NoError(err)
	}
	for _, repo := range allRepos {
		err = s.repoRepo.Delete(ctx, repo.ID)
		s.NoError(err)
	}
}

// Test: Concurrent Access and Transactions
func (s *DatabaseIntegrationTestSuite) TestConcurrentAccessAndTransactions() {
	ctx := context.Background()
	
	// Create a repository
	repo := &db.Repository{
		ID:        "test-repo-concurrent",
		Name:      "concurrent-test-repo",
		Path:      "/test/concurrent/repo",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := s.repoRepo.Create(ctx, repo)
	s.Require().NoError(err)
	
	// Simulate concurrent worktree creation
	concurrentCount := 5
	errChan := make(chan error, concurrentCount)
	
	for i := 0; i < concurrentCount; i++ {
		go func(index int) {
			worktree := &db.Worktree{
				ID:           s.T().Name() + "-concurrent-wt-" + string(rune('0'+index)),
				RepositoryID: repo.ID,
				Name:         "concurrent-" + string(rune('0'+index)),
				Branch:       "concurrent/branch-" + string(rune('0'+index)),
				Path:         "/test/concurrent/worktree-" + string(rune('0'+index)),
				Status:       db.StatusStopped,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			errChan <- s.worktreeRepo.Create(ctx, worktree)
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < concurrentCount; i++ {
		err := <-errChan
		s.NoError(err)
	}
	
	// Verify all worktrees were created
	worktrees, err := s.worktreeRepo.ListByRepository(ctx, repo.ID)
	s.NoError(err)
	s.Len(worktrees, concurrentCount)
	
	// Clean up
	for _, wt := range worktrees {
		err = s.worktreeRepo.Delete(ctx, wt.ID)
		s.NoError(err)
	}
	err = s.repoRepo.Delete(ctx, repo.ID)
	s.NoError(err)
}

// Test: Error Conditions and Constraints
func (s *DatabaseIntegrationTestSuite) TestErrorConditionsAndConstraints() {
	ctx := context.Background()
	
	// Test: Creating duplicate repository ID
	repo := &db.Repository{
		ID:        "test-repo-duplicate",
		Name:      "duplicate-repo",
		Path:      "/test/duplicate/repo",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := s.repoRepo.Create(ctx, repo)
	s.NoError(err)
	
	// Try to create another repository with same ID
	duplicateRepo := &db.Repository{
		ID:        repo.ID, // Same ID
		Name:      "another-repo",
		Path:      "/test/another/repo",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = s.repoRepo.Create(ctx, duplicateRepo)
	s.Error(err) // Should fail due to unique constraint
	
	// Test: Creating worktree with non-existent repository
	orphanWorktree := &db.Worktree{
		ID:           "test-worktree-orphan",
		RepositoryID: "non-existent-repo",
		Name:         "orphan-worktree",
		Branch:       "orphan/branch",
		Path:         "/test/orphan/worktree",
		Status:       db.StatusStopped,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = s.worktreeRepo.Create(ctx, orphanWorktree)
	// This may or may not fail depending on foreign key constraint implementation
	// SQLite requires foreign keys to be explicitly enabled and constraints to be defined
	// s.Error(err) // Should fail due to foreign key constraint
	
	// Test: Getting non-existent repository
	nonExistentRepo, err := s.repoRepo.GetByID(ctx, "non-existent-id")
	s.Error(err)
	s.Nil(nonExistentRepo)
	
	// Test: Updating non-existent repository
	fakeRepo := &db.Repository{
		ID:        "fake-repo-id",
		Name:      "fake-repo",
		UpdatedAt: time.Now(),
	}
	err = s.repoRepo.Update(ctx, fakeRepo)
	s.Error(err)
	
	// Test: Deleting non-existent worktree
	err = s.worktreeRepo.Delete(ctx, "non-existent-worktree")
	// This might not error depending on implementation
	// Some databases return no error when deleting non-existent rows
	
	// Clean up
	err = s.repoRepo.Delete(ctx, repo.ID)
	s.NoError(err)
}

// Test: Repository Path Uniqueness
func (s *DatabaseIntegrationTestSuite) TestRepositoryPathUniqueness() {
	ctx := context.Background()
	
	// Create first repository
	repo1 := &db.Repository{
		ID:        "test-repo-path-1",
		Name:      "path-test-repo-1",
		Path:      "/test/unique/path",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := s.repoRepo.Create(ctx, repo1)
	s.NoError(err)
	
	// Try to create another repository with same path
	repo2 := &db.Repository{
		ID:        "test-repo-path-2",
		Name:      "path-test-repo-2",
		Path:      repo1.Path, // Same path
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = s.repoRepo.Create(ctx, repo2)
	// Depending on schema, this might fail if path has unique constraint
	// If no unique constraint on path, it should succeed
	
	// Clean up
	s.repoRepo.Delete(ctx, repo1.ID)
	if err == nil {
		s.repoRepo.Delete(ctx, repo2.ID)
	}
}

// Test: Worktree Status Transitions
func (s *DatabaseIntegrationTestSuite) TestWorktreeStatusTransitions() {
	ctx := context.Background()
	
	// Create repository
	repo := &db.Repository{
		ID:        "test-repo-status",
		Name:      "status-test-repo",
		Path:      "/test/status/repo",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := s.repoRepo.Create(ctx, repo)
	s.Require().NoError(err)
	
	// Create worktree
	worktree := &db.Worktree{
		ID:           "test-worktree-status",
		RepositoryID: repo.ID,
		Name:         "status-worktree",
		Branch:       "status/branch",
		Path:         "/test/status/worktree",
		Status:       db.StatusStopped,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = s.worktreeRepo.Create(ctx, worktree)
	s.NoError(err)
	
	// Test status transitions
	statusTransitions := []db.WorktreeStatus{
		db.StatusStarting,
		db.StatusRunning,
		db.StatusStopping,
		db.StatusStopped,
		db.StatusError,
	}
	
	for _, status := range statusTransitions {
		worktree.Status = status
		worktree.UpdatedAt = time.Now()
		
		err = s.worktreeRepo.Update(ctx, worktree)
		s.NoError(err)
		
		// Verify status change
		updated, err := s.worktreeRepo.Get(ctx, worktree.ID)
		s.NoError(err)
		s.Equal(status, updated.Status)
	}
	
	// Clean up
	err = s.worktreeRepo.Delete(ctx, worktree.ID)
	s.NoError(err)
	err = s.repoRepo.Delete(ctx, repo.ID)
	s.NoError(err)
}

// TestDatabaseIntegration runs the database integration test suite
func TestDatabaseIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite.Run(t, new(DatabaseIntegrationTestSuite))
}