package commands

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"vibeman/internal/config"
	"vibeman/internal/db"
	"vibeman/internal/logger"
	"vibeman/internal/operations"

	"github.com/spf13/cobra"
)

// RepoCommands creates repository management commands
func RepoCommands(cfg *config.Manager, repoManager db.RepositoryManager, gm GitManager, cm ContainerManager, sm ServiceManager, database *db.DB) []*cobra.Command {
	commands := []*cobra.Command{}

	// Create operations instance
	var repoOps *operations.RepositoryOperations
	if database != nil && gm != nil {
		repoOps = operations.NewRepositoryOperations(cfg, gm, database)
	}

	// vibeman repo add <path-or-url>
	addCmd := &cobra.Command{
		Use:   "add <path-or-url>",
		Short: "Add a repository to Vibeman",
		Long: `Add a repository to Vibeman. Accepts either:
  - A local path to an existing Git repository
  - A Git URL (SSH or HTTPS) to clone`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pathOrURL := args[0]
			name, _ := cmd.Flags().GetString("name")
			description, _ := cmd.Flags().GetString("description")
			
			if repoOps == nil {
				return fmt.Errorf("operations not initialized")
			}
			req := operations.AddRepositoryRequest{
				Path:        pathOrURL,
				Name:        name,
				Description: description,
			}
			_, err := repoOps.AddRepository(cmd.Context(), req)
			return err
		},
	}
	addCmd.Flags().StringP("name", "n", "", "Repository name (defaults to directory name)")
	addCmd.Flags().StringP("description", "d", "", "Repository description")
	commands = append(commands, addCmd)

	// vibeman repo list
	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List all known repositories",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoOps == nil {
				return fmt.Errorf("operations not initialized")
			}
			verbose, _ := cmd.Flags().GetBool("verbose")
			return listRepositoriesWithOps(cmd.Context(), verbose, repoOps)
		},
	}
	listCmd.Flags().BoolP("verbose", "v", false, "Show detailed information")
	commands = append(commands, listCmd)

	// vibeman repo remove <name>
	removeCmd := &cobra.Command{
		Use:     "remove <name>",
		Short:   "Remove a repository from Vibeman tracking",
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if repoOps == nil {
				return fmt.Errorf("operations not initialized")
			}
			// Note: RemoveRepository in operations doesn't have a force parameter
			// The operation handles the prompting internally
			return repoOps.RemoveRepository(cmd.Context(), name)
		},
	}
	removeCmd.Flags().BoolP("force", "f", false, "Force removal without confirmation")
	commands = append(commands, removeCmd)

	return commands
}

func listRepositoriesWithOps(ctx context.Context, verbose bool, repoOps *operations.RepositoryOperations) error {
	repos, err := repoOps.ListRepositories(ctx)
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}
	
	if len(repos) == 0 {
		logger.Info("No repositories found")
		return nil
	}
	
	if verbose {
		// Detailed output
		for i, repo := range repos {
			if i > 0 {
				fmt.Println()
			}
			
			fmt.Printf("Repository: %s\n", repo.Name)
			fmt.Printf("Path: %s\n", repo.Path)
			if repo.Description != "" {
				fmt.Printf("Description: %s\n", repo.Description)
			}
		}
	} else {
		// Table output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPATH\tDESCRIPTION\tWORKTREES")
		
		for _, repo := range repos {
			description := repo.Description
			if description == "" {
				description = "-"
			}
			
			// TODO: Get worktree count from database
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", 
				repo.Name, 
				repo.Path, 
				description,
				0, // Placeholder for worktree count
			)
		}
		
		w.Flush()
	}
	
	return nil
}

func addRepository(ctx context.Context, pathOrURL, name, description string, cfg *config.Manager, repoManager db.RepositoryManager, gm GitManager) error {
	var repoPath string
	
	// Check if it's a URL
	if isGitURL(pathOrURL) {
		// Clone the repository
		logger.WithFields(logger.Fields{"url": pathOrURL}).Info("Cloning repository")
		
		// Get default repos directory from global config
		globalConfig, err := config.LoadGlobalConfig()
		if err != nil {
			return fmt.Errorf("failed to load global config: %w", err)
		}
		
		reposDir := globalConfig.Storage.RepositoriesPath
		if reposDir == "" {
			reposDir = filepath.Join(os.Getenv("HOME"), "vibeman", "repos")
		}
		
		// Expand ~ in path
		if strings.HasPrefix(reposDir, "~/") {
			reposDir = filepath.Join(os.Getenv("HOME"), reposDir[2:])
		}
		
		// Create repos directory if it doesn't exist
		if err := os.MkdirAll(reposDir, 0755); err != nil {
			return fmt.Errorf("failed to create repos directory: %w", err)
		}
		
		// Extract repo name from URL
		repoName := extractRepoNameFromURL(pathOrURL)
		if name != "" {
			repoName = name
		}
		
		repoPath = filepath.Join(reposDir, repoName)
		
		// Check if directory already exists
		if _, err := os.Stat(repoPath); err == nil {
			return fmt.Errorf("directory %s already exists", repoPath)
		}
		
		// Clone the repository
		if err := gm.CloneRepository(ctx, pathOrURL, repoPath); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	} else {
		// Local path
		absPath, err := filepath.Abs(pathOrURL)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}
		
		// Verify it's a git repository
		if !gm.IsRepository(absPath) {
			return fmt.Errorf("not a git repository: %s", absPath)
		}
		
		repoPath = absPath
	}
	
	// Load or create vibeman.toml
	vibemanPath := filepath.Join(repoPath, "vibeman.toml")
	var repoConfig *config.RepositoryConfig
	
	if _, err := os.Stat(vibemanPath); os.IsNotExist(err) {
		// Create default config
		logger.Info("Creating default vibeman.toml")
		
		// Get repository name
		if name == "" {
			name = filepath.Base(repoPath)
		}
		
		// Create minimal config
		if err := createMinimalConfig(vibemanPath, name, description); err != nil {
			return fmt.Errorf("failed to create vibeman.toml: %w", err)
		}
		
		// Load the created config
		repoConfig = &config.RepositoryConfig{}
		data, _ := os.ReadFile(vibemanPath)
		if err := config.ParseRepositoryConfigData(data, repoConfig); err != nil {
			return fmt.Errorf("failed to parse created config: %w", err)
		}
	} else {
		// Load existing config
		repoConfig = &config.RepositoryConfig{}
		data, err := os.ReadFile(vibemanPath)
		if err != nil {
			return fmt.Errorf("failed to read vibeman.toml: %w", err)
		}
		
		if err := config.ParseRepositoryConfigData(data, repoConfig); err != nil {
			return fmt.Errorf("failed to parse vibeman.toml: %w", err)
		}
		
		// Override name if provided
		if name != "" {
			repoConfig.Repository.Name = name
		}
		if description != "" {
			repoConfig.Repository.Description = description
		}
	}
	
	// Add to repository database
	repo := &db.Repository{
		Name:        repoConfig.Repository.Name,
		Path:        repoPath,
		Description: repoConfig.Repository.Description,
	}
	
	if err := repoManager.CreateRepository(ctx, repo); err != nil {
		return fmt.Errorf("failed to add repository to database: %w", err)
	}
	
	logger.WithFields(logger.Fields{
		"name": repo.Name,
		"path": repo.Path,
	}).Info("✓ Repository added successfully")
	
	return nil
}

func listRepositories(ctx context.Context, verbose bool, repoManager db.RepositoryManager, gm GitManager) error {
	repos, err := repoManager.ListRepositories(ctx)
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}
	
	if len(repos) == 0 {
		logger.Info("No repositories found")
		return nil
	}
	
	if verbose {
		// Detailed output
		for i, repo := range repos {
			if i > 0 {
				fmt.Println()
			}
			
			fmt.Printf("Repository: %s\n", repo.Name)
			fmt.Printf("Path: %s\n", repo.Path)
			if repo.Description != "" {
				fmt.Printf("Description: %s\n", repo.Description)
			}
			
			// Get worktrees
			worktrees, err := repoManager.GetWorktreesByRepository(ctx, repo.ID)
			if err == nil && len(worktrees) > 0 {
				fmt.Printf("Worktrees:\n")
				for _, wt := range worktrees {
					fmt.Printf("  - %s (branch: %s)\n", wt.Name, wt.Branch)
				}
			}
		}
	} else {
		// Table output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPATH\tDESCRIPTION\tWORKTREES")
		
		for _, repo := range repos {
			// Get worktree count
			worktrees, err := repoManager.GetWorktreesByRepository(ctx, repo.ID)
			worktreeCount := 0
			if err == nil {
				worktreeCount = len(worktrees)
			}
			
			description := repo.Description
			if description == "" {
				description = "-"
			}
			
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", 
				repo.Name, 
				repo.Path, 
				description,
				worktreeCount,
			)
		}
		
		w.Flush()
	}
	
	return nil
}

func removeRepository(ctx context.Context, name string, force bool, repoManager db.RepositoryManager) error {
	// Find repository
	repo, err := repoManager.GetRepositoryByName(ctx, name)
	if err != nil {
		return fmt.Errorf("repository not found: %s", name)
	}
	
	// Check for active worktrees
	worktrees, err := repoManager.GetWorktreesByRepository(ctx, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to check worktrees: %w", err)
	}
	
	if len(worktrees) > 0 && !force {
		logger.WithFields(logger.Fields{
			"repository": name,
			"worktrees":  len(worktrees),
		}).Warn("Repository has active worktrees")
		
		fmt.Printf("Repository '%s' has %d active worktree(s).\n", name, len(worktrees))
		fmt.Printf("This will only remove the repository from Vibeman tracking.\n")
		fmt.Printf("No files will be deleted.\n\n")
		fmt.Printf("Continue? [y/N]: ")
		
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			return fmt.Errorf("removal cancelled")
		}
	}
	
	// Remove from database
	if err := repoManager.DeleteRepository(ctx, repo.ID); err != nil {
		return fmt.Errorf("failed to remove repository: %w", err)
	}
	
	logger.WithFields(logger.Fields{
		"name": name,
		"path": repo.Path,
	}).Info("✓ Repository removed from Vibeman tracking")
	logger.Info("Note: Repository files remain at " + repo.Path)
	
	return nil
}

// Helper functions

func isGitURL(str string) bool {
	// Check for SSH URLs (git@github.com:user/repo.git)
	if strings.HasPrefix(str, "git@") {
		return true
	}
	
	// Check for HTTP(S) URLs
	u, err := url.Parse(str)
	if err != nil {
		return false
	}
	
	return u.Scheme == "http" || u.Scheme == "https"
}

func extractRepoNameFromURL(gitURL string) string {
	// Handle SSH URLs
	if strings.HasPrefix(gitURL, "git@") {
		parts := strings.Split(gitURL, ":")
		if len(parts) >= 2 {
			path := parts[1]
			return strings.TrimSuffix(filepath.Base(path), ".git")
		}
	}
	
	// Handle HTTP(S) URLs
	u, err := url.Parse(gitURL)
	if err == nil {
		path := strings.TrimSuffix(u.Path, ".git")
		return filepath.Base(path)
	}
	
	// Fallback
	return "repository"
}

func createMinimalConfig(path, name, description string) error {
	if description == "" {
		description = fmt.Sprintf("%s repository", name)
	}
	
	content := fmt.Sprintf(`# Vibeman Repository Configuration

[repository]
name = "%s"
description = "%s"

[repository.container]
# TODO: Add your docker-compose configuration
# compose_file = "./docker-compose.yaml"
# compose_services = ["dev"]

[repository.worktrees]
directory = "../%s-worktrees"

[repository.runtime]
type = "docker"
`, name, description, name)
	
	return os.WriteFile(path, []byte(content), 0644)
}