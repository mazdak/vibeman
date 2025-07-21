package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"vibeman/internal/config"
	"vibeman/internal/container"
	"vibeman/internal/db"
	"vibeman/internal/logger"
	"vibeman/internal/operations"
	"vibeman/internal/xdg"

	"github.com/spf13/cobra"
)

// WorktreeCommands creates worktree management commands
func WorktreeCommands(cfg *config.Manager, cm ContainerManager, gm GitManager, sm ServiceManager, dbRepo db.RepositoryManager, database *db.DB) []*cobra.Command {
	commands := []*cobra.Command{}

	// Create operations instance
	var worktreeOps *operations.WorktreeOperations
	var repoOps *operations.RepositoryOperations
	if database != nil && gm != nil && cm != nil && sm != nil {
		serviceAdapter := &worktreeServiceAdapter{mgr: sm}
		containerAdapter := &containerManagerAdapter{mgr: cm}
		worktreeOps = operations.NewWorktreeOperations(database, gm, containerAdapter, serviceAdapter, cfg)
		repoOps = operations.NewRepositoryOperations(cfg, gm, database)
	}

	// vibeman worktree add [repo-name] <worktree-name>
	addCmd := &cobra.Command{
		Use:   "add [repo-name] <worktree-name>",
		Short: "Create a new worktree development environment",
		Long: `Create a new worktree development environment with:
  - Git worktree for the development branch
  - Dedicated container for the worktree
  - Logging folder structure
  - Generated CLAUDE.md with instructions

If repo-name is not provided, it will be detected from the current git repository.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var repoName, worktreeName string

			if len(args) == 1 {
				// Only worktree name provided, detect repo from context
				worktreeName = args[0]

				currentDir, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				repoName, _, err = gm.GetRepositoryAndEnvironmentFromPath(currentDir)
				if err != nil {
					return fmt.Errorf("failed to detect repository from current directory: %w\nPlease specify the repository name or run from within a git repository", err)
				}

				logger.WithFields(logger.Fields{"repository": repoName}).Info("Detected repository")
			} else {
				// Both repo and worktree name provided
				repoName = args[0]
				worktreeName = args[1]
			}

			// Optional flags
			baseBranch, _ := cmd.Flags().GetString("base")
			skipSetup, _ := cmd.Flags().GetBool("skip-setup")
			containerImage, _ := cmd.Flags().GetString("image")

			if worktreeOps == nil || repoOps == nil {
				return fmt.Errorf("operations not initialized")
			}
			
			// Get repository by name
			repos, err := repoOps.ListRepositories(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to list repositories: %w", err)
			}
			
			var repoID string
			for _, repo := range repos {
				if repo.Name == repoName {
					repoID = repo.ID
					break
				}
			}
			
			if repoID == "" {
				return fmt.Errorf("repository '%s' not found", repoName)
			}
			
			req := operations.CreateWorktreeRequest{
				RepositoryID: repoID,
				Name: worktreeName,
				Branch: worktreeName, // Use worktree name as branch name
				BaseBranch: baseBranch,
				SkipSetup: skipSetup,
				ContainerImage: containerImage,
				AutoStart: true,
			}
			
			_, err = worktreeOps.CreateWorktree(cmd.Context(), req)
			if err != nil {
				return HandleError(err)
			}
			return nil
		},
	}
	addCmd.Flags().StringP("base", "b", "", "Base branch for the worktree (default: repository's default branch)")
	addCmd.Flags().Bool("skip-setup", false, "Skip running setup commands")
	addCmd.Flags().StringP("image", "i", "", "Container image to use (overrides repository config)")

	commands = append(commands, addCmd)

	// vibeman worktree list [repo-name]
	listCmd := &cobra.Command{
		Use:   "list [repo-name]",
		Short: "List all worktree environments",
		Long:  "List all worktree environments. If repo-name is provided, only show features for that repository.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoName := ""
			if len(args) > 0 {
				repoName = args[0]
			}
			if worktreeOps == nil || dbRepo == nil {
				return fmt.Errorf("operations not initialized")
			}
			return listWorktreesWithOps(cmd.Context(), repoName, dbRepo, database)
		},
	}
	commands = append(commands, listCmd)

	// vibeman worktree status [repo-name] [worktree-name]
	statusCmd := &cobra.Command{
		Use:   "status [repo-name] [worktree-name]",
		Short: "Show worktree environment status",
		Long:  "Show the status of a worktree environment. If no arguments are provided and you're in a worktree, shows the current worktree's status.",
		Args:  cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var repoName, worktreeName string

			if len(args) == 0 {
				// No arguments provided, detect both repo and worktree from context
				currentDir, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				var envName string
				repoName, envName, err = gm.GetRepositoryAndEnvironmentFromPath(currentDir)
				if err != nil {
					return fmt.Errorf("failed to detect repository from current directory: %w\nPlease run from within a git repository or specify the repository and worktree names", err)
				}

				// If we're in main branch, we need a worktree name
				if envName == "main" {
					return fmt.Errorf("you are in the main branch, please specify a worktree name")
				}

				worktreeName = envName
				logger.WithFields(logger.Fields{"repository": repoName, "worktree": worktreeName}).Info("Showing status for current worktree")
			} else if len(args) == 1 {
				// Only worktree name provided, detect repo from context
				worktreeName = args[0]

				currentDir, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				repoName, _, err = gm.GetRepositoryAndEnvironmentFromPath(currentDir)
				if err != nil {
					return fmt.Errorf("failed to detect repository from current directory: %w\nPlease specify the repository name or run from within a git repository", err)
				}
			} else {
				// Both repo and worktree name provided
				repoName = args[0]
				worktreeName = args[1]
			}
			return worktreeStatus(cmd.Context(), repoName, worktreeName, cfg, cm, gm)
		},
	}
	commands = append(commands, statusCmd)

	// vibeman worktree remove [repo-name] <worktree-name>
	removeCmd := &cobra.Command{
		Use:   "remove [repo-name] <worktree-name>",
		Short: "Remove a worktree environment",
		Long:  "Remove a worktree environment including its worktree, container, and logs. If repo-name is not provided, it will be detected from the current git repository.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var repoName, worktreeName string

			if len(args) == 1 {
				// Only worktree name provided, detect repo from context
				worktreeName = args[0]

				currentDir, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				repoName, _, err = gm.GetRepositoryAndEnvironmentFromPath(currentDir)
				if err != nil {
					return fmt.Errorf("failed to detect repository from current directory: %w\nPlease specify the repository name or run from within a git repository", err)
				}
			} else {
				// Both repo and worktree name provided
				repoName = args[0]
				worktreeName = args[1]
			}
			force, _ := cmd.Flags().GetBool("force")
			if worktreeOps == nil || dbRepo == nil {
				return fmt.Errorf("operations not initialized")
			}
			
			// Find worktree by repository and name
			worktrees, err := dbRepo.GetWorktreesByRepository(cmd.Context(), repoName)
			if err != nil {
				return fmt.Errorf("failed to list worktrees: %w", err)
			}
			
			var worktreeID string
			for _, wt := range worktrees {
				if wt.Name == worktreeName {
					worktreeID = wt.ID
					break
				}
			}
			
			if worktreeID == "" {
				return fmt.Errorf("worktree '%s' not found in repository '%s'", worktreeName, repoName)
			}
			
			return worktreeOps.RemoveWorktree(cmd.Context(), worktreeID, force)
		},
	}
	removeCmd.Flags().BoolP("force", "f", false, "Force removal without confirmation")
	commands = append(commands, removeCmd)

	// vibeman worktree shell [repo-name] [worktree-name] [service-name]
	shellCmd := &cobra.Command{
		Use:   "shell [repo-name] [worktree-name] [service-name]",
		Short: "Open shell in worktree container",
		Long: `Open an interactive shell in a worktree container.

If no arguments are provided and you're in a worktree, opens shell in that worktree's container.
If only worktree-name is provided, repository is auto-detected from current directory.
If service-name is provided, opens shell in that specific service container.
If multiple services are running and no service is specified, you'll be prompted to select one.`,
		Args: cobra.RangeArgs(0, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			var repoName, worktreeName, serviceName string

			if len(args) == 0 {
				// No arguments provided, detect both repo and worktree from context
				currentDir, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				var envName string
				repoName, envName, err = gm.GetRepositoryAndEnvironmentFromPath(currentDir)
				if err != nil {
					return fmt.Errorf("failed to detect repository from current directory: %w\nPlease run from within a worktree or specify the repository and worktree names", err)
				}

				// If we're in main branch, we need a worktree name
				if envName == "main" || envName == "" {
					return fmt.Errorf("you are in the main branch, please specify a worktree name or navigate to a worktree")
				}

				worktreeName = envName
				logger.WithFields(logger.Fields{"repository": repoName, "worktree": worktreeName}).Info("Opening shell in current worktree")
			} else if len(args) == 1 {
				// Only worktree name provided, detect repo from context
				worktreeName = args[0]

				currentDir, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				repoName, _, err = gm.GetRepositoryAndEnvironmentFromPath(currentDir)
				if err != nil {
					return fmt.Errorf("failed to detect repository from current directory: %w\nPlease specify the repository name or run from within a git repository", err)
				}
			} else if len(args) == 2 {
				// Both repo and worktree name provided
				repoName = args[0]
				worktreeName = args[1]
			} else {
				// All three arguments provided
				repoName = args[0]
				worktreeName = args[1]
				serviceName = args[2]
			}

			user, _ := cmd.Flags().GetString("user")
			shell, _ := cmd.Flags().GetString("shell")
			service, _ := cmd.Flags().GetString("service")
			if service == "" {
				service = serviceName
			}
			return worktreeShell(cmd.Context(), repoName, worktreeName, service, user, shell, cm)
		},
	}
	shellCmd.Flags().StringP("user", "u", "", "User to run shell as")
	shellCmd.Flags().StringP("shell", "s", "", "Shell to use (default: /bin/bash or /bin/sh)")
	shellCmd.Flags().String("service", "", "Service name to connect to (prompts if multiple services)")
	commands = append(commands, shellCmd)

	return commands
}

// listWorktreesWithOps lists worktrees using operations
func listWorktreesWithOps(ctx context.Context, repoName string, dbRepo db.RepositoryManager, database *db.DB) error {
	var worktrees []*db.Worktree
	var err error
	
	if repoName != "" {
		worktrees, err = dbRepo.GetWorktreesByRepository(ctx, repoName)
	} else {
		// List all worktrees by getting all repositories and their worktrees
		repos, err := dbRepo.ListRepositories(ctx)
		if err != nil {
			return fmt.Errorf("failed to list repositories: %w", err)
		}
		
		worktrees = make([]*db.Worktree, 0)
		for _, repo := range repos {
			repoWorktrees, err := dbRepo.GetWorktreesByRepository(ctx, repo.ID)
			if err != nil {
				continue
			}
			worktrees = append(worktrees, repoWorktrees...)
		}
	}
	
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}
	
	if len(worktrees) == 0 {
		if repoName != "" {
			logger.Info("No worktrees found for repository: " + repoName)
		} else {
			logger.Info("No worktrees found")
		}
		return nil
	}
	
	// Print worktrees
	for _, wt := range worktrees {
		logger.WithFields(logger.Fields{
			"repository": wt.RepositoryID,
			"worktree": wt.Name,
			"branch": wt.Branch,
			"path": wt.Path,
			"status": wt.Status,
		}).Info("Worktree")
	}
	
	return nil
}

func createWorktree(ctx context.Context, repoName, worktreeName, baseBranch string, skipSetup bool, containerImage string, cfg *config.Manager, cm ContainerManager, gm GitManager, sm ServiceManager) error {

	// Validate worktree name
	if err := validateWorktreeName(worktreeName); err != nil {
		return fmt.Errorf("invalid worktree name: %w", err)
	}

	// Get base branch (default to project config or main)
	if baseBranch == "" {
		if cfg.Repository != nil && cfg.Repository.Repository.Git.DefaultBranch != "" {
			baseBranch = cfg.Repository.Repository.Git.DefaultBranch
		} else {
			baseBranch = "main"
		}
	}

	// Construct paths
	worktreesDir := ""
	if cfg.Repository != nil && cfg.Repository.Repository.Worktrees.Directory != "" {
		worktreesDir = cfg.Repository.Repository.Worktrees.Directory
	} else {
		// Use default from global config or fallback
		globalCfg, _ := config.LoadGlobalConfig()
		if globalCfg != nil && globalCfg.Storage.WorktreesPath != "" {
			worktreesDir = globalCfg.Storage.WorktreesPath
		} else {
			homeDir, _ := os.UserHomeDir()
			worktreesDir = filepath.Join(homeDir, "vibeman", "worktrees")
		}
	}

	// Ensure worktrees directory is absolute
	if !filepath.IsAbs(worktreesDir) {
		// If relative, make it relative to current directory
		cwd, _ := os.Getwd()
		worktreesDir = filepath.Join(cwd, worktreesDir)
	}

	worktreeDir := filepath.Join(worktreesDir, worktreeName)

	// Check if worktree already exists
	if _, err := os.Stat(worktreeDir); err == nil {
		return fmt.Errorf("worktree directory already exists: %s", worktreeDir)
	}

	logger.WithFields(logger.Fields{"worktree": worktreeName, "repository": repoName}).Info("Creating worktree")

	// Create branch name
	branchPrefix := ""
	if cfg.Repository != nil && cfg.Repository.Repository.Git.WorktreePrefix != "" {
		branchPrefix = cfg.Repository.Repository.Git.WorktreePrefix
	} else {
		branchPrefix = "worktree/"
	}
	branchName := fmt.Sprintf("%s%s", strings.TrimSuffix(branchPrefix, "/"), worktreeName)

	// Get repo URL
	repoURL := "."
	if cfg.Repository != nil && cfg.Repository.Repository.Git.RepoURL != "" {
		repoURL = cfg.Repository.Repository.Git.RepoURL
	}
	
	// Create git worktree
	logger.WithFields(logger.Fields{"path": worktreeDir}).Info("Creating git worktree")
	if err := gm.CreateWorktree(ctx, repoURL, branchName, worktreeDir); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Run post-checkout commands if configured (using worktree init script)
	if !skipSetup && cfg.Repository != nil && cfg.Repository.Repository.Setup.WorktreeInit != "" {
		logger.Info("Running worktree init script...")
		cmd := cfg.Repository.Repository.Setup.WorktreeInit
		if err := runCommandInDirectory(worktreeDir, cmd); err != nil {
			logger.WithFields(logger.Fields{"command": cmd, "error": err}).Warn("Post-checkout command failed")
		}
	}

	// Determine container compose configuration
	var composeFile string
	var services []string
	if containerImage == "" {
		// Try to load project config from the worktree to get compose configuration
		projectConfigPath := filepath.Join(worktreeDir, "vibeman.toml")
		if _, err := os.Stat(projectConfigPath); err == nil {
			projectCfg := config.New()
			if err := projectCfg.Load(); err == nil {
				composeFile = projectCfg.Repository.Repository.Container.ComposeFile
				services = projectCfg.Repository.Repository.Container.Services
			}
		}

		// Fallback to defaults
		if composeFile == "" {
			composeFile = "docker-compose.yml"
		}
		// services can be empty (which means all services)
	}

	// Create container for the worktree
	// Docker compose will create containers with service suffixes
	containerName := fmt.Sprintf("%s-%s", repoName, worktreeName)

	var worktreeContainer *container.Container

	if containerImage != "" {
		// Legacy mode: create with specified image
		logger.WithFields(logger.Fields{"container": containerName, "image": containerImage}).Info("Creating container with image")
		var err error
		worktreeContainer, err = cm.Create(ctx, repoName, worktreeName, containerImage)
		if err != nil {
			return fmt.Errorf("failed to create container: %w", err)
		}
	} else {
		// New mode: create from compose
		logger.WithFields(logger.Fields{"container": containerName, "compose_file": composeFile, "services": strings.Join(services, ", ")}).Info("Creating container using compose")
		// TODO: Implement CreateFromCompose method in ContainerManager
		// For now, use default create method with default image
		var err error
		worktreeContainer, err = cm.Create(ctx, repoName, worktreeName, "ubuntu:22.04")
		if err != nil {
			return fmt.Errorf("failed to create container: %w", err)
		}
	}

	// Set up logging directory structure
	logsDir, err := getWorktreeLogsDirectory(repoName, worktreeName)
	if err != nil {
		return fmt.Errorf("failed to get logs directory: %w", err)
	}

	if err := setupLoggingStructure(logsDir); err != nil {
		return fmt.Errorf("failed to set up logging structure: %w", err)
	}

	// Generate CLAUDE.md with instructions
	claudePath := filepath.Join(worktreeDir, "CLAUDE.md")
	if err := generateWorktreeClaude(claudePath, repoName, worktreeName, branchName, containerName, logsDir); err != nil {
		logger.WithFields(logger.Fields{"error": err}).Warn("Failed to generate CLAUDE.md")
	}

	// Run container setup if not skipped
	if !skipSetup {
		// Try to load project config to get setup commands
		if projectConfig, err := loadProjectConfigFromWorktree(worktreeDir); err == nil && len(projectConfig.Repository.Container.Setup) > 0 {
			logger.Info("Running container setup...")
			for _, cmd := range projectConfig.Repository.Container.Setup {
				logger.WithFields(logger.Fields{"command": cmd}).Info("Running setup command")
				output, err := cm.Exec(ctx, worktreeContainer.ID, strings.Split(cmd, " "))
				if err != nil {
					logger.WithFields(logger.Fields{"command": cmd, "error": err}).Error("Setup command failed")
				} else {
					logger.WithFields(logger.Fields{"output": strings.TrimSpace(string(output))}).Info("Setup command succeeded")
				}
			}
		}
	}

	// Start required services if any
	if projectConfig, err := loadProjectConfigFromWorktree(worktreeDir); err == nil && len(projectConfig.Repository.Services) > 0 {
		logger.Info("Starting required services...")
		for serviceName, serviceReq := range projectConfig.Repository.Services {
			if serviceReq.Required {
				if err := sm.StartService(ctx, serviceName); err != nil {
					logger.WithFields(logger.Fields{"service": serviceName, "error": err}).Warn("Failed to start service")
				} else {
					if err := sm.AddReference(serviceName, containerName); err != nil {
						logger.WithFields(logger.Fields{"error": err}).Warn("Failed to add service reference")
					}
				}
			}
		}
	}

	fmt.Printf("\n✓ Worktree environment created successfully!\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Navigate to the worktree directory:\n")
	fmt.Printf("     cd %s\n", worktreeDir)
	fmt.Printf("  2. Start the container:\n")
	fmt.Printf("     vibeman start %s\n", worktreeName)
	fmt.Printf("  3. Open a shell in the container:\n")
	fmt.Printf("     vibeman worktree shell  # (from within the worktree)\n")
	fmt.Printf("     # or from anywhere:\n")
	fmt.Printf("     vibeman worktree shell %s %s\n", repoName, worktreeName)
	fmt.Printf("\nLogs will be saved to: %s\n", logsDir)

	return nil
}

// listWorktrees lists all worktree environments
func listWorktrees(ctx context.Context, repoName string, cfg *config.Manager, cm ContainerManager, gm GitManager) error {
	// Get all containers
	containers, err := cm.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Filter by repository if specified
	var features []*container.Container
	for _, c := range containers {
		// Skip main environments
		if c.Environment == "main" || c.Environment == "" {
			continue
		}

		if repoName == "" || c.Repository == repoName {
			features = append(features, c)
		}
	}

	if len(features) == 0 {
		if repoName != "" {
			fmt.Printf("No features found for repository '%s'\n", repoName)
		} else {
			logger.Info("No features found")
		}
		return nil
	}

	// Print header
	fmt.Printf("%-20s %-20s %-10s %-12s %-20s\n", "REPOSITORY", "FEATURE", "STATUS", "CONTAINER", "CREATED")
	logger.Info(strings.Repeat("-", 85))

	// Print features
	for _, f := range features {
		logger.Infof("%-20s %-20s %-10s %-12s %-20s",
			f.Repository,
			f.Environment,
			f.Status,
			f.ID[:12],
			f.CreatedAt,
		)
	}

	return nil
}

// worktreeStatus shows the status of a worktree environment
func worktreeStatus(ctx context.Context, repoName, worktreeName string, cfg *config.Manager, cm ContainerManager, gm GitManager) error {
	fmt.Printf("Worktree: %s/%s\n\n", repoName, worktreeName)

	// Check container status
	containerName := fmt.Sprintf("%s-%s", repoName, worktreeName)
	container, err := cm.GetByName(ctx, containerName)
	if err != nil {
		fmt.Printf("Container: Not found\n")
	} else {
		fmt.Printf("Container:\n")
		fmt.Printf("  Name: %s\n", container.Name)
		fmt.Printf("  Status: %s\n", container.Status)
		fmt.Printf("  ID: %s\n", container.ID)
		fmt.Printf("  Image: %s\n", container.Image)
		fmt.Printf("  Created: %s\n", container.CreatedAt)
	}

	// Check worktree status
	worktreesDir := ""
	if cfg.Repository != nil && cfg.Repository.Repository.Worktrees.Directory != "" {
		worktreesDir = cfg.Repository.Repository.Worktrees.Directory
	} else {
		// Use default from global config or fallback
		globalCfg, _ := config.LoadGlobalConfig()
		if globalCfg != nil && globalCfg.Storage.WorktreesPath != "" {
			worktreesDir = globalCfg.Storage.WorktreesPath
		} else {
			homeDir, _ := os.UserHomeDir()
			worktreesDir = filepath.Join(homeDir, "vibeman", "worktrees")
		}
	}
	
	worktreeDir := filepath.Join(worktreesDir, repoName, worktreeName)

	fmt.Printf("\nWorktree:\n")
	if info, err := os.Stat(worktreeDir); err == nil {
		fmt.Printf("  Path: %s\n", worktreeDir)
		fmt.Printf("  Exists: Yes\n")
		fmt.Printf("  Modified: %s\n", info.ModTime().Format(time.RFC3339))

		// Check for uncommitted changes
		if hasChanges, err := gm.HasUncommittedChanges(ctx, worktreeDir); err == nil {
			fmt.Printf("  Uncommitted changes: %v\n", hasChanges)
		}
	} else {
		fmt.Printf("  Path: %s\n", worktreeDir)
		fmt.Printf("  Exists: No\n")
	}

	// Check logs directory
	logsDir, err := getWorktreeLogsDirectory(repoName, worktreeName)
	if err == nil {
		fmt.Printf("\nLogs:\n")
		fmt.Printf("  Directory: %s\n", logsDir)
		if _, err := os.Stat(logsDir); err == nil {
			fmt.Printf("  Exists: Yes\n")
		} else {
			fmt.Printf("  Exists: No\n")
		}
	}

	return nil
}

// removeWorktree removes a worktree environment
func removeWorktree(ctx context.Context, repoName, worktreeName string, force bool, cfg *config.Manager, cm ContainerManager, gm GitManager, sm ServiceManager) error {
	// Get worktree path first to check git status
	var worktreeDir string
	var warnings []string

	// Get worktrees directory
	worktreesDir := ""
	if cfg.Repository != nil && cfg.Repository.Repository.Worktrees.Directory != "" {
		worktreesDir = cfg.Repository.Repository.Worktrees.Directory
	} else {
		// Use default from global config or fallback
		globalCfg, _ := config.LoadGlobalConfig()
		if globalCfg != nil && globalCfg.Storage.WorktreesPath != "" {
			worktreesDir = globalCfg.Storage.WorktreesPath
		} else {
			homeDir, _ := os.UserHomeDir()
			worktreesDir = filepath.Join(homeDir, "vibeman", "worktrees")
		}
	}
	
	worktreeDir = filepath.Join(worktreesDir, repoName, worktreeName)

	// Check for uncommitted changes
	if hasChanges, err := gm.HasUncommittedChanges(ctx, worktreeDir); err == nil && hasChanges {
		warnings = append(warnings, "  ⚠️  Uncommitted changes detected")
	}

	// Check for unpushed commits
	if hasUnpushed, err := gm.HasUnpushedCommits(ctx, worktreeDir); err == nil && hasUnpushed {
		warnings = append(warnings, "  ⚠️  Unpushed commits detected")
	}

	// Check if branch is merged
	if branch, err := gm.GetCurrentBranch(ctx, worktreeDir); err == nil {
		if merged, err := gm.IsBranchMerged(ctx, worktreeDir, branch); err == nil && !merged {
			warnings = append(warnings, fmt.Sprintf("  ⚠️  Branch '%s' has not been merged", branch))
		}
	}

	if !force {
		logger.WithFields(logger.Fields{"repository": repoName, "worktree": worktreeName}).Warn("This will remove the worktree environment including:")
		logger.Info("  - Git worktree")
		logger.Info("  - Container")
		logger.Info("  - Logs")

		if len(warnings) > 0 {
			logger.Warn("\nWarnings:")
			for _, warning := range warnings {
				logger.Warn(warning)
			}
		}

		fmt.Print("\nAre you sure? (y/N): ")

		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			logger.Info("Aborted")
			return nil
		}
	}

	logger.WithFields(logger.Fields{"repository": repoName, "worktree": worktreeName}).Info("Removing worktree")

	// Remove container - try both legacy name and project/environment search
	containerName := fmt.Sprintf("%s-%s", repoName, worktreeName)
	var foundContainer *container.Container

	// First try the legacy name
	if c, err := cm.GetByName(ctx, containerName); err == nil {
		foundContainer = c
	} else {
		// If not found by name, try to find by project and environment
		containers, err := cm.List(ctx)
		if err == nil {
			for _, c := range containers {
				if c.Repository == repoName && c.Environment == worktreeName {
					foundContainer = c
					break
				}
			}
		}
	}

	if foundContainer != nil {
		logger.Info("Stopping container...")
		_ = cm.Stop(ctx, foundContainer.ID)

		logger.Info("Removing container...")
		if err := cm.Remove(ctx, foundContainer.ID); err != nil {
			logger.WithFields(logger.Fields{"error": err}).Warn("Failed to remove container")
		}

		// Remove service references
		if projectConfig, err := loadProjectConfigFromContainer(foundContainer); err == nil {
			for serviceName := range projectConfig.Repository.Services {
				_ = sm.RemoveReference(serviceName, foundContainer.Name)
			}
		}
	} else {
		logger.Info("Container not found (may have been removed already)")
	}

	// Remove worktree
	// Get worktrees directory
	worktreesDir = ""
	if cfg.Repository != nil && cfg.Repository.Repository.Worktrees.Directory != "" {
		worktreesDir = cfg.Repository.Repository.Worktrees.Directory
	} else {
		// Use default from global config or fallback
		globalCfg, _ := config.LoadGlobalConfig()
		if globalCfg != nil && globalCfg.Storage.WorktreesPath != "" {
			worktreesDir = globalCfg.Storage.WorktreesPath
		} else {
			homeDir, _ := os.UserHomeDir()
			worktreesDir = filepath.Join(homeDir, "vibeman", "worktrees")
		}
	}
	
	worktreeDir = filepath.Join(worktreesDir, repoName, worktreeName)

	logger.Info("Removing worktree...")
	if err := gm.RemoveWorktree(ctx, worktreeDir); err != nil {
		// If git command fails, try manual removal
		if err := os.RemoveAll(worktreeDir); err != nil {
			logger.WithFields(logger.Fields{"error": err}).Warn("Failed to remove worktree")
		}
	}

	// Remove logs
	logsDir, err := getWorktreeLogsDirectory(repoName, worktreeName)
	if err == nil {
		logger.Info("Removing logs...")
		if err := os.RemoveAll(logsDir); err != nil {
			logger.WithFields(logger.Fields{"error": err}).Warn("Failed to remove logs")
		}
	}

	logger.WithFields(logger.Fields{"repository": repoName, "worktree": worktreeName}).Info("✓ Worktree removed successfully")
	return nil
}

// Helper functions


func validateWorktreeName(name string) error {
	if name == "" {
		return fmt.Errorf("worktree name cannot be empty")
	}

	// Check for invalid characters
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("worktree name contains invalid character: %s", char)
		}
	}

	// Check length
	if len(name) > 50 {
		return fmt.Errorf("worktree name too long (max 50 characters)")
	}

	return nil
}

func getWorktreeLogsDirectory(repoName, worktreeName string) (string, error) {
	stateDir, err := xdg.StateDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(stateDir, "logs", repoName, worktreeName), nil
}

func setupLoggingStructure(logsDir string) error {
	// Create main logs directory
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return err
	}

	// Create subdirectories for different log types
	subdirs := []string{"build", "runtime", "tests", "debug"}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(logsDir, subdir), 0755); err != nil {
			return err
		}
	}

	// Create initial log files
	timestamp := time.Now().Format("2006-01-02")
	initialFiles := map[string]string{
		"build/build.log":    fmt.Sprintf("# Build log started at %s\n", time.Now().Format(time.RFC3339)),
		"runtime/app.log":    fmt.Sprintf("# Application log started at %s\n", time.Now().Format(time.RFC3339)),
		"tests/test.log":     fmt.Sprintf("# Test log started at %s\n", time.Now().Format(time.RFC3339)),
		"debug/debug.log":    fmt.Sprintf("# Debug log started at %s\n", time.Now().Format(time.RFC3339)),
		"worktree-setup.log": fmt.Sprintf("# Worktree setup log for %s\n", timestamp),
	}

	for filename, content := range initialFiles {
		filepath := filepath.Join(logsDir, filename)
		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

func generateWorktreeClaude(path, repoName, worktreeName, branchName, containerName, logsDir string) error {
	content := fmt.Sprintf(`# Worktree Development: %s

This is a worktree development environment for the '%s' worktree in the '%s' repository.

## Environment Details

- **Worktree Name**: %s
- **Repository**: %s
- **Branch**: %s
- **Container**: %s
- **Logs Directory**: %s

## Quick Start

### Starting the Environment

`+"```bash"+`
# Start the container
vibeman start %s

# Open a shell in the container (from anywhere)
vibeman worktree shell %s %s

# Or if you're in the worktree, just:
vibeman worktree shell
`+"```"+`

### Working with the Code

This worktree is set up for development. You can:

1. Make changes to the code
2. Run tests in the container
3. Build and debug the application
4. Commit changes to the worktree branch

### Container Commands

`+"```bash"+`
# Check container status
vibeman status

# View container logs
vibeman logs %s

# Stop the container
vibeman stop %s
`+"```"+`

## Development Workflow

1. **Code Changes**: Make your changes in this directory
2. **Testing**: Run tests inside the container
3. **Building**: Use the container environment for consistent builds
4. **Debugging**: Logs are saved to the logs directory for troubleshooting

## Logs Structure

Logs are organized in the following structure:
- **build/**: Build-related logs
- **runtime/**: Application runtime logs
- **tests/**: Test execution logs
- **debug/**: Debug and troubleshooting logs

## Notes

- This environment is isolated from other features
- Changes are made to the '%s' branch
- The container provides a consistent development environment
- All logs are preserved in '%s'

## Cleanup

When you're done with this worktree:

`+"```bash"+`
vibeman worktree remove %s %s
`+"```"+`

This will remove the worktree, container, and logs.
`, worktreeName, worktreeName, repoName, worktreeName, repoName, branchName, containerName, logsDir,
		worktreeName, repoName, worktreeName, worktreeName, worktreeName, branchName, logsDir, repoName, worktreeName)

	return os.WriteFile(path, []byte(content), 0644)
}

func runCommandInDirectory(dir, command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func loadProjectConfigFromWorktree(worktreePath string) (*config.RepositoryConfig, error) {
	configPath := filepath.Join(worktreePath, "vibeman.toml")
	if _, err := os.Stat(configPath); err != nil {
		return nil, err
	}

	cfg := config.New()
	// Set working directory temporarily to load the config
	oldWd, _ := os.Getwd()
	os.Chdir(worktreePath)
	defer os.Chdir(oldWd)

	if err := cfg.Load(); err != nil {
		return nil, err
	}

	return cfg.Repository, nil
}

func loadProjectConfigFromContainer(container *container.Container) (*config.RepositoryConfig, error) {
	// Since we can't directly load the config from the container,
	// we'll return a minimal config based on available container metadata

	if container == nil {
		return nil, fmt.Errorf("container is nil")
	}

	// Create a minimal project config based on container info
	projectConfig := &config.RepositoryConfig{}

	// Set repository name from container metadata
	if container.Repository != "" {
		projectConfig.Repository.Name = container.Repository
	} else {
		// Try to extract from container name (format: projectname-environment)
		parts := strings.Split(container.Name, "-")
		if len(parts) > 0 {
			projectConfig.Repository.Name = parts[0]
		}
	}

	// Set container configuration (compose-first approach)
	// Note: In the simplified approach, compose file and service are expected to be
	// defined in the repository's vibeman.toml file, not extracted from container metadata

	// Extract project from container metadata
	if repositoryName, exists := container.EnvVars["VIBEMAN_REPOSITORY"]; exists && projectConfig.Repository.Name == "" {
		projectConfig.Repository.Name = repositoryName
	}

	// Note: This is a minimal implementation. In a production system,
	// you might store the full config path in container labels/metadata
	// and load it directly, or serialize the config into container labels.

	return projectConfig, nil
}

// worktreeShell opens an interactive shell in a worktree container
func worktreeShell(ctx context.Context, repoName, worktreeName, serviceName, user, shell string, cm ContainerManager) error {
	// List all containers for this worktree
	containers, err := cm.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Filter containers for this worktree
	var worktreeContainers []*container.Container
	prefix := fmt.Sprintf("%s-%s", repoName, worktreeName)
	for _, c := range containers {
		if strings.HasPrefix(c.Name, prefix) && c.Type != "ai" {
			// Check if container is running
			status := strings.ToLower(c.Status)
			if strings.Contains(status, "running") || strings.Contains(status, "up") {
				worktreeContainers = append(worktreeContainers, c)
			}
		}
	}

	if len(worktreeContainers) == 0 {
		return fmt.Errorf("no running containers found for worktree '%s/%s'", repoName, worktreeName)
	}

	var selectedContainer *container.Container

	// If service name is provided, find that specific container
	if serviceName != "" {
		containerName := fmt.Sprintf("%s-%s-%s", repoName, worktreeName, serviceName)
		for _, c := range worktreeContainers {
			if c.Name == containerName || strings.HasSuffix(c.Name, "-"+serviceName) {
				selectedContainer = c
				break
			}
		}
		if selectedContainer == nil {
			return fmt.Errorf("service '%s' not found in worktree '%s/%s'", serviceName, repoName, worktreeName)
		}
	} else if len(worktreeContainers) == 1 {
		// Only one container, use it
		selectedContainer = worktreeContainers[0]
	} else {
		// Multiple containers, prompt user to select
		fmt.Println("Multiple services available. Select one:")
		for i, c := range worktreeContainers {
			// Extract service name from container name
			service := strings.TrimPrefix(c.Name, prefix+"-")
			if service == "" {
				service = "main"
			}
			fmt.Printf("%d) %s\n", i+1, service)
		}

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter number: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		var choice int
		_, err = fmt.Sscanf(strings.TrimSpace(input), "%d", &choice)
		if err != nil || choice < 1 || choice > len(worktreeContainers) {
			return fmt.Errorf("invalid selection")
		}

		selectedContainer = worktreeContainers[choice-1]
	}


	// Determine shell to use
	if shell == "" {
		// Try common shells in order
		shells := []string{"/bin/bash", "/bin/sh", "/bin/zsh", "/bin/ash"}
		for _, s := range shells {
			// Check if shell exists
			checkCmd := []string{"test", "-x", s}
			if _, err := cm.Exec(ctx, selectedContainer.ID, checkCmd); err == nil {
				shell = s
				break
			}
		}
		if shell == "" {
			shell = "/bin/sh" // Fallback
		}
	}

	logger.WithFields(logger.Fields{"container": selectedContainer.Name, "shell": shell}).Info("Opening shell")

	// Use the Shell method from container manager
	return cm.Shell(ctx, selectedContainer.ID, shell)
}
