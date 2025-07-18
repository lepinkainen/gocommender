package services

import (
	"testing"
	"time"

	"gocommender/internal/models"
)

func TestNewDiscogsClient(t *testing.T) {
	client := NewDiscogsClient("test-token")

	if client == nil {
		t.Fatal("NewDiscogsClient returned nil")
	}

	if client.baseURL != "https://api.discogs.com" {
		t.Errorf("Expected baseURL to be https://api.discogs.com, got %s", client.baseURL)
	}

	if client.token != "test-token" {
		t.Errorf("Expected token to be test-token, got %s", client.token)
	}

	if client.httpClient.Timeout != 10*time.Second {
		t.Errorf("Expected timeout to be 10s, got %v", client.httpClient.Timeout)
	}

	client.Close()
}

func TestNewDiscogsClientEmptyToken(t *testing.T) {
	client := NewDiscogsClient("")

	if client == nil {
		t.Fatal("NewDiscogsClient returned nil")
	}

	if client.token != "" {
		t.Errorf("Expected empty token, got %s", client.token)
	}

	client.Close()
}

func TestGetBestImage(t *testing.T) {
	tests := []struct {
		name     string
		images   []DiscogsImage
		expected string
	}{
		{
			name:     "empty images",
			images:   []DiscogsImage{},
			expected: "",
		},
		{
			name: "single image",
			images: []DiscogsImage{
				{Type: "primary", URI: "http://example.com/image.jpg"},
			},
			expected: "http://example.com/image.jpg",
		},
		{
			name: "prefer primary type",
			images: []DiscogsImage{
				{Type: "secondary", URI: "http://example.com/secondary.jpg"},
				{Type: "primary", URI: "http://example.com/primary.jpg"},
			},
			expected: "http://example.com/primary.jpg",
		},
		{
			name: "fallback to any image",
			images: []DiscogsImage{
				{Type: "secondary", URI: "http://example.com/fallback.jpg"},
			},
			expected: "http://example.com/fallback.jpg",
		},
		{
			name: "skip empty URIs",
			images: []DiscogsImage{
				{Type: "primary", URI: ""},
				{Type: "secondary", URI: "http://example.com/valid.jpg"},
			},
			expected: "http://example.com/valid.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBestImage(tt.images)
			if result != tt.expected {
				t.Errorf("getBestImage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCleanDescription(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "This is a test description",
			expected: "This is a test description",
		},
		{
			name:     "remove HTML br tags",
			input:    "Line 1<br>Line 2<br/>Line 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "remove HTML p tags",
			input:    "<p>Paragraph 1</p><p>Paragraph 2</p>",
			expected: "Paragraph 1\nParagraph 2",
		},
		{
			name:     "remove Discogs artist links",
			input:    "Good band\n[a=Related Artist] not included",
			expected: "Good band",
		},
		{
			name:     "mixed formatting",
			input:    "<p>Great artist</p><br/>Born in 1970<br>[a=Similar Artist]",
			expected: "Great artist\nBorn in 1970",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanDescription(tt.input)
			if result != tt.expected {
				t.Errorf("cleanDescription() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDiscogsClientEnrichArtistNoToken(t *testing.T) {
	client := NewDiscogsClient("")
	defer client.Close()

	artist := &models.Artist{
		MBID: "test-mbid",
		Name: "Test Artist",
	}

	err := client.EnrichArtist(artist)
	if err != nil {
		t.Errorf("EnrichArtist should not return error for missing token, got: %v", err)
	}

	if artist.Verified == nil {
		t.Fatal("Expected Verified map to be initialized")
	}

	if artist.Verified["discogs"] != false {
		t.Errorf("Expected discogs verification to be false, got %v", artist.Verified["discogs"])
	}
}

func TestDiscogsClientClose(t *testing.T) {
	client := NewDiscogsClient("test-token")

	// Should not panic
	client.Close()

	// Multiple calls should be safe
	client.Close()
}
