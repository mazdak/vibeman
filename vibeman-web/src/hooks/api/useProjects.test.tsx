/// <reference lib="dom" />
import { test, expect, describe, beforeEach, mock } from "bun:test";
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useProjects } from './useProjects';
import React from 'react';

// Setup happy-dom
import { GlobalRegistrator } from '@happy-dom/global-registrator';
if (!global.happyDOM) {
  GlobalRegistrator.register();
}

// Mock the generated API
mock.module('@/generated/api/@tanstack/react-query.gen', () => ({
  getRepositoriesOptions: () => ({
    queryKey: ['getRepositories'],
    queryFn: async () => {
      return {
        repositories: [
          {
            id: 'repo-1',
            name: 'test-repo',
            path: '/path/to/repo',
            git_url: 'https://github.com/user/test-repo.git',
            created_at: '2023-01-01T00:00:00Z',
            description: 'A test repository',
          },
          {
            id: 'repo-2',
            name: 'another-repo',
            path: '/path/to/another',
            git_url: 'https://github.com/user/another-repo.git',
            created_at: '2023-01-02T00:00:00Z',
          },
        ],
        total: 2,
      };
    },
  }),
  postRepositoriesMutation: () => ({
    mutationFn: async (data: any) => {
      return { 
        id: 'repo-new', 
        name: data.name,
        path: data.path,
        git_url: data.git_url,
        created_at: new Date().toISOString(),
      };
    },
  }),
  deleteRepositoriesByIdMutation: () => ({
    mutationFn: async (options: any) => {
      return { message: 'Repository deleted' };
    },
  }),
}));

describe('useProjects', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
  });

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  test('fetches projects successfully', async () => {
    const { result } = renderHook(() => useProjects(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.projects).toHaveLength(2);
    expect(result.current.projects[0]).toEqual({
      id: 'repo-1',
      name: 'test-repo',
      path: '/path/to/repo',
      git_url: 'https://github.com/user/test-repo.git',
      created_at: '2023-01-01T00:00:00Z',
      description: 'A test repository',
      repository_url: 'https://github.com/user/test-repo.git',
    });
    expect(result.current.error).toBe(null);
  });

  test('handles loading state', () => {
    const { result } = renderHook(() => useProjects(), { wrapper });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.projects).toEqual([]);
  });

  test('provides mutation functions', () => {
    const { result } = renderHook(() => useProjects(), { wrapper });

    expect(result.current.createProject).toBeInstanceOf(Function);
    expect(result.current.deleteProject).toBeInstanceOf(Function);
  });

  test('tracks mutation states', () => {
    const { result } = renderHook(() => useProjects(), { wrapper });

    expect(result.current.isCreating).toBe(false);
    expect(result.current.isDeleting).toBe(false);
  });

  test('transforms repository data correctly', async () => {
    const { result } = renderHook(() => useProjects(), { wrapper });

    await waitFor(() => {
      expect(result.current.projects).toHaveLength(2);
    });

    // Check that repository_url is properly set from git_url
    expect(result.current.projects[0].repository_url).toBe('https://github.com/user/test-repo.git');
    expect(result.current.projects[1].repository_url).toBe('https://github.com/user/another-repo.git');
  });

  test('handles repositories without description', async () => {
    const { result } = renderHook(() => useProjects(), { wrapper });

    await waitFor(() => {
      expect(result.current.projects).toHaveLength(2);
    });

    // Second repository has no description
    expect(result.current.projects[1].description).toBeUndefined();
  });
});