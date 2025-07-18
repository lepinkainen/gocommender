package config

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// InitDatabase initializes the SQLite database with schema
func InitDatabase(dbPath string) (*sql.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create schema
	if err := createSchema(db); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return db, nil
}

func createSchema(db *sql.DB) error {
	schema := `
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
`
	_, err := db.Exec(schema)
	return err
}
