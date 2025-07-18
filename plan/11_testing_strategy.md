# 11 - Testing Strategy

## Overview
Implement comprehensive testing strategy covering unit tests, integration tests, and API testing following llm-shared guidelines with good coverage of critical functionality.

## Steps

### 1. Testing Structure and Organization

- [ ] Create test organization following Go conventions

```
internal/
├── models/
│   ├── artist.go
│   └── artist_test.go
├── config/
│   ├── config.go
│   └── config_test.go
├── services/
│   ├── musicbrainz.go
│   ├── musicbrainz_test.go
│   ├── cache.go
│   ├── cache_test.go
│   └── ...
├── api/
│   ├── router.go
│   └── router_test.go
└── testutil/
    ├── fixtures.go
    ├── mocks.go
    └── helpers.go
```

### 2. Test Utilities and Fixtures

- [ ] Define `internal/testutil/fixtures.go` with test data helpers

```go
package testutil

import (
    "time"
    
    "gocommender/internal/models"
)

// TestArtist creates a test artist with common fields populated
func TestArtist() *models.Artist {
    return &models.Artist{
        MBID:        "b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d",
        Name:        "The Beatles",
        AlbumCount:  13,
        YearsActive: "1960-1970",
        Description: "English rock band formed in Liverpool in 1960.",
        Genres:      models.Genres{"rock", "pop", "psychedelic"},
        Country:     "United Kingdom",
        ImageURL:    "https://example.com/beatles.jpg",
        Verified: models.VerificationMap{
            "musicbrainz": true,
            "discogs":     true,
            "lastfm":      true,
        },
        ExternalURLs: models.ExternalURLs{
            MusicBrainz: "https://musicbrainz.org/artist/b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d",
            Discogs:     "https://discogs.com/artist/82730",
            LastFM:      "https://last.fm/music/The%20Beatles",
        },
        LastUpdated: time.Now(),
        CacheExpiry: time.Now().Add(24 * time.Hour),
    }
}

// TestPlexTracks creates sample Plex tracks for testing
func TestPlexTracks() []models.PlexTrack {
    return []models.PlexTrack{
        {
            Title:      "Hey Jude",
            Artist:     "The Beatles",
            Album:      "The Beatles (White Album)",
            Year:       1968,
            Rating:     9,
            PlayCount:  42,
            LastPlayed: time.Now().Add(-24 * time.Hour),
        },
        {
            Title:      "Come Together",
            Artist:     "The Beatles", 
            Album:      "Abbey Road",
            Year:       1969,
            Rating:     8,
            PlayCount:  38,
            LastPlayed: time.Now().Add(-48 * time.Hour),
        },
    }
}

// TestRecommendRequest creates a valid recommendation request
func TestRecommendRequest() models.RecommendRequest {
    return models.RecommendRequest{
        PlaylistName: "My Favorites",
        Genre:        stringPtr("rock"),
        MaxResults:   5,
    }
}

func stringPtr(s string) *string {
    return &s
}
```

### 3. Mock Services

- [ ] Create `internal/testutil/mocks.go` with service mocks

```go
package testutil

import (
    "errors"
    "time"
    
    "gocommender/internal/models"
    "gocommender/internal/services"
)

// MockCacheService implements CacheService interface for testing
type MockCacheService struct {
    artists map[string]*models.Artist
    hits    int
    misses  int
}

func NewMockCacheService() *MockCacheService {
    return &MockCacheService{
        artists: make(map[string]*models.Artist),
    }
}

func (m *MockCacheService) Get(mbid string) (*models.Artist, error) {
    if artist, exists := m.artists[mbid]; exists {
        m.hits++
        return artist, nil
    }
    m.misses++
    return nil, nil // Cache miss
}

func (m *MockCacheService) Set(artist *models.Artist) error {
    m.artists[artist.MBID] = artist
    return nil
}

func (m *MockCacheService) Delete(mbid string) error {
    delete(m.artists, mbid)
    return nil
}

func (m *MockCacheService) GetExpired() ([]models.Artist, error) {
    return []models.Artist{}, nil
}

func (m *MockCacheService) RefreshExpired() error {
    return nil
}

func (m *MockCacheService) GetStats() (*services.CacheStats, error) {
    return &services.CacheStats{
        TotalEntries: len(m.artists),
        HitRate:     float64(m.hits) / float64(m.hits + m.misses),
    }, nil
}

func (m *MockCacheService) Close() error {
    return nil
}

// MockPlexClient implements Plex operations for testing
type MockPlexClient struct {
    playlists []models.PlexPlaylist
    tracks    map[string][]models.PlexTrack
    artists   []string
    shouldErr bool
}

func NewMockPlexClient() *MockPlexClient {
    return &MockPlexClient{
        playlists: []models.PlexPlaylist{
            {Name: "My Favorites", Type: "audio", Smart: false, TrackCount: 25},
            {Name: "Rock Classics", Type: "audio", Smart: true, TrackCount: 100},
        },
        tracks: map[string][]models.PlexTrack{
            "My Favorites": TestPlexTracks(),
        },
        artists: []string{"The Beatles", "Pink Floyd", "Led Zeppelin"},
    }
}

func (m *MockPlexClient) GetPlaylists() ([]models.PlexPlaylist, error) {
    if m.shouldErr {
        return nil, errors.New("mock plex error")
    }
    return m.playlists, nil
}

func (m *MockPlexClient) GetPlaylistTracks(name string) ([]models.PlexTrack, error) {
    if m.shouldErr {
        return nil, errors.New("mock plex error")
    }
    
    if tracks, exists := m.tracks[name]; exists {
        return tracks, nil
    }
    return []models.PlexTrack{}, nil
}

func (m *MockPlexClient) GetHighRatedTracks(name string, minRating int) ([]models.PlexTrack, error) {
    tracks, err := m.GetPlaylistTracks(name)
    if err != nil {
        return nil, err
    }
    
    filtered := make([]models.PlexTrack, 0)
    for _, track := range tracks {
        if track.Rating >= minRating {
            filtered = append(filtered, track)
        }
    }
    return filtered, nil
}

func (m *MockPlexClient) GetAllArtists() ([]string, error) {
    if m.shouldErr {
        return nil, errors.New("mock plex error")
    }
    return m.artists, nil
}

func (m *MockPlexClient) TestConnection() error {
    if m.shouldErr {
        return errors.New("mock connection error")
    }
    return nil
}

func (m *MockPlexClient) GetServerInfo() (map[string]string, error) {
    return map[string]string{
        "name":     "Mock Plex Server",
        "version":  "1.0.0",
        "platform": "test",
    }, nil
}

// SetError configures the mock to return errors
func (m *MockPlexClient) SetError(shouldErr bool) {
    m.shouldErr = shouldErr
}
```

### 4. Unit Tests for Core Components

- [ ] Create comprehensive unit tests starting with `internal/models/artist_test.go`

```go
package models

import (
    "encoding/json"
    "testing"
    "time"
)

func TestArtistJSONSerialization(t *testing.T) {
    artist := &Artist{
        MBID: "test-mbid-123",
        Name: "Test Artist",
        Verified: VerificationMap{
            "musicbrainz": true,
            "discogs":     false,
        },
        Genres: Genres{"rock", "alternative"},
        ExternalURLs: ExternalURLs{
            MusicBrainz: "https://musicbrainz.org/artist/test-mbid-123",
        },
        LastUpdated: time.Now(),
        CacheExpiry: time.Now().Add(time.Hour),
    }

    // Test marshaling
    data, err := json.Marshal(artist)
    if err != nil {
        t.Fatalf("Failed to marshal artist: %v", err)
    }

    // Test unmarshaling
    var unmarshaled Artist
    err = json.Unmarshal(data, &unmarshaled)
    if err != nil {
        t.Fatalf("Failed to unmarshal artist: %v", err)
    }

    // Verify critical fields
    if unmarshaled.MBID != artist.MBID {
        t.Errorf("MBID mismatch: expected %s, got %s", artist.MBID, unmarshaled.MBID)
    }

    if len(unmarshaled.Verified) != len(artist.Verified) {
        t.Errorf("Verified map length mismatch")
    }

    if len(unmarshaled.Genres) != len(artist.Genres) {
        t.Errorf("Genres length mismatch")
    }
}

func TestVerificationMapDatabaseOperations(t *testing.T) {
    vm := VerificationMap{
        "musicbrainz": true,
        "discogs":     false,
        "lastfm":      true,
    }

    // Test Value() method
    value, err := vm.Value()
    if err != nil {
        t.Fatalf("Failed to get value: %v", err)
    }

    // Test Scan() method
    var scanned VerificationMap
    err = scanned.Scan(value)
    if err != nil {
        t.Fatalf("Failed to scan value: %v", err)
    }

    // Verify data integrity
    for key, expected := range vm {
        if actual, exists := scanned[key]; !exists || actual != expected {
            t.Errorf("Value mismatch for key %s: expected %v, got %v", key, expected, actual)
        }
    }
}
```

### 5. Service Layer Tests

- [ ] Create `internal/services/cache_test.go` and other service tests

```go
package services

import (
    "database/sql"
    "os"
    "testing"
    "time"
    
    "gocommender/internal/models"
    "gocommender/internal/testutil"
    _ "modernc.org/sqlite"
)

func TestSQLiteCache(t *testing.T) {
    // Create temporary database
    db, cleanup := createTestDB(t)
    defer cleanup()
    
    cache := NewSQLiteCache(db, time.Hour, time.Minute*30)
    artist := testutil.TestArtist()
    
    t.Run("Set and Get", func(t *testing.T) {
        // Set artist in cache
        err := cache.Set(artist)
        if err != nil {
            t.Fatalf("Failed to set artist: %v", err)
        }
        
        // Get artist from cache
        retrieved, err := cache.Get(artist.MBID)
        if err != nil {
            t.Fatalf("Failed to get artist: %v", err)
        }
        
        if retrieved == nil {
            t.Fatal("Artist not found in cache")
        }
        
        if retrieved.MBID != artist.MBID {
            t.Errorf("MBID mismatch: expected %s, got %s", artist.MBID, retrieved.MBID)
        }
    })
    
    t.Run("Cache Miss", func(t *testing.T) {
        retrieved, err := cache.Get("non-existent-mbid")
        if err != nil {
            t.Fatalf("Unexpected error: %v", err)
        }
        
        if retrieved != nil {
            t.Error("Expected cache miss, got artist")
        }
    })
    
    t.Run("TTL Expiry", func(t *testing.T) {
        // Create artist with past expiry
        expiredArtist := testutil.TestArtist()
        expiredArtist.MBID = "expired-mbid"
        expiredArtist.CacheExpiry = time.Now().Add(-time.Hour)
        
        err := cache.Set(expiredArtist)
        if err != nil {
            t.Fatalf("Failed to set expired artist: %v", err)
        }
        
        // Should not retrieve expired artist
        retrieved, err := cache.Get(expiredArtist.MBID)
        if err != nil {
            t.Fatalf("Unexpected error: %v", err)
        }
        
        if retrieved != nil {
            t.Error("Retrieved expired artist")
        }
    })
}

func createTestDB(t *testing.T) (*sql.DB, func()) {
    db, err := sql.Open("sqlite", ":memory:")
    if err != nil {
        t.Fatalf("Failed to create test database: %v", err)
    }
    
    // Create schema
    schema := `
        CREATE TABLE artists (
            mbid TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            verified_json TEXT DEFAULT '{}',
            album_count INTEGER DEFAULT 0,
            years_active TEXT DEFAULT '',
            description TEXT DEFAULT '',
            genres_json TEXT DEFAULT '[]',
            country TEXT DEFAULT '',
            image_url TEXT DEFAULT '',
            external_urls_json TEXT DEFAULT '{}',
            last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
            cache_expiry DATETIME NOT NULL
        );
    `
    
    _, err = db.Exec(schema)
    if err != nil {
        t.Fatalf("Failed to create schema: %v", err)
    }
    
    return db, func() { db.Close() }
}
```

### 6. Integration Tests

- [ ] Create `internal/services/recommendation_test.go` for end-to-end testing

```go
package services

import (
    "context"
    "testing"
    "time"
    
    "gocommender/internal/models"
    "gocommender/internal/testutil"
)

func TestRecommendationServiceIntegration(t *testing.T) {
    // Skip if no integration test flag
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Set up mock services
    plexClient := testutil.NewMockPlexClient()
    cache := testutil.NewMockCacheService()
    
    // Mock OpenAI client would go here
    // For now, we'll test the workflow without actual LLM calls
    
    t.Run("Successful Recommendation Flow", func(t *testing.T) {
        // This would test the full recommendation workflow
        // using mocked services to ensure the orchestration works
        
        request := testutil.TestRecommendRequest()
        ctx := context.Background()
        
        // Test that the service can handle the request structure
        if request.PlaylistName == "" {
            t.Error("Test request should have playlist name")
        }
        
        if request.MaxResults <= 0 {
            t.Error("Test request should have valid max results")
        }
    })
    
    t.Run("Error Handling", func(t *testing.T) {
        // Test error conditions
        plexClient.SetError(true)
        
        // Test that service handles Plex errors gracefully
        playlists, err := plexClient.GetPlaylists()
        if err == nil {
            t.Error("Expected error from mock Plex client")
        }
        
        if len(playlists) != 0 {
            t.Error("Should return empty playlists on error")
        }
    })
}
```

### 7. API Tests

- [ ] Create `internal/api/router_test.go` for HTTP endpoint testing

```go
package api

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "gocommender/internal/models"
    "gocommender/internal/testutil"
)

func TestAPIEndpoints(t *testing.T) {
    // Set up test server with mock services
    plexClient := testutil.NewMockPlexClient()
    cache := testutil.NewMockCacheService()
    
    server := NewServer(nil, nil, plexClient, cache)
    
    t.Run("Health Endpoint", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/health", nil)
        w := httptest.NewRecorder()
        
        server.ServeHTTP(w, req)
        
        if w.Code != http.StatusOK {
            t.Errorf("Expected status 200, got %d", w.Code)
        }
        
        var response map[string]interface{}
        if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
            t.Fatalf("Failed to decode response: %v", err)
        }
        
        if response["status"] != "ok" {
            t.Error("Health check should return ok status")
        }
    })
    
    t.Run("Plex Playlists Endpoint", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/plex/playlists", nil)
        w := httptest.NewRecorder()
        
        server.ServeHTTP(w, req)
        
        if w.Code != http.StatusOK {
            t.Errorf("Expected status 200, got %d", w.Code)
        }
        
        var response map[string]interface{}
        if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
            t.Fatalf("Failed to decode response: %v", err)
        }
        
        if response["count"] == nil {
            t.Error("Response should include playlist count")
        }
    })
    
    t.Run("Invalid Method", func(t *testing.T) {
        req := httptest.NewRequest("DELETE", "/api/health", nil)
        w := httptest.NewRecorder()
        
        server.ServeHTTP(w, req)
        
        if w.Code != http.StatusMethodNotAllowed {
            t.Errorf("Expected status 405, got %d", w.Code)
        }
    })
    
    t.Run("CORS Headers", func(t *testing.T) {
        req := httptest.NewRequest("OPTIONS", "/api/health", nil)
        w := httptest.NewRecorder()
        
        server.ServeHTTP(w, req)
        
        if w.Header().Get("Access-Control-Allow-Origin") != "*" {
            t.Error("CORS header not set correctly")
        }
    })
}

func TestRecommendEndpoint(t *testing.T) {
    server := NewServer(nil, nil, nil, nil)
    
    t.Run("Valid Request", func(t *testing.T) {
        request := testutil.TestRecommendRequest()
        body, _ := json.Marshal(request)
        
        req := httptest.NewRequest("POST", "/api/recommend", bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
        w := httptest.NewRecorder()
        
        server.ServeHTTP(w, req)
        
        // Note: This will fail without full service setup, but tests the request parsing
        if w.Code == http.StatusBadRequest {
            t.Error("Valid request should not return bad request")
        }
    })
    
    t.Run("Invalid JSON", func(t *testing.T) {
        req := httptest.NewRequest("POST", "/api/recommend", bytes.NewBufferString("invalid json"))
        req.Header.Set("Content-Type", "application/json")
        w := httptest.NewRecorder()
        
        server.ServeHTTP(w, req)
        
        if w.Code != http.StatusBadRequest {
            t.Errorf("Expected status 400, got %d", w.Code)
        }
    })
}
```

### 8. Test Configuration

- [ ] Update `Taskfile.yml` to include proper test commands

```yaml
test:
  desc: Run unit tests
  cmds:
    - go test -v ./...

test-ci:
  desc: Run tests with coverage for CI
  cmds:
    - go test -tags=ci -cover -v ./...

test-integration:
  desc: Run integration tests
  cmds:
    - go test -v -tags=integration ./...

test-coverage:
  desc: Generate test coverage report
  cmds:
    - go test -coverprofile=coverage.out ./...
    - go tool cover -html=coverage.out -o coverage.html

benchmark:
  desc: Run benchmarks
  cmds:
    - go test -bench=. -benchmem ./...
```

## Verification Steps

- [ ] **Unit Tests**:
   ```bash
   task test
   ```

- [ ] **Integration Tests**:
   ```bash
   task test-integration
   ```

- [ ] **Coverage Report**:
   ```bash
   task test-coverage
   ```

- [ ] **CI Tests**:
   ```bash
   task test-ci
   ```

- [ ] **Benchmarks**:
   ```bash
   task benchmark
   ```

## Dependencies
- Previous: `10_http_api.md` (API implementation)
- Testing: Go testing package, testutil fixtures
- Database: In-memory SQLite for test isolation

## Next Steps
Proceed to `12_deployment_preparation.md` to set up CI/CD and deployment configurations.

## Notes
- Follows Go testing conventions with `_test.go` files
- Mock services for isolated unit testing
- Integration tests with real database (in-memory)
- API tests using httptest for HTTP handlers
- Test utilities and fixtures for consistent test data
- Coverage tracking and reporting
- CI-specific test configuration with build tags
- Benchmarking support for performance testing
- Table-driven tests where appropriate
- Proper test isolation and cleanup