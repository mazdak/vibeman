// Integration test setup - runs before all integration tests
import '@testing-library/jest-dom';
import '@happy-dom/global-registrator';
import { startTestServer, stopTestServer, configureTestClient } from './setup';

// Global test server instance
let testServer: Awaited<ReturnType<typeof startTestServer>> | null = null;

// Start server before all tests
beforeAll(async () => {
  console.log('Starting integration test server...');
  testServer = await startTestServer();
  configureTestClient(testServer.baseUrl);
});

// Stop server after all tests
afterAll(async () => {
  console.log('Stopping integration test server...');
  await stopTestServer();
});

// Export test server info for tests
export function getTestServer() {
  if (!testServer) {
    throw new Error('Test server not initialized');
  }
  return testServer;
}