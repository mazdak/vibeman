package errors

import "fmt"

// Configuration Errors
func ConfigNotFound(path string) *VibemanError {
	return NewWithDetails(ErrConfigNotFound, "Configuration file not found", fmt.Sprintf("Path: %s", path))
}

func ConfigInvalid(reason string) *VibemanError {
	return NewWithDetails(ErrConfigInvalid, "Invalid configuration", reason)
}

func ConfigParseError(cause error) *VibemanError {
	return Wrap(ErrConfigParse, "Failed to parse configuration", cause)
}

func ConfigValidationError(field, reason string) *VibemanError {
	return NewWithDetails(ErrConfigValidation, "Configuration validation failed",
		fmt.Sprintf("Field: %s, Reason: %s", field, reason))
}

// Container Errors
func ContainerNotFound(id string) *VibemanError {
	return NewWithDetails(ErrContainerNotFound, "Container not found", fmt.Sprintf("ID: %s", id))
}

func ContainerCreateFailed(cause error) *VibemanError {
	return Wrap(ErrContainerCreateFailed, "Failed to create container", cause)
}

func ContainerStartFailed(id string, cause error) *VibemanError {
	return WrapWithDetails(ErrContainerStartFailed, "Failed to start container",
		fmt.Sprintf("Container ID: %s", id), cause)
}

func ContainerStopFailed(id string, cause error) *VibemanError {
	return WrapWithDetails(ErrContainerStopFailed, "Failed to stop container",
		fmt.Sprintf("Container ID: %s", id), cause)
}

func ContainerExecFailed(id string, command []string, cause error) *VibemanError {
	return WrapWithDetails(ErrContainerExecFailed, "Failed to execute command in container",
		fmt.Sprintf("Container ID: %s, Command: %v", id, command), cause)
}

func ContainerInvalidID(id string) *VibemanError {
	return NewWithDetails(ErrContainerInvalidID, "Invalid container ID",
		fmt.Sprintf("ID: %s", id))
}

// Service Errors
func ServiceNotFound(name string) *VibemanError {
	return NewWithDetails(ErrServiceNotFound, "Service not found", fmt.Sprintf("Service: %s", name))
}

func ServiceAlreadyRunning(name string) *VibemanError {
	return NewWithDetails(ErrServiceAlreadyRunning, "Service is already running",
		fmt.Sprintf("Service: %s", name))
}

func ServiceStartFailed(name string, cause error) *VibemanError {
	return WrapWithDetails(ErrServiceStartFailed, "Failed to start service",
		fmt.Sprintf("Service: %s", name), cause)
}

func ServiceStopFailed(name string, cause error) *VibemanError {
	return WrapWithDetails(ErrServiceStopFailed, "Failed to stop service",
		fmt.Sprintf("Service: %s", name), cause)
}

func ServiceHealthCheckFailed(name string, cause error) *VibemanError {
	return WrapWithDetails(ErrServiceHealthCheckFail, "Service health check failed",
		fmt.Sprintf("Service: %s", name), cause)
}

func ServiceDependencyFailed(service, dependency string, cause error) *VibemanError {
	return WrapWithDetails(ErrServiceDependencyFail, "Service dependency failed",
		fmt.Sprintf("Service: %s, Dependency: %s", service, dependency), cause)
}

// Git Errors
func GitRepoNotFound(path string) *VibemanError {
	return NewWithDetails(ErrGitRepoNotFound, "Git repository not found", fmt.Sprintf("Path: %s", path))
}

func GitCloneFailed(url string, cause error) *VibemanError {
	return WrapWithDetails(ErrGitCloneFailed, "Failed to clone repository",
		fmt.Sprintf("URL: %s", url), cause)
}

func GitWorktreeFailed(operation, path string, cause error) *VibemanError {
	return WrapWithDetails(ErrGitWorktreeFailed, "Git worktree operation failed",
		fmt.Sprintf("Operation: %s, Path: %s", operation, path), cause)
}

func GitBranchNotFound(branch string) *VibemanError {
	return NewWithDetails(ErrGitBranchNotFound, "Git branch not found", fmt.Sprintf("Branch: %s", branch))
}

func GitUncommittedChanges(path string) *VibemanError {
	return NewWithDetails(ErrGitUncommitted, "Repository has uncommitted changes",
		fmt.Sprintf("Path: %s", path))
}

func GitUnpushedCommits(path string) *VibemanError {
	return NewWithDetails(ErrGitUnpushed, "Repository has unpushed commits",
		fmt.Sprintf("Path: %s", path))
}

// Database Errors
func DatabaseConnectionError(cause error) *VibemanError {
	return Wrap(ErrDatabaseConnection, "Database connection failed", cause)
}

func DatabaseQueryError(query string, cause error) *VibemanError {
	return WrapWithDetails(ErrDatabaseQuery, "Database query failed",
		fmt.Sprintf("Query: %s", query), cause)
}

func DatabaseMigrationError(version string, cause error) *VibemanError {
	return WrapWithDetails(ErrDatabaseMigration, "Database migration failed",
		fmt.Sprintf("Version: %s", version), cause)
}

// Network/API Errors
func NetworkConnectionError(endpoint string, cause error) *VibemanError {
	return WrapWithDetails(ErrNetworkConnection, "Network connection failed",
		fmt.Sprintf("Endpoint: %s", endpoint), cause)
}

func APICallError(method, url string, cause error) *VibemanError {
	return WrapWithDetails(ErrAPICall, "API call failed",
		fmt.Sprintf("Method: %s, URL: %s", method, url), cause)
}

func AuthenticationFailed(reason string) *VibemanError {
	return NewWithDetails(ErrAuthFailed, "Authentication failed", reason)
}

func PermissionDenied(resource, action string) *VibemanError {
	return NewWithDetails(ErrPermissionDenied, "Permission denied",
		fmt.Sprintf("Resource: %s, Action: %s", resource, action))
}

// Validation Errors
func ValidationFailed(field, value, reason string) *VibemanError {
	return NewWithDetails(ErrValidationFailed, "Validation failed",
		fmt.Sprintf("Field: %s, Value: %s, Reason: %s", field, value, reason))
}

func InvalidInput(input, expected string) *VibemanError {
	return NewWithDetails(ErrInvalidInput, "Invalid input",
		fmt.Sprintf("Input: %s, Expected: %s", input, expected))
}

func InvalidPath(path, reason string) *VibemanError {
	return NewWithDetails(ErrInvalidPath, "Invalid path",
		fmt.Sprintf("Path: %s, Reason: %s", path, reason))
}

func InvalidPort(port interface{}, reason string) *VibemanError {
	return NewWithDetails(ErrInvalidPort, "Invalid port",
		fmt.Sprintf("Port: %v, Reason: %s", port, reason))
}

// Internal Errors
func InternalError(details string, cause error) *VibemanError {
	if cause != nil {
		return WrapWithDetails(ErrInternal, "Internal error", details, cause)
	}
	return NewWithDetails(ErrInternal, "Internal error", details)
}

func NotImplemented(feature string) *VibemanError {
	return NewWithDetails(ErrNotImplemented, "Feature not implemented",
		fmt.Sprintf("Feature: %s", feature))
}

func TimeoutError(operation string, duration interface{}) *VibemanError {
	return NewWithDetails(ErrTimeout, "Operation timed out",
		fmt.Sprintf("Operation: %s, Duration: %v", operation, duration))
}
