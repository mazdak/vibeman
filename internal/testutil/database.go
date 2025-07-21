package testutil

import (
	"database/sql"
	"testing"
	
	"vibeman/internal/db"
	_ "github.com/mattn/go-sqlite3"
	"github.com/jmoiron/sqlx"
)

// SetupTestDB creates a new in-memory database for testing
func SetupTestDB(t *testing.T) *db.DB {
	t.Helper()
	
	// Create a real SQLite in-memory database using raw sql package first
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create raw database: %v", err)
	}
	
	// Enable foreign keys
	if _, err := rawDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}
	
	// Create schema directly
	schema := `
		CREATE TABLE repositories (
			id TEXT PRIMARY KEY,
			path TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE worktrees (
			id TEXT PRIMARY KEY,
			repository_id TEXT NOT NULL,
			name TEXT NOT NULL,
			branch TEXT NOT NULL,
			path TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'stopped' CHECK (status IN ('stopped', 'starting', 'running', 'stopping', 'error')),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE,
			UNIQUE(repository_id, name)
		);

		CREATE INDEX idx_worktrees_repository_id ON worktrees(repository_id);
		CREATE INDEX idx_worktrees_status ON worktrees(status);

		CREATE TRIGGER update_repositories_updated_at AFTER UPDATE ON repositories
		BEGIN
			UPDATE repositories SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;

		CREATE TRIGGER update_worktrees_updated_at AFTER UPDATE ON worktrees
		BEGIN
			UPDATE worktrees SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;
	`
	
	if _, err := rawDB.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	
	// Verify tables were created
	var count int
	err = rawDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('repositories', 'worktrees')").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify tables: %v", err)
	}
	if count != 2 {
		t.Fatalf("Expected 2 tables, got %d", count)
	}
	
	// Now wrap it with sqlx
	sqlxDB := sqlx.NewDb(rawDB, "sqlite3")
	
	// Create our custom DB wrapper with the same interface but skip migrations
	database := &db.DB{
		DB: sqlxDB,
	}
	
	// Register cleanup
	t.Cleanup(func() {
		database.Close()
	})
	
	return database
}