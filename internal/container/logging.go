package container

import (
	"vibeman/internal/logger"
)

// LogContainerError logs a container error with structured fields
func LogContainerError(err error, operation string) {
	if err == nil {
		return
	}
	
	fields := logger.Fields{
		"operation": operation,
	}
	
	// If it's a ContainerError, extract additional fields
	if containerErr, ok := err.(*ContainerError); ok {
		fields["error_type"] = string(containerErr.Type)
		if containerErr.ContainerID != "" {
			fields["container_id"] = containerErr.ContainerID
		}
		if containerErr.Output != "" && len(containerErr.Output) < 1000 {
			fields["docker_output"] = containerErr.Output
		}
		if containerErr.IsRetryable() {
			fields["retryable"] = true
		}
	}
	
	logger.WithFields(fields).WithError(err).Error("Container operation failed")
}

// LogContainerWarning logs a container warning with structured fields
func LogContainerWarning(err error, operation string) {
	if err == nil {
		return
	}
	
	fields := logger.Fields{
		"operation": operation,
	}
	
	// If it's a ContainerError, extract additional fields
	if containerErr, ok := err.(*ContainerError); ok {
		fields["error_type"] = string(containerErr.Type)
		if containerErr.ContainerID != "" {
			fields["container_id"] = containerErr.ContainerID
		}
	}
	
	logger.WithFields(fields).WithError(err).Warn("Container operation warning")
}