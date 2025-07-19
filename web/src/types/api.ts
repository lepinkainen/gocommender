// API Types for GoCommender Frontend
// These types match the Go models in internal/models/

export interface Artist {
  mbid: string;
  name: string;
  album_count: number;
  years_active: string;
  description: string;
  genres: string[];
  country: string;
  image_url: string;
  verified: Record<string, boolean>;
  external_urls: ExternalURLs;
  last_updated: string;
}

export interface ExternalURLs {
  discogs?: string;
  musicbrainz?: string;
  lastfm?: string;
  spotify?: string;
}

export interface PlexPlaylist {
  name: string;
  type: string;
  smart: boolean;
  track_count: number;
  duration: number; // milliseconds
  tracks: PlexTrack[];
}

export interface PlexTrack {
  title: string;
  artist: string;
  album: string;
  year: number;
  rating: number; // 1-10 scale
  play_count: number;
  last_played: string;
}

export interface RecommendRequest {
  playlist_name: string;
  genre?: string;
  max_results: number;
}

export interface RecommendResponse {
  status: string;
  request_id: string;
  suggestions: Artist[];
  metadata: RecommendMetadata;
  error?: string;
}

export interface RecommendMetadata {
  seed_track_count: number;
  known_artist_count: number;
  processing_time: string;
  cache_hits: number;
  api_calls_made: number;
  generated_at: string;
}

// Health endpoint response
export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
  version: string;
  database: {
    status: string;
    total_entries?: number;
    valid_entries?: number;
    expired_entries?: number;
    error?: string;
  };
  plex: {
    status: string;
    error?: string;
  };
}

// Build info response
export interface BuildInfo {
  version: string;
  commit: string;
  build_date: string;
  go_version: string;
  platform: string;
}

// Playlists API response
export interface PlaylistsResponse {
  playlists: PlexPlaylist[];
  count: number;
}

// Artist API response
export interface ArtistResponse {
  artist: Artist;
  needs_fetch: boolean;
}

// Generic API error response
export interface ApiError {
  error: string;
  status: string;
  timestamp: string;
}

// Loading states for UI
export interface LoadingState {
  playlists: boolean;
  recommendations: boolean;
  artist: boolean;
  health: boolean;
}

// Form data types
export interface RecommendFormData {
  selectedPlaylist: string;
  genre: string;
  maxResults: number;
}

// Application state
export interface AppState {
  playlists: PlexPlaylist[];
  selectedPlaylist: string | null;
  recommendations: Artist[];
  currentArtist: Artist | null;
  health: HealthResponse | null;
  loading: LoadingState;
  errors: {
    playlists: string | null;
    recommendations: string | null;
    artist: string | null;
    health: string | null;
  };
  form: RecommendFormData;
}