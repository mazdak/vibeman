package container

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"vibeman/internal/config"
	"vibeman/internal/interfaces"
	"vibeman/internal/logger"
	"vibeman/internal/types"
)

// CommandExecutor interface for executing commands (allows mocking in tests)
type CommandExecutor interface {
	CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd
}

// DefaultCommandExecutor implements CommandExecutor using standard exec
type DefaultCommandExecutor struct{}

func (e *DefaultCommandExecutor) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

// Manager handles container lifecycle operations
type Manager struct {
	config       *config.Manager
	git          interfaces.MinimalGitManager
	service      interfaces.MinimalServiceManager
	runtime      ContainerRuntime
	runtimeMutex sync.RWMutex
	factory      *RuntimeFactory
	executor     CommandExecutor // Hold executor for cleanup
}

// Type aliases for backward compatibility
type GitWorktree = types.GitWorktree
type Container = types.Container

// Use minimal interfaces from central interfaces package
type GitManager = interfaces.MinimalGitManager
type ServiceManager = interfaces.MinimalServiceManager

// New creates a new container manager
func New(cfg *config.Manager) *Manager {
	// Create executor based on configuration
	executor := createExecutor(cfg)

	return &Manager{
		config:   cfg,
		factory:  NewRuntimeFactory(executor),
		executor: executor,
	}
}

// createExecutor creates a command executor based on configuration
func createExecutor(cfg *config.Manager) CommandExecutor {
	// Note: Pool configuration removed in simplified approach
	// Always use default executor for now
	return &DefaultCommandExecutor{}
}

// SetManagers sets the git and service managers
func (m *Manager) SetManagers(git GitManager, service ServiceManager) {
	m.git = git
	m.service = service
}

// SetRuntime sets a custom container runtime (for testing)
func (m *Manager) SetRuntime(runtime ContainerRuntime) {
	m.runtimeMutex.Lock()
	defer m.runtimeMutex.Unlock()
	m.runtime = runtime
}

// SetRuntimeFactory sets a custom runtime factory (for testing)
func (m *Manager) SetRuntimeFactory(factory *RuntimeFactory) {
	m.factory = factory
}

// Close cleans up resources used by the manager
func (m *Manager) Close() error {
	// Clean up pooled executor if it exists
	if pooledExec, ok := m.executor.(*PooledCommandExecutor); ok {
		return pooledExec.Close()
	}
	return nil
}

// CreateWithConfig creates a new container with the given configuration
func (m *Manager) CreateWithConfig(ctx context.Context, config *CreateConfig) (*Container, error) {
	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container runtime: %w", err)
	}

	return runtime.Create(ctx, config)
}

// List returns all containers
func (m *Manager) List(ctx context.Context) ([]*Container, error) {
	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container runtime: %w", err)
	}

	return runtime.List(ctx)
}

// Create creates a new container
func (m *Manager) Create(ctx context.Context, repositoryName, environment, image string) (*Container, error) {
	if repositoryName == "" {
		return nil, fmt.Errorf("repository name is required")
	}

	// Build container name (compose will create multiple containers with service suffixes)
	containerName := ""
	if environment != "" {
		containerName = fmt.Sprintf("%s-%s", repositoryName, environment)
	} else {
		containerName = repositoryName
	}

	// Start required services if configured
	if m.service != nil && len(m.config.Repository.Repository.Services) > 0 {
		for serviceName, serviceReq := range m.config.Repository.Repository.Services {
			if serviceReq.Required {
				if err := m.service.StartService(ctx, serviceName); err != nil {
					return nil, fmt.Errorf("failed to start required service %s: %w", serviceName, err)
				}
				if err := m.service.AddReference(serviceName, repositoryName); err != nil {
					return nil, fmt.Errorf("failed to add reference to service %s: %w", serviceName, err)
				}
			}
		}
	}

	// Get runtime and create container
	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container runtime: %w", err)
	}

	// Add VIBEMAN_ENV and VIBEMAN_REPOSITORY environment variables
	// (other environment variables are handled by docker-compose)
	envName := environment
	if envName == "" {
		envName = "main"
	}
	validatedEnvVars := []string{
		fmt.Sprintf("VIBEMAN_ENV=%s", envName),
		fmt.Sprintf("VIBEMAN_REPOSITORY=%s", repositoryName),
	}

	// Note: Ports, volumes, working directory etc. are handled by docker-compose

	config := &CreateConfig{
		Name:        containerName,
		Image:       image,
		Repository:  repositoryName,
		Environment: environment,
		EnvVars:     validatedEnvVars,
		// Docker Compose configuration (simplified approach)
		ComposeFile:     m.config.Repository.Repository.Container.ComposeFile,
		ComposeServices: m.config.Repository.Repository.Container.Services,
		// Note: WorkingDir, Volumes, Ports are handled by compose file
	}

	// Create the container
	container, err := runtime.Create(ctx, config)
	if err != nil {
		return nil, err
	}

	// Run setup commands (simplified approach - no lifecycle hooks or setup scripts)
	if len(m.config.Repository.Repository.Container.Setup) > 0 {
		logger.WithFields(logger.Fields{
			"container": container.Name,
			"operation": "setup",
		}).Info("Running container setup")
		// Get repository directory for setup script path resolution
		projectDir, _ := os.Getwd()
		if err := m.RunSetup(ctx, container.ID, projectDir); err != nil {
			// Log error but don't fail container creation
			logger.WithFields(logger.Fields{
				"container": container.ID,
				"error":     err,
				"operation": "setup",
			}).Warn("Failed to run container setup")
		}
	}

	return container, nil
}

// Start starts a container
func (m *Manager) Start(ctx context.Context, containerID string) error {
	// Validate container ID
	if err := validateContainerID(containerID); err != nil {
		return fmt.Errorf("invalid container ID: %w", err)
	}

	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container runtime: %w", err)
	}

	// Start the container
	if err := runtime.Start(ctx, containerID); err != nil {
		return err
	}

	// Note: Start lifecycle hooks removed in simplified approach

	return nil
}

// Stop stops a container
func (m *Manager) Stop(ctx context.Context, containerID string) error {
	// Validate container ID
	if err := validateContainerID(containerID); err != nil {
		return fmt.Errorf("invalid container ID: %w", err)
	}

	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container runtime: %w", err)
	}

	return runtime.Stop(ctx, containerID)
}

// Remove removes a container
func (m *Manager) Remove(ctx context.Context, containerID string) error {
	// Validate container ID
	if err := validateContainerID(containerID); err != nil {
		return fmt.Errorf("invalid container ID: %w", err)
	}

	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container runtime: %w", err)
	}

	// Get container info to determine repository name
	containers, err := m.List(ctx)
	if err == nil {
		for _, container := range containers {
			if container.ID == containerID {
				// Remove service references if service manager is available
				if m.service != nil && len(m.config.Repository.Repository.Services) > 0 {
					for serviceName := range m.config.Repository.Repository.Services {
						if err := m.service.RemoveReference(serviceName, container.Repository); err != nil {
							// Log error but don't fail container removal
							logger.WithFields(logger.Fields{
								"service":   serviceName,
								"project":   container.Repository,
								"error":     err,
								"operation": "remove_service_reference",
							}).Warn("Failed to remove service reference")
						}
					}
				}
				break
			}
		}
	}

	return runtime.Remove(ctx, containerID)
}

// Exec executes a command in a container
func (m *Manager) Exec(ctx context.Context, containerID string, command []string) ([]byte, error) {
	// Validate container ID
	if err := validateContainerID(containerID); err != nil {
		return nil, fmt.Errorf("invalid container ID: %w", err)
	}

	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container runtime: %w", err)
	}

	return runtime.Exec(ctx, containerID, command)
}

// Logs returns logs from a container
func (m *Manager) Logs(ctx context.Context, containerID string, follow bool) ([]byte, error) {
	// Validate container ID
	if err := validateContainerID(containerID); err != nil {
		return nil, fmt.Errorf("invalid container ID: %w", err)
	}

	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container runtime: %w", err)
	}

	return runtime.Logs(ctx, containerID, follow)
}

// GetByName returns a container by name
func (m *Manager) GetByName(ctx context.Context, name string) (*Container, error) {
	containers, err := m.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		if container.Name == name {
			return container, nil
		}
	}

	return nil, fmt.Errorf("container not found: %s", name)
}

// GetByRepository returns containers for a repository
func (m *Manager) GetByRepository(ctx context.Context, repositoryName string) ([]*Container, error) {
	containers, err := m.List(ctx)
	if err != nil {
		return nil, err
	}

	repositoryContainers := make([]*Container, 0)
	for _, container := range containers {
		if container.Repository == repositoryName {
			repositoryContainers = append(repositoryContainers, container)
		}
	}

	return repositoryContainers, nil
}

// SSH opens an SSH connection to a container
// This is a convenience method that uses the container runtime's exec functionality
// to provide SSH-like access without requiring an actual SSH server
func (m *Manager) SSH(ctx context.Context, containerID string, user string) error {
	// Validate container ID
	if err := validateContainerID(containerID); err != nil {
		return fmt.Errorf("invalid container ID: %w", err)
	}

	// Default to root user if not specified
	if user == "" {
		user = "root"
	}

	// Get runtime
	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container runtime: %w", err)
	}

	// Ensure container is running
	containers, err := m.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}

	var isRunning bool
	for _, c := range containers {
		if c.ID == containerID {
			status := strings.ToLower(c.Status)
			if strings.Contains(status, "running") || strings.Contains(status, "up") {
				isRunning = true
			}
			break
		}
	}

	if !isRunning {
		return fmt.Errorf("container %s is not running", containerID[:12])
	}

	// Use bash as the default shell for SSH-like experience
	shell := "bash"

	// Check for bash availability
	bashCheck := []string{"which", "bash"}
	if _, err := runtime.Exec(ctx, containerID, bashCheck); err != nil {
		shell = "sh"
	}

	logger.WithFields(logger.Fields{
		"container_id": containerID[:12],
		"user":         user,
		"operation":    "exec",
	}).Info("Connecting to container")

	// Execute shell with user context
	var cmd *exec.Cmd
	switch runtime.GetType() {
	case RuntimeTypeDocker:
		args := []string{"exec", "-it"}
		if user != "root" {
			args = append(args, "-u", user)
		}
		args = append(args, containerID, shell)
		cmd = m.factory.executor.CommandContext(ctx, "docker", args...)
	default:
		return fmt.Errorf("SSH not supported for runtime type: %s", runtime.GetType())
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Shell opens an interactive shell in a container
func (m *Manager) Shell(ctx context.Context, containerID string, shell string) error {
	// Validate container ID
	if err := validateContainerID(containerID); err != nil {
		return fmt.Errorf("invalid container ID: %w", err)
	}

	// Validate shell command
	if err := validateShellCommand(shell); err != nil {
		return fmt.Errorf("invalid shell command: %w", err)
	}

	// Default to bash if no shell specified
	if shell == "" {
		shell = "bash"
	}

	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container runtime: %w", err)
	}

	// For interactive shell, we need to use the appropriate command directly
	// This is a runtime-specific operation that may need special handling
	switch runtime.GetType() {
	case RuntimeTypeDocker:
		cmd := m.factory.executor.CommandContext(ctx, "docker", "exec", "-it", containerID, shell)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("interactive shell not supported for runtime type: %s", runtime.GetType())
	}
}

// Attach attaches to a running container
func (m *Manager) Attach(ctx context.Context, containerID string) error {
	// Validate container ID
	if err := validateContainerID(containerID); err != nil {
		return fmt.Errorf("invalid container ID: %w", err)
	}

	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container runtime: %w", err)
	}

	// For interactive attach, we need to use the appropriate command directly
	switch runtime.GetType() {
	case RuntimeTypeDocker:
		cmd := m.factory.executor.CommandContext(ctx, "docker", "attach", containerID)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("attach not supported for runtime type: %s", runtime.GetType())
	}
}

// CopyToContainer copies files to a container
func (m *Manager) CopyToContainer(ctx context.Context, containerID, srcPath, dstPath string) error {
	// Validate container ID
	if err := validateContainerID(containerID); err != nil {
		return fmt.Errorf("invalid container ID: %w", err)
	}

	// Validate and clean paths
	cleanSrcPath, err := validatePath(srcPath)
	if err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}

	cleanDstPath, err := validatePath(dstPath)
	if err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container runtime: %w", err)
	}

	// File copy is runtime-specific
	switch runtime.GetType() {
	case RuntimeTypeDocker:
		args := []string{"cp", cleanSrcPath, fmt.Sprintf("%s:%s", containerID, cleanDstPath)}
		cmd := m.factory.executor.CommandContext(ctx, "docker", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to copy to container: %w, output: %s", err, string(output))
		}
		return nil
	default:
		return fmt.Errorf("copy to container not supported for runtime type: %s", runtime.GetType())
	}
}

// CopyFromContainer copies files from a container
func (m *Manager) CopyFromContainer(ctx context.Context, containerID, srcPath, dstPath string) error {
	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container runtime: %w", err)
	}

	// File copy is runtime-specific
	switch runtime.GetType() {
	case RuntimeTypeDocker:
		args := []string{"cp", fmt.Sprintf("%s:%s", containerID, srcPath), dstPath}
		cmd := m.factory.executor.CommandContext(ctx, "docker", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to copy from container: %w, output: %s", err, string(output))
		}
		return nil
	default:
		return fmt.Errorf("copy from container not supported for runtime type: %s", runtime.GetType())
	}
}

// Top shows process information for a container
func (m *Manager) Top(ctx context.Context, containerID string) ([]byte, error) {
	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container runtime: %w", err)
	}

	// Process info is runtime-specific
	switch runtime.GetType() {
	case RuntimeTypeDocker:
		cmd := m.factory.executor.CommandContext(ctx, "docker", "top", containerID)
		return cmd.Output()
	default:
		return nil, fmt.Errorf("top not supported for runtime type: %s", runtime.GetType())
	}
}

// Port shows port mappings for a container
func (m *Manager) Port(ctx context.Context, containerID string, port string) ([]byte, error) {
	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container runtime: %w", err)
	}

	// Port info is runtime-specific
	switch runtime.GetType() {
	case RuntimeTypeDocker:
		args := []string{"port", containerID}
		if port != "" {
			args = append(args, port)
		}
		cmd := m.factory.executor.CommandContext(ctx, "docker", args...)
		return cmd.Output()
	default:
		return nil, fmt.Errorf("port not supported for runtime type: %s", runtime.GetType())
	}
}

// RunSetup executes setup commands or script in a container
func (m *Manager) RunSetup(ctx context.Context, containerID string, projectPath string) error {
	// Validate container ID
	if err := validateContainerID(containerID); err != nil {
		return fmt.Errorf("invalid container ID: %w", err)
	}

	// Get container config
	container := &m.config.Repository.Repository.Container

	// Check if setup is needed
	if len(container.Setup) == 0 {
		// No setup configured
		return nil
	}

	// Ensure container is running before setup
	containers, err := m.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}

	var targetContainer *Container
	for _, c := range containers {
		if c.ID == containerID {
			targetContainer = c
			break
		}
	}

	if targetContainer == nil {
		return fmt.Errorf("container not found: %s", containerID)
	}

	// Check if container is running (handle different status formats)
	isRunning := false
	status := strings.ToLower(targetContainer.Status)
	if strings.Contains(status, "running") || strings.Contains(status, "up") {
		isRunning = true
	}

	if !isRunning {
		// Try to start the container
		logger.WithFields(logger.Fields{
			"container_id": containerID,
			"status":       targetContainer.Status,
			"operation":    "start",
		}).Info("Container is not running, starting it")
		if err := m.Start(ctx, containerID); err != nil {
			return fmt.Errorf("failed to start container for setup: %w", err)
		}
		// Wait a moment for container to be ready
		time.Sleep(2 * time.Second)
	}

	// Note: SetupScript removed in simplified approach - only inline setup commands supported

	// Handle inline Setup commands
	if len(container.Setup) > 0 {
		logger.WithFields(logger.Fields{
			"container":   containerID,
			"setup_count": len(container.Setup),
			"operation":   "setup_commands",
		}).Info("Running setup commands")
		for i, cmd := range container.Setup {
			// Skip empty commands
			if strings.TrimSpace(cmd) == "" {
				continue
			}

			logger.WithFields(logger.Fields{
				"container": containerID,
				"command":   cmd,
				"step":      i + 1,
				"total":     len(container.Setup),
				"operation": "setup_command",
			}).Info("Running setup command")

			// Execute command using sh -c for proper shell expansion
			execCmd := []string{"sh", "-c", cmd}
			output, err := m.Exec(ctx, containerID, execCmd)
			if err != nil {
				return fmt.Errorf("setup command %d failed: %w\nCommand: %s\nOutput:\n%s", i+1, err, cmd, string(output))
			}
			if len(output) > 0 && strings.TrimSpace(string(output)) != "" {
				logger.WithFields(logger.Fields{
					"container": containerID,
					"command":   cmd,
					"output":    strings.TrimSpace(string(output)),
					"operation": "setup_command_output",
				}).Debug("Setup command output")
			}
		}
		fmt.Println("✓ Setup commands completed successfully")
	}

	return nil
}

// RunLifecycleHook executes lifecycle hook commands for a specific hook type
// Note: RunLifecycleHook removed in simplified approach
func (m *Manager) RunLifecycleHook(ctx context.Context, containerID string, hook string) error {
	return fmt.Errorf("lifecycle hooks not supported in simplified configuration approach")

}

// getRuntime returns the current runtime, creating one if needed
func (m *Manager) getRuntime(ctx context.Context) (ContainerRuntime, error) {
	// Check with read lock first
	m.runtimeMutex.RLock()
	if m.runtime != nil {
		defer m.runtimeMutex.RUnlock()
		return m.runtime, nil
	}
	m.runtimeMutex.RUnlock()

	// Need to create runtime, acquire write lock
	m.runtimeMutex.Lock()
	defer m.runtimeMutex.Unlock()

	// Double-check after acquiring write lock
	if m.runtime != nil {
		return m.runtime, nil
	}

	// Determine runtime type from configuration
	runtimeType := RuntimeType(m.config.Repository.Repository.Runtime.Type)
	if runtimeType == "" {
		// Default to docker if not specified
		runtimeType = RuntimeTypeDocker
	}

	// Validate runtime type
	if runtimeType != RuntimeTypeDocker {
		return nil, fmt.Errorf("invalid runtime type: %s (only 'docker' is supported)", runtimeType)
	}

	runtime, err := m.factory.CreateForType(ctx, runtimeType)
	if err != nil {
		return nil, err
	}

	m.runtime = runtime
	return runtime, nil
}

// handleContainerError enriches container errors with additional context
func (m *Manager) handleContainerError(err error, operation string) error {
	if err == nil {
		return nil
	}
	
	// If it's already a ContainerError, just return it
	if _, ok := err.(*ContainerError); ok {
		return err
	}
	
	// Otherwise, wrap it in a ContainerError
	return &ContainerError{
		Type:       ErrorTypeUnknown,
		Operation:  operation,
		Message:    err.Error(),
		Underlying: err,
	}
}
