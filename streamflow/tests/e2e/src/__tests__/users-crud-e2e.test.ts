import { TestHelper, TestUser, AuthTokens } from '../utils/test-helper';

describe('Users Service CRUD E2E Tests', () => {
  let adminTokens: AuthTokens;
  let adminClient: any;
  let testUserId: string;
  let testUserEmail: string;
  let testUserPassword: string;

  beforeAll(async () => {
    console.log('🚀 Starting Users Service CRUD E2E Tests...');
    
    // Wait for services to be ready
    await TestHelper.waitForServices();
    
    // Get admin authentication for user management
    adminTokens = await TestHelper.getAdminTokens();
    adminClient = TestHelper.createAuthenticatedClient(adminTokens.accessToken);
    
    console.log('✅ Setup complete, starting CRUD tests...');
  });

  describe('1. POST /auth/login - Iniciar sesión', () => {
    describe('Success Case', () => {
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
        expect(response.data.user.email).toBe(adminEmail);
        expect(response.data.user.role).toBe('admin');

        console.log('✅ POST /auth/login - Success case passed');
      });
    });

    describe('Error Case', () => {
      test('should reject login with invalid credentials', async () => {
        const response = await TestHelper['httpClient'].post('/auth/login', {
          email: 'invalid@streamflow.com',
          password: 'wrongpassword'
        });

        expect(response.status).toBe(401);
        expect(response.data).toHaveProperty('message');
        
        console.log('✅ POST /auth/login - Error case passed');
      });
    });
  });

  describe('2. POST /usuarios - Crear usuario', () => {
    describe('Success Case', () => {
      test('should create new user successfully', async () => {
        testUserEmail = TestHelper.generateRandomEmail();
        testUserPassword = 'E2ETest123!';
        
        const userData = {
          email: testUserEmail,
          password: testUserPassword,
          name: 'E2E Test User',
          role: 'cliente'
        };

        const response = await TestHelper['httpClient'].post('/usuarios', userData);

        expect(response.status).toBeOneOf([201, 200]);
        expect(response.data).toHaveProperty('id');
        expect(response.data.email).toBe(userData.email);
        expect(response.data.role).toBe('cliente');
        expect(response.data).not.toHaveProperty('password'); // Password should not be returned

        testUserId = response.data.id;
        console.log(`✅ POST /usuarios - Success case passed, created user ID: ${testUserId}`);
      });
    });

    describe('Error Case', () => {
      test('should reject user creation with duplicate email', async () => {
        const duplicateUserData = {
          email: testUserEmail, // Use the same email from success case
          password: 'AnotherPassword123!',
          name: 'Duplicate User',
          role: 'cliente'
        };

        const response = await TestHelper['httpClient'].post('/usuarios', duplicateUserData);

        expect(response.status).toBe(400);
        expect(response.data).toHaveProperty('message');
        
        console.log('✅ POST /usuarios - Error case passed (duplicate email)');
      });

      test('should reject user creation with invalid email format', async () => {
        const invalidUserData = {
          email: 'invalid-email-format',
          password: 'ValidPassword123!',
          name: 'Invalid Email User',
          role: 'cliente'
        };

        const response = await TestHelper['httpClient'].post('/usuarios', invalidUserData);

        expect(response.status).toBe(400);
        
        console.log('✅ POST /usuarios - Error case passed (invalid email)');
      });
    });
  });

  describe('3. GET /usuarios/{id} - Obtener usuario por ID', () => {
    describe('Success Case', () => {
      test('should get user by valid ID', async () => {
        const response = await adminClient.get(`/usuarios/${testUserId}`);

        expect(response.status).toBe(200);
        expect(response.data).toHaveProperty('id', testUserId);
        expect(response.data.email).toBe(testUserEmail);
        expect(response.data).toHaveProperty('name');
        expect(response.data).toHaveProperty('role');
        expect(response.data).not.toHaveProperty('password'); // Password should not be returned

        console.log('✅ GET /usuarios/{id} - Success case passed');
      });
    });

    describe('Error Case', () => {
      test('should return 404 for non-existent user ID', async () => {
        const nonExistentId = '999999999';
        const response = await adminClient.get(`/usuarios/${nonExistentId}`);

        expect(response.status).toBe(404);

        console.log('✅ GET /usuarios/{id} - Error case passed (non-existent ID)');
      });

      test('should return 401 for unauthenticated request', async () => {
        const response = await TestHelper['httpClient'].get(`/usuarios/${testUserId}`);

        expect(response.status).toBe(401);

        console.log('✅ GET /usuarios/{id} - Error case passed (unauthenticated)');
      });
    });
  });

  describe('4. PATCH /usuarios/{id} - Actualizar usuario', () => {
    describe('Success Case', () => {
      test('should update user information successfully', async () => {
        const updateData = {
          name: 'Updated E2E Test User'
        };

        const response = await adminClient.patch(`/usuarios/${testUserId}`, updateData);

        expect(response.status).toBeOneOf([200, 204]);

        // Verify the update by getting the user
        const getResponse = await adminClient.get(`/usuarios/${testUserId}`);
        expect(getResponse.status).toBe(200);
        expect(getResponse.data.name).toBe(updateData.name);

        console.log('✅ PATCH /usuarios/{id} - Success case passed');
      });
    });

    describe('Error Case', () => {
      test('should reject update for non-existent user', async () => {
        const nonExistentId = '999999999';
        const updateData = {
          name: 'Should not work'
        };

        const response = await adminClient.patch(`/usuarios/${nonExistentId}`, updateData);

        expect(response.status).toBe(404);

        console.log('✅ PATCH /usuarios/{id} - Error case passed (non-existent user)');
      });

      test('should reject update with invalid email format', async () => {
        const invalidUpdateData = {
          email: 'invalid-email-format'
        };

        const response = await adminClient.patch(`/usuarios/${testUserId}`, invalidUpdateData);

        expect(response.status).toBe(400);

        console.log('✅ PATCH /usuarios/{id} - Error case passed (invalid email)');
      });

      test('should reject update without authentication', async () => {
        const updateData = {
          name: 'Unauthorized update'
        };

        const response = await TestHelper['httpClient'].patch(`/usuarios/${testUserId}`, updateData);

        expect(response.status).toBe(401);

        console.log('✅ PATCH /usuarios/{id} - Error case passed (unauthenticated)');
      });
    });
  });

  describe('5. GET /usuarios - Listar todos los usuarios', () => {
    describe('Success Case', () => {
      test('should list all users for admin', async () => {
        const response = await adminClient.get('/usuarios');

        expect(response.status).toBe(200);
        expect(Array.isArray(response.data)).toBe(true);
        expect(response.data.length).toBeGreaterThan(0);

        // Verify our test user is in the list
        const ourUser = response.data.find((user: any) => user.id === testUserId);
        expect(ourUser).toBeDefined();
        expect(ourUser.email).toBe(testUserEmail);

        // Verify no passwords are returned
        response.data.forEach((user: any) => {
          expect(user).not.toHaveProperty('password');
          expect(user).toHaveProperty('email');
          expect(user).toHaveProperty('role');
        });

        console.log(`✅ GET /usuarios - Success case passed (found ${response.data.length} users)`);
      });
    });

    describe('Error Case', () => {
      test('should reject listing users without authentication', async () => {
        const response = await TestHelper['httpClient'].get('/usuarios');

        expect(response.status).toBe(401);

        console.log('✅ GET /usuarios - Error case passed (unauthenticated)');
      });

      test('should reject listing users for regular user', async () => {
        // First, login as the test user we created
        const userLoginResponse = await TestHelper['httpClient'].post('/auth/login', {
          email: testUserEmail,
          password: testUserPassword
        });

        expect(userLoginResponse.status).toBe(200);

        const userClient = TestHelper.createAuthenticatedClient(userLoginResponse.data.access_token);
        const response = await userClient.get('/usuarios');

        expect(response.status).toBeOneOf([401, 403]); // Should be forbidden for regular users

        console.log('✅ GET /usuarios - Error case passed (insufficient privileges)');
      });
    });
  });

  describe('6. DELETE /usuarios/{id} - Eliminar usuario', () => {
    describe('Success Case', () => {
      test('should delete user successfully', async () => {
        const response = await adminClient.delete(`/usuarios/${testUserId}`);

        expect(response.status).toBeOneOf([200, 204]);

        // Verify user is deleted by trying to get it
        const getResponse = await adminClient.get(`/usuarios/${testUserId}`);
        expect(getResponse.status).toBeOneOf([404, 410]); // 404 not found or 410 gone

        console.log('✅ DELETE /usuarios/{id} - Success case passed');
      });
    });

    describe('Error Case', () => {
      test('should return 404 when trying to delete non-existent user', async () => {
        const nonExistentId = '999999999';
        const response = await adminClient.delete(`/usuarios/${nonExistentId}`);

        expect(response.status).toBe(404);

        console.log('✅ DELETE /usuarios/{id} - Error case passed (non-existent user)');
      });

      test('should reject deletion without authentication', async () => {
        // Create another test user for this test
        const anotherUser = await TestHelper.createTestUser({
          email: TestHelper.generateRandomEmail(),
          role: 'cliente'
        });

        const response = await TestHelper['httpClient'].delete(`/usuarios/${anotherUser.id}`);

        expect(response.status).toBe(401);

        console.log('✅ DELETE /usuarios/{id} - Error case passed (unauthenticated)');
      });

      test('should verify deleted user cannot login', async () => {
        // Try to login with the deleted user credentials
        const loginResponse = await TestHelper['httpClient'].post('/auth/login', {
          email: testUserEmail,
          password: testUserPassword
        });

        expect(loginResponse.status).toBe(401);

        console.log('✅ DELETE /usuarios/{id} - Additional verification: deleted user cannot login');
      });
    });
  });

  afterAll(async () => {
    console.log('🧹 Cleaning up Users Service CRUD E2E Tests...');
    // Any additional cleanup if needed
    console.log('✅ Users Service CRUD E2E Tests completed successfully!');
  });
});

// Custom Jest matchers
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

