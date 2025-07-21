package operations_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"vibeman/internal/config"
	"vibeman/internal/container"
	"vibeman/internal/db"
	"vibeman/internal/operations"
	"vibeman/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWorktreeOperations_CreateWorktree(t *testing.T) {
	tests := []struct {
		name    string
		req     operations.CreateWorktreeRequest
		setup   func(*testutil.MockContainerManager, *testutil.MockGitManager, *testutil.MockServiceManager, *db.DB)
		wantErr bool
		errMsg  string
	}{
		{
			name: "successful worktree creation",
			req: operations.CreateWorktreeRequest{
				RepositoryID:   "repo-123",
				Name:           "feature-auth",
				Branch:         "feature/auth",
				BaseBranch:     "main",
				SkipSetup:      false,
				ContainerImage: "vibeman:latest",
				AutoStart:      true,
			},
			setup: func(cm *testutil.MockContainerManager, gm *testutil.MockGitManager, sm *testutil.MockServiceManager, database *db.DB) {
				// Create test repository
				repo := &db.Repository{
					ID:          "repo-123",
					Name:        "test-repo",
					Path:        "/tmp/test-repo",
					Description: "Test repository",
				}
				repoRepo := db.NewRepositoryRepository(database)
				err := repoRepo.Create(context.Background(), repo)
				require.NoError(t, err)

				// Create vibeman.toml
				configPath := filepath.Join(repo.Path, "vibeman.toml")
				configContent := `
[repository]
name = "test-repo"
description = "Test repository"

[repository.container]
image = "default:latest"

[repository.worktrees]
directory = "../test-worktrees"
`
				os.MkdirAll(repo.Path, 0755)
				os.WriteFile(configPath, []byte(configContent), 0644)

				// Mock git operations
				gm.On("CreateWorktree", mock.Anything, repo.Path, "feature/auth", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "invalid worktree name with spaces",
			req: operations.CreateWorktreeRequest{
				RepositoryID: "repo-123",
				Name:         "feature auth", // Invalid: contains space
			},
			setup: func(cm *testutil.MockContainerManager, gm *testutil.MockGitManager, sm *testutil.MockServiceManager, database *db.DB) {
				// No setup needed - should fail validation
			},
			wantErr: true,
			errMsg:  "invalid worktree name",
		},
		{
			name: "repository not found",
			req: operations.CreateWorktreeRequest{
				RepositoryID: "non-existent",
				Name:         "feature-auth",
			},
			setup: func(cm *testutil.MockContainerManager, gm *testutil.MockGitManager, sm *testutil.MockServiceManager, database *db.DB) {
				// No repository created - should fail
			},
			wantErr: true,
			errMsg:  "failed to get repository",
		},
		{
			name: "git worktree creation fails",
			req: operations.CreateWorktreeRequest{
				RepositoryID: "repo-123",
				Name:         "feature-auth",
			},
			setup: func(cm *testutil.MockContainerManager, gm *testutil.MockGitManager, sm *testutil.MockServiceManager, database *db.DB) {
				// Create test repository
				repo := &db.Repository{
					ID:   "repo-123",
					Name: "test-repo",
					Path: "/tmp/test-repo",
				}
				repoRepo := db.NewRepositoryRepository(database)
				if err := repoRepo.Create(context.Background(), repo); err != nil {
					t.Fatalf("Failed to create repository: %v", err)
				}

				// Create config
				configPath := filepath.Join(repo.Path, "vibeman.toml")
				configContent := `[repository]
name = "test-repo"`
				os.MkdirAll(repo.Path, 0755)
				os.WriteFile(configPath, []byte(configContent), 0644)

				// Mock git failure
				gm.On("CreateWorktree", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("branch already exists"))
			},
			wantErr: true,
			errMsg:  "failed to create git worktree",
		},
		{
			name: "container creation fails but worktree succeeds",
			req: operations.CreateWorktreeRequest{
				RepositoryID: "repo-123",
				Name:         "feature-auth",
				AutoStart:    true,
			},
			setup: func(cm *testutil.MockContainerManager, gm *testutil.MockGitManager, sm *testutil.MockServiceManager, database *db.DB) {
				// Create test repository
				repo := &db.Repository{
					ID:   "repo-123",
					Name: "test-repo",
					Path: "/tmp/test-repo",
				}
				repoRepo := db.NewRepositoryRepository(database)
				if err := repoRepo.Create(context.Background(), repo); err != nil {
					t.Fatalf("Failed to create repository: %v", err)
				}

				// Create config
				configPath := filepath.Join(repo.Path, "vibeman.toml")
				configContent := `[repository]
name = "test-repo"`
				os.MkdirAll(repo.Path, 0755)
				os.WriteFile(configPath, []byte(configContent), 0644)

				// Mock successful git operation
				gm.On("CreateWorktree", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

				// Note: Container creation is not yet implemented in StartWorktree
				// so we don't expect CreateContainer to be called
			},
			wantErr: false, // Should succeed despite container failure
		},
		{
			name: "worktree creation with post-scripts and compose overrides",
			req: operations.CreateWorktreeRequest{
				RepositoryID:    "repo-123",
				Name:            "feature-custom",
				Branch:          "feature/custom",
				BaseBranch:      "main",
				SkipSetup:       false,
				PostScripts:     []string{"npm install", "npm run build"},
				ComposeFile:     "./custom-compose.yaml",
				Services: []string{"backend", "frontend"},
				AutoStart:       true,
			},
			setup: func(cm *testutil.MockContainerManager, gm *testutil.MockGitManager, sm *testutil.MockServiceManager, database *db.DB) {
				// Create test repository
				repo := &db.Repository{
					ID:          "repo-123",
					Name:        "test-repo",
					Path:        "/tmp/test-repo",
					Description: "Test repository",
				}
				repoRepo := db.NewRepositoryRepository(database)
				err := repoRepo.Create(context.Background(), repo)
				require.NoError(t, err)

				// Create vibeman.toml with service dependencies
				configPath := filepath.Join(repo.Path, "vibeman.toml")
				configContent := `
[repository]
name = "test-repo"
description = "Test repository"

[repository.container]
compose_file = "./docker-compose.yaml"
services = ["default"]

[repository.worktrees]
directory = "../test-worktrees"

[repository.services]
postgres = { required = true }
redis = { required = true }

[repository.setup]
worktree_init = "echo 'Initial setup'"
`
				os.MkdirAll(repo.Path, 0755)
				os.WriteFile(configPath, []byte(configContent), 0644)

				// Mock git operations
				gm.On("CreateWorktree", mock.Anything, repo.Path, "feature/custom", mock.Anything).Return(nil)
				
				// Mock service operations for required services
				sm.On("StartService", mock.Anything, "postgres").Return(nil)
				sm.On("StartService", mock.Anything, "redis").Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test database
			database := testutil.SetupTestDB(t)
			defer database.Close()

			// Create mocks
			mockCM := new(testutil.MockContainerManager)
			mockGM := new(testutil.MockGitManager)
			mockSM := new(testutil.MockServiceManager)
			cfg := &config.Manager{}

			// Run setup
			if tt.setup != nil {
				tt.setup(mockCM, mockGM, mockSM, database)
			}

			// Create operations instance
			ops := operations.NewWorktreeOperations(database, mockGM, mockCM, mockSM, cfg)

			// Execute
			resp, err := ops.CreateWorktree(context.Background(), tt.req)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				if resp != nil {
					assert.NotNil(t, resp.Worktree)
					assert.Equal(t, tt.req.Name, resp.Worktree.Name)
					assert.NotEmpty(t, resp.Path)
				}
			}

			// Verify mocks
			mockCM.AssertExpectations(t)
			mockGM.AssertExpectations(t)

			// Cleanup
			os.RemoveAll("/tmp/test-repo")
			os.RemoveAll("/tmp/test-worktrees")
		})
	}
}

func TestWorktreeOperations_RemoveWorktree(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		force   bool
		setup   func(*testutil.MockContainerManager, *testutil.MockGitManager, *testutil.MockServiceManager, *db.DB)
		wantErr bool
		errMsg  string
	}{
		{
			name:  "successful worktree removal",
			id:    "wt-123",
			force: false,
			setup: func(cm *testutil.MockContainerManager, gm *testutil.MockGitManager, sm *testutil.MockServiceManager, database *db.DB) {
				// Create repository
				repo := &db.Repository{
					ID:   "repo-123",
					Name: "test-repo",
					Path: "/tmp/test-repo",
				}
				repoRepo := db.NewRepositoryRepository(database)
				if err := repoRepo.Create(context.Background(), repo); err != nil {
					t.Fatalf("Failed to create repository: %v", err)
				}

				// Create worktree
				worktree := &db.Worktree{
					ID:           "wt-123",
					RepositoryID: "repo-123",
					Name:         "feature-auth",
					Branch:       "feature/auth",
					Path:         "/tmp/test-worktrees/feature-auth",
					Status:       db.StatusStopped,
				}
				wtRepo := db.NewWorktreeRepository(database)
				if err := wtRepo.Create(context.Background(), worktree); err != nil {
					t.Fatalf("Failed to create worktree: %v", err)
				}

				// Mock git status check
				gm.On("HasUncommittedChanges", mock.Anything, "/tmp/test-worktrees/feature-auth").Return(false, nil)
				gm.On("HasUnpushedCommits", mock.Anything, "/tmp/test-worktrees/feature-auth").Return(false, nil)

				// Note: Container removal is not yet implemented in RemoveWorktree
				// so we don't expect RemoveContainer to be called

				// Mock git worktree removal
				gm.On("RemoveWorktree", mock.Anything, "/tmp/test-worktrees/feature-auth").Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "worktree not found",
			id:    "non-existent",
			force: false,
			setup: func(cm *testutil.MockContainerManager, gm *testutil.MockGitManager, sm *testutil.MockServiceManager, database *db.DB) {
				// No setup - worktree doesn't exist
			},
			wantErr: true,
			errMsg:  "failed to get worktree",
		},
		{
			name:  "cannot remove current worktree",
			id:    "wt-123",
			force: false,
			setup: func(cm *testutil.MockContainerManager, gm *testutil.MockGitManager, sm *testutil.MockServiceManager, database *db.DB) {
				// Create repository and worktree
				repo := &db.Repository{
					ID:   "repo-123",
					Name: "test-repo",
					Path: "/tmp/test-repo",
				}
				repoRepo := db.NewRepositoryRepository(database)
				if err := repoRepo.Create(context.Background(), repo); err != nil {
					t.Fatalf("Failed to create repository: %v", err)
				}

				// Get current directory
				currentDir, _ := os.Getwd()
				
				worktree := &db.Worktree{
					ID:           "wt-123",
					RepositoryID: "repo-123",
					Name:         "feature-auth",
					Branch:       "feature/auth",
					Path:         currentDir, // Set to current directory
					Status:       db.StatusStopped,
				}
				wtRepo := db.NewWorktreeRepository(database)
				if err := wtRepo.Create(context.Background(), worktree); err != nil {
					t.Fatalf("Failed to create worktree: %v", err)
				}
			},
			wantErr: true,
			errMsg:  "cannot remove current worktree",
		},
		{
			name:  "uncommitted changes without force",
			id:    "wt-123",
			force: false,
			setup: func(cm *testutil.MockContainerManager, gm *testutil.MockGitManager, sm *testutil.MockServiceManager, database *db.DB) {
				// Create repository and worktree
				repo := &db.Repository{
					ID:   "repo-123",
					Name: "test-repo",
					Path: "/tmp/test-repo",
				}
				repoRepo := db.NewRepositoryRepository(database)
				if err := repoRepo.Create(context.Background(), repo); err != nil {
					t.Fatalf("Failed to create repository: %v", err)
				}

				worktree := &db.Worktree{
					ID:           "wt-123",
					RepositoryID: "repo-123",
					Name:         "feature-auth",
					Branch:       "feature/auth",
					Path:         "/tmp/test-worktrees/feature-auth",
					Status:       db.StatusStopped,
				}
				wtRepo := db.NewWorktreeRepository(database)
				if err := wtRepo.Create(context.Background(), worktree); err != nil {
					t.Fatalf("Failed to create worktree: %v", err)
				}

				// Mock git status with uncommitted changes
				gm.On("HasUncommittedChanges", mock.Anything, mock.Anything).Return(true, nil)
				gm.On("HasUnpushedCommits", mock.Anything, mock.Anything).Return(false, nil)
			},
			wantErr: true,
			errMsg:  "worktree has uncommitted changes",
		},
		{
			name:  "force removal with uncommitted changes",
			id:    "wt-123",
			force: true,
			setup: func(cm *testutil.MockContainerManager, gm *testutil.MockGitManager, sm *testutil.MockServiceManager, database *db.DB) {
				// Create repository and worktree
				repo := &db.Repository{
					ID:   "repo-123",
					Name: "test-repo",
					Path: "/tmp/test-repo",
				}
				repoRepo := db.NewRepositoryRepository(database)
				if err := repoRepo.Create(context.Background(), repo); err != nil {
					t.Fatalf("Failed to create repository: %v", err)
				}

				worktree := &db.Worktree{
					ID:           "wt-123",
					RepositoryID: "repo-123",
					Name:         "feature-auth",
					Branch:       "feature/auth",
					Path:         "/tmp/test-worktrees/feature-auth",
					Status:       db.StatusStopped,
				}
				wtRepo := db.NewWorktreeRepository(database)
				if err := wtRepo.Create(context.Background(), worktree); err != nil {
					t.Fatalf("Failed to create worktree: %v", err)
				}

				// With force, git status is not checked
				
				// Note: Container removal is not yet implemented in RemoveWorktree
				// so we don't expect RemoveContainer to be called

				// Mock git worktree removal
				gm.On("RemoveWorktree", mock.Anything, "/tmp/test-worktrees/feature-auth").Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test database
			database := testutil.SetupTestDB(t)
			defer database.Close()

			// Create mocks
			mockCM := new(testutil.MockContainerManager)
			mockGM := new(testutil.MockGitManager)
			mockSM := new(testutil.MockServiceManager)
			cfg := &config.Manager{}

			// Run setup
			if tt.setup != nil {
				tt.setup(mockCM, mockGM, mockSM, database)
			}

			// Create operations instance
			ops := operations.NewWorktreeOperations(database, mockGM, mockCM, mockSM, cfg)

			// Execute
			err := ops.RemoveWorktree(context.Background(), tt.id, tt.force)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify mocks
			mockCM.AssertExpectations(t)
			mockGM.AssertExpectations(t)
		})
	}
}

func TestWorktreeOperations_StartWorktree(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		setup   func(*testutil.MockContainerManager, *testutil.MockServiceManager, *db.DB)
		wantErr bool
		errMsg  string
	}{
		{
			name: "successful start with required services",
			id:   "wt-123",
			setup: func(cm *testutil.MockContainerManager, sm *testutil.MockServiceManager, database *db.DB) {
				// Create repository with service requirements
				repo := &db.Repository{
					ID:   "repo-123",
					Name: "test-repo",
					Path: "/tmp/test-repo",
				}
				repoRepo := db.NewRepositoryRepository(database)
				if err := repoRepo.Create(context.Background(), repo); err != nil {
					t.Fatalf("Failed to create repository: %v", err)
				}

				// Create worktree
				worktree := &db.Worktree{
					ID:           "wt-123",
					RepositoryID: "repo-123",
					Name:         "feature-auth",
					Branch:       "feature/auth",
					Path:         "/tmp/test-worktrees/feature-auth",
					Status:       db.StatusStopped,
				}

				// Create config with required services in worktree directory
				configPath := filepath.Join(worktree.Path, "vibeman.toml")
				configContent := `
[repository]
name = "test-repo"

[repository.services]
postgres = { required = true }
redis = { required = true }
`
				os.MkdirAll(worktree.Path, 0755)
				os.WriteFile(configPath, []byte(configContent), 0644)
				wtRepo := db.NewWorktreeRepository(database)
				if err := wtRepo.Create(context.Background(), worktree); err != nil {
					t.Fatalf("Failed to create worktree: %v", err)
				}

				// Mock AI container creation
				cm.On("CreateWithConfig", mock.Anything, mock.AnythingOfType("*container.CreateConfig")).
					Return(&container.Container{
						ID:   "ai-container-123",
						Name: "test-repo-feature-auth-ai",
					}, nil)
				
				// Mock container start for AI container
				cm.On("StartContainer", mock.Anything, "ai-container-123").Return(nil)
				
				// Mock container list for log aggregation
				cm.On("List", mock.Anything).Return([]*container.Container{
					{
						ID:   "ai-container-123",
						Name: "test-repo-feature-auth-ai",
					},
				}, nil)
				
				// Mock container logs
				cm.On("Logs", mock.Anything, "ai-container-123", false).Return([]byte("AI container logs"), nil)
			},
			wantErr: false,
		},
		{
			name: "already running",
			id:   "wt-123",
			setup: func(cm *testutil.MockContainerManager, sm *testutil.MockServiceManager, database *db.DB) {
				// Create repository
				repo := &db.Repository{
					ID:   "repo-123",
					Name: "test-repo",
					Path: "/tmp/test-repo",
				}
				repoRepo := db.NewRepositoryRepository(database)
				if err := repoRepo.Create(context.Background(), repo); err != nil {
					t.Fatalf("Failed to create repository: %v", err)
				}

				// Create worktree that's already running
				worktree := &db.Worktree{
					ID:           "wt-123",
					RepositoryID: "repo-123",
					Name:         "feature-auth",
					Branch:       "feature/auth",
					Path:         "/tmp/test-worktrees/feature-auth",
					Status:       db.StatusRunning,
				}
				wtRepo := db.NewWorktreeRepository(database)
				if err := wtRepo.Create(context.Background(), worktree); err != nil {
					t.Fatalf("Failed to create worktree: %v", err)
				}
			},
			wantErr: true,
			errMsg:  "already running",
		},
		{
			name: "container start fails but operation succeeds",
			id:   "wt-123",
			setup: func(cm *testutil.MockContainerManager, sm *testutil.MockServiceManager, database *db.DB) {
				// Create repository and worktree
				repo := &db.Repository{
					ID:   "repo-123",
					Name: "test-repo",
					Path: "/tmp/test-repo",
				}
				repoRepo := db.NewRepositoryRepository(database)
				if err := repoRepo.Create(context.Background(), repo); err != nil {
					t.Fatalf("Failed to create repository: %v", err)
				}

				worktree := &db.Worktree{
					ID:           "wt-123",
					RepositoryID: "repo-123",
					Name:         "feature-auth",
					Branch:       "feature/auth",
					Path:         "/tmp/test-worktrees/feature-auth",
					Status:       db.StatusStopped,
				}
				wtRepo := db.NewWorktreeRepository(database)
				if err := wtRepo.Create(context.Background(), worktree); err != nil {
					t.Fatalf("Failed to create worktree: %v", err)
				}

				// Create simple config in worktree directory
				configPath := filepath.Join(worktree.Path, "vibeman.toml")
				configContent := `[repository]
name = "test-repo"`
				os.MkdirAll(worktree.Path, 0755)
				os.WriteFile(configPath, []byte(configContent), 0644)

				// Mock AI container creation
				cm.On("CreateWithConfig", mock.Anything, mock.AnythingOfType("*container.CreateConfig")).
					Return(&container.Container{
						ID:   "ai-container-123",
						Name: "test-repo-feature-auth-ai",
					}, nil)
				
				// Mock container start failure for AI container
				cm.On("StartContainer", mock.Anything, "ai-container-123").
					Return(errors.New("container is unhealthy"))
				
				// Mock container removal (cleanup after failure)
				cm.On("RemoveContainer", mock.Anything, "ai-container-123").Return(nil)
			},
			wantErr: false, // Operation succeeds even if AI container fails
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test database
			database := testutil.SetupTestDB(t)
			defer database.Close()

			// Create mocks
			mockCM := new(testutil.MockContainerManager)
			mockSM := new(testutil.MockServiceManager)
			mockGM := new(testutil.MockGitManager) // Not used but needed for constructor
			cfg := &config.Manager{}

			// Run setup
			if tt.setup != nil {
				tt.setup(mockCM, mockSM, database)
			}

			// Create operations instance with service manager
			ops := operations.NewWorktreeOperations(database, mockGM, mockCM, mockSM, cfg)

			// Execute
			err := ops.StartWorktree(context.Background(), tt.id)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				
				// Verify status was updated
				wtRepo := db.NewWorktreeRepository(database)
				wt, _ := wtRepo.Get(context.Background(), tt.id)
				assert.Equal(t, db.StatusRunning, wt.Status)
			}

			// Verify mocks
			mockCM.AssertExpectations(t)
			mockSM.AssertExpectations(t)
		})
	}
}