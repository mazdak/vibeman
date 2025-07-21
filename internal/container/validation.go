package container

import (
	"vibeman/internal/validation"
)

// validateContainerID validates a container ID or name to prevent injection
func validateContainerID(id string) error {
	return validation.ContainerID(id)
}

// validateShellCommand validates a shell command to prevent injection
func validateShellCommand(shell string) error {
	return validation.ShellCommand(shell)
}

// validatePath validates and cleans a file path
func validatePath(path string) (string, error) {
	return validation.Path(path)
}

// validateEnvironmentVar validates environment variable format
func validateEnvironmentVar(envVar string) error {
	return validation.EnvironmentVariable(envVar)
}

// validatePortMapping validates port mapping format
func validatePortMapping(port string) error {
	return validation.PortMapping(port)
}

// sanitizeCommand escapes shell arguments to prevent injection
func sanitizeCommand(args []string) []string {
	return validation.SanitizeCommandArgs(args)
}

// shellEscape escapes a string for safe use in shell commands
func shellEscape(s string) string {
	return validation.ShellEscape(s)
}
