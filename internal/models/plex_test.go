package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPlexTrack(t *testing.T) {
	t.Run("Valid track", func(t *testing.T) {
		track := PlexTrack{
			Title:      "Test Song",
			Artist:     "Test Artist",
			Album:      "Test Album",
			Year:       2020,
			Rating:     8,
			PlayCount:  15,
			LastPlayed: time.Now().Add(-24 * time.Hour),
		}

		if track.Title == "" {
			t.Error("Title should not be empty")
		}
		if track.Artist == "" {
			t.Error("Artist should not be empty")
		}
		if track.Album == "" {
			t.Error("Album should not be empty")
		}
		if track.Year <= 0 {
			t.Error("Year should be positive")
		}
		if track.Rating < 0 || track.Rating > 10 {
			t.Error("Rating should be between 0 and 10")
		}
		if track.PlayCount < 0 {
			t.Error("PlayCount should not be negative")
		}
		if track.LastPlayed.IsZero() {
			t.Error("LastPlayed should be set")
		}
	})

	t.Run("Minimal track", func(t *testing.T) {
		track := PlexTrack{
			Title:  "Minimal Song",
			Artist: "Minimal Artist",
		}

		if track.Title == "" {
			t.Error("Title should not be empty")
		}
		if track.Artist == "" {
			t.Error("Artist should not be empty")
		}

		// These fields can be zero/empty for minimal track
		if track.Album != "" {
			// Album can be empty
		}
		if track.Year != 0 {
			// Year can be zero
		}
		if track.Rating != 0 {
			// Rating can be zero
		}
		if track.PlayCount != 0 {
			// PlayCount can be zero
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		originalTime := time.Now().Add(-2 * time.Hour)
		track := PlexTrack{
			Title:      "JSON Test Song",
			Artist:     "JSON Test Artist",
			Album:      "JSON Test Album",
			Year:       2021,
			Rating:     9,
			PlayCount:  42,
			LastPlayed: originalTime,
		}

		data, err := json.Marshal(track)
		if err != nil {
			t.Fatalf("Failed to marshal track: %v", err)
		}

		var unmarshaled PlexTrack
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal track: %v", err)
		}

		if unmarshaled.Title != track.Title {
			t.Errorf("Title mismatch: expected %s, got %s", track.Title, unmarshaled.Title)
		}
		if unmarshaled.Artist != track.Artist {
			t.Errorf("Artist mismatch: expected %s, got %s", track.Artist, unmarshaled.Artist)
		}
		if unmarshaled.Album != track.Album {
			t.Errorf("Album mismatch: expected %s, got %s", track.Album, unmarshaled.Album)
		}
		if unmarshaled.Year != track.Year {
			t.Errorf("Year mismatch: expected %d, got %d", track.Year, unmarshaled.Year)
		}
		if unmarshaled.Rating != track.Rating {
			t.Errorf("Rating mismatch: expected %d, got %d", track.Rating, unmarshaled.Rating)
		}
		if unmarshaled.PlayCount != track.PlayCount {
			t.Errorf("PlayCount mismatch: expected %d, got %d", track.PlayCount, unmarshaled.PlayCount)
		}

		// Time comparison with some tolerance for JSON precision
		timeDiff := unmarshaled.LastPlayed.Sub(track.LastPlayed)
		if timeDiff > time.Second || timeDiff < -time.Second {
			t.Errorf("LastPlayed time mismatch: expected %v, got %v", track.LastPlayed, unmarshaled.LastPlayed)
		}
	})

	t.Run("High rated track", func(t *testing.T) {
		track := PlexTrack{
			Title:  "Excellent Song",
			Artist: "Amazing Artist",
			Rating: 10,
		}

		if track.Rating != 10 {
			t.Errorf("Expected rating 10, got %d", track.Rating)
		}
	})

	t.Run("Unrated track", func(t *testing.T) {
		track := PlexTrack{
			Title:  "Unrated Song",
			Artist: "Unknown Artist",
			Rating: 0,
		}

		if track.Rating != 0 {
			t.Errorf("Expected rating 0, got %d", track.Rating)
		}
	})
}

func TestPlexPlaylist(t *testing.T) {
	t.Run("Valid playlist", func(t *testing.T) {
		playlist := PlexPlaylist{
			Name:       "My Favorites",
			Type:       "audio",
			Smart:      false,
			TrackCount: 25,
		}

		if playlist.Name == "" {
			t.Error("Name should not be empty")
		}
		if playlist.Type == "" {
			t.Error("Type should not be empty")
		}
		if playlist.TrackCount < 0 {
			t.Error("TrackCount should not be negative")
		}
	})

	t.Run("Smart playlist", func(t *testing.T) {
		playlist := PlexPlaylist{
			Name:       "Recently Added",
			Type:       "audio",
			Smart:      true,
			TrackCount: 100,
		}

		if !playlist.Smart {
			t.Error("Smart playlist should have Smart=true")
		}
		if playlist.Name == "" {
			t.Error("Smart playlist should have a name")
		}
	})

	t.Run("Empty playlist", func(t *testing.T) {
		playlist := PlexPlaylist{
			Name:       "Empty Playlist",
			Type:       "audio",
			Smart:      false,
			TrackCount: 0,
		}

		if playlist.TrackCount != 0 {
			t.Errorf("Expected TrackCount 0, got %d", playlist.TrackCount)
		}
		if playlist.Name == "" {
			t.Error("Empty playlist should still have a name")
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		playlist := PlexPlaylist{
			Name:       "JSON Test Playlist",
			Type:       "audio",
			Smart:      true,
			TrackCount: 50,
		}

		data, err := json.Marshal(playlist)
		if err != nil {
			t.Fatalf("Failed to marshal playlist: %v", err)
		}

		var unmarshaled PlexPlaylist
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal playlist: %v", err)
		}

		if unmarshaled.Name != playlist.Name {
			t.Errorf("Name mismatch: expected %s, got %s", playlist.Name, unmarshaled.Name)
		}
		if unmarshaled.Type != playlist.Type {
			t.Errorf("Type mismatch: expected %s, got %s", playlist.Type, unmarshaled.Type)
		}
		if unmarshaled.Smart != playlist.Smart {
			t.Errorf("Smart mismatch: expected %t, got %t", playlist.Smart, unmarshaled.Smart)
		}
		if unmarshaled.TrackCount != playlist.TrackCount {
			t.Errorf("TrackCount mismatch: expected %d, got %d", playlist.TrackCount, unmarshaled.TrackCount)
		}
	})

	t.Run("Video playlist", func(t *testing.T) {
		playlist := PlexPlaylist{
			Name:       "Movies",
			Type:       "video",
			Smart:      false,
			TrackCount: 20,
		}

		if playlist.Type != "video" {
			t.Errorf("Expected type 'video', got %s", playlist.Type)
		}
	})
}

func TestPlexModelsEdgeCases(t *testing.T) {
	t.Run("Track with very old last played", func(t *testing.T) {
		oldTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		track := PlexTrack{
			Title:      "Old Song",
			Artist:     "Vintage Artist",
			LastPlayed: oldTime,
		}

		if track.LastPlayed.Year() != 2000 {
			t.Errorf("Expected year 2000, got %d", track.LastPlayed.Year())
		}
	})

	t.Run("Track with future year", func(t *testing.T) {
		track := PlexTrack{
			Title:  "Future Song",
			Artist: "Time Traveler",
			Year:   2050,
		}

		if track.Year != 2050 {
			t.Errorf("Expected year 2050, got %d", track.Year)
		}
	})

	t.Run("Track with very high play count", func(t *testing.T) {
		track := PlexTrack{
			Title:     "Popular Song",
			Artist:    "Hit Artist",
			PlayCount: 999999,
		}

		if track.PlayCount != 999999 {
			t.Errorf("Expected PlayCount 999999, got %d", track.PlayCount)
		}
	})

	t.Run("Playlist with very large track count", func(t *testing.T) {
		playlist := PlexPlaylist{
			Name:       "Massive Collection",
			Type:       "audio",
			Smart:      true,
			TrackCount: 1000000,
		}

		if playlist.TrackCount != 1000000 {
			t.Errorf("Expected TrackCount 1000000, got %d", playlist.TrackCount)
		}
	})
}

func TestPlexTrackCollections(t *testing.T) {
	t.Run("Multiple tracks serialization", func(t *testing.T) {
		tracks := []PlexTrack{
			{
				Title:  "Song 1",
				Artist: "Artist 1",
				Album:  "Album 1",
				Rating: 8,
			},
			{
				Title:  "Song 2",
				Artist: "Artist 2",
				Album:  "Album 2",
				Rating: 9,
			},
		}

		data, err := json.Marshal(tracks)
		if err != nil {
			t.Fatalf("Failed to marshal track collection: %v", err)
		}

		var unmarshaled []PlexTrack
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal track collection: %v", err)
		}

		if len(unmarshaled) != len(tracks) {
			t.Errorf("Track count mismatch: expected %d, got %d", len(tracks), len(unmarshaled))
		}

		for i, track := range tracks {
			if unmarshaled[i].Title != track.Title {
				t.Errorf("Track %d title mismatch: expected %s, got %s", i, track.Title, unmarshaled[i].Title)
			}
			if unmarshaled[i].Artist != track.Artist {
				t.Errorf("Track %d artist mismatch: expected %s, got %s", i, track.Artist, unmarshaled[i].Artist)
			}
			if unmarshaled[i].Rating != track.Rating {
				t.Errorf("Track %d rating mismatch: expected %d, got %d", i, track.Rating, unmarshaled[i].Rating)
			}
		}
	})
}
