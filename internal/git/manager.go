package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	configpkg "vibeman/internal/config"
	containerpkg "vibeman/internal/container"
	"vibeman/internal/xdg"
)

// Manager handles Git operations including worktree management
type Manager struct {
	config *configpkg.Manager
}

// SetConfig updates the configuration manager
func (m *Manager) SetConfig(cfg *configpkg.Manager) {
	m.config = cfg
}


// Worktree represents a git worktree
type Worktree struct {
	Path      string
	Branch    string
	Commit    string
	IsMain    bool
	IsBare    bool
	IsLocked  bool
	CreatedAt time.Time
}

// Repository represents a git repository
type Repository struct {
	Path      string
	URL       string
	Branch    string
	Worktrees []Worktree
}

// New creates a new Git manager
func New(cfg *configpkg.Manager) *Manager {
	return &Manager{
		config: cfg,
	}
}

// CreateWorktree creates a new git worktree
func (m *Manager) CreateWorktree(ctx context.Context, repoURL, branch, path string) error {
	// Ensure the path is absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if path already exists
	if _, err := os.Stat(absPath); err == nil {
		return fmt.Errorf("worktree path already exists: %s", absPath)
	}

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// First check if we have a main repository to create worktree from
	var mainRepoPath string
	if filepath.IsAbs(repoURL) || strings.HasPrefix(repoURL, "./") || strings.HasPrefix(repoURL, "../") {
		// repoURL is already a local path, use it directly
		mainRepoPath = repoURL
	} else {
		// repoURL is a remote URL, get the configured path
		mainRepoPath = m.GetMainRepoPath(repoURL)
	}

	// If main repository doesn't exist, clone it first
	if _, err := os.Stat(mainRepoPath); os.IsNotExist(err) {
		if err := m.cloneMainRepository(ctx, repoURL, mainRepoPath); err != nil {
			return fmt.Errorf("failed to clone main repository: %w", err)
		}
	}

	// Open the main repository
	repo, err := git.PlainOpen(mainRepoPath)
	if err != nil {
		return fmt.Errorf("failed to open main repository: %w", err)
	}

	// Fetch latest changes
	if err := m.fetchRepository(ctx, repo); err != nil && err != git.NoErrAlreadyUpToDate {
		// Log fetch error but continue - it might work offline
		fmt.Fprintf(os.Stderr, "Warning: failed to fetch latest changes: %v\n", err)
	}

	// Check if branch exists locally or remotely
	branchRef := plumbing.NewBranchReferenceName(branch)
	remoteBranchRef := plumbing.NewRemoteReferenceName("origin", branch)

	_, localErr := repo.Reference(branchRef, true)
	_, remoteErr := repo.Reference(remoteBranchRef, true)

	if localErr != nil && remoteErr != nil {
		// Branch doesn't exist locally or remotely, create it
		if err := m.createNewBranch(ctx, repo, branch, mainRepoPath); err != nil {
			return fmt.Errorf("failed to create new branch: %w", err)
		}
	} else if localErr != nil && remoteErr == nil {
		// Remote branch exists but not local, create local tracking branch
		if err := m.createLocalBranch(repo, branch); err != nil {
			return fmt.Errorf("failed to create local branch: %w", err)
		}
	}

	// Use git command to create worktree since go-git doesn't support it directly
	if err := m.createWorktreeWithCmd(ctx, absPath, branch, mainRepoPath); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	return nil
}

// ListWorktrees lists all worktrees for a repository
func (m *Manager) ListWorktrees(ctx context.Context, repoPath string) ([]containerpkg.GitWorktree, error) {
	// If repoPath is a URL, convert to main repo path
	if strings.HasPrefix(repoPath, "http") || strings.HasPrefix(repoPath, "git@") {
		repoPath = m.GetMainRepoPath(repoPath)
	}

	// Check if path exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository path does not exist: %s", repoPath)
	}

	return m.listWorktreesWithCmd(ctx, repoPath)
}

// RemoveWorktree removes a git worktree
func (m *Manager) RemoveWorktree(ctx context.Context, path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if the path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("worktree path does not exist: %s", absPath)
	}

	// Find the main repository that contains this worktree
	mainRepoPath, err := m.findMainRepoForWorktree(absPath)
	if err != nil {
		return fmt.Errorf("failed to find main repository: %w", err)
	}

	// Remove the worktree
	if err := m.removeWorktreeWithCmd(ctx, absPath, mainRepoPath); err != nil {
		// If git command fails, try to force remove
		if forceErr := m.forceRemoveWorktree(ctx, absPath, mainRepoPath); forceErr != nil {
			return fmt.Errorf("failed to remove worktree: %w (force remove also failed: %v)", err, forceErr)
		}
	}

	return nil
}

// SwitchBranch switches to a different branch in the worktree
func (m *Manager) SwitchBranch(ctx context.Context, path, branch string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("worktree path does not exist: %s", absPath)
	}

	repo, err := git.PlainOpen(absPath)
	if err != nil {
		return fmt.Errorf("failed to open repository at %s: %w", absPath, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Check for uncommitted changes
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get worktree status: %w", err)
	}

	if !status.IsClean() {
		return fmt.Errorf("cannot switch branch: worktree has uncommitted changes")
	}

	// Check if branch exists
	branchRef := plumbing.NewBranchReferenceName(branch)
	_, err = repo.Reference(branchRef, true)
	if err != nil {
		// Branch doesn't exist, try to create it from remote
		remoteBranchRef := plumbing.NewRemoteReferenceName("origin", branch)
		if _, err := repo.Reference(remoteBranchRef, true); err != nil {
			return fmt.Errorf("branch %s does not exist locally or remotely", branch)
		}

		if err := m.createLocalBranch(repo, branch); err != nil {
			return fmt.Errorf("failed to create local branch: %w", err)
		}
	}

	// Checkout the branch
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
		Force:  false,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branch, err)
	}

	return nil
}

// UpdateWorktree updates a worktree by pulling latest changes
func (m *Manager) UpdateWorktree(ctx context.Context, path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("worktree path does not exist: %s", absPath)
	}

	repo, err := git.PlainOpen(absPath)
	if err != nil {
		return fmt.Errorf("failed to open repository at %s: %w", absPath, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Check for uncommitted changes
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get worktree status: %w", err)
	}

	if !status.IsClean() {
		return fmt.Errorf("cannot update worktree: uncommitted changes present")
	}

	// Fetch latest changes
	if err := m.fetchRepository(ctx, repo); err != nil && err != git.NoErrAlreadyUpToDate {
		// Log warning but continue - might work offline
		fmt.Fprintf(os.Stderr, "Warning: failed to fetch latest changes: %v\n", err)
	}

	// Pull changes
	err = worktree.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       m.getAuthMethod(),
		Progress:   os.Stdout,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull changes: %w", err)
	}

	return nil
}

// CloneRepository clones a repository to the specified path
func (m *Manager) CloneRepository(ctx context.Context, repoURL, path string) error {
	// Validate repository URL
	if repoURL == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if path already exists
	if _, err := os.Stat(absPath); err == nil {
		return fmt.Errorf("path already exists: %s", absPath)
	}

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Clone repository
	_, err = git.PlainCloneContext(ctx, absPath, false, &git.CloneOptions{
		URL:      repoURL,
		Auth:     m.getAuthMethod(),
		Progress: os.Stdout,
		Depth:    0, // Full clone
	})
	if err != nil {
		// Clean up partial clone on failure
		os.RemoveAll(absPath)

		// Provide more specific error messages
		if strings.Contains(err.Error(), "authentication") {
			return fmt.Errorf("authentication failed while cloning %s: %w", repoURL, err)
		} else if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "repository not found") {
			return fmt.Errorf("repository not found: %s", repoURL)
		} else if ctx.Err() != nil {
			return fmt.Errorf("clone cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("failed to clone repository from %s: %w", repoURL, err)
	}

	return nil
}

// GetRepository returns repository information
func (m *Manager) GetRepository(ctx context.Context, path string) (*Repository, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	repo, err := git.PlainOpen(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get repository URL
	remotes, err := repo.Remotes()
	if err != nil {
		return nil, fmt.Errorf("failed to get remotes: %w", err)
	}

	var repoURL string
	if len(remotes) > 0 {
		repoURL = remotes[0].Config().URLs[0]
	}

	// Get current branch
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	var branch string
	if head.Name().IsBranch() {
		branch = head.Name().Short()
	}

	// Get worktrees
	containerWorktrees, err := m.ListWorktrees(ctx, absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Convert container worktrees to git worktrees
	worktrees := make([]Worktree, len(containerWorktrees))
	for i, cw := range containerWorktrees {
		createdAt, _ := time.Parse(time.RFC3339, cw.CreatedAt)
		worktrees[i] = Worktree{
			Path:      cw.Path,
			Branch:    cw.Branch,
			Commit:    cw.Commit,
			IsMain:    cw.IsMain,
			IsBare:    cw.IsBare,
			IsLocked:  cw.IsLocked,
			CreatedAt: createdAt,
		}
	}

	return &Repository{
		Path:      absPath,
		URL:       repoURL,
		Branch:    branch,
		Worktrees: worktrees,
	}, nil
}

// Helper methods

// GetMainRepoPath returns the path where the main repository should be stored
func (m *Manager) GetMainRepoPath(repoURL string) string {
	// Extract repository name from URL
	parts := strings.Split(repoURL, "/")
	repoName := parts[len(parts)-1]
	if strings.HasSuffix(repoName, ".git") {
		repoName = repoName[:len(repoName)-4]
	}


	// Use project worktrees directory if configured
	baseDir := ""
	if m.config != nil && m.config.Repository != nil && m.config.Repository.Repository.Worktrees.Directory != "" {
		baseDir = m.config.Repository.Repository.Worktrees.Directory
	} else {
		// Default to XDG data directory structure
		dataDir, err := xdg.DataDir()
		if err == nil {
			baseDir = filepath.Join(dataDir, "repos")
		} else {
			// Fallback to sibling directory
			baseDir = "../" + repoName + "-worktrees"
		}
	}

	return filepath.Join(baseDir, ".repos", repoName)
}

// cloneMainRepository clones the main repository as a bare repository
func (m *Manager) cloneMainRepository(ctx context.Context, repoURL, path string) error {
	// Create directory
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Clone as bare repository
	_, err := git.PlainCloneContext(ctx, path, true, &git.CloneOptions{
		URL:      repoURL,
		Auth:     m.getAuthMethod(),
		Progress: os.Stdout,
	})
	if err != nil {
		return fmt.Errorf("failed to clone bare repository: %w", err)
	}

	return nil
}

// fetchRepository fetches latest changes from remote
func (m *Manager) fetchRepository(ctx context.Context, repo *git.Repository) error {
	return repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName: "origin",
		Auth:       m.getAuthMethod(),
	})
}

// createLocalBranch creates a local branch from remote branch
func (m *Manager) createLocalBranch(repo *git.Repository, branch string) error {
	// Get remote branch reference
	remoteBranchRef := plumbing.NewRemoteReferenceName("origin", branch)
	remoteRef, err := repo.Reference(remoteBranchRef, true)
	if err != nil {
		return fmt.Errorf("failed to get remote branch reference: %w", err)
	}

	// Create local branch reference
	localBranchRef := plumbing.NewBranchReferenceName(branch)
	ref := plumbing.NewHashReference(localBranchRef, remoteRef.Hash())

	return repo.Storer.SetReference(ref)
}

// createNewBranch creates a completely new branch
func (m *Manager) createNewBranch(ctx context.Context, repo *git.Repository, branch, mainRepoPath string) error {
	// Get the default branch to base the new branch on
	defaultBranch, err := m.GetDefaultBranch(ctx, mainRepoPath)
	if err != nil {
		defaultBranch = "main" // Fallback to main
	}

	// Get the reference for the default branch
	defaultBranchRef := plumbing.NewBranchReferenceName(defaultBranch)
	defaultRef, err := repo.Reference(defaultBranchRef, true)
	if err != nil {
		// Try remote reference
		remoteBranchRef := plumbing.NewRemoteReferenceName("origin", defaultBranch)
		defaultRef, err = repo.Reference(remoteBranchRef, true)
		if err != nil {
			return fmt.Errorf("failed to find base branch %s: %w", defaultBranch, err)
		}
	}

	// Create local branch reference pointing to the same commit
	localBranchRef := plumbing.NewBranchReferenceName(branch)
	ref := plumbing.NewHashReference(localBranchRef, defaultRef.Hash())

	return repo.Storer.SetReference(ref)
}

// findMainRepoForWorktree finds the main repository path for a given worktree
func (m *Manager) findMainRepoForWorktree(worktreePath string) (string, error) {
	// Check if there's a .git file pointing to the main repo
	gitPath := filepath.Join(worktreePath, ".git")
	if content, err := os.ReadFile(gitPath); err == nil {
		gitDir := strings.TrimPrefix(strings.TrimSpace(string(content)), "gitdir: ")
		if strings.Contains(gitDir, "worktrees") {
			// Extract main repo path from worktree git dir
			parts := strings.Split(gitDir, "/worktrees/")
			if len(parts) > 0 {
				return parts[0], nil
			}
		}
	}

	return "", fmt.Errorf("unable to find main repository for worktree: %s", worktreePath)
}

// getAuthMethod returns the authentication method for Git operations
func (m *Manager) getAuthMethod() transport.AuthMethod {
	// Try SSH key authentication first
	if sshKey := os.Getenv("SSH_KEY_PATH"); sshKey != "" {
		if auth, err := ssh.NewPublicKeysFromFile("git", sshKey, ""); err == nil {
			return auth
		}
	}

	// Try SSH agent
	if auth, err := ssh.NewSSHAgentAuth("git"); err == nil {
		return auth
	}

	// Try username/password from environment
	if username := os.Getenv("GIT_USERNAME"); username != "" {
		if password := os.Getenv("GIT_PASSWORD"); password != "" {
			return &http.BasicAuth{
				Username: username,
				Password: password,
			}
		}
	}

	// Try GitHub token
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return &http.BasicAuth{
			Username: "token",
			Password: token,
		}
	}

	return nil
}

// GetDefaultBranch returns the default branch for the repository
func (m *Manager) GetDefaultBranch(ctx context.Context, repoPath string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	// Try to get the remote's default branch first
	remotes, err := repo.Remotes()
	if err == nil && len(remotes) > 0 {
		// Try to get origin/HEAD to determine default branch
		originHead := plumbing.NewRemoteReferenceName("origin", "HEAD")
		if ref, err := repo.Reference(originHead, true); err == nil {
			if ref.Target().IsBranch() {
				return ref.Target().Short(), nil
			}
		}
	}

	// For local repositories, check what branches exist and pick the most common default
	branches, err := m.GetBranches(ctx, repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to get branches: %w", err)
	}

	// Check for common default branch names in order of preference
	for _, defaultName := range []string{"main", "master", "develop"} {
		for _, branch := range branches {
			if branch == defaultName {
				return defaultName, nil
			}
		}
	}

	// If no common defaults found, return the first branch
	if len(branches) > 0 {
		return branches[0], nil
	}

	// Fallback
	return "main", nil
}

// IsRepository checks if the path is a valid git repository
func (m *Manager) IsRepository(path string) bool {
	_, err := git.PlainOpen(path)
	return err == nil
}

// InitRepository initializes a new git repository
func (m *Manager) InitRepository(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = path
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to initialize repository: %w, output: %s", err, string(output))
	}
	return nil
}

// AddAndCommit stages all changes and creates a commit
func (m *Manager) AddAndCommit(ctx context.Context, path, message string) error {
	// Stage all changes
	cmd := exec.CommandContext(ctx, "git", "add", "-A")
	cmd.Dir = path
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage changes: %w, output: %s", err, string(output))
	}
	
	// Create commit
	cmd = exec.CommandContext(ctx, "git", "commit", "-m", message)
	cmd.Dir = path
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit: %w, output: %s", err, string(output))
	}
	
	return nil
}

// GetCommitInfo returns commit information for a given path
func (m *Manager) GetCommitInfo(ctx context.Context, path string) (*object.Commit, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	return commit, nil
}

// GetBranches returns all branches in the repository
func (m *Manager) GetBranches(ctx context.Context, repoPath string) ([]string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	branches, err := repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	var result []string
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		result = append(result, ref.Name().Short())
		return nil
	})

	return result, err
}

// HasUncommittedChanges checks if the repository has uncommitted changes
func (m *Manager) HasUncommittedChanges(ctx context.Context, path string) (bool, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return false, fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get status: %w", err)
	}

	return !status.IsClean(), nil
}

// Command-based helper methods for worktree operations

// createWorktreeWithCmd creates a worktree using git command
func (m *Manager) createWorktreeWithCmd(ctx context.Context, path, branch, mainRepoPath string) error {
	// Use os/exec to run git worktree add command
	cmd := exec.CommandContext(ctx, "git", "-C", mainRepoPath, "worktree", "add", path, branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create worktree with git command: %w", err)
	}
	return nil
}

// listWorktreesWithCmd lists worktrees using git command
func (m *Manager) listWorktreesWithCmd(ctx context.Context, repoPath string) ([]containerpkg.GitWorktree, error) {
	// Use os/exec to run git worktree list command
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees with git command: %w", err)
	}

	return m.parseWorktreeList(string(output))
}

// removeWorktreeWithCmd removes a worktree using git command
func (m *Manager) removeWorktreeWithCmd(ctx context.Context, path, mainRepoPath string) error {
	// Use os/exec to run git worktree remove command
	cmd := exec.CommandContext(ctx, "git", "-C", mainRepoPath, "worktree", "remove", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove worktree with git command: %w", err)
	}
	return nil
}

// forceRemoveWorktree forcefully removes a worktree
func (m *Manager) forceRemoveWorktree(ctx context.Context, path, mainRepoPath string) error {
	// Use os/exec to run git worktree remove with force flag
	cmd := exec.CommandContext(ctx, "git", "-C", mainRepoPath, "worktree", "remove", "--force", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// As a last resort, try to remove the directory manually
		if removeErr := os.RemoveAll(path); removeErr != nil {
			return fmt.Errorf("failed to force remove worktree: git error: %w, fs error: %v", err, removeErr)
		}
	}
	return nil
}

// IsWorktree detects if the current directory is a Git worktree
func (m *Manager) IsWorktree(path string) bool {
	gitPath := filepath.Join(path, ".git")

	// Check if .git is a file (worktree) rather than a directory (main repo)
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}

	if info.IsDir() {
		return false // Main repository has .git directory
	}

	// Read .git file to confirm it's a worktree
	content, err := os.ReadFile(gitPath)
	if err != nil {
		return false
	}

	gitDirLine := strings.TrimSpace(string(content))
	return strings.HasPrefix(gitDirLine, "gitdir: ") && strings.Contains(gitDirLine, "worktrees")
}

// GetMainRepoPathFromWorktree finds the main repository path from a worktree
func (m *Manager) GetMainRepoPathFromWorktree(worktreePath string) (string, error) {
	if !m.IsWorktree(worktreePath) {
		return "", fmt.Errorf("path %s is not a worktree", worktreePath)
	}

	gitPath := filepath.Join(worktreePath, ".git")
	content, err := os.ReadFile(gitPath)
	if err != nil {
		return "", fmt.Errorf("failed to read .git file: %w", err)
	}

	gitDirLine := strings.TrimSpace(string(content))
	gitDir := strings.TrimPrefix(gitDirLine, "gitdir: ")

	// Extract main repo path from worktree git dir
	// Format: /path/to/main/repo/.git/worktrees/worktree-name
	if !strings.Contains(gitDir, "worktrees") {
		return "", fmt.Errorf("invalid worktree git directory format: %s", gitDir)
	}

	parts := strings.Split(gitDir, "/worktrees/")
	if len(parts) < 2 {
		return "", fmt.Errorf("unable to parse worktree git directory: %s", gitDir)
	}

	mainRepoGitDir := parts[0]
	mainRepoPath := filepath.Dir(mainRepoGitDir) // Remove .git suffix

	return mainRepoPath, nil
}

// GetEnvironmentFromWorktree infers the environment name from worktree branch/path
func (m *Manager) GetEnvironmentFromWorktree(worktreePath string) (string, error) {
	if !m.IsWorktree(worktreePath) {
		return "", fmt.Errorf("path %s is not a worktree", worktreePath)
	}

	// First try to get environment from current branch
	repo, err := git.PlainOpen(worktreePath)
	if err != nil {
		return "", fmt.Errorf("failed to open worktree repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	if head.Name().IsBranch() {
		branchName := head.Name().Short()

		// If branch is main/master, return "main"
		if branchName == "main" || branchName == "master" {
			return "main", nil
		}

		// For worktree branches, extract environment name
		// Common patterns: feature/auth, feature-auth, feat/auth, etc.
		if strings.HasPrefix(branchName, "feature/") {
			return strings.TrimPrefix(branchName, "feature/"), nil
		}
		if strings.HasPrefix(branchName, "feat/") {
			return strings.TrimPrefix(branchName, "feat/"), nil
		}
		if strings.HasPrefix(branchName, "feature-") {
			return strings.TrimPrefix(branchName, "feature-"), nil
		}
		if strings.HasPrefix(branchName, "feat-") {
			return strings.TrimPrefix(branchName, "feat-"), nil
		}

		// For other branches, use the branch name as environment
		return branchName, nil
	}

	// Fallback to directory name pattern
	dirName := filepath.Base(worktreePath)

	// Try to extract environment from directory name
	// Pattern: project-name-environment
	if strings.Contains(dirName, "-") {
		parts := strings.Split(dirName, "-")
		if len(parts) >= 2 {
			// Return the last part as environment
			return parts[len(parts)-1], nil
		}
	}

	// Final fallback
	return "main", nil
}

// GetRepositoryNameFromWorktree extracts the repository name from a worktree
func (m *Manager) GetRepositoryNameFromWorktree(worktreePath string) (string, error) {
	if !m.IsWorktree(worktreePath) {
		return "", fmt.Errorf("path %s is not a worktree", worktreePath)
	}

	// Get the main repository path
	mainRepoPath, err := m.GetMainRepoPathFromWorktree(worktreePath)
	if err != nil {
		return "", fmt.Errorf("failed to get main repository path: %w", err)
	}

	// Extract repository name from the main repo path
	// Main repo path format: /path/to/.vibeman/repos/.repos/repo-name
	// Or: /path/to/repo-name-worktrees/.repos/repo-name
	repoName := filepath.Base(mainRepoPath)

	// If it's a bare repository, it might have .git suffix
	if strings.HasSuffix(repoName, ".git") {
		repoName = strings.TrimSuffix(repoName, ".git")
	}

	// Return the simplified repository name
	return repoName, nil
}

// GetRepositoryAndEnvironmentFromPath detects both repository and environment from current path
func (m *Manager) GetRepositoryAndEnvironmentFromPath(path string) (repoName string, envName string, err error) {
	// Check if we're in a worktree
	if m.IsWorktree(path) {
		repoName, err = m.GetRepositoryNameFromWorktree(path)
		if err != nil {
			return "", "", fmt.Errorf("failed to get repository name: %w", err)
		}

		envName, err = m.GetEnvironmentFromWorktree(path)
		if err != nil {
			return "", "", fmt.Errorf("failed to get environment name: %w", err)
		}

		return repoName, envName, nil
	}

	// Check if we're in a regular git repository
	if m.IsRepository(path) {
		// Try to extract repository name from remote URL
		repo, err := git.PlainOpen(path)
		if err != nil {
			return "", "", fmt.Errorf("failed to open repository: %w", err)
		}

		remotes, err := repo.Remotes()
		if err != nil || len(remotes) == 0 {
			return "", "", fmt.Errorf("no remotes configured for repository")
		}

		remoteURL := remotes[0].Config().URLs[0]

		// Extract repository name from URL
		parts := strings.Split(remoteURL, "/")
		repoName = parts[len(parts)-1]
		if strings.HasSuffix(repoName, ".git") {
			repoName = repoName[:len(repoName)-4]
		}


		// For main repository, environment is "main"
		return repoName, "main", nil
	}

	return "", "", fmt.Errorf("not in a git repository or worktree")
}

// FindProjectConfig finds vibeman.toml in the main repository when in worktree
func (m *Manager) FindProjectConfig(currentPath string) (string, error) {
	// First check if we're in a worktree
	if m.IsWorktree(currentPath) {
		// Get main repository path
		mainRepoPath, err := m.GetMainRepoPathFromWorktree(currentPath)
		if err != nil {
			return "", fmt.Errorf("failed to get main repository path: %w", err)
		}

		// Check for vibeman.toml in main repository
		configPath := filepath.Join(mainRepoPath, "vibeman.toml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		return "", fmt.Errorf("no vibeman.toml found in main repository: %s", mainRepoPath)
	}

	// Not in a worktree, check current directory
	configPath := filepath.Join(currentPath, "vibeman.toml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	return "", fmt.Errorf("no vibeman.toml found in current directory: %s", currentPath)
}

// parseWorktreeList parses the output of git worktree list --porcelain
func (m *Manager) parseWorktreeList(output string) ([]containerpkg.GitWorktree, error) {
	var worktrees []containerpkg.GitWorktree
	lines := strings.Split(strings.TrimSpace(output), "\n")

	var current containerpkg.GitWorktree
	for _, line := range lines {
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
			}
			current = containerpkg.GitWorktree{}
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "worktree":
			current.Path = value
		case "HEAD":
			current.Commit = value
		case "branch":
			current.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "bare":
			current.IsBare = true
		case "locked":
			current.IsLocked = true
		}
	}

	// Add the last worktree if exists
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	// Set creation time and main flag
	for i := range worktrees {
		if stat, err := os.Stat(worktrees[i].Path); err == nil {
			worktrees[i].CreatedAt = stat.ModTime().Format(time.RFC3339)
		}
		// Main worktree doesn't have a branch in the porcelain output
		if worktrees[i].Branch == "" && !worktrees[i].IsBare {
			worktrees[i].IsMain = true
		}
	}

	return worktrees, nil
}

// HasUnpushedCommits checks if there are unpushed commits in the repository
func (m *Manager) HasUnpushedCommits(ctx context.Context, path string) (bool, error) {
	// Open the repository
	repo, err := git.PlainOpen(path)
	if err != nil {
		return false, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get current branch
	head, err := repo.Head()
	if err != nil {
		return false, fmt.Errorf("failed to get HEAD: %w", err)
	}

	if !head.Name().IsBranch() {
		// Detached HEAD state
		return false, nil
	}

	branchName := head.Name().Short()

	// Check if we have a remote tracking branch
	cmd := exec.CommandContext(ctx, "git", "-C", path, "rev-list", "--count", fmt.Sprintf("origin/%s..HEAD", branchName))
	output, err := cmd.Output()
	if err != nil {
		// If command fails, it might be because there's no remote tracking branch
		// Try alternative approach
		cmd = exec.CommandContext(ctx, "git", "-C", path, "cherry", "-v")
		output, err = cmd.Output()
		if err != nil {
			return false, nil // Assume no unpushed commits if we can't check
		}
		return len(strings.TrimSpace(string(output))) > 0, nil
	}

	count := strings.TrimSpace(string(output))
	return count != "0" && count != "", nil
}

// GetCurrentBranch returns the current branch name
func (m *Manager) GetCurrentBranch(ctx context.Context, path string) (string, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	if !head.Name().IsBranch() {
		return "", fmt.Errorf("repository is in detached HEAD state")
	}

	return head.Name().Short(), nil
}

// IsBranchMerged checks if a branch has been merged into the default branch
func (m *Manager) IsBranchMerged(ctx context.Context, path, branch string) (bool, error) {
	// Get the default branch
	defaultBranch, err := m.GetDefaultBranch(ctx, path)
	if err != nil {
		return false, fmt.Errorf("failed to get default branch: %w", err)
	}

	// Check if branch is merged into default branch
	cmd := exec.CommandContext(ctx, "git", "-C", path, "branch", "--merged", defaultBranch)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check merged branches: %w", err)
	}

	// Parse output to see if our branch is in the list
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if trimmed == branch {
			return true, nil
		}
	}

	return false, nil
}
