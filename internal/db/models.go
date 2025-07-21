// Package db provides database models for Vibeman
package db

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// JSONB represents a JSON column that works with both SQLite and PostgreSQL
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*j = nil
			return nil
		}
		return json.Unmarshal(v, j)
	case string:
		if v == "" {
			*j = nil
			return nil
		}
		return json.Unmarshal([]byte(v), j)
	default:
		return errors.New("type assertion to []byte or string failed")
	}
}

// Repository represents a tracked repository
type Repository struct {
	ID          string    `json:"id" db:"id"`
	Path        string    `json:"path" db:"path"` // Local filesystem path
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// TableName returns the table name for Repository
func (Repository) TableName() string {
	return "repositories"
}

// WorktreeStatus represents the status of a worktree
type WorktreeStatus string

const (
	StatusStopped  WorktreeStatus = "stopped"
	StatusStarting WorktreeStatus = "starting"
	StatusRunning  WorktreeStatus = "running"
	StatusStopping WorktreeStatus = "stopping"
	StatusError    WorktreeStatus = "error"
)

// Worktree represents a git worktree with its container environment
type Worktree struct {
	ID           string         `json:"id" db:"id"`
	RepositoryID string         `json:"repository_id" db:"repository_id"`
	Name         string         `json:"name" db:"name"`
	Branch       string         `json:"branch" db:"branch"`
	Path         string         `json:"path" db:"path"` // Filesystem path to worktree
	Status       WorktreeStatus `json:"status" db:"status"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

// TableName returns the table name for Worktree
func (Worktree) TableName() string {
	return "worktrees"
}
