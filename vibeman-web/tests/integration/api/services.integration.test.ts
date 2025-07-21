import { test, expect, describe, beforeEach, afterEach } from 'bun:test';
import { client } from '@/generated/api';
import { cleanupTestData } from '../setup';
import type { ServiceInstance } from '@/generated/api/types.gen';

describe('Services API Integration', () => {
  beforeEach(async () => {
    await cleanupTestData();
  });

  afterEach(async () => {
    // Stop any test services that might still be running
    try {
      const response = await client.GET('/services');
      if (response.data?.services) {
        for (const service of response.data.services) {
          if (service.name.startsWith('test-') && service.status === 'running') {
            await client.POST('/services/{name}/stop', {
              params: { path: { name: service.name } },
            });
          }
        }
      }
    } catch (error) {
      // Ignore cleanup errors
    }
  });

  describe('GET /services', () => {
    test('should list all available services', async () => {
      const response = await client.GET('/services');
      
      expect(response.response.status).toBe(200);
      expect(response.data).toBeDefined();
      expect(Array.isArray(response.data?.services)).toBe(true);
    });

    test('should return service details in correct format', async () => {
      const response = await client.GET('/services');
      
      expect(response.response.status).toBe(200);
      
      const services = response.data?.services || [];
      if (services.length > 0) {
        const service = services[0];
        expect(service).toHaveProperty('name');
        expect(service).toHaveProperty('status');
        expect(service).toHaveProperty('ref_count');
        expect(['stopped', 'starting', 'running', 'stopping', 'error']).toContain(service.status);
      }
    });
  });

  describe('GET /services/{name}', () => {
    test('should get service details by name', async () => {
      // First, get list of services to find a valid service name
      const listResponse = await client.GET('/services');
      const services = listResponse.data?.services || [];
      
      if (services.length > 0) {
        const serviceName = services[0].name;
        
        const response = await client.GET('/services/{name}', {
          params: { path: { name: serviceName } },
        });
        
        expect(response.response.status).toBe(200);
        expect(response.data).toBeDefined();
        expect(response.data?.name).toBe(serviceName);
        expect(response.data).toHaveProperty('status');
        expect(response.data).toHaveProperty('config');
      }
    });

    test('should return 404 for non-existent service', async () => {
      const response = await client.GET('/services/{name}', {
        params: { path: { name: 'non-existent-service' } },
      });
      
      expect(response.response.status).toBe(404);
    });
  });

  describe('POST /services/{name}/start', () => {
    test('should start a service', async () => {
      // Get list of available services
      const listResponse = await client.GET('/services');
      const services = listResponse.data?.services || [];
      
      // Find a stopped service to test with
      const stoppedService = services.find(s => s.status === 'stopped');
      
      if (stoppedService) {
        const response = await client.POST('/services/{name}/start', {
          params: { path: { name: stoppedService.name } },
        });
        
        expect(response.response.status).toBe(200);
        expect(response.data?.message).toContain('started');
        
        // Verify service status changed
        const statusResponse = await client.GET('/services/{name}', {
          params: { path: { name: stoppedService.name } },
        });
        
        expect(['starting', 'running']).toContain(statusResponse.data?.status);
      } else {
        console.log('No stopped services available to test start operation');
      }
    });

    test('should return error when starting already running service', async () => {
      // Get list of available services
      const listResponse = await client.GET('/services');
      const services = listResponse.data?.services || [];
      
      // Find a running service
      const runningService = services.find(s => s.status === 'running');
      
      if (runningService) {
        const response = await client.POST('/services/{name}/start', {
          params: { path: { name: runningService.name } },
        });
        
        expect(response.response.status).toBe(400);
        expect(response.error?.error).toContain('already running');
      }
    });

    test('should return 404 for non-existent service', async () => {
      const response = await client.POST('/services/{name}/start', {
        params: { path: { name: 'non-existent-service' } },
      });
      
      expect(response.response.status).toBe(404);
    });
  });

  describe('POST /services/{name}/stop', () => {
    test('should stop a running service', async () => {
      // First, we need to start a service to test stopping it
      const listResponse = await client.GET('/services');
      const services = listResponse.data?.services || [];
      
      // Find a service we can test with
      const testService = services.find(s => s.status === 'stopped');
      
      if (testService) {
        // Start the service first
        await client.POST('/services/{name}/start', {
          params: { path: { name: testService.name } },
        });
        
        // Wait a bit for service to start
        await new Promise(resolve => setTimeout(resolve, 2000));
        
        // Now stop it
        const response = await client.POST('/services/{name}/stop', {
          params: { path: { name: testService.name } },
        });
        
        expect(response.response.status).toBe(200);
        expect(response.data?.message).toContain('stopped');
        
        // Verify service status changed
        const statusResponse = await client.GET('/services/{name}', {
          params: { path: { name: testService.name } },
        });
        
        expect(['stopping', 'stopped']).toContain(statusResponse.data?.status);
      } else {
        console.log('No services available to test stop operation');
      }
    });

    test('should return error when stopping already stopped service', async () => {
      const listResponse = await client.GET('/services');
      const services = listResponse.data?.services || [];
      
      const stoppedService = services.find(s => s.status === 'stopped');
      
      if (stoppedService) {
        const response = await client.POST('/services/{name}/stop', {
          params: { path: { name: stoppedService.name } },
        });
        
        expect(response.response.status).toBe(400);
        expect(response.error?.error).toContain('not running');
      }
    });

    test('should return 404 for non-existent service', async () => {
      const response = await client.POST('/services/{name}/stop', {
        params: { path: { name: 'non-existent-service' } },
      });
      
      expect(response.response.status).toBe(404);
    });
  });

  describe('POST /services/{name}/restart', () => {
    test('should restart a running service', async () => {
      // Get list of available services
      const listResponse = await client.GET('/services');
      const services = listResponse.data?.services || [];
      
      // Find or create a running service
      let testService = services.find(s => s.status === 'running');
      
      if (!testService) {
        // Start a service if none are running
        const stoppedService = services.find(s => s.status === 'stopped');
        if (stoppedService) {
          await client.POST('/services/{name}/start', {
            params: { path: { name: stoppedService.name } },
          });
          await new Promise(resolve => setTimeout(resolve, 2000));
          testService = stoppedService;
        }
      }
      
      if (testService) {
        const response = await client.POST('/services/{name}/restart', {
          params: { path: { name: testService.name } },
        });
        
        expect(response.response.status).toBe(200);
        expect(response.data?.message).toContain('restarted');
      } else {
        console.log('No services available to test restart operation');
      }
    });

    test('should return 404 for non-existent service', async () => {
      const response = await client.POST('/services/{name}/restart', {
        params: { path: { name: 'non-existent-service' } },
      });
      
      expect(response.response.status).toBe(404);
    });
  });

  describe('GET /services/{name}/logs', () => {
    test('should get service logs', async () => {
      // Get a running service
      const listResponse = await client.GET('/services');
      const services = listResponse.data?.services || [];
      
      const runningService = services.find(s => s.status === 'running');
      
      if (runningService) {
        const response = await client.GET('/services/{name}/logs', {
          params: { path: { name: runningService.name } },
        });
        
        expect(response.response.status).toBe(200);
        expect(response.data).toBeDefined();
        expect(Array.isArray(response.data?.logs)).toBe(true);
      } else {
        console.log('No running services available to test logs');
      }
    });

    test('should support tail parameter', async () => {
      const listResponse = await client.GET('/services');
      const services = listResponse.data?.services || [];
      
      const runningService = services.find(s => s.status === 'running');
      
      if (runningService) {
        const response = await client.GET('/services/{name}/logs', {
          params: { 
            path: { name: runningService.name },
            query: { tail: 10 },
          },
        });
        
        expect(response.response.status).toBe(200);
        expect(Array.isArray(response.data?.logs)).toBe(true);
        expect(response.data?.logs.length).toBeLessThanOrEqual(10);
      }
    });

    test('should return 404 for non-existent service', async () => {
      const response = await client.GET('/services/{name}/logs', {
        params: { path: { name: 'non-existent-service' } },
      });
      
      expect(response.response.status).toBe(404);
    });
  });

  describe('POST /services/{name}/health', () => {
    test('should check service health', async () => {
      const listResponse = await client.GET('/services');
      const services = listResponse.data?.services || [];
      
      const runningService = services.find(s => s.status === 'running');
      
      if (runningService) {
        const response = await client.POST('/services/{name}/health', {
          params: { path: { name: runningService.name } },
        });
        
        expect(response.response.status).toBe(200);
        expect(response.data).toBeDefined();
        expect(response.data).toHaveProperty('healthy');
        expect(typeof response.data?.healthy).toBe('boolean');
        if (response.data?.message) {
          expect(typeof response.data.message).toBe('string');
        }
      } else {
        console.log('No running services available to test health check');
      }
    });

    test('should return unhealthy for stopped service', async () => {
      const listResponse = await client.GET('/services');
      const services = listResponse.data?.services || [];
      
      const stoppedService = services.find(s => s.status === 'stopped');
      
      if (stoppedService) {
        const response = await client.POST('/services/{name}/health', {
          params: { path: { name: stoppedService.name } },
        });
        
        expect(response.response.status).toBe(200);
        expect(response.data?.healthy).toBe(false);
        expect(response.data?.message).toContain('not running');
      }
    });

    test('should return 404 for non-existent service', async () => {
      const response = await client.POST('/services/{name}/health', {
        params: { path: { name: 'non-existent-service' } },
      });
      
      expect(response.response.status).toBe(404);
    });
  });
});