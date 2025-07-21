package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"vibeman/internal/config"
	"vibeman/internal/db"
	"vibeman/internal/interfaces"
	"vibeman/internal/logger"
	"vibeman/internal/types"

	"github.com/spf13/cobra"
)

// CreateAICommand creates the ai command and its subcommands
func CreateAICommand(cfg *config.Manager, containerMgr interfaces.ContainerManager, gitMgr interfaces.GitManager, serviceMgr ServiceManager, database *db.DB) *cobra.Command {
	// Create a simple worktree getter for AI commands
	getWorktrees := func(ctx context.Context) ([]*db.Worktree, error) {
		repo := db.NewWorktreeRepository(database)
		worktrees, err := repo.List(ctx, "", "") // Get all worktrees
		if err != nil {
			return nil, err
		}
		// Convert to pointer slice
		result := make([]*db.Worktree, len(worktrees))
		for i := range worktrees {
			result[i] = &worktrees[i]
		}
		return result, nil
	}

	cmd := &cobra.Command{
		Use:   "ai [worktree]",
		Short: "Start Claude CLI in AI container",
		Long: `Start Claude CLI in the AI container for the current or specified worktree.

Examples:
  vibeman ai                    # Start Claude in current worktree's AI container
  vibeman ai my-feature         # Start Claude in 'my-feature' worktree's AI container

Subcommands:
  attach    Attach to AI container shell
  list      List all AI containers  
  logs      Show AI container logs
  claude    Start Claude CLI (same as default behavior)`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default behavior: start Claude CLI in current worktree's AI container
			return startClaudeInAIContainer(cmd.Context(), containerMgr, getWorktrees, args)
		},
	}

	// Add subcommands
	cmd.AddCommand(createAIAttachCommand(containerMgr, getWorktrees))
	cmd.AddCommand(createAIClaudeCommand(containerMgr, getWorktrees))
	cmd.AddCommand(createAIListCommand(containerMgr))
	cmd.AddCommand(createAILogsCommand(containerMgr, getWorktrees))

	return cmd
}

// createAIAttachCommand creates the 'ai attach' command
func createAIAttachCommand(containerMgr interfaces.ContainerManager, getWorktrees func(context.Context) ([]*db.Worktree, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attach [worktree-name]",
		Short: "Attach to an AI container",
		Long:  "Attach to the AI container associated with a worktree",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Get current worktree if not specified
			worktreeName := ""
			if len(args) > 0 {
				worktreeName = args[0]
			} else {
				// Try to detect current worktree
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				// List all worktrees and check if we're in one
				worktrees, err := getWorktrees(ctx)
				if err != nil {
					return fmt.Errorf("failed to list worktrees: %w", err)
				}

				for _, wt := range worktrees {
					if strings.HasPrefix(cwd, wt.Path) {
						worktreeName = wt.Name
						break
					}
				}

				if worktreeName == "" {
					return fmt.Errorf("not in a worktree directory, please specify worktree name")
				}
			}

			// Find AI container for this worktree
			containers, err := containerMgr.List(ctx)
			if err != nil {
				return fmt.Errorf("failed to list containers: %w", err)
			}

			var aiContainerName string
			for _, c := range containers {
				if c.Type == "ai" && strings.Contains(c.Name, worktreeName) && c.Status == "running" {
					aiContainerName = c.Name
					break
				}
			}

			if aiContainerName == "" {
				return fmt.Errorf("no running AI container found for worktree: %s", worktreeName)
			}

			// Attach to the container
			logger.WithFields(logger.Fields{
				"container": aiContainerName,
				"worktree":  worktreeName,
			}).Info("Attaching to AI container")

			// Use docker attach command
			attachCmd := exec.Command("docker", "exec", "-it", aiContainerName, "/bin/zsh")
			attachCmd.Stdin = os.Stdin
			attachCmd.Stdout = os.Stdout
			attachCmd.Stderr = os.Stderr

			return attachCmd.Run()
		},
	}

	return cmd
}

// createAIClaudeCommand creates the 'ai claude' command
func createAIClaudeCommand(containerMgr interfaces.ContainerManager, getWorktrees func(context.Context) ([]*db.Worktree, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claude [worktree-name]",
		Short: "Start Claude CLI in an AI container",
		Long:  "Start Claude CLI in the AI container associated with a worktree",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Get current worktree if not specified
			worktreeName := ""
			if len(args) > 0 {
				worktreeName = args[0]
			} else {
				// Try to detect current worktree
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				// List all worktrees and check if we're in one
				worktrees, err := getWorktrees(ctx)
				if err != nil {
					return fmt.Errorf("failed to list worktrees: %w", err)
				}

				for _, wt := range worktrees {
					if strings.HasPrefix(cwd, wt.Path) {
						worktreeName = wt.Name
						break
					}
				}

				if worktreeName == "" {
					return fmt.Errorf("not in a worktree directory, please specify worktree name")
				}
			}

			// Find AI container for this worktree
			containers, err := containerMgr.List(ctx)
			if err != nil {
				return fmt.Errorf("failed to list containers: %w", err)
			}

			var aiContainerName string
			for _, c := range containers {
				if c.Type == "ai" && strings.Contains(c.Name, worktreeName) && c.Status == "running" {
					aiContainerName = c.Name
					break
				}
			}

			if aiContainerName == "" {
				return fmt.Errorf("no running AI container found for worktree: %s", worktreeName)
			}

			// Start Claude CLI in the container
			logger.WithFields(logger.Fields{
				"container": aiContainerName,
				"worktree":  worktreeName,
			}).Info("Starting Claude CLI in AI container")

			// Use docker exec to run claude
			claudeCmd := exec.Command("docker", "exec", "-it", aiContainerName, "claude", "--dangerously-skip-permissions")
			claudeCmd.Stdin = os.Stdin
			claudeCmd.Stdout = os.Stdout
			claudeCmd.Stderr = os.Stderr

			return claudeCmd.Run()
		},
	}

	return cmd
}

// createAIListCommand creates the 'ai list' command
func createAIListCommand(containerMgr interfaces.ContainerManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List AI containers",
		Long:  "List all AI containers and their status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// List all containers
			containers, err := containerMgr.List(ctx)
			if err != nil {
				return fmt.Errorf("failed to list containers: %w", err)
			}

			// Filter for AI containers
			var aiContainers []*types.Container
			for _, c := range containers {
				if c.Type == "ai" {
					aiContainers = append(aiContainers, c)
				}
			}

			if len(aiContainers) == 0 {
				fmt.Println("No AI containers found")
				return nil
			}

			// Print header
			fmt.Printf("%-20s %-30s %-15s %-20s %-15s\n", "CONTAINER ID", "NAME", "STATUS", "IMAGE", "WORKTREE")
			fmt.Println(strings.Repeat("-", 100))

			// Print containers
			for _, c := range aiContainers {
				// Extract worktree name from container name
				worktreeName := extractWorktreeFromAIContainer(c.Name)
				fmt.Printf("%-20s %-30s %-15s %-20s %-15s\n", 
					truncateString(c.ID, 20), 
					truncateString(c.Name, 30), 
					c.Status, 
					truncateString(c.Image, 20),
					worktreeName)
			}

			return nil
		},
	}

	return cmd
}

// createAILogsCommand creates the 'ai logs' command
func createAILogsCommand(containerMgr interfaces.ContainerManager, getWorktrees func(context.Context) ([]*db.Worktree, error)) *cobra.Command {
	var follow bool
	
	cmd := &cobra.Command{
		Use:   "logs [worktree-name]",
		Short: "Show AI container logs",
		Long:  "Show logs from the AI container associated with a worktree",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Get worktree name
			worktreeName := ""
			if len(args) > 0 {
				worktreeName = args[0]
			} else {
				// Try to detect current worktree
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}

				// List all worktrees and check if we're in one
				worktrees, err := getWorktrees(ctx)
				if err != nil {
					return fmt.Errorf("failed to list worktrees: %w", err)
				}

				for _, wt := range worktrees {
					if strings.HasPrefix(cwd, wt.Path) {
						worktreeName = wt.Name
						break
					}
				}

				if worktreeName == "" {
					return fmt.Errorf("not in a worktree directory, please specify worktree name")
				}
			}

			// Find AI container for this worktree
			containers, err := containerMgr.List(ctx)
			if err != nil {
				return fmt.Errorf("failed to list containers: %w", err)
			}

			var aiContainerID string
			for _, c := range containers {
				if c.Type == "ai" && strings.Contains(c.Name, worktreeName) {
					aiContainerID = c.ID
					break
				}
			}

			if aiContainerID == "" {
				return fmt.Errorf("no AI container found for worktree: %s", worktreeName)
			}

			// Get logs
			logs, err := containerMgr.Logs(ctx, aiContainerID, follow)
			if err != nil {
				return fmt.Errorf("failed to get container logs: %w", err)
			}

			fmt.Print(string(logs))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	
	return cmd
}

// Helper functions

// extractWorktreeFromAIContainer extracts worktree name from AI container name
func extractWorktreeFromAIContainer(containerName string) string {
	// AI containers are named like: "repo-worktree-ai"
	parts := strings.Split(containerName, "-")
	if len(parts) >= 3 && parts[len(parts)-1] == "ai" {
		// Return the middle parts (worktree name)
		return strings.Join(parts[1:len(parts)-1], "-")
	}
	return "unknown"
}

// truncateString truncates a string to the specified length
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	if length <= 3 {
		return s[:length]
	}
	return s[:length-3] + "..."
}

// startClaudeInAIContainer starts Claude CLI in the AI container for the current or specified worktree
func startClaudeInAIContainer(ctx context.Context, containerMgr interfaces.ContainerManager, getWorktrees func(context.Context) ([]*db.Worktree, error), args []string) error {
	// Get current worktree if not specified
	worktreeName := ""
	if len(args) > 0 {
		worktreeName = args[0]
	} else {
		// Try to detect current worktree
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// List all worktrees and check if we're in one
		worktrees, err := getWorktrees(ctx)
		if err != nil {
			return fmt.Errorf("failed to list worktrees: %w", err)
		}

		for _, wt := range worktrees {
			if strings.HasPrefix(cwd, wt.Path) {
				worktreeName = wt.Name
				break
			}
		}

		if worktreeName == "" {
			return fmt.Errorf("not in a worktree directory, please specify worktree name or run from within a worktree")
		}
	}

	// Find AI container for this worktree
	containers, err := containerMgr.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var aiContainerName string
	for _, c := range containers {
		if c.Type == "ai" && strings.Contains(c.Name, worktreeName) && c.Status != "exited" {
			// Check if container is running (handle different status formats)
			status := strings.ToLower(c.Status)
			if strings.Contains(status, "running") || strings.Contains(status, "up") {
				aiContainerName = c.Name
				break
			}
		}
	}

	if aiContainerName == "" {
		return fmt.Errorf("no running AI container found for worktree: %s\n\nTry starting the worktree first with: vibeman start %s", worktreeName, worktreeName)
	}

	// Start Claude CLI in the container
	logger.WithFields(logger.Fields{
		"container": aiContainerName,
		"worktree":  worktreeName,
	}).Info("Starting Claude CLI in AI container")

	// Use docker exec to run claude
	claudeCmd := exec.Command("docker", "exec", "-it", aiContainerName, "claude", "--dangerously-skip-permissions")
	claudeCmd.Stdin = os.Stdin
	claudeCmd.Stdout = os.Stdout
	claudeCmd.Stderr = os.Stderr

	return claudeCmd.Run()
}