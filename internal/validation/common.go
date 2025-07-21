package validation

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"vibeman/internal/errors"
)

var (
	// containerIDRegex validates container IDs and names
	containerIDRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

	// shellCommandRegex validates shell commands
	shellCommandRegex = regexp.MustCompile(`^(/bin/)?(bash|sh|zsh|fish|dash)$`)

	// portRegex validates port mappings in HOST:CONTAINER format
	portRegex = regexp.MustCompile(`^\d{1,5}:\d{1,5}$`)

	// envVarKeyRegex validates environment variable keys
	envVarKeyRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	// safeStringRegex matches strings that are safe for shell use without escaping
	safeStringRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-./=]+$`)
)

// ContainerID validates a container ID or name to prevent injection
func ContainerID(id string) error {
	if id == "" {
		return errors.ValidationFailed("container_id", id, "cannot be empty")
	}

	if len(id) > 255 {
		return errors.ValidationFailed("container_id", id, "too long (max 255 characters)")
	}

	if !containerIDRegex.MatchString(id) {
		return errors.ContainerInvalidID(id)
	}

	return nil
}

// ShellCommand validates a shell command to prevent injection
func ShellCommand(shell string) error {
	if shell == "" {
		return nil // Empty means default shell
	}

	if !shellCommandRegex.MatchString(shell) {
		return errors.ValidationFailed("shell_command", shell, "must be a valid shell path (/bin/bash, /bin/sh, etc.)")
	}

	return nil
}

// Path validates and cleans a file path to prevent traversal attacks
func Path(path string) (string, error) {
	if path == "" {
		return "", errors.InvalidPath(path, "cannot be empty")
	}

	// Clean the path to prevent traversal
	cleaned := filepath.Clean(path)

	// Check for path traversal attempts by checking if the cleaned path
	// tries to go outside the current directory hierarchy
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." || strings.Contains(cleaned, "/../") {
		return "", errors.InvalidPath(path, "path traversal detected")
	}

	// Also check the original path for obvious traversal attempts
	if strings.Contains(path, "../") {
		return "", errors.InvalidPath(path, "path traversal detected")
	}

	return cleaned, nil
}

// EnvironmentVariable validates environment variable format (KEY=VALUE)
func EnvironmentVariable(envVar string) error {
	parts := strings.SplitN(envVar, "=", 2)
	if len(parts) != 2 {
		return errors.ValidationFailed("environment_variable", envVar, "must be in KEY=VALUE format")
	}

	key := parts[0]
	if key == "" {
		return errors.ValidationFailed("environment_variable", envVar, "key cannot be empty")
	}

	// Validate key format (alphanumeric and underscore)
	if !envVarKeyRegex.MatchString(key) {
		return errors.ValidationFailed("environment_variable_key", key, "must contain only letters, numbers, and underscores")
	}

	return nil
}

// PortMapping validates port mapping format (HOST:CONTAINER)
func PortMapping(port string) error {
	if !portRegex.MatchString(port) {
		return errors.InvalidPort(port, "must be in HOST:CONTAINER format")
	}

	parts := strings.Split(port, ":")
	hostPort := parts[0]
	containerPort := parts[1]

	// Validate port ranges
	for _, p := range []string{hostPort, containerPort} {
		portNum, err := strconv.Atoi(p)
		if err != nil {
			return errors.InvalidPort(p, "must be a valid number")
		}

		if portNum < 1 || portNum > 65535 {
			return errors.InvalidPort(portNum, "must be between 1 and 65535")
		}
	}

	return nil
}

// PortNumber validates a single port number
func PortNumber(port int) error {
	if port <= 0 || port > 65535 {
		return errors.InvalidPort(port, "must be between 1 and 65535")
	}
	return nil
}

// IsValidPortMapping checks if a port mapping follows the HOST:CONTAINER format
func IsValidPortMapping(mapping string) bool {
	parts := strings.Split(mapping, ":")
	if len(parts) != 2 {
		return false
	}

	// Check both parts are valid port numbers
	for _, part := range parts {
		if port, err := strconv.Atoi(part); err != nil || port <= 0 || port > 65535 {
			return false
		}
	}

	return true
}

// IsReservedPortName checks if a port name is reserved
func IsReservedPortName(name string) bool {
	reserved := map[string]bool{
		"system":   true,
		"reserved": true,
		"admin":    true,
		"root":     true,
	}
	return reserved[strings.ToLower(name)]
}

// ProcessCommand validates a process command for safety
func ProcessCommand(command string) error {
	// Basic validation - command should not be just whitespace
	if strings.TrimSpace(command) == "" {
		return errors.ValidationFailed("process_command", command, "cannot be empty or only whitespace")
	}

	// Check for dangerous commands that might cause issues
	dangerousCommands := []string{"rm -rf", "sudo", "chmod 777", "chown", "killall"}
	lowerCommand := strings.ToLower(command)

	for _, dangerous := range dangerousCommands {
		if strings.Contains(lowerCommand, dangerous) {
			return errors.ValidationFailed("process_command", command, "potentially dangerous command detected: "+dangerous)
		}
	}

	return nil
}

// NonEmptyString validates that a string is not empty or only whitespace
func NonEmptyString(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.ValidationFailed("string", s, "cannot be empty or only whitespace")
	}
	return nil
}

// ShellEscape escapes a string for safe use in shell commands
func ShellEscape(s string) string {
	// If the string is simple (alphanumeric + safe chars), return as-is
	if safeStringRegex.MatchString(s) {
		return s
	}

	// Otherwise, wrap in single quotes and escape any single quotes
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// SanitizeCommandArgs escapes shell arguments to prevent injection
func SanitizeCommandArgs(args []string) []string {
	sanitized := make([]string, len(args))
	for i, arg := range args {
		// Escape special shell characters
		sanitized[i] = ShellEscape(arg)
	}
	return sanitized
}
