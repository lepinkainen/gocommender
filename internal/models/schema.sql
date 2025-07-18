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