# Vibeman Server Package

This package provides the HTTP/WebSocket server foundation for Vibeman's client-server architecture.

## Overview

The server package implements:
- RESTful API endpoints for managing tenants, projects, environments, and services
- WebSocket handlers for real-time features (terminal access, log streaming, events)
- Middleware for logging, CORS, request IDs, and authentication (placeholder)
- Configurable server with graceful shutdown

## Architecture

### Core Components

1. **Server** (`server.go`)
   - Main server struct that orchestrates everything
   - Handles startup, shutdown, and configuration
   - Integrates Echo framework for HTTP handling

2. **Routes** (`routes.go`)
   - Defines all API endpoints
   - Maps routes to handler functions
   - Currently contains placeholder implementations

3. **Middleware** (`middleware.go`)
   - Request logging and enrichment
   - CORS configuration
   - Request ID generation
   - Context management
   - Placeholder auth/tenant middleware

4. **WebSocket Manager** (`websocket.go`)
   - Manages WebSocket connections
   - Handles terminal, logs, and event streams
   - Message routing and connection lifecycle

## Usage

### Basic Server Setup

```go
import (
    "vibeman/internal/config"
    "vibeman/internal/server"
)

// Create server with default config
cfg := config.New()
srvConfig := server.DefaultConfig()
srv := server.New(srvConfig, cfg)

// Start server (blocks until shutdown)
if err := srv.Start(); err != nil {
    log.Fatal(err)
}
```

### Custom Configuration

```go
srvConfig := &server.Config{
    Host:            "0.0.0.0",
    Port:            8080,
    ReadTimeout:     10 * time.Second,
    WriteTimeout:    10 * time.Second,
    ShutdownTimeout: 30 * time.Second,
    AllowOrigins:    []string{"http://localhost:3000"},
    LogLevel:        "debug",
    LogFormat:       "json",
}

srv := server.New(srvConfig, configManager)
```

### Integration with Vibeman App

To integrate with the existing app structure:

```go
// In internal/app/app.go
type App struct {
    Config    *config.Manager
    Container *container.Manager
    Git       *git.Manager
    Service   *service.Manager
    CLI       *cli.Manager
    TUI       *tui.Manager
    Server    *server.Server  // Add this
}

// In app initialization
func New() *App {
    cfg := config.New()
    
    // Create server if in server mode
    var srv *server.Server
    if cfg.GetBool("server.enabled") {
        srvConfig := &server.Config{
            Port: cfg.GetInt("server.port"),
            // ... other config
        }
        srv = server.New(srvConfig, cfg)
    }
    
    return &App{
        Config:    cfg,
        Container: container.New(cfg),
        Git:       git.New(cfg),
        Service:   service.New(cfg),
        CLI:       cli.New(cfg),
        TUI:       tui.New(cfg),
        Server:    srv,
    }
}
```

## API Endpoints

### Authentication
- `POST /api/auth/login` - User login
- `POST /api/auth/refresh` - Refresh JWT token
- `POST /api/auth/logout` - User logout

### Tenants
- `GET /api/tenants` - List tenants
- `POST /api/tenants` - Create tenant
- `GET /api/tenants/:id` - Get tenant details
- `PUT /api/tenants/:id` - Update tenant
- `DELETE /api/tenants/:id` - Delete tenant

### Projects
- `GET /api/projects` - List projects
- `POST /api/projects` - Create project
- `GET /api/projects/:id` - Get project details
- `PUT /api/projects/:id` - Update project
- `DELETE /api/projects/:id` - Delete project

### Environments
- `GET /api/environments` - List environments
- `POST /api/environments` - Create environment
- `GET /api/environments/:id` - Get environment details
- `PUT /api/environments/:id` - Update environment
- `DELETE /api/environments/:id` - Delete environment
- `POST /api/environments/:id/start` - Start environment
- `POST /api/environments/:id/stop` - Stop environment

### Services
- `GET /api/services` - List services
- `POST /api/services/:id/start` - Start service
- `POST /api/services/:id/stop` - Stop service

### WebSocket Endpoints
- `WS /api/environments/:id/terminal` - Terminal access
- `WS /api/environments/:id/logs` - Log streaming
- `WS /api/environments/:id/events` - Event streaming

## WebSocket Protocol

### Message Format

All WebSocket messages use JSON:

```json
{
  "type": "message_type",
  "data": { ... }
}
```

### Message Types

#### Terminal
- `input` - Terminal input from client
- `output` - Terminal output to client
- `resize` - Terminal resize event

#### Common
- `error` - Error message
- `close` - Connection close
- `ping`/`pong` - Keep-alive

### Example Terminal Session

```javascript
// Client connects
ws = new WebSocket('ws://localhost:8080/api/environments/123/terminal');

// Send input
ws.send(JSON.stringify({
  type: 'input',
  data: { data: 'ls -la\n' }
}));

// Receive output
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === 'output') {
    console.log(msg.data.data);
  }
};

// Resize terminal
ws.send(JSON.stringify({
  type: 'resize',
  data: { cols: 120, rows: 40 }
}));
```

## Next Steps

1. **Database Integration**
   - Add database models for tenants, users, projects, environments
   - Implement actual CRUD operations in handlers

2. **Authentication**
   - Implement JWT-based authentication
   - Add proper auth middleware
   - User management endpoints

3. **Container Integration**
   - Connect WebSocket handlers to actual container terminals
   - Implement SSH proxy for terminal access
   - Log streaming from containers

4. **Service Discovery**
   - Dynamic service registration
   - Health checking
   - Load balancing for multiple server instances

## Testing

Run tests:
```bash
go test ./internal/server/... -v
```

Run with coverage:
```bash
go test ./internal/server/... -cover
```