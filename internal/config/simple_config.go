package config

import "fmt"

// SimplifiedRepositoryConfig - Compose-first configuration
type SimplifiedRepositoryConfig struct {
	Repository struct {
		Name        string `toml:"name"`
		Description string `toml:"description"`
		Git         struct {
			RepoURL        string `toml:"repo_url"`
			DefaultBranch  string `toml:"default_branch"`
			WorktreePrefix string `toml:"worktree_prefix"`
			AutoSync       bool   `toml:"auto_sync"`
		} `toml:"git"`
		Worktrees struct {
			Directory string `toml:"directory"`
		} `toml:"worktrees"`
		Container struct {
			// Required: Docker compose configuration
			ComposeFile string   `toml:"compose_file"` // Path to docker-compose.yaml
			Services    []string `toml:"services"`     // Services to start from compose file (empty = all)

			// Optional: Setup commands that run inside the container
			Setup []string `toml:"setup"` // Commands to run after container starts
		} `toml:"container"`
		Services map[string]ServiceRequirement `toml:"-"` // Service requirements (custom unmarshal)
		Runtime  struct {
			Type string `toml:"type"` // "apple" or "docker"
		} `toml:"runtime"`
		Setup struct {
			WorktreeInit  string   `toml:"worktree_init"`
			ContainerInit []string `toml:"container_init"`
		} `toml:"setup"`
		AI AIConfig `toml:"ai"`
	} `toml:"repository"`
}

// Validation for the simplified config
func (c *SimplifiedRepositoryConfig) Validate() error {
	if c.Repository.Name == "" {
		return fmt.Errorf("repository name is required")
	}

	if c.Repository.Container.ComposeFile == "" {
		return fmt.Errorf("compose_file is required")
	}

	return nil
}

// This is what the vibeman.toml should look like with the simplified approach:
//
// [repository]
// name = "my-project"
// description = "My awesome repository"
//
// [repository.git]
// repo_url = "https://github.com/user/repo.git"
// default_branch = "main"
//
// [repository.container]
// compose_file = "./docker-compose.dev.yaml"
// services = ["backend", "postgres", "redis"]  # Empty or omitted means all services
// setup = [
//     "npm install",
//     "pip install -r requirements.txt"
// ]
//
// [repository.services]
// postgres = { required = true }
// redis = { required = false }
//
// [repository.ai]
// enabled = true  # Default is true, set to false to disable AI container
// # image = "custom/ai-image:latest"  # Optional custom image
// # [repository.ai.env]
// # CUSTOM_VAR = "value"
// # [repository.ai.volumes]
// # "/host/path" = "/container/path"
//
// That's it! Much cleaner.
