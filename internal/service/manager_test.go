package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"vibeman/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	cfg := &config.Manager{
		Services: &config.ServicesConfig{
			Services: map[string]config.ServiceConfig{
				"postgres": {
					ComposeFile: "docker-compose.yml",
					Service:     "postgres",
					Description: "PostgreSQL database",
				},
			},
		},
	}

	manager := New(cfg)
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.config)
	assert.NotNil(t, manager.services)
	assert.Equal(t, 0, len(manager.services))
}

func TestStartService_NoConfiguration(t *testing.T) {
	cfg := &config.Manager{}
	manager := New(cfg)

	err := manager.StartService(context.Background(), "postgres")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "services configuration not available")
}

func TestStartService_ServiceNotFound(t *testing.T) {
	cfg := &config.Manager{
		Services: &config.ServicesConfig{
			Services: map[string]config.ServiceConfig{},
		},
	}
	manager := New(cfg)

	err := manager.StartService(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service configuration not found")
}

func TestGetService(t *testing.T) {
	cfg := &config.Manager{
		Services: &config.ServicesConfig{
			Services: map[string]config.ServiceConfig{
				"postgres": {
					ComposeFile: "docker-compose.yml",
					Service:     "postgres",
				},
			},
		},
	}
	manager := New(cfg)

	// Add a service instance manually
	instance := &ServiceInstance{
		Name:        "postgres",
		Status:      StatusRunning,
		RefCount:    1,
		ContainerID: "container-123",
		StartTime:   time.Now(),
	}
	manager.services["postgres"] = instance

	// Test getting existing service
	serviceInterface, err := manager.GetService("postgres")
	require.NoError(t, err)
	serviceInfo, ok := serviceInterface.(*ServiceInstance)
	require.True(t, ok, "service should be a *ServiceInstance")
	assert.Equal(t, "postgres", serviceInfo.Name)
	assert.Equal(t, StatusRunning, serviceInfo.Status)
	assert.Equal(t, "container-123", serviceInfo.ContainerID)

	// Test getting non-existent service
	_, err = manager.GetService("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service not found")
}

func TestListServices(t *testing.T) {
	cfg := &config.Manager{
		Services: &config.ServicesConfig{
			Services: map[string]config.ServiceConfig{
				"postgres": {
					ComposeFile: "docker-compose.yml",
					Service:     "postgres",
				},
				"redis": {
					ComposeFile: "docker-compose.yml",
					Service:     "redis",
				},
			},
		},
	}
	manager := New(cfg)

	// Add some service instances
	manager.services["postgres"] = &ServiceInstance{
		Name:     "postgres",
		Status:   StatusRunning,
		RefCount: 2,
	}
	manager.services["redis"] = &ServiceInstance{
		Name:     "redis",
		Status:   StatusStopped,
		RefCount: 0,
	}

	services := manager.ListServices()
	assert.Len(t, services, 2)

	// Check that we got copies, not references
	serviceNames := map[string]bool{}
	for _, service := range services {
		serviceNames[service.Name] = true
		switch service.Name {
		case "postgres":
			assert.Equal(t, StatusRunning, service.Status)
			assert.Equal(t, 2, service.RefCount)
		case "redis":
			assert.Equal(t, StatusStopped, service.Status)
			assert.Equal(t, 0, service.RefCount)
		}
	}
	assert.True(t, serviceNames["postgres"])
	assert.True(t, serviceNames["redis"])
}

func TestAddReference(t *testing.T) {
	cfg := &config.Manager{
		Services: &config.ServicesConfig{
			Services: map[string]config.ServiceConfig{
				"postgres": {
					ComposeFile: "docker-compose.yml",
					Service:     "postgres",
				},
			},
		},
	}
	manager := New(cfg)

	// Test adding reference to non-existent service
	err := manager.AddReference("postgres", "repo1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service not found")

	// Add service instance
	instance := &ServiceInstance{
		Name:         "postgres",
		Status:       StatusRunning,
		RefCount:     1,
		Repositories: []string{},
	}
	manager.services["postgres"] = instance

	// Add reference
	err = manager.AddReference("postgres", "repo1")
	require.NoError(t, err)
	assert.Equal(t, 2, instance.RefCount)
	assert.Contains(t, instance.Repositories, "repo1")

	// Add same repository again - should not duplicate but still increments refcount
	err = manager.AddReference("postgres", "repo1")
	require.NoError(t, err)
	assert.Equal(t, 2, instance.RefCount) // Should still be 2 since repo already referenced
	assert.Equal(t, 1, len(instance.Repositories))

	// Add different repository
	err = manager.AddReference("postgres", "repo2")
	require.NoError(t, err)
	assert.Equal(t, 3, instance.RefCount)
	assert.Equal(t, 2, len(instance.Repositories))
	assert.Contains(t, instance.Repositories, "repo2")
}

func TestRemoveReference(t *testing.T) {
	cfg := &config.Manager{
		Services: &config.ServicesConfig{
			Services: map[string]config.ServiceConfig{
				"postgres": {
					ComposeFile: "docker-compose.yml",
					Service:     "postgres",
				},
			},
		},
	}
	manager := New(cfg)

	// Test removing reference from non-existent service
	err := manager.RemoveReference("postgres", "repo1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service not found")

	// Add service instance with references
	instance := &ServiceInstance{
		Name:         "postgres",
		Status:       StatusRunning,
		RefCount:     3,
		Repositories: []string{"repo1", "repo2"},
	}
	manager.services["postgres"] = instance

	// Remove reference
	err = manager.RemoveReference("postgres", "repo1")
	require.NoError(t, err)
	assert.Equal(t, 2, instance.RefCount)
	assert.NotContains(t, instance.Repositories, "repo1")
	assert.Contains(t, instance.Repositories, "repo2")

	// Remove non-existent repository - should return error
	err = manager.RemoveReference("postgres", "repo3")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository reference not found")
	assert.Equal(t, 2, instance.RefCount) // RefCount should not change

	// Remove last reference
	err = manager.RemoveReference("postgres", "repo2")
	require.NoError(t, err)
	assert.Equal(t, 1, instance.RefCount)
	assert.Equal(t, 0, len(instance.Repositories))
}

func TestHealthCheck(t *testing.T) {
	cfg := &config.Manager{
		Services: &config.ServicesConfig{
			Services: map[string]config.ServiceConfig{
				"postgres": {
					ComposeFile: "docker-compose.yml",
					Service:     "postgres",
				},
			},
		},
	}
	manager := New(cfg)

	// Test health check for non-existent service
	err := manager.HealthCheck(context.Background(), "postgres")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service not found")

	// Add service instance
	instance := &ServiceInstance{
		Name:        "postgres",
		Status:      StatusRunning,
		ContainerID: "container-123",
		Config: config.ServiceConfig{
			ComposeFile: "docker-compose.yml",
			Service:     "postgres",
		},
	}
	manager.services["postgres"] = instance

	// Health check would require actual container runtime
	// For unit test, we can't test the actual execution
	// but we can verify the method exists and handles basic cases
}

func TestServiceInstance_ThreadSafety(t *testing.T) {
	// Test that concurrent access to service instance is safe
	instance := &ServiceInstance{
		Name:         "postgres",
		Status:       StatusRunning,
		RefCount:     0,
		Repositories: []string{},
	}

	// Simulate concurrent reference additions
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			instance.mutex.Lock()
			instance.RefCount++
			instance.Repositories = append(instance.Repositories, fmt.Sprintf("repo%d", id))
			instance.mutex.Unlock()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 10, instance.RefCount)
	assert.Equal(t, 10, len(instance.Repositories))
}

func TestManager_ThreadSafety(t *testing.T) {
	cfg := &config.Manager{
		Services: &config.ServicesConfig{
			Services: map[string]config.ServiceConfig{
				"postgres": {ComposeFile: "docker-compose.yml", Service: "postgres"},
				"redis":    {ComposeFile: "docker-compose.yml", Service: "redis"},
				"mongo":    {ComposeFile: "docker-compose.yml", Service: "mongo"},
			},
		},
	}
	manager := New(cfg)

	// Simulate concurrent operations
	done := make(chan bool)
	
	// Writers
	for i := 0; i < 5; i++ {
		go func(id int) {
			serviceName := fmt.Sprintf("service%d", id)
			manager.mutex.Lock()
			manager.services[serviceName] = &ServiceInstance{
				Name:     serviceName,
				Status:   StatusRunning,
				RefCount: 1,
			}
			manager.mutex.Unlock()
			done <- true
		}(i)
	}

	// Readers
	for i := 0; i < 5; i++ {
		go func() {
			manager.mutex.RLock()
			_ = len(manager.services)
			manager.mutex.RUnlock()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all services were added
	assert.GreaterOrEqual(t, len(manager.services), 5)
}