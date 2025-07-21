package service

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"vibeman/internal/config"
)

// waitForServiceStart waits for a service to finish starting
// This is a safer implementation that avoids the complex lock/unlock pattern
func (m *Manager) waitForServiceStart(ctx context.Context, name string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Get current status with minimal locking
			status, err := m.getServiceStatus(name)
			if err != nil {
				return fmt.Errorf("service instance disappeared during startup: %w", err)
			}

			switch status {
			case StatusRunning:
				// Service started successfully, increment ref count
				return m.incrementRefCount(name)
			case StatusError, StatusStopped:
				return fmt.Errorf("service failed to start: %s", name)
			case StatusStarting:
				// Still starting, continue waiting
				continue
			default:
				return fmt.Errorf("unexpected service status: %s", status)
			}
		}
	}
}

// getServiceStatus safely retrieves the status of a service
func (m *Manager) getServiceStatus(name string) (ServiceStatus, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	instance, exists := m.services[name]
	if !exists {
		return StatusUnknown, fmt.Errorf("service not found: %s", name)
	}

	instance.mutex.RLock()
	defer instance.mutex.RUnlock()

	return instance.Status, nil
}

// incrementRefCount safely increments the reference count for a service
func (m *Manager) incrementRefCount(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	instance, exists := m.services[name]
	if !exists {
		return fmt.Errorf("service not found: %s", name)
	}

	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	instance.RefCount++
	return nil
}

// startServiceSafe is a safer implementation of service starting
// that avoids the complex lock release/reacquire pattern
func (m *Manager) startServiceSafe(ctx context.Context, name string) error {
	// First, check if we need to create the service instance
	needsCreation := false
	m.mutex.RLock()
	_, exists := m.services[name]
	m.mutex.RUnlock()

	if !exists {
		needsCreation = true
	}

	// Create service instance if needed
	if needsCreation {
		m.mutex.Lock()
		// Double-check after acquiring write lock
		if _, exists := m.services[name]; !exists {
			serviceConfig, configExists := m.config.Services.Services[name]
			if !configExists {
				m.mutex.Unlock()
				return fmt.Errorf("service configuration not found: %s", name)
			}

			instance := &ServiceInstance{
				Name:     name,
				Config:   serviceConfig,
				Status:   StatusStopped,
				RefCount: 0,
				Repositories: []string{},
			}
			m.services[name] = instance
		}
		m.mutex.Unlock()
	}

	// Now check the service status
	status, err := m.getServiceStatus(name)
	if err != nil {
		return err
	}

	switch status {
	case StatusRunning:
		// Already running, just increment ref count
		return m.incrementRefCount(name)
	case StatusStarting:
		// Wait for it to complete
		return m.waitForServiceStart(ctx, name)
	case StatusStopped, StatusError:
		// Need to start the service
		return m.doStartService(ctx, name)
	default:
		return fmt.Errorf("unexpected service status: %s", status)
	}
}

// doStartService performs the actual service start
func (m *Manager) doStartService(ctx context.Context, name string) error {
	// Set status to starting
	m.mutex.Lock()
	instance, exists := m.services[name]
	if !exists {
		m.mutex.Unlock()
		return fmt.Errorf("service not found: %s", name)
	}

	instance.mutex.Lock()
	instance.Status = StatusStarting
	instance.mutex.Unlock()
	m.mutex.Unlock()

	// Docker Compose handles dependencies automatically
	// No need to manually start dependencies

	// Start the container (outside of locks)
	if err := m.startServiceContainerSafe(ctx, name); err != nil {
		m.setServiceError(name)
		return fmt.Errorf("failed to start service %s: %w", name, err)
	}

	// Set status to running
	m.setServiceRunning(name)
	return nil
}

// setServiceError safely sets a service status to error
func (m *Manager) setServiceError(name string) {
	m.mutex.RLock()
	instance, exists := m.services[name]
	m.mutex.RUnlock()

	if exists {
		instance.mutex.Lock()
		instance.Status = StatusError
		instance.mutex.Unlock()
	}
}

// setServiceRunning safely sets a service status to running
func (m *Manager) setServiceRunning(name string) {
	m.mutex.RLock()
	instance, exists := m.services[name]
	m.mutex.RUnlock()

	if exists {
		instance.mutex.Lock()
		instance.Status = StatusRunning
		instance.RefCount = 1
		instance.StartTime = time.Now()
		instance.LastHealth = time.Now()
		instance.mutex.Unlock()
	}
}

// startServiceContainerSafe starts a container with proper error handling
func (m *Manager) startServiceContainerSafe(ctx context.Context, name string) error {
	m.mutex.RLock()
	instance, exists := m.services[name]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("service not found: %s", name)
	}

	// Get a copy of the config to use outside of locks
	instance.mutex.RLock()
	config := instance.Config
	instance.mutex.RUnlock()

	// Start the container
	containerID, err := m.createServiceContainer(ctx, config)
	if err != nil {
		return err
	}

	// Update the container ID
	instance.mutex.Lock()
	instance.ContainerID = containerID
	instance.mutex.Unlock()

	return nil
}

// createServiceContainer creates and starts a service container
// This is a separate method to keep the logic clean and testable
func (m *Manager) createServiceContainer(ctx context.Context, config config.ServiceConfig) (string, error) {
	// All services now use Docker Compose
	if !config.IsValid() {
		return "", fmt.Errorf("invalid service configuration: compose_file and service name required")
	}

	// Use docker compose to start the service
	args := []string{
		"compose",
		"-f", config.ComposeFile,
		"up",
		"-d",
		config.Service,
	}

	// Execute docker compose up
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to start compose service: %w, output: %s", err, string(output))
	}

	// Get container ID from docker compose
	containerID, err := m.getComposeContainerID(ctx, config.ComposeFile, config.Service)
	if err != nil {
		return "", fmt.Errorf("failed to get container ID: %w", err)
	}

	return containerID, nil
}
