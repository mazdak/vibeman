// +build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"vibeman/internal/config"
	"vibeman/internal/git"

	"github.com/stretchr/testify/suite"
)

// GitIntegrationTestSuite tests Git operations comprehensively
type GitIntegrationTestSuite struct {
	suite.Suite
	testDir   string
	gitMgr    *git.Manager
	configMgr *config.Manager
}

func (s *GitIntegrationTestSuite) SetupSuite() {
	// Create test directory
	testDir, err := os.MkdirTemp("", "vibeman-git-integration-*")
	s.Require().NoError(err)
	s.testDir = testDir

	// Initialize managers
	s.configMgr = config.New()
	s.gitMgr = git.New(s.configMgr)
}

func (s *GitIntegrationTestSuite) TearDownSuite() {
	if s.testDir != "" {
		os.RemoveAll(s.testDir)
	}
}

func (s *GitIntegrationTestSuite) SetupTest() {
	// Clean up any leftover git repositories from previous tests
	entries, _ := os.ReadDir(s.testDir)
	for _, entry := range entries {
		if entry.IsDir() {
			os.RemoveAll(filepath.Join(s.testDir, entry.Name()))
		}
	}
}

// Test: Repository Initialization and Basic Operations
func (s *GitIntegrationTestSuite) TestRepositoryInitialization() {
	ctx := context.Background()
	repoPath := filepath.Join(s.testDir, "test-repo")

	// Create directory first
	err := os.MkdirAll(repoPath, 0755)
	s.Require().NoError(err)

	// Test repository initialization
	err = s.gitMgr.InitRepository(ctx, repoPath)
	s.NoError(err)

	// Verify repository was created
	s.True(s.gitMgr.IsRepository(repoPath))
	s.DirExists(filepath.Join(repoPath, ".git"))

	// Test creating initial files and commit
	readmePath := filepath.Join(repoPath, "README.md")
	err = os.WriteFile(readmePath, []byte("# Test Repository\n"), 0644)
	s.Require().NoError(err)

	err = s.gitMgr.AddAndCommit(ctx, repoPath, "Initial commit")
	s.NoError(err)

	// Verify commit was created
	commit, err := s.gitMgr.GetCommitInfo(ctx, repoPath)
	s.NoError(err)
	s.NotNil(commit)
	s.Equal("Initial commit", strings.TrimSpace(commit.Message))
}

// Test: Branch Operations
func (s *GitIntegrationTestSuite) TestBranchOperations() {
	ctx := context.Background()
	repoPath := filepath.Join(s.testDir, "branch-repo")

	// Initialize repository with initial commit
	s.setupRepository(repoPath)

	// Test getting current branch (should be main/master)
	currentBranch, err := s.gitMgr.GetCurrentBranch(ctx, repoPath)
	s.NoError(err)
	s.True(currentBranch == "main" || currentBranch == "master")

	// Test getting default branch
	defaultBranch, err := s.gitMgr.GetDefaultBranch(ctx, repoPath)
	s.NoError(err)
	s.NotEmpty(defaultBranch)

	// Test getting all branches
	branches, err := s.gitMgr.GetBranches(ctx, repoPath)
	s.NoError(err)
	s.Contains(branches, currentBranch)

	// For now, test that switching to a non-existent branch fails
	// (This tests the current behavior; in the future we might want to support branch creation)
	err = s.gitMgr.SwitchBranch(ctx, repoPath, "feature/test-branch")
	s.Error(err)
	s.Contains(err.Error(), "does not exist")

	// Create a branch manually using git command to test switching
	// This simulates what would happen if the branch existed
	cmd := exec.Command("git", "-C", repoPath, "checkout", "-b", "feature/test-branch")
	err = cmd.Run()
	s.NoError(err)

	// Now verify we're on the new branch
	newCurrentBranch, err := s.gitMgr.GetCurrentBranch(ctx, repoPath)
	s.NoError(err)
	s.Equal("feature/test-branch", newCurrentBranch)

	// Verify branch appears in branch list
	branches, err = s.gitMgr.GetBranches(ctx, repoPath)
	s.NoError(err)
	s.Contains(branches, "feature/test-branch")

	// Test switching back to main branch
	err = s.gitMgr.SwitchBranch(ctx, repoPath, currentBranch)
	s.NoError(err)

	// Verify we're back on the original branch
	backToBranch, err := s.gitMgr.GetCurrentBranch(ctx, repoPath)
	s.NoError(err)
	s.Equal(currentBranch, backToBranch)
}

// Test: Worktree Operations (Extended from our working tests)
func (s *GitIntegrationTestSuite) TestWorktreeOperations() {
	ctx := context.Background()
	repoPath := filepath.Join(s.testDir, "worktree-repo")
	worktreePath := filepath.Join(s.testDir, "worktree-dir")

	// Initialize repository with initial commit
	s.setupRepository(repoPath)

	// Test creating a worktree
	err := s.gitMgr.CreateWorktree(ctx, repoPath, "feature/worktree-test", worktreePath)
	s.NoError(err)

	// Verify worktree was created
	s.DirExists(worktreePath)
	s.FileExists(filepath.Join(worktreePath, "README.md"))

	// Verify it's recognized as a worktree
	s.True(s.gitMgr.IsWorktree(worktreePath))

	// Test listing worktrees
	worktrees, err := s.gitMgr.ListWorktrees(ctx, repoPath)
	s.NoError(err)
	s.True(len(worktrees) >= 1) // At least the main repo + our worktree

	// Find our worktree in the list
	found := false
	for _, wt := range worktrees {
		if strings.Contains(wt.Path, "worktree-dir") {
			found = true
			s.Equal("feature/worktree-test", wt.Branch)
			break
		}
	}
	s.True(found, "Created worktree should appear in worktree list")

	// Test getting main repo path from worktree
	mainRepoPath, err := s.gitMgr.GetMainRepoPathFromWorktree(worktreePath)
	s.NoError(err)
	// Normalize paths to handle macOS /private/var vs /var differences
	expectedPath, _ := filepath.EvalSymlinks(repoPath)
	actualPath, _ := filepath.EvalSymlinks(mainRepoPath)
	s.Equal(expectedPath, actualPath)

	// Test removing the worktree
	err = s.gitMgr.RemoveWorktree(ctx, worktreePath)
	s.NoError(err)

	// Verify worktree was removed
	s.NoDirExists(worktreePath)
}

// Test: Change Detection and Status
func (s *GitIntegrationTestSuite) TestChangeDetection() {
	ctx := context.Background()
	repoPath := filepath.Join(s.testDir, "changes-repo")

	// Initialize repository with initial commit
	s.setupRepository(repoPath)

	// Initially should have no uncommitted changes
	hasChanges, err := s.gitMgr.HasUncommittedChanges(ctx, repoPath)
	s.NoError(err)
	s.False(hasChanges)

	// Create a new file
	newFile := filepath.Join(repoPath, "new-file.txt")
	err = os.WriteFile(newFile, []byte("new content"), 0644)
	s.Require().NoError(err)

	// Should now have uncommitted changes
	hasChanges, err = s.gitMgr.HasUncommittedChanges(ctx, repoPath)
	s.NoError(err)
	s.True(hasChanges)

	// Commit the changes
	err = s.gitMgr.AddAndCommit(ctx, repoPath, "Add new file")
	s.NoError(err)

	// Should no longer have uncommitted changes
	hasChanges, err = s.gitMgr.HasUncommittedChanges(ctx, repoPath)
	s.NoError(err)
	s.False(hasChanges)

	// Modify existing file
	err = os.WriteFile(newFile, []byte("modified content"), 0644)
	s.Require().NoError(err)

	// Should have uncommitted changes again
	hasChanges, err = s.gitMgr.HasUncommittedChanges(ctx, repoPath)
	s.NoError(err)
	s.True(hasChanges)
}

// Test: Repository Information and Metadata
func (s *GitIntegrationTestSuite) TestRepositoryInformation() {
	ctx := context.Background()
	repoPath := filepath.Join(s.testDir, "info-repo")

	// Initialize repository with initial commit
	s.setupRepository(repoPath)

	// Test getting repository information
	repo, err := s.gitMgr.GetRepository(ctx, repoPath)
	s.NoError(err)
	s.NotNil(repo)
	s.Equal(repoPath, repo.Path)
	s.NotEmpty(repo.Branch)

	// Test commit information
	commit, err := s.gitMgr.GetCommitInfo(ctx, repoPath)
	s.NoError(err)
	s.NotNil(commit)
	s.Equal("Initial commit", strings.TrimSpace(commit.Message))
	s.NotEmpty(commit.Hash.String())

	// Test multiple commits
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	s.Require().NoError(err)

	err = s.gitMgr.AddAndCommit(ctx, repoPath, "Add test file")
	s.NoError(err)

	// Latest commit should be the new one
	commit, err = s.gitMgr.GetCommitInfo(ctx, repoPath)
	s.NoError(err)
	s.Equal("Add test file", strings.TrimSpace(commit.Message))
}

// Test: Worktree Metadata and Path Resolution
func (s *GitIntegrationTestSuite) TestWorktreeMetadata() {
	ctx := context.Background()
	repoPath := filepath.Join(s.testDir, "metadata-repo")
	worktreePath := filepath.Join(s.testDir, "metadata-worktree")

	// Initialize repository
	s.setupRepository(repoPath)

	// Create worktree
	err := s.gitMgr.CreateWorktree(ctx, repoPath, "feature/metadata", worktreePath)
	s.NoError(err)

	// Test path resolution methods
	mainPath, err := s.gitMgr.GetMainRepoPathFromWorktree(worktreePath)
	s.NoError(err)
	// Normalize paths to handle macOS /private/var vs /var differences
	expectedPath, _ := filepath.EvalSymlinks(repoPath)
	actualPath, _ := filepath.EvalSymlinks(mainPath)
	s.Equal(expectedPath, actualPath)

	// Test environment extraction (if applicable)
	// This might return empty or default values in test scenarios
	env, err := s.gitMgr.GetEnvironmentFromWorktree(worktreePath)
	if err == nil {
		s.NotNil(env) // Could be empty string, but shouldn't be nil
	}

	// Test repository name extraction
	repoName, err := s.gitMgr.GetRepositoryNameFromWorktree(worktreePath)
	if err == nil {
		s.NotEmpty(repoName)
	}

	// Test combined path information
	repoNameFromPath, envFromPath, err := s.gitMgr.GetRepositoryAndEnvironmentFromPath(worktreePath)
	if err == nil {
		s.NotEmpty(repoNameFromPath)
		// envFromPath might be empty in test scenarios
		s.NotNil(envFromPath)
	}

	// Clean up
	err = s.gitMgr.RemoveWorktree(ctx, worktreePath)
	s.NoError(err)
}

// Test: Error Conditions and Edge Cases
func (s *GitIntegrationTestSuite) TestErrorConditions() {
	ctx := context.Background()

	// Test operations on non-existent repository
	nonExistentPath := filepath.Join(s.testDir, "non-existent")
	
	s.False(s.gitMgr.IsRepository(nonExistentPath))
	
	// These should return errors for non-existent repos
	_, err := s.gitMgr.GetCurrentBranch(ctx, nonExistentPath)
	s.Error(err)

	_, err = s.gitMgr.GetBranches(ctx, nonExistentPath)
	s.Error(err)

	_, err = s.gitMgr.HasUncommittedChanges(ctx, nonExistentPath)
	s.Error(err)

	// Test creating worktree with invalid parameters
	repoPath := filepath.Join(s.testDir, "error-repo")
	s.setupRepository(repoPath)

	// Try to create worktree in existing directory
	existingDir := filepath.Join(s.testDir, "existing")
	err = os.MkdirAll(existingDir, 0755)
	s.Require().NoError(err)

	err = s.gitMgr.CreateWorktree(ctx, repoPath, "feature/existing", existingDir)
	s.Error(err)
	s.Contains(err.Error(), "already exists")
}

// Test: Branch Merging and Relationships
func (s *GitIntegrationTestSuite) TestBranchRelationships() {
	ctx := context.Background()
	repoPath := filepath.Join(s.testDir, "merge-repo")

	// Initialize repository
	s.setupRepository(repoPath)

	// Make a second commit on main to establish history
	mainFile := filepath.Join(repoPath, "main.txt")
	err := os.WriteFile(mainFile, []byte("main content"), 0644)
	s.Require().NoError(err)
	err = s.gitMgr.AddAndCommit(ctx, repoPath, "Add main file")
	s.NoError(err)

	// Create and switch to feature branch using git command
	cmd := exec.Command("git", "-C", repoPath, "checkout", "-b", "feature/merge-test")
	err = cmd.Run()
	s.NoError(err)

	// Verify we're on the feature branch
	currentBranch, err := s.gitMgr.GetCurrentBranch(ctx, repoPath)
	s.NoError(err)
	s.Equal("feature/merge-test", currentBranch, "Should be on feature branch")

	// Add commit to feature branch
	featureFile := filepath.Join(repoPath, "feature.txt")
	err = os.WriteFile(featureFile, []byte("feature content"), 0644)
	s.Require().NoError(err)

	err = s.gitMgr.AddAndCommit(ctx, repoPath, "Add feature file")
	s.NoError(err)

	// Verify file is tracked on feature branch
	cmd = exec.Command("git", "-C", repoPath, "ls-files")
	featureOutput, err := cmd.Output()
	s.NoError(err)
	filesInFeatureBranch := strings.TrimSpace(string(featureOutput))
	s.Contains(filesInFeatureBranch, "feature.txt", "feature.txt should be tracked in feature branch")

	// Switch back to main branch
	mainBranch, err := s.gitMgr.GetDefaultBranch(ctx, repoPath)
	s.NoError(err)

	err = s.gitMgr.SwitchBranch(ctx, repoPath, mainBranch)
	s.NoError(err)

	// Verify we're actually on the main branch
	currentBranch, err = s.gitMgr.GetCurrentBranch(ctx, repoPath)
	s.NoError(err)
	s.Equal(mainBranch, currentBranch, "Should be on main branch after switch")

	// Debug: List files manually
	cmd = exec.Command("git", "-C", repoPath, "ls-files")
	mainOutput, err := cmd.Output()
	s.NoError(err, "Git ls-files should work")
	
	filesInRepo := strings.TrimSpace(string(mainOutput))
	s.NotContains(filesInRepo, "feature.txt", "feature.txt should not be tracked in main branch")

	// Verify feature file doesn't exist on main branch
	s.NoFileExists(featureFile)

	// Test branch merged status (feature branch should not be merged yet)
	isMerged, err := s.gitMgr.IsBranchMerged(ctx, repoPath, "feature/merge-test")
	s.NoError(err)
	s.False(isMerged)

	// Switch back to feature branch to verify file exists
	err = s.gitMgr.SwitchBranch(ctx, repoPath, "feature/merge-test")
	s.NoError(err)
	s.FileExists(featureFile)
}

// Test: Complex Worktree Scenarios
func (s *GitIntegrationTestSuite) TestComplexWorktreeScenarios() {
	ctx := context.Background()
	repoPath := filepath.Join(s.testDir, "complex-repo")

	// Initialize repository
	s.setupRepository(repoPath)

	// Create multiple worktrees
	worktree1 := filepath.Join(s.testDir, "worktree-1")
	worktree2 := filepath.Join(s.testDir, "worktree-2")
	worktree3 := filepath.Join(s.testDir, "worktree-3")

	err := s.gitMgr.CreateWorktree(ctx, repoPath, "feature/wt1", worktree1)
	s.NoError(err)

	err = s.gitMgr.CreateWorktree(ctx, repoPath, "feature/wt2", worktree2)
	s.NoError(err)

	err = s.gitMgr.CreateWorktree(ctx, repoPath, "feature/wt3", worktree3)
	s.NoError(err)

	// List all worktrees
	worktrees, err := s.gitMgr.ListWorktrees(ctx, repoPath)
	s.NoError(err)
	s.True(len(worktrees) >= 4) // Main repo + 3 worktrees

	// Verify each worktree has correct branch
	worktreeMap := make(map[string]string)
	for _, wt := range worktrees {
		if strings.Contains(wt.Path, "worktree-") {
			worktreeMap[wt.Path] = wt.Branch
		}
	}

	// Normalize paths to handle macOS /private/var vs /var differences
	normalizedWorktree1, _ := filepath.EvalSymlinks(worktree1)
	normalizedWorktree2, _ := filepath.EvalSymlinks(worktree2)
	normalizedWorktree3, _ := filepath.EvalSymlinks(worktree3)

	// Check if any of the normalized paths exist in the map
	found1 := false
	found2 := false
	found3 := false
	for path := range worktreeMap {
		normalizedPath, _ := filepath.EvalSymlinks(path)
		if normalizedPath == normalizedWorktree1 {
			found1 = true
		}
		if normalizedPath == normalizedWorktree2 {
			found2 = true
		}
		if normalizedPath == normalizedWorktree3 {
			found3 = true
		}
	}

	s.True(found1, "Worktree 1 should be found in the list")
	s.True(found2, "Worktree 2 should be found in the list")
	s.True(found3, "Worktree 3 should be found in the list")

	// Make changes in each worktree
	for i, wtPath := range []string{worktree1, worktree2, worktree3} {
		testFile := filepath.Join(wtPath, fmt.Sprintf("wt%d.txt", i+1))
		err = os.WriteFile(testFile, []byte(fmt.Sprintf("content %d", i+1)), 0644)
		s.Require().NoError(err)

		err = s.gitMgr.AddAndCommit(ctx, wtPath, fmt.Sprintf("Add file in worktree %d", i+1))
		s.NoError(err)
	}

	// Verify each worktree has its own changes
	for i, wtPath := range []string{worktree1, worktree2, worktree3} {
		testFile := filepath.Join(wtPath, fmt.Sprintf("wt%d.txt", i+1))
		s.FileExists(testFile)

		// Verify other worktree files don't exist in this worktree
		for j := 1; j <= 3; j++ {
			if j != i+1 {
				otherFile := filepath.Join(wtPath, fmt.Sprintf("wt%d.txt", j))
				s.NoFileExists(otherFile)
			}
		}
	}

	// Clean up all worktrees
	for _, wtPath := range []string{worktree1, worktree2, worktree3} {
		err = s.gitMgr.RemoveWorktree(ctx, wtPath)
		s.NoError(err)
		s.NoDirExists(wtPath)
	}
}

// Helper Methods

func (s *GitIntegrationTestSuite) setupRepository(repoPath string) {
	ctx := context.Background()

	// Create directory first
	err := os.MkdirAll(repoPath, 0755)
	s.Require().NoError(err)

	// Initialize repository
	err = s.gitMgr.InitRepository(ctx, repoPath)
	s.Require().NoError(err)

	// Create initial file
	readmePath := filepath.Join(repoPath, "README.md")
	err = os.WriteFile(readmePath, []byte("# Test Repository\n"), 0644)
	s.Require().NoError(err)

	// Make initial commit
	err = s.gitMgr.AddAndCommit(ctx, repoPath, "Initial commit")
	s.Require().NoError(err)
}

// TestGitIntegration runs the git integration test suite
func TestGitIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite.Run(t, new(GitIntegrationTestSuite))
}