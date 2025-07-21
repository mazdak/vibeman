# Vibeman Specification

Vibeman is a system for managing AI coding environments across multiple Git worktrees. It consists of:
- An OpenAPI server (built with Go)
- A CLI (for Unix-like machines)
- A Web UI (written in React, powered by Bun) for remote access

## Architecture

```
┌─────────────┐
│   Web UI    │
└──────┬──────┘
       │ ▲
       ▼ │
┌─────────────┐                      ┌─────────────┐
│  OpenAPI    │                      │     CLI     │
│   Server    │                      └──────┬──────┘
└──────┬──────┘                             │ ▲
       │ ▲                                  ▼ │
       ▼ │                           ┌─────────────┐
       └────────────────────────────►│  Go API     │
                                     │  Libraries  │
                                     └─────────────┘
```

The architecture shows that:
- The Web UI communicates with the OpenAPI server (HTTP)
- The CLI directly uses the Go API libraries (function calls)
- The OpenAPI server also uses the same Go API libraries internally
- The "API" is traditional Go packages/libraries, not a separate server

Vibeman helps developers easily run AI coding agents like Claude Code in multiple worktrees associated with multiple repositories.

Vibeman uses Docker and Docker Compose for container management and Git with worktrees for source code management.

The Vibeman server (typically port 8080) must be running in the background for the web UI. The CLI works independently, while the web UI communicates with the server to perform tasks such as:

* Getting lists and status of repositories, worktrees, containers, etc.
* Starting and managing new worktrees along with associated container environments

Here is a complete example of a `vibeman.toml` configuration file for a repository:


```toml
# Vibeman configuration example

[repository]
name = "resq-fullstack"
description = "Resq fullstack application"

[repository.container]
# Use docker-compose for application services only
compose_file = "./docker-compose.yaml"
# Launch backend, worker, and frontend for each worktree
services = ["backend", "worker", "frontend"]

[repository.worktrees]
directory = "../resq-fullstack-worktrees"

[repository.git]
repo_url = "."  # Local repository
default_branch = "main"
auto_sync = false

[repository.runtime]
type = "docker"

# Use shared global services instead of per-worktree
[repository.services]
postgres = { required = true }
redis = { required = true }
localstack = { required = true }
```

Here is an example global services configuration file that follows XDG path structure. Global services are shared among multiple worktree container environments:


```toml
# Vibeman Services Configuration Example
# Services are defined by referencing docker-compose files

[services]

# PostgreSQL with PostGIS from resq-fullstack
[services.postgres]
compose_file = "/Users/mazdak/Code/resq-fullstack/docker-compose.yaml"
service = "postgres"
description = "PostgreSQL with PostGIS for development"

# Redis from resq-fullstack
[services.redis]
compose_file = "/Users/mazdak/Code/resq-fullstack/docker-compose.yaml"
service = "redis"
description = "Redis cache server"

# LocalStack from resq-fullstack
[services.localstack]
compose_file = "/Users/mazdak/Code/resq-fullstack/docker-compose.yaml"
service = "localstack"
description = "LocalStack for AWS services emulation"
```

When launching an environment for a worktree, any required services specified in the `vibeman.toml` are started if they are not already running. Services remain running until explicitly stopped by the user. An AI container instance automatically running Claude Code is also created for the worktree. Logs from other containers should be made available in this container so the AI has a complete picture of the system.

## AI Container Integration

Vibeman automatically provisions AI containers for each worktree with the following features:

**Automatic Container Creation**: When a worktree is started, an AI container with Claude CLI and development tools is automatically created and attached.

**Development Environment**: AI containers include:
- Claude CLI (`claude --dangerously-skip-permissions`)
- Modern shell (zsh with oh-my-zsh)
- Development tools (fzf, fd, ast-grep, ripgrep)
- Complete workspace access via volume mounts

**Access Methods**:
- CLI: `vibeman ai` - Direct terminal access to AI container
- Web UI: WebSocket-based terminal via `/api/ai/attach/{worktree}` endpoint
- Full terminal functionality with bidirectional stdin/stdout/stderr

**Configuration**: AI containers can be configured in `vibeman.toml`:
```toml
[repository.ai]
enabled = true  # Default: true
image = "vibeman/ai-assistant:latest"  # Default image
env = { "CUSTOM_VAR" = "value" }  # Custom environment variables
volumes = { "/host/path" = "/container/path" }  # Additional volumes
```

**Log Integration**: AI containers have access to aggregated logs from all worktree containers, providing complete system visibility for AI assistance.


## Vibeman CLI

The Vibeman CLI provides a clean and intuitive interface for managing development environments.

`vibeman server start` - Start the server with required parameters. The global XDG Vibeman configuration can specify important setting overrides like port. For security, the server always listens on `localhost`. Supports `--port`, `--config`, and `--daemon` flags.
`vibeman server stop` - Gracefully stop a running Vibeman server. Sends a SIGTERM signal and waits for the process to exit.
`vibeman server status` - Check if the Vibeman server is running and show basic status information.
`vibeman start` - Read the local `vibeman.toml` and start the AI container for the current worktree. Required global services are started automatically if not already running. Supports specifying a custom `.toml` file path.
`vibeman stop` - Stop containers for the current worktree. Global services remain running and must be stopped explicitly using `vibeman services stop`.
`vibeman services start/stop` - Start or stop all global shared services. Services are shared across worktrees and persist until explicitly stopped.
`vibeman worktree add` - Create a new Git worktree following conventions specified in either global or local Vibeman configuration. Runs post-scripts if specified and optionally changes to the new directory. This command mirrors `git worktree add` with additional Vibeman features.
`vibeman worktree remove` - Remove a worktree. Cannot remove the current worktree. Checks for uncommitted, unstaged, and unpushed files, then prompts for user permission before stopping the worktree's containers and removing the Git worktree and its directory. Global services are not affected.
`vibeman list` - List containers with status for the current directory (reads `vibeman.toml`). Use `--all` to list all active worktree environments with their repositories and containers.
`vibeman ai` - Start Claude CLI in the AI container for the current worktree (default behavior).
`vibeman ai attach [worktree]` - Attach to AI container shell for the specified or current worktree.
`vibeman ai claude [worktree]` - Start Claude CLI in AI container for the specified or current worktree.
`vibeman ai list` - List all AI containers with their status.
`vibeman ai logs [worktree]` - View logs from the AI container for the specified or current worktree.
`vibeman repo add` - Add a repository to the list of known Vibeman repositories. This is useful for both the web UI and the list command. Takes a path argument: if it's a URL (SSH or HTTP), clones to the default repos directory; if it's a local path, verifies it's a valid Git repository before adding. Creates `vibeman.toml` if it doesn't exist, prompting for any required information. 
`vibeman repo list` - List known repositories with their path, name, and description (read from each repository's `vibeman.toml` file). Also shows worktrees associated with each repository.
`vibeman repo remove` - Remove a repository from the tracked list. Does not delete any files, only stops tracking the repository.
`vibeman init` - If global configuration does not exist, guides the user through creating the global configuration. 


## Global Configuration

The global configuration is located in the XDG folder (e.g., `~/.config/vibeman/config.toml`).

Configurable options:
- Server port (default: 8080)
- Web UI port (default: 8081)
- Default repositories location (default: `~/vibeman/repos`)
- Default worktrees location (default: `~/vibeman/worktrees`) - can be overridden per repository
- Default services configuration location (default: `<XDG path>/services.toml`)

## OpenAPI Server Endpoints

The Vibeman server provides a comprehensive REST API for managing repositories, worktrees, containers, and services. All endpoints return JSON responses and follow standard HTTP status codes.

### System Status
- **GET /api/status** - Get comprehensive system health and statistics
  - Returns system status, version, uptime, resource counts
  - Includes health checks for database, container engine, and git
  - Example response:
    ```json
    {
      "status": "healthy",
      "version": "1.0.0",
      "uptime": "2h30m15s",
      "services": {
        "database": "healthy",
        "container_engine": "healthy", 
        "git": "healthy"
      },
      "repositories": 5,
      "worktrees": 12,
      "containers": 3
    }
    ```

### Repository Management
- **GET /api/repositories** - List all tracked repositories
- **POST /api/repositories** - Add a new repository to tracking
- **GET /api/repositories/{id}** - Get specific repository details
- **DELETE /api/repositories/{id}** - Remove repository from tracking

### Worktree Management
- **GET /api/worktrees** - List worktrees with optional filtering
  - Query parameters: `repository_id`, `status`
- **POST /api/worktrees** - Create a new worktree
  - Supports post-scripts, compose overrides, service dependencies
- **GET /api/worktrees/{id}** - Get specific worktree details
- **DELETE /api/worktrees/{id}** - Remove worktree (with safety checks)
- **POST /api/worktrees/{id}/start** - Start worktree containers
- **POST /api/worktrees/{id}/stop** - Stop worktree containers

### Service Management
- **GET /api/services** - List global services with status
- **POST /api/services/{id}/start** - Start a service
- **POST /api/services/{id}/stop** - Stop a service

### Container Management
- **GET /api/containers** - List all containers with metadata
  - Query parameters: `repository`, `worktree`, `status`
- **POST /api/containers** - Create a new container
- **GET /api/containers/{id}** - Get specific container details
- **DELETE /api/containers/{id}** - Remove a container
- **POST /api/containers/{id}/action** - Perform container actions (start/stop/restart)
- **GET /api/containers/{id}/logs** - Get container logs

### AI Container Access
- **GET /api/ai/attach/{worktree}** - WebSocket endpoint for terminal access to AI containers
  - Supports bidirectional terminal communication (stdin/stdout/stderr)
  - Message types: `stdin`, `resize`, `ping` (client) and `stdout`, `stderr`, `exit`, `pong` (server)

### Logs and Monitoring
- **GET /api/worktrees/{id}/logs** - Get worktree logs
  - Query parameters: `lines` (limit number of lines returned)
  - Reads from XDG log files: `~/.local/share/vibeman/logs/{repo}/{worktree}/worktree.log`
- **GET /api/services/{id}/logs** - Get service logs
  - Query parameters: `lines` (limit number of lines returned)
  - Reads from service log files, falls back to container logs if unavailable
  - Log file location: `~/.local/share/vibeman/logs/services/{service}.log`

### Configuration
- **GET /api/config** - Get global configuration (read-only)

All API endpoints use structured error responses with appropriate HTTP status codes:
- **200 OK** - Successful operation
- **201 Created** - Resource created successfully
- **204 No Content** - Successful deletion
- **400 Bad Request** - Invalid request parameters
- **404 Not Found** - Resource not found
- **500 Internal Server Error** - Server error
- **503 Service Unavailable** - Required service unavailable (e.g., database)
