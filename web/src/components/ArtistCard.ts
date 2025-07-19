// Artist card component for displaying individual artist information
import { createElementWithClasses, createElement, createBadge, truncateText } from '../utils/dom.js';
import type { Artist } from '../types/api.js';

export class ArtistCard {
  private element: HTMLElement;
  private artist: Artist;
  private onClickHandler?: (artist: Artist) => void;

  constructor(artist: Artist, onClickHandler?: (artist: Artist) => void) {
    this.artist = artist;
    this.onClickHandler = onClickHandler;
    this.element = this.createElement();
  }

  private createElement(): HTMLElement {
    const card = createElementWithClasses('div', 'artist-card');
    
    if (this.onClickHandler) {
      card.classList.add('artist-card--clickable');
      card.addEventListener('click', () => this.onClickHandler!(this.artist));
      card.setAttribute('role', 'button');
      card.setAttribute('tabindex', '0');
      
      // Keyboard navigation
      card.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          this.onClickHandler!(this.artist);
        }
      });
    }

    // Artist image
    const imageContainer = createElementWithClasses('div', 'artist-image-container');
    const image = createElement('img', {
      className: 'artist-image',
      src: this.artist.image_url || '/placeholder-artist.jpg',
      alt: `${this.artist.name} photo`,
      loading: 'lazy'
    }) as HTMLImageElement;
    
    // Handle image loading errors
    image.addEventListener('error', () => {
      image.src = this.createPlaceholderImage();
    });
    
    imageContainer.appendChild(image);
    card.appendChild(imageContainer);

    // Artist info
    const info = createElementWithClasses('div', 'artist-info');
    
    // Artist name
    const name = createElementWithClasses('h4', 'artist-name', {
      textContent: truncateText(this.artist.name, 50),
      title: this.artist.name
    });
    info.appendChild(name);

    // Years active
    if (this.artist.years_active) {
      const years = createElementWithClasses('p', 'artist-years', {
        textContent: this.artist.years_active
      });
      info.appendChild(years);
    }

    // Country
    if (this.artist.country) {
      const country = createElementWithClasses('p', 'artist-country', {
        textContent: this.artist.country
      });
      info.appendChild(country);
    }

    // Album count
    if (this.artist.album_count > 0) {
      const albums = createElementWithClasses('p', 'artist-albums', {
        textContent: `${this.artist.album_count} album${this.artist.album_count !== 1 ? 's' : ''}`
      });
      info.appendChild(albums);
    }

    // Genres
    if (this.artist.genres && this.artist.genres.length > 0) {
      const genresContainer = createElementWithClasses('div', 'artist-genres');
      const genresLabel = createElementWithClasses('span', 'genres-label', {
        textContent: 'Genres: '
      });
      genresContainer.appendChild(genresLabel);
      
      this.artist.genres.slice(0, 3).forEach((genre, index) => {
        if (index > 0) {
          genresContainer.appendChild(document.createTextNode(', '));
        }
        const genreBadge = createBadge(genre, 'secondary');
        genreBadge.classList.add('genre-badge');
        genresContainer.appendChild(genreBadge);
      });
      
      if (this.artist.genres.length > 3) {
        genresContainer.appendChild(document.createTextNode(` +${this.artist.genres.length - 3} more`));
      }
      
      info.appendChild(genresContainer);
    }

    // Description preview
    if (this.artist.description) {
      const description = createElementWithClasses('p', 'artist-description', {
        textContent: truncateText(this.artist.description, 120),
        title: this.artist.description
      });
      info.appendChild(description);
    }

    card.appendChild(info);

    // Verification badges
    const verificationContainer = this.createVerificationBadges();
    if (verificationContainer) {
      card.appendChild(verificationContainer);
    }

    // External links
    const linksContainer = this.createExternalLinks();
    if (linksContainer) {
      card.appendChild(linksContainer);
    }

    return card;
  }

  private createVerificationBadges(): HTMLElement | null {
    if (!this.artist.verified || Object.keys(this.artist.verified).length === 0) {
      return null;
    }

    const container = createElementWithClasses('div', 'verification-badges');
    
    Object.entries(this.artist.verified).forEach(([service, verified]) => {
      if (verified) {
        const badge = createBadge(service, 'success');
        badge.classList.add('verification-badge');
        badge.title = `Verified on ${service}`;
        container.appendChild(badge);
      }
    });

    return container.children.length > 0 ? container : null;
  }

  private createExternalLinks(): HTMLElement | null {
    const urls = this.artist.external_urls;
    if (!urls || Object.values(urls).every(url => !url)) {
      return null;
    }

    const container = createElementWithClasses('div', 'external-links');
    
    const linkConfigs = [
      { key: 'musicbrainz', label: 'MusicBrainz', icon: '🎵' },
      { key: 'discogs', label: 'Discogs', icon: '💿' },
      { key: 'lastfm', label: 'Last.fm', icon: '📻' },
      { key: 'spotify', label: 'Spotify', icon: '🎧' }
    ];

    linkConfigs.forEach(({ key, label, icon }) => {
      const url = urls[key as keyof typeof urls];
      if (url) {
        const link = createElement('a', {
          href: url,
          target: '_blank',
          rel: 'noopener noreferrer',
          className: 'external-link',
          title: `View on ${label}`,
          textContent: `${icon} ${label}`
        });
        
        // Prevent card click when clicking links
        link.addEventListener('click', (e) => {
          e.stopPropagation();
        });
        
        container.appendChild(link);
      }
    });

    return container.children.length > 0 ? container : null;
  }

  private createPlaceholderImage(): string {
    // Create a simple SVG placeholder
    const svg = `
      <svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
        <rect width="200" height="200" fill="#f0f0f0"/>
        <text x="100" y="100" font-family="Arial" font-size="16" fill="#999" text-anchor="middle" dy=".3em">
          ${this.artist.name.charAt(0).toUpperCase()}
        </text>
      </svg>
    `;
    return `data:image/svg+xml;base64,${btoa(svg)}`;
  }

  // Public methods
  getElement(): HTMLElement {
    return this.element;
  }

  getArtist(): Artist {
    return this.artist;
  }

  update(artist: Artist): void {
    this.artist = artist;
    const oldElement = this.element;
    this.element = this.createElement();
    oldElement.replaceWith(this.element);
  }

  destroy(): void {
    this.element.remove();
  }

  setClickHandler(handler: (artist: Artist) => void): void {
    this.onClickHandler = handler;
    
    // Update the element to reflect the new click handler
    const oldElement = this.element;
    this.element = this.createElement();
    oldElement.replaceWith(this.element);
  }

  highlight(): void {
    this.element.classList.add('artist-card--highlighted');
    setTimeout(() => {
      this.element.classList.remove('artist-card--highlighted');
    }, 2000);
  }
}