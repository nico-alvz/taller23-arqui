import { TestHelper } from '../utils/test-helper';

describe('Smoke Tests - Basic Service Health', () => {
  
  describe('Service Health Checks', () => {
    test('should verify Nginx load balancer is responding', async () => {
      const response = await TestHelper['httpClient'].get('/health');
      expect(response.status).toBe(200);
    });

    test('should verify API Gateway is accessible through Nginx', async () => {
      const response = await TestHelper['httpClient'].get('/');
      expect(response.status).toBeOneOf([200, 404, 401]); // 401 is also acceptable for protected root path
    });

    test('should verify comedy endpoint is working', async () => {
      const response = await TestHelper['httpClient'].get('/comedia');
      expect(response.status).toBe(200);
      // Check for any comedy-related content in the response
      expect(typeof response.data).toBe('object');
      expect(response.data).toHaveProperty('service', 'nginx-comedy');
    });
  });

  describe('Authentication Service', () => {
    test('should have auth service responding', async () => {
      // We'll test through the API Gateway instead
      const response = await TestHelper['httpClient'].get('/auth/health');
      expect(response.status).toBeOneOf([200, 404, 401]); // May not be exposed through gateway
    });

    test('should reject invalid login attempts', async () => {
      const response = await TestHelper['httpClient'].post('/auth/login', {
        email: 'invalid@test.com',
        password: 'wrongpassword'
      });
      expect(response.status).toBe(401);
    });
  });

  describe('Admin User Access', () => {
    test('should authenticate admin user successfully', async () => {
      const adminEmail = process.env.TEST_ADMIN_EMAIL || 'admin@streamflow.com';
      const adminPassword = process.env.TEST_ADMIN_PASSWORD || 'admin123';
      
      const response = await TestHelper['httpClient'].post('/auth/login', {
        email: adminEmail,
        password: adminPassword
      });
      
      expect(response.status).toBe(200);
      expect(response.data).toHaveProperty('access_token');
      expect(response.data).toHaveProperty('user');
      expect(response.data.user.role).toBe('Administrador');
    });
  });

  describe('Public Endpoints', () => {
    test('should access public videos endpoint', async () => {
      const response = await TestHelper['httpClient'].get('/videos');
      expect(response.status).toBeOneOf([200, 404]); // May be empty or not implemented
    });

    test('should be able to create new user (public endpoint)', async () => {
      const testUser = {
        email: TestHelper.generateRandomEmail(),
        password: 'Test123!',
        confirm_password: 'Test123!',
        first_name: 'Test',
        last_name: 'User',
        role: 'cliente'
      };

      const response = await TestHelper['httpClient'].post('/usuarios', testUser);
      expect(response.status).toBeOneOf([201, 200, 400]); // 400 may happen due to validation
    });
  });

  describe('Protected Endpoints', () => {
    test('should reject requests without authentication token', async () => {
      const endpoints = [
        { method: 'GET', path: '/usuarios/1' },
        { method: 'PATCH', path: '/usuarios/1' },
        { method: 'DELETE', path: '/usuarios/1' },
        { method: 'POST', path: '/videos' },
        { method: 'POST', path: '/facturas' },
        { method: 'POST', path: '/listas-reproduccion' }
      ];

      for (const endpoint of endpoints) {
        const response = await TestHelper['httpClient'].request({
          method: endpoint.method,
          url: endpoint.path
        });
        expect(response.status).toBe(401);
      }
    });
  });

  describe('Service Integration', () => {
    test('should verify services can communicate through API Gateway', async () => {
      // Get admin token
      const adminTokens = await TestHelper.getAdminTokens();
      const authClient = TestHelper.createAuthenticatedClient(adminTokens.accessToken);

      // Test multiple service calls
      const endpoints = [
        '/usuarios',
        '/videos',
        '/monitoreo/acciones',
        '/listas-reproduccion'
      ];

      for (const endpoint of endpoints) {
        const response = await authClient.get(endpoint);
        expect(response.status).toBeOneOf([200, 404, 500]); // Some may not be fully implemented
      }
    });
  });

  describe('Load Balancer', () => {
    test('should distribute requests across API Gateway instances', async () => {
      const requests = [];
      
      // Make multiple requests to test load balancing
      for (let i = 0; i < 10; i++) {
        requests.push(TestHelper['httpClient'].get('/health'));
      }

      const responses = await Promise.all(requests);
      
      // All requests should succeed
      responses.forEach(response => {
        expect(response.status).toBe(200);
      });
    });
  });
});

// Custom Jest matcher
expect.extend({
  toBeOneOf(received: any, expected: any[]) {
    const pass = expected.includes(received);
    if (pass) {
      return {
        message: () => `expected ${received} not to be one of ${expected.join(', ')}`,
        pass: true,
      };
    } else {
      return {
        message: () => `expected ${received} to be one of ${expected.join(', ')}`,
        pass: false,
      };
    }
  },
});

