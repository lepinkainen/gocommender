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

// MusicBrainzClient handles MusicBrainz API interactions
type MusicBrainzClient struct {
	baseURL     string
	httpClient  *http.Client
	userAgent   string
	rateLimiter *time.Ticker
}

// MusicBrainzArtist represents artist data from MusicBrainz API
type MusicBrainzArtist struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	Country   string               `json:"country"`
	BeginArea *MusicBrainzArea     `json:"begin-area"`
	LifeSpan  *MusicBrainzLifeSpan `json:"life-span"`
	Releases  []MusicBrainzRelease `json:"releases"`
	Tags      []MusicBrainzTag     `json:"tags"`
	Genres    []MusicBrainzGenre   `json:"genres"`
}

type MusicBrainzArea struct {
	Name string `json:"name"`
}

type MusicBrainzLifeSpan struct {
	Begin string `json:"begin"`
	End   string `json:"end"`
}

type MusicBrainzRelease struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Date  string `json:"date"`
}

type MusicBrainzTag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type MusicBrainzGenre struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// NewMusicBrainzClient creates a new MusicBrainz API client
func NewMusicBrainzClient() *MusicBrainzClient {
	return &MusicBrainzClient{
		baseURL: "https://musicbrainz.org/ws/2",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		userAgent:   "GoCommender/1.0 (https://github.com/lepinkainen/gocommender)",
		rateLimiter: time.NewTicker(1100 * time.Millisecond), // ~50 requests/minute
	}
}

// SearchArtist searches for artists by name and returns the best match with MBID
func (c *MusicBrainzClient) SearchArtist(name string) (*MusicBrainzArtist, error) {
	<-c.rateLimiter.C // Rate limiting

	query := url.QueryEscape(fmt.Sprintf(`artist:"%s"`, name))
	urlStr := fmt.Sprintf("%s/artist?query=%s&fmt=json&limit=1", c.baseURL, query)

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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var searchResult struct {
		Artists []MusicBrainzArtist `json:"artists"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(searchResult.Artists) == 0 {
		return nil, fmt.Errorf("no artists found for '%s'", name)
	}

	return &searchResult.Artists[0], nil
}

// GetArtistByMBID fetches detailed artist information by MusicBrainz ID
func (c *MusicBrainzClient) GetArtistByMBID(mbid string) (*MusicBrainzArtist, error) {
	<-c.rateLimiter.C // Rate limiting

	urlStr := fmt.Sprintf("%s/artist/%s?fmt=json&inc=releases+tags+genres", c.baseURL, mbid)

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
		return nil, fmt.Errorf("artist with MBID %s not found", mbid)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var artist MusicBrainzArtist
	if err := json.NewDecoder(resp.Body).Decode(&artist); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &artist, nil
}

// ToArtistModel converts MusicBrainz data to internal Artist model
func (mb *MusicBrainzArtist) ToArtistModel() *models.Artist {
	artist := &models.Artist{
		MBID:       mb.ID,
		Name:       mb.Name,
		AlbumCount: len(mb.Releases),
		Country:    mb.Country,
		Verified:   models.VerificationMap{"musicbrainz": true},
		ExternalURLs: models.ExternalURLs{
			MusicBrainz: fmt.Sprintf("https://musicbrainz.org/artist/%s", mb.ID),
		},
		LastUpdated: time.Now(),
		CacheExpiry: time.Now().Add(30 * 24 * time.Hour), // 30 days
	}

	// Extract years active from life span
	if mb.LifeSpan != nil {
		artist.YearsActive = formatYearsActive(mb.LifeSpan.Begin, mb.LifeSpan.End)
	}

	// Extract genres from tags and genres
	genres := make([]string, 0)
	for _, tag := range mb.Tags {
		if tag.Count > 5 { // Only include popular tags
			genres = append(genres, tag.Name)
		}
	}
	for _, genre := range mb.Genres {
		genres = append(genres, genre.Name)
	}
	artist.Genres = removeDuplicates(genres)

	return artist
}

func formatYearsActive(begin, end string) string {
	if begin == "" {
		return ""
	}

	// Extract year from date string (YYYY-MM-DD format)
	beginYear := extractYear(begin)
	if end == "" {
		return fmt.Sprintf("%s-present", beginYear)
	}

	endYear := extractYear(end)
	if beginYear == endYear {
		return beginYear
	}

	return fmt.Sprintf("%s-%s", beginYear, endYear)
}

func extractYear(dateStr string) string {
	if len(dateStr) >= 4 {
		return dateStr[:4]
	}
	return dateStr
}

func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// Close gracefully shuts down the client
func (c *MusicBrainzClient) Close() {
	if c.rateLimiter != nil {
		c.rateLimiter.Stop()
	}
}

// IsRateLimited checks if error is due to rate limiting
func IsRateLimited(err error) bool {
	return strings.Contains(err.Error(), "503") ||
		strings.Contains(err.Error(), "rate limit")
}

// RetryWithBackoff retries a function with exponential backoff
func RetryWithBackoff(fn func() error, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		if !IsRateLimited(err) {
			return err // Don't retry non-rate-limit errors
		}

		// Exponential backoff: 2^i seconds
		time.Sleep(time.Duration(1<<uint(i)) * time.Second)
	}
	return err
}
