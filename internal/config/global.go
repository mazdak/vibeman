package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vibeman/internal/constants"
	"vibeman/internal/xdg"

	"github.com/pelletier/go-toml/v2"
)

// GlobalConfig represents the global vibeman configuration
type GlobalConfig struct {
	Server   ServerConfig        `toml:"server"`
	Storage  StorageConfig       `toml:"storage"`
	Services GlobalServicesConfig `toml:"services"`
}

type ServerConfig struct {
	Port      int `toml:"port"`       // Server port (default 8080)
	WebUIPort int `toml:"webui_port"` // Web UI port (default 8081)
}

type StorageConfig struct {
	RepositoriesPath string `toml:"repositories_path" json:"repositories_path" example:"~/vibeman/repos"` // Default repos location
	WorktreesPath    string `toml:"worktrees_path" json:"worktrees_path" example:"~/vibeman/worktrees"`    // Default worktree location
}

type GlobalServicesConfig struct {
	ConfigPath string `toml:"config_path"` // Location of services.toml
}

// DefaultGlobalConfig returns the default global configuration
func DefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		Server: ServerConfig{
			Port:      constants.DefaultServerPort,
			WebUIPort: constants.DefaultWebUIPort,
		},
		Storage: StorageConfig{
			RepositoriesPath: "~/vibeman/repos",
			WorktreesPath:    "~/vibeman/worktrees",
		},
		Services: GlobalServicesConfig{
			ConfigPath: "", // Will use XDG default
		},
	}
}

// getConfigDir returns the XDG config directory for vibeman
func getConfigDir() (string, error) {
	return xdg.ConfigDir()
}

// GetConfigDir returns the XDG config directory for vibeman (exported version)
func GetConfigDir() (string, error) {
	return getConfigDir()
}

// LoadGlobalConfig loads the global configuration from XDG config directory
func LoadGlobalConfig() (*GlobalConfig, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "config.toml")

	// If config doesn't exist, return defaults with expanded paths
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultGlobalConfig()
		// Set services config path to default if empty
		if config.Services.ConfigPath == "" {
			config.Services.ConfigPath = filepath.Join(configDir, "services.toml")
		}
		// Expand tilde paths
		if err := expandPaths(config); err != nil {
			return nil, err
		}
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config GlobalConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Apply defaults for any missing values
	defaults := DefaultGlobalConfig()
	if config.Server.Port == 0 {
		config.Server.Port = defaults.Server.Port
	}
	if config.Server.WebUIPort == 0 {
		config.Server.WebUIPort = defaults.Server.WebUIPort
	}
	if config.Storage.RepositoriesPath == "" {
		config.Storage.RepositoriesPath = defaults.Storage.RepositoriesPath
	}
	if config.Storage.WorktreesPath == "" {
		config.Storage.WorktreesPath = defaults.Storage.WorktreesPath
	}
	if config.Services.ConfigPath == "" {
		// Default to XDG config dir + services.toml
		config.Services.ConfigPath = filepath.Join(configDir, "services.toml")
	}

	// Expand tilde paths
	if err := expandPaths(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveGlobalConfig saves the global configuration to XDG config directory
func SaveGlobalConfig(config *GlobalConfig) error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.toml")

	data, err := toml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// Save saves the global configuration to the specified path
func (g *GlobalConfig) Save(path string) error {
	data, err := toml.Marshal(g)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// ValidateGlobalConfig validates the global configuration
func ValidateGlobalConfig(config *GlobalConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate port ranges
	if config.Server.Port < 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Server.Port)
	}
	if config.Server.WebUIPort < 0 || config.Server.WebUIPort > 65535 {
		return fmt.Errorf("invalid port: %d", config.Server.WebUIPort)
	}

	// Validate paths
	if config.Storage.RepositoriesPath == "" {
		return fmt.Errorf("repositories path cannot be empty")
	}
	if config.Storage.WorktreesPath == "" {
		return fmt.Errorf("worktrees path cannot be empty")
	}

	return nil
}

// expandPaths expands tilde paths in the configuration
func expandPaths(config *GlobalConfig) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	
	// Expand repository path
	if strings.HasPrefix(config.Storage.RepositoriesPath, "~/") {
		config.Storage.RepositoriesPath = filepath.Join(homeDir, config.Storage.RepositoriesPath[2:])
	}
	
	// Expand worktrees path
	if strings.HasPrefix(config.Storage.WorktreesPath, "~/") {
		config.Storage.WorktreesPath = filepath.Join(homeDir, config.Storage.WorktreesPath[2:])
	}
	
	// Expand services config path if needed
	if strings.HasPrefix(config.Services.ConfigPath, "~/") {
		config.Services.ConfigPath = filepath.Join(homeDir, config.Services.ConfigPath[2:])
	}
	
	return nil
}
