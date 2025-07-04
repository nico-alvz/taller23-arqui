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

