package cli

import (
	"context"
	"github.com/spf13/cobra"
	"vibeman/internal/cli/commands"
	"vibeman/internal/config"
	"vibeman/internal/db"
	"vibeman/internal/interfaces"
	"vibeman/internal/service"
)

// Use interfaces from central interfaces package
type ContainerManager = interfaces.ContainerManager
type GitManager = interfaces.GitManager

// ServiceManager is an alias for the commands.ServiceManager interface
type ServiceManager = commands.ServiceManager

// Manager handles CLI operations
type Manager struct {
	config    *config.Manager
	container interfaces.ContainerManager
	git       interfaces.GitManager
	service   ServiceManager
	repoMgr   db.RepositoryManager
	database  *db.DB
	rootCmd   *cobra.Command
}

// New creates a new CLI manager
func New(cfg *config.Manager) *Manager {
	m := &Manager{
		config: cfg,
	}

	// Use the root command from root.go
	m.rootCmd = createRootCommand()

	return m
}

// SetManagers sets the container, git, service, and repository managers
func (m *Manager) SetManagers(container interfaces.ContainerManager, git interfaces.GitManager, svc interface{}, repoMgr db.RepositoryManager, database *db.DB) {
	m.container = container
	m.git = git
	m.repoMgr = repoMgr
	m.database = database

	// Check if we need to wrap the service manager
	if sm, ok := svc.(*service.Manager); ok {
		// Wrap the concrete service manager to match our interface
		m.service = NewServiceManagerWrapper(sm)
	} else if sm, ok := svc.(ServiceManager); ok {
		// It's already compatible
		m.service = sm
	}

	m.setupCommands()
}

// SetupDefaultCommands sets up commands with nil managers (for testing)
func (m *Manager) SetupDefaultCommands() {
	m.setupCommands()
}

// Execute executes the CLI with the given arguments
func (m *Manager) Execute(args []string) error {
	return m.ExecuteWithContext(context.Background(), args)
}

// ExecuteWithContext executes the CLI with the given arguments and context
func (m *Manager) ExecuteWithContext(ctx context.Context, args []string) error {
	m.rootCmd.SetArgs(args)
	return m.rootCmd.ExecuteContext(ctx)
}

// setupCommands sets up all CLI commands
func (m *Manager) setupCommands() {
	// Use the interfaces directly - they're already compatible
	// Add init command (top-level)
	for _, cmd := range commands.InitCommands(m.config, m.container, m.git, m.service) {
		m.rootCmd.AddCommand(cmd)
	}

	// Add repository management commands (both grouped and top-level)
	projectCmd := &cobra.Command{
		Use:     "project",
		Short:   "Repository management commands",
		Aliases: []string{"proj", "p"},
	}
	for _, cmd := range commands.RepositoryCommands(m.config, m.container, m.git, m.service, m.repoMgr, m.database) {
		projectCmd.AddCommand(cmd)
		// Add top-level aliases for common commands
		if cmd.Use == "list" || cmd.Use == "status [project-name]" {
			m.rootCmd.AddCommand(cmd)
		}
		// Add new style commands to root level
		if cmd.Use == "start [worktree]" || cmd.Use == "stop [worktree]" {
			m.rootCmd.AddCommand(cmd)
		}
	}
	m.rootCmd.AddCommand(projectCmd)

	// Add repository management commands
	repoCmd := &cobra.Command{
		Use:     "repo",
		Short:   "Repository management commands",
		Aliases: []string{"repository"},
	}
	if m.repoMgr != nil && m.database != nil {
		for _, cmd := range commands.RepoCommands(m.config, m.repoMgr, m.git, m.container, m.service, m.database) {
			repoCmd.AddCommand(cmd)
		}
	}
	m.rootCmd.AddCommand(repoCmd)

	// Add service management commands
	serviceCmd := &cobra.Command{
		Use:     "service",
		Short:   "Service management commands",
		Aliases: []string{"svc"},
	}
	for _, cmd := range commands.ServiceCommands(m.config, m.container, m.service) {
		serviceCmd.AddCommand(cmd)
	}
	m.rootCmd.AddCommand(serviceCmd)

	// Add plural services management commands
	servicesCmd := &cobra.Command{
		Use:     "services",
		Short:   "Manage all global services",
		Aliases: []string{"svcs"},
	}
	for _, cmd := range commands.ServicesCommands(m.config, m.container, m.service) {
		servicesCmd.AddCommand(cmd)
	}
	m.rootCmd.AddCommand(servicesCmd)

	// Add worktree management commands
	worktreeCmd := &cobra.Command{
		Use:     "worktree",
		Short:   "Worktree development environment commands",
		Aliases: []string{"worktree", "feat"}, // Backward compatibility
	}
	for _, cmd := range commands.WorktreeCommands(m.config, m.container, m.git, m.service, m.repoMgr, m.database) {
		worktreeCmd.AddCommand(cmd)
	}
	m.rootCmd.AddCommand(worktreeCmd)

	// Add configuration commands
	configCmd := &cobra.Command{
		Use:     "config",
		Short:   "Configuration management commands",
		Aliases: []string{"cfg"},
	}
	for _, cmd := range commands.ConfigCommands(m.config) {
		configCmd.AddCommand(cmd)
	}
	m.rootCmd.AddCommand(configCmd)

	// Add service installation commands (only in local mode)
	for _, cmd := range commands.ServiceInstallCommands() {
		m.rootCmd.AddCommand(cmd)
	}

	// Add server management commands
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Server management commands",
		Long:  `Manage the Vibeman HTTP API server. Use these commands to start, stop, and check the status of the server.`,
	}
	for _, cmd := range commands.ServerCommands(m.config) {
		serverCmd.AddCommand(cmd)
	}
	m.rootCmd.AddCommand(serverCmd)
	
	// Add AI container commands
	aiCmd := commands.CreateAICommand(m.config, m.container, m.git, m.service, m.database)
	m.rootCmd.AddCommand(aiCmd)
}

