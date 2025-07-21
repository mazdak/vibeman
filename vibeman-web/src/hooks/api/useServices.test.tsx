/// <reference lib="dom" />
import { test, expect, describe, beforeEach, mock } from "bun:test";
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useServices } from './useServices';
import React from 'react';

// Setup happy-dom
import { GlobalRegistrator } from '@happy-dom/global-registrator';
if (!global.happyDOM) {
  GlobalRegistrator.register();
}

// Mock the legacy API
mock.module('@/lib/legacy-api', () => ({
  legacyApi: {
    getServices: async () => [
      {
        id: 'svc-1',
        name: 'postgres',
        container_id: 'postgres_container',
        status: 'running',
        ref_count: 2,
        projects: ['project-1', 'project-2'],
        start_time: '2023-01-01T00:00:00Z',
        last_health: '2023-01-01T01:00:00Z',
        ports: [{ host_port: 5432, container_port: 5432, protocol: 'tcp' }],
      },
      {
        id: 'svc-2',
        name: 'redis',
        container_id: 'redis_container',
        status: 'stopped',
        ref_count: 1,
        projects: ['project-1'],
        start_time: '',
        last_health: '',
        ports: [],
      },
    ],
    startService: async (name: string) => {
      return { message: `Started ${name}` };
    },
    stopService: async (name: string) => {
      return { message: `Stopped ${name}` };
    },
    restartService: async (name: string) => {
      return { message: `Restarted ${name}` };
    },
  },
}));

describe('useServices', () => {
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

  test('fetches services successfully', async () => {
    const { result } = renderHook(() => useServices(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.services).toHaveLength(2);
    expect(result.current.services[0]).toEqual({
      id: 'svc-1',
      name: 'postgres',
      container_id: 'postgres_container',
      status: 'running',
      ref_count: 2,
      projects: ['project-1', 'project-2'],
      start_time: '2023-01-01T00:00:00Z',
      last_health: '2023-01-01T01:00:00Z',
      ports: [{ host_port: 5432, container_port: 5432, protocol: 'tcp' }],
    });
    expect(result.current.error).toBe(null);
  });

  test('handles loading state', () => {
    const { result } = renderHook(() => useServices(), { wrapper });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.services).toEqual([]);
  });

  test('filters services by status', async () => {
    const { result } = renderHook(
      () => useServices({ status: 'running' }),
      { wrapper }
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.services).toHaveLength(1);
    expect(result.current.services[0].status).toBe('running');
    expect(result.current.services[0].name).toBe('postgres');
  });

  test('provides mutation functions', () => {
    const { result } = renderHook(() => useServices(), { wrapper });

    expect(result.current.startService).toBeInstanceOf(Function);
    expect(result.current.stopService).toBeInstanceOf(Function);
    expect(result.current.restartService).toBeInstanceOf(Function);
  });

  test('tracks mutation states', () => {
    const { result } = renderHook(() => useServices(), { wrapper });

    expect(result.current.isStarting).toBe(false);
    expect(result.current.isStopping).toBe(false);
    expect(result.current.isRestarting).toBe(false);
  });

  test('filters services by multiple statuses', async () => {
    const { result: runningResult } = renderHook(
      () => useServices({ status: 'running' }),
      { wrapper }
    );

    const { result: stoppedResult } = renderHook(
      () => useServices({ status: 'stopped' }),
      { wrapper }
    );

    await waitFor(() => {
      expect(runningResult.current.isLoading).toBe(false);
      expect(stoppedResult.current.isLoading).toBe(false);
    });

    expect(runningResult.current.services).toHaveLength(1);
    expect(runningResult.current.services[0].name).toBe('postgres');

    expect(stoppedResult.current.services).toHaveLength(1);
    expect(stoppedResult.current.services[0].name).toBe('redis');
  });

  test('returns empty array when no services match filter', async () => {
    const { result } = renderHook(
      () => useServices({ status: 'error' }),
      { wrapper }
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.services).toHaveLength(0);
  });
});