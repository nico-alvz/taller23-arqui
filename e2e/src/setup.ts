import dotenv from 'dotenv';
import { TestHelper } from './utils/test-helper';

// Load test environment variables
dotenv.config({ path: '.env.test' });

// Global test setup
beforeAll(async () => {
  console.log('🚀 Starting E2E Tests Setup...');
  
  // Wait for services to be ready
  await TestHelper.waitForServices();
  
  // Clean up test data
  await TestHelper.cleanupTestData();
  
  console.log('✅ E2E Tests Setup Complete');
});

afterAll(async () => {
  console.log('🧹 Cleaning up after E2E Tests...');
  
  // Clean up test data
  await TestHelper.cleanupTestData();
  
  console.log('✅ E2E Tests Cleanup Complete');
});

// Increase timeout for E2E tests
jest.setTimeout(30000);

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

