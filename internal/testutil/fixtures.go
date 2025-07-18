package testutil

import (
	"time"

	"gocommender/internal/models"
)

// TestArtist creates a test artist with common fields populated
func TestArtist() *models.Artist {
	return &models.Artist{
		MBID:        "b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d",
		Name:        "The Beatles",
		AlbumCount:  13,
		YearsActive: "1960-1970",
		Description: "English rock band formed in Liverpool in 1960.",
		Genres:      models.Genres{"rock", "pop", "psychedelic"},
		Country:     "United Kingdom",
		ImageURL:    "https://example.com/beatles.jpg",
		Verified: models.VerificationMap{
			"musicbrainz": true,
			"discogs":     true,
			"lastfm":      true,
		},
		ExternalURLs: models.ExternalURLs{
			MusicBrainz: "https://musicbrainz.org/artist/b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d",
			Discogs:     "https://discogs.com/artist/82730",
			LastFM:      "https://last.fm/music/The%20Beatles",
		},
		LastUpdated: time.Now(),
		CacheExpiry: time.Now().Add(24 * time.Hour),
	}
}

// TestArtistMinimal creates a minimal test artist for basic testing
func TestArtistMinimal() *models.Artist {
	return &models.Artist{
		MBID:        "minimal-test-mbid-123",
		Name:        "Test Artist",
		LastUpdated: time.Now(),
		CacheExpiry: time.Now().Add(time.Hour),
	}
}

// TestArtistUnverified creates an artist with no successful verifications
func TestArtistUnverified() *models.Artist {
	return &models.Artist{
		MBID: "unverified-test-mbid-456",
		Name: "Unverified Artist",
		Verified: models.VerificationMap{
			"musicbrainz": false,
			"discogs":     false,
			"lastfm":      false,
		},
		LastUpdated: time.Now(),
		CacheExpiry: time.Now().Add(time.Hour),
	}
}

// TestArtistExpired creates an artist with expired cache
func TestArtistExpired() *models.Artist {
	return &models.Artist{
		MBID:        "expired-test-mbid-789",
		Name:        "Expired Artist",
		LastUpdated: time.Now().Add(-2 * time.Hour),
		CacheExpiry: time.Now().Add(-time.Hour),
	}
}

// TestPlexTracks creates sample Plex tracks for testing
func TestPlexTracks() []models.PlexTrack {
	return []models.PlexTrack{
		{
			Title:      "Hey Jude",
			Artist:     "The Beatles",
			Album:      "The Beatles (White Album)",
			Year:       1968,
			Rating:     9,
			PlayCount:  42,
			LastPlayed: time.Now().Add(-24 * time.Hour),
		},
		{
			Title:      "Come Together",
			Artist:     "The Beatles",
			Album:      "Abbey Road",
			Year:       1969,
			Rating:     8,
			PlayCount:  38,
			LastPlayed: time.Now().Add(-48 * time.Hour),
		},
		{
			Title:      "Wish You Were Here",
			Artist:     "Pink Floyd",
			Album:      "Wish You Were Here",
			Year:       1975,
			Rating:     10,
			PlayCount:  55,
			LastPlayed: time.Now().Add(-12 * time.Hour),
		},
	}
}

// TestPlexPlaylist creates a sample Plex playlist
func TestPlexPlaylist() models.PlexPlaylist {
	return models.PlexPlaylist{
		Name:       "My Favorites",
		Type:       "audio",
		Smart:      false,
		TrackCount: 25,
	}
}

// TestPlexPlaylists creates multiple sample Plex playlists
func TestPlexPlaylists() []models.PlexPlaylist {
	return []models.PlexPlaylist{
		{
			Name:       "My Favorites",
			Type:       "audio",
			Smart:      false,
			TrackCount: 25,
		},
		{
			Name:       "Rock Classics",
			Type:       "audio",
			Smart:      true,
			TrackCount: 100,
		},
		{
			Name:       "Recently Added",
			Type:       "audio",
			Smart:      true,
			TrackCount: 50,
		},
	}
}

// TestRecommendRequest creates a valid recommendation request
func TestRecommendRequest() models.RecommendRequest {
	return models.RecommendRequest{
		PlaylistName: "My Favorites",
		Genre:        stringPtr("rock"),
		MaxResults:   5,
	}
}

// TestRecommendRequestBasic creates a minimal recommendation request
func TestRecommendRequestBasic() models.RecommendRequest {
	return models.RecommendRequest{
		PlaylistName: "Test Playlist",
		MaxResults:   3,
	}
}

// TestRecommendResponse creates a sample recommendation response
func TestRecommendResponse() *models.RecommendResponse {
	return &models.RecommendResponse{
		Status:    "success",
		RequestID: "test-request-123",
		Suggestions: []models.Artist{
			*TestArtist(),
			*TestArtistMinimal(),
		},
		Metadata: models.RecommendMetadata{
			SeedTrackCount:   10,
			KnownArtistCount: 5,
			ProcessingTime:   "2.5s",
			CacheHits:        3,
			APICallsMade:     2,
			GeneratedAt:      time.Now(),
		},
	}
}

// TestArtistCollection creates a collection of test artists for bulk operations
func TestArtistCollection() []models.Artist {
	return []models.Artist{
		*TestArtist(),
		{
			MBID:        "second-test-mbid-456",
			Name:        "Pink Floyd",
			AlbumCount:  15,
			YearsActive: "1965-1995",
			Description: "English progressive rock band.",
			Genres:      models.Genres{"progressive rock", "psychedelic"},
			Country:     "United Kingdom",
			Verified: models.VerificationMap{
				"musicbrainz": true,
				"discogs":     true,
			},
			LastUpdated: time.Now(),
			CacheExpiry: time.Now().Add(24 * time.Hour),
		},
		{
			MBID:        "third-test-mbid-789",
			Name:        "Led Zeppelin",
			AlbumCount:  8,
			YearsActive: "1968-1980",
			Description: "English rock band.",
			Genres:      models.Genres{"hard rock", "blues rock"},
			Country:     "United Kingdom",
			Verified: models.VerificationMap{
				"musicbrainz": true,
			},
			LastUpdated: time.Now(),
			CacheExpiry: time.Now().Add(24 * time.Hour),
		},
	}
}

// TestVerificationMap creates a test verification map
func TestVerificationMap() models.VerificationMap {
	return models.VerificationMap{
		"musicbrainz": true,
		"discogs":     true,
		"lastfm":      false,
	}
}

// TestGenres creates a test genres slice
func TestGenres() models.Genres {
	return models.Genres{"rock", "alternative", "indie"}
}

// TestExternalURLs creates test external URLs
func TestExternalURLs() models.ExternalURLs {
	return models.ExternalURLs{
		MusicBrainz: "https://musicbrainz.org/artist/test-mbid",
		Discogs:     "https://discogs.com/artist/123",
		LastFM:      "https://last.fm/music/Test%20Artist",
	}
}

// stringPtr is a helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// intPtr is a helper function to create int pointers
func intPtr(i int) *int {
	return &i
}

// timePtr is a helper function to create time pointers
func timePtr(t time.Time) *time.Time {
	return &t
}
