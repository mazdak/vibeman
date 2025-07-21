import { test, expect, describe, beforeEach, afterEach } from 'bun:test';
import { client } from '@/generated/api';
import { cleanupTestData } from '../setup';

describe('Logs API Integration', () => {
  let testRepoId: string | undefined;
  let testWorktreeId: string | undefined;

  beforeEach(async () => {
    await cleanupTestData();
    
    // Create test repository and worktree for log tests
    const repoResponse = await client.POST('/repositories', {
      body: {
        name: 'test-logs-repo',
        path: `/tmp/test-logs-repo-${Date.now()}`,
      },
    });
    
    expect(repoResponse.response.status).toBe(201);
    testRepoId = repoResponse.data?.repository.id;

    const worktreeResponse = await client.POST('/worktrees', {
      body: {
        repository_id: testRepoId!,
        name: 'test-logs-worktree',
        skip_setup: true,
      },
    });
    
    expect(worktreeResponse.response.status).toBe(201);
    testWorktreeId = worktreeResponse.data?.worktree.id;
  });

  afterEach(async () => {
    // Clean up
    if (testWorktreeId) {
      try {
        await client.DELETE('/worktrees/{id}', {
          params: { path: { id: testWorktreeId } },
        });
      } catch (error) {
        // Ignore
      }
    }
    
    if (testRepoId) {
      try {
        await client.DELETE('/repositories/{id}', {
          params: { path: { id: testRepoId } },
        });
      } catch (error) {
        // Ignore
      }
    }
  });

  describe('GET /worktrees/{id}/logs', () => {
    test('should get worktree logs', async () => {
      const response = await client.GET('/worktrees/{id}/logs', {
        params: { path: { id: testWorktreeId! } },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data).toBeDefined();
      expect(Array.isArray(response.data?.logs)).toBe(true);
    });

    test('should support tail parameter', async () => {
      const response = await client.GET('/worktrees/{id}/logs', {
        params: { 
          path: { id: testWorktreeId! },
          query: { tail: 20 },
        },
      });
      
      expect(response.response.status).toBe(200);
      expect(Array.isArray(response.data?.logs)).toBe(true);
      if (response.data?.logs && response.data.logs.length > 0) {
        expect(response.data.logs.length).toBeLessThanOrEqual(20);
      }
    });

    test('should support follow parameter', async () => {
      // This test just verifies the endpoint accepts the parameter
      // Actually testing streaming would require WebSocket support
      const response = await client.GET('/worktrees/{id}/logs', {
        params: { 
          path: { id: testWorktreeId! },
          query: { follow: true, tail: 5 },
        },
      });
      
      expect(response.response.status).toBe(200);
    });

    test('should support container filter', async () => {
      const response = await client.GET('/worktrees/{id}/logs', {
        params: { 
          path: { id: testWorktreeId! },
          query: { container: 'ai' },
        },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data).toBeDefined();
    });

    test('should return 404 for non-existent worktree', async () => {
      const response = await client.GET('/worktrees/{id}/logs', {
        params: { path: { id: 'non-existent-id' } },
      });
      
      expect(response.response.status).toBe(404);
    });
  });

  describe('POST /worktrees/{id}/logs/search', () => {
    test('should search logs with query', async () => {
      const response = await client.POST('/worktrees/{id}/logs/search', {
        params: { path: { id: testWorktreeId! } },
        body: {
          query: 'error',
          case_sensitive: false,
        },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data).toBeDefined();
      expect(Array.isArray(response.data?.results)).toBe(true);
      expect(response.data).toHaveProperty('total_matches');
    });

    test('should support regex search', async () => {
      const response = await client.POST('/worktrees/{id}/logs/search', {
        params: { path: { id: testWorktreeId! } },
        body: {
          query: 'error|warning',
          regex: true,
          case_sensitive: false,
        },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data).toBeDefined();
      expect(Array.isArray(response.data?.results)).toBe(true);
    });

    test('should support time range filter', async () => {
      const now = new Date();
      const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000);
      
      const response = await client.POST('/worktrees/{id}/logs/search', {
        params: { path: { id: testWorktreeId! } },
        body: {
          query: '*',
          start_time: oneHourAgo.toISOString(),
          end_time: now.toISOString(),
        },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data).toBeDefined();
    });

    test('should support container filter in search', async () => {
      const response = await client.POST('/worktrees/{id}/logs/search', {
        params: { path: { id: testWorktreeId! } },
        body: {
          query: 'test',
          containers: ['ai'],
        },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data).toBeDefined();
    });

    test('should support limit parameter', async () => {
      const response = await client.POST('/worktrees/{id}/logs/search', {
        params: { path: { id: testWorktreeId! } },
        body: {
          query: '*',
          limit: 10,
        },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data?.results).toBeDefined();
      if (response.data?.results && response.data.results.length > 0) {
        expect(response.data.results.length).toBeLessThanOrEqual(10);
      }
    });

    test('should return 400 for invalid regex', async () => {
      const response = await client.POST('/worktrees/{id}/logs/search', {
        params: { path: { id: testWorktreeId! } },
        body: {
          query: '[invalid regex',
          regex: true,
        },
      });
      
      expect(response.response.status).toBe(400);
      expect(response.error?.error).toContain('regex');
    });

    test('should return 404 for non-existent worktree', async () => {
      const response = await client.POST('/worktrees/{id}/logs/search', {
        params: { path: { id: 'non-existent-id' } },
        body: {
          query: 'test',
        },
      });
      
      expect(response.response.status).toBe(404);
    });
  });

  describe('DELETE /worktrees/{id}/logs', () => {
    test('should clear worktree logs', async () => {
      const response = await client.DELETE('/worktrees/{id}/logs', {
        params: { path: { id: testWorktreeId! } },
      });
      
      expect(response.response.status).toBe(204);
      
      // Verify logs are cleared
      const logsResponse = await client.GET('/worktrees/{id}/logs', {
        params: { path: { id: testWorktreeId! } },
      });
      
      expect(logsResponse.response.status).toBe(200);
      expect(logsResponse.data?.logs).toEqual([]);
    });

    test('should support container filter for deletion', async () => {
      const response = await client.DELETE('/worktrees/{id}/logs', {
        params: { 
          path: { id: testWorktreeId! },
          query: { container: 'ai' },
        },
      });
      
      expect(response.response.status).toBe(204);
    });

    test('should return 404 for non-existent worktree', async () => {
      const response = await client.DELETE('/worktrees/{id}/logs', {
        params: { path: { id: 'non-existent-id' } },
      });
      
      expect(response.response.status).toBe(404);
    });
  });

  describe('GET /logs/aggregated', () => {
    test('should get aggregated logs from all sources', async () => {
      const response = await client.GET('/logs/aggregated');
      
      expect(response.response.status).toBe(200);
      expect(response.data).toBeDefined();
      expect(Array.isArray(response.data?.logs)).toBe(true);
      expect(response.data).toHaveProperty('sources');
    });

    test('should support filtering by source', async () => {
      const response = await client.GET('/logs/aggregated', {
        params: {
          query: {
            sources: ['worktree', 'service'],
          },
        },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data).toBeDefined();
    });

    test('should support time range filter', async () => {
      const now = new Date();
      const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000);
      
      const response = await client.GET('/logs/aggregated', {
        params: {
          query: {
            start_time: oneHourAgo.toISOString(),
            end_time: now.toISOString(),
          },
        },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data).toBeDefined();
    });

    test('should support pagination', async () => {
      const response = await client.GET('/logs/aggregated', {
        params: {
          query: {
            limit: 50,
            offset: 0,
          },
        },
      });
      
      expect(response.response.status).toBe(200);
      expect(response.data?.logs).toBeDefined();
      if (response.data?.logs && response.data.logs.length > 0) {
        expect(response.data.logs.length).toBeLessThanOrEqual(50);
      }
    });
  });
});