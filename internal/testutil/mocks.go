package testutil

import (
	"context"
	"fmt"
	"sync"

	"vibeman/internal/container"
	"vibeman/internal/types"
	
	"github.com/stretchr/testify/mock"
)

// WorktreeStatus represents the status of a worktree
type WorktreeStatus struct {
	HasUncommittedChanges bool
	HasUntrackedFiles     bool
	HasUnpushedCommits    bool
}

// MockContainerManager is a mock implementation of container.Manager for testing
type MockContainerManager struct {
	mock.Mock
	mu         sync.RWMutex
	containers map[string]*container.Container
	calls      map[string][]interface{}
	errors     map[string]error
	nextID     int
	
	// ListReturn is returned by List when using simple mocking
	ListReturn []*container.Container
	ListError error
	
	// CreateFn is called by Create when set
	CreateFn func(ctx context.Context, config container.CreateConfig) (*container.Container, error)
	// StartFn is called by Start when set
	StartFn func(ctx context.Context, containerID string) error
	// GetByIDFn is called by GetByID when set
	GetByIDFn func(ctx context.Context, id string) (*container.Container, error)
	// RemoveFn is called by Remove when set
	RemoveFn func(ctx context.Context, id string) error
	// StopFn is called by Stop when set
	StopFn func(ctx context.Context, id string) error
	// RestartFn is called by Restart when set  
	RestartFn func(ctx context.Context, id string) error
	// LogsFn is called by Logs when set
	LogsFn func(ctx context.Context, id string, follow bool) ([]byte, error)
	// GetByNameFn is called by GetByName when set
	GetByNameFn func(ctx context.Context, name string) (*container.Container, error)
}

// NewMockContainerManager creates a new mock container manager
func NewMockContainerManager() *MockContainerManager {
	return &MockContainerManager{
		containers: make(map[string]*container.Container),
		calls:      make(map[string][]interface{}),
		errors:     make(map[string]error),
		nextID:     1,
	}
}

// SetError sets an error to be returned for a specific method
func (m *MockContainerManager) SetError(method string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[method] = err
}

// GetCalls returns the calls made to a specific method
func (m *MockContainerManager) GetCalls(method string) []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.calls[method]
}

// recordCall records a method call
func (m *MockContainerManager) recordCall(method string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls[method] = append(m.calls[method], args)
}

// checkError checks if an error should be returned for a method
func (m *MockContainerManager) checkError(method string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errors[method]
}


// Create creates a new container
func (m *MockContainerManager) Create(ctx context.Context, repositoryName, environment, image string) (*container.Container, error) {
	// If CreateFn is set, use it
	if m.CreateFn != nil {
		cfg := container.CreateConfig{
			Name: fmt.Sprintf("vibeman-%s-%s", repositoryName, environment),
			Image: image,
			Repository: repositoryName,
			Environment: environment,
		}
		return m.CreateFn(ctx, cfg)
	}
	
	args := m.Called(ctx, repositoryName, environment, image)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*container.Container), args.Error(1)
}

// CreateWithConfig creates a container with full configuration
func (m *MockContainerManager) CreateWithConfig(ctx context.Context, config *container.CreateConfig) (*container.Container, error) {
	// If CreateFn is set, use it
	if m.CreateFn != nil {
		return m.CreateFn(ctx, *config)
	}
	
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*container.Container), args.Error(1)
}

// StartContainer starts a container
func (m *MockContainerManager) StartContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

// StopContainer stops a container
func (m *MockContainerManager) StopContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

// RemoveContainer removes a container
func (m *MockContainerManager) RemoveContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

// Start starts a container (implementing operations.ContainerManager interface)
func (m *MockContainerManager) Start(ctx context.Context, containerID string) error {
	// If StartFn is set, use it
	if m.StartFn != nil {
		return m.StartFn(ctx, containerID)
	}
	return m.StartContainer(ctx, containerID)
}

// Stop stops a container (implementing operations.ContainerManager interface)
func (m *MockContainerManager) Stop(ctx context.Context, containerID string) error {
	// If StopFn is set, use it
	if m.StopFn != nil {
		return m.StopFn(ctx, containerID)
	}
	return m.StopContainer(ctx, containerID)
}

// Remove removes a container (implementing operations.ContainerManager interface)
func (m *MockContainerManager) Remove(ctx context.Context, containerID string) error {
	// Use function if set
	if m.RemoveFn != nil {
		return m.RemoveFn(ctx, containerID)
	}
	return m.RemoveContainer(ctx, containerID)
}

// List lists containers
func (m *MockContainerManager) List(ctx context.Context) ([]*container.Container, error) {
	// Use simple return values if set
	if m.ListReturn != nil || m.ListError != nil {
		return m.ListReturn, m.ListError
	}
	
	// Otherwise use testify mock
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*container.Container), args.Error(1)
}


// GetByID gets a container by ID
func (m *MockContainerManager) GetByID(ctx context.Context, id string) (*container.Container, error) {
	// Use function if set
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*container.Container), args.Error(1)
}

// GetByName gets a container by name
func (m *MockContainerManager) GetByName(ctx context.Context, name string) (*container.Container, error) {
	// Use function if set
	if m.GetByNameFn != nil {
		return m.GetByNameFn(ctx, name)
	}

	// Check if we're using testify/mock
	if m.ExpectedCalls != nil {
		args := m.Called(ctx, name)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*container.Container), args.Error(1)
	}
	
	// Simple mocking mode
	if m.calls != nil {
		m.recordCall("GetByName", name)
	}

	if m.errors != nil {
		if err := m.checkError("GetByName"); err != nil {
			return nil, err
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.containers != nil {
		for _, c := range m.containers {
			if c.Name == name {
				return c, nil
			}
		}
	}

	return nil, fmt.Errorf("container not found: %s", name)
}


// GetByRepository gets containers by repository
func (m *MockContainerManager) GetByRepository(ctx context.Context, repository string) ([]*container.Container, error) {
	m.recordCall("GetByRepository", repository)

	if err := m.checkError("GetByRepository"); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	containers := make([]*container.Container, 0)
	for _, c := range m.containers {
		if c.Repository == repository {
			containers = append(containers, c)
		}
	}

	return containers, nil
}

// Exec executes a command in a container
func (m *MockContainerManager) Exec(ctx context.Context, containerID string, command []string) ([]byte, error) {
	m.recordCall("Exec", containerID, command)

	if err := m.checkError("Exec"); err != nil {
		return nil, err
	}

	return []byte("mock output"), nil
}

// Logs gets container logs (implementing operations.ContainerManager interface)
func (m *MockContainerManager) Logs(ctx context.Context, containerID string, follow bool) ([]byte, error) {
	// Use function if set
	if m.LogsFn != nil {
		return m.LogsFn(ctx, containerID, follow)
	}
	
	// Check if we're using testify/mock
	if m.ExpectedCalls != nil {
		args := m.Called(ctx, containerID, follow)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).([]byte), args.Error(1)
	}
	
	m.recordCall("Logs", containerID, follow)

	if err := m.checkError("Logs"); err != nil {
		return nil, err
	}

	return []byte("mock logs"), nil
}

// Shell opens a shell in a container
func (m *MockContainerManager) Shell(ctx context.Context, containerID string, shell string) error {
	m.recordCall("Shell", containerID, shell)

	return m.checkError("Shell")
}

// SSH opens an SSH connection to a container
func (m *MockContainerManager) SSH(ctx context.Context, containerID string, user string) error {
	m.recordCall("SSH", containerID, user)

	return m.checkError("SSH")
}

// Attach attaches to a container
func (m *MockContainerManager) Attach(ctx context.Context, containerID string) error {
	m.recordCall("Attach", containerID)

	return m.checkError("Attach")
}

// CopyToContainer copies files to a container
func (m *MockContainerManager) CopyToContainer(ctx context.Context, containerID, srcPath, dstPath string) error {
	m.recordCall("CopyToContainer", containerID, srcPath, dstPath)

	return m.checkError("CopyToContainer")
}

// CopyFromContainer copies files from a container
func (m *MockContainerManager) CopyFromContainer(ctx context.Context, containerID, srcPath, dstPath string) error {
	m.recordCall("CopyFromContainer", containerID, srcPath, dstPath)

	return m.checkError("CopyFromContainer")
}

// RunSetup runs setup commands in a container
func (m *MockContainerManager) RunSetup(ctx context.Context, containerID string, projectPath string) error {
	m.recordCall("RunSetup", containerID, projectPath)

	return m.checkError("RunSetup")
}

// RunLifecycleHook runs a lifecycle hook in a container
func (m *MockContainerManager) RunLifecycleHook(ctx context.Context, containerID string, hook string) error {
	m.recordCall("RunLifecycleHook", containerID, hook)

	return m.checkError("RunLifecycleHook")
}

// MockServiceManager is a mock implementation of service.Manager for testing
type MockServiceManager struct {
	mock.Mock
	mu       sync.RWMutex
	services map[string]*types.ServiceInstance
	calls    map[string][]interface{}
	errors   map[string]error
	
	// GetServiceFn is called by GetService when set
	GetServiceFn func(ctx context.Context, name string) (*types.ServiceInstance, error)
}

// NewMockServiceManager creates a new mock service manager
func NewMockServiceManager() *MockServiceManager {
	return &MockServiceManager{
		services: make(map[string]*types.ServiceInstance),
		calls:    make(map[string][]interface{}),
		errors:   make(map[string]error),
	}
}

// SetError sets an error to be returned for a specific method
func (m *MockServiceManager) SetError(method string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[method] = err
}

// recordCall records a method call
func (m *MockServiceManager) recordCall(method string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls[method] = append(m.calls[method], args)
}

// checkError checks if an error should be returned for a method
func (m *MockServiceManager) checkError(method string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errors[method]
}

// StartService starts a service
func (m *MockServiceManager) StartService(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

// StopService stops a service
func (m *MockServiceManager) StopService(ctx context.Context, name string) error {
	m.recordCall("StopService", name)
	return m.checkError("StopService")
}


// HealthCheck checks service health
func (m *MockServiceManager) HealthCheck(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

// AddReference adds a reference to a service
func (m *MockServiceManager) AddReference(serviceName, repositoryName string) error {
	m.recordCall("AddReference", serviceName, repositoryName)
	return m.checkError("AddReference")
}

// RemoveReference removes a reference from a service
func (m *MockServiceManager) RemoveReference(serviceName, repositoryName string) error {
	m.recordCall("RemoveReference", serviceName, repositoryName)
	return m.checkError("RemoveReference")
}


// GetService gets a service by name
func (m *MockServiceManager) GetService(name string) (interface{}, error) {
	// If GetServiceFn is set, use it (converting context parameter)
	if m.GetServiceFn != nil {
		return m.GetServiceFn(context.Background(), name)
	}

	if m.calls != nil {
		m.recordCall("GetService", name)
	}

	if err := m.checkError("GetService"); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	service, exists := m.services[name]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", name)
	}

	return service, nil
}

// MockGitManager is a mock implementation of GitManager for testing
type MockGitManager struct {
	mock.Mock
	mu        sync.RWMutex
	calls     map[string][]interface{}
	errors    map[string]error
	worktrees map[string][]types.GitWorktree
}

// NewMockGitManager creates a new mock git manager
func NewMockGitManager() *MockGitManager {
	return &MockGitManager{
		calls:     make(map[string][]interface{}),
		errors:    make(map[string]error),
		worktrees: make(map[string][]types.GitWorktree),
	}
}

// SetError sets an error to be returned for a specific method
func (m *MockGitManager) SetError(method string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[method] = err
}

// recordCall records a method call
func (m *MockGitManager) recordCall(method string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls[method] = append(m.calls[method], args)
}

// checkError checks if an error should be returned for a method
func (m *MockGitManager) checkError(method string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errors[method]
}

// CreateWorktree creates a worktree
func (m *MockGitManager) CreateWorktree(ctx context.Context, repoPath, branch, worktreePath string) error {
	args := m.Called(ctx, repoPath, branch, worktreePath)
	return args.Error(0)
}

// ListWorktrees lists worktrees
func (m *MockGitManager) ListWorktrees(ctx context.Context, repoPath string) ([]types.GitWorktree, error) {
	m.recordCall("ListWorktrees", repoPath)

	if err := m.checkError("ListWorktrees"); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if worktrees, exists := m.worktrees[repoPath]; exists {
		return worktrees, nil
	}

	return []types.GitWorktree{}, nil
}

// RemoveWorktree removes a worktree
func (m *MockGitManager) RemoveWorktree(ctx context.Context, worktreePath string) error {
	args := m.Called(ctx, worktreePath)
	return args.Error(0)
}


// SwitchBranch switches branch in a worktree
func (m *MockGitManager) SwitchBranch(ctx context.Context, path, branch string) error {
	m.recordCall("SwitchBranch", path, branch)
	return m.checkError("SwitchBranch")
}

// UpdateWorktree updates a worktree
func (m *MockGitManager) UpdateWorktree(ctx context.Context, path string) error {
	m.recordCall("UpdateWorktree", path)
	return m.checkError("UpdateWorktree")
}

// CloneRepository clones a repository
func (m *MockGitManager) CloneRepository(ctx context.Context, repoURL, path string) error {
	m.recordCall("CloneRepository", repoURL, path)
	return m.checkError("CloneRepository")
}

// IsRepository checks if path is a repository
func (m *MockGitManager) IsRepository(path string) bool {
	m.recordCall("IsRepository", path)
	// Return true by default for testing
	return true
}

// GetDefaultBranch gets the default branch
func (m *MockGitManager) GetDefaultBranch(ctx context.Context, repoPath string) (string, error) {
	m.recordCall("GetDefaultBranch", repoPath)

	if err := m.checkError("GetDefaultBranch"); err != nil {
		return "", err
	}

	return "main", nil
}

// HasUncommittedChanges checks for uncommitted changes
func (m *MockGitManager) HasUncommittedChanges(ctx context.Context, path string) (bool, error) {
	args := m.Called(ctx, path)
	return args.Bool(0), args.Error(1)
}

// HasUnpushedCommits checks for unpushed commits
func (m *MockGitManager) HasUnpushedCommits(ctx context.Context, path string) (bool, error) {
	args := m.Called(ctx, path)
	return args.Bool(0), args.Error(1)
}

// GetCurrentBranch gets the current branch
func (m *MockGitManager) GetCurrentBranch(ctx context.Context, path string) (string, error) {
	m.recordCall("GetCurrentBranch", path)

	if err := m.checkError("GetCurrentBranch"); err != nil {
		return "", err
	}

	return "main", nil
}

// IsBranchMerged checks if a branch is merged
func (m *MockGitManager) IsBranchMerged(ctx context.Context, path, branch string) (bool, error) {
	m.recordCall("IsBranchMerged", path, branch)

	if err := m.checkError("IsBranchMerged"); err != nil {
		return false, err
	}

	return true, nil
}

// GetRepositoryAndEnvironmentFromPath gets repository and environment from path
func (m *MockGitManager) GetRepositoryAndEnvironmentFromPath(path string) (repoName string, envName string, err error) {
	m.recordCall("GetRepositoryAndEnvironmentFromPath", path)

	if err := m.checkError("GetRepositoryAndEnvironmentFromPath"); err != nil {
		return "", "", err
	}

	return "test-repo", "test-env", nil
}

// SetWorktrees sets worktrees for a repo path (for testing)
func (m *MockGitManager) SetWorktrees(repoPath string, worktrees []types.GitWorktree) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.worktrees[repoPath] = worktrees
}
