package operations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vibeman/internal/config"
	"vibeman/internal/constants"
	"vibeman/internal/db"
	"vibeman/internal/errors"
	"vibeman/internal/logger"
	"vibeman/internal/validation"

	"github.com/google/uuid"
)

// RepositoryOperations provides shared backend functions for repository management
type RepositoryOperations struct {
	cfg    *config.Manager
	gitMgr GitManager
	db     *db.DB
}

// NewRepositoryOperations creates a new RepositoryOperations instance
func NewRepositoryOperations(cfg *config.Manager, gm GitManager, database *db.DB) *RepositoryOperations {
	return &RepositoryOperations{
		cfg:    cfg,
		gitMgr: gm,
		db:     database,
	}
}

// AddRepositoryRequest contains parameters for adding a repository
type AddRepositoryRequest struct {
	Path        string // Can be local path or git URL
	Name        string // Optional, will be detected if not provided
	Description string // Optional description
}

// AddRepository adds a new repository to tracking
func (ro *RepositoryOperations) AddRepository(ctx context.Context, req AddRepositoryRequest) (*db.Repository, error) {
	var repoPath string
	var isClone bool

	// Determine if it's a URL or local path
	if strings.HasPrefix(req.Path, "http://") || strings.HasPrefix(req.Path, "https://") || strings.HasPrefix(req.Path, "git@") || strings.Contains(req.Path, ":") {
		// It's a URL, we need to clone it
		isClone = true
		
		// Get default repos directory
		globalCfg, err := config.LoadGlobalConfig()
		if err != nil {
			return nil, errors.Wrap(errors.ErrConfigNotFound, "failed to load global config", err)
		}
		
		reposDir := globalCfg.Storage.RepositoriesPath
		if reposDir == "" {
			homeDir, _ := os.UserHomeDir()
			reposDir = filepath.Join(homeDir, "vibeman", "repos")
		}
		
		// Extract repo name from URL
		repoName := extractRepoNameFromURL(req.Path)
		if req.Name != "" {
			repoName = req.Name
		}
		
		repoPath = filepath.Join(reposDir, repoName)
		
		// Check if already exists
		if _, err := os.Stat(repoPath); err == nil {
			return nil, errors.New(errors.ErrConfigValidation, "repository already exists").WithContext("path", repoPath)
		}
		
		// Clone the repository
		logger.WithFields(logger.Fields{
			"url":  req.Path,
			"path": repoPath,
		}).Info("Cloning repository")
		
		if err := ro.gitMgr.CloneRepository(ctx, req.Path, repoPath); err != nil {
			return nil, errors.Wrap(errors.ErrGitCloneFailed, "failed to clone repository", err).WithContext("url", req.Path).WithContext("path", repoPath)
		}
	} else {
		// It's a local path - validate it first
		cleanedPath, err := validation.Path(req.Path)
		if err != nil {
			return nil, errors.Wrap(errors.ErrInvalidPath, "invalid repository path", err)
		}
		repoPath = cleanedPath
		
		// Make absolute if relative
		if !filepath.IsAbs(repoPath) {
			cwd, err := os.Getwd()
			if err != nil {
				return nil, errors.Wrap(errors.ErrFileSystem, "failed to get current directory", err)
			}
			repoPath = filepath.Join(cwd, repoPath)
			// Validate the absolute path again
			repoPath, err = validation.Path(repoPath)
			if err != nil {
				return nil, errors.Wrap(errors.ErrInvalidPath, "invalid absolute repository path", err)
			}
		}
		
		// Verify it's a git repository
		if !ro.gitMgr.IsRepository(repoPath) {
			return nil, errors.New(errors.ErrGitRepoNotFound, "not a valid git repository").WithContext("path", repoPath)
		}
	}

	// Check if vibeman.toml exists
	configPath := filepath.Join(repoPath, "vibeman.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default vibeman.toml
		if err := createDefaultVibemanConfig(repoPath, req.Name); err != nil {
			logger.WithError(err).Warn("Failed to create default vibeman.toml")
		}
	}

	// Parse repository configuration
	repoConfig, err := config.ParseRepositoryConfig(repoPath)
	if err != nil {
		return nil, errors.Wrap(errors.ErrConfigParse, "failed to parse repository config", err).WithContext("path", repoPath)
	}

	// Create database record
	repo := &db.Repository{
		ID:          uuid.New().String(),
		Path:        repoPath,
		Name:        repoConfig.Repository.Name,
		Description: repoConfig.Repository.Description,
	}

	if repo.Name == "" {
		repo.Name = filepath.Base(repoPath)
	}

	repoRepo := db.NewRepositoryRepository(ro.db)
	if err := repoRepo.Create(ctx, repo); err != nil {
		// If we cloned it, clean up
		if isClone {
			os.RemoveAll(repoPath)
		}
		return nil, errors.Wrap(errors.ErrDatabaseQuery, "failed to create repository record", err)
	}

	logger.WithFields(logger.Fields{
		"name": repo.Name,
		"path": repo.Path,
	}).Info("Repository added successfully")

	return repo, nil
}

// RemoveRepository removes a repository from tracking (does not delete files)
func (ro *RepositoryOperations) RemoveRepository(ctx context.Context, repositoryID string) error {
	// Get repository
	repoRepo := db.NewRepositoryRepository(ro.db)
	repo, err := repoRepo.GetByID(ctx, repositoryID)
	if err != nil {
		return errors.Wrap(errors.ErrDatabaseQuery, "failed to get repository", err).WithContext("repository_id", repositoryID)
	}

	// Check for active worktrees
	worktreeRepo := db.NewWorktreeRepository(ro.db)
	worktrees, err := worktreeRepo.List(ctx, repositoryID, "")
	if err != nil {
		return errors.Wrap(errors.ErrDatabaseQuery, "failed to check worktrees", err).WithContext("repository_id", repositoryID)
	}

	if len(worktrees) > 0 {
		return errors.New(errors.ErrInvalidState, "repository has active worktrees, remove them first").WithContext("worktree_count", len(worktrees))
	}

	// Remove from database
	if err := repoRepo.Delete(ctx, repositoryID); err != nil {
		return errors.Wrap(errors.ErrDatabaseQuery, "failed to remove repository", err).WithContext("repository_id", repositoryID)
	}

	logger.WithFields(logger.Fields{
		"name": repo.Name,
		"path": repo.Path,
	}).Info("Repository removed from tracking (files not deleted)")

	return nil
}

// ListRepositories lists all tracked repositories with their worktree counts
func (ro *RepositoryOperations) ListRepositories(ctx context.Context) ([]*db.Repository, error) {
	repoRepo := db.NewRepositoryRepository(ro.db)
	repositories, err := repoRepo.List(ctx)
	if err != nil {
		return nil, errors.Wrap(errors.ErrDatabaseQuery, "failed to list repositories", err)
	}

	// Add worktree counts
	worktreeRepo := db.NewWorktreeRepository(ro.db)
	for _, repo := range repositories {
		worktrees, err := worktreeRepo.List(ctx, repo.ID, "")
		if err != nil {
			logger.WithError(err).Warn("Failed to get worktree count")
			continue
		}
		// Note: WorktreeCount field has been removed from Repository model
		_ = worktrees // Suppress unused variable warning
	}

	return repositories, nil
}

// Helper functions

func extractRepoNameFromURL(url string) string {
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")
	
	// Get the last part of the URL
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	
	// Fallback for other formats
	parts = strings.Split(url, ":")
	if len(parts) > 1 {
		return filepath.Base(parts[1])
	}
	
	return "repository"
}

func createDefaultVibemanConfig(repoPath, name string) error {
	if name == "" {
		name = filepath.Base(repoPath)
	}

	content := fmt.Sprintf(`[repository]
name = "%s"
description = ""

[repository.container]
# Uncomment and configure as needed
# image = "vibeman-dev:latest"
# compose_file = "./docker-compose.yaml"
# compose_services = ["dev"]

[repository.worktrees]
# Directory for worktrees (relative to repo or absolute)
directory = "../%s-worktrees"

[repository.git]
default_branch = "main"
# worktree_prefix = "feature/"

[repository.runtime]
type = "docker"
`, name, name)

	configPath := filepath.Join(repoPath, "vibeman.toml")
	return os.WriteFile(configPath, []byte(content), constants.FilePermissions)
}