package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"vibeman/internal/config"
	"vibeman/internal/xdg"

	"github.com/spf13/cobra"
)

// ServerCommands creates server management commands
func ServerCommands(cfg *config.Manager) []*cobra.Command {
	commands := []*cobra.Command{}

	// vibeman server start
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the Vibeman server",
		Long: `Start the Vibeman HTTP API server. The server provides a web interface
and API endpoints for managing repositories, worktrees, and services.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetInt("port")
			configPath, _ := cmd.Flags().GetString("config")
			daemon, _ := cmd.Flags().GetBool("daemon")

			return startServer(cmd.Context(), port, configPath, daemon)
		},
	}
	
	// Load global config to get default port
	globalConfig, err := config.LoadGlobalConfig()
	defaultPort := 8080
	if err == nil {
		defaultPort = globalConfig.Server.Port
	}
	
	startCmd.Flags().IntP("port", "p", defaultPort, "Port to run the server on")
	startCmd.Flags().StringP("config", "c", "", "Path to configuration file")
	startCmd.Flags().BoolP("daemon", "d", false, "Run server in daemon mode (background)")
	commands = append(commands, startCmd)

	// vibeman server stop
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the Vibeman server",
		Long:  `Stop a running Vibeman server by sending a graceful shutdown signal.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return stopServer(cmd.Context())
		},
	}
	commands = append(commands, stopCmd)

	// vibeman server status
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check server status",
		Long:  `Check if the Vibeman server is running and show basic status information.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return serverStatus(cmd.Context())
		},
	}
	commands = append(commands, statusCmd)

	return commands
}

// startServer starts the Vibeman server
func startServer(ctx context.Context, port int, configPath string, daemon bool) error {
	if daemon {
		return startServerDaemon(port, configPath)
	}

	// Start server in foreground by re-executing with internal server command
	args := []string{"server"}
	if port != 8080 {
		args = append(args, "--port", strconv.Itoa(port))
	}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}

	// Execute the internal server command (handled by app.go)
	cmd := exec.CommandContext(ctx, os.Args[0], args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	return cmd.Run()
}

// startServerDaemon starts the server in background daemon mode
func startServerDaemon(port int, configPath string) error {
	args := []string{"server"}
	if port != 8080 {
		args = append(args, "--port", strconv.Itoa(port))
	}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}

	cmd := exec.Command(os.Args[0], args...)
	
	// Create log files in XDG-compliant location
	logsDir := xdg.LogsDir()
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	logFile, err := os.OpenFile(fmt.Sprintf("%s/server.log", logsDir), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the process detached
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server daemon: %w", err)
	}

	// Save PID for later shutdown
	pidFile := fmt.Sprintf("%s/server.pid", logsDir)
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		// Kill the process since we can't track it
		cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	fmt.Printf("Vibeman server started in daemon mode on port %d (PID: %d)\n", port, cmd.Process.Pid)
	fmt.Printf("Logs: %s/server.log\n", logsDir)
	fmt.Printf("Use 'vibeman server stop' to stop the server\n")

	return nil
}

// stopServer stops a running Vibeman server
func stopServer(ctx context.Context) error {
	// Look for PID file
	logsDir := xdg.LogsDir()
	pidFile := fmt.Sprintf("%s/server.pid", logsDir)
	
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No server PID file found. Server may not be running.")
			return nil
		}
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pidStr := strings.TrimSpace(string(pidBytes))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid PID in file: %w", err)
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM for graceful shutdown
	fmt.Printf("Sending shutdown signal to server (PID: %d)...\n", pid)
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send shutdown signal: %w", err)
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("Process exited with error: %v\n", err)
		} else {
			fmt.Println("Server stopped successfully")
		}
	case <-time.After(10 * time.Second):
		fmt.Println("Server didn't stop gracefully, sending SIGKILL...")
		process.Kill()
		fmt.Println("Server force-stopped")
	}

	// Clean up PID file
	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: failed to remove PID file: %v\n", err)
	}

	return nil
}

// serverStatus checks if the server is running
func serverStatus(ctx context.Context) error {
	// Check PID file
	logsDir := xdg.LogsDir()
	pidFile := fmt.Sprintf("%s/server.pid", logsDir)
	
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Server: Not running (no PID file)")
			return nil
		}
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pidStr := strings.TrimSpace(string(pidBytes))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		fmt.Printf("Server: Unknown (invalid PID file: %s)\n", pidStr)
		return nil
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("Server: Not running (PID %d not found)\n", pid)
		return nil
	}

	// Send signal 0 to check if process is alive
	if err := process.Signal(syscall.Signal(0)); err != nil {
		fmt.Printf("Server: Not running (PID %d is dead)\n", pid)
		// Clean up stale PID file
		os.Remove(pidFile)
		return nil
	}

	fmt.Printf("Server: Running (PID: %d)\n", pid)
	fmt.Printf("Logs: %s/server.log\n", logsDir)
	
	// TODO: Could also try to make an HTTP request to check if it's responding
	
	return nil
}