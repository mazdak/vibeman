// Package lazy provides lazy loading utilities
package lazy

import (
	"context"
	"sync"
)

// Loader function signature for lazy loading
type Loader[T any] func(ctx context.Context) (T, error)

// Lazy represents a lazy-loaded value
type Lazy[T any] struct {
	loader Loader[T]
	value  T
	err    error
	loaded bool
	mutex  sync.Mutex
}

// New creates a new lazy value with a loader function
func New[T any](loader Loader[T]) *Lazy[T] {
	return &Lazy[T]{
		loader: loader,
	}
}

// Get returns the value, loading it if necessary
func (l *Lazy[T]) Get(ctx context.Context) (T, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if !l.loaded {
		l.value, l.err = l.loader(ctx)
		l.loaded = true
	}

	return l.value, l.err
}

// IsLoaded returns true if the value has been loaded
func (l *Lazy[T]) IsLoaded() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.loaded
}

// Reset clears the cached value, forcing reload on next Get
func (l *Lazy[T]) Reset() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	var zero T
	l.value = zero
	l.err = nil
	l.loaded = false
}

// LazyWorktreeDetails implements lazy loading for worktree details
type LazyWorktreeDetails struct {
	worktreeID string
	loader     WorktreeDetailsLoader
	lazy       *Lazy[*WorktreeDetails]
}

// WorktreeDetailsLoader interface for loading worktree details
type WorktreeDetailsLoader interface {
	LoadDetails(ctx context.Context, worktreeID string) (*WorktreeDetails, error)
}

// WorktreeDetails contains expensive-to-load worktree information
type WorktreeDetails struct {
	ContainerInfo    *ContainerInfo `json:"container_info,omitempty"`
	GitInfo          *GitInfo       `json:"git_info,omitempty"`
	ServiceInstances []*ServiceInfo `json:"service_instances,omitempty"`
	VolumeMetrics    *VolumeMetrics `json:"volume_metrics,omitempty"`
	LogFiles         []LogFileInfo  `json:"log_files,omitempty"`
}

// ContainerInfo contains container details
type ContainerInfo struct {
	ID       string            `json:"id"`
	Status   string            `json:"status"`
	Image    string            `json:"image"`
	Ports    map[string]string `json:"ports"`
	Networks []string          `json:"networks"`
	Volumes  []string          `json:"volumes"`
	Env      map[string]string `json:"env"`
}

// GitInfo contains git repository details
type GitInfo struct {
	Branch           string   `json:"branch"`
	CommitHash       string   `json:"commit_hash"`
	CommitMessage    string   `json:"commit_message"`
	Author           string   `json:"author"`
	UncommittedFiles []string `json:"uncommitted_files,omitempty"`
	UnpushedCommits  int      `json:"unpushed_commits"`
	RemoteURL        string   `json:"remote_url"`
}

// ServiceInfo contains service instance details
type ServiceInfo struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Status    string                 `json:"status"`
	Health    string                 `json:"health"`
	Endpoints []string               `json:"endpoints"`
	Config    map[string]interface{} `json:"config"`
}

// VolumeMetrics contains volume usage metrics
type VolumeMetrics struct {
	TotalSize     int64 `json:"total_size"`
	UsedSize      int64 `json:"used_size"`
	AvailableSize int64 `json:"available_size"`
	FileCount     int   `json:"file_count"`
}

// LogFileInfo contains log file information
type LogFileInfo struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified"`
	Type         string `json:"type"`
}

// NewLazyWorktreeDetails creates a new lazy worktree details loader
func NewLazyWorktreeDetails(worktreeID string, loader WorktreeDetailsLoader) *LazyWorktreeDetails {
	lazyDetails := &LazyWorktreeDetails{
		worktreeID: worktreeID,
		loader:     loader,
	}

	lazyDetails.lazy = New(func(ctx context.Context) (*WorktreeDetails, error) {
		return loader.LoadDetails(ctx, worktreeID)
	})

	return lazyDetails
}

// GetDetails returns the worktree details, loading them if necessary
func (l *LazyWorktreeDetails) GetDetails(ctx context.Context) (*WorktreeDetails, error) {
	return l.lazy.Get(ctx)
}

// IsLoaded returns true if details have been loaded
func (l *LazyWorktreeDetails) IsLoaded() bool {
	return l.lazy.IsLoaded()
}

// Refresh forces a reload of the details
func (l *LazyWorktreeDetails) Refresh() {
	l.lazy.Reset()
}

// LazyServiceManager implements lazy loading for service manager operations
type LazyServiceManager struct {
	services map[string]*Lazy[*ServiceInfo]
	loader   ServiceInfoLoader
	mutex    sync.RWMutex
}

// ServiceInfoLoader interface for loading service information
type ServiceInfoLoader interface {
	LoadServiceInfo(ctx context.Context, serviceName string) (*ServiceInfo, error)
}

// NewLazyServiceManager creates a new lazy service manager
func NewLazyServiceManager(loader ServiceInfoLoader) *LazyServiceManager {
	return &LazyServiceManager{
		services: make(map[string]*Lazy[*ServiceInfo]),
		loader:   loader,
	}
}

// GetService returns service information, loading it if necessary
func (l *LazyServiceManager) GetService(ctx context.Context, serviceName string) (*ServiceInfo, error) {
	l.mutex.RLock()
	lazyService, exists := l.services[serviceName]
	l.mutex.RUnlock()

	if !exists {
		l.mutex.Lock()
		// Double-check in case another goroutine added it
		if lazyService, exists = l.services[serviceName]; !exists {
			lazyService = New(func(ctx context.Context) (*ServiceInfo, error) {
				return l.loader.LoadServiceInfo(ctx, serviceName)
			})
			l.services[serviceName] = lazyService
		}
		l.mutex.Unlock()
	}

	return lazyService.Get(ctx)
}

// IsServiceLoaded returns true if service info has been loaded
func (l *LazyServiceManager) IsServiceLoaded(serviceName string) bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if lazyService, exists := l.services[serviceName]; exists {
		return lazyService.IsLoaded()
	}
	return false
}

// RefreshService forces a reload of service information
func (l *LazyServiceManager) RefreshService(serviceName string) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if lazyService, exists := l.services[serviceName]; exists {
		lazyService.Reset()
	}
}

// RefreshAll forces a reload of all cached services
func (l *LazyServiceManager) RefreshAll() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	for _, lazyService := range l.services {
		lazyService.Reset()
	}
}

// LazyCollection represents a lazily-loaded collection of items
type LazyCollection[T any] struct {
	loader func(ctx context.Context) ([]T, error)
	items  []T
	loaded bool
	mutex  sync.RWMutex
}

// NewLazyCollection creates a new lazy collection
func NewLazyCollection[T any](loader func(ctx context.Context) ([]T, error)) *LazyCollection[T] {
	return &LazyCollection[T]{
		loader: loader,
	}
}

// Get returns all items, loading them if necessary
func (l *LazyCollection[T]) Get(ctx context.Context) ([]T, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if !l.loaded {
		items, err := l.loader(ctx)
		if err != nil {
			return nil, err
		}
		l.items = items
		l.loaded = true
	}

	return l.items, nil
}

// IsLoaded returns true if the collection has been loaded
func (l *LazyCollection[T]) IsLoaded() bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.loaded
}

// Reset clears the cached items
func (l *LazyCollection[T]) Reset() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.items = nil
	l.loaded = false
}

// Size returns the number of items (loads if necessary)
func (l *LazyCollection[T]) Size(ctx context.Context) (int, error) {
	items, err := l.Get(ctx)
	if err != nil {
		return 0, err
	}
	return len(items), nil
}
