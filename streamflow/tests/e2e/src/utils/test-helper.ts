import axios, { AxiosInstance } from 'axios';
import { v4 as uuidv4 } from 'uuid';

export interface TestUser {
  id?: string;
  email: string;
  password: string;
  role: 'admin' | 'cliente';
  name?: string;
}

export interface TestVideo {
  id?: string;
  title: string;
  genre: string;
  description?: string;
  duration?: number;
}

export interface TestPlaylist {
  id?: string;
  name: string;
  description?: string;
  videos?: string[];
}

export interface AuthTokens {
  accessToken: string;
  refreshToken?: string;
  user: any;
}

export class TestHelper {
  private static baseUrl = process.env.BASE_URL || 'http://localhost:80';
  private static apiUrl = process.env.API_BASE_URL || 'http://localhost:8080';
  private static authUrl = process.env.AUTH_SERVICE_URL || 'http://localhost:8001';
  
  private static httpClient: AxiosInstance;
  private static authClient: AxiosInstance;

  static {
    this.httpClient = axios.create({
      baseURL: this.baseUrl,
      timeout: 10000,
      validateStatus: () => true, // Don't throw on HTTP error codes
    });

    this.authClient = axios.create({
      baseURL: this.authUrl,
      timeout: 10000,
      validateStatus: () => true,
    });
  }

  /**
   * Wait for all services to be ready
   */
  static async waitForServices(): Promise<void> {
    const services = [
      { name: 'Nginx Load Balancer', url: `${this.baseUrl}/health` },
      { name: 'API Gateway', url: `${this.apiUrl}/health` },
      { name: 'Auth Service', url: `${this.authUrl}/health` },
    ];

    const maxRetries = parseInt(process.env.HEALTH_CHECK_RETRIES || '10');
    const delay = parseInt(process.env.HEALTH_CHECK_DELAY || '5000');

    for (const service of services) {
      await this.waitForService(service.name, service.url, maxRetries, delay);
    }
  }

  /**
   * Wait for a specific service to be ready
   */
  private static async waitForService(
    serviceName: string,
    url: string,
    maxRetries: number,
    delay: number
  ): Promise<void> {
    for (let i = 0; i < maxRetries; i++) {
      try {
        const response = await axios.get(url, { timeout: 5000 });
        if (response.status === 200) {
          console.log(`✅ ${serviceName} is ready`);
          return;
        }
      } catch (error) {
        console.log(`⏳ Waiting for ${serviceName}... (${i + 1}/${maxRetries})`);
      }
      
      if (i < maxRetries - 1) {
        await this.sleep(delay);
      }
    }
    
    throw new Error(`❌ ${serviceName} failed to start after ${maxRetries} retries`);
  }

  /**
   * Clean up test data
   */
  static async cleanupTestData(): Promise<void> {
    // This is a placeholder - implement actual cleanup based on your data structure
    console.log('🧹 Cleaning up test data...');
  }

  /**
   * Create a test user
   */
  static async createTestUser(userData?: Partial<TestUser>): Promise<TestUser> {
    const user: TestUser = {
      email: `test-${uuidv4()}@streamflow.com`,
      password: 'Test123!',
      role: 'cliente',
      name: 'Test User',
      ...userData,
    };

    const response = await this.httpClient.post('/usuarios', user);
    
    if (response.status !== 201) {
      throw new Error(`Failed to create test user: ${response.data?.message || 'Unknown error'}`);
    }

    return { ...user, id: response.data.id };
  }

  /**
   * Authenticate a user and return tokens
   */
  static async authenticateUser(email: string, password: string): Promise<AuthTokens> {
    const response = await this.httpClient.post('/auth/login', {
      email,
      password,
    });

    if (response.status !== 200) {
      throw new Error(`Authentication failed: ${response.data?.message || 'Invalid credentials'}`);
    }

    return {
      accessToken: response.data.access_token,
      refreshToken: response.data.refresh_token,
      user: response.data.user,
    };
  }

  /**
   * Create an authenticated HTTP client
   */
  static createAuthenticatedClient(accessToken: string): AxiosInstance {
    return axios.create({
      baseURL: this.baseUrl,
      timeout: 10000,
      headers: {
        'Authorization': `Bearer ${accessToken}`,
        'Content-Type': 'application/json',
      },
      validateStatus: () => true,
    });
  }

  /**
   * Get admin authentication tokens
   */
  static async getAdminTokens(): Promise<AuthTokens> {
    const adminEmail = process.env.TEST_ADMIN_EMAIL || 'admin@streamflow.com';
    const adminPassword = process.env.TEST_ADMIN_PASSWORD || 'admin123';
    
    return await this.authenticateUser(adminEmail, adminPassword);
  }

  /**
   * Create a test video
   */
  static async createTestVideo(
    authClient: AxiosInstance,
    videoData?: Partial<TestVideo>
  ): Promise<TestVideo> {
    const video: TestVideo = {
      title: `Test Video ${uuidv4()}`,
      genre: 'Drama',
      description: 'Test video description',
      duration: 120,
      ...videoData,
    };

    const response = await authClient.post('/videos', video);
    
    if (response.status !== 201) {
      throw new Error(`Failed to create test video: ${response.data?.message || 'Unknown error'}`);
    }

    return { ...video, id: response.data.id };
  }

  /**
   * Create a test playlist
   */
  static async createTestPlaylist(
    authClient: AxiosInstance,
    playlistData?: Partial<TestPlaylist>
  ): Promise<TestPlaylist> {
    const playlist: TestPlaylist = {
      name: `Test Playlist ${uuidv4()}`,
      description: 'Test playlist description',
      videos: [],
      ...playlistData,
    };

    const response = await authClient.post('/listas-reproduccion', playlist);
    
    if (response.status !== 201) {
      throw new Error(`Failed to create test playlist: ${response.data?.message || 'Unknown error'}`);
    }

    return { ...playlist, id: response.data.id };
  }

  /**
   * Generate random test data
   */
  static generateRandomEmail(): string {
    return `test-${uuidv4()}@streamflow.com`;
  }

  static generateRandomString(length: number = 8): string {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    for (let i = 0; i < length; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
  }

  /**
   * Utility function to sleep
   */
  static sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * Retry a function with exponential backoff
   */
  static async retry<T>(
    fn: () => Promise<T>,
    retries: number = 3,
    delay: number = 1000
  ): Promise<T> {
    for (let i = 0; i < retries; i++) {
      try {
        return await fn();
      } catch (error) {
        if (i === retries - 1) {
          throw error;
        }
        await this.sleep(delay * Math.pow(2, i));
      }
    }
    throw new Error('Retry failed');
  }

  /**
   * Assert HTTP response status and return data
   */
  static assertResponse(response: any, expectedStatus: number, message?: string): any {
    if (response.status !== expectedStatus) {
      throw new Error(
        message || 
        `Expected status ${expectedStatus}, got ${response.status}: ${
          response.data?.message || JSON.stringify(response.data)
        }`
      );
    }
    return response.data;
  }

  /**
   * Wait for a condition to be true
   */
  static async waitFor(
    condition: () => Promise<boolean>,
    timeout: number = 10000,
    interval: number = 1000
  ): Promise<void> {
    const start = Date.now();
    
    while (Date.now() - start < timeout) {
      if (await condition()) {
        return;
      }
      await this.sleep(interval);
    }
    
    throw new Error(`Condition not met within ${timeout}ms`);
  }
}

