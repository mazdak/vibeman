package operations

import (
	"context"
	
	"vibeman/internal/container"
)

// GitManager defines the interface for git operations used by operations
type GitManager interface {
	// Repository operations
	CloneRepository(ctx context.Context, url, path string) error
	IsRepository(path string) bool
	
	// Worktree operations
	CreateWorktree(ctx context.Context, repoPath, branch, worktreePath string) error
	RemoveWorktree(ctx context.Context, worktreePath string) error
	HasUncommittedChanges(ctx context.Context, path string) (bool, error)
	HasUnpushedCommits(ctx context.Context, path string) (bool, error)
}

// ContainerManager defines the interface for container operations used by operations
type ContainerManager interface {
	// Container operations
	Create(ctx context.Context, repositoryName, environment, image string) (*container.Container, error)
	CreateWithConfig(ctx context.Context, config *container.CreateConfig) (*container.Container, error)
	Start(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string) error
	Remove(ctx context.Context, containerID string) error
	List(ctx context.Context) ([]*container.Container, error)
	GetByName(ctx context.Context, name string) (*container.Container, error)
	Logs(ctx context.Context, containerID string, follow bool) ([]byte, error)
}

// ServiceManager defines the interface for service operations used by operations
type ServiceManager interface {
	StartService(ctx context.Context, name string) error
	StopService(ctx context.Context, name string) error
	GetService(name string) (interface{}, error)
	HealthCheck(ctx context.Context, name string) error
	AddReference(serviceName, repoName string) error
	RemoveReference(serviceName, repoName string) error
}

