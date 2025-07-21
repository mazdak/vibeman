package db

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
)

// RepositoryManager defines the interface for repository database operations
type RepositoryManager interface {
	CreateRepository(ctx context.Context, repo *Repository) error
	ListRepositories(ctx context.Context) ([]*Repository, error)
	GetRepositoryByName(ctx context.Context, name string) (*Repository, error)
	GetRepositoryByID(ctx context.Context, id string) (*Repository, error)
	GetRepositoryByPath(ctx context.Context, path string) (*Repository, error)
	DeleteRepository(ctx context.Context, id string) error
	GetWorktreesByRepository(ctx context.Context, repoID string) ([]*Worktree, error)
}

// RepositoryRepository handles database operations for repositories
type RepositoryRepository struct {
	db *DB
}

// NewRepositoryRepository creates a new repository repository
func NewRepositoryRepository(db *DB) *RepositoryRepository {
	return &RepositoryRepository{db: db}
}

// List returns all tracked repositories
func (r *RepositoryRepository) List(ctx context.Context) ([]*Repository, error) {
	query := `
		SELECT id, path, name, description, created_at, updated_at
		FROM repositories
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query repositories: %w", err)
	}
	defer rows.Close()

	var repositories []*Repository
	for rows.Next() {
		repo := &Repository{}
		err := rows.Scan(
			&repo.ID,
			&repo.Path,
			&repo.Name,
			&repo.Description,
			&repo.CreatedAt,
			&repo.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}
		repositories = append(repositories, repo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating repositories: %w", err)
	}

	return repositories, nil
}

// GetByID returns a repository by ID
func (r *RepositoryRepository) GetByID(ctx context.Context, id string) (*Repository, error) {
	query := `
		SELECT id, path, name, description, created_at, updated_at
		FROM repositories
		WHERE id = ?
	`

	repo := &Repository{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&repo.ID,
		&repo.Path,
		&repo.Name,
		&repo.Description,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("repository not found")
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repo, nil
}

// GetByPath returns a repository by path
func (r *RepositoryRepository) GetByPath(ctx context.Context, path string) (*Repository, error) {
	query := `
		SELECT id, path, name, description, created_at, updated_at
		FROM repositories
		WHERE path = ?
	`

	repo := &Repository{}
	err := r.db.QueryRowContext(ctx, query, path).Scan(
		&repo.ID,
		&repo.Path,
		&repo.Name,
		&repo.Description,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("repository not found")
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repo, nil
}

// ExistsWithPath checks if a repository exists with the given path
func (r *RepositoryRepository) ExistsWithPath(ctx context.Context, path string) (bool, error) {
	query := `SELECT COUNT(*) FROM repositories WHERE path = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, path).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check repository existence: %w", err)
	}

	return count > 0, nil
}

// Create creates a new repository
func (r *RepositoryRepository) Create(ctx context.Context, repo *Repository) error {
	query := `
		INSERT INTO repositories (id, path, name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	_, err := r.db.ExecContext(ctx, query, repo.ID, repo.Path, repo.Name, repo.Description)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	return nil
}

// Update updates a repository
func (r *RepositoryRepository) Update(ctx context.Context, repo *Repository) error {
	query := `
		UPDATE repositories
		SET name = ?, description = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, repo.Name, repo.Description, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("repository not found")
	}

	return nil
}

// Delete deletes a repository
func (r *RepositoryRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM repositories WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("repository not found")
	}

	return nil
}

// Interface method implementations for RepositoryManager

// CreateRepository creates a new repository (implements RepositoryManager interface)
func (r *RepositoryRepository) CreateRepository(ctx context.Context, repo *Repository) error {
	// Generate ID if not provided
	if repo.ID == "" {
		repo.ID = uuid.New().String()
	}
	return r.Create(ctx, repo)
}

// ListRepositories returns all repositories (implements RepositoryManager interface)
func (r *RepositoryRepository) ListRepositories(ctx context.Context) ([]*Repository, error) {
	return r.List(ctx)
}

// GetRepositoryByName returns a repository by name
func (r *RepositoryRepository) GetRepositoryByName(ctx context.Context, name string) (*Repository, error) {
	query := `
		SELECT id, path, name, description, created_at, updated_at
		FROM repositories
		WHERE name = ?
	`

	repo := &Repository{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&repo.ID,
		&repo.Path,
		&repo.Name,
		&repo.Description,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("repository not found")
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repo, nil
}

// GetRepositoryByID returns a repository by ID (implements RepositoryManager interface)
func (r *RepositoryRepository) GetRepositoryByID(ctx context.Context, id string) (*Repository, error) {
	return r.GetByID(ctx, id)
}

// GetRepositoryByPath returns a repository by path (implements RepositoryManager interface)
func (r *RepositoryRepository) GetRepositoryByPath(ctx context.Context, path string) (*Repository, error) {
	return r.GetByPath(ctx, path)
}

// DeleteRepository deletes a repository (implements RepositoryManager interface)
func (r *RepositoryRepository) DeleteRepository(ctx context.Context, id string) error {
	return r.Delete(ctx, id)
}

// GetWorktreesByRepository returns all worktrees for a repository
func (r *RepositoryRepository) GetWorktreesByRepository(ctx context.Context, repoID string) ([]*Worktree, error) {
	query := `
		SELECT id, repository_id, name, branch, path, created_at, updated_at
		FROM worktrees
		WHERE repository_id = ?
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query worktrees: %w", err)
	}
	defer rows.Close()

	var worktrees []*Worktree
	for rows.Next() {
		wt := &Worktree{}
		err := rows.Scan(
			&wt.ID,
			&wt.RepositoryID,
			&wt.Name,
			&wt.Branch,
			&wt.Path,
			&wt.CreatedAt,
			&wt.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan worktree: %w", err)
		}
		worktrees = append(worktrees, wt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating worktrees: %w", err)
	}

	return worktrees, nil
}
