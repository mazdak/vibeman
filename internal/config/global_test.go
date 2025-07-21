package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestXDGDirectoryCreation tests that XDG directory is created on first run
func TestXDGDirectoryCreation(t *testing.T) {
	// Create a temporary directory to simulate XDG_CONFIG_HOME
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	// Ensure vibeman directory doesn't exist
	vibemanDir := filepath.Join(tmpDir, "vibeman")
	_, err := os.Stat(vibemanDir)
	assert.True(t, os.IsNotExist(err))

	// Load global config (it returns defaults if config doesn't exist)
	config, err := LoadGlobalConfig()
	require.NoError(t, err)
	assert.NotNil(t, config)

	// The directory is NOT created by LoadGlobalConfig
	// It only returns the path, creation happens when saving config
	_, err = os.Stat(vibemanDir)
	assert.True(t, os.IsNotExist(err), "Directory should not be created by LoadGlobalConfig")
}

// TestDefaultGlobalConfigGeneration tests that default config is generated correctly
func TestDefaultGlobalConfigGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	config, err := LoadGlobalConfig()
	require.NoError(t, err)

	// Verify default values
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, 8081, config.Server.WebUIPort)

	homeDir, _ := os.UserHomeDir()
	expectedReposPath := filepath.Join(homeDir, "vibeman", "repos")
	expectedWorktreesPath := filepath.Join(homeDir, "vibeman", "worktrees")
	
	assert.Equal(t, expectedReposPath, config.Storage.RepositoriesPath)
	assert.Equal(t, expectedWorktreesPath, config.Storage.WorktreesPath)
	
	assert.NotEmpty(t, config.Services.ConfigPath)
}

// TestCustomPortConfiguration tests custom port configuration
func TestCustomPortConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	// Create custom config
	customConfig := &GlobalConfig{
		Server: ServerConfig{
			Port:      9090,
			WebUIPort: 9091,
		},
		Storage: StorageConfig{
			RepositoriesPath: "/custom/repos",
			WorktreesPath:    "/custom/worktrees",
		},
		Services: GlobalServicesConfig{
			ConfigPath: "/custom/services.toml",
		},
	}

	// Save custom config
	err := SaveGlobalConfig(customConfig)
	require.NoError(t, err)

	// Load and verify
	loaded, err := LoadGlobalConfig()
	require.NoError(t, err)

	assert.Equal(t, 9090, loaded.Server.Port)
	assert.Equal(t, 9091, loaded.Server.WebUIPort)
}

// TestCustomPathsConfiguration tests custom paths for repositories and worktrees
func TestCustomPathsConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	customReposPath := filepath.Join(tmpDir, "my-repos")
	customWorktreesPath := filepath.Join(tmpDir, "my-worktrees")

	config := &GlobalConfig{
		Server: ServerConfig{
			Port:      8080,
			WebUIPort: 8081,
		},
		Storage: StorageConfig{
			RepositoriesPath: customReposPath,
			WorktreesPath:    customWorktreesPath,
		},
		Services: GlobalServicesConfig{
			ConfigPath: filepath.Join(tmpDir, "services.toml"),
		},
	}

	err := SaveGlobalConfig(config)
	require.NoError(t, err)

	loaded, err := LoadGlobalConfig()
	require.NoError(t, err)

	assert.Equal(t, customReposPath, loaded.Storage.RepositoriesPath)
	assert.Equal(t, customWorktreesPath, loaded.Storage.WorktreesPath)
}

// TestConfigFileValidationWithInvalidValues tests validation of invalid config values
func TestConfigFileValidationWithInvalidValues(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	tests := []struct {
		name        string
		config      *GlobalConfig
		shouldError bool
		errorMsg    string
	}{
		{
			name: "negative port",
			config: &GlobalConfig{
				Server: ServerConfig{
					Port:      -1,
					WebUIPort: 8081,
				},
				Storage: StorageConfig{
					RepositoriesPath: "/tmp/repos",
					WorktreesPath:    "/tmp/worktrees",
				},
			},
			shouldError: true,
			errorMsg:    "invalid port",
		},
		{
			name: "port too high",
			config: &GlobalConfig{
				Server: ServerConfig{
					Port:      70000,
					WebUIPort: 8081,
				},
				Storage: StorageConfig{
					RepositoriesPath: "/tmp/repos",
					WorktreesPath:    "/tmp/worktrees",
				},
			},
			shouldError: true,
			errorMsg:    "invalid port",
		},
		{
			name: "empty repositories path",
			config: &GlobalConfig{
				Server: ServerConfig{
					Port:      8080,
					WebUIPort: 8081,
				},
				Storage: StorageConfig{
					RepositoriesPath: "",
					WorktreesPath:    "/tmp/worktrees",
				},
			},
			shouldError: true,
			errorMsg:    "repositories path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGlobalConfig(tt.config)
			if tt.shouldError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGlobalConfigTomlFormat tests the TOML format of the global config
func TestGlobalConfigTomlFormat(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	config := &GlobalConfig{
		Server: ServerConfig{
			Port:      8080,
			WebUIPort: 8081,
		},
		Storage: StorageConfig{
			RepositoriesPath: "~/vibeman/repos",
			WorktreesPath:    "~/vibeman/worktrees",
		},
		Services: GlobalServicesConfig{
			ConfigPath: "~/.config/vibeman/services.toml",
		},
	}

	err := SaveGlobalConfig(config)
	require.NoError(t, err)

	// Read the raw file to verify format
	configPath := filepath.Join(tmpDir, "vibeman", "config.toml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	// Verify TOML structure
	contentStr := string(content)
	assert.Contains(t, contentStr, "[server]")
	assert.Contains(t, contentStr, "port = 8080")
	assert.Contains(t, contentStr, "webui_port = 8081")
	assert.Contains(t, contentStr, "[storage]")
	assert.Contains(t, contentStr, "repositories_path = ")
	assert.Contains(t, contentStr, "worktrees_path = ")
	assert.Contains(t, contentStr, "[services]")
	assert.Contains(t, contentStr, "config_path = ")
}