# Vibeman Development Guidelines

## API Development Workflow

### OpenAPI/TypeScript Generation
When adding or modifying backend API endpoints, follow this workflow to ensure type safety between Go backend and TypeScript frontend:

1. **Add Swagger annotations** to Go handlers in `internal/server/routes.go`:
   ```go
   // @Summary List worktrees
   // @Description Get a list of worktrees with optional filters
   // @Tags worktrees
   // @Accept json
   // @Produce json
   // @Param repository_id query string false "Filter by repository ID"
   // @Success 200 {object} WorktreesResponse
   // @Failure 500 {object} ErrorResponse
   // @Router /worktrees [get]
   func (s *Server) handleListWorktrees(c echo.Context) error {
   ```

2. **Define request/response models** in `internal/server/models.go` with proper JSON tags:
   ```go
   type WorktreesResponse struct {
       Worktrees []db.Worktree `json:"worktrees"`
       Total     int           `json:"total" example:"10"`
   }
   ```

3. **Generate OpenAPI spec and TypeScript client** - Use the Makefile:
   ```bash
   # Generate just OpenAPI spec
   make swagger
   
   # Generate OpenAPI spec AND TypeScript client
   make generate-api
   
   # Build everything (runs swagger generation automatically)
   make build
   
   # Or use go generate
   go generate ./...
   ```

4. **Frontend build automatically generates types**:
   ```bash
   cd vibeman-web && bun run build
   ```
   This automatically:
   - Regenerates the OpenAPI spec from Go annotations
   - Generates TypeScript types and TanStack Query hooks
   - Builds the frontend application

5. **Use generated hooks** in React components:
   ```typescript
   import { useWorktrees } from '@/hooks/api/useWorktrees';
   
   const { worktrees, isLoading, error, refetch } = useWorktrees({ 
     projectId: selectedProject?.id 
   });
   ```

### Important Notes
- **Always regenerate** types after backend API changes
- **Use TanStack Query hooks** for data fetching (not manual API calls)
- Generated files are in `vibeman-web/src/generated/api/`
- Create custom wrapper hooks in `vibeman-web/src/hooks/api/` for additional logic
- The generated client uses `@hey-api/client-fetch` for better TypeScript support

## Code Quality Standards

### Testing and Code Quality
- Always create tests for new features and functions
- Use idiomatic Go patterns and conventions
- Review code after major feature implementations
- Run linter (`go vet`) and formatter (`gofmt`) on all code
- Ensure comprehensive test coverage

### Code Structure
- Keep functions reasonably sized (prefer smaller, focused functions)
- Break large files into logical modules
- Prefer early return patterns over nested structures
- Use clear, descriptive names for functions and variables
- Follow Go naming conventions

### Development Workflow
- Use multiple subagents to parallelize work whenever possible
- Keep progress and todo lists updated in documentation
- Document architectural decisions and rationale
- Maintain clear separation of concerns between packages

## Project Structure

```
vibeman/
├── main.go                    # Application entry point
├── go.mod                     # Go module definition
├── CLAUDE.md                  # AI agent guidelines (this file)
├── README.md                  # Project documentation
├── DEVELOPMENT.md             # Development progress and todos
├── RESEARCH.md                # Research and design documentation
└── internal/
    ├── api/                   # Shared API client for CLI commands
    │   └── client.go          # HTTP client used by CLI
    ├── app/                   # Application orchestration
    ├── cli/                   # Command-line interface
    │   └── commands/          # CLI command implementations
    │       ├── service.go     # Service command definitions
    │       └── api_service_commands.go # API-based service functions
    ├── config/                # Configuration management
    ├── container/             # Container lifecycle management
    ├── db/                    # Database models and repositories
    ├── git/                   # Git operations and worktree management
    ├── server/                # HTTP API server
    │   ├── routes.go          # API route handlers
    │   └── models.go          # Request/response models
    └── service/               # Service orchestration (Docker/Compose)
```

## Key Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `github.com/pelletier/go-toml/v2` - TOML parsing
- `github.com/go-git/go-git/v5` - Git operations

## Architecture Principles

- **Dependency Injection**: Pass dependencies explicitly through interfaces
- **Interface Segregation**: Define small, focused interfaces
- **Error Handling**: Use explicit error handling, wrap errors with context
- **Configuration**: Use TOML files with sensible defaults
- **Modularity**: Keep packages focused on single responsibilities
- **Unified API Architecture**: Both CLI and web UI must use the same backend API for all operations

### Critical: CLI/Web UI Unified Architecture

**IMPORTANT**: The CLI and web UI must use the same backend operations/business logic. This ensures consistency, prevents data divergence, and maintains a single source of truth.

#### Current Architecture (✅ Correct)
```
┌─────────────────┐                      ┌──────────────────┐
│   CLI Commands  │ ────────────────────►│  Go Operations   │
└─────────────────┘   Direct Go calls    │  (Libraries)     │
                                         └──────────────────┘
                                                  ▲
                                                  │ Direct Go calls
┌─────────────────┐    HTTP API          ┌──────────────────┐    
│   Web UI        │ ───────────────────► │  OpenAPI Server  │
└─────────────────┘                      └──────────────────┘
```

#### Implementation Details

- **CLI Commands**: Directly import and use `internal/operations/` Go packages
- **Web UI**: Uses generated TypeScript client (from OpenAPI) to call HTTP API
- **OpenAPI Server**: Handlers in `internal/server/routes.go` use the same `internal/operations/` packages
- **Operations Layer**: `internal/operations/` contains all business logic (repository, worktree, service management)
- **Service Manager**: `internal/service/manager.go` manages Docker/Compose containers

#### Key Files for Unified Architecture

- `internal/operations/` - Shared business logic used by both CLI and server:
  - `repository.go` - Repository management operations
  - `worktree.go` - Worktree management operations
  - `service.go` - Service management operations
- `internal/cli/commands/` - CLI commands that directly use operations
- `internal/server/routes.go` - OpenAPI handlers that use operations
- `vibeman-web/src/lib/api-client.ts` - Web UI API client configuration

#### Web UI API Configuration

**Important**: The web UI API client must use relative URLs (`baseUrl: '/api'`) to go through the Bun dev server proxy, not direct URLs to the Go server.

```typescript
// ✅ Correct - uses proxy
export const createClientConfig: CreateClientConfig = (config) => ({
  baseUrl: '/api', // Relative URL goes through Bun proxy
});

// ❌ Wrong - bypasses proxy
export const createClientConfig: CreateClientConfig = (config) => ({
  baseUrl: 'http://localhost:8080/api', // Direct connection
});
```

#### When Adding New Features

1. **Operations First**: Implement business logic in `internal/operations/`
2. **CLI Integration**: Import operations packages directly in CLI commands
3. **API Integration**: OpenAPI handlers should be thin wrappers around operations
4. **Web UI Integration**: Use generated TypeScript client and TanStack Query hooks
5. **Never Split Logic**: Both CLI and API use the same operations layer

## Testing Strategy - PRIME DIRECTIVE

**CRITICAL RULE: Everything the system does must be tested with both:**
1. **REAL scenario integration tests** - Testing actual system behavior end-to-end
2. **Unit tests with mocks** - Testing individual components in isolation

### Testing Requirements
- **Integration tests must be run and validated after each phase of development**
- Keep `INTEGRATION_TEST_PLAN.md` updated with all integration test scenarios
- Unit tests for individual functions and components
- Integration tests for package interactions and end-to-end workflows
- Mock external dependencies (Docker, git commands, file system operations) in unit tests only
- Test error conditions and edge cases extensively
- Maintain test coverage above 80%
- **No feature is complete without both unit and integration tests**

### Testing Principles
- **Unit Tests**: Use mocks for all external dependencies (Docker, Git, filesystem)
- **Integration Tests**: Use REAL implementations - no mocks allowed!
  - Real Docker containers must be created, started, stopped, and removed
  - Real Docker Compose services must be orchestrated
  - Real Git repositories and worktrees must be created
  - Real filesystem operations must be performed
  - Integration tests use build tag `// +build integration`
  - Run with: `go test -tags=integration ./internal/integration/...`

### Test Organization
- Unit tests: `*_test.go` files alongside source code
- Integration tests: `internal/integration/*_integration_test.go`
- Test utilities: `internal/testutil/` for shared mocks and helpers
- Build tag `// +build integration` for integration tests

### Test Execution
- Unit tests: `go test ./...`
- Integration tests: `go test -tags=integration ./internal/integration/...`
- All tests: `go test -tags=integration ./...`

### Test Guidelines
- Unit tests for individual functions with mocked dependencies
- Integration tests for end-to-end workflows with real systems
- Test error conditions and edge cases extensively
- Maintain test coverage above 80%
- Integration tests must clean up all resources they create
- Each phase of development must validate integration tests pass

## Performance Considerations

- Use context.Context for cancellation and timeouts
- Implement proper resource cleanup
- Cache expensive operations where appropriate
- Use goroutines for concurrent operations
- Profile and optimize hot paths