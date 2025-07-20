# 06 - Caching Layer Implementation

## Overview

Implement SQLite-based caching with TTL management, background refresh, and CRUD operations using MusicBrainz ID as the primary key.

## Steps

### 1. Create Cache Service Interface

- [ ] Define `internal/services/cache.go` with cache interface and SQLite implementation

```go
package services

import (
    "database/sql"
    "time"

    "gocommender/internal/models"
)

// CacheService defines the interface for artist caching
type CacheService interface {
    Get(mbid string) (*models.Artist, error)
    Set(artist *models.Artist) error
    Delete(mbid string) error
    GetExpired() ([]models.Artist, error)
    RefreshExpired() error
    GetStats() (*CacheStats, error)
    Close() error
}

// CacheStats provides cache performance metrics
type CacheStats struct {
    TotalEntries    int       `json:"total_entries"`
    ExpiredEntries  int       `json:"expired_entries"`
    HitRate         float64   `json:"hit_rate"`
    LastRefresh     time.Time `json:"last_refresh"`
    DatabaseSize    int64     `json:"database_size_bytes"`
}

// SQLiteCache implements CacheService using SQLite
type SQLiteCache struct {
    db           *sql.DB
    ttlSuccess   time.Duration
    ttlFailure   time.Duration
    hitCount     int64
    missCount    int64
    lastRefresh  time.Time
}
```

### 2. Implement Cache Operations

- [ ] Create core CRUD functionality with TTL management

```go
// NewSQLiteCache creates a new SQLite-based cache
func NewSQLiteCache(db *sql.DB, ttlSuccess, ttlFailure time.Duration) *SQLiteCache {
    return &SQLiteCache{
        db:         db,
        ttlSuccess: ttlSuccess,
        ttlFailure: ttlFailure,
    }
}

// Get retrieves an artist from cache by MBID
func (c *SQLiteCache) Get(mbid string) (*models.Artist, error) {
    query := `
        SELECT mbid, name, verified_json, album_count, years_active,
               description, genres_json, country, image_url,
               external_urls_json, last_updated, cache_expiry
        FROM artists
        WHERE mbid = ? AND cache_expiry > datetime('now')
    `

    var artist models.Artist
    var verifiedJSON, genresJSON, externalURLsJSON string

    err := c.db.QueryRow(query, mbid).Scan(
        &artist.MBID,
        &artist.Name,
        &verifiedJSON,
        &artist.AlbumCount,
        &artist.YearsActive,
        &artist.Description,
        &genresJSON,
        &artist.Country,
        &artist.ImageURL,
        &externalURLsJSON,
        &artist.LastUpdated,
        &artist.CacheExpiry,
    )

    if err == sql.ErrNoRows {
        c.missCount++
        return nil, nil // Cache miss
    }
    if err != nil {
        c.missCount++
        return nil, fmt.Errorf("failed to query artist: %w", err)
    }

    // Deserialize JSON fields
    if err := artist.Verified.Scan(verifiedJSON); err != nil {
        return nil, fmt.Errorf("failed to scan verified: %w", err)
    }
    if err := artist.Genres.Scan(genresJSON); err != nil {
        return nil, fmt.Errorf("failed to scan genres: %w", err)
    }
    if err := artist.ExternalURLs.Scan(externalURLsJSON); err != nil {
        return nil, fmt.Errorf("failed to scan external URLs: %w", err)
    }

    c.hitCount++
    return &artist, nil
}

// Set stores an artist in cache with appropriate TTL
func (c *SQLiteCache) Set(artist *models.Artist) error {
    // Determine TTL based on verification status
    ttl := c.ttlFailure
    hasSuccessfulVerification := false

    for _, verified := range artist.Verified {
        if verified {
            hasSuccessfulVerification = true
            break
        }
    }

    if hasSuccessfulVerification {
        ttl = c.ttlSuccess
    }

    artist.CacheExpiry = time.Now().Add(ttl)
    artist.LastUpdated = time.Now()

    // Serialize JSON fields
    verifiedJSON, err := artist.Verified.Value()
    if err != nil {
        return fmt.Errorf("failed to serialize verified: %w", err)
    }

    genresJSON, err := artist.Genres.Value()
    if err != nil {
        return fmt.Errorf("failed to serialize genres: %w", err)
    }

    externalURLsJSON, err := artist.ExternalURLs.Value()
    if err != nil {
        return fmt.Errorf("failed to serialize external URLs: %w", err)
    }

    query := `
        INSERT OR REPLACE INTO artists (
            mbid, name, verified_json, album_count, years_active,
            description, genres_json, country, image_url,
            external_urls_json, last_updated, cache_expiry
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

    _, err = c.db.Exec(query,
        artist.MBID,
        artist.Name,
        verifiedJSON,
        artist.AlbumCount,
        artist.YearsActive,
        artist.Description,
        genresJSON,
        artist.Country,
        artist.ImageURL,
        externalURLsJSON,
        artist.LastUpdated,
        artist.CacheExpiry,
    )

    if err != nil {
        return fmt.Errorf("failed to insert artist: %w", err)
    }

    return nil
}

// Delete removes an artist from cache
func (c *SQLiteCache) Delete(mbid string) error {
    query := "DELETE FROM artists WHERE mbid = ?"
    _, err := c.db.Exec(query, mbid)
    if err != nil {
        return fmt.Errorf("failed to delete artist: %w", err)
    }
    return nil
}
```

### 3. Implement Cache Management

- [ ] Add expiry handling and maintenance functions

```go
// GetExpired returns all expired cache entries
func (c *SQLiteCache) GetExpired() ([]models.Artist, error) {
    query := `
        SELECT mbid, name, verified_json, album_count, years_active,
               description, genres_json, country, image_url,
               external_urls_json, last_updated, cache_expiry
        FROM artists
        WHERE cache_expiry <= datetime('now')
        ORDER BY cache_expiry ASC
        LIMIT 100
    `

    rows, err := c.db.Query(query)
    if err != nil {
        return nil, fmt.Errorf("failed to query expired artists: %w", err)
    }
    defer rows.Close()

    var artists []models.Artist
    for rows.Next() {
        var artist models.Artist
        var verifiedJSON, genresJSON, externalURLsJSON string

        err := rows.Scan(
            &artist.MBID,
            &artist.Name,
            &verifiedJSON,
            &artist.AlbumCount,
            &artist.YearsActive,
            &artist.Description,
            &genresJSON,
            &artist.Country,
            &artist.ImageURL,
            &externalURLsJSON,
            &artist.LastUpdated,
            &artist.CacheExpiry,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan artist: %w", err)
        }

        // Deserialize JSON fields
        artist.Verified.Scan(verifiedJSON)
        artist.Genres.Scan(genresJSON)
        artist.ExternalURLs.Scan(externalURLsJSON)

        artists = append(artists, artist)
    }

    return artists, nil
}

// RefreshExpired removes expired entries and optionally re-fetches them
func (c *SQLiteCache) RefreshExpired() error {
    // Delete expired entries older than 2x TTL to avoid perpetual failures
    cutoff := time.Now().Add(-2 * c.ttlFailure)

    query := "DELETE FROM artists WHERE cache_expiry < ?"
    result, err := c.db.Exec(query, cutoff)
    if err != nil {
        return fmt.Errorf("failed to delete expired artists: %w", err)
    }

    deleted, _ := result.RowsAffected()
    c.lastRefresh = time.Now()

    log.Printf("Cache refresh: deleted %d expired entries", deleted)
    return nil
}

// GetStats returns cache performance statistics
func (c *SQLiteCache) GetStats() (*CacheStats, error) {
    var total, expired int

    // Count total entries
    err := c.db.QueryRow("SELECT COUNT(*) FROM artists").Scan(&total)
    if err != nil {
        return nil, fmt.Errorf("failed to count total artists: %w", err)
    }

    // Count expired entries
    err = c.db.QueryRow(
        "SELECT COUNT(*) FROM artists WHERE cache_expiry <= datetime('now')",
    ).Scan(&expired)
    if err != nil {
        return nil, fmt.Errorf("failed to count expired artists: %w", err)
    }

    // Calculate hit rate
    totalRequests := c.hitCount + c.missCount
    hitRate := 0.0
    if totalRequests > 0 {
        hitRate = float64(c.hitCount) / float64(totalRequests)
    }

    // Get database file size
    var dbSize int64
    err = c.db.QueryRow("SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()").Scan(&dbSize)
    if err != nil {
        dbSize = 0 // Don't fail on this
    }

    return &CacheStats{
        TotalEntries:   total,
        ExpiredEntries: expired,
        HitRate:        hitRate,
        LastRefresh:    c.lastRefresh,
        DatabaseSize:   dbSize,
    }, nil
}

// Close gracefully shuts down the cache
func (c *SQLiteCache) Close() error {
    return c.db.Close()
}
```

### 4. Background Refresh Worker

- [ ] Create `internal/services/cache_worker.go` for background cache refresh

```go
package services

import (
    "context"
    "log"
    "time"
)

// CacheWorker handles background cache maintenance
type CacheWorker struct {
    cache           CacheService
    enricher        *ArtistEnricher
    refreshInterval time.Duration
    ctx             context.Context
    cancel          context.CancelFunc
}

// NewCacheWorker creates a new background cache worker
func NewCacheWorker(cache CacheService, enricher *ArtistEnricher,
                   refreshInterval time.Duration) *CacheWorker {
    ctx, cancel := context.WithCancel(context.Background())

    return &CacheWorker{
        cache:           cache,
        enricher:        enricher,
        refreshInterval: refreshInterval,
        ctx:             ctx,
        cancel:          cancel,
    }
}

// Start begins the background refresh process
func (w *CacheWorker) Start() {
    go w.refreshLoop()
}

// Stop gracefully stops the background worker
func (w *CacheWorker) Stop() {
    w.cancel()
}

func (w *CacheWorker) refreshLoop() {
    ticker := time.NewTicker(w.refreshInterval)
    defer ticker.Stop()

    for {
        select {
        case <-w.ctx.Done():
            return
        case <-ticker.C:
            w.performRefresh()
        }
    }
}

func (w *CacheWorker) performRefresh() {
    log.Println("Starting cache refresh cycle")

    // Clean up very old expired entries
    if err := w.cache.RefreshExpired(); err != nil {
        log.Printf("Cache refresh failed: %v", err)
        return
    }

    // Get recently expired entries that might be worth refreshing
    expired, err := w.cache.GetExpired()
    if err != nil {
        log.Printf("Failed to get expired entries: %v", err)
        return
    }

    refreshed := 0
    for _, artist := range expired {
        // Only refresh if it was successfully verified before
        hasVerification := false
        for _, verified := range artist.Verified {
            if verified {
                hasVerification = true
                break
            }
        }

        if !hasVerification {
            continue // Skip entries that were never successfully verified
        }

        // Re-enrich the artist data
        fresh, err := w.enricher.EnrichArtist(artist.Name)
        if err != nil {
            log.Printf("Failed to refresh artist %s: %v", artist.Name, err)
            continue
        }

        // Store refreshed data
        if err := w.cache.Set(fresh); err != nil {
            log.Printf("Failed to cache refreshed artist %s: %v", artist.Name, err)
            continue
        }

        refreshed++

        // Rate limit refreshes to avoid API abuse
        time.Sleep(2 * time.Second)
    }

    if refreshed > 0 {
        log.Printf("Cache refresh complete: refreshed %d entries", refreshed)
    }
}
```

### 5. Cache-Aware Artist Service

- [ ] Create `internal/services/artist_service.go` with cache integration

```go
package services

import (
    "fmt"

    "gocommender/internal/models"
)

// ArtistService provides cached artist lookup with fallback to API enrichment
type ArtistService struct {
    cache    CacheService
    enricher *ArtistEnricher
}

// NewArtistService creates a new artist service
func NewArtistService(cache CacheService, enricher *ArtistEnricher) *ArtistService {
    return &ArtistService{
        cache:    cache,
        enricher: enricher,
    }
}

// GetArtist retrieves artist information, using cache when available
func (s *ArtistService) GetArtist(name string) (*models.Artist, error) {
    // First try to find by name (requires secondary lookup)
    artist, err := s.findArtistByName(name)
    if err == nil && artist != nil {
        return artist, nil
    }

    // Cache miss - enrich from APIs
    enriched, err := s.enricher.EnrichArtist(name)
    if err != nil {
        return nil, fmt.Errorf("failed to enrich artist: %w", err)
    }

    // Store in cache for future use
    if err := s.cache.Set(enriched); err != nil {
        // Log error but don't fail the request
        log.Printf("Failed to cache artist %s: %v", name, err)
    }

    return enriched, nil
}

// GetArtistByMBID retrieves artist by MusicBrainz ID (primary cache key)
func (s *ArtistService) GetArtistByMBID(mbid string) (*models.Artist, error) {
    // Try cache first
    artist, err := s.cache.Get(mbid)
    if err != nil {
        return nil, fmt.Errorf("cache error: %w", err)
    }

    if artist != nil {
        return artist, nil // Cache hit
    }

    // Cache miss - need to get from MusicBrainz first to get name
    mbClient := NewMusicBrainzClient()
    defer mbClient.Close()

    mbArtist, err := mbClient.GetArtistByMBID(mbid)
    if err != nil {
        return nil, fmt.Errorf("failed to get artist from MusicBrainz: %w", err)
    }

    // Now enrich with all sources
    enriched, err := s.enricher.EnrichArtist(mbArtist.Name)
    if err != nil {
        return nil, fmt.Errorf("failed to enrich artist: %w", err)
    }

    // Ensure MBID is set correctly
    enriched.MBID = mbid

    // Store in cache
    if err := s.cache.Set(enriched); err != nil {
        log.Printf("Failed to cache artist %s: %v", enriched.Name, err)
    }

    return enriched, nil
}

func (s *ArtistService) findArtistByName(name string) (*models.Artist, error) {
    // This would require a secondary index on name
    // For now, we'll use the enrichment process which starts with search
    return nil, fmt.Errorf("lookup by name not implemented")
}
```

## Verification Steps

- [ ] **Cache CRUD Operations**:

   ```bash
   go test ./internal/services -run TestSQLiteCache
   ```

- [ ] **TTL Management**:

   ```bash
   go test ./internal/services -run TestCacheTTL
   ```

- [ ] **Background Refresh**:

   ```bash
   go run ./cmd/test-cache-worker -duration 30s
   ```

- [ ] **Performance Testing**:

   ```bash
   go run ./cmd/cache-benchmark -operations 1000
   ```

- [ ] **Cache Statistics**:

   ```bash
   go run ./cmd/cache-stats
   ```

## Dependencies

- Previous: `05_external_apis.md` (API enrichment services)
- Database: SQLite with modernc.org/sqlite
- Background: context, time packages

## Next Steps

Proceed to `07_plex_integration.md` to implement Plex API client for playlist and artist data.

## Notes

- MusicBrainz ID serves as primary cache key for uniqueness
- TTL varies based on verification success (30 days vs 7 days)
- Background worker refreshes only previously successful verifications
- Cache statistics help monitor performance and hit rates
- Graceful degradation when cache is unavailable
- JSON serialization handles complex data types in SQLite
- Rate limiting in background refresh prevents API abuse
- Cache-aware service provides transparent fallback to APIs
