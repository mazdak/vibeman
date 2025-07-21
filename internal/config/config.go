package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"vibeman/internal/validation"
)

// Manager handles configuration loading and validation
type Manager struct {
	Repository *RepositoryConfig
	Services   *ServicesConfig
}

// ContainerLifecycle is deprecated - lifecycle is now managed by docker-compose

// AIConfig configures the AI assistant container
type AIConfig struct {
	Enabled bool              `toml:"enabled"` // Enable AI assistant container
	Image   string            `toml:"image"`   // Container image for AI assistant (built from Dockerfile)
	Env     map[string]string `toml:"env"`     // Additional environment variables
	Volumes map[string]string `toml:"volumes"` // Additional volume mounts
}

// RepositoryConfig represents a repository configuration
type RepositoryConfig struct {
	Repository struct {
		Name        string `toml:"name"`
		Description string `toml:"description"`
		Git         struct {
			RepoURL        string `toml:"repo_url"`
			DefaultBranch  string `toml:"default_branch"`
			WorktreePrefix string `toml:"worktree_prefix"`
			AutoSync       bool   `toml:"auto_sync"`
		} `toml:"git"`
		Worktrees struct {
			Directory string `toml:"directory"`
		} `toml:"worktrees"`
		Container struct {
			// Required: Docker compose configuration
			ComposeFile string   `toml:"compose_file"` // Path to docker-compose.yaml
			Services    []string `toml:"services"`     // Services to use from compose file (empty = all)
			// Optional: Setup commands that run inside the container
			Setup []string `toml:"setup"` // Commands to run after container starts
			// Additional container configuration
			Environment map[string]string `toml:"environment"` // Additional environment variables
			AI          *AIConfig         `toml:"ai"`          // AI assistant configuration
		} `toml:"container"`
		Services map[string]ServiceRequirement `toml:"-"` // Service requirements (custom unmarshal)
		Runtime  struct {
			Type string `toml:"type"` // "apple" or "docker"
		} `toml:"runtime"`
		Setup struct {
			WorktreeInit  string   `toml:"worktree_init"`
			ContainerInit []string `toml:"container_init"`
		} `toml:"setup"`
		AI AIConfig `toml:"ai"`
	} `toml:"repository"`
}

// ServicesConfig represents the services configuration
type ServicesConfig struct {
	Services map[string]ServiceConfig `toml:"-"` // Custom unmarshal for both formats
}

// Save saves the services configuration to the specified path
func (s *ServicesConfig) Save(path string) error {
	// Create a wrapper struct for marshaling
	wrapper := struct {
		Services map[string]ServiceConfig `toml:"services"`
	}{
		Services: s.Services,
	}

	data, err := toml.Marshal(wrapper)
	if err != nil {
		return fmt.Errorf("failed to marshal services config: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// UnmarshalTOML implements custom TOML unmarshaling for ServicesConfig
func (s *ServicesConfig) UnmarshalTOML(data interface{}) error {
	if s.Services == nil {
		s.Services = make(map[string]ServiceConfig)
	}

	// Handle the data as a map
	if m, ok := data.(map[string]interface{}); ok {
		if servicesSection, exists := m["services"]; exists {
			if servicesMap, ok := servicesSection.(map[string]interface{}); ok {
				for name, service := range servicesMap {
					if serviceMap, ok := service.(map[string]interface{}); ok {
						// Marshal it to ServiceConfig
						var sc ServiceConfig
						if err := remarshalServiceConfig(serviceMap, &sc); err != nil {
							return fmt.Errorf("failed to parse service %s: %w", name, err)
						}
						s.Services[name] = sc
					}
				}
			}
		}
	}

	return nil
}

// remarshalServiceConfig converts a map to ServiceConfig
func remarshalServiceConfig(data map[string]interface{}, sc *ServiceConfig) error {
	// Marshal the map back to TOML bytes
	bytes, err := toml.Marshal(data)
	if err != nil {
		return err
	}
	// Unmarshal into the ServiceConfig struct
	return toml.Unmarshal(bytes, sc)
}

// ServiceConfig represents a service configuration
type ServiceConfig struct {
	// Docker Compose integration - the only supported approach
	ComposeFile string `toml:"compose_file"`
	Service     string `toml:"service"`
	Description string `toml:"description,omitempty"`
}

// IsValid returns true if this service configuration is valid
func (sc *ServiceConfig) IsValid() bool {
	return sc.ComposeFile != "" && sc.Service != ""
}

// ServiceRequirement represents a service requirement in repository config
type ServiceRequirement struct {
	Required bool `toml:"required"`
}

// New creates a new configuration manager
func New() *Manager {
	return &Manager{
		Repository: &RepositoryConfig{},
		Services:   &ServicesConfig{},
	}
}

// Load loads configuration from various sources
func (m *Manager) Load() error {
	// Load repository config if exists
	if err := m.loadRepositoryConfig(); err != nil {
		return fmt.Errorf("failed to load repository config: %w", err)
	}

	// Load services config
	if err := m.loadServicesConfig(); err != nil {
		return fmt.Errorf("failed to load services config: %w", err)
	}

	// Apply defaults
	m.applyDefaults()

	return nil
}

// findRepositoryConfigPath finds the repository configuration file, supporting worktrees
func (m *Manager) findRepositoryConfigPath() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// First check current directory
	configPath := filepath.Join(currentDir, "vibeman.toml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	// Check if we're in a worktree
	gitPath := filepath.Join(currentDir, ".git")
	if info, err := os.Stat(gitPath); err == nil && !info.IsDir() {
		// Read .git file to check if it's a worktree
		content, err := os.ReadFile(gitPath)
		if err == nil {
			gitDirLine := strings.TrimSpace(string(content))
			if strings.HasPrefix(gitDirLine, "gitdir: ") && strings.Contains(gitDirLine, "worktrees") {
				// Extract main repo path from worktree git dir
				gitDir := strings.TrimPrefix(gitDirLine, "gitdir: ")
				if strings.Contains(gitDir, "worktrees") {
					parts := strings.Split(gitDir, "/worktrees/")
					if len(parts) >= 2 {
						mainRepoGitDir := parts[0]
						mainRepoPath := filepath.Dir(mainRepoGitDir)

						// Check for vibeman.toml in main repository
						mainConfigPath := filepath.Join(mainRepoPath, "vibeman.toml")
						if _, err := os.Stat(mainConfigPath); err == nil {
							return mainConfigPath, nil
						}
					}
				}
			}
		}
	}

	// No config found
	return "", fmt.Errorf("no vibeman.toml found")
}

// loadRepositoryConfig loads repository configuration
func (m *Manager) loadRepositoryConfig() error {
	configPath, err := m.findRepositoryConfigPath()
	if err != nil {
		// No repository config is fine
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read repository config file %s: %w", configPath, err)
	}

	// Parse the full configuration using a raw map to handle nested structures
	var rawConfig map[string]interface{}
	if err := toml.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("failed to parse repository config file %s: %w", configPath, err)
	}

	// Check for "repository" section
	configSection, ok := rawConfig["repository"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("no [repository] section found in config file %s", configPath)
	}

	// Parse the configuration
	var tempConfig struct {
		Repository struct {
			Name        string `toml:"name"`
			Description string `toml:"description"`
			Git         struct {
				RepoURL        string `toml:"repo_url"`
				DefaultBranch  string `toml:"default_branch"`
				WorktreePrefix string `toml:"worktree_prefix"`
				AutoSync       bool   `toml:"auto_sync"`
			} `toml:"git"`
			Worktrees struct {
				Directory string `toml:"directory"`
			} `toml:"worktrees"`
			Container struct {
				ComposeFile string   `toml:"compose_file"`
				Services    []string `toml:"services"`
				Setup       []string `toml:"setup"`
			} `toml:"container"`
			Runtime struct {
				Type string `toml:"type"`
			} `toml:"runtime"`
			Setup struct {
				WorktreeInit  string   `toml:"worktree_init"`
				ContainerInit []string `toml:"container_init"`
			} `toml:"setup"`
			AI AIConfig `toml:"ai"`
		} `toml:"repository"`
	}

	if err := toml.Unmarshal(data, &tempConfig); err != nil {
		return fmt.Errorf("failed to parse repository config file %s: %w", configPath, err)
	}

	// Map to main structure
	m.Repository.Repository.Name = tempConfig.Repository.Name
	m.Repository.Repository.Description = tempConfig.Repository.Description
	m.Repository.Repository.Git = tempConfig.Repository.Git
	m.Repository.Repository.Worktrees = tempConfig.Repository.Worktrees
	m.Repository.Repository.Container.ComposeFile = tempConfig.Repository.Container.ComposeFile
	m.Repository.Repository.Container.Services = tempConfig.Repository.Container.Services
	m.Repository.Repository.Container.Setup = tempConfig.Repository.Container.Setup
	m.Repository.Repository.Runtime = tempConfig.Repository.Runtime
	m.Repository.Repository.Setup = tempConfig.Repository.Setup
	m.Repository.Repository.AI = tempConfig.Repository.AI
	
	// Handle AI configuration with defaults
	if aiSection, ok := configSection["ai"].(map[string]interface{}); ok {
		// AI section exists, parse it
		if enabled, ok := aiSection["enabled"].(bool); ok {
			m.Repository.Repository.AI.Enabled = enabled
		} else {
			// Default to true if not specified
			m.Repository.Repository.AI.Enabled = true
		}
		
		if image, ok := aiSection["image"].(string); ok {
			m.Repository.Repository.AI.Image = image
		}
		
		// Parse env vars
		if envSection, ok := aiSection["env"].(map[string]interface{}); ok {
			m.Repository.Repository.AI.Env = make(map[string]string)
			for k, v := range envSection {
				if strVal, ok := v.(string); ok {
					m.Repository.Repository.AI.Env[k] = strVal
				}
			}
		}
		
		// Parse volumes
		if volSection, ok := aiSection["volumes"].(map[string]interface{}); ok {
			m.Repository.Repository.AI.Volumes = make(map[string]string)
			for k, v := range volSection {
				if strVal, ok := v.(string); ok {
					m.Repository.Repository.AI.Volumes[k] = strVal
				}
			}
		}
	} else {
		// No AI section, default to enabled
		m.Repository.Repository.AI.Enabled = true
	}
	
	// Handle environment variables from container section
	if containerSection, ok := configSection["container"].(map[string]interface{}); ok {
		if envSection, ok := containerSection["environment"].(map[string]interface{}); ok {
			m.Repository.Repository.Container.Environment = make(map[string]string)
			for k, v := range envSection {
				if strVal, ok := v.(string); ok {
					m.Repository.Repository.Container.Environment[k] = strVal
				}
			}
		}
	}

	// Handle services - support both string and object formats
	m.Repository.Repository.Services = make(map[string]ServiceRequirement)
	if services, ok := configSection["services"].(map[string]interface{}); ok {
		for name, service := range services {
			// Check if it's a string (simple format)
			if _, ok := service.(string); ok {
				// Simple format: service = "image" means required = true
				m.Repository.Repository.Services[name] = ServiceRequirement{Required: true}
			} else if serviceMap, ok := service.(map[string]interface{}); ok {
				// Detailed format: parse the ServiceRequirement
				sr := ServiceRequirement{}
				if required, ok := serviceMap["required"].(bool); ok {
					sr.Required = required
				}
				m.Repository.Repository.Services[name] = sr
			}
		}
	}

	// Note: Ports are now handled by docker-compose, no need to parse them here

	return nil
}

// loadServicesConfig loads services configuration
func (m *Manager) loadServicesConfig() error {
	// Use XDG config directory
	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "services.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default services config if it doesn't exist
		if err := m.createDefaultServicesConfig(configPath); err != nil {
			return fmt.Errorf("failed to create default services config: %w", err)
		}
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read services config file %s: %w", configPath, err)
	}

	// Parse raw data to handle both formats
	var rawData interface{}
	if err := toml.Unmarshal(data, &rawData); err != nil {
		return fmt.Errorf("failed to parse services config file %s: %w", configPath, err)
	}

	// Use custom unmarshaler
	if err := m.Services.UnmarshalTOML(rawData); err != nil {
		return fmt.Errorf("failed to unmarshal services config: %w", err)
	}

	return nil
}

// createDefaultServicesConfig creates a default services configuration
func (m *Manager) createDefaultServicesConfig(configPath string) error {
	// Create default services config content
	defaultContent := `# Vibeman Services Configuration
# Services are defined by referencing docker-compose files

[services]

# Example: PostgreSQL service from a docker-compose file
# [services.postgres]
# compose_file = "/path/to/docker-compose.yaml"
# service = "postgres"
# description = "PostgreSQL database for development"

# Example: Redis service from a docker-compose file
# [services.redis]
# compose_file = "/path/to/docker-compose.yaml"
# service = "redis"
# description = "Redis cache server"

# Example: LocalStack service
# [services.localstack]
# compose_file = "/path/to/docker-compose.yaml"
# service = "localstack"
# description = "AWS services emulation"
`

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	// Write the default content
	if err := os.WriteFile(configPath, []byte(defaultContent), 0644); err != nil {
		return err
	}

	// Parse it back to populate m.Services
	var rawData interface{}
	if err := toml.Unmarshal([]byte(defaultContent), &rawData); err != nil {
		return err
	}

	return m.Services.UnmarshalTOML(rawData)
}

// applyDefaults applies default values to configuration
func (m *Manager) applyDefaults() {
	// Apply repository defaults
	if m.Repository.Repository.Name != "" {
		if m.Repository.Repository.Worktrees.Directory == "" {
			// Default to sibling directory
			m.Repository.Repository.Worktrees.Directory = "../" + m.Repository.Repository.Name + "-worktrees"
		}
		if m.Repository.Repository.Runtime.Type == "" {
			m.Repository.Repository.Runtime.Type = "docker"
		}
		// Note: Container image, working directory, ports etc. are now handled by docker-compose
	}
}

// Validate validates the configuration
func (m *Manager) Validate() error {
	// Validate repository configuration
	if m.Repository != nil && m.Repository.Repository.Name != "" {
		// Validate services map
		if m.Services != nil {
			for serviceName := range m.Repository.Repository.Services {
				if _, exists := m.Services.Services[serviceName]; !exists {
					return fmt.Errorf("service %q not found in services configuration", serviceName)
				}
			}
		}

		// Validate port configurations
		if err := m.validatePorts(); err != nil {
			return fmt.Errorf("port configuration validation failed: %w", err)
		}

		// Validate process configurations
		if err := m.validateProcesses(); err != nil {
			return fmt.Errorf("process configuration validation failed: %w", err)
		}

		// Validate setup configurations
		if err := m.validateSetup(); err != nil {
			return fmt.Errorf("setup configuration validation failed: %w", err)
		}

		// Validate runtime configurations
		if err := m.validateRuntime(); err != nil {
			return fmt.Errorf("runtime configuration validation failed: %w", err)
		}
	}

	return nil
}

// validatePorts validates port configurations
func (m *Manager) validatePorts() error {
	// Note: Port configuration is now handled by docker-compose
	// No validation needed at the vibeman level
	return nil
}

// isValidPortMapping checks if a port mapping follows the HOST:CONTAINER format
func isValidPortMapping(mapping string) bool {
	return validation.IsValidPortMapping(mapping)
}

// isReservedPortName checks if a port name is reserved
func isReservedPortName(name string) bool {
	return validation.IsReservedPortName(name)
}

// validateProcesses validates process configurations
func (m *Manager) validateProcesses() error {
	// Note: Process configuration is now handled by docker-compose
	// No validation needed at the vibeman level
	return nil
}

// validateProcessCommand validates a single process command
func validateProcessCommand(command string) error {
	return validation.ProcessCommand(command)
}

// validateSetup validates setup configurations
func (m *Manager) validateSetup() error {
	if m.Repository == nil || m.Repository.Repository.Name == "" {
		return nil
	}

	container := &m.Repository.Repository.Container

	// Validate Setup commands are non-empty strings
	for i, cmd := range container.Setup {
		if err := validation.NonEmptyString(cmd); err != nil {
			return fmt.Errorf("setup command %d: %w", i, err)
		}
	}

	// Note: SetupScript and Lifecycle are removed in simplified approach
	// Setup is now just an array of commands to run after container starts

	return nil
}

// validateRuntime validates runtime configurations
func (m *Manager) validateRuntime() error {
	if m.Repository == nil || m.Repository.Repository.Name == "" {
		return nil
	}

	runtime := &m.Repository.Repository.Runtime

	// Validate runtime type
	if runtime.Type != "" && runtime.Type != "docker" {
		return fmt.Errorf("invalid runtime type %q, must be 'docker'", runtime.Type)
	}

	// Note: Pool configuration removed in simplified approach - compose handles resource management

	return nil
}

// CreateDefaultRepositoryConfig creates an example repository configuration file
func CreateDefaultRepositoryConfig(path string) error {
	example := `# Vibeman Repository Configuration (Compose-First Approach)
# This uses docker-compose for all container configuration

[repository]
name = "my-repository"
description = "My awesome repository"

[repository.git]
repo_url = "https://github.com/user/my-repository.git"
default_branch = "main"
auto_sync = true

[repository.worktrees]
directory = "../my-repository-worktrees"

[repository.container]
# Required: Docker compose configuration
compose_file = "./docker-compose.dev.yaml"
services = ["backend", "postgres", "redis"]

# Optional: Setup commands to run after container starts
setup = [
    "npm install",
    "npm run build",
    "echo 'Container setup complete'"
]

# Service dependencies (defined in services.toml or docker-compose)
[repository.services]
postgres = { required = true }
redis = { required = false }

[repository.runtime]
type = "docker"  # Options: "docker" or "apple"

[repository.setup]
# Script to run after creating a new worktree
worktree_init = "echo 'New worktree created'"

# Commands to run after container initialization
container_init = [
    "echo 'Container initialized'",
    "echo 'Ready for development'"
]

# Note: All container configuration (image, ports, volumes, environment, etc.)
# is now handled by the docker-compose.dev.yaml file.
`
	return os.WriteFile(path, []byte(example), 0644)
}

// ParseRepositoryConfigFromPath loads and parses a repository config from a file path
func ParseRepositoryConfig(repoPath string) (*RepositoryConfig, error) {
	configPath := filepath.Join(repoPath, "vibeman.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	cfg := &RepositoryConfig{}
	if err := ParseRepositoryConfigData(data, cfg); err != nil {
		return nil, err
	}
	
	return cfg, nil
}

// ParseRepositoryConfigData parses TOML data into a RepositoryConfig struct
func ParseRepositoryConfigData(data []byte, cfg *RepositoryConfig) error {
	// Parse the full configuration using a raw map to handle nested structures
	var rawConfig map[string]interface{}
	if err := toml.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("failed to parse repository config: %w", err)
	}

	// Check for "repository" section
	configSection, ok := rawConfig["repository"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("no [repository] section found in config")
	}

	// Parse the configuration
	var tempConfig struct {
		Repository struct {
			Name        string `toml:"name"`
			Description string `toml:"description"`
			Git         struct {
				RepoURL        string `toml:"repo_url"`
				DefaultBranch  string `toml:"default_branch"`
				WorktreePrefix string `toml:"worktree_prefix"`
				AutoSync       bool   `toml:"auto_sync"`
			} `toml:"git"`
			Worktrees struct {
				Directory string `toml:"directory"`
			} `toml:"worktrees"`
			Container struct {
				ComposeFile string   `toml:"compose_file"`
				Services    []string `toml:"services"`
				Setup       []string `toml:"setup"`
			} `toml:"container"`
			Runtime struct {
				Type string `toml:"type"`
			} `toml:"runtime"`
			Setup struct {
				WorktreeInit  string   `toml:"worktree_init"`
				ContainerInit []string `toml:"container_init"`
			} `toml:"setup"`
			AI AIConfig `toml:"ai"`
		} `toml:"repository"`
	}

	if err := toml.Unmarshal(data, &tempConfig); err != nil {
		return fmt.Errorf("failed to parse repository config: %w", err)
	}

	// Map to main structure
	cfg.Repository.Name = tempConfig.Repository.Name
	cfg.Repository.Description = tempConfig.Repository.Description
	cfg.Repository.Git = tempConfig.Repository.Git
	cfg.Repository.Worktrees = tempConfig.Repository.Worktrees
	cfg.Repository.Container.ComposeFile = tempConfig.Repository.Container.ComposeFile
	cfg.Repository.Container.Services = tempConfig.Repository.Container.Services
	cfg.Repository.Container.Setup = tempConfig.Repository.Container.Setup
	cfg.Repository.Runtime = tempConfig.Repository.Runtime
	cfg.Repository.Setup = tempConfig.Repository.Setup
	cfg.Repository.AI = tempConfig.Repository.AI
	
	// Handle environment variables from container section
	if containerSection, ok := configSection["container"].(map[string]interface{}); ok {
		if envSection, ok := containerSection["environment"].(map[string]interface{}); ok {
			cfg.Repository.Container.Environment = make(map[string]string)
			for k, v := range envSection {
				if strVal, ok := v.(string); ok {
					cfg.Repository.Container.Environment[k] = strVal
				}
			}
		}
	}

	// Handle AI configuration with defaults
	if aiSection, ok := configSection["ai"].(map[string]interface{}); ok {
		// AI section exists, parse it
		if enabled, ok := aiSection["enabled"].(bool); ok {
			cfg.Repository.AI.Enabled = enabled
		} else {
			// Default to true if not specified
			cfg.Repository.AI.Enabled = true
		}
		
		if image, ok := aiSection["image"].(string); ok {
			cfg.Repository.AI.Image = image
		}
		
		// Parse env vars
		if envSection, ok := aiSection["env"].(map[string]interface{}); ok {
			cfg.Repository.AI.Env = make(map[string]string)
			for k, v := range envSection {
				if strVal, ok := v.(string); ok {
					cfg.Repository.AI.Env[k] = strVal
				}
			}
		}
		
		// Parse volumes
		if volSection, ok := aiSection["volumes"].(map[string]interface{}); ok {
			cfg.Repository.AI.Volumes = make(map[string]string)
			for k, v := range volSection {
				if strVal, ok := v.(string); ok {
					cfg.Repository.AI.Volumes[k] = strVal
				}
			}
		}
	} else {
		// No AI section, default to enabled
		cfg.Repository.AI.Enabled = true
	}

	// Handle services - support both string and object formats
	cfg.Repository.Services = make(map[string]ServiceRequirement)
	if services, ok := configSection["services"].(map[string]interface{}); ok {
		for name, service := range services {
			// Check if it's a string (simple format)
			if _, ok := service.(string); ok {
				// Simple format: service = "image" means required = true
				cfg.Repository.Services[name] = ServiceRequirement{Required: true}
			} else if serviceMap, ok := service.(map[string]interface{}); ok {
				// Detailed format: parse the ServiceRequirement
				sr := ServiceRequirement{}
				if required, ok := serviceMap["required"].(bool); ok {
					sr.Required = required
				}
				cfg.Repository.Services[name] = sr
			}
		}
	}

	return nil
}



// LoadServicesConfig loads services configuration from a file
func LoadServicesConfig(path string) (*ServicesConfig, error) {
	// If no path provided, use default XDG location
	if path == "" {
		configDir, err := getConfigDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(configDir, "services.toml")
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return empty config if file doesn't exist
		return &ServicesConfig{
			Services: make(map[string]ServiceConfig),
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read services config: %w", err)
	}

	// Parse raw data to handle both formats
	var rawData interface{}
	if err := toml.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse services config: %w", err)
	}

	// Use custom unmarshaler
	config := &ServicesConfig{}
	if err := config.UnmarshalTOML(rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal services config: %w", err)
	}

	// Initialize map if nil
	if config.Services == nil {
		config.Services = make(map[string]ServiceConfig)
	}

	return config, nil
}

// ValidateServiceConfig validates a service configuration
func ValidateServiceConfig(service *ServiceConfig) error {
	if service == nil {
		return fmt.Errorf("service config cannot be nil")
	}

	if service.Service == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if service.ComposeFile == "" {
		return fmt.Errorf("compose file cannot be empty")
	}

	// Check if compose file exists
	if _, err := os.Stat(service.ComposeFile); os.IsNotExist(err) {
		return fmt.Errorf("compose file not found: %s", service.ComposeFile)
	}

	// TODO: Validate that the service exists in the compose file
	// This would require parsing the docker-compose.yaml file

	return nil
}

// SaveRepositoryConfig saves a repository configuration to vibeman.toml
func SaveRepositoryConfig(path string, config *RepositoryConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	configPath := filepath.Join(path, "vibeman.toml")
	
	// Create a wrapper to match the TOML structure
	wrapper := struct {
		Repository interface{} `toml:"repository"`
	}{
		Repository: config.Repository,
	}

	// Marshal to TOML
	data, err := toml.Marshal(wrapper)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
