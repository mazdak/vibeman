-- Simple schema for Vibeman (aligned with SPEC)

-- Repositories table (tracked repositories)
CREATE TABLE IF NOT EXISTS repositories (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,  -- Local filesystem path
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Worktrees table
CREATE TABLE IF NOT EXISTS worktrees (
    id TEXT PRIMARY KEY,
    repository_id TEXT NOT NULL,
    name TEXT NOT NULL,
    branch TEXT NOT NULL,
    path TEXT NOT NULL,  -- Filesystem path to worktree
    status TEXT NOT NULL DEFAULT 'stopped' CHECK (status IN ('stopped', 'starting', 'running', 'stopping', 'error')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE,
    UNIQUE(repository_id, name)
);

-- Indexes for performance
CREATE INDEX idx_worktrees_repository_id ON worktrees(repository_id);
CREATE INDEX idx_worktrees_status ON worktrees(status);

-- Triggers for updated_at timestamp
CREATE TRIGGER update_repositories_updated_at AFTER UPDATE ON repositories
BEGIN
    UPDATE repositories SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_worktrees_updated_at AFTER UPDATE ON worktrees
BEGIN
    UPDATE worktrees SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;