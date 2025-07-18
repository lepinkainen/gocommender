package services

import (
	"testing"
	"time"

	"gocommender/internal/models"
)

func TestNewEnrichmentService(t *testing.T) {
	service := NewEnrichmentService("discogs-token", "lastfm-key", "lastfm-secret")
	defer service.Close()

	if service == nil {
		t.Fatal("NewEnrichmentService returned nil")
	}

	if service.musicbrainz == nil {
		t.Error("Expected musicbrainz client to be initialized")
	}

	if service.discogs == nil {
		t.Error("Expected discogs client to be initialized")
	}

	if service.lastfm == nil {
		t.Error("Expected lastfm client to be initialized")
	}
}

func TestNewEnrichmentServiceEmptyTokens(t *testing.T) {
	service := NewEnrichmentService("", "", "")
	defer service.Close()

	if service == nil {
		t.Fatal("NewEnrichmentService returned nil")
	}

	// Should still initialize clients even with empty tokens
	if service.musicbrainz == nil {
		t.Error("Expected musicbrainz client to be initialized")
	}

	if service.discogs == nil {
		t.Error("Expected discogs client to be initialized")
	}

	if service.lastfm == nil {
		t.Error("Expected lastfm client to be initialized")
	}
}

func TestHasSuccessfulVerification(t *testing.T) {
	tests := []struct {
		name     string
		artist   *models.Artist
		expected bool
	}{
		{
			name: "nil verification map",
			artist: &models.Artist{
				Name: "Test Artist",
			},
			expected: false,
		},
		{
			name: "empty verification map",
			artist: &models.Artist{
				Name:     "Test Artist",
				Verified: models.VerificationMap{},
			},
			expected: false,
		},
		{
			name: "all false verifications",
			artist: &models.Artist{
				Name: "Test Artist",
				Verified: models.VerificationMap{
					"musicbrainz": false,
					"discogs":     false,
					"lastfm":      false,
				},
			},
			expected: false,
		},
		{
			name: "one true verification",
			artist: &models.Artist{
				Name: "Test Artist",
				Verified: models.VerificationMap{
					"musicbrainz": true,
					"discogs":     false,
					"lastfm":      false,
				},
			},
			expected: true,
		},
		{
			name: "multiple true verifications",
			artist: &models.Artist{
				Name: "Test Artist",
				Verified: models.VerificationMap{
					"musicbrainz": true,
					"discogs":     true,
					"lastfm":      false,
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasSuccessfulVerification(tt.artist)
			if result != tt.expected {
				t.Errorf("hasSuccessfulVerification() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetEnrichmentStatus(t *testing.T) {
	service := NewEnrichmentService("", "", "")
	defer service.Close()

	artist := &models.Artist{
		MBID: "test-mbid",
		Name: "Test Artist",
		Verified: models.VerificationMap{
			"musicbrainz": true,
			"discogs":     false,
		},
		Description: "Test description",
		ImageURL:    "",
		Genres:      []string{"rock", "pop"},
		Country:     "US",
		ExternalURLs: models.ExternalURLs{
			MusicBrainz: "https://musicbrainz.org/artist/test-mbid",
			Discogs:     "",
			LastFM:      "https://last.fm/music/Test+Artist",
		},
		LastUpdated: time.Now(),
		CacheExpiry: time.Now().Add(24 * time.Hour),
	}

	status := service.GetEnrichmentStatus(artist)

	// Check basic fields
	if status["mbid"] != artist.MBID {
		t.Errorf("Expected MBID %s, got %v", artist.MBID, status["mbid"])
	}

	if status["name"] != artist.Name {
		t.Errorf("Expected name %s, got %v", artist.Name, status["name"])
	}

	// Check verification map
	verified, ok := status["verified"].(models.VerificationMap)
	if !ok {
		t.Fatal("Expected verified to be VerificationMap")
	}

	if verified["musicbrainz"] != true {
		t.Errorf("Expected musicbrainz verification to be true, got %v", verified["musicbrainz"])
	}

	// Check sources
	sources, ok := status["sources"].(map[string]bool)
	if !ok {
		t.Fatal("Expected sources to be map[string]bool")
	}

	if !sources["musicbrainz"] {
		t.Error("Expected musicbrainz source to be true")
	}

	if sources["discogs"] {
		t.Error("Expected discogs source to be false (empty URL)")
	}

	if !sources["lastfm"] {
		t.Error("Expected lastfm source to be true")
	}

	// Check data completeness
	if status["has_description"] != true {
		t.Error("Expected has_description to be true")
	}

	if status["has_image"] != false {
		t.Error("Expected has_image to be false")
	}

	if status["has_genres"] != true {
		t.Error("Expected has_genres to be true")
	}

	if status["has_country"] != true {
		t.Error("Expected has_country to be true")
	}
}

func TestValidateEnrichmentConfig(t *testing.T) {
	tests := []struct {
		name         string
		discogsToken string
		lastfmKey    string
		expected     map[string]bool
	}{
		{
			name:         "all tokens empty",
			discogsToken: "",
			lastfmKey:    "",
			expected: map[string]bool{
				"musicbrainz": true,  // Always available
				"discogs":     false, // No token
				"lastfm":      false, // No key
			},
		},
		{
			name:         "discogs token only",
			discogsToken: "test-token",
			lastfmKey:    "",
			expected: map[string]bool{
				"musicbrainz": true,  // Always available
				"discogs":     true,  // Has token
				"lastfm":      false, // No key
			},
		},
		{
			name:         "lastfm key only",
			discogsToken: "",
			lastfmKey:    "test-key",
			expected: map[string]bool{
				"musicbrainz": true,  // Always available
				"discogs":     false, // No token
				"lastfm":      true,  // Has key
			},
		},
		{
			name:         "all tokens provided",
			discogsToken: "test-token",
			lastfmKey:    "test-key",
			expected: map[string]bool{
				"musicbrainz": true, // Always available
				"discogs":     true, // Has token
				"lastfm":      true, // Has key
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewEnrichmentService(tt.discogsToken, tt.lastfmKey, "")
			defer service.Close()

			config := service.ValidateEnrichmentConfig()

			for service, expectedAvailable := range tt.expected {
				if config[service] != expectedAvailable {
					t.Errorf("Expected %s to be %v, got %v", service, expectedAvailable, config[service])
				}
			}
		})
	}
}

func TestEnrichmentServiceClose(t *testing.T) {
	service := NewEnrichmentService("test-token", "test-key", "test-secret")

	// Should not panic
	service.Close()

	// Multiple calls should be safe
	service.Close()
}

func TestEnrichExistingArtistCacheValid(t *testing.T) {
	service := NewEnrichmentService("", "", "")
	defer service.Close()

	// Create artist with future cache expiry
	artist := &models.Artist{
		MBID:        "test-mbid",
		Name:        "Test Artist",
		CacheExpiry: time.Now().Add(1 * time.Hour), // Still valid
		Verified:    models.VerificationMap{"musicbrainz": true},
	}

	options := &EnrichmentOptions{
		ForceUpdate: false,
	}

	err := service.EnrichExistingArtist(artist, options)
	if err != nil {
		t.Errorf("EnrichExistingArtist should not return error for valid cache, got: %v", err)
	}
}

func TestEnrichExistingArtistForceUpdate(t *testing.T) {
	service := NewEnrichmentService("", "", "")
	defer service.Close()

	artist := &models.Artist{
		MBID:        "test-mbid",
		Name:        "Test Artist",
		CacheExpiry: time.Now().Add(1 * time.Hour), // Still valid but force update
		Verified:    models.VerificationMap{},
	}

	options := &EnrichmentOptions{
		ForceUpdate:    true,
		SourcePriority: []string{"discogs", "lastfm"},
	}

	// Should not fail even with empty tokens (graceful degradation)
	err := service.EnrichExistingArtist(artist, options)
	if err != nil {
		// Errors are expected but shouldn't panic
		t.Logf("Expected enrichment errors with empty tokens: %v", err)
	}

	// Should update timestamps
	if artist.LastUpdated.IsZero() {
		t.Error("Expected LastUpdated to be set")
	}
}
