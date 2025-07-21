// Package interfaces provides common interface definitions used throughout vibeman.
// This package consolidates all shared interfaces to avoid duplication and ensure consistency.
package interfaces

import (
	"context"
	"io"

	"vibeman/internal/types"
)

// ContainerManager interface for container operations - composed of smaller interfaces
type ContainerManager interface {
	ContainerLifecycle
	ContainerQuery
	ContainerExecution
	ContainerInteraction
	ContainerFileOperations
	ContainerSetup
}

// ContainerLifecycle handles container lifecycle operations
type ContainerLifecycle interface {
	Create(ctx context.Context, repositoryName, environment, image string) (*types.Container, error)
	Start(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string) error
	Remove(ctx context.Context, containerID string) error
}

// ContainerQuery handles container queries
type ContainerQuery interface {
	List(ctx context.Context) ([]*types.Container, error)
	GetByName(ctx context.Context, name string) (*types.Container, error)
	GetByRepository(ctx context.Context, repository string) ([]*types.Container, error)
}

// ContainerExecution handles command execution in containers
type ContainerExecution interface {
	Exec(ctx context.Context, containerID string, command []string) ([]byte, error)
	Logs(ctx context.Context, containerID string, follow bool) ([]byte, error)
}

// ContainerInteraction handles user interaction with containers
type ContainerInteraction interface {
	Shell(ctx context.Context, containerID string, shell string) error
	SSH(ctx context.Context, containerID string, user string) error
	Attach(ctx context.Context, containerID string) error
}

// ContainerFileOperations handles file operations with containers
type ContainerFileOperations interface {
	CopyToContainer(ctx context.Context, containerID, srcPath, dstPath string) error
	CopyFromContainer(ctx context.Context, containerID, srcPath, dstPath string) error
}

// ContainerSetup handles container setup operations
type ContainerSetup interface {
	RunSetup(ctx context.Context, containerID string, repositoryPath string) error
	RunLifecycleHook(ctx context.Context, containerID string, hook string) error
}

// GitManager interface for git operations - composed of smaller interfaces
type GitManager interface {
	GitWorktreeOperations
	GitRepositoryOperations
	GitStatusOperations
	GitBranchOperations
}

// GitWorktreeOperations handles worktree management
type GitWorktreeOperations interface {
	CreateWorktree(ctx context.Context, repoURL, branch, path string) error
	ListWorktrees(ctx context.Context, repoPath string) ([]types.GitWorktree, error)
	RemoveWorktree(ctx context.Context, path string) error
	UpdateWorktree(ctx context.Context, path string) error
}

// GitRepositoryOperations handles repository operations
type GitRepositoryOperations interface {
	CloneRepository(ctx context.Context, repoURL, path string) error
	IsRepository(path string) bool
	GetRepositoryAndEnvironmentFromPath(path string) (repoName string, envName string, err error)
}

// GitStatusOperations handles git status queries
type GitStatusOperations interface {
	HasUncommittedChanges(ctx context.Context, path string) (bool, error)
	HasUnpushedCommits(ctx context.Context, path string) (bool, error)
}

// GitBranchOperations handles branch operations
type GitBranchOperations interface {
	SwitchBranch(ctx context.Context, path, branch string) error
	GetDefaultBranch(ctx context.Context, repoPath string) (string, error)
	GetCurrentBranch(ctx context.Context, path string) (string, error)
	IsBranchMerged(ctx context.Context, path, branch string) (bool, error)
}

// ServiceManager interface for service operations - composed of smaller interfaces
type ServiceManager interface {
	ServiceControl
	ServiceReferences
	ServiceQuery
}

// ServiceControl handles service lifecycle operations
type ServiceControl interface {
	StartService(ctx context.Context, name string) error
	StopService(ctx context.Context, name string) error
	HealthCheck(ctx context.Context, name string) error
}

// ServiceReferences handles service reference management
type ServiceReferences interface {
	AddReference(serviceName, repositoryName string) error
	RemoveReference(serviceName, repositoryName string) error
}

// ServiceQuery handles service queries
type ServiceQuery interface {
	GetService(name string) (interface{}, error)
}

// MinimalGitManager interface with only the essential methods needed by container manager
type MinimalGitManager interface {
	CreateWorktree(ctx context.Context, repoURL, branch, path string) error
	ListWorktrees(ctx context.Context, repoPath string) ([]types.GitWorktree, error)
	RemoveWorktree(ctx context.Context, path string) error
	SwitchBranch(ctx context.Context, path, branch string) error
	UpdateWorktree(ctx context.Context, path string) error
	CloneRepository(ctx context.Context, repoURL, path string) error
	IsRepository(path string) bool
	GetDefaultBranch(ctx context.Context, repoPath string) (string, error)
	HasUncommittedChanges(ctx context.Context, path string) (bool, error)
}

// MinimalServiceManager interface with only the essential methods needed by container manager
type MinimalServiceManager interface {
	StartService(ctx context.Context, name string) error
	StopService(ctx context.Context, name string) error
	AddReference(serviceName, repositoryName string) error
	RemoveReference(serviceName, repositoryName string) error
	HealthCheck(ctx context.Context, name string) error
}

// DatabaseInterface for database operations
type DatabaseInterface interface {
	// Add database methods as needed
	io.Closer
}

// AuthInterface for authentication operations
type AuthInterface interface {
	// Add auth methods as needed
}

// WebSocketManagerInterface for WebSocket operations
type WebSocketManagerInterface interface {
	// Add WebSocket methods as needed
}
