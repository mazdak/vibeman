package container

import (
	"fmt"
	"strings"
)

// ErrorHandler provides user-friendly error messages and recovery suggestions
type ErrorHandler struct{}

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{}
}

// GetUserMessage returns a user-friendly error message with recovery suggestions
func (h *ErrorHandler) GetUserMessage(err error) string {
	containerErr, ok := err.(*ContainerError)
	if !ok {
		return err.Error()
	}
	
	var message strings.Builder
	message.WriteString(containerErr.Message)
	
	// Add specific guidance based on error type
	switch containerErr.Type {
	case ErrorTypeRuntimeNotFound:
		message.WriteString("\n\nPossible solutions:")
		message.WriteString("\n• Ensure Docker is installed: https://docs.docker.com/get-docker/")
		message.WriteString("\n• Check if Docker daemon is running: 'docker ps'")
		message.WriteString("\n• On macOS, ensure Docker Desktop is running")
		
	case ErrorTypeImageNotFound:
		message.WriteString("\n\nPossible solutions:")
		message.WriteString("\n• Check if the image name is correct")
		message.WriteString("\n• Try pulling the image manually: 'docker pull <image>'")
		message.WriteString("\n• Verify you have access to the registry")
		
	case ErrorTypePermissionDenied:
		message.WriteString("\n\nPossible solutions:")
		message.WriteString("\n• Add your user to the docker group: 'sudo usermod -aG docker $USER'")
		message.WriteString("\n• Log out and back in for group changes to take effect")
		message.WriteString("\n• On Linux, you may need to use sudo")
		
	case ErrorTypeNetworkError:
		if strings.Contains(containerErr.Output, "port is already allocated") {
			message.WriteString("\n\nPort conflict detected. Possible solutions:")
			message.WriteString("\n• Stop the container using the port: 'docker ps' to find it")
			message.WriteString("\n• Use a different port in your configuration")
		} else {
			message.WriteString("\n\nNetwork issue detected. Possible solutions:")
			message.WriteString("\n• Check your network connectivity")
			message.WriteString("\n• Verify Docker network settings")
			message.WriteString("\n• Try restarting Docker")
		}
		
	case ErrorTypeVolumeError:
		message.WriteString("\n\nVolume issue detected. Possible solutions:")
		message.WriteString("\n• Check if the host path exists and is accessible")
		message.WriteString("\n• Verify file permissions on the host path")
		message.WriteString("\n• Try using absolute paths for volumes")
		
	case ErrorTypeContainerNotFound:
		message.WriteString("\n\nContainer not found. Possible solutions:")
		message.WriteString("\n• Check if the container ID or name is correct")
		message.WriteString("\n• List all containers: 'docker ps -a'")
		message.WriteString("\n• The container may have been removed")
	}
	
	// Add the actual error output if available and not too long
	if containerErr.Output != "" && containerErr.Type != ErrorTypeUnknown {
		cleaned := strings.TrimSpace(containerErr.Output)
		if len(cleaned) > 0 && len(cleaned) < 500 {
			message.WriteString("\n\nDocker output:\n")
			message.WriteString(cleaned)
		}
	}
	
	return message.String()
}

// IsRecoverable returns true if the error might be resolved by user action
func (h *ErrorHandler) IsRecoverable(err error) bool {
	containerErr, ok := err.(*ContainerError)
	if !ok {
		return false
	}
	
	switch containerErr.Type {
	case ErrorTypeRuntimeNotFound, ErrorTypeImageNotFound, 
	     ErrorTypePermissionDenied, ErrorTypeNetworkError, 
	     ErrorTypeVolumeError:
		return true
	default:
		return false
	}
}

// ShouldRetry returns true if the operation should be retried
func (h *ErrorHandler) ShouldRetry(err error) bool {
	containerErr, ok := err.(*ContainerError)
	if !ok {
		return false
	}
	
	return containerErr.IsRetryable()
}

// GetExitCode extracts the exit code from a container error if available
func (h *ErrorHandler) GetExitCode(err error) (int, bool) {
	containerErr, ok := err.(*ContainerError)
	if !ok || containerErr.Output == "" {
		return 0, false
	}
	
	// Try to parse exit code from output
	// Docker often includes "exit status N" in error messages
	if idx := strings.Index(containerErr.Output, "exit status "); idx != -1 {
		var code int
		if _, err := fmt.Sscanf(containerErr.Output[idx:], "exit status %d", &code); err == nil {
			return code, true
		}
	}
	
	return 0, false
}