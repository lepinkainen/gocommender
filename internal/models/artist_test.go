package models

import (
	"encoding/json"
	"testing"
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
