package client

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
	"vibeman/internal/container"
	"vibeman/internal/types"
)

// ContainerManagerAdapter adapts the Client to implement cli.ContainerManager
type ContainerManagerAdapter struct {
	client *Client
}

// NewContainerManager creates a new container manager adapter
func NewContainerManager(client *Client) *ContainerManagerAdapter {
	return &ContainerManagerAdapter{client: client}
}

// Create creates a new container
func (a *ContainerManagerAdapter) Create(ctx context.Context, repositoryName, environment, image string) (*container.Container, error) {
	return a.client.CreateContainer(ctx, repositoryName, environment, image)
}

// Start starts a container
func (a *ContainerManagerAdapter) Start(ctx context.Context, containerID string) error {
	return a.client.StartContainer(ctx, containerID)
}

// Stop stops a container
func (a *ContainerManagerAdapter) Stop(ctx context.Context, containerID string) error {
	return a.client.StopContainer(ctx, containerID)
}

// Remove removes a container
func (a *ContainerManagerAdapter) Remove(ctx context.Context, containerID string) error {
	return a.client.RemoveContainer(ctx, containerID)
}

// List lists all containers
func (a *ContainerManagerAdapter) List(ctx context.Context) ([]*container.Container, error) {
	return a.client.ListContainers(ctx)
}

// GetByName gets a container by name
func (a *ContainerManagerAdapter) GetByName(ctx context.Context, name string) (*container.Container, error) {
	return a.client.GetContainerByName(ctx, name)
}

// GetByRepository gets containers by repository
func (a *ContainerManagerAdapter) GetByRepository(ctx context.Context, repository string) ([]*container.Container, error) {
	return a.client.GetContainersByRepository(ctx, repository)
}

// Exec executes a command in a container
func (a *ContainerManagerAdapter) Exec(ctx context.Context, containerID string, command []string) ([]byte, error) {
	return a.client.ExecContainer(ctx, containerID, command)
}

// Logs retrieves container logs
func (a *ContainerManagerAdapter) Logs(ctx context.Context, containerID string, follow bool) ([]byte, error) {
	return a.client.ContainerLogs(ctx, containerID, follow)
}

// Shell opens an interactive shell in a container
func (a *ContainerManagerAdapter) Shell(ctx context.Context, containerID string, shell string) error {
	conn, err := a.client.ContainerShell(ctx, containerID, shell)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Handle WebSocket shell interaction
	return handleWebSocketShell(conn)
}

// SSH opens an SSH connection to a container
func (a *ContainerManagerAdapter) SSH(ctx context.Context, containerID string, user string) error {
	// Use the shell endpoint with user parameter
	conn, err := a.client.ContainerShell(ctx, containerID, "bash")
	if err != nil {
		return err
	}
	defer conn.Close()

	// Handle WebSocket shell interaction
	return handleWebSocketShell(conn)
}

// Attach attaches to a container
func (a *ContainerManagerAdapter) Attach(ctx context.Context, containerID string) error {
	conn, err := a.client.ContainerAttach(ctx, containerID)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Handle WebSocket attach interaction
	return handleWebSocketShell(conn)
}

// CopyToContainer copies files to a container
func (a *ContainerManagerAdapter) CopyToContainer(ctx context.Context, containerID, srcPath, dstPath string) error {
	return a.client.CopyToContainer(ctx, containerID, srcPath, dstPath)
}

// CopyFromContainer copies files from a container
func (a *ContainerManagerAdapter) CopyFromContainer(ctx context.Context, containerID, srcPath, dstPath string) error {
	return a.client.CopyFromContainer(ctx, containerID, srcPath, dstPath)
}

// RunSetup runs setup in a container
func (a *ContainerManagerAdapter) RunSetup(ctx context.Context, containerID string, projectPath string) error {
	return a.client.RunContainerSetup(ctx, containerID, projectPath)
}

// RunLifecycleHook runs a lifecycle hook in a container
func (a *ContainerManagerAdapter) RunLifecycleHook(ctx context.Context, containerID string, hook string) error {
	return a.client.RunContainerLifecycleHook(ctx, containerID, hook)
}

// GitManagerAdapter adapts the Client to implement cli.GitManager
type GitManagerAdapter struct {
	client *Client
}

// NewGitManager creates a new git manager adapter
func NewGitManager(client *Client) *GitManagerAdapter {
	return &GitManagerAdapter{client: client}
}

// CreateWorktree creates a new git worktree
func (a *GitManagerAdapter) CreateWorktree(ctx context.Context, repoURL, branch, path string) error {
	return a.client.CreateWorktree(ctx, repoURL, branch, path)
}

// ListWorktrees lists git worktrees
func (a *GitManagerAdapter) ListWorktrees(ctx context.Context, repoPath string) ([]container.GitWorktree, error) {
	return a.client.ListWorktrees(ctx, repoPath)
}

// RemoveWorktree removes a git worktree
func (a *GitManagerAdapter) RemoveWorktree(ctx context.Context, path string) error {
	return a.client.RemoveWorktree(ctx, path)
}

// SwitchBranch switches branch in a worktree
func (a *GitManagerAdapter) SwitchBranch(ctx context.Context, path, branch string) error {
	return a.client.SwitchBranch(ctx, path, branch)
}

// UpdateWorktree updates a git worktree
func (a *GitManagerAdapter) UpdateWorktree(ctx context.Context, path string) error {
	return a.client.UpdateWorktree(ctx, path)
}

// CloneRepository clones a git repository
func (a *GitManagerAdapter) CloneRepository(ctx context.Context, repoURL, path string) error {
	return a.client.CloneRepository(ctx, repoURL, path)
}

// IsRepository checks if a path is a git repository
func (a *GitManagerAdapter) IsRepository(path string) bool {
	// For client mode, we need to make this async but the interface is sync
	// This is a design limitation that should be addressed in the interface
	result, _ := a.client.IsRepository(context.Background(), path)
	return result
}

// GetDefaultBranch gets the default branch of a repository
func (a *GitManagerAdapter) GetDefaultBranch(ctx context.Context, repoPath string) (string, error) {
	return a.client.GetDefaultBranch(ctx, repoPath)
}

// HasUncommittedChanges checks if a worktree has uncommitted changes
func (a *GitManagerAdapter) HasUncommittedChanges(ctx context.Context, path string) (bool, error) {
	return a.client.HasUncommittedChanges(ctx, path)
}

// HasUnpushedCommits checks if there are unpushed commits in the repository
func (a *GitManagerAdapter) HasUnpushedCommits(ctx context.Context, path string) (bool, error) {
	// For client mode, this would need a server-side implementation
	// For now, return false indicating no unpushed commits
	return false, nil
}

// GetCurrentBranch returns the current branch name
func (a *GitManagerAdapter) GetCurrentBranch(ctx context.Context, path string) (string, error) {
	// For client mode, this would need a server-side implementation
	// For now, return an error indicating this is not supported in client mode
	return "", fmt.Errorf("getting current branch is not supported in client mode")
}

// IsBranchMerged checks if a branch has been merged into the default branch
func (a *GitManagerAdapter) IsBranchMerged(ctx context.Context, path, branch string) (bool, error) {
	// For client mode, this would need a server-side implementation
	// For now, return false indicating branch is not merged
	return false, nil
}

// GetRepositoryAndEnvironmentFromPath detects repository and environment from path
func (a *GitManagerAdapter) GetRepositoryAndEnvironmentFromPath(path string) (repoName string, envName string, err error) {
	// For client mode, this would need a server-side implementation
	// For now, return an error indicating this is not supported in client mode
	return "", "", fmt.Errorf("repository detection from path is not supported in client mode")
}

// ServiceManagerAdapter adapts the Client to implement cli.ServiceManager
type ServiceManagerAdapter struct {
	client *Client
}

// NewServiceManager creates a new service manager adapter
func NewServiceManager(client *Client) *ServiceManagerAdapter {
	return &ServiceManagerAdapter{client: client}
}

// StartService starts a service
func (a *ServiceManagerAdapter) StartService(ctx context.Context, name string) error {
	return a.client.StartService(ctx, name)
}

// StopService stops a service
func (a *ServiceManagerAdapter) StopService(ctx context.Context, name string) error {
	return a.client.StopService(ctx, name)
}

// AddReference adds a repository reference to a service
func (a *ServiceManagerAdapter) AddReference(serviceName, repositoryName string) error {
	return a.client.AddServiceReference(context.Background(), serviceName, repositoryName)
}

// RemoveReference removes a repository reference from a service
func (a *ServiceManagerAdapter) RemoveReference(serviceName, repositoryName string) error {
	return a.client.RemoveServiceReference(context.Background(), serviceName, repositoryName)
}

// HealthCheck performs a health check on a service
func (a *ServiceManagerAdapter) HealthCheck(ctx context.Context, name string) error {
	return a.client.ServiceHealthCheck(ctx, name)
}

// GetService retrieves service information
func (a *ServiceManagerAdapter) GetService(name string) (*types.ServiceInstance, error) {
	return a.client.GetService(context.Background(), name)
}

// handleWebSocketShell handles WebSocket shell interaction
func handleWebSocketShell(conn *websocket.Conn) error {
	// This is a simplified implementation
	// In a real implementation, you would handle:
	// - Terminal I/O
	// - Signal handling
	// - Window resizing
	// - Error handling

	// For now, return an error indicating this needs implementation
	return fmt.Errorf("WebSocket shell interaction not yet implemented")
}
