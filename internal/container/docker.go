package container

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// DockerRuntime implements ContainerRuntime for Docker
type DockerRuntime struct {
	executor CommandExecutor
}

// NewDockerRuntime creates a new Docker runtime
func NewDockerRuntime(executor CommandExecutor) *DockerRuntime {
	if executor == nil {
		executor = &DefaultCommandExecutor{}
	}
	return &DockerRuntime{
		executor: executor,
	}
}

// GetType returns the runtime type
func (r *DockerRuntime) GetType() RuntimeType {
	return RuntimeTypeDocker
}

// IsAvailable checks if Docker is available on the system
func (r *DockerRuntime) IsAvailable(ctx context.Context) bool {
	cmd := r.executor.CommandContext(ctx, "docker", "--version")
	return cmd.Run() == nil
}

// List returns all containers
func (r *DockerRuntime) List(ctx context.Context) ([]*Container, error) {
	cmd := r.executor.CommandContext(ctx, "docker", "ps", "-a", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	containers := []*Container{}
	if len(output) == 0 {
		return containers, nil
	}

	// Docker returns newline-separated JSON objects, not a JSON array
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var dockerContainer map[string]interface{}
		if err := json.Unmarshal([]byte(line), &dockerContainer); err != nil {
			// Skip malformed JSON lines
			continue
		}

		container := &Container{
			ID:      getStringField(dockerContainer, "ID"),
			Name:    strings.TrimPrefix(getStringField(dockerContainer, "Names"), "/"), // Docker prefixes names with /
			Image:   getStringField(dockerContainer, "Image"),
			Status:  getStringField(dockerContainer, "Status"),
			Command: getStringField(dockerContainer, "Command"),
		}

		// Parse additional fields from name (our naming convention: project-environment)
		if container.Name != "" {
			parts := strings.Split(container.Name, "-")
			if len(parts) >= 2 {
				container.Repository = parts[0]
				container.Environment = strings.Join(parts[1:], "-")
			} else {
				container.Repository = container.Name
			}
		}

		// Parse creation time if available
		if createdAt := getStringField(dockerContainer, "CreatedAt"); createdAt != "" {
			container.CreatedAt = createdAt
		}

		// Parse ports if available
		if ports := getStringField(dockerContainer, "Ports"); ports != "" {
			container.Ports = parseDockerPorts(ports)
		}

		// Get environment variables using docker inspect
		if envVars, err := r.getContainerEnvVars(ctx, container.ID); err == nil {
			container.EnvVars = envVars

			// Update project and environment from env vars if available
			if project, ok := envVars["VIBEMAN_REPOSITORY"]; ok && project != "" {
				container.Repository = project
			}
			if env, ok := envVars["VIBEMAN_ENV"]; ok && env != "" {
				container.Environment = env
			}
		}

		// Get labels to determine container type
		if labels, err := r.getContainerLabels(ctx, container.ID); err == nil {
			if containerType, ok := labels["vibeman.type"]; ok && containerType != "" {
				container.Type = containerType
			} else {
				// Default to "worktree" for backwards compatibility
				container.Type = "worktree"
			}
		}

		containers = append(containers, container)
	}

	return containers, nil
}

// Create creates a new container
func (r *DockerRuntime) Create(ctx context.Context, config *CreateConfig) (*Container, error) {
	if config.Name == "" {
		return nil, &ContainerError{
			Type:      ErrorTypeConfigError,
			Operation: "create",
			Message:   "container name is required",
		}
	}

	// Check if using docker-compose
	if config.ComposeFile != "" {
		// If compose file is specified, use compose (even if image is empty)
		return r.createFromCompose(ctx, config)
	}

	// Direct container creation
	if config.Image == "" {
		return nil, &ContainerError{
			Type:      ErrorTypeConfigError,
			Operation: "create",
			Message:   "container image is required",
		}
	}

	// Build docker run command
	args := []string{"run"}
	
	// Add interactive/TTY flags if requested
	if config.Interactive {
		args = append(args, "-it")
	} else {
		args = append(args, "-d")
	}
	
	args = append(args,
		"--name", config.Name,
		// Add labels for better identification
		"--label", fmt.Sprintf("vibeman.repository=%s", config.Repository),
		"--label", fmt.Sprintf("vibeman.environment=%s", config.Environment),
		"--label", "vibeman.managed=true",
		"--label", fmt.Sprintf("vibeman.type=%s", config.Type),
	)

	// Set working directory if specified
	if config.WorkingDir != "" {
		args = append(args, "-w", config.WorkingDir)
	}

	// Add environment variables
	for _, env := range config.EnvVars {
		args = append(args, "-e", env)
	}

	// Add volume mounts
	for _, volume := range config.Volumes {
		args = append(args, "-v", volume)
	}

	// Add port mappings
	for _, port := range config.Ports {
		args = append(args, "-p", port)
	}

	// Add image
	args = append(args, config.Image)
	
	// Add default command for alpine to keep it running
	if config.Image == "alpine:latest" || config.Image == "alpine" {
		args = append(args, "sh", "-c", "while true; do sleep 30; done")
	}

	cmd := r.executor.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check for specific error types
		outputStr := string(output)
		if strings.Contains(outputStr, "repository does not exist") || strings.Contains(outputStr, "pull access denied") {
			return nil, &ContainerError{
				Type:      ErrorTypeImageNotFound,
				Operation: "create",
				Message:   fmt.Sprintf("image not found: %s", config.Image),
				Underlying: fmt.Errorf("failed to create container: %w, output: %s", err, outputStr),
			}
		}
		if strings.Contains(outputStr, "container name") && strings.Contains(outputStr, "already in use") {
			return nil, &ContainerError{
				Type:      ErrorTypeConfigError,
				Operation: "create",
				Message:   fmt.Sprintf("container name %s already in use", config.Name),
				Underlying: fmt.Errorf("failed to create container: %w, output: %s", err, outputStr),
			}
		}
		return nil, &ContainerError{
			Type:      ErrorTypeUnknown,
			Operation: "create",
			Message:   "failed to create container",
			Underlying: fmt.Errorf("%w, output: %s", err, outputStr),
		}
	}

	containerID := strings.TrimSpace(string(output))

	return &Container{
		ID:          containerID,
		Name:        config.Name,
		Image:       config.Image,
		Status:      "Created",
		Repository:  config.Repository,
		Environment: config.Environment,
		Type:        config.Type,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}, nil
}

// createFromCompose creates a container using docker-compose
func (r *DockerRuntime) createFromCompose(ctx context.Context, config *CreateConfig) (*Container, error) {
	// Ensure compose file exists
	composeFile := config.ComposeFile
	if !filepath.IsAbs(composeFile) {
		// Handle relative paths - assume relative to current directory
		cwd, err := filepath.Abs(".")
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		composeFile = filepath.Join(cwd, composeFile)
	}

	// Use docker-compose to create and start the services
	// Use project-worktree as compose repository name (service will be appended by compose)
	composeRepositoryName := config.Name
	if config.Environment != "" {
		// For worktrees, use project-environment as compose project
		composeRepositoryName = fmt.Sprintf("%s-%s", config.Repository, config.Environment)
	} else {
		// For main, just use repository name
		composeRepositoryName = config.Repository
	}
	
	// Build docker-compose command
	args := []string{
		"compose",
		"-p", composeRepositoryName,
		"-f", composeFile,
		"up", "-d",
	}
	
	// Handle service selection
	if len(config.ComposeServices) > 0 {
		// Start only specified services
		args = append(args, "--no-deps") // Don't start linked services
		args = append(args, config.ComposeServices...)
	} else if config.ComposeService != "" {
		// Backward compatibility: single service
		args = append(args, "--no-deps")
		args = append(args, config.ComposeService)
	}
	// If no services specified, start all services (no --no-deps flag)

	cmd := r.executor.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to start compose service: %w, output: %s", err, string(output))
	}

	// Get container IDs for the started services
	args = []string{
		"compose",
		"-p", composeRepositoryName,
		"-f", composeFile,
		"ps", "-q",
	}
	
	// Add specific service if using backward compatibility mode
	if config.ComposeService != "" && len(config.ComposeServices) == 0 {
		args = append(args, config.ComposeService)
	} else if len(config.ComposeServices) == 1 {
		// If only one service specified, get that specific container
		args = append(args, config.ComposeServices[0])
	}
	// Otherwise get all containers for the compose project

	cmd = r.executor.CommandContext(ctx, "docker", args...)
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get container IDs: %w", err)
	}

	containerIDs := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(containerIDs) == 0 || containerIDs[0] == "" {
		return nil, fmt.Errorf("no containers found for compose project")
	}
	
	// Use the first container ID as the primary container
	containerID := containerIDs[0]

	// Get container info
	container, err := r.GetInfo(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	// Update with our project info
	container.Repository = config.Repository
	container.Environment = config.Environment

	return container, nil
}

// Start starts a container
func (r *DockerRuntime) Start(ctx context.Context, containerID string) error {
	cmd := r.executor.CommandContext(ctx, "docker", "start", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "No such container") {
			return &ContainerError{
				Type:      ErrorTypeContainerNotFound,
				Operation: "start",
				Message:   fmt.Sprintf("container not found: %s", containerID),
				Underlying: fmt.Errorf("failed to start container: %w, output: %s", err, outputStr),
			}
		}
		return &ContainerError{
			Type:      ErrorTypeUnknown,
			Operation: "start",
			Message:   "failed to start container",
			Underlying: fmt.Errorf("%w, output: %s", err, outputStr),
		}
	}
	return nil
}

// Stop stops a container
func (r *DockerRuntime) Stop(ctx context.Context, containerID string) error {
	cmd := r.executor.CommandContext(ctx, "docker", "stop", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop container: %w, output: %s", err, string(output))
	}
	return nil
}

// Remove removes a container
func (r *DockerRuntime) Remove(ctx context.Context, containerID string) error {
	cmd := r.executor.CommandContext(ctx, "docker", "rm", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove container: %w, output: %s", err, string(output))
	}
	return nil
}

// Exec executes a command in a container
func (r *DockerRuntime) Exec(ctx context.Context, containerID string, command []string) ([]byte, error) {
	args := append([]string{"exec", containerID}, command...)
	cmd := r.executor.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, &ContainerError{
			Type:        parseDockerError(string(output), err),
			Operation:   "exec",
			ContainerID: containerID,
			Message:     "failed to exec in container",
			Underlying:  err,
			Output:      string(output),
		}
	}
	return output, nil
}

// Logs returns logs from a container
func (r *DockerRuntime) Logs(ctx context.Context, containerID string, follow bool) ([]byte, error) {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, containerID)

	cmd := r.executor.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w, output: %s", err, string(output))
	}
	return output, nil
}

// GetInfo returns detailed information about a container
func (r *DockerRuntime) GetInfo(ctx context.Context, containerID string) (*Container, error) {
	cmd := r.executor.CommandContext(ctx, "docker", "inspect", containerID)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Docker inspect returns an array of container objects
	var dockerContainers []map[string]interface{}
	if err := json.Unmarshal(output, &dockerContainers); err != nil {
		return nil, fmt.Errorf("failed to parse container info: %w", err)
	}

	if len(dockerContainers) == 0 {
		return nil, fmt.Errorf("container not found")
	}

	dockerContainer := dockerContainers[0]

	// Extract container info from Docker's complex structure
	container := &Container{
		ID:     getStringField(dockerContainer, "Id"),
		Name:   strings.TrimPrefix(getStringField(dockerContainer, "Name"), "/"),
		Status: getDockerStatus(dockerContainer),
	}

	// Get config section for image and command
	if config, ok := dockerContainer["Config"].(map[string]interface{}); ok {
		container.Image = getStringField(config, "Image")
		if cmd, ok := config["Cmd"].([]interface{}); ok && len(cmd) > 0 {
			cmdParts := make([]string, len(cmd))
			for i, part := range cmd {
				if str, ok := part.(string); ok {
					cmdParts[i] = str
				}
			}
			container.Command = strings.Join(cmdParts, " ")
		}

		// Get environment variables
		if envVars, ok := config["Env"].([]interface{}); ok {
			container.EnvVars = parseDockerEnvVars(envVars)

			// Update project and environment from env vars if available
			if project, ok := container.EnvVars["VIBEMAN_REPOSITORY"]; ok && project != "" {
				container.Repository = project
			}
			if env, ok := container.EnvVars["VIBEMAN_ENV"]; ok && env != "" {
				container.Environment = env
			}
		}
	}

	// Parse additional fields from name (our naming convention: project-environment)
	// Only use name parsing as fallback if env vars are not set
	if container.Repository == "" && container.Name != "" {
		parts := strings.Split(container.Name, "-")
		if len(parts) >= 2 {
			container.Repository = parts[0]
			container.Environment = strings.Join(parts[1:], "-")
		} else {
			container.Repository = container.Name
		}
	}

	// Get creation time
	if created := getStringField(dockerContainer, "Created"); created != "" {
		container.CreatedAt = created
	}

	// Get port mappings
	if networkSettings, ok := dockerContainer["NetworkSettings"].(map[string]interface{}); ok {
		if ports, ok := networkSettings["Ports"].(map[string]interface{}); ok {
			container.Ports = parseDockerInspectPorts(ports)
		}
	}

	return container, nil
}

// Helper functions

// getDockerStatus extracts status from Docker container state
func getDockerStatus(dockerContainer map[string]interface{}) string {
	if state, ok := dockerContainer["State"].(map[string]interface{}); ok {
		if status := getStringField(state, "Status"); status != "" {
			return status
		}
	}
	return "unknown"
}

// parseDockerPorts parses Docker port string format (e.g., "0.0.0.0:8080->80/tcp")
func parseDockerPorts(portsStr string) map[string]string {
	portMap := make(map[string]string)

	if portsStr == "" {
		return portMap
	}

	// Split by comma for multiple port mappings
	mappings := strings.Split(portsStr, ", ")
	for _, mapping := range mappings {
		// Parse format like "0.0.0.0:8080->80/tcp"
		if strings.Contains(mapping, "->") {
			parts := strings.Split(mapping, "->")
			if len(parts) == 2 {
				hostPart := parts[0]
				containerPart := strings.Split(parts[1], "/")[0] // Remove /tcp or /udp

				// Extract port from host part (may include IP)
				if strings.Contains(hostPart, ":") {
					hostPortParts := strings.Split(hostPart, ":")
					if len(hostPortParts) > 0 {
						hostPort := hostPortParts[len(hostPortParts)-1]
						portMap[containerPart] = hostPort
					}
				}
			}
		}
	}

	return portMap
}

// getContainerLabels gets labels from a container using docker inspect
func (r *DockerRuntime) getContainerLabels(ctx context.Context, containerID string) (map[string]string, error) {
	cmd := r.executor.CommandContext(ctx, "docker", "inspect", containerID, "--format", "{{json .Config.Labels}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get container labels: %w", err)
	}

	var labels map[string]string
	if err := json.Unmarshal(output, &labels); err != nil {
		return nil, fmt.Errorf("failed to parse container labels: %w", err)
	}

	return labels, nil
}

// getContainerEnvVars gets environment variables for a container using docker inspect
func (r *DockerRuntime) getContainerEnvVars(ctx context.Context, containerID string) (map[string]string, error) {
	cmd := r.executor.CommandContext(ctx, "docker", "inspect", containerID, "--format", "{{json .Config.Env}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get container env vars: %w", err)
	}

	var envArray []string
	if err := json.Unmarshal(output, &envArray); err != nil {
		return nil, fmt.Errorf("failed to parse env vars: %w", err)
	}

	return parseEnvArray(envArray), nil
}

// parseDockerEnvVars parses environment variables from docker inspect Config.Env
func parseDockerEnvVars(envVars []interface{}) map[string]string {
	envArray := make([]string, 0, len(envVars))
	for _, env := range envVars {
		if envStr, ok := env.(string); ok {
			envArray = append(envArray, envStr)
		}
	}
	return parseEnvArray(envArray)
}

// parseEnvArray parses an array of KEY=VALUE strings into a map
func parseEnvArray(envArray []string) map[string]string {
	envMap := make(map[string]string)
	for _, env := range envArray {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	return envMap
}

// parseDockerInspectPorts parses port mappings from docker inspect output
func parseDockerInspectPorts(ports map[string]interface{}) map[string]string {
	portMap := make(map[string]string)

	for containerPort, hostBindings := range ports {
		// Remove protocol suffix (e.g., "80/tcp" -> "80")
		cleanContainerPort := strings.Split(containerPort, "/")[0]

		if bindings, ok := hostBindings.([]interface{}); ok && len(bindings) > 0 {
			// Take the first binding
			if binding, ok := bindings[0].(map[string]interface{}); ok {
				if hostPort := getStringField(binding, "HostPort"); hostPort != "" {
					portMap[cleanContainerPort] = hostPort
				}
			}
		}
	}

	return portMap
}

// getStringField safely extracts a string field from a map
func getStringField(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
