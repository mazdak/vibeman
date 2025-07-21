// +build integration

package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vibeman/internal/config"
	"vibeman/internal/container"
	"vibeman/internal/db"
	"vibeman/internal/git"
	"vibeman/internal/operations"
	"vibeman/internal/service"

	"github.com/stretchr/testify/suite"
)

// ServiceLifecycleTestSuite tests service operations and dependencies
type ServiceLifecycleTestSuite struct {
	suite.Suite
	testDir      string
	db           *db.DB
	configMgr    *config.Manager
	containerMgr *container.Manager
	gitMgr       *git.Manager
	serviceMgr   *service.Manager
	serviceOps   *operations.ServiceOperations
}

func (s *ServiceLifecycleTestSuite) SetupSuite() {
	// Skip if Docker is not available
	if !s.isDockerAvailable() {
		s.T().Skip("Docker is not available, skipping service tests")
	}

	// Create test directory
	testDir, err := os.MkdirTemp("", "vibeman-service-test-*")
	s.Require().NoError(err)
	s.testDir = testDir

	// Create test config directory
	configDir := filepath.Join(testDir, "config")
	err = os.MkdirAll(configDir, 0755)
	s.Require().NoError(err)

	// Initialize database
	dbConfig := &db.Config{
		Driver: "sqlite3",
		DSN:    filepath.Join(testDir, "test.db"),
	}
	s.db, err = db.New(dbConfig)
	s.Require().NoError(err)
	s.Require().NoError(s.db.Migrate())

	// Create global config with test services
	globalConfig := &config.GlobalConfig{
		Storage: config.StorageConfig{
			RepositoriesPath: filepath.Join(testDir, "repos"),
			WorktreesPath:    filepath.Join(testDir, "worktrees"),
		},
	}

	// Create test docker-compose file with dynamic ports to avoid conflicts
	// Use 0 to let Docker assign random available ports
	composeContent := `services:
  postgres:
    image: postgres:13-alpine
    environment:
      POSTGRES_PASSWORD: testpass
      POSTGRES_DB: testdb
    ports:
      - "5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 3
  redis:
    image: redis:6-alpine
    ports:
      - "6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 3
`
	composeFile := filepath.Join(testDir, "docker-compose.test.yaml")
	err = os.WriteFile(composeFile, []byte(composeContent), 0644)
	s.Require().NoError(err)

	// Create services config
	servicesConfig := &config.ServicesConfig{
		Services: map[string]config.ServiceConfig{
			"test-postgres": {
				ComposeFile: composeFile,
				Service:     "postgres",
				Description: "PostgreSQL database for testing",
			},
			"test-redis": {
				ComposeFile: composeFile,
				Service:     "redis",
				Description: "Redis cache for testing",
			},
		},
	}

	// Save configs
	globalConfigPath := filepath.Join(configDir, "config.toml")
	err = globalConfig.Save(globalConfigPath)
	s.Require().NoError(err)

	servicesConfigPath := filepath.Join(configDir, "services.toml")
	err = servicesConfig.Save(servicesConfigPath)
	s.Require().NoError(err)

	// Initialize managers with test config
	s.configMgr = config.New()
	s.configMgr.Services = servicesConfig

	s.gitMgr = git.New(s.configMgr)
	s.containerMgr = container.New(s.configMgr)
	s.serviceMgr = service.New(s.configMgr)
	
	// Initialize operations with adapter
	adapter := &serviceAdapter{mgr: s.serviceMgr}
	s.serviceOps = operations.NewServiceOperations(s.configMgr, adapter)
}

func (s *ServiceLifecycleTestSuite) TearDownTest() {
	// Clean up any running services after each test
	ctx := context.Background()
	services, err := s.serviceOps.ListServices(ctx)
	if err == nil {
		for _, svc := range services {
			if svc.Status == service.StatusRunning {
				s.serviceMgr.StopService(ctx, svc.Name)
			}
		}
	}
}

func (s *ServiceLifecycleTestSuite) TearDownSuite() {
	// Stop all test services using service manager
	ctx := context.Background()
	services, _ := s.serviceOps.ListServices(ctx)
	for _, svc := range services {
		if svc.Status != service.StatusStopped {
			s.serviceMgr.StopService(ctx, svc.Name)
		}
	}

	// Also clean up any Docker Compose projects that might be left over
	// Docker Compose creates containers with project names based on directory
	if s.testDir != "" {
		projectName := filepath.Base(s.testDir)
		// Try to stop and remove the compose project
		cmd := exec.Command("docker", "compose", "-p", projectName, "down", "-v")
		cmd.Run() // Ignore errors - this is cleanup
	}

	// Clean up any containers that match our test pattern
	containers, _ := s.containerMgr.List(ctx)
	for _, c := range containers {
		if strings.Contains(c.Name, "vibeman-service-test") {
			s.containerMgr.Stop(ctx, c.ID)
			s.containerMgr.Remove(ctx, c.ID)
		}
	}

	if s.db != nil {
		s.db.Close()
	}
	os.RemoveAll(s.testDir)
}

func (s *ServiceLifecycleTestSuite) TestServiceStartStop() {
	ctx := context.Background()

	// List services - all should be stopped
	services, err := s.serviceOps.ListServices(ctx)
	s.NoError(err)
	s.Len(services, 2) // postgres and redis

	for _, svc := range services {
		s.Equal(service.StatusStopped, svc.Status)
	}

	// Start postgres service
	err = s.serviceOps.StartService(ctx, "test-postgres")
	s.NoError(err)

	// Wait for service to start and become healthy
	time.Sleep(10 * time.Second)

	// Verify postgres is running
	postgresInfo, err := s.serviceOps.GetService(ctx, "test-postgres")
	s.NoError(err)
	s.Equal(service.StatusRunning, postgresInfo.Status)
	s.NotEmpty(postgresInfo.ContainerID)

	// Start redis service
	err = s.serviceOps.StartService(ctx, "test-redis")
	s.NoError(err)

	// Wait for service to start and become healthy
	time.Sleep(10 * time.Second)

	// Verify both services are running
	services, err = s.serviceOps.ListServices(ctx)
	s.NoError(err)
	
	runningCount := 0
	for _, svc := range services {
		if svc.Status == service.StatusRunning {
			runningCount++
		}
	}
	s.Equal(2, runningCount)

	// Stop postgres
	err = s.serviceOps.StopService(ctx, "test-postgres")
	s.NoError(err)

	// Verify postgres is stopped but redis still running
	postgresInfo, err = s.serviceOps.GetService(ctx, "test-postgres")
	s.NoError(err)
	s.Equal(service.StatusStopped, postgresInfo.Status)

	redisInfo, err := s.serviceOps.GetService(ctx, "test-redis")
	s.NoError(err)
	s.Equal(service.StatusRunning, redisInfo.Status)

	// Stop redis
	err = s.serviceOps.StopService(ctx, "test-redis")
	s.NoError(err)
}

func (s *ServiceLifecycleTestSuite) TestServiceReferences() {
	ctx := context.Background()

	// Start postgres service
	err := s.serviceMgr.StartService(ctx, "test-postgres")
	s.NoError(err)

	// Add references from repositories
	err = s.serviceMgr.AddReference("test-postgres", "repo1")
	s.NoError(err)
	err = s.serviceMgr.AddReference("test-postgres", "repo2")
	s.NoError(err)

	// Get service info - should show references
	svcRaw, err := s.serviceMgr.GetService("test-postgres")
	s.NoError(err)
	svcInstance, ok := svcRaw.(*service.ServiceInstance)
	s.True(ok, "GetService should return *ServiceInstance")
	// StartService adds 1 ref, plus our 2 explicit refs = 3
	s.Equal(3, svcInstance.RefCount)
	s.Contains(svcInstance.Repositories, "repo1")
	s.Contains(svcInstance.Repositories, "repo2")

	// Remove one reference
	err = s.serviceMgr.RemoveReference("test-postgres", "repo1")
	s.NoError(err)

	// Service should still be running with 2 references (1 from start + 1 explicit)
	svcRaw, err = s.serviceMgr.GetService("test-postgres")
	s.NoError(err)
	svcInstance, ok = svcRaw.(*service.ServiceInstance)
	s.True(ok, "GetService should return *ServiceInstance")
	s.Equal(2, svcInstance.RefCount)
	s.Equal(service.StatusRunning, svcInstance.Status)

	// Remove second explicit reference - still has 1 from StartService
	err = s.serviceMgr.RemoveReference("test-postgres", "repo2")
	s.NoError(err)

	// Service should still be running with 1 reference
	svcRaw, err = s.serviceMgr.GetService("test-postgres")
	s.NoError(err)
	svcInstance, ok = svcRaw.(*service.ServiceInstance)
	s.True(ok, "GetService should return *ServiceInstance")
	s.Equal(1, svcInstance.RefCount)
	s.Equal(service.StatusRunning, svcInstance.Status)
	
	// Now stop the service which should remove the implicit reference
	err = s.serviceMgr.StopService(ctx, "test-postgres")
	s.NoError(err)
	
	// Verify service is stopped
	postgresInfo, err := s.serviceOps.GetService(ctx, "test-postgres")
	s.NoError(err)
	s.Equal(service.StatusStopped, postgresInfo.Status)
}

func (s *ServiceLifecycleTestSuite) TestHealthChecks() {
	ctx := context.Background()

	// Start redis with health check
	err := s.serviceOps.StartService(ctx, "test-redis")
	s.NoError(err)

	// Wait for initial health check
	time.Sleep(15 * time.Second)

	// Service should be healthy
	redisInfo, err := s.serviceOps.GetService(ctx, "test-redis")
	s.NoError(err)
	s.Equal(service.StatusRunning, redisInfo.Status)
	s.Empty(redisInfo.HealthError)

	// Stop the service
	err = s.serviceOps.StopService(ctx, "test-redis")
	s.NoError(err)
}

// Helper methods

func (s *ServiceLifecycleTestSuite) isDockerAvailable() bool {
	// Check if docker is available by running docker ps
	cmd := exec.Command("docker", "ps")
	err := cmd.Run()
	return err == nil
}

// serviceAdapter adapts service.Manager to operations.ServiceManager interface
type serviceAdapter struct {
	mgr *service.Manager
}

func (a *serviceAdapter) StartService(ctx context.Context, name string) error {
	return a.mgr.StartService(ctx, name)
}

func (a *serviceAdapter) StopService(ctx context.Context, name string) error {
	return a.mgr.StopService(ctx, name)
}

func (a *serviceAdapter) GetService(name string) (interface{}, error) {
	return a.mgr.GetService(name)
}

func (a *serviceAdapter) HealthCheck(ctx context.Context, name string) error {
	return a.mgr.HealthCheck(ctx, name)
}

func (a *serviceAdapter) AddReference(serviceName, repoName string) error {
	return a.mgr.AddReference(serviceName, repoName)
}

func (a *serviceAdapter) RemoveReference(serviceName, repoName string) error {
	return a.mgr.RemoveReference(serviceName, repoName)
}

// TestServiceLifecycleIntegration runs the service integration test suite
func TestServiceLifecycleIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	suite.Run(t, new(ServiceLifecycleTestSuite))
}