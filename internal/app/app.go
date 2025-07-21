package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"vibeman/internal/cli"
	"vibeman/internal/client"
	"vibeman/internal/config"
	"vibeman/internal/container"
	"vibeman/internal/db"
	"vibeman/internal/git"
	"vibeman/internal/logger"
	"vibeman/internal/server"
	"vibeman/internal/service"
)

// App represents the main application
type App struct {
	// Server components (only used in server mode)
	Config    *config.Manager
	Container *container.Manager
	Git       *git.Manager
	Service   *service.Manager
	Server    *server.Server
	DB        *db.DB

	// Client components (only used in client mode)
	Client *client.Client
	CLI    *cli.Manager
}

// New creates a new application instance
func New() *App {
	return &App{}
}

// Run starts the application in the appropriate mode
func (a *App) Run(args []string) error {
	return a.RunWithContext(context.Background(), args)
}

// RunWithContext starts the application with a context for cancellation
func (a *App) RunWithContext(ctx context.Context, args []string) error {
	// Determine mode based on arguments
	if len(args) > 0 && args[0] == "server" {
		// Check if this is the new "server start" command or old "server" command
		if len(args) > 1 && (args[1] == "start" || args[1] == "stop" || args[1] == "status") {
			// New server subcommands - handle via CLI
			return a.runLocal(ctx, args)
		}
		// Old "server" command for backward compatibility - initialize server components and start
		return a.runServer(ctx, args[1:])
	}

	// Check if client mode is requested via environment variable or --server flag
	serverEnv := os.Getenv("VIBEMAN_SERVER")
	hasServerFlag := false
	for _, arg := range args {
		if arg == "--server" || strings.HasPrefix(arg, "--server=") {
			hasServerFlag = true
			break
		}
	}

	if serverEnv != "" || hasServerFlag {
		// Client mode - connect to remote server
		return a.runClient(ctx, args)
	}

	// Local mode - use local managers
	return a.runLocal(ctx, args)
}

// runLocal runs the application in local mode (without server)
func (a *App) runLocal(ctx context.Context, args []string) error {
	// Initialize local components
	cfg := config.New()
	if err := cfg.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	a.Config = cfg
	a.Container = container.New(cfg)
	a.Git = git.New(cfg)
	a.Service = service.New(cfg)

	// Initialize database with XDG-compliant path (needed for repo commands)
	dbConfig := db.DefaultConfig()
	database, err := db.New(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	a.DB = database

	// Run migrations
	if err := database.Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize CLI with local managers
	a.CLI = cli.New(cfg)

	// Create repository manager from database
	repoMgr := db.NewRepositoryRepository(database)

	// Set the local managers
	a.CLI.SetManagers(a.Container, a.Git, a.Service, repoMgr, database)

	// Show help if no arguments provided
	if len(args) == 0 {
		return a.CLI.ExecuteWithContext(ctx, []string{"--help"})
	}

	// CLI mode
	return a.CLI.ExecuteWithContext(ctx, args)
}

// runClient runs the application in client mode
func (a *App) runClient(ctx context.Context, args []string) error {
	// Get server URL from environment variable or flag
	serverURL := os.Getenv("VIBEMAN_SERVER")
	if serverURL == "" {
		// Extract from --server flag if provided
		for i, arg := range args {
			if arg == "--server" && i+1 < len(args) {
				serverURL = args[i+1]
				break
			} else if strings.HasPrefix(arg, "--server=") {
				serverURL = strings.TrimPrefix(arg, "--server=")
				break
			}
		}
	}

	if serverURL == "" {
		return fmt.Errorf("server URL not specified for client mode")
	}

	// Create API client
	apiClient, err := client.New(serverURL)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}
	a.Client = apiClient

	// Create manager adapters that use the API client
	containerMgr := client.NewContainerManager(apiClient)
	gitMgr := client.NewGitManager(apiClient)
	serviceMgr := client.NewServiceManager(apiClient)

	// Initialize CLI with API client adapters
	cfg := config.New() // Client-side config for preferences only
	a.CLI = cli.New(cfg)

	// Set the API client-based managers (no repo manager or database in client mode yet)
	a.CLI.SetManagers(containerMgr, gitMgr, serviceMgr, nil, nil)

	// Show help if no arguments provided
	if len(args) == 0 {
		return a.CLI.ExecuteWithContext(ctx, []string{"--help"})
	}

	// CLI mode
	return a.CLI.ExecuteWithContext(ctx, args)
}

// runServer runs the application in server mode
func (a *App) runServer(ctx context.Context, args []string) error {
	// Initialize server components
	cfg := config.New()
	if err := cfg.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	a.Config = cfg
	a.Container = container.New(cfg)
	a.Git = git.New(cfg)
	a.Service = service.New(cfg)

	// Initialize database with XDG-compliant path
	dbConfig := db.DefaultConfig()
	database, err := db.New(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	a.DB = database

	// Run migrations
	if err := database.Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Load global config
	globalConfig, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// Parse server-specific arguments
	var port int = globalConfig.Server.Port // Use config as default
	var configPath string

	for i, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--port="):
			fmt.Sscanf(arg, "--port=%d", &port)
		case strings.HasPrefix(arg, "--config="):
			configPath = strings.TrimPrefix(arg, "--config=")
		case arg == "--port" && i+1 < len(args):
			fmt.Sscanf(args[i+1], "%d", &port)
		case arg == "--config" && i+1 < len(args):
			configPath = args[i+1]
		}
	}

	// Initialize server configuration
	serverConfig := &server.Config{
		Port:       port,
		ConfigPath: configPath,
	}

	// Create server with configuration manager
	a.Server = server.NewWithConfigManager(serverConfig, cfg)
	// Set the additional dependencies
	a.Server.SetDependencies(a.Container, a.Git, a.Service, a.DB)

	// Start the server
	logger.WithFields(logger.Fields{
		"port":      port,
		"operation": "server_start",
	}).Info("Starting Vibeman server")
	return a.Server.Start(ctx)
}
