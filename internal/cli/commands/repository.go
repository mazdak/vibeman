package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"vibeman/internal/config"
	"vibeman/internal/db"
	"vibeman/internal/logger"
	"vibeman/internal/operations"
	"vibeman/internal/types"

	"github.com/spf13/cobra"
)

// RepositoryCommands creates repository management commands
func RepositoryCommands(cfg *config.Manager, cm ContainerManager, gm GitManager, sm ServiceManager, dbRepo db.RepositoryManager, database *db.DB) []*cobra.Command {
	commands := []*cobra.Command{}

	// Create operations instances
	var worktreeOps *operations.WorktreeOperations
	if database != nil && gm != nil && cm != nil && sm != nil {
		serviceAdapter := &worktreeServiceAdapter{mgr: sm}
		containerAdapter := &containerManagerAdapter{mgr: cm}
		worktreeOps = operations.NewWorktreeOperations(database, gm, containerAdapter, serviceAdapter, cfg)
	}

	// Clean, modern commands only - no deprecated cruft!

	// vibeman list
	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List all repositories and worktrees",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return listRepositoriesFromContainers(cmd.Context(), cfg, cm)
		},
	}
	commands = append(commands, listCmd)

	// vibeman status [repository-name] - Updated to be optional
	statusCmd := &cobra.Command{
		Use:   "status [repository-name]",
		Short: "Show repository status",
		Long:  "Show repository status. If no repository name is provided, uses the current directory's repository.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repositoryName := ""
			if len(args) > 0 {
				repositoryName = args[0]
			} else {
				// Try to get repository name from current directory
				if name, err := GetRepositoryName(cfg); err == nil {
					repositoryName = name
				} else {
					return fmt.Errorf("no repository name specified and not in a repository directory: %w", err)
				}
			}
			return repositoryStatus(cmd.Context(), repositoryName, cfg, cm, gm, sm)
		},
	}
	commands = append(commands, statusCmd)

	// Add new container operation commands that work within repository directories
	// vibeman start [worktree]
	newStartCmd := &cobra.Command{
		Use:   "start [worktree]",
		Short: "Start worktree containers",
		Long:  "Start worktree containers. If no worktree is specified, starts the main worktree.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repositoryName, currentEnv, err := GetCurrentRepositoryAndEnv(cfg)
			if err != nil {
				return RepositoryConfigError()
			}

			env := currentEnv
			if len(args) > 0 {
				env = args[0]
			}

			if worktreeOps == nil {
				return fmt.Errorf("operations not initialized")
			}
			// Find the worktree and start it
			worktrees, err := dbRepo.GetWorktreesByRepository(cmd.Context(), repositoryName)
			if err != nil {
				return fmt.Errorf("failed to list worktrees: %w", err)
			}
			for _, wt := range worktrees {
				if wt.Name == env || (env == "main" && wt.Name == repositoryName) {
					return worktreeOps.StartWorktree(cmd.Context(), wt.ID)
				}
			}
			return fmt.Errorf("worktree '%s' not found", env)
		},
	}
	commands = append(commands, newStartCmd)

	// vibeman stop [worktree]
	newStopCmd := &cobra.Command{
		Use:   "stop [worktree]",
		Short: "Stop worktree containers",
		Long:  "Stop worktree containers. If no worktree is specified, stops the main worktree.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repositoryName, currentEnv, err := GetCurrentRepositoryAndEnv(cfg)
			if err != nil {
				return RepositoryConfigError()
			}

			env := currentEnv
			if len(args) > 0 {
				env = args[0]
			}

			if worktreeOps == nil {
				return fmt.Errorf("operations not initialized")
			}
			// Find the worktree and stop it
			worktrees, err := dbRepo.GetWorktreesByRepository(cmd.Context(), repositoryName)
			if err != nil {
				return fmt.Errorf("failed to list worktrees: %w", err)
			}
			for _, wt := range worktrees {
				if wt.Name == env || (env == "main" && wt.Name == repositoryName) {
					return worktreeOps.StopWorktree(cmd.Context(), wt.ID)
				}
			}
			return fmt.Errorf("worktree '%s' not found", env)
		},
	}
	commands = append(commands, newStopCmd)

	// vibeman shell [worktree]
	shellCmd := &cobra.Command{
		Use:   "shell [worktree]",
		Short: "Open shell in worktree container",
		Long:  "Open an interactive shell in the worktree container. If no worktree is specified, uses the main worktree.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repositoryName, currentEnv, err := GetCurrentRepositoryAndEnv(cfg)
			if err != nil {
				return RepositoryConfigError()
			}

			worktree := currentEnv
			if len(args) > 0 {
				worktree = args[0]
			}

			user, _ := cmd.Flags().GetString("user")
			shell, _ := cmd.Flags().GetString("shell")
			return repositoryShell(cmd.Context(), repositoryName, worktree, user, shell, cfg, cm)
		},
	}
	shellCmd.Flags().StringP("user", "u", "", "User to run shell as")
	shellCmd.Flags().StringP("shell", "s", "", "Shell to use (default: /bin/bash or /bin/sh)")
	commands = append(commands, shellCmd)

	// vibeman ssh [worktree]
	sshCmd := &cobra.Command{
		Use:   "ssh [worktree]",
		Short: "SSH into worktree container",
		Long:  "Open an SSH-like connection to the worktree container. If no worktree is specified, uses the main worktree.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repositoryName, currentEnv, err := GetCurrentRepositoryAndEnv(cfg)
			if err != nil {
				return RepositoryConfigError()
			}

			worktree := currentEnv
			if len(args) > 0 {
				worktree = args[0]
			}

			user, _ := cmd.Flags().GetString("user")
			return repositorySSH(cmd.Context(), repositoryName, worktree, user, cfg, cm)
		},
	}
	sshCmd.Flags().StringP("user", "u", "", "User to SSH as (default: root)")
	commands = append(commands, sshCmd)

	// vibeman logs [worktree]
	logsCmd := &cobra.Command{
		Use:   "logs [worktree]",
		Short: "Show container logs",
		Long:  "Show container logs for the worktree. If no worktree is specified, uses the main worktree.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repositoryName, currentEnv, err := GetCurrentRepositoryAndEnv(cfg)
			if err != nil {
				return RepositoryConfigError()
			}

			worktree := currentEnv
			if len(args) > 0 {
				worktree = args[0]
			}

			follow, _ := cmd.Flags().GetBool("follow")
			tail, _ := cmd.Flags().GetInt("tail")
			since, _ := cmd.Flags().GetString("since")
			timestamps, _ := cmd.Flags().GetBool("timestamps")
			return repositoryLogs(cmd.Context(), repositoryName, worktree, follow, tail, since, timestamps, cfg, cm)
		},
	}
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	logsCmd.Flags().IntP("tail", "n", 100, "Number of lines to show from the end of the logs")
	logsCmd.Flags().StringP("since", "s", "", "Show logs since timestamp")
	logsCmd.Flags().BoolP("timestamps", "t", false, "Show timestamps")
	commands = append(commands, logsCmd)

	return commands
}

func createRepository(ctx context.Context, name string, cfg *config.Manager, cm ContainerManager, gm GitManager, sm ServiceManager) error {
	// Validate repository configuration exists
	if cfg.Repository.Repository.Name == "" {
		return fmt.Errorf("no repository configuration found. Please create a vibeman.toml file")
	}

	if cfg.Repository.Repository.Name != name {
		return fmt.Errorf("repository name '%s' does not match configuration name '%s'", name, cfg.Repository.Repository.Name)
	}

	logger.WithFields(logger.Fields{"repository": name}).Info("Creating repository")

	// Create git worktree if configured
	if cfg.Repository.Repository.Git.RepoURL != "" {
		worktreeDir := filepath.Join("../", name+"-worktrees", name)
		mainRepoDir := filepath.Join("../", name+"-worktrees", ".repos", name)

		logger.Info("Setting up git repository...")

		// Clone repository if not exists
		if !gm.IsRepository(mainRepoDir) {
			logger.WithFields(logger.Fields{"repo_url": cfg.Repository.Repository.Git.RepoURL}).Info("Cloning repository")
			if err := gm.CloneRepository(ctx, cfg.Repository.Repository.Git.RepoURL, mainRepoDir); err != nil {
				return fmt.Errorf("failed to clone repository: %w", err)
			}
		}

		// Create worktree for main branch
		branch := cfg.Repository.Repository.Git.DefaultBranch
		if branch == "" {
			branch = "main"
		}

		logger.WithFields(logger.Fields{"branch": branch}).Info("Creating worktree")
		if err := gm.CreateWorktree(ctx, mainRepoDir, branch, worktreeDir); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	}

	// Start required services
	if len(cfg.Repository.Repository.Services) > 0 {
		logger.Info("Starting required services...")
		for serviceName, serviceReq := range cfg.Repository.Repository.Services {
			if serviceReq.Required {
				if err := sm.StartService(ctx, serviceName); err != nil {
					return fmt.Errorf("failed to start service %s: %w", serviceName, err)
				}
				if err := sm.AddReference(serviceName, name); err != nil {
					return fmt.Errorf("failed to add service reference: %w", err)
				}
				logger.WithFields(logger.Fields{"service": serviceName}).Info("✓ Started service")
			}
		}
	}

	// Create container using compose configuration
	composeFile := cfg.Repository.Repository.Container.ComposeFile
	services := cfg.Repository.Repository.Container.Services

	if composeFile == "" {
		return fmt.Errorf("repository configuration missing compose_file")
	}

	logger.WithFields(logger.Fields{"compose_file": composeFile, "services": strings.Join(services, ", ")}).Info("Creating container using compose")
	// TODO: Implement CreateFromCompose method in ContainerManager
	// For now, use default create method with default image
	container, err := cm.Create(ctx, name, "main", "ubuntu:22.04")
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Get repository directory for reference (compose handles working directory)
	_, err = os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Run setup commands
	if len(cfg.Repository.Repository.Container.Setup) > 0 {
		logger.Info("Running setup commands...")
		for _, cmd := range cfg.Repository.Repository.Container.Setup {
			logger.WithFields(logger.Fields{"command": cmd}).Info("Running setup command")
			output, err := cm.Exec(ctx, container.ID, strings.Split(cmd, " "))
			if err != nil {
				logger.WithFields(logger.Fields{"command": cmd, "error": err}).Error("Setup command failed")
			} else {
				logger.WithFields(logger.Fields{"output": strings.TrimSpace(string(output))}).Info("Setup command succeeded")
			}
		}
	}

	logger.WithFields(logger.Fields{"repository": name, "container_id": container.ID}).Info("✓ Repository created successfully")

	return nil
}

func startRepository(ctx context.Context, name string, env string, cfg *config.Manager, cm ContainerManager, gm GitManager, sm ServiceManager) error {
	logger.WithFields(logger.Fields{"repository": name, "environment": env}).Info("Starting repository")

	// Find container
	containerName := fmt.Sprintf("%s-%s", name, env)
	if env == "main" {
		containerName = name
	}

	container, err := cm.GetByName(ctx, containerName)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	// Start required services
	if cfg.Repository.Repository.Name == name && len(cfg.Repository.Repository.Services) > 0 {
		for serviceName, serviceReq := range cfg.Repository.Repository.Services {
			if serviceReq.Required {
				if err := sm.StartService(ctx, serviceName); err != nil {
					logger.WithFields(logger.Fields{"service": serviceName, "error": err}).Warn("Failed to start service")
				}
			}
		}
	}

	// Start container
	if err := cm.Start(ctx, container.ID); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Note: Lifecycle hooks are now handled by docker-compose

	// Run container setup commands if configured
	if cfg.Repository.Repository.Name == name && len(cfg.Repository.Repository.Container.Setup) > 0 {
		logger.Info("Running setup commands...")
		for _, cmd := range cfg.Repository.Repository.Container.Setup {
			logger.WithFields(logger.Fields{"command": cmd}).Info("Running setup command")
			output, err := cm.Exec(ctx, container.ID, strings.Split(cmd, " "))
			if err != nil {
				logger.WithFields(logger.Fields{"command": cmd, "error": err}).Error("Setup command failed")
			} else {
				logger.WithFields(logger.Fields{"output": strings.TrimSpace(string(output))}).Info("Setup command succeeded")
			}
		}
	}

	logger.WithFields(logger.Fields{"repository": name}).Info("✓ Repository started successfully")
	return nil
}

func stopRepository(ctx context.Context, name string, env string, cfg *config.Manager, cm ContainerManager, sm ServiceManager) error {
	logger.WithFields(logger.Fields{"repository": name, "environment": env}).Info("Stopping repository")

	// Find container
	containerName := fmt.Sprintf("%s-%s", name, env)
	if env == "main" {
		containerName = name
	}

	container, err := cm.GetByName(ctx, containerName)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	// Note: Pre-stop commands are now handled by docker-compose

	// Stop container
	if err := cm.Stop(ctx, container.ID); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	logger.WithFields(logger.Fields{"repository": name}).Info("✓ Repository stopped successfully")
	return nil
}

func destroyRepository(ctx context.Context, name string, env string, cfg *config.Manager, cm ContainerManager, gm GitManager, sm ServiceManager) error {
	logger.WithFields(logger.Fields{"repository": name, "environment": env}).Info("Destroying repository")

	// Find container
	containerName := fmt.Sprintf("%s-%s", name, env)
	if env == "main" {
		containerName = name
	}

	container, err := cm.GetByName(ctx, containerName)
	if err != nil {
		// Container might not exist
		logger.Info("Container not found, checking worktree...")
	} else {
		// Stop container if running
		_ = cm.Stop(ctx, container.ID)

		// Remove container
		if err := cm.Remove(ctx, container.ID); err != nil {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}

	// Remove service references
	if cfg.Repository.Repository.Name == name {
		for serviceName := range cfg.Repository.Repository.Services {
			_ = sm.RemoveReference(serviceName, name)
		}
	}

	// Remove worktree if it's not the main environment
	if env != "main" && false { // Git config removed
		worktreeDir := filepath.Join("../", name+"-worktrees", fmt.Sprintf("%s-%s", name, env))
		if err := gm.RemoveWorktree(ctx, worktreeDir); err != nil {
			logger.WithFields(logger.Fields{"error": err}).Warn("Failed to remove worktree")
		}
	}

	logger.WithFields(logger.Fields{"repository": name}).Info("✓ Repository destroyed successfully")
	return nil
}

func listRepositoriesFromContainers(ctx context.Context, cfg *config.Manager, cm ContainerManager) error {
	containers, err := cm.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		logger.Info("No containers found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "REPOSITORY\tWORKTREE\tSTATUS\tCONTAINER ID\tCREATED")

	for _, c := range containers {
		repository := c.Repository
		worktree := c.Environment
		if worktree == "" {
			worktree = "main"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", repository, worktree, c.Status, c.ID[:12], c.CreatedAt)
	}

	w.Flush()
	return nil
}

func repositoryStatus(ctx context.Context, name string, cfg *config.Manager, cm ContainerManager, gm GitManager, sm ServiceManager) error {
	logger.WithFields(logger.Fields{"repository": name}).Info("Repository status")

	// Container status
	containers, err := cm.GetByRepository(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get repository containers: %w", err)
	}

	logger.Info("Containers:")
	if len(containers) == 0 {
		logger.Info("  No containers found")
	} else {
		for _, c := range containers {
			worktree := c.Environment
			if worktree == "" {
				worktree = "main"
			}
			logger.WithFields(logger.Fields{
				"container": c.Name,
				"worktree":  worktree,
				"status":    c.Status,
				"id":        c.ID,
				"image":     c.Image,
			}).Info("Container details")
		}
	}

	// Git status
	if false { // Git config removed
		logger.Info("\nGit Worktrees:")
		mainRepoDir := filepath.Join("../", name+"-worktrees", ".repos", name)
		if gm.IsRepository(mainRepoDir) {
			worktrees, err := gm.ListWorktrees(ctx, mainRepoDir)
			if err == nil {
				for _, wt := range worktrees {
					logger.WithFields(logger.Fields{
						"path":   wt.Path,
						"branch": wt.Branch,
						"head":   wt.Commit[:8],
					}).Info("Worktree details")
				}
			}
		}
	}

	// Service status
	if cfg.Repository.Repository.Name == name && len(cfg.Repository.Repository.Services) > 0 {
		logger.Info("\nServices:")
		for serviceName := range cfg.Repository.Repository.Services {
			svcInterface, err := sm.GetService(serviceName)
			if err != nil {
				logger.WithFields(logger.Fields{"service": serviceName}).Info("Service not found")
			} else {
				// Type assert to access fields
				svc, ok := svcInterface.(*types.ServiceInstance)
				if !ok {
					logger.WithFields(logger.Fields{"service": serviceName}).Error("Unexpected service type")
					continue
				}
				
				if svc.Status == types.ServiceStatusRunning {
					logger.WithFields(logger.Fields{
						"service": serviceName,
						"status":  svc.Status,
						"uptime":  time.Since(svc.StartTime).Round(time.Second),
					}).Info("Service details")
				} else {
					logger.WithFields(logger.Fields{
						"service": serviceName,
						"status":  svc.Status,
					}).Info("Service details")
				}
			}
		}
	}

	return nil
}

// New helper functions for the updated commands
func repositoryShell(ctx context.Context, repositoryName, worktree, user, shell string, cfg *config.Manager, cm ContainerManager) error {
	// Find container
	containerName := fmt.Sprintf("%s-%s", repositoryName, worktree)
	if worktree == "main" {
		containerName = repositoryName
	}
	container, err := cm.GetByName(ctx, containerName)
	if err != nil {
		return fmt.Errorf("worktree '%s' not found", worktree)
	}

	// Check if container is running (handle different status formats)
	status := strings.ToLower(container.Status)
	if !strings.Contains(status, "running") && !strings.Contains(status, "up") {
		return fmt.Errorf("container is not running (status: %s)", container.Status)
	}

	// Determine shell to use
	if shell == "" {
		// Try common shells in order
		shells := []string{"/bin/bash", "/bin/sh", "/bin/zsh", "/bin/ash"}
		for _, s := range shells {
			// Check if shell exists
			checkCmd := []string{"test", "-x", s}
			if _, err := cm.Exec(ctx, container.ID, checkCmd); err == nil {
				shell = s
				break
			}
		}
		if shell == "" {
			shell = "/bin/sh" // Fallback
		}
	}

	logger.WithFields(logger.Fields{"container": containerName, "shell": shell}).Info("Opening shell")

	// Use the Shell method from container manager
	return cm.Shell(ctx, container.ID, shell)
}

func repositorySSH(ctx context.Context, repositoryName, worktree, user string, cfg *config.Manager, cm ContainerManager) error {
	// Find container
	containerName := fmt.Sprintf("%s-%s", repositoryName, worktree)
	if worktree == "main" {
		containerName = repositoryName
	}
	container, err := cm.GetByName(ctx, containerName)
	if err != nil {
		return fmt.Errorf("worktree '%s' not found", worktree)
	}

	// Check if container is running (handle different status formats)
	status := strings.ToLower(container.Status)
	if !strings.Contains(status, "running") && !strings.Contains(status, "up") {
		return fmt.Errorf("container is not running (status: %s)", container.Status)
	}

	// Use the SSH method from container manager
	return cm.SSH(ctx, container.ID, user)
}

func repositoryLogs(ctx context.Context, repositoryName, worktree string, follow bool, tail int, since string, timestamps bool, cfg *config.Manager, cm ContainerManager) error {
	// Find container
	containerName := fmt.Sprintf("%s-%s", repositoryName, worktree)
	if worktree == "main" {
		containerName = repositoryName
	}
	container, err := cm.GetByName(ctx, containerName)
	if err != nil {
		return fmt.Errorf("worktree '%s' not found", worktree)
	}

	// Get logs
	logs, err := cm.Logs(ctx, container.ID, follow)
	if err != nil {
		return err
	}

	// Print logs
	fmt.Print(string(logs))
	return nil
}
