# 05 - External APIs Integration

## Overview

Implement Discogs and Last.fm API clients for additional metadata enrichment. These APIs provide rich descriptions, images, and social data to complement MusicBrainz information.

## Steps

### 1. Create Discogs Client

- [ ] Define `internal/services/discogs.go` with client structure

```go
package services

import (
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "time"
)

// DiscogsClient handles Discogs API interactions
type DiscogsClient struct {
    baseURL     string
    token       string
    httpClient  *http.Client
    userAgent   string
    rateLimiter *time.Ticker
}

// DiscogsArtist represents artist data from Discogs API
type DiscogsArtist struct {
    ID           int                    `json:"id"`
    Name         string                 `json:"name"`
    Profile      string                 `json:"profile"`
    Images       []DiscogsImage         `json:"images"`
    URLs         []string               `json:"urls"`
    Members      []DiscogsMember        `json:"members"`
    Releases     []DiscogsRelease       `json:"releases"`
    DataQuality  string                 `json:"data_quality"`
}

type DiscogsImage struct {
    Type        string `json:"type"`
    URI         string `json:"uri"`
    URI150      string `json:"uri150"`
    Width       int    `json:"width"`
    Height      int    `json:"height"`
}

type DiscogsMember struct {
    Name string `json:"name"`
    ID   int    `json:"id"`
}

type DiscogsRelease struct {
    ID       int    `json:"id"`
    Title    string `json:"title"`
    Year     int    `json:"year"`
    Type     string `json:"type"`
    Role     string `json:"role"`
}

// NewDiscogsClient creates a new Discogs API client
func NewDiscogsClient(token string) *DiscogsClient {
    return &DiscogsClient{
        baseURL: "https://api.discogs.com",
        token:   token,
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
        userAgent: "GoCommender/1.0",
        rateLimiter: time.NewTicker(1100 * time.Millisecond), // ~50 requests/minute
    }
}
```

### 2. Implement Discogs Methods

- [ ] Add search and enrichment functionality

```go
// SearchArtist searches for artists by name in Discogs
func (c *DiscogsClient) SearchArtist(name string) (*DiscogsArtist, error) {
    if c.token == "" {
        return nil, fmt.Errorf("discogs token not configured")
    }

    <-c.rateLimiter.C // Rate limiting

    query := url.QueryEscape(name)
    url := fmt.Sprintf("%s/database/search?q=%s&type=artist&token=%s",
                      c.baseURL, query, c.token)

    req, err := http.NewRequest("GET", url, nil)
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
        Results []struct {
            ID   int    `json:"id"`
            Type string `json:"type"`
        } `json:"results"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    if len(searchResult.Results) == 0 {
        return nil, fmt.Errorf("no artists found for '%s'", name)
    }

    // Get detailed artist information
    return c.GetArtistByID(searchResult.Results[0].ID)
}

// GetArtistByID fetches detailed artist information by Discogs ID
func (c *DiscogsClient) GetArtistByID(id int) (*DiscogsArtist, error) {
    <-c.rateLimiter.C // Rate limiting

    url := fmt.Sprintf("%s/artists/%d?token=%s", c.baseURL, id, c.token)

    req, err := http.NewRequest("GET", url, nil)
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

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
    }

    var artist DiscogsArtist
    if err := json.NewDecoder(resp.Body).Decode(&artist); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &artist, nil
}
```

### 3. Create Last.fm Client

- [ ] Define `internal/services/lastfm.go` with client methods

```go
package services

import (
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "time"
)

// LastFMClient handles Last.fm API interactions
type LastFMClient struct {
    baseURL     string
    apiKey      string
    httpClient  *http.Client
    rateLimiter *time.Ticker
}

// LastFMArtist represents artist data from Last.fm API
type LastFMArtist struct {
    Name       string           `json:"name"`
    MBID       string           `json:"mbid"`
    Bio        LastFMBio        `json:"bio"`
    Images     []LastFMImage    `json:"image"`
    Tags       LastFMTags       `json:"tags"`
    Stats      LastFMStats      `json:"stats"`
    Similar    LastFMSimilar    `json:"similar"`
}

type LastFMBio struct {
    Summary   string `json:"summary"`
    Content   string `json:"content"`
    Published string `json:"published"`
}

type LastFMImage struct {
    Text string `json:"#text"`
    Size string `json:"size"`
}

type LastFMTags struct {
    Tag []LastFMTag `json:"tag"`
}

type LastFMTag struct {
    Name string `json:"name"`
    URL  string `json:"url"`
}

type LastFMStats struct {
    Listeners string `json:"listeners"`
    PlayCount string `json:"playcount"`
}

type LastFMSimilar struct {
    Artist []LastFMSimilarArtist `json:"artist"`
}

type LastFMSimilarArtist struct {
    Name  string        `json:"name"`
    MBID  string        `json:"mbid"`
    Image []LastFMImage `json:"image"`
}

// NewLastFMClient creates a new Last.fm API client
func NewLastFMClient(apiKey string) *LastFMClient {
    return &LastFMClient{
        baseURL: "http://ws.audioscrobbler.com/2.0",
        apiKey:  apiKey,
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
        rateLimiter: time.NewTicker(200 * time.Millisecond), // 5 requests/second
    }
}

// GetArtistInfo fetches artist information from Last.fm
func (c *LastFMClient) GetArtistInfo(name string) (*LastFMArtist, error) {
    if c.apiKey == "" {
        return nil, fmt.Errorf("last.fm API key not configured")
    }

    <-c.rateLimiter.C // Rate limiting

    params := url.Values{}
    params.Set("method", "artist.getinfo")
    params.Set("artist", name)
    params.Set("api_key", c.apiKey)
    params.Set("format", "json")

    url := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

    resp, err := c.httpClient.Get(url)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
    }

    var result struct {
        Artist LastFMArtist `json:"artist"`
        Error  int          `json:"error"`
        Message string      `json:"message"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    if result.Error != 0 {
        return nil, fmt.Errorf("last.fm API error: %s", result.Message)
    }

    return &result.Artist, nil
}
```

### 4. Data Enrichment Functions

- [ ] Create enrichment logic in `internal/services/enrichment.go`

```go
package services

import (
    "fmt"
    "strings"
    "time"

    "gocommender/internal/models"
)

// ArtistEnricher combines data from multiple sources
type ArtistEnricher struct {
    musicbrainz *MusicBrainzClient
    discogs     *DiscogsClient
    lastfm      *LastFMClient
}

// NewArtistEnricher creates a new artist enricher
func NewArtistEnricher(discogsToken, lastfmKey string) *ArtistEnricher {
    return &ArtistEnricher{
        musicbrainz: NewMusicBrainzClient(),
        discogs:     NewDiscogsClient(discogsToken),
        lastfm:      NewLastFMClient(lastfmKey),
    }
}

// EnrichArtist fetches and combines data from all available sources
func (e *ArtistEnricher) EnrichArtist(name string) (*models.Artist, error) {
    artist := &models.Artist{
        Name:         name,
        Verified:     make(models.VerificationMap),
        Genres:       make(models.Genres, 0),
        ExternalURLs: models.ExternalURLs{},
        LastUpdated:  time.Now(),
        CacheExpiry:  time.Now().Add(30 * 24 * time.Hour),
    }

    // Start with MusicBrainz as primary source
    if mbArtist, err := e.musicbrainz.SearchArtist(name); err == nil {
        enrichFromMusicBrainz(artist, mbArtist)
        artist.Verified["musicbrainz"] = true
    } else {
        artist.Verified["musicbrainz"] = false
    }

    // Enrich with Discogs data
    if e.discogs.token != "" {
        if discArtist, err := e.discogs.SearchArtist(name); err == nil {
            enrichFromDiscogs(artist, discArtist)
            artist.Verified["discogs"] = true
        } else {
            artist.Verified["discogs"] = false
        }
    }

    // Enrich with Last.fm data
    if e.lastfm.apiKey != "" {
        if lfmArtist, err := e.lastfm.GetArtistInfo(name); err == nil {
            enrichFromLastFM(artist, lfmArtist)
            artist.Verified["lastfm"] = true
        } else {
            artist.Verified["lastfm"] = false
        }
    }

    return artist, nil
}

func enrichFromMusicBrainz(artist *models.Artist, mb *MusicBrainzArtist) {
    if artist.MBID == "" {
        artist.MBID = mb.ID
    }
    artist.AlbumCount = len(mb.Releases)
    if mb.Country != "" {
        artist.Country = mb.Country
    }
    if mb.LifeSpan != nil {
        artist.YearsActive = formatYearsActive(mb.LifeSpan.Begin, mb.LifeSpan.End)
    }
    artist.ExternalURLs.MusicBrainz = fmt.Sprintf("https://musicbrainz.org/artist/%s", mb.ID)

    // Extract genres from tags
    for _, tag := range mb.Tags {
        if tag.Count > 5 {
            artist.Genres = append(artist.Genres, tag.Name)
        }
    }
}

func enrichFromDiscogs(artist *models.Artist, discogs *DiscogsArtist) {
    if artist.Description == "" && discogs.Profile != "" {
        artist.Description = cleanDescription(discogs.Profile)
    }

    // Get the best image
    if artist.ImageURL == "" && len(discogs.Images) > 0 {
        artist.ImageURL = getBestImage(discogs.Images)
    }

    artist.ExternalURLs.Discogs = fmt.Sprintf("https://discogs.com/artist/%d", discogs.ID)

    // Update album count if higher
    albumCount := countAlbums(discogs.Releases)
    if albumCount > artist.AlbumCount {
        artist.AlbumCount = albumCount
    }
}

func enrichFromLastFM(artist *models.Artist, lastfm *LastFMArtist) {
    if artist.Description == "" && lastfm.Bio.Summary != "" {
        artist.Description = cleanDescription(lastfm.Bio.Summary)
    }

    // Get image if not already set
    if artist.ImageURL == "" {
        for _, img := range lastfm.Images {
            if img.Size == "large" || img.Size == "extralarge" {
                artist.ImageURL = img.Text
                break
            }
        }
    }

    // Add genres from tags
    for _, tag := range lastfm.Tags.Tag {
        artist.Genres = append(artist.Genres, tag.Name)
    }

    artist.ExternalURLs.LastFM = fmt.Sprintf("https://last.fm/music/%s",
                                           url.QueryEscape(lastfm.Name))
}

func cleanDescription(desc string) string {
    // Remove HTML tags and clean up text
    desc = strings.ReplaceAll(desc, "<a href", " <a href")
    desc = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(desc, "")
    desc = strings.TrimSpace(desc)

    // Limit length to ~500 characters
    if len(desc) > 500 {
        desc = desc[:497] + "..."
    }

    return desc
}

func getBestImage(images []DiscogsImage) string {
    // Prefer primary images, then largest
    for _, img := range images {
        if img.Type == "primary" {
            return img.URI
        }
    }

    // Return first available image
    if len(images) > 0 {
        return images[0].URI
    }

    return ""
}

func countAlbums(releases []DiscogsRelease) int {
    count := 0
    for _, release := range releases {
        if release.Type == "master" || release.Role == "Main" {
            count++
        }
    }
    return count
}
```

## Verification Steps

- [ ] **Discogs Integration**:

   ```bash
   DISCOGS_TOKEN=your_token go test ./internal/services -run TestDiscogsClient
   ```

- [ ] **Last.fm Integration**:

   ```bash
   LASTFM_API_KEY=your_key go test ./internal/services -run TestLastFMClient
   ```

- [ ] **Data Enrichment**:

   ```bash
   # Test combined enrichment
   go run ./cmd/test-enrichment -artist "Radiohead"
   # Should return rich artist data from all sources
   ```

- [ ] **Graceful Degradation**:

   ```bash
   # Test without API keys
   go run ./cmd/test-enrichment -artist "Radiohead"
   # Should work with MusicBrainz only
   ```

- [ ] **Rate Limiting**:

   ```bash
   go run ./cmd/stress-test-apis
   # Should respect all API rate limits
   ```

## Dependencies

- Previous: `04_musicbrainz_integration.md` (MusicBrainz client)
- New imports: `regexp`, `net/url`
- Environment: `DISCOGS_TOKEN`, `LASTFM_API_KEY` (optional)

## Next Steps

Proceed to `06_caching_layer.md` to implement SQLite caching for enriched artist data.

## Notes

- Both Discogs and Last.fm APIs are optional - graceful degradation when tokens missing
- Rate limiting prevents API abuse across all services
- Data prioritization: MusicBrainz > Discogs > Last.fm for conflicts
- Image selection prefers primary/large images
- Description cleaning removes HTML and limits length
- Album counting logic handles different release types
- All external URLs are tracked for future reference
- Verification map tracks which services successfully returned data
