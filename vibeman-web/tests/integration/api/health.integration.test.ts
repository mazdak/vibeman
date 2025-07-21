import { test, expect, describe } from 'bun:test';
import { getHealth } from '@/generated/api';

describe('Health API Integration', () => {
  test('should check API health', async () => {
    const response = await getHealth();
    
    expect(response.response.status).toBe(200);
    expect(response.data).toBeDefined();
    expect(response.data?.status).toBe('healthy');
  });
});