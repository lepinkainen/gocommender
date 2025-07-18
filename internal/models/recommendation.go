package models

import "time"

// RecommendRequest represents the API request for recommendations
type RecommendRequest struct {
	PlaylistName string  `json:"playlist_name" validate:"required"`
	Genre        *string `json:"genre,omitempty"`
	MaxResults   int     `json:"max_results,omitempty"` // Default: 5
}

// RecommendResponse represents the API response
type RecommendResponse struct {
	Status      string            `json:"status"`
	RequestID   string            `json:"request_id"`
	Suggestions []Artist          `json:"suggestions"`
	Metadata    RecommendMetadata `json:"metadata"`
	Error       string            `json:"error,omitempty"`
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
