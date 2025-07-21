package container

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ComposeService represents a service from docker-compose.yml
type ComposeService struct {
	Image       string            `yaml:"image"`
	Environment []string          `yaml:"environment"`
	Volumes     []string          `yaml:"volumes"`
	Ports       []string          `yaml:"ports"`
	Command     string            `yaml:"command"`
	WorkingDir  string            `yaml:"working_dir"`
	DependsOn   []string          `yaml:"depends_on"`
	Labels      map[string]string `yaml:"labels"`
}

// ComposeFile represents a simplified docker-compose.yml structure
type ComposeFile struct {
	Version  string                    `yaml:"version"`
	Services map[string]ComposeService `yaml:"services"`
}

// ReadComposeFile reads a docker-compose.yml file and returns its contents
func ReadComposeFile(path string) (*ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	return &compose, nil
}

// ConvertComposeToCreateConfig converts a compose service to CreateConfig
func ConvertComposeToCreateConfig(name string, service ComposeService) *CreateConfig {
	return &CreateConfig{
		Name:       name,
		Image:      service.Image,
		WorkingDir: service.WorkingDir,
		EnvVars:    service.Environment,
		Volumes:    service.Volumes,
		Ports:      service.Ports,
	}
}

// ImportFromCompose creates containers from a docker-compose.yml file
func (m *Manager) ImportFromCompose(ctx context.Context, composePath string, repositoryName string) error {
	compose, err := ReadComposeFile(composePath)
	if err != nil {
		return fmt.Errorf("failed to read compose file: %w", err)
	}

	runtime, err := m.getRuntime(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container runtime: %w", err)
	}

	// Only import if using Docker runtime
	if runtime.GetType() != RuntimeTypeDocker {
		return fmt.Errorf("compose import is only supported with Docker runtime")
	}

	// Create containers for each service
	for serviceName, service := range compose.Services {
		containerName := fmt.Sprintf("%s-%s", repositoryName, serviceName)

		config := ConvertComposeToCreateConfig(containerName, service)
		config.Repository = repositoryName
		config.Environment = serviceName

		_, err := runtime.Create(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to create container for service %s: %w", serviceName, err)
		}
	}

	return nil
}
