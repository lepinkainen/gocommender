# 02 - Data Models

## Overview

Define the core data structures with MusicBrainz ID as primary key and create the SQLite database schema for caching artist information.

## Steps

### 1. Define Artist Data Model

- [x] Create `internal/models/artist.go` with comprehensive artist information

```go
package models

import (
    "database/sql/driver"
    "encoding/json"
    "time"
)

// Artist represents a musical artist with enriched metadata
type Artist struct {
    MBID         string            `json:"mbid" db:"mbid"`                    // MusicBrainz ID (primary key)
    Name         string            `json:"name" db:"name"`
    Verified     VerificationMap   `json:"verified" db:"verified_json"`      // Service verification status
    AlbumCount   int               `json:"album_count" db:"album_count"`
    YearsActive  string            `json:"years_active" db:"years_active"`   // e.g., "1970-present", "1980-1995"
    Description  string            `json:"description" db:"description"`     // Biography/description
    Genres       []string          `json:"genres" db:"genres_json"`
    Country      string            `json:"country" db:"country"`
    ImageURL     string            `json:"image_url" db:"image_url"`
    ExternalURLs ExternalURLs      `json:"external_urls" db:"external_urls_json"`
    LastUpdated  time.Time         `json:"last_updated" db:"last_updated"`
    CacheExpiry  time.Time         `json:"-" db:"cache_expiry"`
}

// VerificationMap tracks which services have verified this artist
type VerificationMap map[string]bool

// ExternalURLs contains links to external services
type ExternalURLs struct {
    Discogs     string `json:"discogs,omitempty"`
    MusicBrainz string `json:"musicbrainz,omitempty"`      // Full URL to MB page
    LastFM      string `json:"lastfm,omitempty"`
    Spotify     string `json:"spotify,omitempty"`
}
```

### 2. Implement JSON Serialization for Custom Types

- [x] Add Value/Scan methods for database compatibility

```go
// VerificationMap database methods
func (v VerificationMap) Value() (driver.Value, error) {
    return json.Marshal(v)
}

func (v *VerificationMap) Scan(value interface{}) error {
    if value == nil {
        *v = make(VerificationMap)
        return nil
    }
    return json.Unmarshal(value.([]byte), v)
}

// Similar methods for ExternalURLs and []string (Genres)
```

### 3. Create Database Schema

- [x] Define `internal/models/schema.sql` with artist table structure

```sql
CREATE TABLE IF NOT EXISTS artists (
    mbid TEXT PRIMARY KEY,                    -- MusicBrainz ID
    name TEXT NOT NULL,
    verified_json TEXT DEFAULT '{}',         -- JSON: {"discogs": true, "musicbrainz": true, "lastfm": false}
    album_count INTEGER DEFAULT 0,
    years_active TEXT DEFAULT '',
    description TEXT DEFAULT '',
    genres_json TEXT DEFAULT '[]',           -- JSON array: ["rock", "alternative"]
    country TEXT DEFAULT '',
    image_url TEXT DEFAULT '',
    external_urls_json TEXT DEFAULT '{}',    -- JSON: {"discogs": "url", "musicbrainz": "url"}
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
    cache_expiry DATETIME NOT NULL
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_cache_expiry ON artists(cache_expiry);
CREATE INDEX IF NOT EXISTS idx_last_updated ON artists(last_updated);
CREATE INDEX IF NOT EXISTS idx_name ON artists(name);
CREATE INDEX IF NOT EXISTS idx_verified ON artists(verified_json);
```

### 4. Create Recommendation Models

- [x] Define request/response structures in `internal/models/recommendation.go`

```go
// RecommendRequest represents the API request for recommendations
type RecommendRequest struct {
    PlaylistName string  `json:"playlist_name" validate:"required"`
    Genre        *string `json:"genre,omitempty"`
    MaxResults   int     `json:"max_results,omitempty"`     // Default: 5
}

// RecommendResponse represents the API response
type RecommendResponse struct {
    Status       string            `json:"status"`
    RequestID    string            `json:"request_id"`
    Suggestions  []Artist          `json:"suggestions"`
    Metadata     RecommendMetadata `json:"metadata"`
    Error        string            `json:"error,omitempty"`
}

// RecommendMetadata provides context about the recommendation
type RecommendMetadata struct {
    SeedTrackCount   int       `json:"seed_track_count"`
    KnownArtistCount int       `json:"known_artist_count"`
    ProcessingTime   string    `json:"processing_time"`
    CacheHits        int       `json:"cache_hits"`
    APICallsMade     int       `json:"api_calls_made"`
    GeneratedAt      time.Time `json:"generated_at"`
}
```

### 5. Create Plex Models

- [x] Define Plex API structures in `internal/models/plex.go`

```go
// PlexTrack represents a track from Plex playlist
type PlexTrack struct {
    Title       string `json:"title"`
    Artist      string `json:"artist"`
    Album       string `json:"album"`
    Year        int    `json:"year"`
    Rating      int    `json:"rating"`          // 1-10 scale
    PlayCount   int    `json:"play_count"`
    LastPlayed  time.Time `json:"last_played"`
}

// PlexPlaylist represents a Plex playlist
type PlexPlaylist struct {
    Name        string      `json:"name"`
    Type        string      `json:"type"`         // "audio"
    Smart       bool        `json:"smart"`
    TrackCount  int         `json:"track_count"`
    Duration    int         `json:"duration"`     // milliseconds
    Tracks      []PlexTrack `json:"tracks"`
}
```

## Verification Steps

- [ ] **Model Compilation**:

   ```bash
   go build ./internal/models/...
   ```

- [ ] **JSON Serialization Test**:

   ```go
   // Test that custom types serialize/deserialize correctly
   artist := &Artist{MBID: "test", Verified: VerificationMap{"musicbrainz": true}}
   data, err := json.Marshal(artist)
   // Should succeed without errors
   ```

- [ ] **Database Schema Validation**:

   ```bash
   sqlite3 test.db < internal/models/schema.sql
   # Should create tables without errors
   ```

- [ ] **Type Safety Check**:

   ```bash
   go vet ./internal/models/...
   ```

## Dependencies

- Previous: `01_project_setup.md` (Go module and structure)
- Imports: `database/sql/driver`, `encoding/json`, `time`

## Next Steps

Proceed to `03_configuration.md` to implement configuration management using the defined models.

## Notes

- MBID (MusicBrainz ID) serves as the primary key for artists
- All custom types implement driver.Valuer and sql.Scanner for database compatibility
- Verification map allows tracking which external services have confirmed an artist
- Cache expiry enables intelligent refresh of stale data
- Comprehensive metadata helps with API debugging and optimization
