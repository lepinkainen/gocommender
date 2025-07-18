package services

import (
	"testing"
	"time"

	"gocommender/internal/models"
)

func TestNewLastFMClient(t *testing.T) {
	client := NewLastFMClient("test-api-key", "test-secret")

	if client == nil {
		t.Fatal("NewLastFMClient returned nil")
	}

	if client.baseURL != "https://ws.audioscrobbler.com/2.0" {
		t.Errorf("Expected baseURL to be https://ws.audioscrobbler.com/2.0, got %s", client.baseURL)
	}

	if client.apiKey != "test-api-key" {
		t.Errorf("Expected apiKey to be test-api-key, got %s", client.apiKey)
	}

	if client.secret != "test-secret" {
		t.Errorf("Expected secret to be test-secret, got %s", client.secret)
	}

	if client.httpClient.Timeout != 10*time.Second {
		t.Errorf("Expected timeout to be 10s, got %v", client.httpClient.Timeout)
	}

	client.Close()
}

func TestNewLastFMClientEmptyCredentials(t *testing.T) {
	client := NewLastFMClient("", "")

	if client == nil {
		t.Fatal("NewLastFMClient returned nil")
	}

	if client.apiKey != "" {
		t.Errorf("Expected empty apiKey, got %s", client.apiKey)
	}

	if client.secret != "" {
		t.Errorf("Expected empty secret, got %s", client.secret)
	}

	client.Close()
}

func TestGetBestLastFMImage(t *testing.T) {
	tests := []struct {
		name     string
		images   []LastFMImage
		expected string
	}{
		{
			name:     "empty images",
			images:   []LastFMImage{},
			expected: "",
		},
		{
			name: "single image",
			images: []LastFMImage{
				{Text: "http://example.com/image.jpg", Size: "large"},
			},
			expected: "http://example.com/image.jpg",
		},
		{
			name: "prefer extralarge",
			images: []LastFMImage{
				{Text: "http://example.com/small.jpg", Size: "small"},
				{Text: "http://example.com/extralarge.jpg", Size: "extralarge"},
				{Text: "http://example.com/large.jpg", Size: "large"},
			},
			expected: "http://example.com/extralarge.jpg",
		},
		{
			name: "prefer large over medium",
			images: []LastFMImage{
				{Text: "http://example.com/small.jpg", Size: "small"},
				{Text: "http://example.com/medium.jpg", Size: "medium"},
				{Text: "http://example.com/large.jpg", Size: "large"},
			},
			expected: "http://example.com/large.jpg",
		},
		{
			name: "skip empty URLs",
			images: []LastFMImage{
				{Text: "", Size: "extralarge"},
				{Text: "http://example.com/valid.jpg", Size: "large"},
			},
			expected: "http://example.com/valid.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBestLastFMImage(tt.images)
			if result != tt.expected {
				t.Errorf("getBestLastFMImage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCleanLastFMBio(t *testing.T) {
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
			input:    "This is a test biography",
			expected: "This is a test biography",
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
			name:     "remove Creative Commons suffix",
			input:    "Great artist biography. User-contributed text is available under the Creative Commons...",
			expected: "Great artist biography.",
		},
		{
			name:     "remove Last.fm link suffix",
			input:    "Artist info here. <a href=\"https://www.last.fm/music/ArtistName\">Read more</a>",
			expected: "Artist info here.",
		},
		{
			name:     "remove lines with HTML tags",
			input:    "Good info\n<span>Bad line</span>\nMore good info",
			expected: "Good info\nMore good info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanLastFMBio(tt.input)
			if result != tt.expected {
				t.Errorf("cleanLastFMBio() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildURL(t *testing.T) {
	client := NewLastFMClient("test-key", "")
	defer client.Close()

	params := map[string]string{
		"method":  "artist.getinfo",
		"artist":  "Test Artist",
		"api_key": "test-key",
		"format":  "json",
	}

	url := client.buildURL(params)

	// Should contain base URL
	if !contains(url, client.baseURL) {
		t.Errorf("URL should contain base URL %s, got %s", client.baseURL, url)
	}

	// Should contain parameters
	expectedParams := []string{"method=artist.getinfo", "api_key=test-key", "format=json"}
	for _, param := range expectedParams {
		if !contains(url, param) {
			t.Errorf("URL should contain parameter %s, got %s", param, url)
		}
	}
}

func TestLastFMClientEnrichArtistNoKey(t *testing.T) {
	client := NewLastFMClient("", "")
	defer client.Close()

	artist := &models.Artist{
		MBID: "test-mbid",
		Name: "Test Artist",
	}

	err := client.EnrichArtist(artist)
	if err != nil {
		t.Errorf("EnrichArtist should not return error for missing API key, got: %v", err)
	}

	if artist.Verified == nil {
		t.Fatal("Expected Verified map to be initialized")
	}

	if artist.Verified["lastfm"] != false {
		t.Errorf("Expected lastfm verification to be false, got %v", artist.Verified["lastfm"])
	}
}

func TestLastFMClientClose(t *testing.T) {
	client := NewLastFMClient("test-key", "test-secret")

	// Should not panic
	client.Close()

	// Multiple calls should be safe
	client.Close()
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexString(s, substr) >= 0))
}

func indexString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
