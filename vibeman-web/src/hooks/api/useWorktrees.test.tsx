/// <reference lib="dom" />
import { test, expect, describe, beforeEach, mock } from "bun:test";
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useWorktrees } from './useWorktrees';
import React from 'react';

// Setup happy-dom
import { GlobalRegistrator } from '@happy-dom/global-registrator';
if (!global.happyDOM) {
  GlobalRegistrator.register();
}

// Mock the generated API
mock.module('@/generated/api/@tanstack/react-query.gen', () => ({
  getWorktreesOptions: (options: any) => ({
    queryKey: ['getWorktrees', options],
    queryFn: async () => {
      // Mock response
      return {
        worktrees: [
          {
            id: '1',
            name: 'test-worktree',
            branch: 'main',
            status: 'running',
            repository_id: 'project-1',
          },
        ],
        total: 1,
      };
    },
  }),
  getWorktreesQueryKey: (options: any) => ['getWorktrees', options],
  postWorktreesMutation: () => ({
    mutationFn: async (data: any) => {
      // Mock create response
      return { id: '2', ...data };
    },
  }),
  postWorktreesByIdStartMutation: () => ({
    mutationFn: async () => {
      // Mock start response
      return { message: 'Started' };
    },
  }),
  postWorktreesByIdStopMutation: () => ({
    mutationFn: async () => {
      // Mock stop response
      return { message: 'Stopped' };
    },
  }),
}));

describe('useWorktrees', () => {
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

  test('fetches worktrees successfully', async () => {
    const { result } = renderHook(
      () => useWorktrees({ repositoryId: 'project-1' }),
      { wrapper }
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.worktrees).toHaveLength(1);
    expect(result.current.worktrees[0]).toEqual({
      id: '1',
      name: 'test-worktree',
      branch: 'main',
      status: 'running',
      repository_id: 'project-1',
      container_status: 'active',
      last_activity: expect.any(String),
      setup_type: 'script',
    });
    expect(result.current.error).toBe(null);
  });

  test('handles loading state', () => {
    const { result } = renderHook(
      () => useWorktrees({ repositoryId: 'project-1' }),
      { wrapper }
    );

    expect(result.current.isLoading).toBe(true);
    expect(result.current.worktrees).toEqual([]);
  });

  test('provides mutation functions', () => {
    const { result } = renderHook(
      () => useWorktrees({ repositoryId: 'project-1' }),
      { wrapper }
    );

    expect(result.current.createWorktree).toBeInstanceOf(Function);
    expect(result.current.startWorktree).toBeInstanceOf(Function);
    expect(result.current.stopWorktree).toBeInstanceOf(Function);
  });

  test('tracks mutation states', () => {
    const { result } = renderHook(
      () => useWorktrees({ repositoryId: 'project-1' }),
      { wrapper }
    );

    expect(result.current.isCreating).toBe(false);
    expect(result.current.isStarting).toBe(false);
    expect(result.current.isStopping).toBe(false);
  });
});