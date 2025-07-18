package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestArtistJSONSerialization(t *testing.T) {
	artist := &Artist{
		MBID: "test-mbid-123",
		Name: "Test Artist",
		Verified: VerificationMap{
			"musicbrainz": true,
			"discogs":     false,
		},
		Genres: Genres{"rock", "alternative"},
		ExternalURLs: ExternalURLs{
			MusicBrainz: "https://musicbrainz.org/artist/test-mbid-123",
			Discogs:     "https://discogs.com/artist/123",
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(artist)
	if err != nil {
		t.Fatalf("Failed to marshal artist: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled Artist
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal artist: %v", err)
	}

	// Verify data integrity
	if unmarshaled.MBID != artist.MBID {
		t.Errorf("MBID mismatch: expected %s, got %s", artist.MBID, unmarshaled.MBID)
	}

	if len(unmarshaled.Verified) != len(artist.Verified) {
		t.Errorf("Verified map length mismatch: expected %d, got %d", len(artist.Verified), len(unmarshaled.Verified))
	}

	if len(unmarshaled.Genres) != len(artist.Genres) {
		t.Errorf("Genres length mismatch: expected %d, got %d", len(artist.Genres), len(unmarshaled.Genres))
	}
}

func TestVerificationMapDatabaseTypes(t *testing.T) {
	vm := VerificationMap{
		"musicbrainz": true,
		"discogs":     false,
		"lastfm":      true,
	}

	// Test Value() method
	value, err := vm.Value()
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	// Test Scan() method
	var scanned VerificationMap
	err = scanned.Scan(value)
	if err != nil {
		t.Fatalf("Failed to scan value: %v", err)
	}

	if len(scanned) != len(vm) {
		t.Errorf("Length mismatch after scan: expected %d, got %d", len(vm), len(scanned))
	}

	for key, expected := range vm {
		if actual, exists := scanned[key]; !exists || actual != expected {
			t.Errorf("Value mismatch for key %s: expected %v, got %v", key, expected, actual)
		}
	}
}

func TestGenresDatabaseOperations(t *testing.T) {
	genres := Genres{"rock", "pop", "alternative"}

	t.Run("Value method", func(t *testing.T) {
		value, err := genres.Value()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if value == nil {
			t.Fatal("Expected value to not be nil")
		}

		// Should be JSON bytes
		var jsonStr string
		switch v := value.(type) {
		case []byte:
			jsonStr = string(v)
		case string:
			jsonStr = v
		default:
			t.Fatalf("Expected value to be []byte or string, got %T", value)
		}

		var unmarshaled []string
		err = json.Unmarshal([]byte(jsonStr), &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(unmarshaled) != len(genres) {
			t.Errorf("Length mismatch: expected %d, got %d", len(genres), len(unmarshaled))
		}
	})

	t.Run("Scan method", func(t *testing.T) {
		// Test with string JSON
		jsonStr := `["rock","pop","alternative"]`
		var scanned Genres
		err := scanned.Scan(jsonStr)
		if err != nil {
			t.Fatalf("Failed to scan: %v", err)
		}

		if len(scanned) != len(genres) {
			t.Errorf("Length mismatch: expected %d, got %d", len(genres), len(scanned))
		}
	})

	t.Run("Scan method with nil", func(t *testing.T) {
		var scanned Genres
		err := scanned.Scan(nil)
		if err != nil {
			t.Fatalf("Failed to scan nil: %v", err)
		}

		if len(scanned) != 0 {
			t.Errorf("Expected empty genres, got %d items", len(scanned))
		}
	})

	t.Run("Scan method with invalid type", func(t *testing.T) {
		var scanned Genres
		err := scanned.Scan(123)
		// Note: The implementation doesn't return error for invalid types, it just creates empty slice
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(scanned) != 0 {
			t.Error("Expected empty genres for invalid scan type")
		}
	})
}

func TestExternalURLsDatabaseOperations(t *testing.T) {
	urls := ExternalURLs{
		MusicBrainz: "https://musicbrainz.org/artist/test",
		Discogs:     "https://discogs.com/artist/123",
		LastFM:      "https://last.fm/music/test",
	}

	t.Run("Value method", func(t *testing.T) {
		value, err := urls.Value()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if value == nil {
			t.Fatal("Expected value to not be nil")
		}

		// Should be JSON bytes
		var jsonStr string
		switch v := value.(type) {
		case []byte:
			jsonStr = string(v)
		case string:
			jsonStr = v
		default:
			t.Fatalf("Expected value to be []byte or string, got %T", value)
		}

		var unmarshaled ExternalURLs
		err = json.Unmarshal([]byte(jsonStr), &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if unmarshaled.MusicBrainz != urls.MusicBrainz {
			t.Errorf("MusicBrainz URL mismatch: expected %s, got %s", urls.MusicBrainz, unmarshaled.MusicBrainz)
		}
	})

	t.Run("Scan method", func(t *testing.T) {
		jsonStr := `{"musicbrainz":"https://musicbrainz.org/artist/test","discogs":"https://discogs.com/artist/123","lastfm":"https://last.fm/music/test"}`
		var scanned ExternalURLs
		err := scanned.Scan(jsonStr)
		if err != nil {
			t.Fatalf("Failed to scan: %v", err)
		}

		if scanned.MusicBrainz != urls.MusicBrainz {
			t.Errorf("MusicBrainz URL mismatch: expected %s, got %s", urls.MusicBrainz, scanned.MusicBrainz)
		}
	})
}

func TestVerificationMapOperations(t *testing.T) {
	t.Run("Empty verification map", func(t *testing.T) {
		vm := VerificationMap{}
		value, err := vm.Value()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		var scanned VerificationMap
		err = scanned.Scan(value)
		if err != nil {
			t.Fatalf("Failed to scan: %v", err)
		}

		if len(scanned) != 0 {
			t.Errorf("Expected empty map, got %d items", len(scanned))
		}
	})

	t.Run("Mixed verifications", func(t *testing.T) {
		vm := VerificationMap{
			"musicbrainz": true,
			"discogs":     false,
			"lastfm":      true,
		}

		value, err := vm.Value()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		var scanned VerificationMap
		err = scanned.Scan(value)
		if err != nil {
			t.Fatalf("Failed to scan: %v", err)
		}

		if len(scanned) != len(vm) {
			t.Errorf("Length mismatch: expected %d, got %d", len(vm), len(scanned))
		}

		for key, expected := range vm {
			if actual, exists := scanned[key]; !exists || actual != expected {
				t.Errorf("Value mismatch for key %s: expected %v, got %v", key, expected, actual)
			}
		}
	})

	t.Run("Scan with invalid JSON", func(t *testing.T) {
		var vm VerificationMap
		err := vm.Scan("invalid json")
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})
}

func TestArtistValidation(t *testing.T) {
	t.Run("Artist cache expiry logic", func(t *testing.T) {
		artist := &Artist{
			MBID:        "test-mbid",
			Name:        "Test Artist",
			CacheExpiry: time.Now().Add(time.Hour),
		}

		if time.Now().After(artist.CacheExpiry) {
			t.Error("Artist should not be expired")
		}

		// Test expired artist
		artist.CacheExpiry = time.Now().Add(-time.Hour)
		if !time.Now().After(artist.CacheExpiry) {
			t.Error("Artist should be expired")
		}
	})

	t.Run("Artist with all fields", func(t *testing.T) {
		artist := &Artist{
			MBID:        "full-test-mbid",
			Name:        "Full Test Artist",
			AlbumCount:  10,
			YearsActive: "2000-2020",
			Description: "A comprehensive test artist",
			Genres:      Genres{"rock", "pop"},
			Country:     "Test Country",
			ImageURL:    "https://example.com/image.jpg",
			Verified: VerificationMap{
				"musicbrainz": true,
				"discogs":     true,
				"lastfm":      false,
			},
			ExternalURLs: ExternalURLs{
				MusicBrainz: "https://musicbrainz.org/artist/full-test-mbid",
				Discogs:     "https://discogs.com/artist/123",
				LastFM:      "https://last.fm/music/Full%20Test%20Artist",
			},
			LastUpdated: time.Now(),
			CacheExpiry: time.Now().Add(24 * time.Hour),
		}

		// Verify all fields are set
		if artist.MBID == "" {
			t.Error("MBID should not be empty")
		}
		if artist.Name == "" {
			t.Error("Name should not be empty")
		}
		if artist.AlbumCount <= 0 {
			t.Error("AlbumCount should be positive")
		}
		if len(artist.Genres) == 0 {
			t.Error("Genres should not be empty")
		}
		if len(artist.Verified) == 0 {
			t.Error("Verified should not be empty")
		}
		if artist.LastUpdated.IsZero() {
			t.Error("LastUpdated should be set")
		}
		if artist.CacheExpiry.IsZero() {
			t.Error("CacheExpiry should be set")
		}
	})
}
