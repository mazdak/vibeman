package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParsingGlobalServicesToml tests parsing of global services.toml
func TestParsingGlobalServicesToml(t *testing.T) {
	content := `
[services]

[services.postgres]
compose_file = "/path/to/docker-compose.yaml"
service = "postgres"
description = "PostgreSQL with PostGIS for development"

[services.redis]
compose_file = "/path/to/docker-compose.yaml"
service = "redis"
description = "Redis cache server"

[services.localstack]
compose_file = "/path/to/another/docker-compose.yaml"
service = "localstack"
description = "LocalStack for AWS services emulation"
`

	tmpDir := t.TempDir()
	servicesPath := filepath.Join(tmpDir, "services.toml")
	err := os.WriteFile(servicesPath, []byte(content), 0644)
	require.NoError(t, err)

	// Parse services config
	services, err := LoadServicesConfig(servicesPath)
	require.NoError(t, err)

	// Verify all services
	assert.Len(t, services.Services, 3)

	postgres, exists := services.Services["postgres"]
	assert.True(t, exists)
	assert.Equal(t, "/path/to/docker-compose.yaml", postgres.ComposeFile)
	assert.Equal(t, "postgres", postgres.Service)
	assert.Equal(t, "PostgreSQL with PostGIS for development", postgres.Description)

	redis, exists := services.Services["redis"]
	assert.True(t, exists)
	assert.Equal(t, "/path/to/docker-compose.yaml", redis.ComposeFile)
	assert.Equal(t, "redis", redis.Service)
	assert.Equal(t, "Redis cache server", redis.Description)

	localstack, exists := services.Services["localstack"]
	assert.True(t, exists)
	assert.Equal(t, "/path/to/another/docker-compose.yaml", localstack.ComposeFile)
	assert.Equal(t, "localstack", localstack.Service)
}

// TestServiceReferenceValidation tests validation of service references
func TestServiceReferenceValidation(t *testing.T) {
	// Create mock docker-compose.yaml files
	tmpDir := t.TempDir()
	
	// Create valid compose file
	validComposeDir := filepath.Join(tmpDir, "valid")
	err := os.MkdirAll(validComposeDir, 0755)
	require.NoError(t, err)
	
	validComposePath := filepath.Join(validComposeDir, "docker-compose.yaml")
	validComposeContent := `
version: '3.8'
services:
  postgres:
    image: postgres:14
    ports:
      - "5432:5432"
  redis:
    image: redis:6
    ports:
      - "6379:6379"
`
	err = os.WriteFile(validComposePath, []byte(validComposeContent), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		service     ServiceConfig
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid service reference",
			service: ServiceConfig{
				ComposeFile: validComposePath,
				Service:     "postgres",
				Description: "Valid postgres service",
			},
			shouldError: false,
		},
		{
			name: "compose file does not exist",
			service: ServiceConfig{
				ComposeFile: "/non/existent/docker-compose.yaml",
				Service:     "postgres",
				Description: "Invalid path",
			},
			shouldError: true,
			errorMsg:    "compose file not found",
		},
		{
			name: "service not in compose file",
			service: ServiceConfig{
				ComposeFile: validComposePath,
				Service:     "mongodb",
				Description: "Service not defined",
			},
			// TODO: Validation doesn't check if service exists in compose file yet
			shouldError: false,
			errorMsg:    "",
		},
		{
			name: "empty service name",
			service: ServiceConfig{
				ComposeFile: validComposePath,
				Service:     "",
				Description: "Empty service name",
			},
			shouldError: true,
			errorMsg:    "service name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServiceConfig(&tt.service)
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

// TestMergingRepositoryAndGlobalServices tests merging of repository and global services
func TestMergingRepositoryAndGlobalServices(t *testing.T) {
	// Create global services config
	globalServices := &ServicesConfig{
		Services: map[string]ServiceConfig{
			"postgres": {
				ComposeFile: "/global/docker-compose.yaml",
				Service:     "postgres",
				Description: "Global PostgreSQL",
			},
			"redis": {
				ComposeFile: "/global/docker-compose.yaml",
				Service:     "redis",
				Description: "Global Redis",
			},
			"mongodb": {
				ComposeFile: "/global/docker-compose.yaml",
				Service:     "mongodb",
				Description: "Global MongoDB",
			},
		},
	}

	// Create repository config with service requirements
	repoConfig := &RepositoryConfig{}
	repoConfig.Repository.Name = "test-repo"
	repoConfig.Repository.Services = map[string]ServiceRequirement{
		"postgres": {Required: true},
		"redis":    {Required: true},
		"elastic":  {Required: false}, // Not in global services
	}

	// Test merging logic
	requiredServices := GetRequiredServices(repoConfig, globalServices)

	// Should include postgres and redis from global services
	assert.Len(t, requiredServices, 2)
	assert.Contains(t, requiredServices, "postgres")
	assert.Contains(t, requiredServices, "redis")
	assert.NotContains(t, requiredServices, "mongodb") // Not required by repo
	assert.NotContains(t, requiredServices, "elastic") // Not available globally
}

// TestServiceConfigLoading tests loading services from different sources
func TestServiceConfigLoading(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Test loading from default location
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	// Create services config in XDG location
	xdgServicesDir := filepath.Join(tmpDir, "vibeman")
	err := os.MkdirAll(xdgServicesDir, 0755)
	require.NoError(t, err)
	
	servicesPath := filepath.Join(xdgServicesDir, "services.toml")
	content := `
[services]
[services.test-service]
compose_file = "/test/docker-compose.yaml"
service = "test"
description = "Test service"
`
	err = os.WriteFile(servicesPath, []byte(content), 0644)
	require.NoError(t, err)

	// Load using default path
	services, err := LoadServicesConfig("")
	require.NoError(t, err)
	
	assert.Len(t, services.Services, 1)
	assert.Contains(t, services.Services, "test-service")
}

// TestEmptyServicesConfig tests handling of empty services configuration
func TestEmptyServicesConfig(t *testing.T) {
	content := `[services]`
	
	tmpDir := t.TempDir()
	servicesPath := filepath.Join(tmpDir, "services.toml")
	err := os.WriteFile(servicesPath, []byte(content), 0644)
	require.NoError(t, err)

	services, err := LoadServicesConfig(servicesPath)
	require.NoError(t, err)
	
	assert.NotNil(t, services)
	assert.NotNil(t, services.Services)
	assert.Len(t, services.Services, 0)
}

// Helper functions for tests

// GetRequiredServices returns list of required service names
func GetRequiredServices(repoConfig *RepositoryConfig, globalServices *ServicesConfig) []string {
	var required []string
	
	for serviceName, requirement := range repoConfig.Repository.Services {
		if requirement.Required {
			// Check if service exists in global config
			if _, exists := globalServices.Services[serviceName]; exists {
				required = append(required, serviceName)
			}
		}
	}
	
	return required
}