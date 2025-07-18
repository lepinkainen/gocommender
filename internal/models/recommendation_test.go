package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRecommendRequest(t *testing.T) {
	t.Run("Valid request", func(t *testing.T) {
		request := RecommendRequest{
			PlaylistName: "My Favorites",
			Genre:        stringPtr("rock"),
			MaxResults:   5,
		}

		if request.PlaylistName == "" {
			t.Error("PlaylistName should not be empty")
		}
		if request.Genre == nil {
			t.Error("Genre should be set")
		}
		if *request.Genre != "rock" {
			t.Errorf("Expected genre 'rock', got %s", *request.Genre)
		}
		if request.MaxResults <= 0 {
			t.Error("MaxResults should be positive")
		}
	})

	t.Run("Minimal request", func(t *testing.T) {
		request := RecommendRequest{
			PlaylistName: "Test Playlist",
		}

		if request.PlaylistName == "" {
			t.Error("PlaylistName should not be empty")
		}
		if request.Genre != nil {
			t.Error("Genre should be nil for minimal request")
		}
		if request.MaxResults != 0 {
			t.Error("MaxResults should be zero for minimal request")
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		request := RecommendRequest{
			PlaylistName: "JSON Test",
			Genre:        stringPtr("pop"),
			MaxResults:   3,
		}

		data, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		var unmarshaled RecommendRequest
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal request: %v", err)
		}

		if unmarshaled.PlaylistName != request.PlaylistName {
			t.Errorf("PlaylistName mismatch: expected %s, got %s", request.PlaylistName, unmarshaled.PlaylistName)
		}
		if unmarshaled.MaxResults != request.MaxResults {
			t.Errorf("MaxResults mismatch: expected %d, got %d", request.MaxResults, unmarshaled.MaxResults)
		}
		if unmarshaled.Genre == nil || *unmarshaled.Genre != *request.Genre {
			t.Error("Genre mismatch after JSON round-trip")
		}
	})
}

func TestRecommendResponse(t *testing.T) {
	t.Run("Valid response", func(t *testing.T) {
		response := RecommendResponse{
			Status:    "success",
			RequestID: "test-123",
			Suggestions: []Artist{
				{
					MBID: "artist-1",
					Name: "Test Artist 1",
				},
				{
					MBID: "artist-2",
					Name: "Test Artist 2",
				},
			},
			Metadata: RecommendMetadata{
				SeedTrackCount:   10,
				KnownArtistCount: 5,
				ProcessingTime:   "2.5s",
				CacheHits:        3,
				APICallsMade:     2,
				GeneratedAt:      time.Now(),
			},
		}

		if response.Status != "success" {
			t.Errorf("Expected status 'success', got %s", response.Status)
		}
		if response.RequestID == "" {
			t.Error("RequestID should not be empty")
		}
		if len(response.Suggestions) != 2 {
			t.Errorf("Expected 2 suggestions, got %d", len(response.Suggestions))
		}
		if response.Metadata.SeedTrackCount != 10 {
			t.Errorf("Expected 10 seed tracks, got %d", response.Metadata.SeedTrackCount)
		}
	})

	t.Run("Error response", func(t *testing.T) {
		response := RecommendResponse{
			Status:      "error",
			RequestID:   "error-123",
			Suggestions: []Artist{},
			Error:       "Test error message",
		}

		if response.Status != "error" {
			t.Errorf("Expected status 'error', got %s", response.Status)
		}
		if response.Error == "" {
			t.Error("Error message should not be empty for error response")
		}
		if len(response.Suggestions) != 0 {
			t.Error("Error response should have no suggestions")
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		response := RecommendResponse{
			Status:    "success",
			RequestID: "json-test",
			Suggestions: []Artist{
				{MBID: "test-mbid", Name: "Test Artist"},
			},
			Metadata: RecommendMetadata{
				SeedTrackCount: 5,
				ProcessingTime: "1s",
				GeneratedAt:    time.Now(),
			},
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal response: %v", err)
		}

		var unmarshaled RecommendResponse
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if unmarshaled.Status != response.Status {
			t.Errorf("Status mismatch: expected %s, got %s", response.Status, unmarshaled.Status)
		}
		if len(unmarshaled.Suggestions) != len(response.Suggestions) {
			t.Errorf("Suggestions count mismatch: expected %d, got %d", len(response.Suggestions), len(unmarshaled.Suggestions))
		}
		if unmarshaled.Metadata.SeedTrackCount != response.Metadata.SeedTrackCount {
			t.Errorf("Metadata mismatch: expected %d seed tracks, got %d", response.Metadata.SeedTrackCount, unmarshaled.Metadata.SeedTrackCount)
		}
	})
}

func TestRecommendMetadata(t *testing.T) {
	t.Run("Valid metadata", func(t *testing.T) {
		metadata := RecommendMetadata{
			SeedTrackCount:   15,
			KnownArtistCount: 8,
			ProcessingTime:   "3.2s",
			CacheHits:        5,
			APICallsMade:     10,
			GeneratedAt:      time.Now(),
		}

		if metadata.SeedTrackCount <= 0 {
			t.Error("SeedTrackCount should be positive")
		}
		if metadata.KnownArtistCount <= 0 {
			t.Error("KnownArtistCount should be positive")
		}
		if metadata.ProcessingTime == "" {
			t.Error("ProcessingTime should not be empty")
		}
		if metadata.CacheHits < 0 {
			t.Error("CacheHits should not be negative")
		}
		if metadata.APICallsMade < 0 {
			t.Error("APICallsMade should not be negative")
		}
		if metadata.GeneratedAt.IsZero() {
			t.Error("GeneratedAt should be set")
		}
	})

	t.Run("Zero values metadata", func(t *testing.T) {
		metadata := RecommendMetadata{
			SeedTrackCount:   0,
			KnownArtistCount: 0,
			ProcessingTime:   "0s",
			CacheHits:        0,
			APICallsMade:     0,
			GeneratedAt:      time.Now(),
		}

		// Zero values should be acceptable
		if metadata.SeedTrackCount < 0 {
			t.Error("SeedTrackCount should not be negative")
		}
		if metadata.KnownArtistCount < 0 {
			t.Error("KnownArtistCount should not be negative")
		}
		if metadata.CacheHits < 0 {
			t.Error("CacheHits should not be negative")
		}
		if metadata.APICallsMade < 0 {
			t.Error("APICallsMade should not be negative")
		}
	})

	t.Run("JSON serialization with timestamps", func(t *testing.T) {
		now := time.Now()
		metadata := RecommendMetadata{
			SeedTrackCount:   7,
			KnownArtistCount: 3,
			ProcessingTime:   "1.8s",
			CacheHits:        2,
			APICallsMade:     4,
			GeneratedAt:      now,
		}

		data, err := json.Marshal(metadata)
		if err != nil {
			t.Fatalf("Failed to marshal metadata: %v", err)
		}

		var unmarshaled RecommendMetadata
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal metadata: %v", err)
		}

		if unmarshaled.SeedTrackCount != metadata.SeedTrackCount {
			t.Errorf("SeedTrackCount mismatch: expected %d, got %d", metadata.SeedTrackCount, unmarshaled.SeedTrackCount)
		}

		// Time comparison with some tolerance for JSON precision
		timeDiff := unmarshaled.GeneratedAt.Sub(metadata.GeneratedAt)
		if timeDiff > time.Second || timeDiff < -time.Second {
			t.Errorf("GeneratedAt time mismatch: expected %v, got %v", metadata.GeneratedAt, unmarshaled.GeneratedAt)
		}
	})
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}
