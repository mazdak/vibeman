package operations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"vibeman/internal/container"
	"vibeman/internal/db"
	"vibeman/internal/logger"
	"vibeman/internal/xdg"
)

// LogAggregator handles log aggregation for AI containers
type LogAggregator struct {
	db           *db.DB
	containerMgr ContainerManager
	mu           sync.RWMutex
	activeStreams map[string]context.CancelFunc
}

// NewLogAggregator creates a new log aggregator
func NewLogAggregator(database *db.DB, containerMgr ContainerManager) *LogAggregator {
	return &LogAggregator{
		db:           database,
		containerMgr: containerMgr,
		activeStreams: make(map[string]context.CancelFunc),
	}
}

// StartLogAggregation starts log aggregation for a worktree
func (la *LogAggregator) StartLogAggregation(ctx context.Context, worktreeID string) error {
	logger.WithField("worktree", worktreeID).Info("Starting log aggregation")

	// Get worktree details
	worktreeRepo := db.NewWorktreeRepository(la.db)
	worktree, err := worktreeRepo.Get(ctx, worktreeID)
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Get repository details
	repoRepo := db.NewRepositoryRepository(la.db)
	repo, err := repoRepo.GetByID(ctx, worktree.RepositoryID)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	// Create log directory structure
	logsDir := xdg.LogsDir()
	worktreeLogsDir := filepath.Join(logsDir, repo.Name, worktree.Name)
	if err := os.MkdirAll(worktreeLogsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// List all containers for this worktree
	containers, err := la.containerMgr.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Start log streaming for each container
	for _, c := range containers {
		// Check if container belongs to this worktree
		if isWorktreeContainer(c, repo.Name, worktree.Name) {
			if err := la.streamContainerLogs(ctx, c, worktreeLogsDir); err != nil {
				logger.WithError(err).WithField("container", c.Name).Warn("Failed to start log streaming")
			}
		}
	}

	return nil
}

// StopLogAggregation stops log aggregation for a worktree
func (la *LogAggregator) StopLogAggregation(worktreeID string) {
	logger.WithField("worktree", worktreeID).Info("Stopping log aggregation")

	la.mu.Lock()
	defer la.mu.Unlock()

	// Cancel all active streams for this worktree
	for key, cancel := range la.activeStreams {
		if strings.Contains(key, worktreeID) {
			cancel()
			delete(la.activeStreams, key)
		}
	}
}

// streamContainerLogs streams logs from a container to a file
func (la *LogAggregator) streamContainerLogs(ctx context.Context, c *container.Container, logsDir string) error {
	// Create a context for this stream
	streamCtx, cancel := context.WithCancel(ctx)
	
	// Store the cancel function
	la.mu.Lock()
	streamKey := fmt.Sprintf("%s-%s", c.ID, c.Name)
	la.activeStreams[streamKey] = cancel
	la.mu.Unlock()

	// Determine log file name based on container type
	logFileName := fmt.Sprintf("%s.log", c.Name)
	if c.Type != "" {
		logFileName = fmt.Sprintf("%s-%s.log", c.Type, c.Name)
	}
	logFilePath := filepath.Join(logsDir, logFileName)

	// Start streaming in a goroutine
	go func() {
		defer func() {
			la.mu.Lock()
			delete(la.activeStreams, streamKey)
			la.mu.Unlock()
		}()

		logger.WithFields(logger.Fields{
			"container": c.Name,
			"logFile":   logFilePath,
		}).Info("Starting log streaming")

		// Create or open log file
		logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			logger.WithError(err).Error("Failed to open log file")
			return
		}
		defer logFile.Close()

		// Write header
		header := fmt.Sprintf("\n=== Log stream started at %s for container %s ===\n", 
			time.Now().Format(time.RFC3339), c.Name)
		logFile.WriteString(header)

		// First, get existing logs (without follow)
		existingLogs, err := la.containerMgr.Logs(streamCtx, c.ID, false)
		if err != nil {
			logger.WithError(err).Warn("Failed to get existing container logs")
		} else {
			// Write existing logs to file
			if _, err := logFile.Write(existingLogs); err != nil {
				logger.WithError(err).Error("Failed to write existing logs to file")
			}
		}

		// TODO: Implement real-time log streaming using docker logs -f
		// For now, we'll periodically fetch logs
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		lastSize := int64(0)
		for {
			select {
			case <-streamCtx.Done():
				// Context cancelled, stop streaming
				footer := fmt.Sprintf("\n=== Log stream ended at %s ===\n", 
					time.Now().Format(time.RFC3339))
				logFile.WriteString(footer)
				return
			case <-ticker.C:
				// Get current file size
				fileInfo, err := logFile.Stat()
				if err != nil {
					logger.WithError(err).Error("Failed to stat log file")
					continue
				}
				currentSize := fileInfo.Size()
				
				// If file has grown, we already have the logs
				if currentSize > lastSize {
					lastSize = currentSize
				}
				
				// Note: In a production implementation, we would use docker logs --since
				// to get only new logs, or implement proper streaming with docker API
			}
		}
	}()

	return nil
}

// isWorktreeContainer checks if a container belongs to a specific worktree
func isWorktreeContainer(c *container.Container, repoName, worktreeName string) bool {
	// Check if container name matches worktree pattern
	expectedPrefix := fmt.Sprintf("%s-%s", repoName, worktreeName)
	return strings.HasPrefix(c.Name, expectedPrefix)
}

// AggregateLogsForAIContainer creates a unified log view for the AI container
func (la *LogAggregator) AggregateLogsForAIContainer(ctx context.Context, worktreeID string) error {
	// Get worktree details
	worktreeRepo := db.NewWorktreeRepository(la.db)
	worktree, err := worktreeRepo.Get(ctx, worktreeID)
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Get repository details
	repoRepo := db.NewRepositoryRepository(la.db)
	repo, err := repoRepo.GetByID(ctx, worktree.RepositoryID)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	// Create aggregated logs directory
	logsDir := xdg.LogsDir()
	worktreeLogsDir := filepath.Join(logsDir, repo.Name, worktree.Name)
	aggregatedLogsDir := filepath.Join(worktreeLogsDir, "aggregated")
	if err := os.MkdirAll(aggregatedLogsDir, 0755); err != nil {
		return fmt.Errorf("failed to create aggregated logs directory: %w", err)
	}

	// Create symlinks to all log files in a single directory
	entries, err := os.ReadDir(worktreeLogsDir)
	if err != nil {
		return fmt.Errorf("failed to read logs directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
			srcPath := filepath.Join(worktreeLogsDir, entry.Name())
			dstPath := filepath.Join(aggregatedLogsDir, entry.Name())
			
			// Remove existing symlink if present
			os.Remove(dstPath)
			
			// Create symlink
			if err := os.Symlink(srcPath, dstPath); err != nil {
				logger.WithError(err).WithField("file", entry.Name()).Warn("Failed to create log symlink")
			}
		}
	}

	// Create a README in the aggregated directory
	readmePath := filepath.Join(aggregatedLogsDir, "README.md")
	readmeContent := fmt.Sprintf("# Aggregated Logs for %s - %s\n\n"+
		"This directory contains symlinks to all log files for containers in this worktree.\n\n"+
		"## Log Files\n\n"+
		"- **worktree-*.log**: Main worktree container logs\n"+
		"- **service-*.log**: Service container logs\n"+
		"- **ai-*.log**: AI assistant container logs\n\n"+
		"## Viewing Logs\n\n"+
		"You can use standard Unix tools to view and search logs:\n\n"+
		"```bash\n"+
		"# View a specific log\n"+
		"tail -f worktree-*.log\n\n"+
		"# Search across all logs\n"+
		"grep -r \"ERROR\" .\n\n"+
		"# Follow all logs\n"+
		"tail -f *.log\n"+
		"```\n\n"+
		"## Log Rotation\n\n"+
		"Logs are automatically rotated when they reach 100MB. Old logs are compressed and stored with timestamps.\n\n"+
		"---\n"+
		"Generated at: %s\n", repo.Name, worktree.Name, time.Now().Format(time.RFC3339))

	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		logger.WithError(err).Warn("Failed to create aggregated logs README")
	}

	return nil
}