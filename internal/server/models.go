package server

import (
	"time"
	"vibeman/internal/config"
	"vibeman/internal/db"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"Resource not found"`
}

// SuccessResponse represents a successful operation response
type SuccessResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// System Status API models

// SystemStatusResponse represents the overall system status
type SystemStatusResponse struct {
	Status       string              `json:"status" example:"healthy"`
	Version      string              `json:"version" example:"1.0.0"`
	Uptime       string              `json:"uptime" example:"2h30m15s"`
	Services     ServiceHealthStatus `json:"services"`
	Repositories int                 `json:"repositories" example:"5"`
	Worktrees    int                 `json:"worktrees" example:"12"`
	Containers   int                 `json:"containers" example:"3"`
}

// ServiceHealthStatus represents the health status of various services
type ServiceHealthStatus struct {
	Database        string `json:"database" example:"healthy"`
	ContainerEngine string `json:"container_engine" example:"healthy"`
	Git             string `json:"git" example:"healthy"`
}

// Repository API models

// AddRepositoryRequest represents a request to add a repository
type AddRepositoryRequest struct {
	Path        string `json:"path" validate:"required" example:"/home/user/projects/myapp"`
	Name        string `json:"name" validate:"required" example:"myapp"`
	Description string `json:"description" example:"My awesome application"`
}

// RepositoriesResponse represents a list of repositories
type RepositoriesResponse struct {
	Repositories []*db.Repository `json:"repositories"`
	Total        int              `json:"total" example:"10"`
}

// Worktree API models

// CreateWorktreeRequest represents a request to create a worktree
type CreateWorktreeRequest struct {
	RepositoryID    string   `json:"repository_id" validate:"required"`
	Name            string   `json:"name" validate:"required" example:"feature-auth"`
	Branch          string   `json:"branch" example:"feature/auth"`
	BaseBranch      string   `json:"base_branch" example:"main"`
	SkipSetup       bool     `json:"skip_setup" example:"false"`
	ContainerImage  string   `json:"container_image" example:"vibeman-dev:latest"`
	AutoStart       bool     `json:"auto_start" example:"true"`
	ComposeFile     string   `json:"compose_file" example:"./docker-compose.yaml"`
	ComposeServices []string `json:"compose_services" example:"[\"backend\", \"frontend\"]"`
	PostScripts     []string `json:"post_scripts" example:"[\"npm install\", \"npm run build\"]"`
}

// WorktreesResponse represents a list of worktrees
type WorktreesResponse struct {
	Worktrees []db.Worktree `json:"worktrees"`
	Total     int           `json:"total" example:"5"`
}

// WorktreeStatusResponse represents a worktree status update response
type WorktreeStatusResponse struct {
	Message string `json:"message" example:"Worktree started successfully"`
	ID      string `json:"id"`
	Status  string `json:"status" example:"running"`
}

// Service API models

// Service represents a global service
// Service represents a running service (e.g., database, cache)
type Service struct {
	ID          string    `json:"id"`
	Name        string    `json:"name" example:"postgres"`
	Type        string    `json:"type" example:"database"`
	Status      string    `json:"status" example:"running"`
	Port        int       `json:"port" example:"5432"`
	ContainerID string    `json:"container_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ServicesResponse represents a list of services
type ServicesResponse struct {
	Services []Service `json:"services"`
	Total    int       `json:"total" example:"3"`
}

// ServiceStatusResponse represents a service status update response
type ServiceStatusResponse struct {
	Message string `json:"message" example:"Service started successfully"`
	ID      string `json:"id"`
	Status  string `json:"status" example:"starting"`
}

// Config API models

// GitConfig represents git configuration
type GitConfig struct {
	DefaultBranchPrefix string `json:"default_branch_prefix" example:"feature/"`
	AutoFetch           bool   `json:"auto_fetch" example:"true"`
}

// ContainerConfig represents container configuration
type ContainerConfig struct {
	DefaultRuntime string `json:"default_runtime" example:"docker"`
	AutoStart      bool   `json:"auto_start" example:"true"`
}

// ConfigResponse represents the global configuration
type ConfigResponse struct {
	Storage   config.StorageConfig `json:"storage"`
	Git       GitConfig            `json:"git"`
	Container ContainerConfig      `json:"container"`
}

// Container API models

// ContainerResponse represents a container in API responses
type ContainerResponse struct {
	ID         string            `json:"id" example:"abc123"`
	Name       string            `json:"name" example:"vibeman-myapp-dev"`
	Image      string            `json:"image" example:"node:18-alpine"`
	Status     string            `json:"status" example:"running"`
	State      string            `json:"state" example:"Up 2 hours"`
	Ports      []string          `json:"ports" example:"[\"8080:3000\"]"`
	Repository string            `json:"repository" example:"myapp"`
	Worktree   string            `json:"worktree" example:"feature-auth"`
	Labels     map[string]string `json:"labels"`
	CreatedAt  string            `json:"created_at" example:"2023-01-01T12:00:00Z"`
}

// ContainersResponse represents a list of containers
type ContainersResponse struct {
	Containers []*ContainerResponse `json:"containers"`
	Total      int                  `json:"total" example:"5"`
}

// CreateContainerRequest represents a request to create a container
type CreateContainerRequest struct {
	Repository string            `json:"repository" validate:"required" example:"myapp"`
	Worktree   string            `json:"worktree" example:"feature-auth"`
	Image      string            `json:"image" validate:"required" example:"node:18-alpine"`
	Ports      []string          `json:"ports" example:"[\"8080:3000\"]"`
	Env        map[string]string `json:"env" example:"{\"NODE_ENV\":\"development\"}"`
	AutoStart  bool              `json:"auto_start" example:"true"`
}

// ContainerActionRequest represents a request to perform an action on a container
type ContainerActionRequest struct {
	Action string `json:"action" validate:"required,oneof=start stop restart" example:"start"`
}

// ContainerLogsResponse represents container logs
type ContainerLogsResponse struct {
	Logs      []string `json:"logs"`
	Timestamp string   `json:"timestamp" example:"2023-01-01T12:00:00Z"`
}

// Logs API models

// LogsResponse represents logs from worktrees or services
type LogsResponse struct {
	Logs      []string `json:"logs"`
	Source    string   `json:"source" example:"worktree" enum:"worktree,service,container"`
	ID        string   `json:"id" example:"worktree-123"`
	Timestamp string   `json:"timestamp" example:"2023-01-01T12:00:00Z"`
	Lines     int      `json:"lines" example:"50"`
}
