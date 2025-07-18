package services

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"gocommender/internal/models"
)

// LastFMClient handles Last.fm API interactions
type LastFMClient struct {
	baseURL     string
	httpClient  *http.Client
	apiKey      string
	secret      string
	rateLimiter *time.Ticker
}

// LastFMArtist represents artist data from Last.fm API
type LastFMArtist struct {
	Name       string        `json:"name"`
	MBID       string        `json:"mbid"`
	URL        string        `json:"url"`
	Image      []LastFMImage `json:"image"`
	Streamable string        `json:"streamable"`
	Bio        LastFMBio     `json:"bio"`
	Tags       LastFMTags    `json:"tags"`
	Similar    LastFMSimilar `json:"similar"`
	Stats      LastFMStats   `json:"stats"`
}

type LastFMImage struct {
	Text string `json:"#text"`
	Size string `json:"size"`
}

type LastFMBio struct {
	Links     LastFMLinks `json:"links"`
	Published string      `json:"published"`
	Summary   string      `json:"summary"`
	Content   string      `json:"content"`
}

type LastFMLinks struct {
	Link LastFMLink `json:"link"`
}

type LastFMLink struct {
	Text string `json:"#text"`
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

type LastFMTags struct {
	Tag []LastFMTag `json:"tag"`
}

type LastFMTag struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type LastFMSimilar struct {
	Artist []LastFMSimilarArtist `json:"artist"`
}

type LastFMSimilarArtist struct {
	Name  string        `json:"name"`
	URL   string        `json:"url"`
	Image []LastFMImage `json:"image"`
}

type LastFMStats struct {
	Listeners string `json:"listeners"`
	Playcount string `json:"playcount"`
}

// LastFMResponse wraps the API response
type LastFMResponse struct {
	Artist LastFMArtist `json:"artist"`
}

// NewLastFMClient creates a new Last.fm API client
func NewLastFMClient(apiKey, secret string) *LastFMClient {
	return &LastFMClient{
		baseURL: "https://ws.audioscrobbler.com/2.0",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		apiKey:      apiKey,
		secret:      secret,
		rateLimiter: time.NewTicker(250 * time.Millisecond), // ~240 requests/minute (5 per second limit)
	}
}

// GetArtistInfo fetches detailed artist information by name
func (c *LastFMClient) GetArtistInfo(name string) (*LastFMArtist, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("last.fm API key not configured")
	}

	<-c.rateLimiter.C // Rate limiting

	params := map[string]string{
		"method":  "artist.getinfo",
		"artist":  name,
		"api_key": c.apiKey,
		"format":  "json",
	}

	urlStr := c.buildURL(params)

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "GoCommender/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("invalid last.fm API key")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var response LastFMResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response.Artist, nil
}

// GetArtistInfoByMBID fetches detailed artist information by MusicBrainz ID
func (c *LastFMClient) GetArtistInfoByMBID(mbid string) (*LastFMArtist, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("last.fm API key not configured")
	}

	<-c.rateLimiter.C // Rate limiting

	params := map[string]string{
		"method":  "artist.getinfo",
		"mbid":    mbid,
		"api_key": c.apiKey,
		"format":  "json",
	}

	urlStr := c.buildURL(params)

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "GoCommender/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("invalid last.fm API key")
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("artist with MBID %s not found", mbid)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var response LastFMResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response.Artist, nil
}

// EnrichArtist enriches an existing Artist model with Last.fm data
func (c *LastFMClient) EnrichArtist(artist *models.Artist) error {
	if c.apiKey == "" {
		// Graceful degradation - just mark as not verified
		if artist.Verified == nil {
			artist.Verified = make(models.VerificationMap)
		}
		artist.Verified["lastfm"] = false
		return nil
	}

	var lastfmArtist *LastFMArtist
	var err error

	// Try by MBID first, then by name
	if artist.MBID != "" {
		lastfmArtist, err = c.GetArtistInfoByMBID(artist.MBID)
	}

	if err != nil || lastfmArtist == nil {
		lastfmArtist, err = c.GetArtistInfo(artist.Name)
	}

	if err != nil {
		// Mark as failed verification but don't return error
		if artist.Verified == nil {
			artist.Verified = make(models.VerificationMap)
		}
		artist.Verified["lastfm"] = false
		return nil
	}

	// Enrich the artist with Last.fm data
	if artist.Verified == nil {
		artist.Verified = make(models.VerificationMap)
	}
	artist.Verified["lastfm"] = true

	// Add description if not already present and available
	if artist.Description == "" && lastfmArtist.Bio.Summary != "" {
		artist.Description = cleanLastFMBio(lastfmArtist.Bio.Summary)
	}

	// Add image URL if not already present and available
	if artist.ImageURL == "" {
		imageURL := getBestLastFMImage(lastfmArtist.Image)
		if imageURL != "" {
			artist.ImageURL = imageURL
		}
	}

	// Add genres from tags
	if len(artist.Genres) == 0 && len(lastfmArtist.Tags.Tag) > 0 {
		genres := make([]string, 0, len(lastfmArtist.Tags.Tag))
		for _, tag := range lastfmArtist.Tags.Tag {
			if tag.Name != "" {
				genres = append(genres, tag.Name)
			}
		}
		if len(genres) > 0 {
			artist.Genres = genres
		}
	}

	// Add Last.fm URL
	if lastfmArtist.URL != "" {
		artist.ExternalURLs.LastFM = lastfmArtist.URL
	}

	return nil
}

// buildURL builds the Last.fm API URL with parameters
func (c *LastFMClient) buildURL(params map[string]string) string {
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	return fmt.Sprintf("%s?%s", c.baseURL, values.Encode())
}

// buildAPISignature builds the API signature for authenticated requests
func (c *LastFMClient) buildAPISignature(params map[string]string) string {
	if c.secret == "" {
		return ""
	}

	// Sort parameters by key
	keys := make([]string, 0, len(params))
	for key := range params {
		if key != "format" { // format is not included in signature
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	// Build signature string
	var sigString strings.Builder
	for _, key := range keys {
		sigString.WriteString(key)
		sigString.WriteString(params[key])
	}
	sigString.WriteString(c.secret)

	// Return MD5 hash
	return fmt.Sprintf("%x", md5.Sum([]byte(sigString.String())))
}

// getBestLastFMImage selects the best image from Last.fm images
func getBestLastFMImage(images []LastFMImage) string {
	if len(images) == 0 {
		return ""
	}

	// Prefer larger images: extralarge > large > medium > small
	sizePriority := map[string]int{
		"extralarge": 4,
		"large":      3,
		"medium":     2,
		"small":      1,
	}

	bestImage := ""
	bestPriority := 0

	for _, img := range images {
		if img.Text != "" {
			priority := sizePriority[img.Size]
			if priority > bestPriority {
				bestImage = img.Text
				bestPriority = priority
			}
		}
	}

	return bestImage
}

// cleanLastFMBio removes HTML tags and Last.fm specific formatting from bio text
func cleanLastFMBio(bio string) string {
	// Remove Last.fm specific suffixes
	if idx := strings.Index(bio, "User-contributed text is available under the Creative Commons"); idx != -1 {
		bio = bio[:idx]
	}
	if idx := strings.Index(bio, "<a href=\"https://www.last.fm/music/"); idx != -1 {
		bio = bio[:idx]
	}

	// Remove HTML tags
	bio = strings.ReplaceAll(bio, "<br>", "\n")
	bio = strings.ReplaceAll(bio, "<br/>", "\n")
	bio = strings.ReplaceAll(bio, "<p>", "")
	bio = strings.ReplaceAll(bio, "</p>", "\n")

	// Remove any remaining HTML tags (simple approach)
	lines := strings.Split(bio, "\n")
	cleaned := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "<") {
			cleaned = append(cleaned, line)
		}
	}

	result := strings.Join(cleaned, "\n")
	return strings.TrimSpace(result)
}

// Close gracefully shuts down the client
func (c *LastFMClient) Close() {
	if c.rateLimiter != nil {
		c.rateLimiter.Stop()
	}
}
