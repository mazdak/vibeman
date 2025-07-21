# Git Manager

The Git manager provides comprehensive Git operations for the Vibeman project, with a focus on worktree management for containerized development environments.

## Features

- **Worktree Management**: Create, list, remove, and switch between Git worktrees
- **Repository Operations**: Clone repositories, check status, get branch information
- **Authentication Support**: SSH key, SSH agent, username/password, and GitHub token authentication
- **Integration Ready**: Designed to work with the container manager and project configuration

## Usage

### Basic Setup

```go
import (
    "context"
    "vibeman/internal/config"
    "vibeman/internal/git"
)

cfg := config.New()
err := cfg.Load()
if err != nil {
    // handle error
}

gitManager := git.New(cfg)
```

### Worktree Operations

#### Create a Worktree

```go
ctx := context.Background()
repoURL := "https://github.com/user/repo.git"
branch := "feature-branch"
path := "/path/to/worktree"

err := gitManager.CreateWorktree(ctx, repoURL, branch, path)
if err != nil {
    // handle error
}
```

#### List Worktrees

```go
worktrees, err := gitManager.ListWorktrees(ctx, repoURL)
if err != nil {
    // handle error
}

for _, wt := range worktrees {
    fmt.Printf("Path: %s, Branch: %s, Commit: %s\n", wt.Path, wt.Branch, wt.Commit)
}
```

#### Remove a Worktree

```go
err := gitManager.RemoveWorktree(ctx, "/path/to/worktree")
if err != nil {
    // handle error
}
```

#### Switch Branch in Worktree

```go
err := gitManager.SwitchBranch(ctx, "/path/to/worktree", "new-branch")
if err != nil {
    // handle error
}
```

### Repository Operations

#### Clone Repository

```go
err := gitManager.CloneRepository(ctx, "https://github.com/user/repo.git", "/path/to/clone")
if err != nil {
    // handle error
}
```

#### Check Repository Status

```go
isRepo := gitManager.IsRepository("/path/to/directory")
if isRepo {
    fmt.Println("Directory is a Git repository")
}

hasChanges, err := gitManager.HasUncommittedChanges(ctx, "/path/to/repo")
if err != nil {
    // handle error
}
if hasChanges {
    fmt.Println("Repository has uncommitted changes")
}
```

#### Get Repository Information

```go
repo, err := gitManager.GetRepository(ctx, "/path/to/repo")
if err != nil {
    // handle error
}

fmt.Printf("URL: %s, Branch: %s, Worktrees: %d\n", repo.URL, repo.Branch, len(repo.Worktrees))
```

#### Get Branches

```go
branches, err := gitManager.GetBranches(ctx, "/path/to/repo")
if err != nil {
    // handle error
}

for _, branch := range branches {
    fmt.Printf("Branch: %s\n", branch)
}
```

#### Get Commit Information

```go
commit, err := gitManager.GetCommitInfo(ctx, "/path/to/repo")
if err != nil {
    // handle error
}

fmt.Printf("Commit: %s, Author: %s, Message: %s\n", 
    commit.Hash, commit.Author.Name, commit.Message)
```

### Authentication

The Git manager supports multiple authentication methods through environment variables:

#### SSH Key Authentication

```bash
export SSH_KEY_PATH="/path/to/your/private/key"
```

#### SSH Agent Authentication

The manager will automatically try to use SSH agent if available.

#### Username/Password Authentication

```bash
export GIT_USERNAME="your-username"
export GIT_PASSWORD="your-password"
```

#### GitHub Token Authentication

```bash
export GITHUB_TOKEN="your-github-token"
```

### Configuration

The Git manager uses the global configuration from `config.Manager`:

```toml
[global.git]
default_branch = "main"
worktree_dir = "/home/user/dev/worktrees"
auto_fetch = true
fetch_interval = "5m"
```

## Implementation Details

### Worktree Management

- Main repositories are stored as bare repositories in `{worktree_dir}/.repos/`
- Worktrees are created using the `git worktree add` command for better compatibility
- The manager automatically handles branch creation and remote tracking

### Error Handling

All methods return descriptive errors that can be used for debugging and user feedback.

### Testing

The manager includes comprehensive tests covering:
- Unit tests for all public methods
- Integration tests with real Git repositories
- Benchmark tests for performance monitoring
- Authentication method testing

Run tests with:
```bash
go test ./internal/git/... -v
```

Run benchmarks with:
```bash
go test ./internal/git/... -bench=.
```

## Thread Safety

The Git manager is designed to be thread-safe for concurrent use, though individual operations on the same repository should be serialized to avoid conflicts.

## Performance

The manager uses the go-git library for most operations to avoid external dependencies, with selective use of the git command for worktree operations that are not yet supported by go-git.

Benchmark results on Apple M4:
- `IsRepository`: ~14µs per operation
- `GetCommitInfo`: ~54µs per operation