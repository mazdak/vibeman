# Vibeman

> AI-powered development environment management for Git worktrees

Vibeman helps developers easily run AI coding agents like Claude in isolated, reproducible development environments across multiple Git worktrees. Each worktree gets its own containerized environment with AI assistance built-in.

## What is Vibeman?

Vibeman solves the complexity of managing multiple development environments when working with AI coding assistants. It provides:

- **🌳 Git Worktree Management**: Create and manage multiple worktrees from a single repository
- **🐳 Containerized Environments**: Each worktree runs in isolated Docker containers
- **🤖 AI Integration**: Built-in AI containers with Claude CLI and development tools
- **🔄 Shared Services**: Reuse databases, caches, and other services across worktrees
- **🖥️ Multiple Interfaces**: CLI, Web UI, and native Mac app

## Key Features

### For Developers
- **Instant Environment Setup**: Spin up a new development environment in seconds
- **AI-Powered Coding**: Each worktree includes an AI container with Claude CLI pre-configured
- **Consistent Environments**: Docker Compose ensures everyone has the same setup
- **Resource Efficient**: Share services like databases across multiple worktrees

### For Teams
- **Reproducible Environments**: `vibeman.toml` defines your entire setup
- **Easy Onboarding**: New developers can start coding immediately
- **Parallel Development**: Work on multiple features simultaneously
- **No More "Works on My Machine"**: Everything runs in containers

## Getting Started

### Installation

1. **Install Dependencies**:
   - Docker Desktop
   - Git
   - Go 1.21+ (for building from source)

2. **Install Vibeman**:
   ```bash
   # Clone the repository
   git clone https://github.com/yourusername/vibeman.git
   cd vibeman
   
   # Build the CLI
   go build -o vibeman
   
   # Move to PATH
   sudo mv vibeman /usr/local/bin/
   ```

3. **Initialize Your Repository**:
   ```bash
   cd your-project
   vibeman init
   ```

### Quick Start

1. **Create a `vibeman.toml`** in your repository:
   ```toml
   [repository]
   name = "my-project"
   description = "My awesome project"
   
   [repository.container]
   compose_file = "./docker-compose.yaml"
   services = ["backend", "frontend"]  # or leave empty for all services
   
   [repository.worktrees]
   directory = "../my-project-worktrees"
   ```

2. **Start the Vibeman server**:
   ```bash
   vibeman server start
   ```

3. **Create a new worktree**:
   ```bash
   vibeman worktree add feature-xyz
   ```

4. **Access your AI assistant**:
   ```bash
   vibeman ai
   ```

## Architecture

Vibeman consists of three main components:

### 1. CLI (`vibeman`)
The command-line interface for managing repositories, worktrees, and containers.

### 2. Web UI
A React-based web interface for visual management and monitoring.

### 3. Mac App (Optional)
A native macOS menu bar application that bundles everything together.

## Usage Examples

### Managing Worktrees
```bash
# List all worktrees
vibeman list

# Create a new worktree
vibeman worktree add feature-auth

# Start containers for current worktree
vibeman start

# Shell into a container
vibeman worktree shell

# Remove a worktree
vibeman worktree remove feature-auth
```

### Working with AI
```bash
# Open AI assistant in current worktree
vibeman ai

# Attach to AI container shell
vibeman ai attach

# View AI container logs
vibeman ai logs
```

### Managing Services
```bash
# Start shared services (databases, etc.)
vibeman services start

# Stop all services
vibeman services stop

# List service status
vibeman services list
```

## Configuration

### Repository Configuration (`vibeman.toml`)
```toml
[repository]
name = "my-app"
description = "My application"

[repository.container]
compose_file = "./docker-compose.yaml"
services = ["web", "worker"]  # Services to start (empty = all)
setup = [
    "npm install",
    "npm run build"
]

[repository.services]
postgres = { required = true }
redis = { required = false }

[repository.ai]
enabled = true  # Enable AI container (default: true)
image = "vibeman/ai-assistant:latest"
```

### Global Configuration
Located at `~/.config/vibeman/config.toml`:
```toml
[server]
port = 8080
webui_port = 8081

[storage]
repositories_path = "~/vibeman/repos"
worktrees_path = "~/vibeman/worktrees"
```

## Web UI

Access the web interface at http://localhost:8081 (when server is running).

Features:
- Visual repository and worktree management
- Container status monitoring
- Log viewing
- Service management
- Terminal access to AI containers

## Mac App

The native Mac app provides:
- Menu bar access
- Automatic server management
- Integrated web UI
- One-click CLI installation
- Auto-updates

## AI Container Features

Each worktree automatically gets an AI container with:
- **Claude CLI** pre-installed and configured
- **Development Tools**: ripgrep, fd, fzf, ast-grep, bat, exa
- **Modern Shell**: zsh with oh-my-zsh
- **Log Access**: Aggregated logs from all containers
- **Full Workspace Access**: Complete access to worktree files

## Requirements

- **macOS** 12.0+ (for Mac app)
- **Docker** Desktop or compatible container runtime
- **Git** 2.20+ (for worktree support)
- **Bun** (for web UI development)

## Development

### Building from Source

```bash
# Build CLI
go build -o vibeman

# Build Web UI
cd vibeman-web
bun install
bun run build

# Build Mac App
cd vibeman-mac
./build.sh
```

### Running Tests

```bash
# Go tests
go test ./...

# Integration tests
go test -tags=integration ./...

# Web UI tests
cd vibeman-web
bun test
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

[MIT License](LICENSE)

## Support

- **Documentation**: [docs.vibeman.dev](https://docs.vibeman.dev)
- **Issues**: [GitHub Issues](https://github.com/yourusername/vibeman/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/vibeman/discussions)