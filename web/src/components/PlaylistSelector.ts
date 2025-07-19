// Playlist selector component for choosing Plex playlists
import { apiClient, ApiError } from '../services/api.js';
import { store } from '../utils/state.js';
import { createElementWithClasses, createSelect, createButton, createLoadingSpinner, createErrorMessage } from '../utils/dom.js';
import type { PlexPlaylist } from '../types/api.js';

export class PlaylistSelector {
  private element: HTMLElement;
  private selectElement: HTMLSelectElement | null = null;

  constructor() {
    this.element = this.createElement();
    this.loadPlaylists(); // Initial load
  }

  private createElement(): HTMLElement {
    const container = createElementWithClasses('div', 'playlist-selector');
    
    const header = createElementWithClasses('div', 'playlist-header');
    const title = createElementWithClasses('h3', 'playlist-title', {
      textContent: 'Select Playlist'
    });
    
    const refreshButton = createButton('Refresh', () => this.loadPlaylists(), 'outline');
    refreshButton.classList.add('playlist-refresh');
    
    header.appendChild(title);
    header.appendChild(refreshButton);
    container.appendChild(header);
    
    const content = createElementWithClasses('div', 'playlist-content');
    content.id = 'playlist-content';
    container.appendChild(content);
    
    // Subscribe to state changes
    store.subscribe((state) => {
      this.updateDisplay(state.playlists, state.loading.playlists, state.errors.playlists, state.form.selectedPlaylist);
    });
    
    return container;
  }

  private updateDisplay(
    playlists: PlexPlaylist[], 
    loading: boolean, 
    error: string | null,
    selectedPlaylist: string
  ): void {
    const content = this.element.querySelector('#playlist-content') as HTMLElement;
    if (!content) return;

    // Clear previous content
    content.innerHTML = '';

    if (loading) {
      const spinner = createLoadingSpinner('medium');
      const loadingText = createElementWithClasses('p', 'loading-text', {
        textContent: 'Loading playlists...'
      });
      content.appendChild(spinner);
      content.appendChild(loadingText);
      return;
    }

    if (error) {
      const errorElement = createErrorMessage(error, false);
      content.appendChild(errorElement);
      return;
    }

    if (playlists.length === 0) {
      const emptyMessage = createElementWithClasses('p', 'empty-message', {
        textContent: 'No playlists found. Make sure your Plex server is configured and has music playlists.'
      });
      content.appendChild(emptyMessage);
      return;
    }

    // Create select element
    const options = playlists.map(playlist => ({
      value: playlist.name,
      text: `${playlist.name} (${playlist.track_count} tracks)${playlist.smart ? ' - Smart' : ''}`,
      selected: playlist.name === selectedPlaylist
    }));

    this.selectElement = createSelect(options, 'Choose a playlist...');
    this.selectElement.addEventListener('change', (e) => {
      const target = e.target as HTMLSelectElement;
      store.updateForm({ selectedPlaylist: target.value });
      store.setState({ selectedPlaylist: target.value || null });
    });

    // Set initial value if we have a selected playlist
    if (selectedPlaylist) {
      this.selectElement.value = selectedPlaylist;
    }

    content.appendChild(this.selectElement);

    // Add playlist stats
    if (playlists.length > 0) {
      const stats = this.createPlaylistStats(playlists);
      content.appendChild(stats);
    }
  }

  private createPlaylistStats(playlists: PlexPlaylist[]): HTMLElement {
    const stats = createElementWithClasses('div', 'playlist-stats');
    
    const totalPlaylists = playlists.length;
    const smartPlaylists = playlists.filter(p => p.smart).length;
    const totalTracks = playlists.reduce((sum, p) => sum + p.track_count, 0);
    
    const statsText = createElementWithClasses('p', 'stats-text', {
      textContent: `${totalPlaylists} playlists found (${smartPlaylists} smart) with ${totalTracks.toLocaleString()} total tracks`
    });
    
    stats.appendChild(statsText);
    return stats;
  }

  private async loadPlaylists(): Promise<void> {
    store.setLoading('playlists', true);
    store.clearError('playlists');

    try {
      const response = await apiClient.getPlaylists();
      store.setState({ playlists: response.playlists });
    } catch (error) {
      console.error('Failed to load playlists:', error);
      const errorMessage = error instanceof ApiError ? error.message : 'Failed to load playlists';
      store.setError('playlists', errorMessage);
    } finally {
      store.setLoading('playlists', false);
    }
  }

  // Public methods
  getElement(): HTMLElement {
    return this.element;
  }

  getSelectedPlaylist(): string | null {
    return this.selectElement?.value || null;
  }

  refresh(): void {
    this.loadPlaylists();
  }

  reset(): void {
    if (this.selectElement) {
      this.selectElement.value = '';
    }
    store.updateForm({ selectedPlaylist: '' });
    store.setState({ selectedPlaylist: null });
  }
}