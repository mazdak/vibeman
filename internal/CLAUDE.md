# Go Best Practices and Patterns


## 1. 🏗️ Architecture Patterns

### Dependency Injection Pattern

```go
// Good: Dependencies are explicit and testable
func NewDockerCommand(log *logrus.Entry, osCommand *OSCommand, tr *i18n.TranslationSet, config *config.AppConfig, errorChan chan error) (*DockerCommand, error) {
    // All dependencies injected, making testing easy
}
```

**Lesson**: Always inject dependencies rather than creating them internally. This makes code more testable and loosely coupled.

### Clean Architecture Layers
```
main.go → app → gui/commands/config → external APIs
```

**Lesson**: Maintain clear boundaries between layers. The GUI doesn't directly talk to Docker; it goes through a command layer.

## 2. 🔌 Interface Design

### Small, Focused Interfaces
```go
// Good: Interface segregation
type LimitedDockerCommand interface {
    GetContainers() ([]*Container, error)
    RefreshImages() ([]*Image, error)
    // Only methods the GUI actually needs
}
```

**Lesson**: Create interfaces based on what consumers need, not what providers offer.

### Interface Naming Convention
- Interfaces end with `-er` when possible: `Closer`, `Reader`
- Use descriptive names when `-er` doesn't work: `ContainerRuntime`

## 3. 📦 Generic Patterns

### Type-Safe Generic Panels
```go
type SideListPanel[T any] struct {
    List         *FilteredList[T]
    View         *gocui.View
    GetTableCells func(T) []string
    // Generic implementation reusable for any type
}
```

**Lesson**: Use generics to create reusable components while maintaining type safety.

## 4. ❌ Error Handling

### Structured Error Types
```go
// Define specific error variables
var (
    ErrDockerCommandNotAvailable = errors.New("docker command not available")
    ErrUnsupportedRuntime        = errors.New("unsupported runtime")
)

// Use them for precise error handling
if err == ErrDockerCommandNotAvailable {
    // Handle specifically
}
```

**Lesson**: Define error variables for expected errors. This enables precise error handling and testing.

### Error Wrapping with Context
```go
func WrapError(err error) error {
    if err == nil {
        return err
    }
    return goErrors.Wrap(err, 0)
}
```

**Lesson**: Wrap errors with stack traces for debugging while maintaining the original error for comparison.

## 5. 🧪 Testing Patterns

### Table-Driven Tests
```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "TEST",
            wantErr:  false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

**Lesson**: Use table-driven tests for comprehensive coverage with minimal code duplication.

### Mock Interfaces, Not Structs
The codebase creates interfaces for external dependencies, making them easy to mock:
```go
type OSCommand interface {
    RunCommand(string) error
    // Mockable interface
}
```

## 6. 🎨 GUI/TUI Patterns

### Separation of Presentation and Logic
```go
// Presentation layer (pure functions)
func GetContainerDisplayStrings(config *GuiConfig, container *Container) []string {
    // Only formatting, no business logic
}

// Business logic stays in commands
func (d *DockerCommand) GetContainers() ([]*Container, error) {
    // Actual Docker interaction
}
```

**Lesson**: Keep presentation logic separate from business logic for better testability and maintainability.

### Event-Driven Updates
```go
// Listen for Docker events and update UI
messageChan, errChan := client.Events(context.Background(), events.ListOptions{})
```

**Lesson**: Use event streams for real-time updates rather than polling when possible.

## 7. 🔧 Configuration Management

### Layered Configuration
```go
type AppConfig struct {
    Debug       bool   // From CLI flags
    ConfigDir   string // From environment
    UserConfig  *UserConfig // From YAML file
}
```

**Lesson**: Support multiple configuration sources with clear precedence.

### Self-Documenting Config
```go
type UserConfig struct {
    Gui GuiConfig `yaml:"gui" jsonschema:"description=GUI configuration"`
    // JSON schema annotations for auto-documentation
}
```

## 8. 🌍 Internationalization

### Structured Translation System
```go
type TranslationSet struct {
    ContainersTitle string
    ImagesTitle     string
    // All UI strings in one place
}
```

**Lesson**: Centralize all user-facing strings for easy internationalization.

## 9. 🚀 Performance Patterns

### Lazy Loading
```go
func (c *Container) DetailsLoaded() bool {
    return c.Details != nil
}

// Load details only when needed
if !container.DetailsLoaded() {
    container.LoadDetails()
}
```

**Lesson**: Don't load expensive data until it's actually needed.

### Concurrent Operations
```go
// Parallel refresh of different resource types
go func() { gui.refreshContainers() }()
go func() { gui.refreshImages() }()
go func() { gui.refreshVolumes() }()
```

**Lesson**: Use goroutines for independent operations, but be careful with shared state.

## 10. 🛡️ Defensive Programming

### Nil Checks at Boundaries
```go
func (g *GuiContainerCommand) GetClient() interface{} {
    if g.dockerCommand != nil {
        return g.dockerCommand.Client
    }
    return nil
}
```

**Lesson**: Always check for nil at API boundaries to prevent panics.

### Graceful Degradation
```go
if !isAppleContainerAvailable() {
    // Fall back to Docker or show appropriate error
}
```

**Lesson**: Design systems to degrade gracefully when optional features are unavailable.

## 12. 🔨 Build and Development

### Build Tags for Platform-Specific Code
```go
// +build !windows

package commands
```

**Lesson**: Use build tags to handle platform-specific functionality cleanly.

### Version Information
```go
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)
// Set during build with -ldflags
```

**Lesson**: Include version information in binaries for easier debugging.

## 13. 🎯 Key Takeaways

1. **Interfaces are boundaries**: Use them to decouple components
2. **Errors are values**: Treat them as first-class citizens
3. **Configuration is code**: Make it type-safe and validated
4. **Tests are documentation**: Write them to explain behavior
5. **Concurrency is not parallelism**: Use goroutines wisely
6. **Generics reduce duplication**: But don't overuse them
7. **Logging is not printf**: Use structured logging
8. **Dependencies are explicit**: Inject them, don't hide them

## 14. 🚫 Anti-Patterns to Avoid

Here are patterns to avoid:

1. **Global state**: Carefully avoid globals except for build-time constants
2. **Interface pollution**: Don't create interfaces before you need them
3. **Premature optimization**: The codebase focuses on clarity over micro-optimizations
4. **Ignoring errors**: Every error is either handled or explicitly propagated
5. **Tight coupling**: Components communicate through interfaces, not concrete types

## 15. 🌟 Unique Patterns

### Command Pattern with Templates
```go
type CommandObject struct {
    Container   *Container
    DockerCompose string
}
// Templates allow user customization while maintaining safety
```

### Contextual Keybindings
Different keybindings based on the focused panel - a pattern useful for any TUI application.

### Resource Monitoring
The stats monitoring pattern using Docker's stats API could be adapted for any resource monitoring need.

---

These patterns and lessons from lazydocker demonstrate mature Go development practices that lead to maintainable, testable, and user-friendly applications.
