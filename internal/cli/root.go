package cli

import (
	"github.com/spf13/cobra"
)

// createRootCommand creates the root command with global flags
func createRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "vibeman",
		Short: "Development environment management tool with Git worktree integration",
		Long: `vibeman is a development environment management tool that provides isolated
containerized environments for Git worktrees. It supports multiple container
runtimes (Docker, Apple Container, Docker Compose) and offers CLI and
web interfaces for managing your development workflow.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to showing help if no subcommand
			return cmd.Help()
		},
	}

	return rootCmd
}
