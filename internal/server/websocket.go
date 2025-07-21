package server

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"vibeman/internal/logger"
	"vibeman/internal/validation"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		
		// Allow connections without origin header (e.g., CLI tools)
		if origin == "" {
			return true
		}
		
		// Parse origin to check if it's localhost
		// Allow common localhost origins
		allowedOrigins := []string{
			"http://localhost",
			"https://localhost",
			"http://127.0.0.1",
			"https://127.0.0.1",
			"http://[::1]",
			"https://[::1]",
		}
		
		// Check if origin starts with any allowed origin
		for _, allowed := range allowedOrigins {
			if strings.HasPrefix(origin, allowed) {
				return true
			}
		}
		
		logger.WithFields(logger.Fields{
			"origin": origin,
			"remote": r.RemoteAddr,
		}).Warn("WebSocket connection rejected - invalid origin")
		
		return false
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// ClientMessage represents messages from client to server
type ClientMessage struct {
	Type string `json:"type"` // 'stdin', 'resize', 'ping'
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

// ServerMessage represents messages from server to client
type ServerMessage struct {
	Type     string `json:"type"` // 'stdout', 'stderr', 'exit', 'pong'
	Data     string `json:"data,omitempty"`
	ExitCode *int   `json:"exitCode,omitempty"`
}

// TerminalSession manages a WebSocket terminal session
type TerminalSession struct {
	ws        *websocket.Conn
	ctx       context.Context
	cancel    context.CancelFunc
	container string
	worktree  string
	cmd       *exec.Cmd
}

// handleAIWebSocket handles WebSocket connections for AI container terminal access
// @Summary WebSocket endpoint for AI container terminal
// @Description Establish WebSocket connection for terminal access to AI containers
// @Tags ai,websocket
// @Param worktree path string true "Worktree name"
// @Success 101 {string} string "Switching Protocols"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/ai/attach/{worktree} [get]
func (s *Server) handleAIWebSocket(c echo.Context) error {
	worktreeName := c.Param("worktree")
	if worktreeName == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Worktree name is required",
		})
	}

	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return err
	}
	defer ws.Close()

	// Find AI container for the worktree
	containerMgr, err := s.getContainerManagerInterface()
	if err != nil {
		ws.WriteJSON(ServerMessage{
			Type: "stderr",
			Data: "Container manager not available\r\n",
		})
		return err
	}

	containers, err := containerMgr.List(c.Request().Context())
	if err != nil {
		ws.WriteJSON(ServerMessage{
			Type: "stderr",
			Data: "Failed to list containers\r\n",
		})
		return err
	}

	var aiContainerName string
	for _, container := range containers {
		if container.Type == "ai" && strings.Contains(container.Name, worktreeName) {
			// Check if container is running
			status := strings.ToLower(container.Status)
			if strings.Contains(status, "running") || strings.Contains(status, "up") {
				aiContainerName = container.Name
				break
			}
		}
	}

	if aiContainerName == "" {
		ws.WriteJSON(ServerMessage{
			Type: "stderr",
			Data: fmt.Sprintf("No running AI container found for worktree: %s\r\n", worktreeName),
		})
		return fmt.Errorf("no AI container found for worktree: %s", worktreeName)
	}

	logger.WithFields(logger.Fields{
		"worktree":  worktreeName,
		"container": aiContainerName,
	}).Info("Starting WebSocket terminal session")

	// Validate container name to prevent command injection
	if err := validation.ContainerID(aiContainerName); err != nil {
		ws.WriteJSON(ServerMessage{
			Type: "stderr",
			Data: "Invalid container name\r\n",
		})
		return err
	}

	// Create terminal session
	ctx, cancel := context.WithCancel(c.Request().Context())
	session := &TerminalSession{
		ws:        ws,
		ctx:       ctx,
		cancel:    cancel,
		container: aiContainerName,
		worktree:  worktreeName,
	}

	// Handle WebSocket terminal session
	return session.handleSession()
}

// handleSession manages the WebSocket terminal session
func (ts *TerminalSession) handleSession() error {
	defer ts.cancel()

	// Start docker exec command for shell access
	ts.cmd = exec.CommandContext(ts.ctx, "docker", "exec", "-it", ts.container, "/bin/zsh")

	// Create pseudo-terminal
	stdin, err := ts.cmd.StdinPipe()
	if err != nil {
		ts.sendError("Failed to create stdin pipe")
		return err
	}
	defer stdin.Close()

	stdout, err := ts.cmd.StdoutPipe()
	if err != nil {
		ts.sendError("Failed to create stdout pipe")
		return err
	}
	defer stdout.Close()

	stderr, err := ts.cmd.StderrPipe()
	if err != nil {
		ts.sendError("Failed to create stderr pipe")
		return err
	}
	defer stderr.Close()

	// Start the command
	if err := ts.cmd.Start(); err != nil {
		ts.sendError(fmt.Sprintf("Failed to start shell: %v", err))
		return err
	}

	// Ensure command is killed if we exit early
	defer func() {
		if ts.cmd.Process != nil {
			ts.cmd.Process.Kill()
		}
	}()

	// Handle stdout/stderr output
	go ts.handleOutput(stdout, "stdout")
	go ts.handleOutput(stderr, "stderr")

	// Handle WebSocket messages
	go ts.handleWebSocketMessages(stdin)

	// Wait for command to finish
	go func() {
		if err := ts.cmd.Wait(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode := exitError.ExitCode()
				ts.ws.WriteJSON(ServerMessage{
					Type:     "exit",
					ExitCode: &exitCode,
				})
			} else {
				ts.sendError(fmt.Sprintf("Shell exited with error: %v", err))
			}
		} else {
			exitCode := 0
			ts.ws.WriteJSON(ServerMessage{
				Type:     "exit",
				ExitCode: &exitCode,
			})
		}
		ts.cancel()
	}()

	// Keep session alive until context is cancelled
	<-ts.ctx.Done()
	return nil
}

// handleOutput reads from stdout/stderr and sends to WebSocket
func (ts *TerminalSession) handleOutput(pipe interface{}, outputType string) {
	buffer := make([]byte, 4096)
	reader := pipe.(interface{ Read([]byte) (int, error) })

	for {
		select {
		case <-ts.ctx.Done():
			return
		default:
			n, err := reader.Read(buffer)
			if err != nil {
				return
			}
			if n > 0 {
				ts.ws.WriteJSON(ServerMessage{
					Type: outputType,
					Data: string(buffer[:n]),
				})
			}
		}
	}
}

// handleWebSocketMessages reads from WebSocket and handles client messages
func (ts *TerminalSession) handleWebSocketMessages(stdin interface{ Write([]byte) (int, error) }) {
	writer := stdin

	for {
		select {
		case <-ts.ctx.Done():
			return
		default:
			var msg ClientMessage
			if err := ts.ws.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.WithError(err).Error("WebSocket read error")
				}
				ts.cancel()
				return
			}

			switch msg.Type {
			case "stdin":
				if _, err := writer.Write([]byte(msg.Data)); err != nil {
					logger.WithError(err).Error("Failed to write to stdin")
					ts.cancel()
					return
				}
			case "resize":
				// Handle terminal resize (docker exec doesn't support this directly)
				// This would need a more sophisticated PTY implementation
				logger.WithFields(logger.Fields{
					"cols": msg.Cols,
					"rows": msg.Rows,
				}).Debug("Terminal resize requested")
			case "ping":
				ts.ws.WriteJSON(ServerMessage{Type: "pong"})
			default:
				logger.WithField("type", msg.Type).Warn("Unknown message type")
			}
		}
	}
}

// sendError sends an error message to the WebSocket client
func (ts *TerminalSession) sendError(message string) {
	ts.ws.WriteJSON(ServerMessage{
		Type: "stderr",
		Data: message + "\r\n",
	})
}