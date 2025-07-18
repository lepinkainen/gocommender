package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Artist represents a musical artist with enriched metadata
type Artist struct {
	MBID         string          `json:"mbid" db:"mbid"` // MusicBrainz ID (primary key)
	Name         string          `json:"name" db:"name"`
	Verified     VerificationMap `json:"verified" db:"verified_json"` // Service verification status
	AlbumCount   int             `json:"album_count" db:"album_count"`
	YearsActive  string          `json:"years_active" db:"years_active"` // e.g., "1970-present", "1980-1995"
	Description  string          `json:"description" db:"description"`   // Biography/description
	Genres       Genres          `json:"genres" db:"genres_json"`
	Country      string          `json:"country" db:"country"`
	ImageURL     string          `json:"image_url" db:"image_url"`
	ExternalURLs ExternalURLs    `json:"external_urls" db:"external_urls_json"`
	LastUpdated  time.Time       `json:"last_updated" db:"last_updated"`
	CacheExpiry  time.Time       `json:"-" db:"cache_expiry"`
}

// VerificationMap tracks which services have verified this artist
type VerificationMap map[string]bool

// Genres represents a slice of genre strings for database compatibility
type Genres []string

// ExternalURLs contains links to external services
type ExternalURLs struct {
	Discogs     string `json:"discogs,omitempty"`
	MusicBrainz string `json:"musicbrainz,omitempty"` // Full URL to MB page
	LastFM      string `json:"lastfm,omitempty"`
	Spotify     string `json:"spotify,omitempty"`
}

// Value implements driver.Valuer for VerificationMap
func (v VerificationMap) Value() (driver.Value, error) {
	if v == nil {
		return "{}", nil
	}
	return json.Marshal(v)
}

// Scan implements sql.Scanner for VerificationMap
func (v *VerificationMap) Scan(value interface{}) error {
	if value == nil {
		*v = make(VerificationMap)
		return nil
	}

	var bytes []byte
	switch val := value.(type) {
	case []byte:
		bytes = val
	case string:
		bytes = []byte(val)
	default:
		*v = make(VerificationMap)
		return nil
	}

	return json.Unmarshal(bytes, v)
}

// Value implements driver.Valuer for Genres
func (g Genres) Value() (driver.Value, error) {
	if g == nil {
		return "[]", nil
	}
	return json.Marshal(g)
}

// Scan implements sql.Scanner for Genres
func (g *Genres) Scan(value interface{}) error {
	if value == nil {
		*g = make(Genres, 0)
		return nil
	}

	var bytes []byte
	switch val := value.(type) {
	case []byte:
		bytes = val
	case string:
		bytes = []byte(val)
	default:
		*g = make(Genres, 0)
		return nil
	}

	return json.Unmarshal(bytes, g)
}

// Value implements driver.Valuer for ExternalURLs
func (e ExternalURLs) Value() (driver.Value, error) {
	return json.Marshal(e)
}

// Scan implements sql.Scanner for ExternalURLs
func (e *ExternalURLs) Scan(value interface{}) error {
	if value == nil {
		*e = ExternalURLs{}
		return nil
	}

	var bytes []byte
	switch val := value.(type) {
	case []byte:
		bytes = val
	case string:
		bytes = []byte(val)
	default:
		*e = ExternalURLs{}
		return nil
	}

	return json.Unmarshal(bytes, e)
}
