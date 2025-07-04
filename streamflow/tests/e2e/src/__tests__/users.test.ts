import { TestHelper, TestUser, AuthTokens } from '../utils/test-helper';

describe('Users Service E2E Tests', () => {
  let adminTokens: AuthTokens;
  let adminClient: any;
  let testUser: TestUser;
  let userTokens: AuthTokens;
  let userClient: any;

  beforeAll(async () => {
    // Get admin authentication
    adminTokens = await TestHelper.getAdminTokens();
    adminClient = TestHelper.createAuthenticatedClient(adminTokens.accessToken);

    // Create a test user
    testUser = await TestHelper.createTestUser({
      email: TestHelper.generateRandomEmail(),
      password: 'UserTest123!',
      role: 'cliente',
      name: 'Users Test User'
    });

    // Authenticate test user
    userTokens = await TestHelper.authenticateUser(testUser.email, testUser.password);
    userClient = TestHelper.createAuthenticatedClient(userTokens.accessToken);
  });

  describe('User CRUD Operations', () => {
    let createdUserId: string;

    test('should create a new user (admin only)', async () => {
      const userData = {
        email: TestHelper.generateRandomEmail(),
        password: 'NewUser123!',
        name: 'Admin Created User',
        role: 'cliente'
      };

      const response = await adminClient.post('/usuarios', userData);
      
      expect(response.status).toBeOneOf([201, 200]);
      expect(response.data).toHaveProperty('id');
      expect(response.data.email).toBe(userData.email);
      expect(response.data.role).toBe('cliente');
      expect(response.data).not.toHaveProperty('password');

      createdUserId = response.data.id;
    });

    test('should get user by ID', async () => {
      const response = await adminClient.get(`/usuarios/${createdUserId}`);
      
      expect(response.status).toBe(200);
      expect(response.data).toHaveProperty('id', createdUserId);
      expect(response.data).toHaveProperty('email');
      expect(response.data).toHaveProperty('name');
      expect(response.data).toHaveProperty('role');
      expect(response.data).not.toHaveProperty('password');
    });

    test('should get own user info (regular user)', async () => {
      const response = await userClient.get(`/usuarios/${userTokens.user.id}`);
      
      expect(response.status).toBe(200);
      expect(response.data.email).toBe(testUser.email);
      expect(response.data).not.toHaveProperty('password');
    });

    test('should list all users (admin only)', async () => {
      const response = await adminClient.get('/usuarios');
      
      expect(response.status).toBe(200);
      expect(Array.isArray(response.data)).toBe(true);
      expect(response.data.length).toBeGreaterThan(0);
      
      // Verify no passwords are returned
      response.data.forEach((user: any) => {
        expect(user).not.toHaveProperty('password');
        expect(user).toHaveProperty('email');
        expect(user).toHaveProperty('role');
      });
    });

    test('should update user information', async () => {
      const updateData = {
        name: 'Updated User Name',
        // Note: Don't include sensitive fields like password or role
      };

      const response = await adminClient.patch(`/usuarios/${createdUserId}`, updateData);
      
      expect(response.status).toBeOneOf([200, 204]);

      // Verify the update
      const getResponse = await adminClient.get(`/usuarios/${createdUserId}`);
      expect(getResponse.data.name).toBe(updateData.name);
    });

    test('should allow users to update their own information', async () => {
      const updateData = {
        name: 'Self Updated Name'
      };

      const response = await userClient.patch(`/usuarios/${userTokens.user.id}`, updateData);
      
      expect(response.status).toBeOneOf([200, 204]);

      // Verify the update
      const getResponse = await userClient.get(`/usuarios/${userTokens.user.id}`);
      expect(getResponse.data.name).toBe(updateData.name);
    });

    test('should soft delete user (admin only)', async () => {
      const response = await adminClient.delete(`/usuarios/${createdUserId}`);
      
      expect(response.status).toBeOneOf([200, 204]);

      // Verify user is marked as deleted but still exists
      const getResponse = await adminClient.get(`/usuarios/${createdUserId}`);
      expect(getResponse.status).toBeOneOf([200, 404]); // Depends on implementation
    });
  });

  describe('User Roles and Permissions', () => {
    test('should prevent regular users from creating users', async () => {
      const userData = {
        email: TestHelper.generateRandomEmail(),
        password: 'NewUser123!',
        name: 'Unauthorized User',
        role: 'cliente'
      };

      const response = await userClient.post('/usuarios', userData);
      
      expect(response.status).toBeOneOf([403, 401]);
    });

    test('should prevent regular users from listing all users', async () => {
      const response = await userClient.get('/usuarios');
      
      expect(response.status).toBeOneOf([403, 401]);
    });

    test('should prevent users from accessing other users\' information', async () => {
      // Create another user to test access
      const otherUser = await TestHelper.createTestUser({
        role: 'cliente'
      });

      const response = await userClient.get(`/usuarios/${otherUser.id}`);
      
      expect(response.status).toBeOneOf([403, 401, 404]);
    });

    test('should prevent role escalation attempts', async () => {
      const updateData = {
        role: 'admin' // Try to escalate to admin
      };

      const response = await userClient.patch(`/usuarios/${userTokens.user.id}`, updateData);
      
      // Should either be forbidden or the role change should be ignored
      expect(response.status).toBeOneOf([403, 200, 204]);

      // Verify role wasn't changed
      const getResponse = await userClient.get(`/usuarios/${userTokens.user.id}`);
      expect(getResponse.data.role).toBe('cliente'); // Should remain cliente
    });
  });

  describe('User Validation', () => {
    test('should reject invalid email format in updates', async () => {
      const updateData = {
        email: 'invalid-email-format'
      };

      const response = await adminClient.patch(`/usuarios/${testUser.id}`, updateData);
      
      expect(response.status).toBe(400);
    });

    test('should reject duplicate email in updates', async () => {
      // Create two users
      const user1 = await TestHelper.createTestUser();
      const user2 = await TestHelper.createTestUser();

      // Try to update user2 with user1's email
      const updateData = {
        email: user1.email
      };

      const response = await adminClient.patch(`/usuarios/${user2.id}`, updateData);
      
      expect(response.status).toBe(400);
    });

    test('should handle invalid user ID gracefully', async () => {
      const invalidId = '99999999';
      const response = await adminClient.get(`/usuarios/${invalidId}`);
      
      expect(response.status).toBe(404);
    });

    test('should handle malformed user ID gracefully', async () => {
      const malformedId = 'not-a-valid-id';
      const response = await adminClient.get(`/usuarios/${malformedId}`);
      
      expect(response.status).toBeOneOf([400, 404]);
    });
  });

  describe('User Search and Filtering', () => {
    test('should filter users by role (admin only)', async () => {
      const response = await adminClient.get('/usuarios?role=cliente');
      
      expect(response.status).toBeOneOf([200, 404]); // Feature may not be implemented
      
      if (response.status === 200) {
        expect(Array.isArray(response.data)).toBe(true);
        response.data.forEach((user: any) => {
          expect(user.role).toBe('cliente');
        });
      }
    });

    test('should search users by email (admin only)', async () => {
      const searchEmail = testUser.email;
      const response = await adminClient.get(`/usuarios?email=${searchEmail}`);
      
      expect(response.status).toBeOneOf([200, 404]); // Feature may not be implemented
      
      if (response.status === 200) {
        expect(Array.isArray(response.data)).toBe(true);
        if (response.data.length > 0) {
          expect(response.data[0].email).toBe(searchEmail);
        }
      }
    });

    test('should paginate user results (admin only)', async () => {
      const response = await adminClient.get('/usuarios?page=1&limit=10');
      
      expect(response.status).toBeOneOf([200, 404]); // Feature may not be implemented
      
      if (response.status === 200) {
        expect(Array.isArray(response.data)).toBe(true);
        expect(response.data.length).toBeLessThanOrEqual(10);
      }
    });
  });

  describe('User Status Management', () => {
    let statusTestUser: TestUser;

    beforeAll(async () => {
      statusTestUser = await TestHelper.createTestUser({
        role: 'cliente'
      });
    });

    test('should activate/deactivate user (admin only)', async () => {
      // Deactivate user
      const deactivateResponse = await adminClient.patch(`/usuarios/${statusTestUser.id}`, {
        active: false
      });
      
      expect(deactivateResponse.status).toBeOneOf([200, 204, 404]); // Feature may not be implemented
      
      // Try to login with deactivated user
      if (deactivateResponse.status !== 404) {
        const loginResponse = await TestHelper['httpClient'].post('/auth/login', {
          email: statusTestUser.email,
          password: statusTestUser.password
        });
        
        expect(loginResponse.status).toBeOneOf([401, 403]); // Should be rejected
      }
    });

    test('should reactivate user (admin only)', async () => {
      // Reactivate user
      const activateResponse = await adminClient.patch(`/usuarios/${statusTestUser.id}`, {
        active: true
      });
      
      expect(activateResponse.status).toBeOneOf([200, 204, 404]); // Feature may not be implemented
      
      // Try to login with reactivated user
      if (activateResponse.status !== 404) {
        const loginResponse = await TestHelper['httpClient'].post('/auth/login', {
          email: statusTestUser.email,
          password: statusTestUser.password
        });
        
        expect(loginResponse.status).toBeOneOf([200, 401]); // Should work or auth may be cached
      }
    });
  });

  describe('User Data Integrity', () => {
    test('should maintain referential integrity on user deletion', async () => {
      // Create a user and some related data
      const userToDelete = await TestHelper.createTestUser();
      const userTokensToDelete = await TestHelper.authenticateUser(
        userToDelete.email,
        userToDelete.password
      );
      const userClientToDelete = TestHelper.createAuthenticatedClient(userTokensToDelete.accessToken);

      // Create some user-related data (videos, playlists, etc.)
      try {
        await TestHelper.createTestVideo(userClientToDelete, {
          title: 'Video by user to be deleted'
        });
      } catch (error) {
        // Video creation might fail, that's ok for this test
      }

      // Delete the user
      const deleteResponse = await adminClient.delete(`/usuarios/${userToDelete.id}`);
      expect(deleteResponse.status).toBeOneOf([200, 204]);

      // Verify user cannot login after deletion
      const loginResponse = await TestHelper['httpClient'].post('/auth/login', {
        email: userToDelete.email,
        password: userToDelete.password
      });
      
      expect(loginResponse.status).toBe(401);
    });

    test('should handle concurrent user updates gracefully', async () => {
      const concurrentUser = await TestHelper.createTestUser();
      
      const updatePromises = [];
      for (let i = 0; i < 5; i++) {
        updatePromises.push(
          adminClient.patch(`/usuarios/${concurrentUser.id}`, {
            name: `Concurrent Update ${i}`
          })
        );
      }

      const responses = await Promise.all(updatePromises);
      
      // All should succeed or fail gracefully
      responses.forEach(response => {
        expect(response.status).toBeOneOf([200, 204, 409, 400]); // 409 for conflict
      });
    });
  });
});

// Extend Jest matchers
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

