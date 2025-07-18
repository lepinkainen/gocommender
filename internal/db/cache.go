package db

import (
	"database/sql"
	"fmt"
	"time"

	"gocommender/internal/models"
)

// CacheManager provides high-level caching operations with TTL management
type CacheManager struct {
	artistDB *ArtistDB
	db       *sql.DB
}

// NewCacheManager creates a new cache manager
func NewCacheManager(db *sql.DB) *CacheManager {
	return &CacheManager{
		artistDB: NewArtistDB(db),
		db:       db,
	}
}

// CacheConfig defines TTL policies for different scenarios
type CacheConfig struct {
	VerifiedTTL   time.Duration // TTL for successfully verified artists
	UnverifiedTTL time.Duration // TTL for failed verification attempts
	RefreshTTL    time.Duration // TTL for background refresh
}

// DefaultCacheConfig returns the default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		VerifiedTTL:   30 * 24 * time.Hour, // 30 days for verified artists
		UnverifiedTTL: 7 * 24 * time.Hour,  // 7 days for failed lookups
		RefreshTTL:    24 * time.Hour,      // 24 hours for background refresh
	}
}

// GetOrFetchArtist retrieves an artist from cache or indicates if fetch is needed
func (cm *CacheManager) GetOrFetchArtist(mbid string) (*models.Artist, bool, error) {
	artist, err := cm.artistDB.GetArtist(mbid)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get artist from cache: %w", err)
	}

	// Artist not in cache
	if artist == nil {
		return nil, true, nil // needsFetch = true
	}

	// Check if cache has expired
	expired, err := cm.artistDB.IsExpired(mbid)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check cache expiry: %w", err)
	}

	if expired {
		return artist, true, nil // Return cached data but indicate refresh needed
	}

	return artist, false, nil // Fresh cache hit
}

// CacheArtist stores an artist with appropriate TTL based on verification status
func (cm *CacheManager) CacheArtist(artist *models.Artist, config CacheConfig) error {
	if artist.MBID == "" {
		return fmt.Errorf("artist MBID cannot be empty")
	}

	// Set cache expiry based on verification status
	artist.LastUpdated = time.Now()
	artist.CacheExpiry = cm.calculateExpiry(artist, config)

	return cm.artistDB.SaveArtist(artist)
}

// calculateExpiry determines the cache expiry time based on artist verification status
func (cm *CacheManager) calculateExpiry(artist *models.Artist, config CacheConfig) time.Time {
	now := time.Now()

	// If artist has any successful verifications, use longer TTL
	hasVerification := false
	if artist.Verified != nil {
		for _, verified := range artist.Verified {
			if verified {
				hasVerification = true
				break
			}
		}
	}

	if hasVerification {
		return now.Add(config.VerifiedTTL)
	}

	// No successful verifications, use shorter TTL
	return now.Add(config.UnverifiedTTL)
}

// RefreshExpiredArtists gets a batch of expired artists for background refresh
func (cm *CacheManager) RefreshExpiredArtists(limit int) ([]models.Artist, error) {
	return cm.artistDB.GetExpiredArtists(limit)
}

// CleanupExpiredEntries removes entries that have been expired for too long
func (cm *CacheManager) CleanupExpiredEntries(maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge)
	query := "DELETE FROM artists WHERE cache_expiry < ?"

	result, err := cm.db.Exec(query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired entries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// GetCacheStats returns comprehensive cache statistics
func (cm *CacheManager) GetCacheStats() (CacheStats, error) {
	return cm.artistDB.GetCacheStats()
}

// UpdateCacheExpiry updates the cache expiry for an artist without changing other data
func (cm *CacheManager) UpdateCacheExpiry(mbid string, newExpiry time.Time) error {
	query := "UPDATE artists SET cache_expiry = ?, last_updated = ? WHERE mbid = ?"
	_, err := cm.db.Exec(query, newExpiry, time.Now(), mbid)
	return err
}

// BulkCacheArtists efficiently caches multiple artists in a transaction
func (cm *CacheManager) BulkCacheArtists(artists []models.Artist, config CacheConfig) error {
	if len(artists) == 0 {
		return nil
	}

	tx, err := cm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
INSERT INTO artists (
    mbid, name, verified_json, album_count, years_active, 
    description, genres_json, country, image_url, 
    external_urls_json, last_updated, cache_expiry
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(mbid) DO UPDATE SET
    name = excluded.name,
    verified_json = excluded.verified_json,
    album_count = excluded.album_count,
    years_active = excluded.years_active,
    description = excluded.description,
    genres_json = excluded.genres_json,
    country = excluded.country,
    image_url = excluded.image_url,
    external_urls_json = excluded.external_urls_json,
    last_updated = excluded.last_updated,
    cache_expiry = excluded.cache_expiry
`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, artist := range artists {
		if artist.MBID == "" {
			continue // Skip artists without MBID
		}

		artist.LastUpdated = time.Now()
		artist.CacheExpiry = cm.calculateExpiry(&artist, config)

		_, err := stmt.Exec(
			artist.MBID,
			artist.Name,
			artist.Verified,
			artist.AlbumCount,
			artist.YearsActive,
			artist.Description,
			artist.Genres,
			artist.Country,
			artist.ImageURL,
			artist.ExternalURLs,
			artist.LastUpdated,
			artist.CacheExpiry,
		)
		if err != nil {
			return fmt.Errorf("failed to execute statement for artist %s: %w", artist.MBID, err)
		}
	}

	return tx.Commit()
}
