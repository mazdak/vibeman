package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"vibeman/internal/container"
)

// CreateWorktreeRequest represents the worktree creation request
type CreateWorktreeRequest struct {
	RepoURL string `json:"repoURL"`
	Branch  string `json:"branch"`
	Path    string `json:"path"`
}

// Git operations

// CreateWorktree creates a new git worktree
func (c *Client) CreateWorktree(ctx context.Context, repoURL, branch, path string) error {
	req := CreateWorktreeRequest{
		RepoURL: repoURL,
		Branch:  branch,
		Path:    path,
	}

	resp, err := c.doRequest(ctx, "POST", "/api/git/worktrees", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create worktree: %s", resp.Status)
	}

	return nil
}

// ListWorktrees lists git worktrees
func (c *Client) ListWorktrees(ctx context.Context, repoPath string) ([]container.GitWorktree, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/git/worktrees?repo=%s", repoPath), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list worktrees: %s", resp.Status)
	}

	var worktrees []container.GitWorktree
	if err := json.NewDecoder(resp.Body).Decode(&worktrees); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return worktrees, nil
}

// RemoveWorktree removes a git worktree
func (c *Client) RemoveWorktree(ctx context.Context, path string) error {
	req := struct {
		Path string `json:"path"`
	}{
		Path: path,
	}

	resp, err := c.doRequest(ctx, "DELETE", "/api/git/worktrees", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to remove worktree: %s", resp.Status)
	}

	return nil
}

// SwitchBranch switches branch in a worktree
func (c *Client) SwitchBranch(ctx context.Context, path, branch string) error {
	req := struct {
		Path   string `json:"path"`
		Branch string `json:"branch"`
	}{
		Path:   path,
		Branch: branch,
	}

	resp, err := c.doRequest(ctx, "POST", "/api/git/switch-branch", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to switch branch: %s", resp.Status)
	}

	return nil
}

// UpdateWorktree updates a git worktree
func (c *Client) UpdateWorktree(ctx context.Context, path string) error {
	req := struct {
		Path string `json:"path"`
	}{
		Path: path,
	}

	resp, err := c.doRequest(ctx, "POST", "/api/git/update", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to update worktree: %s", resp.Status)
	}

	return nil
}

// CloneRepository clones a git repository
func (c *Client) CloneRepository(ctx context.Context, repoURL, path string) error {
	req := struct {
		RepoURL string `json:"repoURL"`
		Path    string `json:"path"`
	}{
		RepoURL: repoURL,
		Path:    path,
	}

	resp, err := c.doRequest(ctx, "POST", "/api/git/clone", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to clone repository: %s", resp.Status)
	}

	return nil
}

// IsRepository checks if a path is a git repository
func (c *Client) IsRepository(ctx context.Context, path string) (bool, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/git/is-repository?path=%s", path), nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to check repository: %s", resp.Status)
	}

	var result struct {
		IsRepository bool `json:"isRepository"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.IsRepository, nil
}

// GetDefaultBranch gets the default branch of a repository
func (c *Client) GetDefaultBranch(ctx context.Context, repoPath string) (string, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/git/default-branch?path=%s", repoPath), nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get default branch: %s", resp.Status)
	}

	var result struct {
		DefaultBranch string `json:"defaultBranch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.DefaultBranch, nil
}

// HasUncommittedChanges checks if a worktree has uncommitted changes
func (c *Client) HasUncommittedChanges(ctx context.Context, path string) (bool, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/git/uncommitted-changes?path=%s", path), nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to check uncommitted changes: %s", resp.Status)
	}

	var result struct {
		HasChanges bool `json:"hasChanges"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.HasChanges, nil
}
