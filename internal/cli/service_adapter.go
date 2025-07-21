package cli

import (
	"context"
	"fmt"

	"vibeman/internal/service"
	"vibeman/internal/types"
)

// ServiceManagerWrapper wraps the concrete service.Manager to implement the commands.ServiceManager interface
type ServiceManagerWrapper struct {
	manager *service.Manager
}

// NewServiceManagerWrapper creates a new wrapper
func NewServiceManagerWrapper(sm *service.Manager) *ServiceManagerWrapper {
	return &ServiceManagerWrapper{manager: sm}
}

// StartService starts a service
func (w *ServiceManagerWrapper) StartService(ctx context.Context, name string) error {
	return w.manager.StartService(ctx, name)
}

// StopService stops a service
func (w *ServiceManagerWrapper) StopService(ctx context.Context, name string) error {
	return w.manager.StopService(ctx, name)
}

// AddReference adds a repository reference to a service
func (w *ServiceManagerWrapper) AddReference(serviceName, repositoryName string) error {
	return w.manager.AddReference(serviceName, repositoryName)
}

// RemoveReference removes a repository reference from a service
func (w *ServiceManagerWrapper) RemoveReference(serviceName, repositoryName string) error {
	return w.manager.RemoveReference(serviceName, repositoryName)
}

// HealthCheck performs a health check on a service
func (w *ServiceManagerWrapper) HealthCheck(ctx context.Context, name string) error {
	return w.manager.HealthCheck(ctx, name)
}

// GetService retrieves service information
func (w *ServiceManagerWrapper) GetService(name string) (interface{}, error) {
	svc, err := w.manager.GetService(name)
	if err != nil {
		return nil, err
	}

	// Type assert to service.ServiceInstance
	svcInstance, ok := svc.(*service.ServiceInstance)
	if !ok {
		return nil, fmt.Errorf("unexpected service type: %T", svc)
	}

	// Convert from service.ServiceInstance to types.ServiceInstance
	return &types.ServiceInstance{
		Name:        svcInstance.Name,
		ContainerID: svcInstance.ContainerID,
		Status:      types.ServiceStatus(svcInstance.Status),
		RefCount:    svcInstance.RefCount,
		Repositories: svcInstance.Repositories,
		Config:      svcInstance.Config,
		StartTime:   svcInstance.StartTime,
		LastHealth:  svcInstance.LastHealth,
		HealthError: svcInstance.HealthError,
	}, nil
}
