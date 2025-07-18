package services

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gocommender/internal/models"
)

// PlexClient handles Plex Media Server API interactions
type PlexClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// PlexMediaContainer represents the root XML response from Plex
type PlexMediaContainer struct {
	XMLName   xml.Name          `xml:"MediaContainer"`
	Size      int               `xml:"size,attr"`
	Tracks    []PlexTrackXML    `xml:"Track"`
	Artists   []PlexArtistXML   `xml:"Directory"`
	Playlists []PlexPlaylistXML `xml:"Playlist"`
}

// PlexTrackXML represents a track from Plex API
type PlexTrackXML struct {
	XMLName          xml.Name `xml:"Track"`
	RatingKey        string   `xml:"ratingKey,attr"`
	Title            string   `xml:"title,attr"`
	GrandparentTitle string   `xml:"grandparentTitle,attr"` // Artist
	ParentTitle      string   `xml:"parentTitle,attr"`      // Album
	Year             int      `xml:"year,attr"`
	UserRating       int      `xml:"userRating,attr"` // 1-10 scale
	ViewCount        int      `xml:"viewCount,attr"`
	LastViewedAt     int64    `xml:"lastViewedAt,attr"`
	Duration         int      `xml:"duration,attr"` // milliseconds
}

// PlexArtistXML represents an artist from Plex API
type PlexArtistXML struct {
	XMLName   xml.Name `xml:"Directory"`
	RatingKey string   `xml:"ratingKey,attr"`
	Title     string   `xml:"title,attr"`
	Type      string   `xml:"type,attr"`
}

// PlexPlaylistXML represents a playlist from Plex API
type PlexPlaylistXML struct {
	XMLName      xml.Name `xml:"Playlist"`
	RatingKey    string   `xml:"ratingKey,attr"`
	Title        string   `xml:"title,attr"`
	Type         string   `xml:"type,attr"`
	Smart        bool     `xml:"smart,attr"`
	PlaylistType string   `xml:"playlistType,attr"`
	LeafCount    int      `xml:"leafCount,attr"`
	Duration     int      `xml:"duration,attr"`
}

// PlexError represents specific Plex API errors
type PlexError struct {
	StatusCode int
	Message    string
	URL        string
}

func (e *PlexError) Error() string {
	return fmt.Sprintf("Plex API error %d: %s (URL: %s)",
		e.StatusCode, e.Message, e.URL)
}

// IsNotFound checks if error indicates resource not found
func IsNotFound(err error) bool {
	if plexErr, ok := err.(*PlexError); ok {
		return plexErr.StatusCode == http.StatusNotFound
	}
	return false
}

// IsUnauthorized checks if error indicates authentication failure
func IsUnauthorized(err error) bool {
	if plexErr, ok := err.(*PlexError); ok {
		return plexErr.StatusCode == http.StatusUnauthorized
	}
	return false
}

// NewPlexClient creates a new Plex API client
func NewPlexClient(baseURL, token string) *PlexClient {
	return &PlexClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TestConnection verifies the Plex server is accessible
func (c *PlexClient) TestConnection() error {
	url := fmt.Sprintf("%s/?X-Plex-Token=%s", c.baseURL, c.token)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to Plex server: %w", err)
	}
	defer resp.Body.Close()

	if err := c.validatePlexResponse(resp, url); err != nil {
		return err
	}

	return nil
}

// GetServerInfo retrieves basic server information for validation
func (c *PlexClient) GetServerInfo() (map[string]string, error) {
	url := fmt.Sprintf("%s/?X-Plex-Token=%s", c.baseURL, c.token)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}
	defer resp.Body.Close()

	if err := c.validatePlexResponse(resp, url); err != nil {
		return nil, err
	}

	var container struct {
		XMLName      xml.Name `xml:"MediaContainer"`
		FriendlyName string   `xml:"friendlyName,attr"`
		Version      string   `xml:"version,attr"`
		Platform     string   `xml:"platform,attr"`
		Size         int      `xml:"size,attr"`
	}

	if err := xml.NewDecoder(resp.Body).Decode(&container); err != nil {
		return nil, fmt.Errorf("failed to decode server info: %w", err)
	}

	return map[string]string{
		"name":     container.FriendlyName,
		"version":  container.Version,
		"platform": container.Platform,
	}, nil
}

// GetPlaylists retrieves all playlists from Plex server
func (c *PlexClient) GetPlaylists() ([]models.PlexPlaylist, error) {
	url := fmt.Sprintf("%s/playlists?X-Plex-Token=%s", c.baseURL, c.token)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlists: %w", err)
	}
	defer resp.Body.Close()

	if err := c.validatePlexResponse(resp, url); err != nil {
		return nil, err
	}

	var container PlexMediaContainer
	if err := xml.NewDecoder(resp.Body).Decode(&container); err != nil {
		return nil, fmt.Errorf("failed to decode playlists: %w", err)
	}

	playlists := make([]models.PlexPlaylist, 0, len(container.Playlists))
	for _, p := range container.Playlists {
		playlists = append(playlists, models.PlexPlaylist{
			Name:       p.Title,
			Type:       p.Type,
			Smart:      p.Smart,
			TrackCount: p.LeafCount,
			Duration:   p.Duration,
		})
	}

	return playlists, nil
}

// GetPlaylistTracks retrieves tracks from a specific playlist
func (c *PlexClient) GetPlaylistTracks(playlistName string) ([]models.PlexTrack, error) {
	// First, find the playlist by name
	playlistKey, err := c.findPlaylistKey(playlistName)
	if err != nil {
		return nil, fmt.Errorf("failed to find playlist: %w", err)
	}

	// Get tracks from the playlist
	url := fmt.Sprintf("%s/playlists/%s/items?X-Plex-Token=%s",
		c.baseURL, playlistKey, c.token)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist tracks: %w", err)
	}
	defer resp.Body.Close()

	if err := c.validatePlexResponse(resp, url); err != nil {
		return nil, err
	}

	var container PlexMediaContainer
	if err := xml.NewDecoder(resp.Body).Decode(&container); err != nil {
		return nil, fmt.Errorf("failed to decode tracks: %w", err)
	}

	tracks := make([]models.PlexTrack, 0, len(container.Tracks))
	for _, t := range container.Tracks {
		track := models.PlexTrack{
			Title:     t.Title,
			Artist:    t.GrandparentTitle,
			Album:     t.ParentTitle,
			Year:      t.Year,
			Rating:    t.UserRating,
			PlayCount: t.ViewCount,
		}

		// Convert Unix timestamp to time.Time
		if t.LastViewedAt > 0 {
			track.LastPlayed = time.Unix(t.LastViewedAt, 0)
		}

		tracks = append(tracks, track)
	}

	return tracks, nil
}

// GetAllArtists retrieves all artists from the music library
func (c *PlexClient) GetAllArtists() ([]string, error) {
	// First, find the music library section
	musicSectionKey, err := c.findMusicSectionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to find music section: %w", err)
	}

	// Get all artists from the music section
	url := fmt.Sprintf("%s/library/sections/%s/all?type=8&X-Plex-Token=%s",
		c.baseURL, musicSectionKey, c.token)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get artists: %w", err)
	}
	defer resp.Body.Close()

	if err := c.validatePlexResponse(resp, url); err != nil {
		return nil, err
	}

	var container PlexMediaContainer
	if err := xml.NewDecoder(resp.Body).Decode(&container); err != nil {
		return nil, fmt.Errorf("failed to decode artists: %w", err)
	}

	artists := make([]string, 0, len(container.Artists))
	for _, a := range container.Artists {
		if a.Title != "" {
			artists = append(artists, a.Title)
		}
	}

	return artists, nil
}

// GetHighRatedTracks retrieves tracks with high user ratings from a playlist
func (c *PlexClient) GetHighRatedTracks(playlistName string, minRating int) ([]models.PlexTrack, error) {
	if minRating < 1 || minRating > 10 {
		minRating = 7 // Default to 7+ rating
	}

	tracks, err := c.GetPlaylistTracks(playlistName)
	if err != nil {
		return nil, err
	}

	var highRated []models.PlexTrack
	for _, track := range tracks {
		if track.Rating >= minRating {
			highRated = append(highRated, track)
		}
	}

	return highRated, nil
}

// GetArtistsByGenre retrieves artists filtered by genre
func (c *PlexClient) GetArtistsByGenre(genre string) ([]string, error) {
	musicSectionKey, err := c.findMusicSectionKey()
	if err != nil {
		return nil, err
	}

	// Search for artists by genre
	genreQuery := url.QueryEscape(genre)
	url := fmt.Sprintf("%s/library/sections/%s/all?type=8&genre=%s&X-Plex-Token=%s",
		c.baseURL, musicSectionKey, genreQuery, c.token)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get artists by genre: %w", err)
	}
	defer resp.Body.Close()

	if err := c.validatePlexResponse(resp, url); err != nil {
		return nil, err
	}

	var container PlexMediaContainer
	if err := xml.NewDecoder(resp.Body).Decode(&container); err != nil {
		return nil, fmt.Errorf("failed to decode artists: %w", err)
	}

	artists := make([]string, 0, len(container.Artists))
	for _, a := range container.Artists {
		if a.Title != "" {
			artists = append(artists, a.Title)
		}
	}

	return artists, nil
}

// findPlaylistKey searches for a playlist by name and returns its key
func (c *PlexClient) findPlaylistKey(name string) (string, error) {
	url := fmt.Sprintf("%s/playlists?X-Plex-Token=%s", c.baseURL, c.token)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to search playlists: %w", err)
	}
	defer resp.Body.Close()

	if err := c.validatePlexResponse(resp, url); err != nil {
		return "", err
	}

	var container PlexMediaContainer
	if err := xml.NewDecoder(resp.Body).Decode(&container); err != nil {
		return "", fmt.Errorf("failed to decode playlists: %w", err)
	}

	for _, playlist := range container.Playlists {
		if strings.EqualFold(playlist.Title, name) {
			return playlist.RatingKey, nil
		}
	}

	return "", fmt.Errorf("playlist '%s' not found", name)
}

// findMusicSectionKey finds the key for the music library section
func (c *PlexClient) findMusicSectionKey() (string, error) {
	url := fmt.Sprintf("%s/library/sections?X-Plex-Token=%s", c.baseURL, c.token)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get library sections: %w", err)
	}
	defer resp.Body.Close()

	if err := c.validatePlexResponse(resp, url); err != nil {
		return "", err
	}

	var container struct {
		XMLName     xml.Name `xml:"MediaContainer"`
		Directories []struct {
			Key   string `xml:"key,attr"`
			Type  string `xml:"type,attr"`
			Title string `xml:"title,attr"`
		} `xml:"Directory"`
	}

	if err := xml.NewDecoder(resp.Body).Decode(&container); err != nil {
		return "", fmt.Errorf("failed to decode sections: %w", err)
	}

	for _, dir := range container.Directories {
		if dir.Type == "artist" {
			return dir.Key, nil
		}
	}

	return "", fmt.Errorf("music library section not found")
}

// validatePlexResponse checks response status and creates appropriate errors
func (c *PlexClient) validatePlexResponse(resp *http.Response, requestURL string) error {
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return &PlexError{
			StatusCode: resp.StatusCode,
			Message:    "Resource not found",
			URL:        requestURL,
		}
	case http.StatusUnauthorized:
		return &PlexError{
			StatusCode: resp.StatusCode,
			Message:    "Invalid or missing Plex token",
			URL:        requestURL,
		}
	case http.StatusForbidden:
		return &PlexError{
			StatusCode: resp.StatusCode,
			Message:    "Access forbidden - check server permissions",
			URL:        requestURL,
		}
	default:
		return &PlexError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("Unexpected status code: %d", resp.StatusCode),
			URL:        requestURL,
		}
	}
}
