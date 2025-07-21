package types

import (
	"time"

	"vibeman/internal/config"
)

// ServiceStatus represents the status of a service
type ServiceStatus string

const (
	ServiceStatusStopped  ServiceStatus = "stopped"
	ServiceStatusStarting ServiceStatus = "starting"
	ServiceStatusRunning  ServiceStatus = "running"
	ServiceStatusStopping ServiceStatus = "stopping"
	ServiceStatusError    ServiceStatus = "error"
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
}
