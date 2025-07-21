package operations

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"vibeman/internal/config"
	"vibeman/internal/constants"
	"vibeman/internal/container"
	"vibeman/internal/db"
	"vibeman/internal/errors"
	"vibeman/internal/logger"
	"vibeman/internal/validation"
	"vibeman/internal/xdg"

	"github.com/google/uuid"
)

// WorktreeOperations provides shared backend functions for worktree management
type WorktreeOperations struct {
	cfg          *config.Manager
	containerMgr ContainerManager
	gitMgr       GitManager
	db           *db.DB
	serviceMgr   ServiceManager
	logAggregator *LogAggregator
}

// NewWorktreeOperations creates a new WorktreeOperations instance
func NewWorktreeOperations(database *db.DB, gm GitManager, cm ContainerManager, sm ServiceManager, cfg *config.Manager) *WorktreeOperations {
	return &WorktreeOperations{
		cfg:          cfg,
		containerMgr: cm,
		gitMgr:       gm,
		db:           database,
		serviceMgr:   sm,
		logAggregator: NewLogAggregator(database, cm),
	}
}

// CreateWorktreeRequest contains all parameters for creating a worktree
type CreateWorktreeRequest struct {
	RepositoryID    string
	Name            string
	Branch          string
	BaseBranch      string
	SkipSetup       bool
	ContainerImage  string
	AutoStart       bool
	ComposeFile     string   // Override default compose file
	Services []string // Override default compose services
	PostScripts     []string // Additional setup scripts to run after worktree creation
}

// CreateWorktreeResponse contains the result of creating a worktree
type CreateWorktreeResponse struct {
	Worktree *db.Worktree
	Path     string
}

// CreateWorktree creates a new worktree with all associated resources
func (wo *WorktreeOperations) CreateWorktree(ctx context.Context, req CreateWorktreeRequest) (*CreateWorktreeResponse, error) {
	// Validate worktree name
	if err := validateWorktreeName(req.Name); err != nil {
		return nil, errors.Wrap(errors.ErrInvalidInput, "invalid worktree name", err)
	}

	// Get repository from database
	repoRepo := db.NewRepositoryRepository(wo.db)
	repo, err := repoRepo.GetByID(ctx, req.RepositoryID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrDatabaseQuery, "failed to get repository", err).WithContext("repository_id", req.RepositoryID)
	}

	// Load repository configuration
	repoConfig, err := config.ParseRepositoryConfig(repo.Path)
	if err != nil {
		return nil, errors.Wrap(errors.ErrConfigParse, "failed to load repository config", err).WithContext("path", repo.Path)
	}

	// Determine base branch
	baseBranch := req.BaseBranch
	if baseBranch == "" {
		if repoConfig.Repository.Git.DefaultBranch != "" {
			baseBranch = repoConfig.Repository.Git.DefaultBranch
		} else {
			baseBranch = "main"
		}
	}

	// Determine worktrees directory
	worktreesDir := repoConfig.Repository.Worktrees.Directory
	if worktreesDir == "" {
		globalCfg, _ := config.LoadGlobalConfig()
		if globalCfg != nil && globalCfg.Storage.WorktreesPath != "" {
			worktreesDir = globalCfg.Storage.WorktreesPath
		} else {
			homeDir, _ := os.UserHomeDir()
			worktreesDir = filepath.Join(homeDir, "vibeman", "worktrees")
		}
	}

	// Ensure worktrees directory is absolute and validate it
	if !filepath.IsAbs(worktreesDir) {
		worktreesDir = filepath.Join(filepath.Dir(repo.Path), worktreesDir)
	}
	
	// Validate the worktrees directory path
	cleanedWorktreesDir, err := validation.Path(worktreesDir)
	if err != nil {
		return nil, errors.Wrap(errors.ErrInvalidPath, "invalid worktrees directory", err)
	}
	worktreesDir = cleanedWorktreesDir

	// Validate worktree name doesn't contain path separators
	if strings.Contains(req.Name, "/") || strings.Contains(req.Name, "\\") {
		return nil, errors.New(errors.ErrInvalidInput, "worktree name cannot contain path separators")
	}

	worktreeDir := filepath.Join(worktreesDir, req.Name)
	
	// Validate the final worktree directory
	cleanedWorktreeDir, err := validation.Path(worktreeDir)
	if err != nil {
		return nil, errors.Wrap(errors.ErrInvalidPath, "invalid worktree directory", err)
	}
	worktreeDir = cleanedWorktreeDir

	// Check if worktree already exists
	if _, err := os.Stat(worktreeDir); err == nil {
		return nil, errors.New(errors.ErrInvalidState, "worktree directory already exists").WithContext("path", worktreeDir)
	}

	// Create branch name
	branchPrefix := ""
	if repoConfig.Repository.Git.WorktreePrefix != "" {
		branchPrefix = repoConfig.Repository.Git.WorktreePrefix
	}
	branchName := req.Branch
	if branchName == "" {
		if branchPrefix != "" {
			branchName = fmt.Sprintf("%s%s", branchPrefix, req.Name)
		} else {
			branchName = req.Name
		}
	}

	logger.WithFields(logger.Fields{
		"worktree":   req.Name,
		"repository": repo.Name,
		"branch":     branchName,
		"base":       baseBranch,
	}).Info("Creating worktree")

	// Create git worktree (this will create the directory)
	if err := wo.gitMgr.CreateWorktree(ctx, repo.Path, branchName, worktreeDir); err != nil {
		return nil, errors.Wrap(errors.ErrGitWorktreeFailed, "failed to create git worktree", err).WithContext("branch", branchName).WithContext("path", worktreeDir)
	}

	// Create database record
	worktree := &db.Worktree{
		ID:           generateID(),
		RepositoryID: req.RepositoryID,
		Name:         req.Name,
		Branch:       branchName,
		Path:         worktreeDir,
		Status:       db.StatusStopped,
	}

	worktreeRepo := db.NewWorktreeRepository(wo.db)
	if err := worktreeRepo.Create(ctx, worktree); err != nil {
		// Clean up on failure
		wo.gitMgr.RemoveWorktree(ctx, worktreeDir)
		os.RemoveAll(worktreeDir)
		return nil, errors.Wrap(errors.ErrDatabaseQuery, "failed to create worktree record", err)
	}

	// Create logs directory
	logsDir := xdg.LogsDir()
	worktreeLogsDir := filepath.Join(logsDir, repo.Name, req.Name)
	if err := os.MkdirAll(worktreeLogsDir, constants.DirPermissions); err != nil {
		logger.WithError(err).Warn("Failed to create logs directory")
	}

	// Create CLAUDE.md file
	claudePath := filepath.Join(worktreeDir, "CLAUDE.md")
	if err := createClaudeFile(claudePath, repo.Name, req.Name, worktreeDir); err != nil {
		logger.WithError(err).Warn("Failed to create CLAUDE.md")
	}

	// Copy vibeman.toml to worktree and update with overrides if specified
	srcConfig := filepath.Join(repo.Path, "vibeman.toml")
	dstConfig := filepath.Join(worktreeDir, "vibeman.toml")
	if err := copyFile(srcConfig, dstConfig); err != nil {
		logger.WithError(err).Warn("Failed to copy vibeman.toml")
	}
	
	// Apply overrides if specified
	if req.ComposeFile != "" || len(req.Services) > 0 {
		// Load the config from the worktree
		worktreeConfig, err := config.ParseRepositoryConfig(worktreeDir)
		if err == nil {
			// Apply overrides
			if req.ComposeFile != "" {
				worktreeConfig.Repository.Container.ComposeFile = req.ComposeFile
			}
			if len(req.Services) > 0 {
				worktreeConfig.Repository.Container.Services = req.Services
			}
			// Save the updated config
			if err := config.SaveRepositoryConfig(worktreeDir, worktreeConfig); err != nil {
				logger.WithError(err).Warn("Failed to save updated config with overrides")
			}
		}
	}

	// Start required services before running setup
	if !req.SkipSetup && len(repoConfig.Repository.Services) > 0 {
		logger.Info("Starting required services")
		for serviceName, serviceReq := range repoConfig.Repository.Services {
			if serviceReq.Required {
				logger.WithField("service", serviceName).Info("Starting required service")
				if err := wo.serviceMgr.StartService(ctx, serviceName); err != nil {
					logger.WithError(err).WithField("service", serviceName).Warn("Failed to start required service")
				}
			}
		}
	}

	// Run setup commands if not skipped
	if !req.SkipSetup {
		// Run repository setup command first
		if repoConfig.Repository.Setup.WorktreeInit != "" {
			logger.Info("Running worktree setup command")
			if err := runCommand(ctx, worktreeDir, repoConfig.Repository.Setup.WorktreeInit); err != nil {
				logger.WithError(err).WithField("command", repoConfig.Repository.Setup.WorktreeInit).Warn("Setup command failed")
			}
		}
		
		// Run additional post-scripts
		for _, script := range req.PostScripts {
			logger.WithField("script", script).Info("Running post-script")
			if err := runCommand(ctx, worktreeDir, script); err != nil {
				logger.WithError(err).WithField("script", script).Warn("Post-script failed")
			}
		}
	}

	// Auto-start container if requested
	if req.AutoStart {
		logger.Info("Auto-starting worktree container")
		if err := wo.StartWorktree(ctx, worktree.ID); err != nil {
			logger.WithError(err).Warn("Failed to auto-start worktree")
			// Don't fail the entire operation, just log the error
		}
	}

	return &CreateWorktreeResponse{
		Worktree: worktree,
		Path:     worktreeDir,
	}, nil
}

// RemoveWorktree removes a worktree and all associated resources
func (wo *WorktreeOperations) RemoveWorktree(ctx context.Context, worktreeID string, force bool) error {
	// Get worktree from database
	worktreeRepo := db.NewWorktreeRepository(wo.db)
	worktree, err := worktreeRepo.Get(ctx, worktreeID)
	if err != nil {
		return errors.Wrap(errors.ErrDatabaseQuery, "failed to get worktree", err).WithContext("worktree_id", worktreeID)
	}

	// Get repository
	repoRepo := db.NewRepositoryRepository(wo.db)
	repo, err := repoRepo.GetByID(ctx, worktree.RepositoryID)
	if err != nil {
		return errors.Wrap(errors.ErrDatabaseQuery, "failed to get repository", err).WithContext("repository_id", worktree.RepositoryID)
	}

	// Check if we're in the worktree being removed
	currentDir, _ := os.Getwd()
	if strings.HasPrefix(currentDir, worktree.Path) {
		return errors.New(errors.ErrInvalidState, "cannot remove current worktree, please change to a different directory first")
	}

	// Check for uncommitted changes if not forced
	if !force {
		hasUncommitted, err := wo.gitMgr.HasUncommittedChanges(ctx, worktree.Path)
		if err != nil {
			return errors.Wrap(errors.ErrGitWorktreeFailed, "failed to check uncommitted changes", err).WithContext("path", worktree.Path)
		}
		
		hasUnpushed, err := wo.gitMgr.HasUnpushedCommits(ctx, worktree.Path)
		if err != nil {
			return errors.Wrap(errors.ErrGitWorktreeFailed, "failed to check unpushed commits", err).WithContext("path", worktree.Path)
		}

		if hasUncommitted || hasUnpushed {
			if hasUncommitted {
				return errors.ErrWorktreeNotClean
			}
			return errors.ErrBranchHasUnpushedCommits
		}
	}

	logger.WithFields(logger.Fields{
		"worktree":   worktree.Name,
		"repository": repo.Name,
	}).Info("Removing worktree")

	// Remove AI container if it exists
	aiContainerName := fmt.Sprintf("%s-%s-ai", repo.Name, worktree.Name)
	if aiContainer, err := wo.containerMgr.GetByName(ctx, aiContainerName); err == nil {
		logger.WithFields(logger.Fields{
			"container_id":   aiContainer.ID,
			"container_name": aiContainerName,
		}).Info("Removing AI container")
		
		// Stop container first if running
		wo.containerMgr.Stop(ctx, aiContainer.ID)
		
		// Remove the container
		if err := wo.containerMgr.Remove(ctx, aiContainer.ID); err != nil {
			logger.WithError(err).Warn("Failed to remove AI container")
		}
	}

	// Remove git worktree
	if err := wo.gitMgr.RemoveWorktree(ctx, worktree.Path); err != nil {
		logger.WithError(err).Warn("Failed to remove git worktree")
	}

	// Remove worktree directory
	if err := os.RemoveAll(worktree.Path); err != nil {
		logger.WithError(err).Warn("Failed to remove worktree directory")
	}

	// Remove logs directory
	logsDir := xdg.LogsDir()
	worktreeLogsDir := filepath.Join(logsDir, repo.Name, worktree.Name)
	if err := os.RemoveAll(worktreeLogsDir); err != nil {
		logger.WithError(err).Warn("Failed to remove logs directory")
	}

	// Remove database record
	if err := worktreeRepo.Delete(ctx, worktreeID); err != nil {
		return errors.Wrap(errors.ErrDatabaseQuery, "failed to delete worktree record", err).WithContext("worktree_id", worktreeID)
	}

	return nil
}

// StartWorktree starts a worktree's container and services
func (wo *WorktreeOperations) StartWorktree(ctx context.Context, worktreeID string) error {
	// Get worktree from database
	worktreeRepo := db.NewWorktreeRepository(wo.db)
	worktree, err := worktreeRepo.Get(ctx, worktreeID)
	if err != nil {
		return errors.Wrap(errors.ErrDatabaseQuery, "failed to get worktree", err).WithContext("worktree_id", worktreeID)
	}

	// Check if already running
	if worktree.Status == db.StatusRunning {
		return errors.ErrServiceAlreadyRunningError
	}

	// Update status to starting
	if err := worktreeRepo.UpdateStatus(ctx, worktreeID, db.StatusStarting); err != nil {
		return errors.Wrap(errors.ErrDatabaseQuery, "failed to update worktree status", err).WithContext("worktree_id", worktreeID)
	}

	// Get repository for config
	repoRepo := db.NewRepositoryRepository(wo.db)
	repo, err := repoRepo.GetByID(ctx, worktree.RepositoryID)
	if err != nil {
		// Revert status on error
		worktreeRepo.UpdateStatus(ctx, worktreeID, db.StatusStopped)
		return errors.Wrap(errors.ErrDatabaseQuery, "failed to get repository", err).WithContext("repository_id", worktree.RepositoryID)
	}

	// Load repository configuration
	repoConfig, err := config.ParseRepositoryConfig(worktree.Path)
	if err != nil {
		// Revert status on error
		worktreeRepo.UpdateStatus(ctx, worktreeID, db.StatusStopped)
		return errors.Wrap(errors.ErrConfigParse, "failed to load repository config", err).WithContext("path", worktree.Path)
	}

	// Start AI container if enabled
	if repoConfig.Repository.AI.Enabled {
		logger.WithFields(logger.Fields{
			"worktree":   worktree.Name,
			"repository": repo.Name,
		}).Info("Starting AI container for worktree")

		// Determine AI image
		aiImage := repoConfig.Repository.AI.Image
		if aiImage == "" {
			aiImage = "vibeman/ai-assistant:latest"
		}

		// Build container name
		aiContainerName := fmt.Sprintf("%s-%s-ai", repo.Name, worktree.Name)

		// Prepare environment variables
		envVars := []string{
			fmt.Sprintf("VIBEMAN_WORKTREE_ID=%s", worktree.ID),
			fmt.Sprintf("VIBEMAN_REPOSITORY=%s", repo.Name),
			fmt.Sprintf("VIBEMAN_WORKTREE=%s", worktree.Name),
			fmt.Sprintf("VIBEMAN_WORKTREE_PATH=%s", worktree.Path),
			fmt.Sprintf("VIBEMAN_LOG_DIR=/logs"),
			fmt.Sprintf("VIBEMAN_ALL_LOGS_DIR=/all-logs"),
		}

		// Add service endpoints if any services are running
		if len(repoConfig.Repository.Services) > 0 {
			// TODO: Get actual service endpoints from service manager
			envVars = append(envVars, "VIBEMAN_SERVICES={}")
		}

		// Add custom environment variables
		for k, v := range repoConfig.Repository.AI.Env {
			envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
		}

		// Prepare volume mounts
		logsDir := xdg.LogsDir()
		worktreeLogsDir := filepath.Join(logsDir, repo.Name, worktree.Name)
		volumes := []string{
			fmt.Sprintf("%s:/workspace", worktree.Path),     // Worktree code directory
			fmt.Sprintf("%s:/logs:ro", worktreeLogsDir),     // Worktree-specific logs (read-only)
			fmt.Sprintf("%s:/all-logs:ro", logsDir),         // All logs directory (read-only)
		}

		// Add custom volumes
		for host, container := range repoConfig.Repository.AI.Volumes {
			volumes = append(volumes, fmt.Sprintf("%s:%s", host, container))
		}

		// Create AI container configuration
		createConfig := &container.CreateConfig{
			Name:        aiContainerName,
			Image:       aiImage,
			Repository:  repo.Name,
			Environment: worktree.Name,
			Type:        "ai",
			EnvVars:     envVars,
			Volumes:     volumes,
			WorkingDir:  "/workspace",
			Interactive: false, // Run detached, users can attach later
		}

		// Create the AI container
		aiContainer, err := wo.containerMgr.CreateWithConfig(ctx, createConfig)
		if err != nil {
			logger.WithError(err).Warn("Failed to create AI container")
			// Don't fail the entire operation if AI container fails
		} else {
			// Start the AI container
			if err := wo.containerMgr.Start(ctx, aiContainer.ID); err != nil {
				logger.WithError(err).Warn("Failed to start AI container")
				// Clean up the created container
				wo.containerMgr.Remove(ctx, aiContainer.ID)
			} else {
				logger.WithFields(logger.Fields{
					"container_id": aiContainer.ID,
					"container_name": aiContainerName,
				}).Info("AI container started successfully")
				
				// Start log aggregation for the AI container
				if err := wo.logAggregator.StartLogAggregation(ctx, worktree.ID); err != nil {
					logger.WithError(err).Warn("Failed to start log aggregation")
				}
				
				// Create aggregated logs view
				if err := wo.logAggregator.AggregateLogsForAIContainer(ctx, worktree.ID); err != nil {
					logger.WithError(err).Warn("Failed to create aggregated logs view")
				}
			}
		}
	}

	// TODO: Start associated services based on repository config

	// Update status to running
	if err := worktreeRepo.UpdateStatus(ctx, worktreeID, db.StatusRunning); err != nil {
		return errors.Wrap(errors.ErrDatabaseQuery, "failed to update worktree status", err).WithContext("worktree_id", worktreeID)
	}

	return nil
}

// StopWorktree stops a worktree's container and services
func (wo *WorktreeOperations) StopWorktree(ctx context.Context, worktreeID string) error {
	// Get worktree from database
	worktreeRepo := db.NewWorktreeRepository(wo.db)
	worktree, err := worktreeRepo.Get(ctx, worktreeID)
	if err != nil {
		return errors.Wrap(errors.ErrDatabaseQuery, "failed to get worktree", err).WithContext("worktree_id", worktreeID)
	}

	// Check if already stopped
	if worktree.Status == db.StatusStopped {
		return errors.New(errors.ErrInvalidState, "worktree is already stopped")
	}

	// Update status to stopping
	if err := worktreeRepo.UpdateStatus(ctx, worktreeID, db.StatusStopping); err != nil {
		return errors.Wrap(errors.ErrDatabaseQuery, "failed to update worktree status", err).WithContext("worktree_id", worktreeID)
	}

	// Get repository for config
	repoRepo := db.NewRepositoryRepository(wo.db)
	repo, err := repoRepo.GetByID(ctx, worktree.RepositoryID)
	if err != nil {
		// Log error but continue with status update
		logger.WithError(err).WithFields(logger.Fields{
			"repository_id": worktree.RepositoryID,
		}).Warn("Failed to get repository")
	} else {
		// Stop AI container if it exists
		aiContainerName := fmt.Sprintf("%s-%s-ai", repo.Name, worktree.Name)
		if aiContainer, err := wo.containerMgr.GetByName(ctx, aiContainerName); err == nil {
			logger.WithFields(logger.Fields{
				"container_id":   aiContainer.ID,
				"container_name": aiContainerName,
			}).Info("Stopping AI container")
			
			// Stop log aggregation first
			wo.logAggregator.StopLogAggregation(worktree.ID)
			
			if err := wo.containerMgr.Stop(ctx, aiContainer.ID); err != nil {
				logger.WithError(err).Warn("Failed to stop AI container")
			}
		}
	}

	// TODO: Stop associated services based on repository config

	// Update status to stopped
	if err := worktreeRepo.UpdateStatus(ctx, worktreeID, db.StatusStopped); err != nil {
		return errors.Wrap(errors.ErrDatabaseQuery, "failed to update worktree status", err).WithContext("worktree_id", worktreeID)
	}

	return nil
}

// Helper functions

func validateWorktreeName(name string) error {
	if name == "" {
		return errors.ErrEmptyInput
	}
	if strings.Contains(name, " ") {
		return errors.New(errors.ErrInvalidInput, "name cannot contain spaces")
	}
	if strings.Contains(name, "/") {
		return errors.New(errors.ErrInvalidInput, "name cannot contain slashes")
	}
	return nil
}

func generateID() string {
	// Use proper UUID generation
	return uuid.New().String()
}

func createClaudeFile(path, repoName, worktreeName, worktreeDir string) error {
	content := fmt.Sprintf(`# %s - %s Worktree

This is a development worktree for the %s repository.

## Worktree Information
- Repository: %s
- Worktree Name: %s
- Directory: %s

## Getting Started

1. Your development environment is containerized
2. All changes should be made within this worktree
3. Use git commands as normal within this directory

## Container Commands

Start container:
vibeman start

Stop container:
vibeman stop

View logs:
vibeman logs

## Notes

This file was automatically generated by Vibeman.
`, repoName, worktreeName, repoName, repoName, worktreeName, worktreeDir)

	return os.WriteFile(path, []byte(content), 0644)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func runCommand(ctx context.Context, dir, command string) error {
	logger.WithFields(logger.Fields{
		"directory": dir,
		"command":   command,
	}).Info("Running command")

	// Validate directory path
	cleanedDir, err := validation.Path(dir)
	if err != nil {
		return errors.Wrap(errors.ErrInvalidPath, "invalid directory", err)
	}
	dir = cleanedDir

	// For security, we restrict commands to a safe set of operations
	// These are the commands used in setup scripts
	allowedCommands := map[string]bool{
		"npm":     true,
		"yarn":    true,
		"pnpm":    true,
		"go":      true,
		"make":    true,
		"docker":  true,
		"git":     true,
		"echo":    true,
		"mkdir":   true,
		"cp":      true,
		"mv":      true,
		"chmod":   true,
		"chown":   true,
	}

	// Parse the command to extract the base command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return errors.New(errors.ErrInvalidInput, "empty command")
	}

	baseCmd := parts[0]
	
	// Check if it's a path to a script (starts with ./ or /)
	if strings.HasPrefix(baseCmd, "./") || strings.HasPrefix(baseCmd, "/") {
		// For scripts, ensure they're within the worktree directory
		scriptPath := filepath.Join(dir, baseCmd)
		cleanedScriptPath, err := validation.Path(scriptPath)
		if err != nil {
			return errors.Wrap(errors.ErrInvalidPath, "script path outside worktree", err)
		}
		scriptPath = cleanedScriptPath
		if !strings.HasPrefix(filepath.Clean(scriptPath), filepath.Clean(dir)) {
			return errors.New(errors.ErrInvalidPath, "script must be within worktree directory")
		}
	} else if !allowedCommands[baseCmd] {
		// Check if command is in allowed list
		return errors.New(errors.ErrInvalidInput, fmt.Sprintf("command '%s' is not allowed", baseCmd))
	}

	// Use exec.Command with explicit arguments instead of shell
	// This prevents shell injection but still allows pipes and redirects in a controlled way
	var cmd *exec.Cmd
	
	// Check if command contains shell operators that we need to handle
	if strings.ContainsAny(command, "|>&<;") {
		// For complex commands with pipes/redirects, use sh but with strict validation
		// Only allow if the base command is in our allowed list
		if !allowedCommands[baseCmd] {
			return errors.New(errors.ErrInvalidInput, "complex shell operations only allowed with approved commands")
		}
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	} else {
		// For simple commands, execute directly without shell
		cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
	}
	
	cmd.Dir = dir
	
	// Set a reasonable timeout for commands
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
		cmd.Dir = dir
	}
	
	// Capture output for logging
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"command": command,
			"output":  string(output),
		}).Error("Command failed")
		return fmt.Errorf("command failed: %s: %w", string(output), err)
	}
	
	logger.WithFields(logger.Fields{
		"command": command,
		"output":  string(output),
	}).Debug("Command completed successfully")
	
	return nil
}