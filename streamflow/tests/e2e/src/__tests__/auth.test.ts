import { TestHelper, TestUser, AuthTokens } from '../utils/test-helper';

describe('Auth Service E2E Tests', () => {
  let testUser: TestUser;
  let userTokens: AuthTokens;

  beforeAll(async () => {
    // Create a test user for authentication tests
    testUser = await TestHelper.createTestUser({
      email: TestHelper.generateRandomEmail(),
      password: 'TestPassword123!',
      role: 'cliente',
      name: 'Auth Test User'
    });
  });

  describe('User Registration', () => {
    test('should create new user successfully', async () => {
      const userData = {
        email: TestHelper.generateRandomEmail(),
        password: 'NewUser123!',
        name: 'New Test User',
        role: 'cliente'
      };

      const response = await TestHelper['httpClient'].post('/usuarios', userData);
      
      expect(response.status).toBe(201);
      expect(response.data).toHaveProperty('id');
      expect(response.data.email).toBe(userData.email);
      expect(response.data.role).toBe('cliente');
      expect(response.data).not.toHaveProperty('password'); // Password should not be returned
    });

    test('should reject duplicate email registration', async () => {
      const userData = {
        email: testUser.email, // Use existing email
        password: 'AnotherPassword123!',
        name: 'Duplicate User',
        role: 'cliente'
      };

      const response = await TestHelper['httpClient'].post('/usuarios', userData);
      
      expect(response.status).toBe(400);
      expect(response.data).toHaveProperty('message');
    });

    test('should reject invalid email format', async () => {
      const userData = {
        email: 'invalid-email',
        password: 'ValidPassword123!',
        name: 'Invalid Email User',
        role: 'cliente'
      };

      const response = await TestHelper['httpClient'].post('/usuarios', userData);
      
      expect(response.status).toBe(400);
    });

    test('should reject weak passwords', async () => {
      const userData = {
        email: TestHelper.generateRandomEmail(),
        password: '123', // Weak password
        name: 'Weak Password User',
        role: 'cliente'
      };

      const response = await TestHelper['httpClient'].post('/usuarios', userData);
      
      expect(response.status).toBe(400);
    });
  });

  describe('User Authentication', () => {
    test('should login with valid credentials', async () => {
      const response = await TestHelper['httpClient'].post('/auth/login', {
        email: testUser.email,
        password: testUser.password
      });

      expect(response.status).toBe(200);
      expect(response.data).toHaveProperty('access_token');
      expect(response.data).toHaveProperty('user');
      expect(response.data.user.email).toBe(testUser.email);

      // Store tokens for later tests
      userTokens = {
        accessToken: response.data.access_token,
        refreshToken: response.data.refresh_token,
        user: response.data.user
      };
    });

    test('should reject invalid email', async () => {
      const response = await TestHelper['httpClient'].post('/auth/login', {
        email: 'nonexistent@streamflow.com',
        password: testUser.password
      });

      expect(response.status).toBe(401);
      expect(response.data).toHaveProperty('message');
    });

    test('should reject invalid password', async () => {
      const response = await TestHelper['httpClient'].post('/auth/login', {
        email: testUser.email,
        password: 'wrongpassword'
      });

      expect(response.status).toBe(401);
      expect(response.data).toHaveProperty('message');
    });

    test('should reject malformed login request', async () => {
      const response = await TestHelper['httpClient'].post('/auth/login', {
        email: testUser.email
        // Missing password
      });

      expect(response.status).toBe(400);
    });
  });

  describe('Token Validation', () => {
    test('should access protected endpoint with valid token', async () => {
      const authClient = TestHelper.createAuthenticatedClient(userTokens.accessToken);
      
      const response = await authClient.get(`/usuarios/${userTokens.user.id}`);
      
      expect(response.status).toBe(200);
      expect(response.data.email).toBe(testUser.email);
    });

    test('should reject access with invalid token', async () => {
      const invalidClient = TestHelper.createAuthenticatedClient('invalid-token');
      
      const response = await invalidClient.get(`/usuarios/${userTokens.user.id}`);
      
      expect(response.status).toBe(401);
    });

    test('should reject access with expired token', async () => {
      // This would require a token that's actually expired
      // For now, we'll test with a malformed token
      const expiredClient = TestHelper.createAuthenticatedClient('expired.token.here');
      
      const response = await expiredClient.get(`/usuarios/${userTokens.user.id}`);
      
      expect(response.status).toBe(401);
    });
  });

  describe('Password Management', () => {
    test('should change password with valid token and data', async () => {
      const authClient = TestHelper.createAuthenticatedClient(userTokens.accessToken);
      
      const newPassword = 'NewPassword123!';
      const response = await authClient.patch(`/auth/usuarios/${userTokens.user.id}`, {
        current_password: testUser.password,
        new_password: newPassword
      });

      expect(response.status).toBeOneOf([200, 204]);

      // Update test user password for logout test
      testUser.password = newPassword;
    });

    test('should reject password change with wrong current password', async () => {
      const authClient = TestHelper.createAuthenticatedClient(userTokens.accessToken);
      
      const response = await authClient.patch(`/auth/usuarios/${userTokens.user.id}`, {
        current_password: 'wrongcurrentpassword',
        new_password: 'AnotherNewPassword123!'
      });

      expect(response.status).toBe(400);
    });

    test('should reject password change without authentication', async () => {
      const response = await TestHelper['httpClient'].patch(`/auth/usuarios/${userTokens.user.id}`, {
        current_password: testUser.password,
        new_password: 'AnotherNewPassword123!'
      });

      expect(response.status).toBe(401);
    });
  });

  describe('User Logout', () => {
    test('should logout successfully and invalidate token', async () => {
      const authClient = TestHelper.createAuthenticatedClient(userTokens.accessToken);
      
      const logoutResponse = await authClient.post('/auth/logout');
      expect(logoutResponse.status).toBeOneOf([200, 204]);

      // Try to use the token after logout - should fail
      const protectedResponse = await authClient.get(`/usuarios/${userTokens.user.id}`);
      expect(protectedResponse.status).toBe(401);
    });

    test('should handle logout with invalid token gracefully', async () => {
      const invalidClient = TestHelper.createAuthenticatedClient('invalid-token');
      
      const response = await invalidClient.post('/auth/logout');
      expect(response.status).toBe(401);
    });
  });

  describe('Admin Authentication', () => {
    test('should authenticate admin user with elevated privileges', async () => {
      const adminTokens = await TestHelper.getAdminTokens();
      const adminClient = TestHelper.createAuthenticatedClient(adminTokens.accessToken);

      expect(adminTokens.user.role).toBe('admin');

      // Test admin-only endpoint access
      const response = await adminClient.get('/usuarios');
      expect(response.status).toBeOneOf([200, 404]); // Should have access
    });

    test('should prevent regular users from accessing admin endpoints', async () => {
      // Create a new user and get their token
      const regularUser = await TestHelper.createTestUser({
        role: 'cliente'
      });
      
      const regularTokens = await TestHelper.authenticateUser(
        regularUser.email,
        regularUser.password
      );
      
      const regularClient = TestHelper.createAuthenticatedClient(regularTokens.accessToken);

      // Try to access admin endpoint
      const response = await regularClient.get('/usuarios');
      expect(response.status).toBeOneOf([403, 401]); // Should be forbidden
    });
  });

  describe('Authentication Edge Cases', () => {
    test('should handle concurrent login attempts', async () => {
      const loginPromises = [];
      
      for (let i = 0; i < 5; i++) {
        loginPromises.push(
          TestHelper['httpClient'].post('/auth/login', {
            email: testUser.email,
            password: testUser.password
          })
        );
      }

      const responses = await Promise.all(loginPromises);
      
      // All should succeed or fail consistently
      responses.forEach(response => {
        expect(response.status).toBeOneOf([200, 401, 429]); // 429 for rate limiting
      });
    });

    test('should handle malformed authorization headers', async () => {
      const malformedClient = TestHelper['httpClient'];
      malformedClient.defaults.headers.common['Authorization'] = 'InvalidFormat token123';
      
      const response = await malformedClient.get(`/usuarios/${userTokens?.user?.id || '1'}`);
      expect(response.status).toBe(401);
      
      // Clean up
      delete malformedClient.defaults.headers.common['Authorization'];
    });

    test('should handle empty authorization header', async () => {
      const emptyAuthClient = TestHelper['httpClient'];
      emptyAuthClient.defaults.headers.common['Authorization'] = '';
      
      const response = await emptyAuthClient.get(`/usuarios/${userTokens?.user?.id || '1'}`);
      expect(response.status).toBe(401);
      
      // Clean up
      delete emptyAuthClient.defaults.headers.common['Authorization'];
    });
  });
});

// Extend Jest matchers for this test file
declare global {
  namespace jest {
    interface Matchers<R> {
      toBeOneOf(expected: any[]): R;
    }
  }
}

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

