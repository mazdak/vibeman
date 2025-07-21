// Package repository provides generic repository patterns for data access
package repository

import (
	"context"
	"time"
)

// Filter represents generic query filters
type Filter struct {
	// Pagination
	Limit  int
	Offset int

	// Sorting
	OrderBy string
	Order   string // "asc" or "desc"

	// Time-based filters
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	UpdatedAfter  *time.Time
	UpdatedBefore *time.Time

	// Custom filters
	Conditions map[string]interface{}
}

// Repository defines a generic repository interface
type Repository[T any] interface {
	// Create creates a new entity
	Create(ctx context.Context, entity T) error

	// Get retrieves an entity by ID
	Get(ctx context.Context, id string) (T, error)

	// List retrieves multiple entities with optional filtering
	List(ctx context.Context, filter Filter) ([]T, error)

	// Update updates an existing entity
	Update(ctx context.Context, id string, entity T) error

	// Delete removes an entity
	Delete(ctx context.Context, id string) error

	// Count returns the total number of entities matching the filter
	Count(ctx context.Context, filter Filter) (int64, error)

	// Exists checks if an entity exists
	Exists(ctx context.Context, id string) (bool, error)
}

// BatchRepository extends Repository with batch operations
type BatchRepository[T any] interface {
	Repository[T]

	// CreateBatch creates multiple entities
	CreateBatch(ctx context.Context, entities []T) error

	// UpdateBatch updates multiple entities
	UpdateBatch(ctx context.Context, updates map[string]T) error

	// DeleteBatch deletes multiple entities
	DeleteBatch(ctx context.Context, ids []string) error
}

// TransactionalRepository extends Repository with transaction support
type TransactionalRepository[T any] interface {
	Repository[T]

	// WithTransaction executes a function within a transaction
	WithTransaction(ctx context.Context, fn func(Repository[T]) error) error
}

// CachedRepository provides caching capabilities
type CachedRepository[T any] interface {
	Repository[T]

	// InvalidateCache invalidates cache entries
	InvalidateCache(ctx context.Context, ids ...string) error

	// RefreshCache refreshes cache entries
	RefreshCache(ctx context.Context, ids ...string) error
}

// Identifiable represents an entity with an ID
type Identifiable interface {
	GetID() string
}

// Timestamped represents an entity with timestamps
type Timestamped interface {
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	SetUpdatedAt(time.Time)
}

// SoftDeletable represents an entity that supports soft deletion
type SoftDeletable interface {
	IsDeleted() bool
	SetDeleted(deleted bool)
	GetDeletedAt() *time.Time
	SetDeletedAt(t *time.Time)
}
