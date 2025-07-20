# 14 - Frontend Implementation with TypeScript + Vite

## Overview
Create a modern, type-safe frontend for GoCommender using Vite + TypeScript with minimal configuration and zero-framework approach.

## Steps

### 1. Project Initialization

- [ ] Create `web/` directory in project root
- [ ] Initialize Vite project: `npm create vite@latest web -- --template vanilla-ts`
- [ ] Install dependencies and verify development server
- [ ] Update root `.gitignore` to include `web/node_modules/` and `web/dist/`

### 2. Project Structure Setup

```
web/
├── src/
│   ├── types/
│   │   └── api.ts          # API response types
│   ├── services/
│   │   └── api.ts          # HTTP client and API calls
│   ├── components/
│   │   ├── PlaylistSelector.ts
│   │   ├── RecommendationForm.ts
│   │   ├── ArtistGrid.ts
│   │   ├── ArtistCard.ts
│   │   └── HealthIndicator.ts
│   ├── utils/
│   │   └── dom.ts          # DOM helpers
│   ├── styles/
│   │   ├── main.css
│   │   └── components.css
│   └── main.ts             # Application entry point
├── index.html
├── vite.config.ts
└── package.json
```

### 3. TypeScript API Types

- [ ] Define interfaces matching GoCommender API responses
  - `Artist`, `PlexPlaylist`, `RecommendRequest`, `RecommendResponse`
  - `HealthResponse`, `ApiError`
- [ ] Create utility types for loading states and form data

```typescript
// Example types structure
interface Artist {
  mbid: string;
  name: string;
  album_count: number;
  years_active: string;
  description: string;
  genres: string[];
  country: string;
  image_url: string;
  verified: Record<string, boolean>;
  external_urls: {
    musicbrainz: string;
    discogs?: string;
    lastfm?: string;
  };
  last_updated: string;
  cache_expiry: string;
}

interface PlexPlaylist {
  name: string;
  type: string;
  smart: boolean;
  track_count: number;
}

interface RecommendRequest {
  playlist_name: string;
  genre?: string;
  max_results: number;
}

interface RecommendResponse {
  recommendations: Artist[];
  metadata: {
    playlist_name: string;
    total_recommendations: number;
    processing_time_ms: number;
  };
}
```

### 4. HTTP Client Implementation

- [ ] Create typed API client with fetch wrapper
- [ ] Implement endpoints:
  - `GET /api/health` - Health check
  - `GET /api/plex/playlists` - List playlists
  - `POST /api/recommend` - Generate recommendations
  - `GET /api/artists/{mbid}` - Artist details
- [ ] Add error handling and response validation
- [ ] Include loading states and retry logic

```typescript
// Example API client structure
class ApiClient {
  private baseUrl: string;
  
  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }
  
  async getHealth(): Promise<HealthResponse> { ... }
  async getPlaylists(): Promise<PlexPlaylist[]> { ... }
  async getRecommendations(request: RecommendRequest): Promise<RecommendResponse> { ... }
  async getArtist(mbid: string): Promise<Artist> { ... }
}
```

### 5. Core Components

#### PlaylistSelector Component
- [ ] Fetch and display Plex playlists
- [ ] Handle loading and error states
- [ ] Emit selection events

#### RecommendationForm Component
- [ ] Form fields: playlist selection, genre filter, max results
- [ ] Form validation and submission
- [ ] Loading state during API calls

#### ArtistGrid Component
- [ ] Display recommended artists in responsive grid
- [ ] Handle empty states and loading
- [ ] Artist card click handling

#### ArtistCard Component
- [ ] Display artist image, name, genres
- [ ] Show verification badges
- [ ] Click handler for detailed view

#### HealthIndicator Component
- [ ] Real-time connection status to backend
- [ ] Visual indicator (green/red/yellow)
- [ ] Auto-refresh health checks

### 6. Application State Management

- [ ] Simple reactive state object for:
  - Current playlist selection
  - Recommendation results
  - Loading states
  - Error messages
- [ ] Event-driven updates between components
- [ ] Persist form state in localStorage

```typescript
// Example state management
interface AppState {
  playlists: PlexPlaylist[];
  selectedPlaylist: string | null;
  recommendations: Artist[];
  loading: {
    playlists: boolean;
    recommendations: boolean;
  };
  errors: {
    playlists: string | null;
    recommendations: string | null;
  };
}
```

### 7. Styling and UX

- [ ] Clean, modern CSS with CSS Grid/Flexbox
- [ ] Responsive design for mobile/desktop
- [ ] Loading spinners and transitions
- [ ] Error message styling
- [ ] Dark/light theme support (optional)

### 8. Main Application Integration

- [ ] Initialize components in `main.ts`
- [ ] Set up event listeners and state management
- [ ] Handle application-level error boundaries
- [ ] Add keyboard navigation support

### 9. Development Configuration

- [ ] Configure Vite for API proxy to backend (development)
- [ ] Set up environment variables for API base URL
- [ ] Configure TypeScript strict mode
- [ ] Add development scripts to `package.json`

```typescript
// vite.config.ts example
export default defineConfig({
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  }
});
```

### 10. Build and Production Setup

- [ ] Configure Vite build for production
- [ ] Set up static file serving
- [ ] Add build scripts to root `Taskfile.yml`
- [ ] Configure CORS handling with backend

## Verification Steps

- [ ] **Development Server**:
  ```bash
  cd web && npm run dev
  # Should start on http://localhost:5173
  ```

- [ ] **Backend Integration**:
  ```bash
  # Start backend: task dev
  # Frontend should connect to http://localhost:8080/api
  ```

- [ ] **Core Workflow**:
  - Load playlists from Plex
  - Submit recommendation request
  - Display artist results
  - View artist details

- [ ] **Build Process**:
  ```bash
  cd web && npm run build
  # Should create dist/ folder with optimized assets
  ```

- [ ] **TypeScript Compilation**:
  ```bash
  cd web && npm run type-check
  # Should pass without errors
  ```

## Dependencies
- Previous: `10_http_api.md` (REST API endpoints)
- Vite + TypeScript (minimal dependencies)
- No external UI frameworks required

## Next Steps
After frontend implementation, proceed with `12_deployment_preparation.md` to include frontend build in deployment pipeline.

## Notes
- Zero-framework approach using vanilla TypeScript
- Type safety for all API interactions
- Minimal build configuration with Vite
- Progressive enhancement friendly
- Easy to extend with additional libraries if needed
- Follows modern web standards and best practices
- Responsive design with CSS Grid/Flexbox
- Error handling and loading states throughout
- Local storage for form persistence
- Keyboard navigation support
- Auto-refresh health monitoring