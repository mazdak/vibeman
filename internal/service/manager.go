package service

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"vibeman/internal/config"
)

// ServiceStatus represents the status of a service
type ServiceStatus string

const (
	StatusStopped  ServiceStatus = "stopped"
	StatusStarting ServiceStatus = "starting"
	StatusRunning  ServiceStatus = "running"
	StatusStopping ServiceStatus = "stopping"
	StatusError    ServiceStatus = "error"
	StatusUnknown  ServiceStatus = "unknown"
)

// ServiceInstance represents a running service instance
type ServiceInstance struct {
	Name        string               `json:"name"`
	ContainerID string               `json:"container_id"`
	Status      ServiceStatus        `json:"status"`
	RefCount    int                  `json:"ref_count"`
	Repositories []string             `json:"repositories"`
	Config      config.ServiceConfig `json:"config"`
	StartTime   time.Time            `json:"start_time"`
	LastHealth  time.Time            `json:"last_health"`
	HealthError string               `json:"health_error,omitempty"`
	mutex       sync.RWMutex         `json:"-"`
}

// Manager handles service lifecycle operations
type Manager struct {
	config   *config.Manager
	services map[string]*ServiceInstance
	mutex    sync.RWMutex
}

// New creates a new service manager
func New(cfg *config.Manager) *Manager {
	return &Manager{
		config:   cfg,
		services: make(map[string]*ServiceInstance),
	}
}

// StartService starts a service with reference counting
func (m *Manager) StartService(ctx context.Context, name string) error {
	// Check if services configuration is available
	m.mutex.Lock()
	if m.config.Services == nil {
		m.mutex.Unlock()
		return fmt.Errorf("services configuration not available")
	}

	// Check if service configuration exists
	serviceConfig, exists := m.config.Services.Services[name]
	if !exists {
		m.mutex.Unlock()
		return fmt.Errorf("service configuration not found: %s", name)
	}

	// Get or create service instance
	instance, exists := m.services[name]
	if !exists {
		instance = &ServiceInstance{
			Name:     name,
			Config:   serviceConfig,
			Status:   StatusStopped,
			RefCount: 0,
			Repositories: []string{},
		}
		m.services[name] = instance
	}
	m.mutex.Unlock()

	// Now handle the instance-specific logic
	instance.mutex.Lock()

	// If service is already running, just increment reference count
	if instance.Status == StatusRunning {
		instance.RefCount++
		instance.mutex.Unlock()
		return nil
	}

	// If service is starting, wait for it to complete
	if instance.Status == StatusStarting {
		instance.mutex.Unlock()

		// Wait for the service to finish starting
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				instance.mutex.RLock()
				status := instance.Status
				instance.mutex.RUnlock()

				if status == StatusRunning {
					// Service is now running, increment ref count and return
					instance.mutex.Lock()
					instance.RefCount++
					instance.mutex.Unlock()
					return nil
				}
				if status == StatusError || status == StatusStopped {
					return fmt.Errorf("service failed to start: %s", name)
				}
			}
		}
	}

	// Mark as starting
	instance.Status = StatusStarting
	instance.mutex.Unlock()

	// Docker Compose handles dependencies automatically
	var dependencies []string

	// Start dependencies without holding any locks
	if err := m.startDependencies(ctx, dependencies); err != nil {
		// Update status to error
		instance.mutex.Lock()
		instance.Status = StatusError
		instance.mutex.Unlock()
		return fmt.Errorf("failed to start dependencies for %s: %w", name, err)
	}

	// Start the service container
	if err := m.startServiceContainer(ctx, instance); err != nil {
		instance.mutex.Lock()
		instance.Status = StatusError
		instance.mutex.Unlock()
		return fmt.Errorf("failed to start service %s: %w", name, err)
	}

	// Update status to running
	instance.mutex.Lock()
	instance.Status = StatusRunning
	instance.RefCount = 1
	instance.StartTime = time.Now()
	instance.LastHealth = time.Now()
	instance.mutex.Unlock()

	return nil
}

// StopService stops a service with reference counting
func (m *Manager) StopService(ctx context.Context, name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	instance, exists := m.services[name]
	if !exists {
		// Check if it's a compose service we can stop directly
		serviceConfig, configExists := m.config.Services.Services[name]
		if !configExists {
			return fmt.Errorf("service not found: %s", name)
		}

		if serviceConfig.IsValid() {
			// For compose services, stop directly without tracking
			return m.stopComposeService(ctx, &ServiceInstance{
				Name:   name,
				Config: serviceConfig,
			})
		}

		return fmt.Errorf("service not found: %s", name)
	}

	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	if instance.Status != StatusRunning {
		return fmt.Errorf("service is not running: %s", name)
	}

	// Decrement reference count
	instance.RefCount--

	// If still referenced, don't stop
	if instance.RefCount > 0 {
		return nil
	}

	// Stop the service
	instance.Status = StatusStopping

	if err := m.stopServiceContainer(ctx, instance); err != nil {
		instance.Status = StatusError
		return fmt.Errorf("failed to stop service %s: %w", name, err)
	}

	instance.Status = StatusStopped
	instance.ContainerID = ""
	instance.Repositories = []string{}

	return nil
}

// GetService returns a service instance by name
func (m *Manager) GetService(name string) (interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	instance, exists := m.services[name]
	if !exists {
		// For compose services, create a minimal instance with actual status
		serviceConfig, configExists := m.config.Services.Services[name]
		if !configExists {
			return nil, fmt.Errorf("service not found: %s", name)
		}

		if serviceConfig.IsValid() {
			// Create temporary instance to check compose status
			status, err := m.getComposeServiceStatus(context.Background(), serviceConfig.ComposeFile, serviceConfig.Service)
			if err != nil {
				status = StatusUnknown
			}

			// Get container ID if running
			containerID := ""
			if status == StatusRunning {
				if id, err := m.getComposeContainerID(context.Background(), serviceConfig.ComposeFile, serviceConfig.Service); err == nil {
					containerID = id
				}
			}

			return &ServiceInstance{
				Name:        name,
				ContainerID: containerID,
				Status:      status,
				RefCount:    0,
				Repositories: []string{},
				Config:      serviceConfig,
				StartTime:   time.Now(), // Use current time for uptime calculation
				LastHealth:  time.Now(),
				HealthError: "",
			}, nil
		}

		return nil, fmt.Errorf("service not found: %s", name)
	}

	instance.mutex.RLock()
	defer instance.mutex.RUnlock()

	// Return a copy to avoid race conditions
	return &ServiceInstance{
		Name:        instance.Name,
		ContainerID: instance.ContainerID,
		Status:      instance.Status,
		RefCount:    instance.RefCount,
		Repositories: append([]string{}, instance.Repositories...),
		Config:      instance.Config,
		StartTime:   instance.StartTime,
		LastHealth:  instance.LastHealth,
		HealthError: instance.HealthError,
	}, nil
}

// ListServices returns all service instances
func (m *Manager) ListServices() []*ServiceInstance {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	instances := make([]*ServiceInstance, 0, len(m.services))
	for _, instance := range m.services {
		instance.mutex.RLock()
		instances = append(instances, &ServiceInstance{
			Name:        instance.Name,
			ContainerID: instance.ContainerID,
			Status:      instance.Status,
			RefCount:    instance.RefCount,
			Repositories: append([]string{}, instance.Repositories...),
			Config:      instance.Config,
			StartTime:   instance.StartTime,
			LastHealth:  instance.LastHealth,
			HealthError: instance.HealthError,
		})
		instance.mutex.RUnlock()
	}

	return instances
}

// HealthCheck performs a health check on a service
func (m *Manager) HealthCheck(ctx context.Context, name string) error {
	m.mutex.RLock()
	instance, exists := m.services[name]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("service not found: %s", name)
	}

	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	if instance.Status != StatusRunning || instance.ContainerID == "" {
		return fmt.Errorf("service is not running: %s", name)
	}

	// For compose services, just check if container is running
	// Docker compose handles health checks internally
	if err := m.checkContainerStatus(ctx, instance.ContainerID); err != nil {
		instance.HealthError = err.Error()
		instance.LastHealth = time.Now()
		return err
	}

	instance.HealthError = ""
	instance.LastHealth = time.Now()
	return nil
}

// AddReference adds a repository reference to a service
func (m *Manager) AddReference(serviceName, repositoryName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	instance, exists := m.services[serviceName]
	if !exists {
		return fmt.Errorf("service not found: %s", serviceName)
	}

	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	// Check if project is already referenced
	for _, repository := range instance.Repositories {
		if repository == repositoryName {
			return nil // Already referenced
		}
	}

	instance.Repositories = append(instance.Repositories, repositoryName)
	instance.RefCount++

	return nil
}

// RemoveReference removes a repository reference from a service
func (m *Manager) RemoveReference(serviceName, repositoryName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	instance, exists := m.services[serviceName]
	if !exists {
		return fmt.Errorf("service not found: %s", serviceName)
	}

	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	// Find and remove project reference
	for i, repository := range instance.Repositories {
		if repository == repositoryName {
			instance.Repositories = append(instance.Repositories[:i], instance.Repositories[i+1:]...)
			instance.RefCount--
			return nil
		}
	}

	return fmt.Errorf("repository reference not found: %s", repositoryName)
}

// startDependencies starts all service dependencies
func (m *Manager) startDependencies(ctx context.Context, dependencies []string) error {
	for _, dep := range dependencies {
		if err := m.StartService(ctx, dep); err != nil {
			return fmt.Errorf("failed to start dependency %s: %w", dep, err)
		}
	}
	return nil
}

// startServiceContainer starts the actual container for a service
func (m *Manager) startServiceContainer(ctx context.Context, instance *ServiceInstance) error {
	// All services must use Docker Compose
	if !instance.Config.IsValid() {
		return fmt.Errorf("invalid service configuration: compose_file and service name are required")
	}

	return m.startComposeService(ctx, instance)
}

// stopServiceContainer stops the container for a service
func (m *Manager) stopServiceContainer(ctx context.Context, instance *ServiceInstance) error {
	// All services must use Docker Compose
	if !instance.Config.IsValid() {
		return fmt.Errorf("invalid service configuration: compose_file and service name are required")
	}

	return m.stopComposeService(ctx, instance)
}

// checkContainerStatus checks if a container is running
func (m *Manager) checkContainerStatus(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, "docker", "inspect", containerID, "--format", "{{.State.Running}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check container status: %w, output: %s", err, string(output))
	}

	if strings.TrimSpace(string(output)) != "true" {
		return fmt.Errorf("container is not running")
	}

	return nil
}

// runHealthCheck runs the configured health check for a service
func (m *Manager) runHealthCheck(ctx context.Context, instance *ServiceInstance) error {
	// For compose services, docker-compose handles health checks
	// We just check if the container is running
	if instance.ContainerID == "" {
		return fmt.Errorf("no container ID for service %s", instance.Name)
	}

	return m.checkContainerStatus(ctx, instance.ContainerID)
}

// startComposeService starts a service using Docker Compose
func (m *Manager) startComposeService(ctx context.Context, instance *ServiceInstance) error {
	composeFile := instance.Config.ComposeFile
	serviceName := instance.Config.Service

	// Build docker compose command
	args := []string{
		"compose",
		"-f", composeFile,
		"up",
		"-d",
		serviceName,
	}

	// Execute docker compose up
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start compose service: %w, output: %s", err, string(output))
	}

	// Get container ID from docker compose
	containerID, err := m.getComposeContainerID(ctx, composeFile, serviceName)
	if err != nil {
		return fmt.Errorf("failed to get container ID: %w", err)
	}

	instance.ContainerID = containerID
	return nil
}

// stopComposeService stops a service using Docker Compose
func (m *Manager) stopComposeService(ctx context.Context, instance *ServiceInstance) error {
	composeFile := instance.Config.ComposeFile
	serviceName := instance.Config.Service

	// Build docker compose stop command
	args := []string{
		"compose",
		"-f", composeFile,
		"stop",
		serviceName,
	}

	// Execute docker compose stop
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop compose service: %w, output: %s", err, string(output))
	}

	return nil
}

// getComposeContainerID gets the container ID for a compose service
func (m *Manager) getComposeContainerID(ctx context.Context, composeFile, serviceName string) (string, error) {
	args := []string{
		"compose",
		"-f", composeFile,
		"ps",
		"-q",
		serviceName,
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get container ID: %w, output: %s", err, string(output))
	}

	containerID := strings.TrimSpace(string(output))
	if containerID == "" {
		return "", fmt.Errorf("no container found for service %s", serviceName)
	}

	return containerID, nil
}

// getComposeServiceStatus gets the status of a compose service
func (m *Manager) getComposeServiceStatus(ctx context.Context, composeFile, serviceName string) (ServiceStatus, error) {
	args := []string{
		"compose",
		"-f", composeFile,
		"ps",
		"--format", "json",
		serviceName,
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return StatusUnknown, fmt.Errorf("failed to get service status: %w, output: %s", err, string(output))
	}

	// Parse the JSON output to determine status
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" || outputStr == "[]" {
		return StatusStopped, nil
	}

	// Simple status detection based on output
	if strings.Contains(outputStr, "running") || strings.Contains(outputStr, "Up") {
		return StatusRunning, nil
	}

	return StatusStopped, nil
}
