package models

import "time"

// PlexTrack represents a track from Plex playlist
type PlexTrack struct {
	Title      string    `json:"title"`
	Artist     string    `json:"artist"`
	Album      string    `json:"album"`
	Year       int       `json:"year"`
	Rating     int       `json:"rating"` // 1-10 scale
	PlayCount  int       `json:"play_count"`
	LastPlayed time.Time `json:"last_played"`
}

// PlexPlaylist represents a Plex playlist
type PlexPlaylist struct {
	Name       string      `json:"name"`
	Type       string      `json:"type"` // "audio"
	Smart      bool        `json:"smart"`
	TrackCount int         `json:"track_count"`
	Duration   int         `json:"duration"` // milliseconds
	Tracks     []PlexTrack `json:"tracks"`
}
