import { spawn, type Subprocess } from 'bun';
import { client } from '@/generated/api/client.gen';

export interface TestServer {
  process: Subprocess;
  baseUrl: string;
  port: number;
}

let serverProcess: Subprocess | null = null;

/**
 * Start the Vibeman API server for integration tests
 */
export async function startTestServer(port: number = 18080): Promise<TestServer> {
  // Kill any existing server process
  await stopTestServer();

  console.log(`Starting Vibeman API server on port ${port}...`);
  
  // Start the server process
  serverProcess = spawn(['../vibeman', 'server', 'start', '--port', port.toString()], {
    cwd: process.cwd(),
    env: {
      ...process.env,
      VIBEMAN_ENV: 'test',
    },
    stdout: 'pipe',
    stderr: 'pipe',
  });

  // Wait for server to be ready
  const baseUrl = `http://localhost:${port}`;
  await waitForServer(baseUrl, 30000); // 30 second timeout

  console.log(`Vibeman API server started successfully on ${baseUrl}`);

  return {
    process: serverProcess,
    baseUrl,
    port,
  };
}

/**
 * Stop the test server
 */
export async function stopTestServer(): Promise<void> {
  if (serverProcess) {
    console.log('Stopping Vibeman API server...');
    serverProcess.kill();
    await serverProcess.exited;
    serverProcess = null;
    console.log('Vibeman API server stopped');
  }
}

/**
 * Wait for the server to be ready
 */
async function waitForServer(baseUrl: string, timeout: number): Promise<void> {
  const startTime = Date.now();
  
  while (Date.now() - startTime < timeout) {
    try {
      const response = await fetch(`${baseUrl}/health`);
      if (response.ok) {
        return;
      }
    } catch (error) {
      // Server not ready yet
    }
    
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  
  throw new Error(`Server failed to start within ${timeout}ms`);
}

/**
 * Configure the API client for tests
 */
export function configureTestClient(baseUrl: string): void {
  // Configure the generated client to use the test server
  client.setConfig({
    baseUrl: baseUrl,
  });
}

/**
 * Clean up test data
 */
export async function cleanupTestData(): Promise<void> {
  try {
    // Import SDK functions
    const { getRepositories, deleteRepositoriesById } = await import('@/generated/api');
    
    // Clean up any test repositories
    const reposResponse = await getRepositories();
    if (reposResponse.data?.repositories) {
      for (const repo of reposResponse.data.repositories) {
        if (repo.name.startsWith('test-')) {
          await deleteRepositoriesById({
            path: { id: repo.id },
          });
        }
      }
    }
  } catch (error) {
    console.error('Error cleaning up test data:', error);
  }
}