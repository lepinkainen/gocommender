package db

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"gocommender/internal/models"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Create in-memory database for testing
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
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

CREATE INDEX idx_cache_expiry ON artists(cache_expiry);
CREATE INDEX idx_last_updated ON artists(last_updated);
CREATE INDEX idx_name ON artists(name);
CREATE INDEX idx_verified ON artists(verified_json);
`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	return db
}

func TestCacheManager_GetOrFetchArtist(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cm := NewCacheManager(db)
	config := DefaultCacheConfig()

	// Test 1: Artist not in cache
	artist, needsFetch, err := cm.GetOrFetchArtist("test-mbid-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if artist != nil {
		t.Errorf("Expected nil artist for non-existent entry")
	}
	if !needsFetch {
		t.Errorf("Expected needsFetch=true for non-existent entry")
	}

	// Test 2: Cache artist and retrieve
	testArtist := &models.Artist{
		MBID: "test-mbid-1",
		Name: "Test Artist",
		Verified: models.VerificationMap{
			"musicbrainz": true,
			"discogs":     true,
		},
		Genres:      models.Genres{"rock", "alternative"},
		Description: "Test description",
		Country:     "US",
	}

	err = cm.CacheArtist(testArtist, config)
	if err != nil {
		t.Fatalf("Failed to cache artist: %v", err)
	}

	// Retrieve cached artist
	cachedArtist, needsFetch, err := cm.GetOrFetchArtist("test-mbid-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if cachedArtist == nil {
		t.Fatalf("Expected cached artist")
	}
	if needsFetch {
		t.Errorf("Expected needsFetch=false for fresh cache")
	}
	if cachedArtist.Name != "Test Artist" {
		t.Errorf("Expected artist name 'Test Artist', got '%s'", cachedArtist.Name)
	}
}

func TestCacheManager_TTLLogic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cm := NewCacheManager(db)
	config := CacheConfig{
		VerifiedTTL:   1 * time.Hour,
		UnverifiedTTL: 30 * time.Minute,
	}

	// Test verified artist gets longer TTL
	verifiedArtist := &models.Artist{
		MBID: "verified-mbid",
		Name: "Verified Artist",
		Verified: models.VerificationMap{
			"musicbrainz": true,
		},
	}

	err := cm.CacheArtist(verifiedArtist, config)
	if err != nil {
		t.Fatalf("Failed to cache verified artist: %v", err)
	}

	// Test unverified artist gets shorter TTL
	unverifiedArtist := &models.Artist{
		MBID: "unverified-mbid",
		Name: "Unverified Artist",
		Verified: models.VerificationMap{
			"musicbrainz": false,
		},
	}

	err = cm.CacheArtist(unverifiedArtist, config)
	if err != nil {
		t.Fatalf("Failed to cache unverified artist: %v", err)
	}

	// Retrieve and check TTL differences
	verifiedCached, _, err := cm.GetOrFetchArtist("verified-mbid")
	if err != nil {
		t.Fatalf("Failed to get verified artist: %v", err)
	}

	unverifiedCached, _, err := cm.GetOrFetchArtist("unverified-mbid")
	if err != nil {
		t.Fatalf("Failed to get unverified artist: %v", err)
	}

	// Verified artist should have later expiry than unverified
	if !verifiedCached.CacheExpiry.After(unverifiedCached.CacheExpiry) {
		t.Errorf("Expected verified artist to have later expiry time")
	}
}

func TestCacheManager_BulkOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cm := NewCacheManager(db)
	config := DefaultCacheConfig()

	// Create test artists
	artists := []models.Artist{
		{
			MBID:     "bulk-1",
			Name:     "Bulk Artist 1",
			Verified: models.VerificationMap{"musicbrainz": true},
		},
		{
			MBID:     "bulk-2",
			Name:     "Bulk Artist 2",
			Verified: models.VerificationMap{"musicbrainz": true},
		},
		{
			MBID:     "bulk-3",
			Name:     "Bulk Artist 3",
			Verified: models.VerificationMap{"musicbrainz": false},
		},
	}

	// Bulk cache
	err := cm.BulkCacheArtists(artists, config)
	if err != nil {
		t.Fatalf("Failed to bulk cache artists: %v", err)
	}

	// Verify all were cached
	for _, artist := range artists {
		cached, needsFetch, err := cm.GetOrFetchArtist(artist.MBID)
		if err != nil {
			t.Fatalf("Failed to get artist %s: %v", artist.MBID, err)
		}
		if cached == nil {
			t.Errorf("Artist %s was not cached", artist.MBID)
		}
		if needsFetch {
			t.Errorf("Artist %s cache should be fresh", artist.MBID)
		}
		if cached.Name != artist.Name {
			t.Errorf("Expected name %s, got %s", artist.Name, cached.Name)
		}
	}
}

func TestCacheManager_CacheStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cm := NewCacheManager(db)
	config := DefaultCacheConfig()

	// Initial stats should be empty
	stats, err := cm.GetCacheStats()
	if err != nil {
		t.Fatalf("Failed to get cache stats: %v", err)
	}
	if stats.Total != 0 {
		t.Errorf("Expected total count 0, got %d", stats.Total)
	}

	// Add some artists
	artists := []models.Artist{
		{MBID: "stats-1", Name: "Artist 1", Verified: models.VerificationMap{"musicbrainz": true}},
		{MBID: "stats-2", Name: "Artist 2", Verified: models.VerificationMap{"musicbrainz": true}},
	}

	err = cm.BulkCacheArtists(artists, config)
	if err != nil {
		t.Fatalf("Failed to cache artists: %v", err)
	}

	// Check updated stats
	stats, err = cm.GetCacheStats()
	if err != nil {
		t.Fatalf("Failed to get updated cache stats: %v", err)
	}
	if stats.Total != 2 {
		t.Errorf("Expected total count 2, got %d", stats.Total)
	}
	if stats.Valid != 2 {
		t.Errorf("Expected valid count 2, got %d", stats.Valid)
	}
	if stats.Expired != 0 {
		t.Errorf("Expected expired count 0, got %d", stats.Expired)
	}
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}
