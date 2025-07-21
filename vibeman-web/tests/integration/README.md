# Vibeman Frontend Integration Tests

This directory contains integration tests for the Vibeman frontend that test real API calls against a running Vibeman server.

## Overview

These tests ensure that:
- All API endpoints are working correctly
- The TypeScript client generated from OpenAPI spec works properly
- Request/response formats match the API specification
- Error handling works as expected
- WebSocket connections function correctly

## Test Structure

```
tests/integration/
├── setup.ts                    # Test server management utilities
├── setup-integration.ts        # Global test setup/teardown
├── api/
│   ├── repositories.integration.test.ts
│   ├── worktrees.integration.test.ts
│   ├── services.integration.test.ts
│   ├── logs.integration.test.ts
│   └── websocket.integration.test.ts
└── README.md
```

## Running Integration Tests

### Prerequisites

1. Build the Vibeman binary:
   ```bash
   cd ../
   go build -o vibeman main.go
   ```

2. Ensure you have the latest API types generated:
   ```bash
   bun run generate-api
   ```

### Running Tests

Run all integration tests:
```bash
bun run test:integration
```

Run only unit tests (excluding integration):
```bash
bun run test:unit
```

Run all tests (unit + integration):
```bash
bun test
```

### Test Behavior

- Integration tests automatically start a Vibeman server on port 18080
- The server is started once before all tests and stopped after
- Each test suite cleans up its test data after running
- Test repositories and worktrees are prefixed with "test-" for easy identification

## Writing New Integration Tests

1. Create a new test file in `tests/integration/api/` with `.integration.test.ts` suffix
2. Import the generated API client:
   ```typescript
   import { client } from '@/generated/api';
   ```
3. Use `beforeEach` and `afterEach` to set up and clean up test data
4. Test both success and error cases
5. Verify response status codes and data structures

### Example Test Structure

```typescript
import { test, expect, describe, beforeEach, afterEach } from 'bun:test';
import { client } from '@/generated/api';
import { cleanupTestData } from '../setup';

describe('Feature API Integration', () => {
  let testResourceId: string | undefined;

  beforeEach(async () => {
    await cleanupTestData();
    // Set up test data
  });

  afterEach(async () => {
    // Clean up created resources
    if (testResourceId) {
      await client.DELETE('/resource/{id}', {
        params: { path: { id: testResourceId } },
      });
    }
  });

  test('should perform action', async () => {
    const response = await client.POST('/resource', {
      body: { /* request data */ },
    });
    
    expect(response.response.status).toBe(201);
    expect(response.data).toBeDefined();
  });
});
```

## API Coverage

Current test coverage includes:

### Repositories API
- ✅ Create repository
- ✅ List repositories
- ✅ Get repository by ID
- ✅ Update repository
- ✅ Delete repository
- ✅ Refresh repository

### Worktrees API
- ✅ Create worktree
- ✅ List worktrees (with filtering)
- ✅ Get worktree by ID
- ✅ Delete worktree (with force option)
- ✅ Start worktree container
- ✅ Stop worktree container
- ✅ Execute shell commands

### Services API
- ✅ List services
- ✅ Get service details
- ✅ Start service
- ✅ Stop service
- ✅ Restart service
- ✅ Get service logs
- ✅ Check service health

### Logs API
- ✅ Get worktree logs
- ✅ Search logs
- ✅ Clear logs
- ✅ Get aggregated logs

### WebSocket APIs
- ✅ AI container terminal access
- ✅ Log streaming
- ✅ Terminal commands and responses
- ✅ Origin validation

## Troubleshooting

### Server fails to start
- Check if port 18080 is already in use
- Ensure the Vibeman binary is built and accessible
- Check server logs in the test output

### Tests fail with 404 errors
- Ensure the API server is running (check console output)
- Verify the OpenAPI spec is up to date
- Regenerate TypeScript types: `bun run generate-api`

### Cleanup issues
- If tests leave behind test data, manually clean up:
  ```bash
  # Stop the server and remove test data
  ../vibeman server stop
  ```

## Future Improvements

1. Add performance benchmarks
2. Test concurrent API calls
3. Add stress testing for WebSocket connections
4. Test rate limiting and authentication (when implemented)
5. Add visual regression tests for UI components