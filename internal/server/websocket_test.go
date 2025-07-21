package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vibeman/internal/container"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockContainerForWebSocket for WebSocket tests
type MockContainerForWebSocket struct {
	mock.Mock
}

func (m *MockContainerForWebSocket) List(ctx context.Context) ([]*container.Container, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*container.Container), args.Error(1)
}

func (m *MockContainerForWebSocket) Get(ctx context.Context, id string) (*container.Container, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*container.Container), args.Error(1)
}

func (m *MockContainerForWebSocket) GetByName(ctx context.Context, name string) (*container.Container, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*container.Container), args.Error(1)
}

func (m *MockContainerForWebSocket) Create(ctx context.Context, repositoryName, environment, image string) (*container.Container, error) {
	args := m.Called(ctx, repositoryName, environment, image)
	return args.Get(0).(*container.Container), args.Error(1)
}

func (m *MockContainerForWebSocket) CreateWithConfig(ctx context.Context, config *container.CreateConfig) (*container.Container, error) {
	args := m.Called(ctx, config)
	return args.Get(0).(*container.Container), args.Error(1)
}

func (m *MockContainerForWebSocket) Start(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockContainerForWebSocket) Stop(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockContainerForWebSocket) Remove(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockContainerForWebSocket) Logs(ctx context.Context, id string, follow bool) ([]byte, error) {
	args := m.Called(ctx, id, follow)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockContainerForWebSocket) Exec(ctx context.Context, id, command string) (string, error) {
	args := m.Called(ctx, id, command)
	return args.Get(0).(string), args.Error(1)
}

func TestHandleAIWebSocket_MissingWorktree(t *testing.T) {
	// Create Echo instance and server
	e := echo.New()
	server := &Server{
		echo: e,
	}

	// Create request without worktree parameter
	req := httptest.NewRequest(http.MethodGet, "/api/ai/attach/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("worktree")
	c.SetParamValues("")

	// Test the handler
	err := server.handleAIWebSocket(c)

	// Should return no error (handler returns JSON response, not error)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	
	// Check response body
	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Worktree name is required", response.Error)
}

func TestWebSocketMessageTypes(t *testing.T) {
	tests := []struct {
		name     string
		message  ClientMessage
		expected string
	}{
		{
			name: "stdin message",
			message: ClientMessage{
				Type: "stdin",
				Data: "ls -la\n",
			},
			expected: "stdin",
		},
		{
			name: "resize message",
			message: ClientMessage{
				Type: "resize",
				Cols: 80,
				Rows: 24,
			},
			expected: "resize",
		},
		{
			name: "ping message",
			message: ClientMessage{
				Type: "ping",
			},
			expected: "ping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.message.Type)
		})
	}
}

func TestServerMessageTypes(t *testing.T) {
	tests := []struct {
		name     string
		message  ServerMessage
		expected string
	}{
		{
			name: "stdout message",
			message: ServerMessage{
				Type: "stdout",
				Data: "file1.txt\nfile2.txt\n",
			},
			expected: "stdout",
		},
		{
			name: "stderr message",
			message: ServerMessage{
				Type: "stderr",
				Data: "command not found\n",
			},
			expected: "stderr",
		},
		{
			name: "exit message",
			message: ServerMessage{
				Type:     "exit",
				ExitCode: intPtr(0),
			},
			expected: "exit",
		},
		{
			name: "pong message",
			message: ServerMessage{
				Type: "pong",
			},
			expected: "pong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.message.Type)
			if tt.message.ExitCode != nil {
				assert.Equal(t, 0, *tt.message.ExitCode)
			}
		})
	}
}

func TestTerminalSessionContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	session := &TerminalSession{
		ctx:       ctx,
		cancel:    cancel,
		container: "test-container",
		worktree:  "test-worktree",
	}

	// Test that context cancellation works
	select {
	case <-session.ctx.Done():
		assert.True(t, true, "Context should be cancelled")
	case <-time.After(200 * time.Millisecond):
		t.Error("Context should have been cancelled")
	}
}

// Helper function for test
func intPtr(i int) *int {
	return &i
}