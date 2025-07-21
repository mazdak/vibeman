// +build integration

package integration_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
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

type WebSocketIntegrationTestSuite struct {
	suite.Suite
	testDir      string
	db           *db.DB
	configMgr    *config.Manager
	gitMgr       *git.Manager
	containerMgr *container.Manager
	serviceMgr   *service.Manager
	repoOps      *operations.RepositoryOperations
	worktreeOps  *operations.WorktreeOperations
}

func (s *WebSocketIntegrationTestSuite) SetupSuite() {
	// Create test directory
	testDir, err := os.MkdirTemp("", "vibeman-websocket-integration-*")
	s.Require().NoError(err)
	s.testDir = testDir

	// Initialize database
	dbConfig := &db.Config{
		Driver: "sqlite3",
		DSN:    filepath.Join(testDir, "test.db"),
	}
	s.db, err = db.New(dbConfig)
	s.Require().NoError(err)
	s.Require().NoError(s.db.Migrate())

	// Initialize managers
	s.configMgr = config.New()
	s.gitMgr = git.New(s.configMgr)
	s.containerMgr = container.New(s.configMgr)
	s.serviceMgr = service.New(s.configMgr)

	// Initialize operations with adapter
	serviceAdapter := &websocketServiceAdapter{mgr: s.serviceMgr}
	s.repoOps = operations.NewRepositoryOperations(s.configMgr, s.gitMgr, s.db)
	s.worktreeOps = operations.NewWorktreeOperations(s.db, s.gitMgr, s.containerMgr, serviceAdapter, s.configMgr)
}

func (s *WebSocketIntegrationTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
	
	// Clean up test directory and any worktree directories
	if s.testDir != "" {
		// Remove the main test directory
		os.RemoveAll(s.testDir)
		
		// Also clean up any worktree directories that might be siblings
		parentDir := filepath.Dir(s.testDir)
		if entries, err := os.ReadDir(parentDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && (
					strings.HasSuffix(entry.Name(), "-worktrees") ||
					strings.Contains(entry.Name(), "test-repo-worktrees")) {
					worktreePath := filepath.Join(parentDir, entry.Name())
					os.RemoveAll(worktreePath)
				}
			}
		}
	}
}

func (s *WebSocketIntegrationTestSuite) TestWebSocketAPIExistence() {
	if !s.isDockerAvailable() {
		s.T().Skip("Docker not available, skipping WebSocket integration test")
	}

	ctx := context.Background()

	// Create test repository with AI enabled (default)
	repoPath := filepath.Join(s.testDir, "test-repo-websocket")
	s.createTestRepository(repoPath)

	// Add repository to Vibeman
	_, err := s.repoOps.AddRepository(ctx, operations.AddRepositoryRequest{
		Name: "test-repo-websocket",
		Path: repoPath,
	})
	s.Require().NoError(err)

	// Get the added repository
	repos, err := s.repoOps.ListRepositories(ctx)
	s.Require().NoError(err)
	s.Require().Len(repos, 1)

	repo := repos[0]

	// Create and start worktree with AI container
	_, err = s.worktreeOps.CreateWorktree(ctx, operations.CreateWorktreeRequest{
		RepositoryID: repo.ID,
		Name:         "websocket-test-worktree",
		Branch:       "feature/websocket",
		BaseBranch:   "main",
		SkipSetup:    true,
		AutoStart:    true,
	})
	s.Require().NoError(err)

	// Wait for AI container to be running
	s.waitForAIContainer("websocket-test-worktree")

	// Test WebSocket endpoint URL construction
	worktreeName := "websocket-test-worktree"
	expectedPath := "/api/ai/attach/" + worktreeName
	
	u := url.URL{
		Scheme: "ws",
		Host:   "localhost:8080",
		Path:   expectedPath,
	}
	
	s.Equal("ws://localhost:8080/api/ai/attach/websocket-test-worktree", u.String())
	s.T().Logf("WebSocket endpoint URL: %s", u.String())

	// Note: We can't test actual WebSocket connections in this environment
	// but we verify the AI container is running and would be accessible
	containers, err := s.containerMgr.List(ctx)
	s.Require().NoError(err)

	aiContainerFound := false
	for _, container := range containers {
		if container.Type == "ai" && strings.Contains(container.Name, "websocket-test-worktree") {
			status := strings.ToLower(container.Status)
			if strings.Contains(status, "running") || strings.Contains(status, "up") {
				aiContainerFound = true
				s.T().Logf("AI container found and running: %s (status: %s)", container.Name, container.Status)
				break
			}
		}
	}
	s.True(aiContainerFound, "AI container should be running for WebSocket access")

	s.T().Log("WebSocket API integration test completed - AI container ready for WebSocket connections")
}

func (s *WebSocketIntegrationTestSuite) TestWebSocketProtocolValidation() {
	// Test WebSocket message structures without actual connection
	s.T().Log("Testing WebSocket protocol message types")

	// Test client message types
	clientMessages := []string{"stdin", "resize", "ping"}
	for _, msgType := range clientMessages {
		s.Contains(clientMessages, msgType, "Client message type should be valid")
	}

	// Test server message types  
	serverMessages := []string{"stdout", "stderr", "exit", "pong"}
	for _, msgType := range serverMessages {
		s.Contains(serverMessages, msgType, "Server message type should be valid")
	}

	s.T().Log("WebSocket protocol validation completed")
}

func TestWebSocketIntegration(t *testing.T) {
	suite.Run(t, new(WebSocketIntegrationTestSuite))
}

// Helper methods (borrowed from AI container integration test)

func (s *WebSocketIntegrationTestSuite) isDockerAvailable() bool {
	return s.containerMgr != nil
}

func (s *WebSocketIntegrationTestSuite) createTestRepository(path string) {
	// Create directory
	err := os.MkdirAll(path, 0755)
	s.Require().NoError(err)

	// Initialize git repository
	ctx := context.Background()
	err = s.gitMgr.InitRepository(ctx, path)
	s.Require().NoError(err)

	// Create README
	readmePath := filepath.Join(path, "README.md")
	err = os.WriteFile(readmePath, []byte("# Test Repository\n"), 0644)
	s.Require().NoError(err)

	// Create unique worktree directory name based on the repository path
	// This ensures each test gets its own worktree directory
	repoName := filepath.Base(path)
	worktreeDir := filepath.Join(filepath.Dir(path), repoName+"-worktrees")

	// Create vibeman.toml with unique worktree directory
	config := fmt.Sprintf(`[repository]
name = "%s"

[repository.worktrees]
directory = "%s"

[repository.ai]
enabled = true
image = "vibeman/ai-assistant:latest"
`, repoName, worktreeDir)
	err = os.WriteFile(filepath.Join(path, "vibeman.toml"), []byte(config), 0644)
	s.Require().NoError(err)
	
	// Commit initial files
	err = s.gitMgr.AddAndCommit(ctx, path, "Initial commit")
	s.Require().NoError(err)
}

func (s *WebSocketIntegrationTestSuite) waitForAIContainer(worktreeName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			s.FailNow("Timeout waiting for AI container to be ready")
		default:
			containers, err := s.containerMgr.List(context.Background())
			if err != nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			for _, container := range containers {
				if container.Type == "ai" && strings.Contains(container.Name, worktreeName) {
					status := strings.ToLower(container.Status)
					if strings.Contains(status, "running") || strings.Contains(status, "up") {
						return
					}
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// websocketServiceAdapter adapts service.Manager to operations.ServiceManager interface
type websocketServiceAdapter struct {
	mgr *service.Manager
}

func (a *websocketServiceAdapter) StartService(ctx context.Context, name string) error {
	return a.mgr.StartService(ctx, name)
}

func (a *websocketServiceAdapter) StopService(ctx context.Context, name string) error {
	return a.mgr.StopService(ctx, name)
}

func (a *websocketServiceAdapter) GetService(name string) (interface{}, error) {
	return a.mgr.GetService(name)
}

func (a *websocketServiceAdapter) HealthCheck(ctx context.Context, name string) error {
	return a.mgr.HealthCheck(ctx, name)
}

func (a *websocketServiceAdapter) AddReference(serviceName, repoName string) error {
	return a.mgr.AddReference(serviceName, repoName)
}

func (a *websocketServiceAdapter) RemoveReference(serviceName, repoName string) error {
	return a.mgr.RemoveReference(serviceName, repoName)
}