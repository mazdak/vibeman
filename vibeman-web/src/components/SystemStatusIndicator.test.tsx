/// <reference lib="dom" />
import { test, expect, describe, beforeEach, mock } from "bun:test";
import { render } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SystemStatusIndicator } from './SystemStatusIndicator';
import React from 'react';

// Setup happy-dom
import { GlobalRegistrator } from '@happy-dom/global-registrator';
if (!global.happyDOM) {
  GlobalRegistrator.register();
}

// Mock the useSystemStatus hook
mock.module('@/hooks/api/useSystemStatus', () => ({
  useSystemStatus: () => {
    const mockScenario = (global as any).mockScenario || 'success';
    
    switch (mockScenario) {
      case 'loading':
        return { data: null, isLoading: true, error: null };
      case 'error':
        return { data: null, isLoading: false, error: new Error('API Error') };
      case 'unhealthy':
        return {
          data: {
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
          },
          isLoading: false,
          error: null
        };
      default:
        return {
          data: {
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
          },
          isLoading: false,
          error: null
        };
    }
  }
}));

describe('SystemStatusIndicator', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    (global as any).mockScenario = 'success';
  });

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  test('shows loading state', () => {
    (global as any).mockScenario = 'loading';
    
    const { getByText } = render(<SystemStatusIndicator />, { wrapper });
    
    expect(getByText('Connecting...')).toBeTruthy();
  });

  test('shows error state', () => {
    (global as any).mockScenario = 'error';
    
    const { getByText } = render(<SystemStatusIndicator />, { wrapper });
    
    expect(getByText('API Error')).toBeTruthy();
  });

  test('shows healthy status with counts', () => {
    (global as any).mockScenario = 'success';
    
    const { getByText } = render(<SystemStatusIndicator />, { wrapper });
    
    expect(getByText('healthy')).toBeTruthy();
    expect(getByText('• 5 worktrees • 3 containers')).toBeTruthy();
  });

  test('shows unhealthy status', () => {
    (global as any).mockScenario = 'unhealthy';
    
    const { getByText } = render(<SystemStatusIndicator />, { wrapper });
    
    expect(getByText('unhealthy')).toBeTruthy();
    expect(getByText('• 2 worktrees • 1 containers')).toBeTruthy();
  });
});