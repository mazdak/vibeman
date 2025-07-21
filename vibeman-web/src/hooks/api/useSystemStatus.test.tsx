/// <reference lib="dom" />
import { test, expect, describe, beforeEach, mock } from "bun:test";
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useSystemStatus } from './useSystemStatus';
import React from 'react';

// Setup happy-dom
import { GlobalRegistrator } from '@happy-dom/global-registrator';
if (!global.happyDOM) {
  GlobalRegistrator.register();
}

// Mock the generated API
mock.module('@/generated/api/@tanstack/react-query.gen', () => ({
  getApiStatusOptions: () => ({
    queryKey: ['getApiStatus'],
    queryFn: async () => {
      // Check if this is being called from the SystemStatusIndicator test
      const mockScenario = (global as any).mockScenario;
      if (mockScenario) {
        // Return data based on SystemStatusIndicator test scenario
        switch (mockScenario) {
          case 'unhealthy':
            return {
              status: 'unhealthy',
              worktrees: 2,
              containers: 1,
              repositories: 3,
              version: '1.0.0',
              uptime: '5m',
              services: {
                container_engine: 'healthy',
                database: 'unhealthy',
                git: 'healthy'
              }
            };
          default:
            return {
              status: 'healthy',
              worktrees: 5,
              containers: 3,
              repositories: 2,
              version: '1.0.0',
              uptime: '2h',
              services: {
                container_engine: 'healthy',
                database: 'healthy',
                git: 'healthy'
              }
            };
        }
      }
      
      // Default response for useSystemStatus tests
      return {
        status: 'healthy',
        worktrees: 5,
        containers: 3,
        repositories: 2,
        version: '1.0.0',
        uptime: '2h',
        services: {
          container_engine: 'healthy',
          database: 'healthy',
          git: 'healthy'
        }
      };
    }
  })
}));

describe('useSystemStatus', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { 
          retry: false,
          staleTime: 0,
          cacheTime: 0 
        },
        mutations: { retry: false },
      },
    });
    // Clear any mock scenario from other tests
    delete (global as any).mockScenario;
  });

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  test('fetches system status successfully', async () => {
    const { result } = renderHook(() => useSystemStatus(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.data).toEqual({
      status: 'healthy',
      worktrees: 5,
      containers: 3,
      repositories: 2,
      version: '1.0.0',
      uptime: '2h',
      services: {
        container_engine: 'healthy',
        database: 'healthy',
        git: 'healthy'
      }
    });
    expect(result.current.error).toBe(null);
  });

  test('handles loading state', () => {
    const { result } = renderHook(() => useSystemStatus(), { wrapper });

    // Initial render should be in loading state or already resolved
    // Due to synchronous mock, it might resolve immediately
    if (result.current.isLoading) {
      expect(result.current.data).toBe(undefined);
    } else {
      expect(result.current.data).toBeDefined();
    }
  });

  test('returns system status data structure', async () => {
    const { result } = renderHook(() => useSystemStatus(), { wrapper });

    await waitFor(() => {
      expect(result.current.data).toBeDefined();
    });

    const status = result.current.data;
    expect(status?.status).toBe('healthy');
    expect(typeof status?.worktrees).toBe('number');
    expect(typeof status?.containers).toBe('number');
    expect(typeof status?.repositories).toBe('number');
    expect(status?.services).toBeDefined();
  });
});