package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"vibeman/internal/db"

	"github.com/labstack/echo/v4"
)

// handleGetWorktreeLogs godoc
// @Summary Get worktree logs
// @Description Get logs from a specific worktree
// @Tags worktrees
// @Accept json
// @Produce json
// @Param id path string true "Worktree ID"
// @Param lines query int false "Number of lines to retrieve" default(50)
// @Param follow query bool false "Follow log output"
// @Success 200 {object} LogsResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/worktrees/{id}/logs [get]
func (s *Server) handleGetWorktreeLogs(c echo.Context) error {
	dbInstance, err := s.getDB()
	if err != nil {
		return c.JSON(503, ErrorResponse{
			Error: "Database not available",
		})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(400, ErrorResponse{
			Error: "Worktree ID is required",
		})
	}

	// Get worktree details
	worktreeRepo := db.NewWorktreeRepository(dbInstance)
	worktree, err := worktreeRepo.Get(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err, "Failed to get worktree")
	}

	// Get repository info for log path construction
	repoRepo := db.NewRepositoryRepository(dbInstance)
	repo, err := repoRepo.GetByID(c.Request().Context(), worktree.RepositoryID)
	if err != nil {
		return handleError(c, err, "Failed to get repository")
	}

	// Try to read logs from the worktree logs directory
	logsDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "vibeman", "logs", repo.Name, worktree.Name)
	logFile := filepath.Join(logsDir, "worktree.log")

	// Read log file if it exists
	var logLines []string
	if logData, err := os.ReadFile(logFile); err == nil {
		// Split into lines and filter empty ones
		allLines := strings.Split(string(logData), "\n")
		for _, line := range allLines {
			if strings.TrimSpace(line) != "" {
				logLines = append(logLines, line)
			}
		}
	} else {
		// If no log file exists, provide a helpful message
		logLines = []string{"No logs available for this worktree"}
	}

	// Apply lines limit if specified
	linesParam := c.QueryParam("lines")
	if linesParam != "" {
		if lines, parseErr := strconv.Atoi(linesParam); parseErr == nil && lines > 0 {
			if lines < len(logLines) {
				// Get the last N lines
				logLines = logLines[len(logLines)-lines:]
			}
		}
	}

	response := LogsResponse{
		Logs:      logLines,
		Source:    "worktree",
		ID:        id,
		Timestamp: time.Now().Format(time.RFC3339),
		Lines:     len(logLines),
	}

	return c.JSON(200, response)
}

// handleGetServiceLogs godoc
// @Summary Get service logs
// @Description Get logs from a specific service
// @Tags services
// @Accept json
// @Produce json
// @Param id path string true "Service ID"
// @Param lines query int false "Number of lines to retrieve" default(50)
// @Param follow query bool false "Follow log output"
// @Success 200 {object} LogsResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/services/{id}/logs [get]
func (s *Server) handleGetServiceLogs(c echo.Context) error {
	if s.serviceMgr == nil {
		return c.JSON(503, ErrorResponse{
			Error: "Service manager not available",
		})
	}

	id := c.Param("id")
	if id == "" {
		return c.JSON(400, ErrorResponse{
			Error: "Service ID is required",
		})
	}

	// Verify service exists
	_, err := s.serviceMgr.GetService(id)
	if err != nil {
		return handleError(c, err, "Failed to get service")
	}

	// Try to read logs from service log files
	// Services typically log to ~/.local/share/vibeman/logs/services/{service-name}.log
	logsDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "vibeman", "logs", "services")
	logFile := filepath.Join(logsDir, fmt.Sprintf("%s.log", id))

	// Read log file if it exists
	var logLines []string
	if logData, err := os.ReadFile(logFile); err == nil {
		// Split into lines and filter empty ones
		allLines := strings.Split(string(logData), "\n")
		for _, line := range allLines {
			if strings.TrimSpace(line) != "" {
				logLines = append(logLines, line)
			}
		}
	} else {
		// If no log file exists, try to get logs via container manager if service is containerized
		if s.containerMgr != nil {
			// Try to find a container with the same name as the service
			if container, getErr := s.containerMgr.GetByName(c.Request().Context(), id); getErr == nil {
				// Get logs from the container
				if logsBytes, logsErr := s.containerMgr.Logs(c.Request().Context(), container.ID, false); logsErr == nil {
					logsStr := string(logsBytes)
					allLines := strings.Split(strings.TrimSpace(logsStr), "\n")
					for _, line := range allLines {
						if strings.TrimSpace(line) != "" {
							logLines = append(logLines, line)
						}
					}
				}
			}
		}

		// If still no logs, provide a helpful message
		if len(logLines) == 0 {
			logLines = []string{"No logs available for this service"}
		}
	}

	// Apply lines limit if specified
	linesParam := c.QueryParam("lines")
	if linesParam != "" {
		if lines, parseErr := strconv.Atoi(linesParam); parseErr == nil && lines > 0 {
			if lines < len(logLines) {
				// Get the last N lines
				logLines = logLines[len(logLines)-lines:]
			}
		}
	}

	response := LogsResponse{
		Logs:      logLines,
		Source:    "service",
		ID:        id,
		Timestamp: time.Now().Format(time.RFC3339),
		Lines:     len(logLines),
	}

	return c.JSON(200, response)
}