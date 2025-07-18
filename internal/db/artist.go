package db

import (
	"database/sql"
	"fmt"
	"time"

	"gocommender/internal/models"
)

// ArtistDB handles database operations for Artist entities
type ArtistDB struct {
	db *sql.DB
}

// NewArtistDB creates a new ArtistDB instance
func NewArtistDB(db *sql.DB) *ArtistDB {
	return &ArtistDB{db: db}
}

// SaveArtist saves or updates an artist in the database
func (adb *ArtistDB) SaveArtist(artist *models.Artist) error {
	query := `
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
`

	_, err := adb.db.Exec(query,
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

	return err
}

// GetArtist retrieves an artist by MBID
func (adb *ArtistDB) GetArtist(mbid string) (*models.Artist, error) {
	query := `
SELECT mbid, name, verified_json, album_count, years_active,
       description, genres_json, country, image_url, 
       external_urls_json, last_updated, cache_expiry
FROM artists 
WHERE mbid = ?
`

	var artist models.Artist
	err := adb.db.QueryRow(query, mbid).Scan(
		&artist.MBID,
		&artist.Name,
		&artist.Verified,
		&artist.AlbumCount,
		&artist.YearsActive,
		&artist.Description,
		&artist.Genres,
		&artist.Country,
		&artist.ImageURL,
		&artist.ExternalURLs,
		&artist.LastUpdated,
		&artist.CacheExpiry,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Artist not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get artist: %w", err)
	}

	return &artist, nil
}

// IsExpired checks if an artist's cache has expired
func (adb *ArtistDB) IsExpired(mbid string) (bool, error) {
	query := "SELECT cache_expiry FROM artists WHERE mbid = ?"

	var cacheExpiry time.Time
	err := adb.db.QueryRow(query, mbid).Scan(&cacheExpiry)
	if err == sql.ErrNoRows {
		return true, nil // Not found means it needs to be fetched
	}
	if err != nil {
		return true, fmt.Errorf("failed to check expiry: %w", err)
	}

	return time.Now().After(cacheExpiry), nil
}

// GetExpiredArtists returns artists whose cache has expired
func (adb *ArtistDB) GetExpiredArtists(limit int) ([]models.Artist, error) {
	query := `
SELECT mbid, name, verified_json, album_count, years_active,
       description, genres_json, country, image_url, 
       external_urls_json, last_updated, cache_expiry
FROM artists 
WHERE cache_expiry < ? 
ORDER BY cache_expiry ASC
LIMIT ?
`

	rows, err := adb.db.Query(query, time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired artists: %w", err)
	}
	defer rows.Close()

	var artists []models.Artist
	for rows.Next() {
		var artist models.Artist
		err := rows.Scan(
			&artist.MBID,
			&artist.Name,
			&artist.Verified,
			&artist.AlbumCount,
			&artist.YearsActive,
			&artist.Description,
			&artist.Genres,
			&artist.Country,
			&artist.ImageURL,
			&artist.ExternalURLs,
			&artist.LastUpdated,
			&artist.CacheExpiry,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artist: %w", err)
		}
		artists = append(artists, artist)
	}

	return artists, rows.Err()
}

// DeleteArtist removes an artist from the database
func (adb *ArtistDB) DeleteArtist(mbid string) error {
	query := "DELETE FROM artists WHERE mbid = ?"
	_, err := adb.db.Exec(query, mbid)
	return err
}

// GetArtistCount returns the total number of artists in the database
func (adb *ArtistDB) GetArtistCount() (int, error) {
	query := "SELECT COUNT(*) FROM artists"
	var count int
	err := adb.db.QueryRow(query).Scan(&count)
	return count, err
}

// GetCacheStats returns cache statistics
func (adb *ArtistDB) GetCacheStats() (CacheStats, error) {
	var stats CacheStats

	// Total count
	if err := adb.db.QueryRow("SELECT COUNT(*) FROM artists").Scan(&stats.Total); err != nil {
		return stats, fmt.Errorf("failed to get total count: %w", err)
	}

	// Expired count
	if err := adb.db.QueryRow("SELECT COUNT(*) FROM artists WHERE cache_expiry < ?", time.Now()).Scan(&stats.Expired); err != nil {
		return stats, fmt.Errorf("failed to get expired count: %w", err)
	}

	stats.Valid = stats.Total - stats.Expired
	return stats, nil
}

// CacheStats represents cache statistics
type CacheStats struct {
	Total   int `json:"total"`
	Valid   int `json:"valid"`
	Expired int `json:"expired"`
}
