package types

// GitWorktree represents a git worktree
type GitWorktree struct {
	Path      string
	Branch    string
	Commit    string
	IsMain    bool
	IsBare    bool
	IsLocked  bool
	CreatedAt string
}

// Container represents a container instance
type Container struct {
	ID          string
	Name        string
	Image       string
	Status      string
	Repository  string
	Environment string
	CreatedAt   string
	Ports       map[string]string
	Command     string
	EnvVars     map[string]string // All environment variables
	Type        string            // Container type: "worktree", "service", "ai"
}
