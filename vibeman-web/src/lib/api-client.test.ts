/// <reference lib="dom" />
import { test, expect, describe, beforeEach, mock } from "bun:test";
import { transformWorktree, transformRepository, createClientConfig } from './api-client';
import type { DbWorktree, DbRepository } from '@/generated/api/types.gen';

// Setup happy-dom
import { GlobalRegistrator } from '@happy-dom/global-registrator';
if (!global.happyDOM) {
  GlobalRegistrator.register();
}

describe('API Client', () => {
  describe('transformWorktree', () => {
    test('transforms DbWorktree to UIWorktree with all required fields', () => {
      const dbWorktree: DbWorktree = {
        id: 'wt-123',
        name: 'feature-branch',
        branch: 'feature/new-feature',
        repository_id: 'repo-456',
        path: '/path/to/worktree',
        status: 'running',
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-02T00:00:00Z',
      };

      const result = transformWorktree(dbWorktree);

      expect(result).toEqual({
        id: 'wt-123',
        name: 'feature-branch',
        branch: 'feature/new-feature',
        repository_id: 'repo-456',
        path: '/path/to/worktree',
        status: 'running',
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-02T00:00:00Z',
        container_status: 'active',
        last_activity: '2023-01-02T00:00:00Z',
        setup_type: 'script',
      });
    });

    test('handles missing updated_at by using created_at', () => {
      const dbWorktree: DbWorktree = {
        id: 'wt-123',
        name: 'test',
        branch: 'main',
        repository_id: 'repo-456',
        path: '/path',
        status: 'stopped',
        created_at: '2023-01-01T00:00:00Z',
      };

      const result = transformWorktree(dbWorktree);

      expect(result.last_activity).toBe('2023-01-01T00:00:00Z');
    });

    test('handles missing both timestamps by using current time', () => {
      const dbWorktree: DbWorktree = {
        id: 'wt-123',
        name: 'test',
        branch: 'main',
        repository_id: 'repo-456',
        path: '/path',
        status: 'stopped',
      };

      const result = transformWorktree(dbWorktree);

      expect(result.last_activity).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/);
    });

    test('maps different statuses to container_status correctly', () => {
      const testCases = [
        { status: 'running', expected: 'active' },
        { status: 'starting', expected: 'building' },
        { status: 'stopping', expected: 'building' },
        { status: 'error', expected: 'failed' },
        { status: 'stopped', expected: 'inactive' },
      ] as const;

      testCases.forEach(({ status, expected }) => {
        const dbWorktree: DbWorktree = {
          id: 'wt-123',
          name: 'test',
          branch: 'main',
          repository_id: 'repo-456',
          path: '/path',
          status,
          created_at: '2023-01-01T00:00:00Z',
        };

        const result = transformWorktree(dbWorktree);
        expect(result.container_status).toBe(expected);
      });
    });
  });

  describe('transformRepository', () => {
    test('transforms DbRepository to UIRepository with required fields', () => {
      const dbRepository: DbRepository = {
        id: 'repo-123',
        name: 'my-project',
        path: '/path/to/repo',
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-02T00:00:00Z',
        git_url: 'https://github.com/user/repo.git',
        description: 'A test repository',
      };

      const result = transformRepository(dbRepository);

      expect(result).toEqual({
        id: 'repo-123',
        name: 'my-project',
        path: '/path/to/repo',
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-02T00:00:00Z',
        git_url: 'https://github.com/user/repo.git',
        description: 'A test repository',
        repository_url: 'https://github.com/user/repo.git',
      });
    });

    test('uses git_url as repository_url when available', () => {
      const dbRepository: DbRepository = {
        id: 'repo-123',
        name: 'my-project',
        path: '/path/to/repo',
        created_at: '2023-01-01T00:00:00Z',
        git_url: 'https://github.com/user/repo.git',
      };

      const result = transformRepository(dbRepository);
      expect(result.repository_url).toBe('https://github.com/user/repo.git');
    });

    test('provides fallback repository_url when git_url is missing', () => {
      const dbRepository: DbRepository = {
        id: 'repo-123',
        name: 'my-project',
        path: '/path/to/repo',
        created_at: '2023-01-01T00:00:00Z',
      };

      const result = transformRepository(dbRepository);
      expect(result.repository_url).toBe('');
    });
  });

  describe('createClientConfig', () => {
    test('returns config with relative baseUrl', () => {
      const config = createClientConfig();

      expect(config).toEqual({
        baseUrl: '/api',
      });
    });

    test('uses relative URL for proxy compatibility', () => {
      // The baseUrl should be relative to go through the Bun dev server proxy
      const config = createClientConfig();
      
      // Should not be an absolute URL
      expect(config.baseUrl).not.toMatch(/^https?:\/\//);
      expect(config.baseUrl).toBe('/api');
    });
  });
});