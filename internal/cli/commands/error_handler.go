package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"
	
	"vibeman/internal/container"
	"vibeman/internal/logger"
)

// HandleError processes errors and provides user-friendly output
func HandleError(err error) error {
	if err == nil {
		return nil
	}
	
	// Check if it's a container error
	var containerErr *container.ContainerError
	if errors.As(err, &containerErr) {
		errHandler := container.NewErrorHandler()
		userMessage := errHandler.GetUserMessage(containerErr)
		
		// Log the full error for debugging
		logger.WithError(err).Debug("Container operation failed")
		
		// Return user-friendly message
		return fmt.Errorf("%s", userMessage)
	}
	
	// Check for common error patterns and provide helpful messages
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "permission denied"):
		return fmt.Errorf("%v\n\nTip: You may need elevated permissions. Try running with sudo or check file permissions.", err)
		
	case strings.Contains(errStr, "no such file or directory"):
		return fmt.Errorf("%v\n\nTip: Check if the path exists and is accessible.", err)
		
	case strings.Contains(errStr, "repository not found"):
		return fmt.Errorf("%v\n\nTip: Use 'vibeman repo list' to see available repositories.", err)
		
	case strings.Contains(errStr, "worktree not found"):
		return fmt.Errorf("%v\n\nTip: Use 'vibeman worktree list' to see available worktrees.", err)
		
	case strings.Contains(errStr, "git"):
		return fmt.Errorf("%v\n\nTip: Ensure you have git installed and configured.", err)
		
	default:
		return err
	}
}

// ExitOnError handles errors consistently across CLI commands
func ExitOnError(err error) {
	if err == nil {
		return
	}
	
	// Process the error for better user experience
	processedErr := HandleError(err)
	
	// Print the error message
	fmt.Fprintf(os.Stderr, "Error: %v\n", processedErr)
	
	// Exit with appropriate code
	var containerErr *container.ContainerError
	if errors.As(err, &containerErr) {
		// Use specific exit codes for different error types
		switch containerErr.Type {
		case container.ErrorTypeRuntimeNotFound:
			os.Exit(127) // Command not found
		case container.ErrorTypePermissionDenied:
			os.Exit(126) // Permission denied
		case container.ErrorTypeContainerNotFound, container.ErrorTypeImageNotFound:
			os.Exit(2) // No such file or directory
		default:
			os.Exit(1) // General error
		}
	}
	
	os.Exit(1)
}