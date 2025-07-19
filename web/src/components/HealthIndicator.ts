// Health indicator component for backend connection status
import { apiClient, ApiError } from '../services/api.js';
import { store } from '../utils/state.js';
import { createElementWithClasses, createBadge } from '../utils/dom.js';
import type { HealthResponse } from '../types/api.js';

export class HealthIndicator {
  private element: HTMLElement;
  private intervalId: number | null = null;
  private refreshInterval = 30000; // 30 seconds

  constructor() {
    this.element = this.createElement();
    this.startAutoRefresh();
    this.checkHealth(); // Initial check
  }

  private createElement(): HTMLElement {
    const container = createElementWithClasses('div', 'health-indicator');
    
    const label = createElementWithClasses('span', 'health-label', {
      textContent: 'Backend Status:'
    });
    
    const status = createElementWithClasses('div', 'health-status');
    const badge = createBadge('Checking...', 'secondary');
    badge.id = 'health-badge';
    
    status.appendChild(badge);
    container.appendChild(label);
    container.appendChild(status);
    
    // Subscribe to state changes
    store.subscribe((state) => {
      this.updateDisplay(state.health, state.loading.health, state.errors.health);
    });
    
    return container;
  }

  private updateDisplay(health: HealthResponse | null, loading: boolean, error: string | null): void {
    const badge = this.element.querySelector('#health-badge') as HTMLElement;
    if (!badge) return;

    // Clear existing classes
    badge.className = 'badge';
    
    if (loading) {
      badge.textContent = 'Checking...';
      badge.classList.add('badge--secondary');
      return;
    }

    if (error) {
      badge.textContent = 'Error';
      badge.classList.add('badge--error');
      badge.title = error;
      return;
    }

    if (!health) {
      badge.textContent = 'Unknown';
      badge.classList.add('badge--warning');
      return;
    }

    // Determine overall health status
    const isHealthy = health.status === 'ok' && 
                     health.database.status === 'connected' && 
                     health.plex.status === 'connected';

    if (isHealthy) {
      badge.textContent = 'Healthy';
      badge.classList.add('badge--success');
      badge.title = `All systems operational. Database: ${health.database.total_entries || 0} entries`;
    } else {
      badge.textContent = 'Issues';
      badge.classList.add('badge--warning');
      
      const issues = [];
      if (health.database.status !== 'connected') {
        issues.push(`Database: ${health.database.status}`);
      }
      if (health.plex.status !== 'connected') {
        issues.push(`Plex: ${health.plex.status}`);
      }
      badge.title = issues.join(', ');
    }
  }

  private async checkHealth(): Promise<void> {
    store.setLoading('health', true);
    store.clearError('health');

    try {
      const health = await apiClient.getHealth();
      store.setState({ health });
    } catch (error) {
      console.error('Health check failed:', error);
      const errorMessage = error instanceof ApiError ? error.message : 'Network error';
      store.setError('health', errorMessage);
    } finally {
      store.setLoading('health', false);
    }
  }

  private startAutoRefresh(): void {
    this.intervalId = window.setInterval(() => {
      this.checkHealth();
    }, this.refreshInterval);
  }

  private stopAutoRefresh(): void {
    if (this.intervalId !== null) {
      clearInterval(this.intervalId);
      this.intervalId = null;
    }
  }

  // Public methods
  getElement(): HTMLElement {
    return this.element;
  }

  refresh(): void {
    this.checkHealth();
  }

  destroy(): void {
    this.stopAutoRefresh();
    this.element.remove();
  }

  setRefreshInterval(intervalMs: number): void {
    this.refreshInterval = intervalMs;
    if (this.intervalId !== null) {
      this.stopAutoRefresh();
      this.startAutoRefresh();
    }
  }
}