package operations

import (
	"vibeman/internal/service"
)

// ServiceManagerAdapter adapts service.Manager to the ServiceManager interface
type ServiceManagerAdapter struct {
	*service.Manager
}

// NewServiceManagerAdapter creates a new adapter
func NewServiceManagerAdapter(sm *service.Manager) ServiceManager {
	return &ServiceManagerAdapter{Manager: sm}
}

// GetService adapts the return type to interface{}
func (a *ServiceManagerAdapter) GetService(name string) (interface{}, error) {
	return a.Manager.GetService(name)
}