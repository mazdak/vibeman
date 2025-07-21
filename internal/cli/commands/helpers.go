package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"vibeman/internal/config"
	"vibeman/internal/container"
	"vibeman/internal/git"
	"vibeman/internal/interfaces"
	"vibeman/internal/types"
)

// RepositoryHelpers provides utilities for working with repository directories
type RepositoryHelpers struct {
	gitManager *git.Manager
}

// FindRepositoryConfig looks for a vibeman.toml file in the current directory or main repository if in worktree
func (h *RepositoryHelpers) FindRepositoryConfig() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize git manager if not already done
	if h.gitManager == nil {
		h.gitManager = git.New(nil)
	}

	// Use git manager's FindRepositoryConfig method which handles worktrees
	configPath, err := h.gitManager.FindProjectConfig(currentDir)
	if err != nil {
		return "", fmt.Errorf("no vibeman.toml found. Please run 'vibeman init' to initialize a repository")
	}

	return configPath, nil
}

// GetRepositoryName extracts the repository name from the config in the current directory
func GetRepositoryName(cfg *config.Manager) (string, error) {
	// Check if we have a valid repository configuration
	if cfg.Repository.Repository.Name == "" {
		return "", fmt.Errorf("no repository configuration found. Please run 'vibeman init' to initialize a repository")
	}

	return cfg.Repository.Repository.Name, nil
}

// ValidateRepositoryDirectory ensures we're in a valid repository directory
func ValidateRepositoryDirectory(cfg *config.Manager) error {
	h := &RepositoryHelpers{}
	if _, err := h.FindRepositoryConfig(); err != nil {
		return err
	}

	if cfg.Repository.Repository.Name == "" {
		return fmt.Errorf("invalid repository configuration. Please check your vibeman.toml file")
	}

	return nil
}

// GetContainerName generates the appropriate container name for a repository and worktree
func (h *RepositoryHelpers) GetContainerName(repositoryName, worktreeName string) string {
	if worktreeName == "main" || worktreeName == "" {
		return repositoryName
	}
	return fmt.Sprintf("%s-%s", repositoryName, worktreeName)
}

// worktreeServiceAdapter adapts ServiceManager to operations.ServiceManager
type worktreeServiceAdapter struct {
	mgr ServiceManager
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

func (a *worktreeServiceAdapter) AddReference(serviceName, repositoryName string) error {
	return a.mgr.AddReference(serviceName, repositoryName)
}

func (a *worktreeServiceAdapter) RemoveReference(serviceName, repositoryName string) error {
	return a.mgr.RemoveReference(serviceName, repositoryName)
}

func (a *worktreeServiceAdapter) HealthCheck(ctx context.Context, name string) error {
	return a.mgr.HealthCheck(ctx, name)
}

// containerManagerAdapter adapts interfaces.ContainerManager to operations.ContainerManager
type containerManagerAdapter struct {
	mgr interfaces.ContainerManager
}

func (a *containerManagerAdapter) Create(ctx context.Context, repositoryName, environment, image string) (*container.Container, error) {
	typesContainer, err := a.mgr.Create(ctx, repositoryName, environment, image)
	if err != nil {
		return nil, err
	}
	return convertTypesContainerToContainer(typesContainer), nil
}

func (a *containerManagerAdapter) CreateWithConfig(ctx context.Context, config *container.CreateConfig) (*container.Container, error) {
	// This is the method that's missing from interfaces.ContainerManager
	// We'll need to use the Create method and build the config ourselves
	// For now, let's create a simple implementation
	typesContainer, err := a.mgr.Create(ctx, config.Repository, config.Environment, config.Image)
	if err != nil {
		return nil, err
	}
	return convertTypesContainerToContainer(typesContainer), nil
}

func (a *containerManagerAdapter) Start(ctx context.Context, containerID string) error {
	return a.mgr.Start(ctx, containerID)
}

func (a *containerManagerAdapter) Stop(ctx context.Context, containerID string) error {
	return a.mgr.Stop(ctx, containerID)
}

func (a *containerManagerAdapter) Remove(ctx context.Context, containerID string) error {
	return a.mgr.Remove(ctx, containerID)
}

func (a *containerManagerAdapter) List(ctx context.Context) ([]*container.Container, error) {
	typesContainers, err := a.mgr.List(ctx)
	if err != nil {
		return nil, err
	}
	
	containers := make([]*container.Container, len(typesContainers))
	for i, tc := range typesContainers {
		containers[i] = convertTypesContainerToContainer(tc)
	}
	return containers, nil
}

func (a *containerManagerAdapter) GetByName(ctx context.Context, name string) (*container.Container, error) {
	typesContainer, err := a.mgr.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return convertTypesContainerToContainer(typesContainer), nil
}

func (a *containerManagerAdapter) Logs(ctx context.Context, containerID string, follow bool) ([]byte, error) {
	return a.mgr.Logs(ctx, containerID, follow)
}

// convertTypesContainerToContainer converts types.Container to container.Container
func convertTypesContainerToContainer(tc *types.Container) *container.Container {
	return &container.Container{
		ID:          tc.ID,
		Name:        tc.Name,
		Image:       tc.Image,
		Status:      tc.Status,
		Repository:  tc.Repository,
		Environment: tc.Environment,
		CreatedAt:   tc.CreatedAt,
		Ports:       tc.Ports,
		Command:     tc.Command,
		EnvVars:     tc.EnvVars,
		Type:        tc.Type,
	}
}

// GetContainerNameWithService generates container name including compose service
func (h *RepositoryHelpers) GetContainerNameWithService(repositoryName, worktreeName, serviceName string) string {
	if serviceName == "" {
		return h.GetContainerName(repositoryName, worktreeName)
	}

	if worktreeName == "main" || worktreeName == "" {
		return fmt.Sprintf("%s-%s", repositoryName, serviceName)
	}
	return fmt.Sprintf("%s-%s-%s", repositoryName, worktreeName, serviceName)
}

// GetContainerNameForService is a static helper for generating container names with service
func GetContainerNameForService(repositoryName, worktreeName, serviceName string) string {
	if serviceName == "" {
		// No service, use traditional naming
		if worktreeName == "main" || worktreeName == "" {
			return repositoryName
		}
		return fmt.Sprintf("%s-%s", repositoryName, worktreeName)
	}

	// With service, include it in the name
	if worktreeName == "main" || worktreeName == "" {
		return fmt.Sprintf("%s-%s", repositoryName, serviceName)
	}
	return fmt.Sprintf("%s-%s-%s", repositoryName, worktreeName, serviceName)
}

// InferWorktree tries to determine the current worktree based on the working directory
func (h *RepositoryHelpers) InferWorktree(repositoryName string) string {
	currentDir, err := os.Getwd()
	if err != nil {
		return "main"
	}

	// Initialize git manager if not already done
	if h.gitManager == nil {
		h.gitManager = git.New(nil)
	}

	// First check if we're in a worktree and can infer from branch/path
	// For now, we'll use directory-based inference until git manager methods are updated

	// Fallback to directory name pattern for non-worktree or failed worktree detection
	dirName := filepath.Base(currentDir)

	// Check if directory name matches repository-worktree pattern
	if len(dirName) > len(repositoryName)+1 && dirName[:len(repositoryName)] == repositoryName && dirName[len(repositoryName)] == '-' {
		return dirName[len(repositoryName)+1:]
	}

	// Default to main worktree
	return "main"
}

// GetCurrentRepositoryAndWorktree returns the current repository name and worktree
func (h *RepositoryHelpers) GetCurrentRepositoryAndWorktree(cfg *config.Manager) (string, string, error) {
	repositoryName, err := GetRepositoryName(cfg)
	if err != nil {
		return "", "", err
	}

	worktreeName := h.InferWorktree(repositoryName)
	return repositoryName, worktreeName, nil
}

// GetCurrentRepositoryAndEnv is deprecated - use GetCurrentRepositoryAndWorktree instead
func (h *RepositoryHelpers) GetCurrentRepositoryAndEnv(cfg *config.Manager) (string, string, error) {
	return h.GetCurrentRepositoryAndWorktree(cfg)
}

// Helper function to create error messages for missing repository configuration
func RepositoryConfigError() error {
	return fmt.Errorf(`no repository configuration found in current directory.

To initialize a new repository:
  vibeman init

To work with an existing repository:
  cd /path/to/your/repository
  vibeman <command>`)
}

// Helper function to create error messages for missing worktree
func (h *RepositoryHelpers) WorktreeNotFoundError(worktreeName string) error {
	return fmt.Errorf(`worktree '%s' not found.

To create a new worktree:
  vibeman worktree create %s

To list available worktrees:
  vibeman worktree list`, worktreeName, worktreeName)
}

// NewRepositoryHelpers creates a new RepositoryHelpers instance
func NewRepositoryHelpers(cfg *config.Manager) *RepositoryHelpers {
	return &RepositoryHelpers{
		gitManager: git.New(cfg),
	}
}

// Global helper instance
var helpers = &RepositoryHelpers{}

// GetCurrentRepositoryAndWorktree is a global helper function
func GetCurrentRepositoryAndWorktree(cfg *config.Manager) (string, string, error) {
	return helpers.GetCurrentRepositoryAndWorktree(cfg)
}

// GetCurrentRepositoryAndEnv is a deprecated global helper function
func GetCurrentRepositoryAndEnv(cfg *config.Manager) (string, string, error) {
	return helpers.GetCurrentRepositoryAndEnv(cfg)
}
