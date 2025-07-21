// Package cache provides generic caching utilities
package cache

import (
	"context"
	"sync"
	"time"
)

// Cache represents a generic in-memory cache
type Cache[K comparable, V any] struct {
	items      map[K]*Item[V]
	mutex      sync.RWMutex
	defaultTTL time.Duration
	maxSize    int
}

// Item represents a cached item with expiration
type Item[V any] struct {
	Value     V
	ExpiresAt time.Time
	LastUsed  time.Time
}

// NewCache creates a new cache instance
func NewCache[K comparable, V any](defaultTTL time.Duration, maxSize int) *Cache[K, V] {
	cache := &Cache[K, V]{
		items:      make(map[K]*Item[V]),
		defaultTTL: defaultTTL,
		maxSize:    maxSize,
	}

	// Start cleanup goroutine
	go cache.startCleanup()

	return cache
}

// Set stores a value in the cache with default TTL
func (c *Cache[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL stores a value in the cache with custom TTL
func (c *Cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if we need to evict items
	if len(c.items) >= c.maxSize {
		c.evictLRU()
	}

	now := time.Now()
	c.items[key] = &Item[V]{
		Value:     value,
		ExpiresAt: now.Add(ttl),
		LastUsed:  now,
	}
}

// Get retrieves a value from the cache
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	item, exists := c.items[key]
	if !exists {
		var zero V
		return zero, false
	}

	// Check if expired
	if time.Now().After(item.ExpiresAt) {
		delete(c.items, key)
		var zero V
		return zero, false
	}

	// Update last used time
	item.LastUsed = time.Now()

	return item.Value, true
}

// Delete removes a value from the cache
func (c *Cache[K, V]) Delete(key K) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache[K, V]) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[K]*Item[V])
}

// Size returns the number of items in the cache
func (c *Cache[K, V]) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.items)
}

// Has checks if a key exists in the cache
func (c *Cache[K, V]) Has(key K) bool {
	_, exists := c.Get(key)
	return exists
}

// Keys returns all keys in the cache
func (c *Cache[K, V]) Keys() []K {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	keys := make([]K, 0, len(c.items))
	for key := range c.items {
		keys = append(keys, key)
	}

	return keys
}

// evictLRU removes the least recently used item
func (c *Cache[K, V]) evictLRU() {
	var oldestKey K
	var oldestTime time.Time
	first := true

	for key, item := range c.items {
		if first || item.LastUsed.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.LastUsed
			first = false
		}
	}

	if !first {
		delete(c.items, oldestKey)
	}
}

// startCleanup runs a cleanup goroutine to remove expired items
func (c *Cache[K, V]) startCleanup() {
	ticker := time.NewTicker(c.defaultTTL / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired items
func (c *Cache[K, V]) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.ExpiresAt) {
			delete(c.items, key)
		}
	}
}

// CachedRepository wraps a repository with caching
type CachedRepository[T any] struct {
	repo  Repository[T]
	cache *Cache[string, T]
}

// Repository interface for dependency injection
type Repository[T any] interface {
	Get(ctx context.Context, id string) (T, error)
	List(ctx context.Context, filter interface{}) ([]T, error)
	Create(ctx context.Context, entity T) error
	Update(ctx context.Context, id string, entity T) error
	Delete(ctx context.Context, id string) error
}

// NewCachedRepository creates a new cached repository
func NewCachedRepository[T any](repo Repository[T], ttl time.Duration, maxSize int) *CachedRepository[T] {
	return &CachedRepository[T]{
		repo:  repo,
		cache: NewCache[string, T](ttl, maxSize),
	}
}

// Get retrieves an entity, first checking the cache
func (r *CachedRepository[T]) Get(ctx context.Context, id string) (T, error) {
	// Check cache first
	if value, exists := r.cache.Get(id); exists {
		return value, nil
	}

	// Get from repository
	entity, err := r.repo.Get(ctx, id)
	if err != nil {
		return entity, err
	}

	// Store in cache
	r.cache.Set(id, entity)

	return entity, nil
}

// List retrieves entities (bypasses cache for simplicity)
func (r *CachedRepository[T]) List(ctx context.Context, filter interface{}) ([]T, error) {
	return r.repo.List(ctx, filter)
}

// Create creates an entity and invalidates related cache entries
func (r *CachedRepository[T]) Create(ctx context.Context, entity T) error {
	err := r.repo.Create(ctx, entity)
	if err != nil {
		return err
	}

	// Clear cache to ensure consistency
	r.cache.Clear()

	return nil
}

// Update updates an entity and invalidates cache
func (r *CachedRepository[T]) Update(ctx context.Context, id string, entity T) error {
	err := r.repo.Update(ctx, id, entity)
	if err != nil {
		return err
	}

	// Remove from cache
	r.cache.Delete(id)

	return nil
}

// Delete deletes an entity and removes from cache
func (r *CachedRepository[T]) Delete(ctx context.Context, id string) error {
	err := r.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	// Remove from cache
	r.cache.Delete(id)

	return nil
}

// InvalidateCache removes specific keys from cache
func (r *CachedRepository[T]) InvalidateCache(ids ...string) {
	for _, id := range ids {
		r.cache.Delete(id)
	}
}

// RefreshCache refreshes specific entries in the cache
func (r *CachedRepository[T]) RefreshCache(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		entity, err := r.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		r.cache.Set(id, entity)
	}
	return nil
}
