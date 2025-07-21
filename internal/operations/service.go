package operations

import (
	"context"
	"fmt"
	"time"

	"vibeman/internal/config"
	"vibeman/internal/logger"
	"vibeman/internal/service"
)

// ServiceInfo represents information about a service
type ServiceInfo struct {
	Name         string              `json:"name"`
	Status       service.ServiceStatus `json:"status"`
	ContainerID  string              `json:"container_id,omitempty"`
	RefCount     int                 `json:"ref_count"`
	Repositories []string            `json:"repositories"`
	StartTime    *time.Time          `json:"start_time,omitempty"`
	Uptime       string              `json:"uptime,omitempty"`
	HealthError  string              `json:"health_error,omitempty"`
	Config       config.ServiceConfig `json:"config"`
}

// ServiceOperations provides shared backend functions for service management
type ServiceOperations struct {
	cfg        *config.Manager
	serviceMgr ServiceManager
}

// NewServiceOperations creates a new ServiceOperations instance
func NewServiceOperations(cfg *config.Manager, sm ServiceManager) *ServiceOperations {
	return &ServiceOperations{
		cfg:        cfg,
		serviceMgr: sm,
	}
}


// ListServices returns all configured services with their status
func (so *ServiceOperations) ListServices(ctx context.Context) ([]*ServiceInfo, error) {
	if so.cfg.Services == nil {
		return nil, fmt.Errorf("services configuration not available")
	}

	var services []*ServiceInfo

	// Get all configured services
	for name, serviceConfig := range so.cfg.Services.Services {
		// Get the service instance if it exists
		instanceInterface, err := so.serviceMgr.GetService(name)
		if err != nil {
			// Service not running, create a minimal info
			services = append(services, &ServiceInfo{
				Name:   name,
				Status: service.StatusStopped,
				Config: serviceConfig,
			})
			continue
		}

		// Type assert to *service.ServiceInstance
		instance, ok := instanceInterface.(*service.ServiceInstance)
		if !ok {
			logger.WithField("service", name).Error("GetService returned unexpected type")
			continue
		}

		// Calculate uptime if running
		var uptime string
		var startTime *time.Time
		if instance.Status == service.StatusRunning && !instance.StartTime.IsZero() {
			startTime = &instance.StartTime
			duration := time.Since(instance.StartTime)
			uptime = formatDuration(duration)
		}

		services = append(services, &ServiceInfo{
			Name:         instance.Name,
			Status:       instance.Status,
			ContainerID:  instance.ContainerID,
			RefCount:     instance.RefCount,
			Repositories: instance.Repositories,
			StartTime:    startTime,
			Uptime:       uptime,
			HealthError:  instance.HealthError,
			Config:       instance.Config,
		})
	}

	return services, nil
}

// GetService returns information about a specific service
func (so *ServiceOperations) GetService(ctx context.Context, name string) (*ServiceInfo, error) {
	if so.cfg.Services == nil {
		return nil, fmt.Errorf("services configuration not available")
	}

	// Check if service configuration exists
	serviceConfig, exists := so.cfg.Services.Services[name]
	if !exists {
		return nil, fmt.Errorf("service configuration not found: %s", name)
	}

	// Get the service instance
	instanceInterface, err := so.serviceMgr.GetService(name)
	if err != nil {
		// Service not running, return minimal info
		return &ServiceInfo{
			Name:   name,
			Status: service.StatusStopped,
			Config: serviceConfig,
		}, nil
	}

	// Type assert to *service.ServiceInstance
	instance, ok := instanceInterface.(*service.ServiceInstance)
	if !ok {
		return nil, fmt.Errorf("GetService returned unexpected type for service: %s", name)
	}

	// Calculate uptime if running
	var uptime string
	var startTime *time.Time
	if instance.Status == service.StatusRunning && !instance.StartTime.IsZero() {
		startTime = &instance.StartTime
		duration := time.Since(instance.StartTime)
		uptime = formatDuration(duration)
	}

	return &ServiceInfo{
		Name:         instance.Name,
		Status:       instance.Status,
		ContainerID:  instance.ContainerID,
		RefCount:     instance.RefCount,
		Repositories: instance.Repositories,
		StartTime:    startTime,
		Uptime:       uptime,
		HealthError:  instance.HealthError,
		Config:       instance.Config,
	}, nil
}

// StartService starts a service
func (so *ServiceOperations) StartService(ctx context.Context, name string) error {
	logger.WithField("service", name).Info("Starting service")

	if err := so.serviceMgr.StartService(ctx, name); err != nil {
		return fmt.Errorf("failed to start service %s: %w", name, err)
	}

	logger.WithField("service", name).Info("Service started successfully")
	return nil
}

// StopService stops a service
func (so *ServiceOperations) StopService(ctx context.Context, name string) error {
	logger.WithField("service", name).Info("Stopping service")

	if err := so.serviceMgr.StopService(ctx, name); err != nil {
		return fmt.Errorf("failed to stop service %s: %w", name, err)
	}

	logger.WithField("service", name).Info("Service stopped successfully")
	return nil
}

// RestartService restarts a service
func (so *ServiceOperations) RestartService(ctx context.Context, name string) error {
	logger.WithField("service", name).Info("Restarting service")

	// Stop the service first
	if err := so.serviceMgr.StopService(ctx, name); err != nil {
		// If it's not running, that's okay
		logger.WithError(err).Debug("Service was not running")
	}

	// Wait a moment for cleanup
	time.Sleep(1 * time.Second)

	// Start the service
	if err := so.serviceMgr.StartService(ctx, name); err != nil {
		return fmt.Errorf("failed to restart service %s: %w", name, err)
	}

	logger.WithField("service", name).Info("Service restarted successfully")
	return nil
}

// HealthCheckService performs a health check on a service
func (so *ServiceOperations) HealthCheckService(ctx context.Context, name string) error {
	return so.serviceMgr.HealthCheck(ctx, name)
}

// Helper functions

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}