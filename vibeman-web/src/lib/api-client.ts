import { client } from '../generated/api/client.gen';
import type { DbWorktree, DbWorktreeStatus, DbRepository } from '../generated/api/types.gen';

// Data transformation utilities
// Removed transformation - use backend statuses directly

// Transform raw worktree data to UI format (maintains backward compatibility)
export interface UIWorktree extends DbWorktree {
  container_status: 'active' | 'inactive' | 'building' | 'failed';
  last_activity: string;
  setup_type: 'script' | 'inline';
}

export const transformWorktree = (wt: DbWorktree): UIWorktree => ({
  ...wt,
  status: wt.status,
  last_activity: wt.updated_at || wt.created_at || new Date().toISOString(),
  setup_type: 'script' as const,
});

// Transform repository data to UI format
export interface UIRepository extends DbRepository {
  repository_url: string;
}

export const transformRepository = (repo: DbRepository): UIRepository => ({
  ...repo,
  repository_url: repo.git_url || '',
});

// Client configuration function
export const createClientConfig = () => ({
  baseUrl: '/api',
});

// Initialize the generated client with our configuration
client.setConfig({
  baseUrl: '/api', // Use relative URL to go through the proxy
  headers: {
    'Content-Type': 'application/json',
  },
  credentials: 'include',
  interceptors: {
    request: {
      onRequest: (request) => {
        // Add auth token if available
        const token = localStorage.getItem('auth_token');
        if (token) {
          request.headers = {
            ...request.headers,
            Authorization: `Bearer ${token}`,
          };
        }
        return request;
      },
    },
    response: {
      onResponse: (response) => {
        // Handle successful responses
        return response;
      },
      onError: (error) => {
        console.error('API Error:', error);
        if (error.response?.status === 401) {
          localStorage.removeItem('auth_token');
          window.location.href = '/login';
        }
        throw new Error(error.response?.data?.error || 'API request failed');
      },
    },
  },
});

// Export the configured client
export { client };

// Helper to set auth token
export const setAuthToken = (token: string | null) => {
  if (token) {
    localStorage.setItem('auth_token', token);
  } else {
    localStorage.removeItem('auth_token');
  }
};

// Helper to get auth token from storage
export const getStoredAuthToken = (): string | null => {
  return localStorage.getItem('auth_token');
};

// Re-export commonly used types for convenience
export type { 
  DbWorktree,
  DbRepository,
  ServerConfigResponse,
  ServerCreateWorktreeRequest,
  ServerAddRepositoryRequest,
  ServerSystemStatusResponse,
} from '../generated/api/types.gen';