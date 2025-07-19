// HTTP Client for GoCommender API
import type {
  ArtistResponse,
  HealthResponse,
  PlaylistsResponse,
  RecommendRequest,
  RecommendResponse,
  ApiError as ApiErrorType
} from '../types/api.js';

export class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string = '/api') {
    this.baseUrl = baseUrl;
  }

  // Generic fetch wrapper with error handling
  private async fetchApi<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    
    const config: RequestInit = {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    };

    try {
      const response = await fetch(url, config);
      
      if (!response.ok) {
        const errorData: ApiErrorType = await response.json().catch(() => ({
          error: `HTTP ${response.status}: ${response.statusText}`,
          status: 'error',
          timestamp: new Date().toISOString(),
        }));
        throw new ApiError(errorData.error, response.status);
      }

      return await response.json();
    } catch (error) {
      if (error instanceof ApiError) {
        throw error;
      }
      
      // Network or parsing error
      throw new ApiError(
        error instanceof Error ? error.message : 'Network error',
        0
      );
    }
  }

  // Health check endpoint
  async getHealth(): Promise<HealthResponse> {
    return this.fetchApi<HealthResponse>('/health');
  }

  // Get Plex playlists
  async getPlaylists(): Promise<PlaylistsResponse> {
    return this.fetchApi<PlaylistsResponse>('/plex/playlists');
  }

  // Generate recommendations
  async getRecommendations(request: RecommendRequest): Promise<RecommendResponse> {
    return this.fetchApi<RecommendResponse>('/recommend', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  // Get artist details by MBID
  async getArtist(mbid: string): Promise<ArtistResponse> {
    if (!mbid || !this.isValidMBID(mbid)) {
      throw new ApiError('Invalid artist MBID format', 400);
    }
    
    return this.fetchApi<ArtistResponse>(`/artists/${mbid}`);
  }

  // Test Plex connection
  async testPlex(): Promise<{ status: string; server?: any }> {
    return this.fetchApi<{ status: string; server?: any }>('/plex/test');
  }

  // Get cache statistics
  async getCacheStats(): Promise<any> {
    return this.fetchApi<any>('/cache/stats');
  }

  // Clear cache
  async clearCache(type: 'expired' | 'all' = 'expired'): Promise<any> {
    return this.fetchApi<any>(`/cache/clear?type=${type}`, {
      method: 'POST',
    });
  }

  // Helper method to validate MBID format
  private isValidMBID(mbid: string): boolean {
    // Basic UUID format check: 8-4-4-4-12 characters
    const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
    return uuidRegex.test(mbid);
  }
}

// Custom error class for API errors
export class ApiError extends Error {
  constructor(
    message: string,
    public statusCode: number
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

// Create singleton instance
export const apiClient = new ApiClient();

// Helper functions for common operations with loading states
export async function withLoading<T>(
  operation: () => Promise<T>,
  setLoading: (loading: boolean) => void
): Promise<T> {
  setLoading(true);
  try {
    return await operation();
  } finally {
    setLoading(false);
  }
}

// Retry wrapper for transient failures
export async function withRetry<T>(
  operation: () => Promise<T>,
  maxRetries: number = 3,
  delayMs: number = 1000
): Promise<T> {
  let lastError: Error;
  
  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      return await operation();
    } catch (error) {
      lastError = error instanceof Error ? error : new Error(String(error));
      
      // Don't retry on client errors (4xx)
      if (error instanceof ApiError && error.statusCode >= 400 && error.statusCode < 500) {
        throw error;
      }
      
      if (attempt === maxRetries) {
        break;
      }
      
      // Wait before retrying
      await new Promise(resolve => setTimeout(resolve, delayMs * attempt));
    }
  }
  
  throw lastError!;
}