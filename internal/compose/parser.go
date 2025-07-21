package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeFile represents a docker-compose.yaml file
type ComposeFile struct {
	Version  string                     `yaml:"version"`
	Services map[string]*ComposeService `yaml:"services"`
	Networks map[string]*ComposeNetwork `yaml:"networks"`
	Volumes  map[string]*ComposeVolume  `yaml:"volumes"`
}

// ComposeService represents a service in docker-compose.yaml
type ComposeService struct {
	Name          string        // Service name from compose
	Image         string        `yaml:"image"`
	Build         *BuildConfig  `yaml:"build"`
	Command       StringOrSlice `yaml:"command"`
	Entrypoint    StringOrSlice `yaml:"entrypoint"`
	WorkingDir    string        `yaml:"working_dir"`
	Environment   Environment   `yaml:"environment"`
	EnvFile       StringOrSlice `yaml:"env_file"`
	Volumes       []string      `yaml:"volumes"`
	Ports         []string      `yaml:"ports"`
	Networks      StringOrSlice `yaml:"networks"`
	DependsOn     StringOrSlice `yaml:"depends_on"`
	Deploy        *DeployConfig `yaml:"deploy"`
	MemLimit      string        `yaml:"mem_limit"`
	CPUs          string        `yaml:"cpus"`
	ContainerName string        `yaml:"container_name"`
}

// BuildConfig represents build configuration
type BuildConfig struct {
	Context    string            `yaml:"context"`
	Dockerfile string            `yaml:"dockerfile"`
	Args       map[string]string `yaml:"args"`
}

// DeployConfig represents deployment configuration
type DeployConfig struct {
	Resources *Resources `yaml:"resources"`
}

// Resources represents resource constraints
type Resources struct {
	Limits       *ResourceLimits       `yaml:"limits"`
	Reservations *ResourceReservations `yaml:"reservations"`
}

// ResourceLimits represents resource limits
type ResourceLimits struct {
	CPUs   string `yaml:"cpus"`
	Memory string `yaml:"memory"`
}

// ResourceReservations represents resource reservations
type ResourceReservations struct {
	CPUs   string `yaml:"cpus"`
	Memory string `yaml:"memory"`
}

// ComposeNetwork represents a network definition
type ComposeNetwork struct {
	Driver string `yaml:"driver"`
}

// ComposeVolume represents a volume definition
type ComposeVolume struct {
	Driver string `yaml:"driver"`
}

// VolumeMount represents a parsed volume mount
type VolumeMount struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

// PortMapping represents a parsed port mapping
type PortMapping struct {
	HostPort      int
	ContainerPort int
	Protocol      string // tcp/udp
}

// ParsedService represents a service with parsed values
type ParsedService struct {
	Name          string
	Image         string
	Command       []string
	WorkingDir    string
	Environment   map[string]string
	Volumes       []VolumeMount
	Ports         []PortMapping
	CPUs          float64
	Memory        string
	ContainerName string
}

// StringOrSlice can be either a string or a slice of strings
type StringOrSlice []string

func (s *StringOrSlice) UnmarshalYAML(value *yaml.Node) error {
	var multi []string
	err := value.Decode(&multi)
	if err != nil {
		var single string
		err := value.Decode(&single)
		if err != nil {
			return err
		}
		*s = []string{single}
	} else {
		*s = multi
	}
	return nil
}

// Environment can be either a map or a slice of KEY=VALUE strings
type Environment map[string]string

func (e *Environment) UnmarshalYAML(value *yaml.Node) error {
	*e = make(map[string]string)

	// Try to decode as a map first
	var envMap map[string]string
	if err := value.Decode(&envMap); err == nil {
		for k, v := range envMap {
			(*e)[k] = v
		}
		return nil
	}

	// Try to decode as a slice
	var envSlice []string
	if err := value.Decode(&envSlice); err == nil {
		for _, env := range envSlice {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				(*e)[parts[0]] = parts[1]
			} else if len(parts) == 1 {
				// Environment variable without value
				(*e)[parts[0]] = ""
			}
		}
		return nil
	}

	return fmt.Errorf("environment must be a map or slice of strings")
}

// ParseComposeFile reads and parses a docker-compose.yaml file
func ParseComposeFile(path string) (*ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading compose file: %w", err)
	}

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("parsing compose file: %w", err)
	}

	// Set service names
	for name, service := range compose.Services {
		service.Name = name
	}

	return &compose, nil
}

// ParseService converts a ComposeService to a ParsedService with resolved values
func (c *ComposeFile) ParseService(serviceName string, baseDir string) (*ParsedService, error) {
	service, ok := c.Services[serviceName]
	if !ok {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	parsed := &ParsedService{
		Name:          serviceName,
		Image:         service.Image,
		Command:       service.Command,
		WorkingDir:    service.WorkingDir,
		Environment:   make(map[string]string),
		ContainerName: service.ContainerName,
	}

	// If no container name specified, use service name
	if parsed.ContainerName == "" {
		parsed.ContainerName = serviceName
	}

	// Parse environment
	for k, v := range service.Environment {
		parsed.Environment[k] = v
	}

	// Parse volumes
	for _, vol := range service.Volumes {
		mount, err := parseVolumeString(vol, baseDir)
		if err != nil {
			return nil, fmt.Errorf("parsing volume %s: %w", vol, err)
		}
		parsed.Volumes = append(parsed.Volumes, mount)
	}

	// Parse ports
	for _, port := range service.Ports {
		mapping, err := parsePortString(port)
		if err != nil {
			return nil, fmt.Errorf("parsing port %s: %w", port, err)
		}
		parsed.Ports = append(parsed.Ports, mapping)
	}

	// Parse resource limits
	if service.Deploy != nil && service.Deploy.Resources != nil && service.Deploy.Resources.Limits != nil {
		if service.Deploy.Resources.Limits.CPUs != "" {
			cpus, err := strconv.ParseFloat(service.Deploy.Resources.Limits.CPUs, 64)
			if err == nil {
				parsed.CPUs = cpus
			}
		}
		if service.Deploy.Resources.Limits.Memory != "" {
			parsed.Memory = service.Deploy.Resources.Limits.Memory
		}
	}

	// Legacy resource limits
	if service.CPUs != "" {
		cpus, err := strconv.ParseFloat(service.CPUs, 64)
		if err == nil {
			parsed.CPUs = cpus
		}
	}
	if service.MemLimit != "" {
		parsed.Memory = service.MemLimit
	}

	return parsed, nil
}

// parseVolumeString parses a volume string like "./app:/workspace:ro"
func parseVolumeString(vol string, baseDir string) (VolumeMount, error) {
	parts := strings.Split(vol, ":")
	if len(parts) < 2 {
		return VolumeMount{}, fmt.Errorf("invalid volume format: %s", vol)
	}

	mount := VolumeMount{
		HostPath:      parts[0],
		ContainerPath: parts[1],
	}

	// Handle relative paths
	if strings.HasPrefix(mount.HostPath, "./") || strings.HasPrefix(mount.HostPath, "../") {
		mount.HostPath = filepath.Join(baseDir, mount.HostPath)
	}

	// Check for read-only flag
	if len(parts) > 2 && parts[2] == "ro" {
		mount.ReadOnly = true
	}

	return mount, nil
}

// parsePortString parses a port string like "8080:80/tcp"
func parsePortString(port string) (PortMapping, error) {
	// Remove protocol suffix if present
	protocol := "tcp"
	if strings.Contains(port, "/") {
		parts := strings.Split(port, "/")
		port = parts[0]
		protocol = parts[1]
	}

	parts := strings.Split(port, ":")
	if len(parts) != 2 {
		return PortMapping{}, fmt.Errorf("invalid port format: %s", port)
	}

	hostPort, err := strconv.Atoi(parts[0])
	if err != nil {
		return PortMapping{}, fmt.Errorf("invalid host port: %s", parts[0])
	}

	containerPort, err := strconv.Atoi(parts[1])
	if err != nil {
		return PortMapping{}, fmt.Errorf("invalid container port: %s", parts[1])
	}

	return PortMapping{
		HostPort:      hostPort,
		ContainerPort: containerPort,
		Protocol:      protocol,
	}, nil
}

// GetServiceNames returns all service names in the compose file
func (c *ComposeFile) GetServiceNames() []string {
	names := make([]string, 0, len(c.Services))
	for name := range c.Services {
		names = append(names, name)
	}
	return names
}
