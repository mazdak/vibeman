import { test, expect, describe, beforeEach, afterEach } from 'bun:test';
import { 
  getRepositories, 
  postRepositories, 
  deleteRepositoriesById
} from '@/generated/api';
import { cleanupTestData } from '../setup';
import type { DbRepository } from '@/generated/api/types.gen';

describe('Repositories API Integration', () => {
  let testRepoId: string | undefined;

  beforeEach(async () => {
    await cleanupTestData();
  });

  afterEach(async () => {
    // Clean up any created repositories
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

  describe('POST /repositories', () => {
    test('should create a new repository', async () => {
      const tempDir = `/tmp/test-repo-${Date.now()}`;
      
      const response = await postRepositories({
        body: {
          name: 'test-repo',
          path: tempDir,
          git_url: 'https://github.com/test/repo.git',
        },
      });

      expect(response.response.status).toBe(201);
      expect(response.data).toBeDefined();
      expect(response.data?.repository).toBeDefined();
      expect(response.data?.repository.name).toBe('test-repo');
      expect(response.data?.repository.path).toBe(tempDir);
      
      testRepoId = response.data?.repository.id;
    });

    test('should return 400 for invalid input', async () => {
      const response = await postRepositories({
        body: {
          name: '', // Empty name should be invalid
          path: '/tmp/test',
        },
      });

      expect(response.response.status).toBe(400);
      expect(response.error).toBeDefined();
    });

    test('should return 409 for duplicate repository', async () => {
      const tempDir = `/tmp/test-repo-${Date.now()}`;
      
      // Create first repository
      const response1 = await postRepositories({
        body: {
          name: 'test-duplicate',
          path: tempDir,
        },
      });
      
      expect(response1.response.status).toBe(201);
      testRepoId = response1.data?.repository.id;

      // Try to create duplicate
      const response2 = await postRepositories({
        body: {
          name: 'test-duplicate',
          path: tempDir,
        },
      });

      expect(response2.response.status).toBe(409);
    });
  });

  describe('GET /repositories', () => {
    test('should list all repositories', async () => {
      // Create a test repository first
      const createResponse = await postRepositories({
        body: {
          name: 'test-list-repo',
          path: `/tmp/test-list-${Date.now()}`,
        },
      });
      
      expect(createResponse.response.status).toBe(201);
      testRepoId = createResponse.data?.repository.id;

      // List repositories
      const response = await getRepositories();
      
      expect(response.response.status).toBe(200);
      expect(response.data).toBeDefined();
      expect(Array.isArray(response.data?.repositories)).toBe(true);
      
      const repos = response.data?.repositories || [];
      const testRepo = repos.find(r => r.name === 'test-list-repo');
      expect(testRepo).toBeDefined();
    });

    test('should return empty array when no repositories exist', async () => {
      await cleanupTestData();
      
      const response = await getRepositories();
      
      expect(response.response.status).toBe(200);
      expect(response.data?.repositories).toEqual([]);
      expect(response.data?.total).toBe(0);
    });
  });

  describe('DELETE /repositories/{id}', () => {
    test('should delete repository', async () => {
      // Create a test repository
      const createResponse = await postRepositories({
        body: {
          name: 'test-delete-repo',
          path: `/tmp/test-delete-${Date.now()}`,
        },
      });
      
      expect(createResponse.response.status).toBe(201);
      const repoId = createResponse.data?.repository.id;

      // Delete the repository
      const response = await deleteRepositoriesById({
        path: { id: repoId! },
      });
      
      expect(response.response.status).toBe(204);

      // Clear testRepoId since it's already deleted
      testRepoId = undefined;
    });

    test('should return 404 for non-existent repository', async () => {
      const response = await deleteRepositoriesById({
        path: { id: 'non-existent-id' },
      });
      
      expect(response.response.status).toBe(404);
    });
  });
});