package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vibeman/internal/container"
	"vibeman/internal/db"
	"vibeman/internal/testutil"
	"vibeman/internal/types"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleGetWorktreeLogs(t *testing.T) {
	tests := []struct {
		name           string
		worktreeID     string
		setupData      func(t *testing.T, dbInstance *db.DB) string // returns temp dir path
		queryParams    map[string]string
		expectedStatus int
		checkResponse  func(t *testing.T, body string, tempDir string)
	}{
		{
			name:       "successful log retrieval",
			worktreeID: "worktree-123",
			setupData: func(t *testing.T, dbInstance *db.DB) string {
				ctx := context.Background()
				
				// Create test repository
				repoRepo := db.NewRepositoryRepository(dbInstance)
				repo := &db.Repository{
					ID:   "repo-123",
					Name: "test-repo",
					Path: "/tmp/test-repo",
				}
				require.NoError(t, repoRepo.Create(ctx, repo))
				
				// Create test worktree
				worktreeRepo := db.NewWorktreeRepository(dbInstance)
				worktree := &db.Worktree{
					ID:           "worktree-123",
					RepositoryID: "repo-123",
					Name:         "feature-test",
					Branch:       "feature/test",
					Path:         "/tmp/test-worktree",
					Status:       db.StatusRunning,
				}
				require.NoError(t, worktreeRepo.Create(ctx, worktree))
				
				// Create temporary log directory and file
				tempDir := t.TempDir()
				oldHome := os.Getenv("HOME")
				os.Setenv("HOME", tempDir)
				t.Cleanup(func() { os.Setenv("HOME", oldHome) })
				
				logsDir := filepath.Join(tempDir, ".local", "share", "vibeman", "logs", "test-repo", "feature-test")
				require.NoError(t, os.MkdirAll(logsDir, 0755))
				
				logFile := filepath.Join(logsDir, "worktree.log")
				logContent := "Log line 1\nLog line 2\nLog line 3\n\n"
				require.NoError(t, os.WriteFile(logFile, []byte(logContent), 0644))
				
				return tempDir
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string, tempDir string) {
				var response LogsResponse
				require.NoError(t, json.Unmarshal([]byte(body), &response))
				
				assert.Equal(t, "worktree", response.Source)
				assert.Equal(t, "worktree-123", response.ID)
				assert.Equal(t, 3, response.Lines)
				assert.Len(t, response.Logs, 3)
				assert.Equal(t, "Log line 1", response.Logs[0])
				assert.Equal(t, "Log line 2", response.Logs[1])
				assert.Equal(t, "Log line 3", response.Logs[2])
			},
		},
		{
			name:       "worktree with no logs",
			worktreeID: "worktree-456",
			setupData: func(t *testing.T, dbInstance *db.DB) string {
				ctx := context.Background()
				
				// Create test repository
				repoRepo := db.NewRepositoryRepository(dbInstance)
				repo := &db.Repository{
					ID:   "repo-456",
					Name: "empty-repo",
					Path: "/tmp/empty-repo",
				}
				require.NoError(t, repoRepo.Create(ctx, repo))
				
				// Create test worktree
				worktreeRepo := db.NewWorktreeRepository(dbInstance)
				worktree := &db.Worktree{
					ID:           "worktree-456",
					RepositoryID: "repo-456",
					Name:         "main",
					Branch:       "main",
					Path:         "/tmp/empty-worktree",
					Status:       db.StatusRunning,
				}
				require.NoError(t, worktreeRepo.Create(ctx, worktree))
				
				// Don't create log file
				tempDir := t.TempDir()
				oldHome := os.Getenv("HOME")
				os.Setenv("HOME", tempDir)
				t.Cleanup(func() { os.Setenv("HOME", oldHome) })
				
				return tempDir
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string, tempDir string) {
				var response LogsResponse
				require.NoError(t, json.Unmarshal([]byte(body), &response))
				
				assert.Equal(t, "worktree", response.Source)
				assert.Equal(t, "worktree-456", response.ID)
				assert.Equal(t, 1, response.Lines)
				assert.Len(t, response.Logs, 1)
				assert.Equal(t, "No logs available for this worktree", response.Logs[0])
			},
		},
		{
			name:       "limit log lines",
			worktreeID: "worktree-789",
			setupData: func(t *testing.T, dbInstance *db.DB) string {
				ctx := context.Background()
				
				// Create test repository
				repoRepo := db.NewRepositoryRepository(dbInstance)
				repo := &db.Repository{
					ID:   "repo-789",
					Name: "large-repo",
					Path: "/tmp/large-repo",
				}
				require.NoError(t, repoRepo.Create(ctx, repo))
				
				// Create test worktree
				worktreeRepo := db.NewWorktreeRepository(dbInstance)
				worktree := &db.Worktree{
					ID:           "worktree-789",
					RepositoryID: "repo-789",
					Name:         "develop",
					Branch:       "develop",
					Path:         "/tmp/large-worktree",
					Status:       db.StatusRunning,
				}
				require.NoError(t, worktreeRepo.Create(ctx, worktree))
				
				// Create temporary log directory and file with many lines
				tempDir := t.TempDir()
				oldHome := os.Getenv("HOME")
				os.Setenv("HOME", tempDir)
				t.Cleanup(func() { os.Setenv("HOME", oldHome) })
				
				logsDir := filepath.Join(tempDir, ".local", "share", "vibeman", "logs", "large-repo", "develop")
				require.NoError(t, os.MkdirAll(logsDir, 0755))
				
				logFile := filepath.Join(logsDir, "worktree.log")
				logContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10\n"
				require.NoError(t, os.WriteFile(logFile, []byte(logContent), 0644))
				
				return tempDir
			},
			queryParams:    map[string]string{"lines": "3"},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string, tempDir string) {
				var response LogsResponse
				require.NoError(t, json.Unmarshal([]byte(body), &response))
				
				assert.Equal(t, "worktree", response.Source)
				assert.Equal(t, "worktree-789", response.ID)
				assert.Equal(t, 3, response.Lines)
				assert.Len(t, response.Logs, 3)
				// Should get the last 3 lines
				assert.Equal(t, "Line 8", response.Logs[0])
				assert.Equal(t, "Line 9", response.Logs[1])
				assert.Equal(t, "Line 10", response.Logs[2])
			},
		},
		{
			name:       "worktree not found",
			worktreeID: "nonexistent",
			setupData:  func(t *testing.T, dbInstance *db.DB) string { return t.TempDir() },
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body string, tempDir string) {
				assert.Contains(t, body, "Failed to get worktree")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbInstance := testutil.SetupTestDB(t)
			tempDir := tt.setupData(t, dbInstance)
			
			server := &Server{
				db: dbInstance,
			}
			
			e := echo.New()
			e.HTTPErrorHandler = ErrorHandler
			req := httptest.NewRequest(http.MethodGet, "/api/worktrees/"+tt.worktreeID+"/logs", nil)
			
			// Add query parameters
			if len(tt.queryParams) > 0 {
				q := req.URL.Query()
				for key, value := range tt.queryParams {
					q.Add(key, value)
				}
				req.URL.RawQuery = q.Encode()
			}
			
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/worktrees/:id/logs")
			c.SetParamNames("id")
			c.SetParamValues(tt.worktreeID)
			
			err := server.handleGetWorktreeLogs(c)
			
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}
			
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.checkResponse(t, rec.Body.String(), tempDir)
		})
	}
}

func TestHandleGetServiceLogs(t *testing.T) {
	tests := []struct {
		name           string
		serviceID      string
		setupMocks     func(*testutil.MockServiceManager, *testutil.MockContainerManager) string // returns temp dir
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:      "service logs from file",
			serviceID: "postgres",
			setupMocks: func(serviceMgr *testutil.MockServiceManager, containerMgr *testutil.MockContainerManager) string {
				// Mock service exists
				serviceMgr.GetServiceFn = func(ctx context.Context, name string) (*types.ServiceInstance, error) {
					if name == "postgres" {
						return &types.ServiceInstance{
							Name:   "postgres",
							Status: types.ServiceStatusRunning,
						}, nil
					}
					return nil, fmt.Errorf("service not found")
				}
				
				// Create temporary log directory and file
				tempDir := t.TempDir()
				oldHome := os.Getenv("HOME")
				os.Setenv("HOME", tempDir)
				t.Cleanup(func() { os.Setenv("HOME", oldHome) })
				
				logsDir := filepath.Join(tempDir, ".local", "share", "vibeman", "logs", "services")
				require.NoError(t, os.MkdirAll(logsDir, 0755))
				
				logFile := filepath.Join(logsDir, "postgres.log")
				logContent := "PostgreSQL log line 1\nPostgreSQL log line 2\nPostgreSQL log line 3\n"
				require.NoError(t, os.WriteFile(logFile, []byte(logContent), 0644))
				
				return tempDir
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var response LogsResponse
				require.NoError(t, json.Unmarshal([]byte(body), &response))
				
				assert.Equal(t, "service", response.Source)
				assert.Equal(t, "postgres", response.ID)
				assert.Equal(t, 3, response.Lines)
				assert.Len(t, response.Logs, 3)
				assert.Equal(t, "PostgreSQL log line 1", response.Logs[0])
				assert.Equal(t, "PostgreSQL log line 2", response.Logs[1])
				assert.Equal(t, "PostgreSQL log line 3", response.Logs[2])
			},
		},
		{
			name:      "service logs from container",
			serviceID: "redis",
			setupMocks: func(serviceMgr *testutil.MockServiceManager, containerMgr *testutil.MockContainerManager) string {
				// Mock service exists
				serviceMgr.GetServiceFn = func(ctx context.Context, name string) (*types.ServiceInstance, error) {
					if name == "redis" {
						return &types.ServiceInstance{
							Name:   "redis",
							Status: types.ServiceStatusRunning,
						}, nil
					}
					return nil, fmt.Errorf("service not found")
				}
				
				// Mock container exists with same name as service
				containerMgr.GetByNameFn = func(ctx context.Context, name string) (*container.Container, error) {
					if name == "redis" {
						return &container.Container{
							ID:   "redis-container-123",
							Name: "redis",
						}, nil
					}
					return nil, fmt.Errorf("container not found")
				}
				
				// Mock container logs
				containerMgr.LogsFn = func(ctx context.Context, containerID string, follow bool) ([]byte, error) {
					if containerID == "redis-container-123" {
						return []byte("Redis container log 1\nRedis container log 2\n"), nil
					}
					return nil, fmt.Errorf("container not found")
				}
				
				// Don't create log file - should fall back to container logs
				tempDir := t.TempDir()
				oldHome := os.Getenv("HOME")
				os.Setenv("HOME", tempDir)
				t.Cleanup(func() { os.Setenv("HOME", oldHome) })
				
				return tempDir
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var response LogsResponse
				require.NoError(t, json.Unmarshal([]byte(body), &response))
				
				assert.Equal(t, "service", response.Source)
				assert.Equal(t, "redis", response.ID)
				assert.Equal(t, 2, response.Lines)
				assert.Len(t, response.Logs, 2)
				assert.Equal(t, "Redis container log 1", response.Logs[0])
				assert.Equal(t, "Redis container log 2", response.Logs[1])
			},
		},
		{
			name:      "service with no logs",
			serviceID: "localstack",
			setupMocks: func(serviceMgr *testutil.MockServiceManager, containerMgr *testutil.MockContainerManager) string {
				// Mock service exists
				serviceMgr.GetServiceFn = func(ctx context.Context, name string) (*types.ServiceInstance, error) {
					if name == "localstack" {
						return &types.ServiceInstance{
							Name:   "localstack",
							Status: types.ServiceStatusRunning,
						}, nil
					}
					return nil, fmt.Errorf("service not found")
				}
				
				// Mock container not found
				containerMgr.GetByNameFn = func(ctx context.Context, name string) (*container.Container, error) {
					return nil, fmt.Errorf("container not found")
				}
				
				// Don't create log file
				tempDir := t.TempDir()
				oldHome := os.Getenv("HOME")
				os.Setenv("HOME", tempDir)
				t.Cleanup(func() { os.Setenv("HOME", oldHome) })
				
				return tempDir
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var response LogsResponse
				require.NoError(t, json.Unmarshal([]byte(body), &response))
				
				assert.Equal(t, "service", response.Source)
				assert.Equal(t, "localstack", response.ID)
				assert.Equal(t, 1, response.Lines)
				assert.Len(t, response.Logs, 1)
				assert.Equal(t, "No logs available for this service", response.Logs[0])
			},
		},
		{
			name:      "service not found",
			serviceID: "nonexistent",
			setupMocks: func(serviceMgr *testutil.MockServiceManager, containerMgr *testutil.MockContainerManager) string {
				serviceMgr.GetServiceFn = func(ctx context.Context, name string) (*types.ServiceInstance, error) {
					return nil, fmt.Errorf("service not found")
				}
				return t.TempDir()
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Failed to get service")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceMgr := &testutil.MockServiceManager{}
			containerMgr := &testutil.MockContainerManager{}
			tt.setupMocks(serviceMgr, containerMgr)
			
			server := &Server{
				serviceMgr:   serviceMgr,
				containerMgr: containerMgr,
			}
			
			e := echo.New()
			e.HTTPErrorHandler = ErrorHandler
			req := httptest.NewRequest(http.MethodGet, "/api/services/"+tt.serviceID+"/logs", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/services/:id/logs")
			c.SetParamNames("id")
			c.SetParamValues(tt.serviceID)
			
			err := server.handleGetServiceLogs(c)
			
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}
			
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.checkResponse(t, rec.Body.String())
		})
	}
}

func TestLogsParsing(t *testing.T) {
	t.Run("filter empty lines", func(t *testing.T) {
		logContent := "Line 1\n\nLine 2\n\n\nLine 3\n\n"
		lines := strings.Split(logContent, "\n")
		
		var filteredLines []string
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				filteredLines = append(filteredLines, line)
			}
		}
		
		assert.Len(t, filteredLines, 3)
		assert.Equal(t, "Line 1", filteredLines[0])
		assert.Equal(t, "Line 2", filteredLines[1])
		assert.Equal(t, "Line 3", filteredLines[2])
	})
	
	t.Run("apply lines limit", func(t *testing.T) {
		logLines := []string{"Line 1", "Line 2", "Line 3", "Line 4", "Line 5"}
		limit := 3
		
		if limit < len(logLines) {
			logLines = logLines[len(logLines)-limit:]
		}
		
		assert.Len(t, logLines, 3)
		assert.Equal(t, "Line 3", logLines[0])
		assert.Equal(t, "Line 4", logLines[1])
		assert.Equal(t, "Line 5", logLines[2])
	})
}