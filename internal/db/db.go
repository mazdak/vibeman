// Package db provides database connectivity and initialization for Vibeman
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"vibeman/internal/xdg"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Config represents database configuration
type Config struct {
	// Driver specifies the database driver (sqlite3, postgres)
	Driver string
	// DSN is the data source name
	DSN string
	// MaxOpenConns is the maximum number of open connections
	MaxOpenConns int
	// MaxIdleConns is the maximum number of idle connections
	MaxIdleConns int
	// ConnMaxLifetime is the maximum lifetime of a connection
	ConnMaxLifetime time.Duration
	// ConnMaxIdleTime is the maximum idle time of a connection
	ConnMaxIdleTime time.Duration
}

// getDefaultDatabasePath returns the XDG-compliant database path
func getDefaultDatabasePath() string {
	dataDir, err := xdg.DataDir()
	if err != nil {
		// Fallback to ~/.local/share/vibeman
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, ".local", "share", "vibeman", "vibeman.db")
	}
	return filepath.Join(dataDir, "vibeman.db")
}

// DefaultConfig returns a default SQLite configuration
func DefaultConfig() *Config {
	dbPath := getDefaultDatabasePath()

	return &Config{
		Driver:          "sqlite3",
		DSN:             dbPath,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}
}

// DB wraps sqlx.DB with additional functionality
type DB struct {
	*sqlx.DB
	config *Config
}

// New creates a new database connection
func New(cfg *Config) (*DB, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Ensure directory exists for SQLite
	if cfg.Driver == "sqlite3" {
		dir := filepath.Dir(cfg.DSN)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Open database connection
	db, err := sqlx.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign keys for SQLite
	if cfg.Driver == "sqlite3" {
		if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
		}
	}

	return &DB{
		DB:     db,
		config: cfg,
	}, nil
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	var dbDriver string
	var dbInstance database.Driver

	switch db.config.Driver {
	case "sqlite3":
		dbDriver = "sqlite3"
		dbInstance, err = sqlite3.WithInstance(db.DB.DB, &sqlite3.Config{})
		if err != nil {
			return fmt.Errorf("failed to create sqlite3 driver instance: %w", err)
		}
	case "postgres":
		// TODO: Add PostgreSQL support
		return fmt.Errorf("PostgreSQL support not yet implemented")
	default:
		return fmt.Errorf("unsupported database driver: %s", db.config.Driver)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, dbDriver, dbInstance)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// BeginTx starts a new transaction
func (db *DB) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return db.DB.BeginTxx(ctx, nil)
}

// Transaction executes a function within a transaction
func (db *DB) Transaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx failed: %v, unable to rollback: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Simple query to ensure database is responsive
	var result int
	if err := db.GetContext(ctx, &result, "SELECT 1"); err != nil {
		return fmt.Errorf("health check query failed: %w", err)
	}

	return nil
}

// Stats returns database statistics
func (db *DB) Stats() sql.DBStats {
	return db.DB.Stats()
}
