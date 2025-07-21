// Artist grid component for displaying recommendation results
import { store } from '../utils/state.js';
import { createElementWithClasses, createLoadingSpinner, createErrorMessage } from '../utils/dom.js';
import { ArtistCard } from './ArtistCard.js';
import type { Artist } from '../types/api.js';

export class ArtistGrid {
  private element: HTMLElement;
  private gridContainer!: HTMLElement;
  private artistCards: Map<string, ArtistCard> = new Map();
  private onArtistClickHandler?: (artist: Artist) => void;

  constructor(onArtistClickHandler?: (artist: Artist) => void) {
    this.onArtistClickHandler = onArtistClickHandler;
    this.element = this.createElement();
  }

  private createElement(): HTMLElement {
    const container = createElementWithClasses('div', 'artist-grid-container');
    
    const header = createElementWithClasses('div', 'grid-header');
    const title = createElementWithClasses('h3', 'grid-title', {
      textContent: 'Recommendations'
    });
    header.appendChild(title);
    container.appendChild(header);

    // Grid content area
    const content = createElementWithClasses('div', 'grid-content');
    content.id = 'grid-content';
    
    this.gridContainer = createElementWithClasses('div', 'artist-grid');
    this.gridContainer.id = 'artist-grid';
    
    content.appendChild(this.gridContainer);
    container.appendChild(content);

    // Subscribe to state changes
    store.subscribe((state) => {
      this.updateDisplay(
        state.recommendations, 
        state.loading.recommendations, 
        state.errors.recommendations
      );
    });

    return container;
  }

  private updateDisplay(
    recommendations: Artist[], 
    loading: boolean, 
    error: string | null
  ): void {
    const content = this.element.querySelector('#grid-content') as HTMLElement;
    if (!content) return;

    // Update title with count
    const title = this.element.querySelector('.grid-title') as HTMLElement;
    if (title) {
      if (recommendations.length > 0) {
        title.textContent = `Recommendations (${recommendations.length})`;
      } else {
        title.textContent = 'Recommendations';
      }
    }

    // Clear grid
    this.clearGrid();

    if (loading) {
      this.showLoading();
      return;
    }

    if (error) {
      this.showError(error);
      return;
    }

    if (recommendations.length === 0) {
      this.showEmpty();
      return;
    }

    this.showRecommendations(recommendations);
  }

  private clearGrid(): void {
    this.artistCards.clear();
    this.gridContainer.innerHTML = '';
  }

  private showLoading(): void {
    const loadingContainer = createElementWithClasses('div', 'grid-loading');
    const spinner = createLoadingSpinner('large');
    const text = createElementWithClasses('p', 'loading-text', {
      textContent: 'Generating recommendations...'
    });
    
    loadingContainer.appendChild(spinner);
    loadingContainer.appendChild(text);
    this.gridContainer.appendChild(loadingContainer);
  }

  private showError(error: string): void {
    const errorElement = createErrorMessage(error, false);
    errorElement.classList.add('grid-error');
    this.gridContainer.appendChild(errorElement);
  }

  private showEmpty(): void {
    const emptyContainer = createElementWithClasses('div', 'grid-empty');
    const message = createElementWithClasses('p', 'empty-message', {
      textContent: 'No recommendations yet. Fill out the form above and generate some recommendations!'
    });
    emptyContainer.appendChild(message);
    this.gridContainer.appendChild(emptyContainer);
  }

  private showRecommendations(recommendations: Artist[]): void {
    // Create and add artist cards
    recommendations.forEach((artist, index) => {
      const card = new ArtistCard(artist, this.onArtistClickHandler);
      
      // Add animation delay for staggered appearance
      const cardElement = card.getElement();
      cardElement.style.animationDelay = `${index * 100}ms`;
      cardElement.classList.add('artist-card--animate-in');
      
      this.artistCards.set(artist.mbid, card);
      this.gridContainer.appendChild(cardElement);
    });

    // Add metadata display if available
    const state = store.getState();
    if (state.recommendations.length > 0) {
      this.addMetadataDisplay();
    }
  }

  private addMetadataDisplay(): void {
    // Remove any existing stats displays to prevent duplicates
    const existingStats = this.element.querySelectorAll('.grid-stats');
    existingStats.forEach(stats => stats.remove());
    
    // This would show processing time, cache hits, etc.
    // For now, just show a simple stats footer
    const statsContainer = createElementWithClasses('div', 'grid-stats');
    const state = store.getState();
    
    const statsText = createElementWithClasses('p', 'stats-text', {
      textContent: `Found ${state.recommendations.length} artist recommendations`
    });
    
    statsContainer.appendChild(statsText);
    this.element.appendChild(statsContainer);
  }

  // Public methods
  getElement(): HTMLElement {
    return this.element;
  }

  getArtistCards(): ArtistCard[] {
    return Array.from(this.artistCards.values());
  }

  getArtistCard(mbid: string): ArtistCard | undefined {
    return this.artistCards.get(mbid);
  }

  highlightArtist(mbid: string): void {
    const card = this.artistCards.get(mbid);
    if (card) {
      card.highlight();
      
      // Scroll to card if not in view
      const cardElement = card.getElement();
      cardElement.scrollIntoView({ 
        behavior: 'smooth', 
        block: 'center' 
      });
    }
  }

  setArtistClickHandler(handler: (artist: Artist) => void): void {
    this.onArtistClickHandler = handler;
    
    // Update existing cards
    this.artistCards.forEach(card => {
      card.setClickHandler(handler);
    });
  }

  refresh(): void {
    const state = store.getState();
    this.updateDisplay(
      state.recommendations, 
      state.loading.recommendations, 
      state.errors.recommendations
    );
  }

  clear(): void {
    store.setState({ recommendations: [] });
    store.clearError('recommendations');
  }

  // Filter methods for enhanced functionality
  filterByGenre(genre: string): void {
    this.artistCards.forEach((card) => {
      const artist = card.getArtist();
      const hasGenre = artist.genres?.some(g => 
        g.toLowerCase().includes(genre.toLowerCase())
      );
      
      const cardElement = card.getElement();
      if (hasGenre || !genre) {
        cardElement.style.display = '';
      } else {
        cardElement.style.display = 'none';
      }
    });
  }

  sortBy(criteria: 'name' | 'albums' | 'country'): void {
    const cards = Array.from(this.artistCards.values());
    
    cards.sort((a, b) => {
      const artistA = a.getArtist();
      const artistB = b.getArtist();
      
      switch (criteria) {
        case 'name':
          return artistA.name.localeCompare(artistB.name);
        case 'albums':
          return artistB.album_count - artistA.album_count;
        case 'country':
          return (artistA.country || '').localeCompare(artistB.country || '');
        default:
          return 0;
      }
    });
    
    // Re-append in sorted order
    this.clearGrid();
    cards.forEach(card => {
      this.gridContainer.appendChild(card.getElement());
    });
  }

  getVisibleArtists(): Artist[] {
    return Array.from(this.artistCards.values())
      .filter(card => card.getElement().style.display !== 'none')
      .map(card => card.getArtist());
  }
}