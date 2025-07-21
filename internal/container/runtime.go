package container

import (
	"context"
	"fmt"
)

// RuntimeType represents the type of container runtime
type RuntimeType string

const (
	// RuntimeTypeDocker represents Docker runtime
	RuntimeTypeDocker RuntimeType = "docker"
)

// ContainerRuntime defines the interface for container operations
// It abstracts the underlying container runtime
type ContainerRuntime interface {
	// List returns all containers with their status
	List(ctx context.Context) ([]*Container, error)

	// Create creates a new container with the specified configuration
	Create(ctx context.Context, config *CreateConfig) (*Container, error)

	// Start starts a container by ID
	Start(ctx context.Context, containerID string) error

	// Stop stops a container by ID
	Stop(ctx context.Context, containerID string) error

	// Remove removes a container by ID
	Remove(ctx context.Context, containerID string) error

	// Exec executes a command in a container
	Exec(ctx context.Context, containerID string, command []string) ([]byte, error)

	// Logs returns logs from a container
	Logs(ctx context.Context, containerID string, follow bool) ([]byte, error)

	// GetInfo returns detailed information about a container
	GetInfo(ctx context.Context, containerID string) (*Container, error)

	// IsAvailable checks if the runtime is available on the system
	IsAvailable(ctx context.Context) bool

	// GetType returns the runtime type
	GetType() RuntimeType
}

// CreateConfig holds configuration for creating a container
type CreateConfig struct {
	Name        string
	Image       string
	WorkingDir  string
	Repository  string
	Environment string
	Type        string   // Container type: "worktree", "service", "ai"
	EnvVars     []string // Environment variables in KEY=VALUE format
	Volumes     []string // Volume mounts in HOST:CONTAINER format
	Ports       []string // Port mappings in HOST:CONTAINER format
	Interactive bool     // Run container with -it flags

	// Docker Compose support
	ComposeFile     string   // Path to docker-compose.yaml
	ComposeService  string   // Service name from compose file (deprecated, use ComposeServices)
	ComposeServices []string // Services to start from compose file (empty = all)
}

// RuntimeFactory provides a convenient way to create Docker runtime
type RuntimeFactory struct {
	executor CommandExecutor
}

// NewRuntimeFactory creates a new runtime factory
func NewRuntimeFactory(executor CommandExecutor) *RuntimeFactory {
	if executor == nil {
		executor = &DefaultCommandExecutor{}
	}
	return &RuntimeFactory{
		executor: executor,
	}
}

// CreateForType creates a runtime (only Docker supported)
func (f *RuntimeFactory) CreateForType(ctx context.Context, runtimeType RuntimeType) (ContainerRuntime, error) {
	if runtimeType != RuntimeTypeDocker {
		return nil, fmt.Errorf("unsupported runtime type: %s (only 'docker' is supported)", runtimeType)
	}

	runtime := NewDockerRuntime(f.executor)
	if !runtime.IsAvailable(ctx) {
		return nil, fmt.Errorf("Docker runtime is not available")
	}
	return runtime, nil
}
