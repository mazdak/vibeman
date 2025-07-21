package container

import (
	"fmt"
	"strings"
)

// ErrorType represents the type of container error
type ErrorType string

const (
	// ErrorTypeRuntimeNotFound indicates the container runtime is not available
	ErrorTypeRuntimeNotFound ErrorType = "runtime_not_found"
	// ErrorTypeContainerNotFound indicates the container was not found
	ErrorTypeContainerNotFound ErrorType = "container_not_found"
	// ErrorTypeImageNotFound indicates the container image was not found
	ErrorTypeImageNotFound ErrorType = "image_not_found"
	// ErrorTypePermissionDenied indicates a permission error
	ErrorTypePermissionDenied ErrorType = "permission_denied"
	// ErrorTypeNetworkError indicates a network-related error
	ErrorTypeNetworkError ErrorType = "network_error"
	// ErrorTypeVolumeError indicates a volume-related error
	ErrorTypeVolumeError ErrorType = "volume_error"
	// ErrorTypeConfigError indicates a configuration error
	ErrorTypeConfigError ErrorType = "config_error"
	// ErrorTypeExecError indicates an error during command execution
	ErrorTypeExecError ErrorType = "exec_error"
	// ErrorTypeUnknown indicates an unknown error
	ErrorTypeUnknown ErrorType = "unknown"
)

// ContainerError represents a detailed container operation error
type ContainerError struct {
	Type        ErrorType
	Operation   string
	ContainerID string
	Message     string
	Underlying  error
	Output      string // stdout/stderr from the command
}

// Error implements the error interface
func (e *ContainerError) Error() string {
	parts := []string{e.Message}
	
	if e.ContainerID != "" {
		parts = append(parts, fmt.Sprintf("container=%s", e.ContainerID))
	}
	
	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation=%s", e.Operation))
	}
	
	if e.Output != "" {
		// Clean up the output for display
		output := strings.TrimSpace(e.Output)
		if len(output) > 200 {
			output = output[:200] + "..."
		}
		parts = append(parts, fmt.Sprintf("output=%s", output))
	}
	
	if e.Underlying != nil {
		parts = append(parts, fmt.Sprintf("cause=%v", e.Underlying))
	}
	
	return strings.Join(parts, ", ")
}

// Unwrap returns the underlying error
func (e *ContainerError) Unwrap() error {
	return e.Underlying
}

// IsRetryable returns true if the error might be resolved by retrying
func (e *ContainerError) IsRetryable() bool {
	switch e.Type {
	case ErrorTypeNetworkError, ErrorTypeVolumeError:
		return true
	default:
		return false
	}
}

// NewContainerError creates a new ContainerError
func NewContainerError(errType ErrorType, operation string, message string, underlying error) *ContainerError {
	return &ContainerError{
		Type:       errType,
		Operation:  operation,
		Message:    message,
		Underlying: underlying,
	}
}

// parseDockerError attempts to determine the error type from Docker output
func parseDockerError(output string, err error) ErrorType {
	outputLower := strings.ToLower(output)
	errStr := ""
	if err != nil {
		errStr = strings.ToLower(err.Error())
	}
	
	combined := outputLower + " " + errStr
	
	// Check for specific error patterns
	switch {
	case strings.Contains(combined, "no such container"):
		return ErrorTypeContainerNotFound
	case strings.Contains(combined, "no such image") || strings.Contains(combined, "pull access denied"):
		return ErrorTypeImageNotFound
	case strings.Contains(combined, "permission denied") || strings.Contains(combined, "access denied"):
		return ErrorTypePermissionDenied
	case strings.Contains(combined, "network") || strings.Contains(combined, "port is already allocated"):
		return ErrorTypeNetworkError
	case strings.Contains(combined, "no such volume") || strings.Contains(combined, "volume"):
		return ErrorTypeVolumeError
	case strings.Contains(combined, "docker daemon") || strings.Contains(combined, "docker: command not found"):
		return ErrorTypeRuntimeNotFound
	default:
		return ErrorTypeUnknown
	}
}