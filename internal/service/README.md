# Service Manager

The Service Manager provides orchestration capabilities for shared container services like PostgreSQL, Redis, and other dependencies used by development environments.

## Features

- **Service Lifecycle Management**: Start, stop, and monitor container services
- **Reference Counting**: Automatically manage service instances based on project usage
- **Health Monitoring**: Built-in health checks with configurable timeouts and retries
- **Dependency Management**: Automatically start service dependencies
- **Thread Safety**: Concurrent access to service operations
- **Service Discovery**: Find and list running services

## Architecture

### Core Components

- **Manager**: Main service orchestration engine
- **ServiceInstance**: Represents a running service with metadata
- **ServiceStatus**: Service state tracking (stopped, starting, running, stopping, error)
- **Health Checks**: Configurable health monitoring for services

### Key Data Structures

```go
type Manager struct {
    config   *config.Manager
    services map[string]*ServiceInstance
    mutex    sync.RWMutex
}

type ServiceInstance struct {
    Name        string
    ContainerID string
    Status      ServiceStatus
    RefCount    int
    Projects    []string
    Config      config.ServiceConfig
    StartTime   time.Time
    LastHealth  time.Time
    HealthError string
}
```

## Usage

### Basic Service Operations

```go
// Create manager
cfg := config.New()
manager := service.New(cfg)

// Start a service
err := manager.StartService(ctx, "postgres")

// Stop a service
err := manager.StopService(ctx, "postgres")

// Get service info
service, err := manager.GetService("postgres")

// List all services
services := manager.ListServices()
```

### Reference Counting

Services use reference counting to determine when they should be stopped:

```go
// Add project reference (increments ref count)
err := manager.AddReference("postgres", "my-project")

// Remove project reference (decrements ref count)
err := manager.RemoveReference("postgres", "my-project")

// Service stops automatically when ref count reaches 0
```

### Health Monitoring

```go
// Perform health check
err := manager.HealthCheck(ctx, "postgres")

// Health check configuration in services.toml:
[services.postgres.healthcheck]
test = ["CMD-SHELL", "pg_isready -U user"]
interval = "30s"
timeout = "5s"
retries = 3
```

## Configuration

Services are configured in `~/.config/vibeman/services.toml`:

```toml
[services.postgres]
image = "postgres:15-alpine"
container_name = "vibeman-postgres"
restart_policy = "unless-stopped"
ports = ["5432:5432"]
environment = [
    "POSTGRES_DB=development",
    "POSTGRES_USER=dev",
    "POSTGRES_PASSWORD=dev123"
]
volumes = ["postgres-data:/var/lib/postgresql/data"]
depends_on = []

[services.postgres.healthcheck]
test = ["CMD-SHELL", "pg_isready -U dev"]
interval = "30s"
timeout = "5s"
retries = 3

[services.redis]
image = "redis:7-alpine"
container_name = "vibeman-redis"
restart_policy = "unless-stopped"
ports = ["6379:6379"]
command = ["redis-server", "--appendonly", "yes"]
volumes = ["redis-data:/data"]
depends_on = ["postgres"]

[services.redis.healthcheck]
test = ["CMD", "redis-cli", "ping"]
interval = "30s"
timeout = "5s"
retries = 3
```

## Integration with Container Manager

The service manager integrates with the container manager to automatically start required services for projects:

```go
// In container manager
func (m *Manager) Create(ctx context.Context, projectName, environment, image string) (*Container, error) {
    // Start required services
    for serviceName, serviceReq := range m.config.Project.Project.Services {
        if serviceReq.Required {
            err := m.service.StartService(ctx, serviceName)
            err = m.service.AddReference(serviceName, projectName)
        }
    }
    
    // Create container...
}
```

## Error Handling

The service manager provides detailed error information:

- **Configuration errors**: Missing service configurations
- **Container errors**: Failed to start/stop containers
- **Health check failures**: Service health monitoring issues
- **Reference errors**: Invalid project references

## Thread Safety

All operations are thread-safe and can be called concurrently:

- Read operations use `sync.RWMutex.RLock()`
- Write operations use `sync.RWMutex.Lock()`
- Individual service instances have their own mutexes

## Apple Container CLI Integration

The service manager uses the Apple Container CLI for container operations:

```bash
# Start service
container run -d --name postgres postgres:15-alpine

# Stop service
container stop <container-id>
container rm <container-id>

# Health check
container exec <container-id> pg_isready -U user

# Status check
container inspect <container-id> --format "{{.State.Running}}"
```

## Testing

The service manager includes comprehensive tests with mocked container operations:

```bash
go test ./internal/service -v
```

Tests cover:
- Service lifecycle operations
- Reference counting
- Health monitoring
- Error conditions
- Concurrent access
- Configuration validation

## Future Enhancements

- **Auto-restart**: Automatically restart failed services
- **Service discovery**: Network-based service discovery
- **Load balancing**: Multiple instances of the same service
- **Metrics collection**: Service performance monitoring
- **Log aggregation**: Centralized logging for services