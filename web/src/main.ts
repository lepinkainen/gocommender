// Main application entry point for GoCommender frontend
import './styles/main.css';
import './styles/components.css';

import { HealthIndicator } from './components/HealthIndicator.js';
import { PlaylistSelector } from './components/PlaylistSelector.js';
import { RecommendationForm } from './components/RecommendationForm.js';
import { ArtistGrid } from './components/ArtistGrid.js';
import { store } from './utils/state.js';
import { createElement, createElementWithClasses } from './utils/dom.js';
import type { Artist } from './types/api.js';

class GoCommenderApp {
  private container: HTMLElement;
  private healthIndicator: HealthIndicator;
  private playlistSelector: PlaylistSelector;
  private recommendationForm: RecommendationForm;
  private artistGrid: ArtistGrid;

  constructor() {
    this.container = this.createAppStructure();
    
    // Initialize components
    this.healthIndicator = new HealthIndicator();
    this.playlistSelector = new PlaylistSelector();
    this.recommendationForm = new RecommendationForm();
    this.artistGrid = new ArtistGrid(this.handleArtistClick.bind(this));

    this.attachComponents();
    this.setupKeyboardNavigation();
    this.setupErrorHandling();
  }

  private createAppStructure(): HTMLElement {
    // Clear existing content
    document.body.innerHTML = '';

    const app = createElementWithClasses('div', 'app');
    
    // Header
    const header = createElementWithClasses('header', 'header');
    const title = createElement('h1', { textContent: 'GoCommender' });
    const subtitle = createElement('p', { 
      textContent: 'AI-powered music discovery using your Plex library' 
    });
    header.appendChild(title);
    header.appendChild(subtitle);
    app.appendChild(header);

    // Health indicator (fixed position)
    const healthContainer = createElementWithClasses('div', 'health-container');
    app.appendChild(healthContainer);

    // Main content grid
    const mainContent = createElementWithClasses('div', 'main-content');
    
    const leftColumn = createElementWithClasses('div', 'left-column');
    const rightColumn = createElementWithClasses('div', 'right-column');
    
    mainContent.appendChild(leftColumn);
    mainContent.appendChild(rightColumn);
    app.appendChild(mainContent);

    // Results section
    const resultsSection = createElementWithClasses('div', 'results-section');
    app.appendChild(resultsSection);

    document.body.appendChild(app);
    return app;
  }

  private attachComponents(): void {
    // Attach health indicator
    const healthContainer = this.container.querySelector('.health-container') as HTMLElement;
    healthContainer.appendChild(this.healthIndicator.getElement());

    // Attach components to columns
    const leftColumn = this.container.querySelector('.left-column') as HTMLElement;
    const rightColumn = this.container.querySelector('.right-column') as HTMLElement;
    const resultsSection = this.container.querySelector('.results-section') as HTMLElement;

    // Left column: Playlist selector
    const playlistCard = createElementWithClasses('div', 'card');
    playlistCard.appendChild(this.playlistSelector.getElement());
    leftColumn.appendChild(playlistCard);

    // Right column: Recommendation form
    const formCard = createElementWithClasses('div', 'card');
    formCard.appendChild(this.recommendationForm.getElement());
    rightColumn.appendChild(formCard);

    // Results section: Artist grid
    resultsSection.appendChild(this.artistGrid.getElement());
  }

  private handleArtistClick(artist: Artist): void {
    // For now, just show artist details in a modal or new view
    this.showArtistDetails(artist);
  }

  private showArtistDetails(artist: Artist): void {
    // Create a simple modal for artist details
    const modal = this.createArtistModal(artist);
    document.body.appendChild(modal);
    
    // Focus management for accessibility
    const closeButton = modal.querySelector('.modal-close') as HTMLElement;
    closeButton?.focus();
  }

  private createArtistModal(artist: Artist): HTMLElement {
    const overlay = createElementWithClasses('div', 'modal-overlay');
    overlay.addEventListener('click', (e) => {
      if (e.target === overlay) {
        overlay.remove();
      }
    });

    const modal = createElementWithClasses('div', 'modal');
    const header = createElementWithClasses('div', 'modal-header');
    
    const title = createElement('h2', { textContent: artist.name });
    const closeButton = createElement('button', {
      className: 'modal-close btn btn--outline',
      textContent: '×'
    });
    closeButton.addEventListener('click', () => overlay.remove());
    
    header.appendChild(title);
    header.appendChild(closeButton);
    modal.appendChild(header);

    const content = createElementWithClasses('div', 'modal-content');
    
    // Artist image
    if (artist.image_url) {
      const image = createElement('img', {
        src: artist.image_url,
        alt: `${artist.name} photo`,
        className: 'modal-artist-image'
      });
      content.appendChild(image);
    }

    // Artist details
    const details = createElementWithClasses('div', 'modal-artist-details');
    
    if (artist.description) {
      const description = createElement('p', { textContent: artist.description });
      details.appendChild(description);
    }

    if (artist.years_active) {
      const years = createElement('p', { textContent: `Active: ${artist.years_active}` });
      details.appendChild(years);
    }

    if (artist.country) {
      const country = createElement('p', { textContent: `Country: ${artist.country}` });
      details.appendChild(country);
    }

    if (artist.album_count > 0) {
      const albums = createElement('p', { 
        textContent: `Albums: ${artist.album_count}` 
      });
      details.appendChild(albums);
    }

    if (artist.genres && artist.genres.length > 0) {
      const genres = createElement('p', { 
        textContent: `Genres: ${artist.genres.join(', ')}` 
      });
      details.appendChild(genres);
    }

    content.appendChild(details);
    modal.appendChild(content);
    overlay.appendChild(modal);

    // Add modal styles
    this.addModalStyles();

    return overlay;
  }

  private addModalStyles(): void {
    if (document.querySelector('#modal-styles')) return;

    const styles = createElement('style', { id: 'modal-styles' });
    styles.textContent = `
      .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.8);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
        padding: 2rem;
      }
      
      .modal {
        background: white;
        border-radius: 1rem;
        max-width: 600px;
        max-height: 80vh;
        overflow-y: auto;
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
      }
      
      .modal-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 1.5rem 1.5rem 1rem 1.5rem;
        border-bottom: 1px solid #eee;
      }
      
      .modal-header h2 {
        margin: 0;
        font-size: 1.5rem;
        font-weight: 600;
      }
      
      .modal-close {
        padding: 0.5rem;
        min-width: auto;
        font-size: 1.5rem;
      }
      
      .modal-content {
        padding: 1.5rem;
      }
      
      .modal-artist-image {
        width: 100%;
        max-width: 200px;
        height: auto;
        border-radius: 0.5rem;
        margin-bottom: 1rem;
      }
      
      .modal-artist-details p {
        margin: 0 0 1rem 0;
        line-height: 1.6;
      }
      
      @media (prefers-color-scheme: dark) {
        .modal {
          background: #2d3748;
          color: #f7fafc;
        }
        
        .modal-header {
          border-bottom-color: #4a5568;
        }
      }
    `;
    document.head.appendChild(styles);
  }

  private setupKeyboardNavigation(): void {
    document.addEventListener('keydown', (e) => {
      // ESC key closes modals
      if (e.key === 'Escape') {
        const modal = document.querySelector('.modal-overlay');
        if (modal) {
          modal.remove();
        }
      }

      // Ctrl/Cmd + R refreshes playlists
      if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
        e.preventDefault();
        this.playlistSelector.refresh();
      }

      // Ctrl/Cmd + Enter submits form
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        const form = document.querySelector('.recommendation-form form') as HTMLFormElement;
        if (form && this.recommendationForm.isValid()) {
          form.dispatchEvent(new Event('submit'));
        }
      }
    });
  }

  private setupErrorHandling(): void {
    // Global error handler for unhandled promise rejections
    window.addEventListener('unhandledrejection', (event) => {
      console.error('Unhandled promise rejection:', event.reason);
      store.setError('recommendations', 'An unexpected error occurred. Please try again.');
    });

    // Global error handler for JavaScript errors
    window.addEventListener('error', (event) => {
      console.error('JavaScript error:', event.error);
      // Don't show errors for missing resources or network issues
      if (!event.filename?.includes('chrome-extension://')) {
        store.setError('recommendations', 'An unexpected error occurred. Please refresh the page.');
      }
    });
  }

  // Public methods for external control
  refresh(): void {
    this.healthIndicator.refresh();
    this.playlistSelector.refresh();
  }

  reset(): void {
    this.playlistSelector.reset();
    this.recommendationForm.reset();
    this.artistGrid.clear();
    store.reset();
  }

  getState() {
    return store.getState();
  }
}

// Initialize the application
const app = new GoCommenderApp();

// Make app available globally for debugging
(window as any).gocommender = app;

// Service worker registration for PWA capabilities (optional)
if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch(() => {
      // Service worker registration failed, but this is not critical
      console.log('Service worker registration failed');
    });
  });
}

console.log('GoCommender frontend initialized successfully!');