package container

import (
	"context"
	"os/exec"
	"sync"
	"time"
)

// PooledExecutor represents a pooled command executor with usage tracking
type PooledExecutor struct {
	executor CommandExecutor
	lastUsed time.Time
}

// ExecutorPool manages a pool of command executors for improved performance
type ExecutorPool struct {
	// Pool configuration
	maxSize     int
	idleTimeout time.Duration

	// Pool state
	executors []CommandExecutor
	available []CommandExecutor
	mu        sync.Mutex

	// Tracking
	lastCleanup time.Time

	// Cleanup
	stopCh      chan struct{}
	cleanupOnce sync.Once
}

// ExecutorPoolConfig holds configuration for the executor pool
type ExecutorPoolConfig struct {
	// MaxSize is the maximum number of executors in the pool (default: 5)
	MaxSize int
	// IdleTimeout is the duration after which idle executors are removed (default: 5 minutes)
	IdleTimeout time.Duration
}

// DefaultExecutorPoolConfig returns the default pool configuration
func DefaultExecutorPoolConfig() *ExecutorPoolConfig {
	return &ExecutorPoolConfig{
		MaxSize:     5,
		IdleTimeout: 5 * time.Minute,
	}
}

// NewExecutorPool creates a new executor pool
func NewExecutorPool(config *ExecutorPoolConfig) *ExecutorPool {
	if config == nil {
		config = DefaultExecutorPoolConfig()
	}

	if config.MaxSize <= 0 {
		config.MaxSize = 5
	}

	if config.IdleTimeout <= 0 {
		config.IdleTimeout = 5 * time.Minute
	}

	pool := &ExecutorPool{
		maxSize:     config.MaxSize,
		idleTimeout: config.IdleTimeout,
		executors:   make([]CommandExecutor, 0, config.MaxSize),
		available:   make([]CommandExecutor, 0, config.MaxSize),
		stopCh:      make(chan struct{}),
		lastCleanup: time.Now(),
	}

	// Start cleanup goroutine
	go pool.cleanupRoutine()

	return pool
}

// CommandContext implements the CommandExecutor interface using pooled executors
func (p *ExecutorPool) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	// Get an executor from the pool
	executor := p.getOrCreateExecutor()

	// Create the command
	cmd := executor.CommandContext(ctx, name, args...)

	// Add a cleanup function that runs when the context is done
	// This ensures the executor is returned to the pool even if the command fails
	go func() {
		<-ctx.Done()
		// Small delay to allow command to finish before returning executor
		time.Sleep(100 * time.Millisecond)
		p.returnExecutor(executor)
	}()

	return cmd
}

// getOrCreateExecutor gets an executor from the pool or creates a new one
func (p *ExecutorPool) getOrCreateExecutor() CommandExecutor {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Periodic cleanup check (non-blocking)
	if time.Since(p.lastCleanup) > 1*time.Minute {
		p.lastCleanup = time.Now()
		// Trim pool if it's too large and has available executors
		if len(p.executors) > p.maxSize && len(p.available) > 0 {
			// Remove excess available executors
			excess := len(p.executors) - p.maxSize
			if excess > len(p.available) {
				excess = len(p.available)
			}
			p.available = p.available[:len(p.available)-excess]
			p.executors = p.executors[:len(p.executors)-excess]
		}
	}

	// Try to get an available executor
	if len(p.available) > 0 {
		executor := p.available[len(p.available)-1]
		p.available = p.available[:len(p.available)-1]
		return executor
	}

	// Create a new executor if under limit
	if len(p.executors) < p.maxSize {
		executor := &DefaultCommandExecutor{}
		p.executors = append(p.executors, executor)
		return executor
	}

	// Pool is full, create a temporary executor
	return &DefaultCommandExecutor{}
}

// cleanupRoutine periodically cleans up the pool
func (p *ExecutorPool) cleanupRoutine() {
	ticker := time.NewTicker(p.idleTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.performCleanup()
		case <-p.stopCh:
			return
		}
	}
}

// performCleanup removes idle executors
func (p *ExecutorPool) performCleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// For now, we simply clear the available pool if it's been idle too long
	// In a real implementation, executors might have state to clean up
	if len(p.available) > 0 {
		p.available = p.available[:0] // Clear available slice
		p.executors = p.executors[:0] // Clear executors slice
	}
}

// Close shuts down the executor pool and cleans up resources
func (p *ExecutorPool) Close() error {
	p.cleanupOnce.Do(func() {
		close(p.stopCh)

		p.mu.Lock()
		defer p.mu.Unlock()

		// Clear all executors
		p.executors = nil
		p.available = nil
	})

	return nil
}

// Stats returns current pool statistics
type PoolStats struct {
	TotalExecutors int
	Available      int
	InUse          int
}

// GetStats returns current pool statistics
func (p *ExecutorPool) GetStats() PoolStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	return PoolStats{
		TotalExecutors: len(p.executors),
		Available:      len(p.available),
		InUse:          len(p.executors) - len(p.available),
	}
}

// returnExecutor returns an executor to the available pool
func (p *ExecutorPool) returnExecutor(executor CommandExecutor) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if this executor is part of our pool
	for _, e := range p.executors {
		if e == executor {
			// Return it to available pool if not already there
			for _, avail := range p.available {
				if avail == executor {
					return // Already in available pool
				}
			}
			p.available = append(p.available, executor)
			return
		}
	}
	// Not a pooled executor, ignore
}

// PooledCommandExecutor wraps an ExecutorPool to implement CommandExecutor
// This allows seamless integration with existing code
type PooledCommandExecutor struct {
	pool *ExecutorPool
}

// NewPooledCommandExecutor creates a new pooled command executor
func NewPooledCommandExecutor(config *ExecutorPoolConfig) *PooledCommandExecutor {
	return &PooledCommandExecutor{
		pool: NewExecutorPool(config),
	}
}

// CommandContext implements the CommandExecutor interface
func (e *PooledCommandExecutor) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return e.pool.CommandContext(ctx, name, args...)
}

// Close shuts down the underlying pool
func (e *PooledCommandExecutor) Close() error {
	return e.pool.Close()
}

// GetStats returns pool statistics
func (e *PooledCommandExecutor) GetStats() PoolStats {
	return e.pool.GetStats()
}
