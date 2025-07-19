// Simple reactive state management for GoCommender
import type { AppState, LoadingState, RecommendFormData } from '../types/api.js';

// Event system for state changes
type StateListener<T> = (newState: T, previousState: T) => void;

class EventEmitter<T> {
  private listeners: StateListener<T>[] = [];

  subscribe(listener: StateListener<T>): () => void {
    this.listeners.push(listener);
    return () => {
      const index = this.listeners.indexOf(listener);
      if (index > -1) {
        this.listeners.splice(index, 1);
      }
    };
  }

  emit(newState: T, previousState: T): void {
    this.listeners.forEach(listener => listener(newState, previousState));
  }
}

// Initial state
const initialState: AppState = {
  playlists: [],
  selectedPlaylist: null,
  recommendations: [],
  currentArtist: null,
  health: null,
  loading: {
    playlists: false,
    recommendations: false,
    artist: false,
    health: false,
  },
  errors: {
    playlists: null,
    recommendations: null,
    artist: null,
    health: null,
  },
  form: {
    selectedPlaylist: '',
    genre: '',
    maxResults: 5,
  },
};

// State store class
class StateStore {
  private state: AppState;
  private emitter = new EventEmitter<AppState>();

  constructor(initialState: AppState) {
    this.state = { ...initialState };
    this.loadFromStorage();
  }

  // Get current state
  getState(): AppState {
    return { ...this.state };
  }

  // Subscribe to state changes
  subscribe(listener: StateListener<AppState>): () => void {
    return this.emitter.subscribe(listener);
  }

  // Update state and notify listeners
  setState(updater: Partial<AppState> | ((state: AppState) => Partial<AppState>)): void {
    const previousState = { ...this.state };
    
    if (typeof updater === 'function') {
      const updates = updater(this.state);
      this.state = { ...this.state, ...updates };
    } else {
      this.state = { ...this.state, ...updater };
    }

    this.saveToStorage();
    this.emitter.emit(this.state, previousState);
  }

  // Convenience methods for common state updates
  setLoading(key: keyof LoadingState, loading: boolean): void {
    this.setState(state => ({
      loading: { ...state.loading, [key]: loading }
    }));
  }

  setError(key: keyof AppState['errors'], error: string | null): void {
    this.setState(state => ({
      errors: { ...state.errors, [key]: error }
    }));
  }

  clearError(key: keyof AppState['errors']): void {
    this.setError(key, null);
  }

  clearAllErrors(): void {
    this.setState({
      errors: {
        playlists: null,
        recommendations: null,
        artist: null,
        health: null,
      }
    });
  }

  updateForm(updates: Partial<RecommendFormData>): void {
    this.setState(state => ({
      form: { ...state.form, ...updates }
    }));
  }

  // Persist form data to localStorage
  private saveToStorage(): void {
    try {
      const dataToSave = {
        form: this.state.form,
        selectedPlaylist: this.state.selectedPlaylist,
      };
      localStorage.setItem('gocommender-state', JSON.stringify(dataToSave));
    } catch (error) {
      console.warn('Failed to save state to localStorage:', error);
    }
  }

  // Load form data from localStorage
  private loadFromStorage(): void {
    try {
      const saved = localStorage.getItem('gocommender-state');
      if (saved) {
        const data = JSON.parse(saved);
        this.state = {
          ...this.state,
          form: { ...this.state.form, ...data.form },
          selectedPlaylist: data.selectedPlaylist || null,
        };
      }
    } catch (error) {
      console.warn('Failed to load state from localStorage:', error);
    }
  }

  // Reset state to initial values
  reset(): void {
    this.setState(initialState);
    localStorage.removeItem('gocommender-state');
  }
}

// Create singleton store
export const store = new StateStore(initialState);

// Helper functions for common operations
export function useLoading(key: keyof LoadingState): [boolean, (loading: boolean) => void] {
  const state = store.getState();
  const setLoading = (loading: boolean) => store.setLoading(key, loading);
  return [state.loading[key], setLoading];
}

export function useError(key: keyof AppState['errors']): [string | null, (error: string | null) => void] {
  const state = store.getState();
  const setError = (error: string | null) => store.setError(key, error);
  return [state.errors[key], setError];
}

// Utility to create a reactive element that updates when state changes
export function createReactiveElement<T extends HTMLElement>(
  createElement: () => T,
  updateElement: (element: T, state: AppState) => void
): T {
  const element = createElement();
  
  // Initial update
  updateElement(element, store.getState());
  
  // Subscribe to state changes
  const unsubscribe = store.subscribe((newState) => {
    updateElement(element, newState);
  });
  
  // Clean up on element removal
  const observer = new MutationObserver((mutations) => {
    mutations.forEach((mutation) => {
      mutation.removedNodes.forEach((node) => {
        if (node === element || node.contains?.(element)) {
          unsubscribe();
          observer.disconnect();
        }
      });
    });
  });
  
  observer.observe(document.body, { childList: true, subtree: true });
  
  return element;
}

// Debounce utility for performance
export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: ReturnType<typeof setTimeout>;
  
  return (...args: Parameters<T>) => {
    clearTimeout(timeout);
    timeout = setTimeout(() => func(...args), wait);
  };
}