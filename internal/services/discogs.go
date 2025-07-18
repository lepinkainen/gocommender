package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gocommender/internal/models"
)

// DiscogsClient handles Discogs API interactions
type DiscogsClient struct {
	baseURL     string
	httpClient  *http.Client
	userAgent   string
	token       string
	rateLimiter *time.Ticker
}

// DiscogsArtist represents artist data from Discogs API
type DiscogsArtist struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`
	Profile     string         `json:"profile"`
	Images      []DiscogsImage `json:"images"`
	URLs        []string       `json:"urls"`
	NameVars    []string       `json:"namevariations"`
	RealName    string         `json:"realname"`
	DataQuality string         `json:"data_quality"`
}

type DiscogsImage struct {
	Type   string `json:"type"`
	URI    string `json:"uri"`
	URI150 string `json:"uri150"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// DiscogsSearchResult represents search results from Discogs API
type DiscogsSearchResult struct {
	Results []DiscogsSearchArtist `json:"results"`
}

type DiscogsSearchArtist struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Type       string `json:"type"`
	Thumb      string `json:"thumb"`
	CoverImage string `json:"cover_image"`
}

// NewDiscogsClient creates a new Discogs API client
func NewDiscogsClient(token string) *DiscogsClient {
	return &DiscogsClient{
		baseURL: "https://api.discogs.com",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		userAgent:   "GoCommender/1.0 +https://github.com/lepinkainen/gocommender",
		token:       token,
		rateLimiter: time.NewTicker(1100 * time.Millisecond), // ~50 requests/minute
	}
}

// SearchArtist searches for artists by name and returns the best match
func (c *DiscogsClient) SearchArtist(name string) (*DiscogsSearchArtist, error) {
	if c.token == "" {
		return nil, fmt.Errorf("discogs token not configured")
	}

	<-c.rateLimiter.C // Rate limiting

	query := url.QueryEscape(name)
	urlStr := fmt.Sprintf("%s/database/search?q=%s&type=artist&token=%s", c.baseURL, query, c.token)

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid discogs token")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var searchResult DiscogsSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(searchResult.Results) == 0 {
		return nil, fmt.Errorf("no artists found for '%s'", name)
	}

	// Find the best match (exact name match preferred)
	for _, artist := range searchResult.Results {
		if strings.EqualFold(artist.Title, name) {
			return &artist, nil
		}
	}

	// Return first result if no exact match
	return &searchResult.Results[0], nil
}

// GetArtistByID fetches detailed artist information by Discogs ID
func (c *DiscogsClient) GetArtistByID(id int) (*DiscogsArtist, error) {
	if c.token == "" {
		return nil, fmt.Errorf("discogs token not configured")
	}

	<-c.rateLimiter.C // Rate limiting

	urlStr := fmt.Sprintf("%s/artists/%d?token=%s", c.baseURL, id, c.token)

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("artist with ID %d not found", id)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid discogs token")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var artist DiscogsArtist
	if err := json.NewDecoder(resp.Body).Decode(&artist); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &artist, nil
}

// EnrichArtist enriches an existing Artist model with Discogs data
func (c *DiscogsClient) EnrichArtist(artist *models.Artist) error {
	if c.token == "" {
		// Graceful degradation - just mark as not verified
		if artist.Verified == nil {
			artist.Verified = make(models.VerificationMap)
		}
		artist.Verified["discogs"] = false
		return nil
	}

	// Search for artist by name
	searchResult, err := c.SearchArtist(artist.Name)
	if err != nil {
		// Mark as failed verification but don't return error
		if artist.Verified == nil {
			artist.Verified = make(models.VerificationMap)
		}
		artist.Verified["discogs"] = false
		return nil
	}

	// Get detailed artist data
	discogsArtist, err := c.GetArtistByID(searchResult.ID)
	if err != nil {
		// Mark as failed verification but don't return error
		if artist.Verified == nil {
			artist.Verified = make(models.VerificationMap)
		}
		artist.Verified["discogs"] = false
		return nil
	}

	// Enrich the artist with Discogs data
	if artist.Verified == nil {
		artist.Verified = make(models.VerificationMap)
	}
	artist.Verified["discogs"] = true

	// Add description if not already present and available
	if artist.Description == "" && discogsArtist.Profile != "" {
		artist.Description = cleanDescription(discogsArtist.Profile)
	}

	// Add image URL if not already present and available
	if artist.ImageURL == "" {
		imageURL := getBestImage(discogsArtist.Images)
		if imageURL != "" {
			artist.ImageURL = imageURL
		}
	}

	// Add Discogs URL
	artist.ExternalURLs.Discogs = fmt.Sprintf("https://www.discogs.com/artist/%d", discogsArtist.ID)

	return nil
}

// getBestImage selects the best image from Discogs images
func getBestImage(images []DiscogsImage) string {
	if len(images) == 0 {
		return ""
	}

	// Prefer primary images first
	for _, img := range images {
		if img.Type == "primary" && img.URI != "" {
			return img.URI
		}
	}

	// Fall back to any available image
	for _, img := range images {
		if img.URI != "" {
			return img.URI
		}
	}

	return ""
}

// cleanDescription removes HTML tags and cleans up Discogs profile text
func cleanDescription(description string) string {
	// Remove common HTML tags
	description = strings.ReplaceAll(description, "<br>", "\n")
	description = strings.ReplaceAll(description, "<br/>", "\n")
	description = strings.ReplaceAll(description, "<p>", "")
	description = strings.ReplaceAll(description, "</p>", "\n")

	// Remove [a=artist] style tags common in Discogs
	lines := strings.Split(description, "\n")
	cleaned := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "[") {
			cleaned = append(cleaned, line)
		}
	}

	result := strings.Join(cleaned, "\n")
	return strings.TrimSpace(result)
}

// Close gracefully shuts down the client
func (c *DiscogsClient) Close() {
	if c.rateLimiter != nil {
		c.rateLimiter.Stop()
	}
}
