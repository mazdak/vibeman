package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vibeman/internal/container"
	"vibeman/internal/errors"
	"vibeman/internal/testutil"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleListContainers(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*testutil.MockContainerManager)
		queryParams    map[string]string
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name: "list all containers",
			setupMocks: func(mockMgr *testutil.MockContainerManager) {
				mockMgr.ListReturn = []*container.Container{
					{
						ID:          "container-1",
						Name:        "vibeman-myapp-dev",
						Image:       "node:18-alpine",
						Status:      "running",
						Repository:  "myapp",
						Environment: "dev",
						Ports:       map[string]string{"8080": "3000"},
						CreatedAt:   time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
					},
					{
						ID:          "container-2",
						Name:        "vibeman-backend-api",
						Image:       "golang:1.21",
						Status:      "stopped",
						Repository:  "backend",
						Environment: "api",
						Ports:       map[string]string{},
						CreatedAt:   time.Now().Add(-3 * time.Hour).Format(time.RFC3339),
					},
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var response ContainersResponse
				require.NoError(t, json.Unmarshal([]byte(body), &response))
				
				assert.Equal(t, 2, response.Total)
				assert.Len(t, response.Containers, 2)
				
				// Check first container
				container1 := response.Containers[0]
				assert.Equal(t, "container-1", container1.ID)
				assert.Equal(t, "vibeman-myapp-dev", container1.Name)
				assert.Equal(t, "node:18-alpine", container1.Image)
				assert.Equal(t, "running", container1.Status)
				assert.Equal(t, "running", container1.State) // State is same as status
				assert.Contains(t, container1.Ports, "8080:3000")
				assert.Equal(t, "myapp", container1.Repository)
				assert.Equal(t, "dev", container1.Worktree) // Environment maps to worktree
			},
		},
		{
			name: "filter by repository",
			setupMocks: func(mockMgr *testutil.MockContainerManager) {
				mockMgr.ListReturn = []*container.Container{
					{
						ID:          "container-1",
						Name:        "vibeman-myapp-dev",
						Repository:  "myapp",
						Environment: "dev",
					},
					{
						ID:          "container-2", 
						Name:        "vibeman-backend-api",
						Repository:  "backend",
						Environment: "api",
					},
				}
			},
			queryParams:    map[string]string{"repository": "myapp"},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var response ContainersResponse
				require.NoError(t, json.Unmarshal([]byte(body), &response))
				
				assert.Equal(t, 1, response.Total)
				assert.Len(t, response.Containers, 1)
				assert.Equal(t, "container-1", response.Containers[0].ID)
			},
		},
		{
			name: "empty container list",
			setupMocks: func(mockMgr *testutil.MockContainerManager) {
				mockMgr.ListReturn = []*container.Container{}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var response ContainersResponse
				require.NoError(t, json.Unmarshal([]byte(body), &response))
				
				assert.Equal(t, 0, response.Total)
				assert.Len(t, response.Containers, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := testutil.NewMockContainerManager()
			tt.setupMocks(mockMgr)
			
			server := &Server{
				containerMgr: mockMgr,
			}
			
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/containers", nil)
			
			// Add query parameters
			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()
			
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			
			err := server.handleListContainers(c)
			require.NoError(t, err)
			
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.checkResponse(t, rec.Body.String())
		})
	}
}

func TestHandleCreateContainer(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    CreateContainerRequest
		setupMocks     func(*testutil.MockContainerManager)
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name: "successful container creation",
			requestBody: CreateContainerRequest{
				Repository: "myapp",
				Worktree:   "feature-auth",
				Image:      "node:18-alpine",
				Ports:      []string{"8080:3000"},
				Env:        map[string]string{"NODE_ENV": "development"},
				AutoStart:  true,
			},
			setupMocks: func(mockMgr *testutil.MockContainerManager) {
				mockMgr.CreateFn = func(ctx context.Context, config container.CreateConfig) (*container.Container, error) {
					return &container.Container{
						ID:     "new-container-id",
						Name:   config.Name,
						Image:  config.Image,
						Status: "created",
					}, nil
				}
				mockMgr.StartFn = func(ctx context.Context, containerID string) error {
					return nil
				}
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body string) {
				var response ContainerResponse
				require.NoError(t, json.Unmarshal([]byte(body), &response))
				
				assert.Equal(t, "new-container-id", response.ID)
				assert.Contains(t, response.Name, "vibeman-myapp-feature-auth")
				assert.Equal(t, "node:18-alpine", response.Image)
			},
		},
		{
			name: "invalid request body",
			requestBody: CreateContainerRequest{
				// Missing required fields
				Repository: "",
				Image:      "",
			},
			setupMocks:     func(*testutil.MockContainerManager) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "repository is required")
			},
		},
		{
			name: "container creation failure",
			requestBody: CreateContainerRequest{
				Repository: "myapp",
				Image:      "invalid-image",
			},
			setupMocks: func(mockMgr *testutil.MockContainerManager) {
				mockMgr.CreateFn = func(ctx context.Context, config container.CreateConfig) (*container.Container, error) {
					return nil, fmt.Errorf("image not found: invalid-image")
				}
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Failed to create container")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := testutil.NewMockContainerManager()
			tt.setupMocks(mockMgr)
			
			server := &Server{
				config:       DefaultConfig(),
				echo:         echo.New(),
				containerMgr: mockMgr,
				startTime:    time.Now(),
			}
			
			// Marshal request body
			reqBody, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)
			
			e := server.Echo()
			req := httptest.NewRequest(http.MethodPost, "/api/containers", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			
			err = server.handleCreateContainer(c)
			
			if tt.expectedStatus == http.StatusCreated {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				// Manually trigger error handler since we're calling handler directly
				e.HTTPErrorHandler(err, c)
			}
			
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.checkResponse(t, rec.Body.String())
		})
	}
}

func TestHandleGetContainer(t *testing.T) {
	tests := []struct {
		name           string
		containerID    string
		setupMocks     func(*testutil.MockContainerManager)
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:        "existing container",
			containerID: "container-123",
			setupMocks: func(mockMgr *testutil.MockContainerManager) {
				mockMgr.GetByNameFn = func(ctx context.Context, name string) (*container.Container, error) {
					if name == "container-123" {
						return &container.Container{
							ID:          "container-123",
							Name:        "vibeman-myapp-dev",
							Image:       "node:18-alpine",
							Status:      "running",
							Repository:  "myapp",
							Environment: "dev",
							Ports:       map[string]string{"8080": "3000"},
							CreatedAt:   time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
						}, nil
					}
					return nil, fmt.Errorf("container not found")
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var response ContainerResponse
				require.NoError(t, json.Unmarshal([]byte(body), &response))
				
				assert.Equal(t, "container-123", response.ID)
				assert.Equal(t, "vibeman-myapp-dev", response.Name)
				assert.Equal(t, "running", response.Status)
			},
		},
		{
			name:        "non-existent container",
			containerID: "nonexistent",
			setupMocks: func(mockMgr *testutil.MockContainerManager) {
				mockMgr.GetByNameFn = func(ctx context.Context, name string) (*container.Container, error) {
					return nil, errors.ContainerNotFound(name)
				}
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Container not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := testutil.NewMockContainerManager()
			tt.setupMocks(mockMgr)
			
			server := &Server{
				config:       DefaultConfig(),
				echo:         echo.New(),
				containerMgr: mockMgr,
				startTime:    time.Now(),
			}
			
			e := server.Echo()
			req := httptest.NewRequest(http.MethodGet, "/api/containers/"+tt.containerID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/containers/:id")
			c.SetParamNames("id")
			c.SetParamValues(tt.containerID)
			
			err := server.handleGetContainer(c)
			
			if tt.expectedStatus == http.StatusOK {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				// Manually trigger error handler since we're calling handler directly
				e.HTTPErrorHandler(err, c)
			}
			
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.checkResponse(t, rec.Body.String())
		})
	}
}

func TestHandleDeleteContainer(t *testing.T) {
	tests := []struct {
		name           string
		containerID    string
		setupMocks     func(*testutil.MockContainerManager)
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:        "successful deletion",
			containerID: "container-123",
			setupMocks: func(mockMgr *testutil.MockContainerManager) {
				mockMgr.StopFn = func(ctx context.Context, containerID string) error {
					// Stop can fail, that's OK
					return nil
				}
				mockMgr.RemoveFn = func(ctx context.Context, containerID string) error {
					if containerID == "container-123" {
						return nil
					}
					return fmt.Errorf("container not found")
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Container deleted successfully")
			},
		},
		{
			name:        "container not found",
			containerID: "nonexistent",
			setupMocks: func(mockMgr *testutil.MockContainerManager) {
				mockMgr.StopFn = func(ctx context.Context, containerID string) error {
					// Stop can fail, that's OK
					return fmt.Errorf("container not found")
				}
				mockMgr.RemoveFn = func(ctx context.Context, containerID string) error {
					return errors.ContainerNotFound(containerID)
				}
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Container not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := testutil.NewMockContainerManager()
			tt.setupMocks(mockMgr)
			
			server := &Server{
				config:       DefaultConfig(),
				echo:         echo.New(),
				containerMgr: mockMgr,
				startTime:    time.Now(),
			}
			
			e := server.Echo()
			req := httptest.NewRequest(http.MethodDelete, "/api/containers/"+tt.containerID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/containers/:id")
			c.SetParamNames("id")
			c.SetParamValues(tt.containerID)
			
			err := server.handleDeleteContainer(c)
			
			if tt.expectedStatus == http.StatusOK {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				// Manually trigger error handler since we're calling handler directly
				e.HTTPErrorHandler(err, c)
			}
			
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.checkResponse(t, rec.Body.String())
		})
	}
}

func TestHandleContainerAction(t *testing.T) {
	tests := []struct {
		name           string
		containerID    string
		action         string
		setupMocks     func(*testutil.MockContainerManager)
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:        "start container",
			containerID: "container-123",
			action:      "start",
			setupMocks: func(mockMgr *testutil.MockContainerManager) {
				mockMgr.StartFn = func(ctx context.Context, containerID string) error {
					if containerID == "container-123" {
						return nil
					}
					return fmt.Errorf("container not found")
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Container start started successfully")
			},
		},
		{
			name:        "stop container",
			containerID: "container-123",
			action:      "stop",
			setupMocks: func(mockMgr *testutil.MockContainerManager) {
				mockMgr.StopFn = func(ctx context.Context, containerID string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Container stop stoped successfully")
			},
		},
		{
			name:        "restart container",
			containerID: "container-123",
			action:      "restart",
			setupMocks: func(mockMgr *testutil.MockContainerManager) {
				mockMgr.StopFn = func(ctx context.Context, containerID string) error {
					// Stop may fail, that's OK for restart
					return nil
				}
				mockMgr.StartFn = func(ctx context.Context, containerID string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Container restart restarted successfully")
			},
		},
		{
			name:        "invalid action",
			containerID: "container-123",
			action:      "invalid",
			setupMocks:  func(*testutil.MockContainerManager) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Invalid action")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := testutil.NewMockContainerManager()
			tt.setupMocks(mockMgr)
			
			server := &Server{
				config:       DefaultConfig(),
				echo:         echo.New(),
				containerMgr: mockMgr,
				startTime:    time.Now(),
			}
			
			// Create request body
			actionReq := ContainerActionRequest{Action: tt.action}
			reqBody, err := json.Marshal(actionReq)
			require.NoError(t, err)
			
			e := server.Echo()
			req := httptest.NewRequest(http.MethodPost, "/api/containers/"+tt.containerID+"/action", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/containers/:id/action")
			c.SetParamNames("id")
			c.SetParamValues(tt.containerID)
			
			err = server.handleContainerAction(c)
			
			if tt.expectedStatus == http.StatusOK {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				// Manually trigger error handler since we're calling handler directly
				e.HTTPErrorHandler(err, c)
			}
			
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.checkResponse(t, rec.Body.String())
		})
	}
}

func TestHandleGetContainerLogs(t *testing.T) {
	tests := []struct {
		name           string
		containerID    string
		setupMocks     func(*testutil.MockContainerManager)
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:        "successful log retrieval",
			containerID: "container-123",
			setupMocks: func(mockMgr *testutil.MockContainerManager) {
				mockMgr.LogsFn = func(ctx context.Context, containerID string, follow bool) ([]byte, error) {
					if containerID == "container-123" {
						return []byte("Log line 1\nLog line 2\nLog line 3\n"), nil
					}
					return nil, fmt.Errorf("container not found")
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var response ContainerLogsResponse
				require.NoError(t, json.Unmarshal([]byte(body), &response))
				
				assert.Len(t, response.Logs, 3)
				assert.Equal(t, "Log line 1", response.Logs[0])
				assert.Equal(t, "Log line 2", response.Logs[1])
				assert.Equal(t, "Log line 3", response.Logs[2])
			},
		},
		{
			name:        "container not found",
			containerID: "nonexistent",
			setupMocks: func(mockMgr *testutil.MockContainerManager) {
				mockMgr.LogsFn = func(ctx context.Context, containerID string, follow bool) ([]byte, error) {
					return nil, errors.ContainerNotFound(containerID)
				}
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Container not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMgr := testutil.NewMockContainerManager()
			tt.setupMocks(mockMgr)
			
			server := &Server{
				config:       DefaultConfig(),
				echo:         echo.New(),
				containerMgr: mockMgr,
				startTime:    time.Now(),
			}
			
			e := server.Echo()
			req := httptest.NewRequest(http.MethodGet, "/api/containers/"+tt.containerID+"/logs", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/containers/:id/logs")
			c.SetParamNames("id")
			c.SetParamValues(tt.containerID)
			
			err := server.handleGetContainerLogs(c)
			
			if tt.expectedStatus == http.StatusOK {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				// Manually trigger error handler since we're calling handler directly
				e.HTTPErrorHandler(err, c)
			}
			
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.checkResponse(t, rec.Body.String())
		})
	}
}