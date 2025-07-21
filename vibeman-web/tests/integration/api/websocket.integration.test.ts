import { test, expect, describe, beforeEach, afterEach } from 'bun:test';
import { 
  postRepositories,
  deleteRepositoriesById,
  postWorktrees
} from '@/generated/api';
import { cleanupTestData } from '../setup';
import { getTestServer } from '../setup-integration';

describe('WebSocket API Integration', () => {
  let testRepoId: string | undefined;
  let testWorktreeId: string | undefined;
  let ws: WebSocket | undefined;

  beforeEach(async () => {
    await cleanupTestData();
    
    // Create test repository and worktree
    const repoResponse = await postRepositories({
      body: {
        name: 'test-ws-repo',
        path: `/tmp/test-ws-repo-${Date.now()}`,
      },
    });
    
    expect(repoResponse.response.status).toBe(201);
    testRepoId = repoResponse.data?.repository.id;

    const worktreeResponse = await postWorktrees({
      body: {
        repository_id: testRepoId!,
        name: 'test-ws-worktree',
        skip_setup: true,
        auto_start: true, // Start container for WebSocket testing
      },
    });
    
    expect(worktreeResponse.response.status).toBe(201);
    testWorktreeId = worktreeResponse.data?.worktree.id;
  });

  afterEach(async () => {
    // Close WebSocket if open
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.close();
    }
    
    // Clean up
    if (testRepoId) {
      try {
        await deleteRepositoriesById({
          path: { id: testRepoId },
        });
      } catch (error) {
        // Ignore
      }
    }
  });

  describe('AI Container Terminal WebSocket', () => {
    test('should connect to AI container terminal', async () => {
      const testServer = getTestServer();
      const wsUrl = `ws://localhost:${testServer.port}/api/ai/attach/test-ws-worktree`;
      
      return new Promise<void>((resolve, reject) => {
        ws = new WebSocket(wsUrl);
        
        ws.onopen = () => {
          expect(ws?.readyState).toBe(WebSocket.OPEN);
          resolve();
        };
        
        ws.onerror = (error) => {
          reject(new Error(`WebSocket error: ${error}`));
        };
        
        // Set timeout
        setTimeout(() => {
          reject(new Error('WebSocket connection timeout'));
        }, 5000);
      });
    });

    test('should handle ping/pong', async () => {
      const testServer = getTestServer();
      const wsUrl = `ws://localhost:${testServer.port}/api/ai/attach/test-ws-worktree`;
      
      return new Promise<void>((resolve, reject) => {
        ws = new WebSocket(wsUrl);
        
        ws.onopen = () => {
          // Send ping
          ws?.send(JSON.stringify({
            type: 'ping',
          }));
        };
        
        ws.onmessage = (event) => {
          const message = JSON.parse(event.data);
          if (message.type === 'pong') {
            expect(message.type).toBe('pong');
            resolve();
          }
        };
        
        ws.onerror = (error) => {
          reject(new Error(`WebSocket error: ${error}`));
        };
        
        setTimeout(() => {
          reject(new Error('Test timeout - no pong received'));
        }, 5000);
      });
    });
  });
});