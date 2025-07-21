package db

import (
	"context"
	"database/sql"
	"fmt"
)

// WorktreeRepository handles database operations for worktrees
type WorktreeRepository struct {
	db *DB
}

// NewWorktreeRepository creates a new worktree repository
func NewWorktreeRepository(db *DB) *WorktreeRepository {
	return &WorktreeRepository{db: db}
}

// List returns worktrees with optional filtering
func (r *WorktreeRepository) List(ctx context.Context, repositoryID, status string) ([]Worktree, error) {
	query := `
		SELECT id, repository_id, name, branch, path, status, created_at, updated_at
		FROM worktrees 
		WHERE 1=1`
	args := []interface{}{}

	if repositoryID != "" {
		query += " AND repository_id = ?"
		args = append(args, repositoryID)
	}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query worktrees: %w", err)
	}
	defer rows.Close()

	var worktrees []Worktree
	for rows.Next() {
		var w Worktree
		err := rows.Scan(
			&w.ID,
			&w.RepositoryID,
			&w.Name,
			&w.Branch,
			&w.Path,
			&w.Status,
			&w.CreatedAt,
			&w.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan worktree: %w", err)
		}
		worktrees = append(worktrees, w)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating worktrees: %w", err)
	}

	return worktrees, nil
}

// Get returns a worktree by ID
func (r *WorktreeRepository) Get(ctx context.Context, id string) (*Worktree, error) {
	query := `
		SELECT id, repository_id, name, branch, path, status, created_at, updated_at
		FROM worktrees 
		WHERE id = ?`

	var w Worktree
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&w.ID,
		&w.RepositoryID,
		&w.Name,
		&w.Branch,
		&w.Path,
		&w.Status,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("worktree not found")
		}
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	return &w, nil
}

// GetByPath returns a worktree by its filesystem path
func (r *WorktreeRepository) GetByPath(ctx context.Context, path string) (*Worktree, error) {
	query := `
		SELECT id, repository_id, name, branch, path, status, created_at, updated_at
		FROM worktrees 
		WHERE path = ?`

	var w Worktree
	err := r.db.QueryRowContext(ctx, query, path).Scan(
		&w.ID,
		&w.RepositoryID,
		&w.Name,
		&w.Branch,
		&w.Path,
		&w.Status,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("worktree not found")
		}
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	return &w, nil
}

// Create creates a new worktree
func (r *WorktreeRepository) Create(ctx context.Context, worktree *Worktree) error {
	query := `
		INSERT INTO worktrees (id, repository_id, name, branch, path, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	_, err := r.db.ExecContext(ctx, query,
		worktree.ID,
		worktree.RepositoryID,
		worktree.Name,
		worktree.Branch,
		worktree.Path,
		worktree.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	return nil
}

// Update updates a worktree
func (r *WorktreeRepository) Update(ctx context.Context, worktree *Worktree) error {
	query := `
		UPDATE worktrees 
		SET name = ?, branch = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, worktree.Name, worktree.Branch, worktree.Status, worktree.ID)
	if err != nil {
		return fmt.Errorf("failed to update worktree: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("worktree not found")
	}

	return nil
}

// UpdateStatus updates only the status of a worktree
func (r *WorktreeRepository) UpdateStatus(ctx context.Context, id string, status WorktreeStatus) error {
	query := `
		UPDATE worktrees 
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update worktree status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("worktree not found")
	}

	return nil
}

// Delete deletes a worktree
func (r *WorktreeRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM worktrees WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete worktree: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("worktree not found")
	}

	return nil
}

// ListByRepository returns all worktrees for a specific repository
func (r *WorktreeRepository) ListByRepository(ctx context.Context, repositoryID string) ([]Worktree, error) {
	return r.List(ctx, repositoryID, "")
}
