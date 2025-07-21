import { test, expect, describe, beforeEach, afterEach } from 'bun:test';
import {
  getWorktrees,
  postWorktrees,
  getWorktreesById,
  postWorktreesByIdStart,
  postWorktreesByIdStop,
  getApiWorktreesByIdLogs,
  postRepositories,
  deleteRepositoriesById
} from '@/generated/api';
import { cleanupTestData } from '../setup';
import type { DbWorktree, DbRepository } from '@/generated/api/types.gen';

describe('Worktrees API Integration', () => {
  let testRepoId: string | undefined;
  let testWorktreeId: string | undefined;

  beforeEach(async () => {
    await cleanupTestData();
    
    // Create a test repository for worktree tests
    const repoResponse = await postRepositories({
      body: {
        name: 'test-worktree-repo',
        path: `/tmp/test-worktree-repo-${Date.now()}`,
      },
    });
    
    expect(repoResponse.response.status).toBe(201);
    testRepoId = repoResponse.data?.repository.id;
  });

  afterEach(async () => {
    // Clean up repository (which should clean up worktrees)
    if (testRepoId) {
      try {
        await deleteRepositoriesById({
          path: { id: testRepoId },
        });
      } catch (error) {
        // Ignore cleanup errors
      }
      testRepoId = undefined;
    }
  });

  describe('POST /worktrees', () => {
    test('should create a new worktree', async () => {
      const response = await postWorktrees({
        body: {
          repository_id: testRepoId!,
          name: 'test-feature-branch',
          branch: 'feature/test-branch',
          skip_setup: true,
        },
      });

      expect(response.response.status).toBe(201);
      expect(response.data).toBeDefined();
      expect(response.data?.worktree).toBeDefined();
      expect(response.data?.worktree.name).toBe('test-feature-branch');
      expect(response.data?.worktree.branch).toBe('feature/test-branch');
      expect(response.data?.worktree.repository_id).toBe(testRepoId);
      
      testWorktreeId = response.data?.worktree.id;
    });

    test('should create worktree with auto-generated branch name', async () => {
      const response = await postWorktrees({
        body: {
          repository_id: testRepoId!,
          name: 'auto-branch-test',
          skip_setup: true,
        },
      });

      expect(response.response.status).toBe(201);
      expect(response.data?.worktree.branch).toBe('auto-branch-test');
      
      testWorktreeId = response.data?.worktree.id;
    });

    test('should return 400 for invalid input', async () => {
      const response = await postWorktrees({
        body: {
          repository_id: testRepoId!,
          name: '', // Empty name should be invalid
        },
      });

      expect(response.response.status).toBe(400);
    });

    test('should return 404 for non-existent repository', async () => {
      const response = await postWorktrees({
        body: {
          repository_id: 'non-existent-repo',
          name: 'test-worktree',
        },
      });

      expect(response.response.status).toBe(404);
    });
  });

  describe('GET /worktrees', () => {
    test('should list all worktrees', async () => {
      // Create a test worktree
      const createResponse = await postWorktrees({
        body: {
          repository_id: testRepoId!,
          name: 'list-test-worktree',
          skip_setup: true,
        },
      });
      
      expect(createResponse.response.status).toBe(201);
      testWorktreeId = createResponse.data?.worktree.id;

      // List worktrees
      const response = await getWorktrees();
      
      expect(response.response.status).toBe(200);
      expect(response.data).toBeDefined();
      expect(Array.isArray(response.data?.worktrees)).toBe(true);
      
      const worktrees = response.data?.worktrees || [];
      const testWorktree = worktrees.find(w => w.name === 'list-test-worktree');
      expect(testWorktree).toBeDefined();
    });

    test('should filter worktrees by repository', async () => {
      // Create a worktree
      const createResponse = await postWorktrees({
        body: {
          repository_id: testRepoId!,
          name: 'filter-test-worktree',
          skip_setup: true,
        },
      });
      
      expect(createResponse.response.status).toBe(201);
      testWorktreeId = createResponse.data?.worktree.id;

      // List worktrees filtered by repository
      const response = await getWorktrees({
        query: { repository_id: testRepoId! },
      });
      
      expect(response.response.status).toBe(200);
      const worktrees = response.data?.worktrees || [];
      expect(worktrees.length).toBeGreaterThan(0);
      expect(worktrees.every(w => w.repository_id === testRepoId)).toBe(true);
    });

    test('should return empty array when no worktrees exist', async () => {
      const response = await getWorktrees({
        query: { repository_id: 'non-existent-repo' },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data?.worktrees).toEqual([]);
    });
  });

  describe('GET /worktrees/{id}', () => {
    test('should get worktree by id', async () => {
      // Create a test worktree
      const createResponse = await postWorktrees({
        body: {
          repository_id: testRepoId!,
          name: 'get-test-worktree',
          skip_setup: true,
        },
      });
      
      expect(createResponse.response.status).toBe(201);
      testWorktreeId = createResponse.data?.worktree.id;

      // Get the worktree
      const response = await getWorktreesById({
        path: { id: testWorktreeId! },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data).toBeDefined();
      expect(response.data?.name).toBe('get-test-worktree');
      expect(response.data?.repository_id).toBe(testRepoId);
    });

    test('should return 404 for non-existent worktree', async () => {
      const response = await getWorktreesById({
        path: { id: 'non-existent-id' },
      });
      
      expect(response.response.status).toBe(404);
    });
  });

  describe('POST /worktrees/{id}/start', () => {
    test('should start worktree container', async () => {
      // Create a test worktree
      const createResponse = await postWorktrees({
        body: {
          repository_id: testRepoId!,
          name: 'start-test-worktree',
          skip_setup: true,
        },
      });
      
      expect(createResponse.response.status).toBe(201);
      testWorktreeId = createResponse.data?.worktree.id;

      // Start the worktree
      const response = await postWorktreesByIdStart({
        path: { id: testWorktreeId! },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data?.message).toContain('started');
    });

    test('should return 404 for non-existent worktree', async () => {
      const response = await postWorktreesByIdStart({
        path: { id: 'non-existent-id' },
      });
      
      expect(response.response.status).toBe(404);
    });
  });

  describe('POST /worktrees/{id}/stop', () => {
    test('should stop worktree container', async () => {
      // Create and start a test worktree
      const createResponse = await postWorktrees({
        body: {
          repository_id: testRepoId!,
          name: 'stop-test-worktree',
          skip_setup: true,
          auto_start: true,
        },
      });
      
      expect(createResponse.response.status).toBe(201);
      testWorktreeId = createResponse.data?.worktree.id;

      // Stop the worktree
      const response = await postWorktreesByIdStop({
        path: { id: testWorktreeId! },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data?.message).toContain('stopped');
    });

    test('should return 404 for non-existent worktree', async () => {
      const response = await postWorktreesByIdStop({
        path: { id: 'non-existent-id' },
      });
      
      expect(response.response.status).toBe(404);
    });
  });

  describe('GET /worktrees/{id}/logs', () => {
    test('should get worktree logs', async () => {
      // Create a test worktree
      const createResponse = await postWorktrees({
        body: {
          repository_id: testRepoId!,
          name: 'logs-test-worktree',
          skip_setup: true,
        },
      });
      
      expect(createResponse.response.status).toBe(201);
      testWorktreeId = createResponse.data?.worktree.id;

      // Get worktree logs
      const response = await getApiWorktreesByIdLogs({
        path: { id: testWorktreeId! },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data).toBeDefined();
      expect(Array.isArray(response.data?.logs)).toBe(true);
    });

    test('should support tail parameter', async () => {
      // Create a test worktree
      const createResponse = await postWorktrees({
        body: {
          repository_id: testRepoId!,
          name: 'logs-tail-worktree',
          skip_setup: true,
        },
      });
      
      expect(createResponse.response.status).toBe(201);
      testWorktreeId = createResponse.data?.worktree.id;

      const response = await getApiWorktreesByIdLogs({
        path: { id: testWorktreeId! },
        query: { tail: 10 },
      });
      
      expect(response.response.status).toBe(200);
      expect(Array.isArray(response.data?.logs)).toBe(true);
    });

    test('should return 404 for non-existent worktree', async () => {
      const response = await getApiWorktreesByIdLogs({
        path: { id: 'non-existent-id' },
      });
      
      expect(response.response.status).toBe(404);
    });
  });
});