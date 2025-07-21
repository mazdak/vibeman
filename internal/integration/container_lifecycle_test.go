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

	"github.com/stretchr/testify/suite"
)

// ContainerLifecycleTestSuite tests container operations
type ContainerLifecycleTestSuite struct {
	suite.Suite
	testDir      string
	configMgr    *config.Manager
	containerMgr *container.Manager
}

func (s *ContainerLifecycleTestSuite) SetupSuite() {
	// Skip if Docker is not available
	if !s.isDockerAvailable() {
		s.T().Skip("Docker is not available, skipping container tests")
	}

	// Create test directory
	testDir, err := os.MkdirTemp("", "vibeman-container-test-*")
	s.Require().NoError(err)
	s.testDir = testDir

	// Initialize managers
	s.configMgr = config.New()
	s.containerMgr = container.New(s.configMgr)
}

func (s *ContainerLifecycleTestSuite) TearDownSuite() {
	// Clean up any test containers
	ctx := context.Background()
	containers, _ := s.containerMgr.List(ctx)
	for _, c := range containers {
		if c.Name != "" && (c.Name == "vibeman-test-container" || c.Name == "vibeman-test-compose") {
			s.containerMgr.Remove(ctx, c.ID)
		}
	}
	
	os.RemoveAll(s.testDir)
}

func (s *ContainerLifecycleTestSuite) TestBasicContainerLifecycle() {
	ctx := context.Background()
	
	// Create a container
	container, err := s.containerMgr.Create(ctx, "test-repo", "test-env", "alpine:latest")
	s.Require().NoError(err)
	s.NotNil(container)
	s.Equal("test-repo-test-env", container.Name)
	s.Equal("Created", string(container.Status))
	
	// Start the container
	err = s.containerMgr.Start(ctx, container.ID)
	s.NoError(err)
	
	// Wait a bit for container to start
	time.Sleep(2 * time.Second)
	
	// Verify container is running
	containers, err := s.containerMgr.List(ctx)
	s.NoError(err)
	
	var found bool
	for _, c := range containers {
		// Docker ps returns truncated IDs, so check if one is a prefix of the other
		if strings.HasPrefix(container.ID, c.ID) || strings.HasPrefix(c.ID, container.ID) {
			found = true
			// Docker returns "Up X seconds" for running containers
			s.Contains(string(c.Status), "Up")
			break
		}
	}
	s.True(found, "Container should be in the list")
	
	// Execute a command in the container
	output, err := s.containerMgr.Exec(ctx, container.ID, []string{"echo", "hello"})
	s.NoError(err)
	s.Contains(string(output), "hello")
	
	// Stop the container
	err = s.containerMgr.Stop(ctx, container.ID)
	s.NoError(err)
	
	// Remove the container
	err = s.containerMgr.Remove(ctx, container.ID)
	s.NoError(err)
	
	// Verify container is gone
	containers, err = s.containerMgr.List(ctx)
	s.NoError(err)
	for _, c := range containers {
		// Docker ps returns truncated IDs, so check prefixes
		s.False(strings.HasPrefix(container.ID, c.ID) || strings.HasPrefix(c.ID, container.ID), 
			"Container should not be in the list after removal")
	}
}

func (s *ContainerLifecycleTestSuite) TestDockerComposeIntegration() {
	ctx := context.Background()
	
	// Create a simple docker-compose.yaml
	composeContent := `version: '3'
services:
  test:
    image: alpine:latest
    command: sleep 3600
    environment:
      - TEST_ENV=vibeman
`
	composePath := filepath.Join(s.testDir, "docker-compose.yaml")
	err := os.WriteFile(composePath, []byte(composeContent), 0644)
	s.Require().NoError(err)
	
	// Update config to use compose file
	repoConfig := &config.RepositoryConfig{}
	repoConfig.Repository.Name = "test-compose-repo"
	repoConfig.Repository.Container.ComposeFile = composePath
	repoConfig.Repository.Container.Services = []string{"test"}  // Use Services list
	s.configMgr.Repository = repoConfig
	
	// Create container using compose
	container, err := s.containerMgr.Create(ctx, "test-compose-repo", "dev", "")
	s.Require().NoError(err)
	s.NotNil(container)
	
	// Verify container is created
	s.NotEmpty(container.ID)
	s.Contains(container.Name, "test")
	
	// Clean up
	err = s.containerMgr.Stop(ctx, container.ID)
	s.NoError(err)
	err = s.containerMgr.Remove(ctx, container.ID)
	s.NoError(err)
}

func (s *ContainerLifecycleTestSuite) TestContainerErrorHandling() {
	ctx := context.Background()
	
	// Test creating container with invalid image
	_, err := s.containerMgr.Create(ctx, "test-repo", "test-env", "this-image-does-not-exist:latest")
	s.Error(err)
	
	// Verify we get a structured error
	var containerErr *container.ContainerError
	if s.ErrorAs(err, &containerErr) {
		s.Equal(container.ErrorTypeImageNotFound, containerErr.Type)
		s.Contains(containerErr.Message, "image not found")
	}
	
	// Test operations on non-existent container
	err = s.containerMgr.Start(ctx, "non-existent-container-id")
	s.Error(err)
	if s.ErrorAs(err, &containerErr) {
		s.Equal(container.ErrorTypeContainerNotFound, containerErr.Type)
	}
}

func (s *ContainerLifecycleTestSuite) TestGetByName() {
	ctx := context.Background()
	
	// Create a container with a specific name
	container, err := s.containerMgr.Create(ctx, "test-repo", "named-env", "alpine:latest")
	s.Require().NoError(err)
	defer s.containerMgr.Remove(ctx, container.ID)
	
	// Get container by name
	found, err := s.containerMgr.GetByName(ctx, container.Name)
	s.NoError(err)
	s.NotNil(found)
	// Compare IDs considering truncation
	s.True(strings.HasPrefix(container.ID, found.ID) || strings.HasPrefix(found.ID, container.ID),
		"Container IDs should match (considering truncation)")
	s.Equal(container.Name, found.Name)
	
	// Test getting non-existent container
	_, err = s.containerMgr.GetByName(ctx, "non-existent-container")
	s.Error(err)
}

// Helper methods

func (s *ContainerLifecycleTestSuite) isDockerAvailable() bool {
	// Check if docker command exists
	_, err := exec.LookPath("docker")
	if err != nil {
		return false
	}
	
	// Try to run docker version
	cmd := exec.Command("docker", "version")
	return cmd.Run() == nil
}

// TestContainerLifecycleIntegration runs the container integration test suite
func TestContainerLifecycleIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	suite.Run(t, new(ContainerLifecycleTestSuite))
}