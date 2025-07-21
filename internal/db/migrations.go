// Package db provides migration utilities
package db

import (
	"context"
	"fmt"
	"time"
)

// MigrationInfo represents information about a migration
type MigrationInfo struct {
	Version     uint      `db:"version"`
	Description string    `db:"description"`
	AppliedAt   time.Time `db:"applied_at"`
}

// GetMigrationHistory returns the migration history
func (db *DB) GetMigrationHistory(ctx context.Context) ([]MigrationInfo, error) {
	query := `
		SELECT version, dirty as description, tstamp as applied_at
		FROM schema_migrations
		ORDER BY version DESC
	`

	var migrations []MigrationInfo
	if err := db.SelectContext(ctx, &migrations, query); err != nil {
		return nil, fmt.Errorf("failed to get migration history: %w", err)
	}

	return migrations, nil
}

// GetCurrentVersion returns the current migration version
func (db *DB) GetCurrentVersion(ctx context.Context) (uint, error) {
	var version uint
	query := `SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1`

	if err := db.GetContext(ctx, &version, query); err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	return version, nil
}
