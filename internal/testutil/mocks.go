package testutil

import (
	"context"
	"errors"
	"time"

	"gocommender/internal/db"
	"gocommender/internal/models"
	"gocommender/internal/services"
)

// MockCacheManager implements CacheManager interface for testing
type MockCacheManager struct {
	artists map[string]*models.Artist
	hits    int
	misses  int
	errors  map[string]error // Map of MBID to error for simulating failures
}

func NewMockCacheManager() *MockCacheManager {
	return &MockCacheManager{
		artists: make(map[string]*models.Artist),
		errors:  make(map[string]error),
	}
}

func (m *MockCacheManager) GetOrFetchArtist(mbid string) (*models.Artist, bool, error) {
	if err, exists := m.errors[mbid]; exists {
		return nil, false, err
	}

	if artist, exists := m.artists[mbid]; exists {
		m.hits++
		// Check if expired
		if time.Now().After(artist.CacheExpiry) {
			return artist, true, nil // Return artist but indicate needs fetch
		}
		return artist, false, nil
	}
	m.misses++
	return nil, true, nil // Not found, needs fetch
}

func (m *MockCacheManager) CacheArtist(artist *models.Artist, config db.CacheConfig) error {
	if artist.MBID == "" {
		return errors.New("artist MBID cannot be empty")
	}
	m.artists[artist.MBID] = artist
	return nil
}

func (m *MockCacheManager) RefreshExpiredArtists(limit int) ([]models.Artist, error) {
	var expired []models.Artist
	count := 0
	for _, artist := range m.artists {
		if count >= limit {
			break
		}
		if time.Now().After(artist.CacheExpiry) {
			expired = append(expired, *artist)
			count++
		}
	}
	return expired, nil
}

func (m *MockCacheManager) CleanupExpiredEntries(maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge)
	deleted := 0
	for mbid, artist := range m.artists {
		if artist.CacheExpiry.Before(cutoff) {
			delete(m.artists, mbid)
			deleted++
		}
	}
	return deleted, nil
}

func (m *MockCacheManager) GetCacheStats() (db.CacheStats, error) {
	total := len(m.artists)
	valid := 0
	expired := 0

	for _, artist := range m.artists {
		if time.Now().After(artist.CacheExpiry) {
			expired++
		} else {
			valid++
		}
	}

	return db.CacheStats{
		Total:   total,
		Valid:   valid,
		Expired: expired,
	}, nil
}

func (m *MockCacheManager) UpdateCacheExpiry(mbid string, newExpiry time.Time) error {
	if artist, exists := m.artists[mbid]; exists {
		artist.CacheExpiry = newExpiry
		artist.LastUpdated = time.Now()
		return nil
	}
	return errors.New("artist not found")
}

func (m *MockCacheManager) BulkCacheArtists(artists []models.Artist, config db.CacheConfig) error {
	for _, artist := range artists {
		if err := m.CacheArtist(&artist, config); err != nil {
			return err
		}
	}
	return nil
}

// SetError configures the mock to return an error for a specific MBID
func (m *MockCacheManager) SetError(mbid string, err error) {
	m.errors[mbid] = err
}

// GetStats returns mock statistics
func (m *MockCacheManager) GetStats() (int, int) {
	return m.hits, m.misses
}

// MockPlexClient implements Plex operations for testing
type MockPlexClient struct {
	playlists  []models.PlexPlaylist
	tracks     map[string][]models.PlexTrack
	artists    []string
	shouldErr  bool
	serverInfo map[string]string
}

func NewMockPlexClient() *MockPlexClient {
	return &MockPlexClient{
		playlists: TestPlexPlaylists(),
		tracks: map[string][]models.PlexTrack{
			"My Favorites": TestPlexTracks(),
		},
		artists: []string{"The Beatles", "Pink Floyd", "Led Zeppelin"},
		serverInfo: map[string]string{
			"name":     "Mock Plex Server",
			"version":  "1.0.0",
			"platform": "test",
		},
	}
}

func (m *MockPlexClient) GetPlaylists() ([]models.PlexPlaylist, error) {
	if m.shouldErr {
		return nil, errors.New("mock plex error")
	}
	return m.playlists, nil
}

func (m *MockPlexClient) GetPlaylistTracks(name string) ([]models.PlexTrack, error) {
	if m.shouldErr {
		return nil, errors.New("mock plex error")
	}

	if tracks, exists := m.tracks[name]; exists {
		return tracks, nil
	}
	return []models.PlexTrack{}, nil
}

func (m *MockPlexClient) GetHighRatedTracks(name string, minRating int) ([]models.PlexTrack, error) {
	tracks, err := m.GetPlaylistTracks(name)
	if err != nil {
		return nil, err
	}

	filtered := make([]models.PlexTrack, 0)
	for _, track := range tracks {
		if track.Rating >= minRating {
			filtered = append(filtered, track)
		}
	}
	return filtered, nil
}

func (m *MockPlexClient) GetAllArtists() ([]string, error) {
	if m.shouldErr {
		return nil, errors.New("mock plex error")
	}
	return m.artists, nil
}

func (m *MockPlexClient) TestConnection() error {
	if m.shouldErr {
		return errors.New("mock connection error")
	}
	return nil
}

func (m *MockPlexClient) GetServerInfo() (map[string]string, error) {
	if m.shouldErr {
		return nil, errors.New("mock server info error")
	}
	return m.serverInfo, nil
}

// SetError configures the mock to return errors
func (m *MockPlexClient) SetError(shouldErr bool) {
	m.shouldErr = shouldErr
}

// AddPlaylist adds a playlist to the mock
func (m *MockPlexClient) AddPlaylist(playlist models.PlexPlaylist, tracks []models.PlexTrack) {
	m.playlists = append(m.playlists, playlist)
	m.tracks[playlist.Name] = tracks
}

// MockEnrichmentService implements enrichment operations for testing
type MockEnrichmentService struct {
	enrichedArtists map[string]*models.Artist
	shouldErr       bool
	enrichmentDelay time.Duration
}

func NewMockEnrichmentService() *MockEnrichmentService {
	return &MockEnrichmentService{
		enrichedArtists: make(map[string]*models.Artist),
	}
}

func (m *MockEnrichmentService) EnrichArtist(ctx context.Context, artist *models.Artist, options services.EnrichmentOptions) (*models.Artist, *services.EnrichmentStats, error) {
	if m.shouldErr {
		return nil, nil, errors.New("mock enrichment error")
	}

	// Simulate processing delay
	if m.enrichmentDelay > 0 {
		time.Sleep(m.enrichmentDelay)
	}

	// Return pre-configured enriched artist or enrich the input
	if enriched, exists := m.enrichedArtists[artist.MBID]; exists {
		return enriched, &services.EnrichmentStats{
			CacheHits:    1,
			CacheMisses:  0,
			APICallsMade: 0,
		}, nil
	}

	// Mock enrichment: add some basic data
	enriched := *artist
	if enriched.Description == "" {
		enriched.Description = "Mock enriched description"
	}
	if len(enriched.Genres) == 0 {
		enriched.Genres = models.Genres{"mock", "genre"}
	}
	if enriched.ImageURL == "" {
		enriched.ImageURL = "https://mock.example.com/image.jpg"
	}

	enriched.Verified = models.VerificationMap{
		"musicbrainz": true,
		"discogs":     true,
		"lastfm":      true,
	}

	return &enriched, &services.EnrichmentStats{
		CacheHits:    0,
		CacheMisses:  1,
		APICallsMade: 3,
	}, nil
}

// SetEnrichedArtist pre-configures an enriched artist response
func (m *MockEnrichmentService) SetEnrichedArtist(mbid string, artist *models.Artist) {
	m.enrichedArtists[mbid] = artist
}

// SetError configures the mock to return errors
func (m *MockEnrichmentService) SetError(shouldErr bool) {
	m.shouldErr = shouldErr
}

// SetDelay configures processing delay for testing timeouts
func (m *MockEnrichmentService) SetDelay(delay time.Duration) {
	m.enrichmentDelay = delay
}

// MockOpenAIClient implements OpenAI operations for testing
type MockOpenAIClient struct {
	responses []models.Artist
	shouldErr bool
	delay     time.Duration
}

func NewMockOpenAIClient() *MockOpenAIClient {
	return &MockOpenAIClient{
		responses: []models.Artist{
			*TestArtist(),
			*TestArtistMinimal(),
		},
	}
}

func (m *MockOpenAIClient) GenerateRecommendations(ctx context.Context, seedArtists []models.Artist, excludeArtists []string, maxResults int) ([]models.Artist, error) {
	if m.shouldErr {
		return nil, errors.New("mock openai error")
	}

	// Simulate processing delay
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	// Return up to maxResults recommendations
	results := make([]models.Artist, 0, maxResults)
	for i := 0; i < maxResults && i < len(m.responses); i++ {
		results = append(results, m.responses[i])
	}

	return results, nil
}

// SetResponses configures the mock recommendations
func (m *MockOpenAIClient) SetResponses(artists []models.Artist) {
	m.responses = artists
}

// SetError configures the mock to return errors
func (m *MockOpenAIClient) SetError(shouldErr bool) {
	m.shouldErr = shouldErr
}

// SetDelay configures processing delay
func (m *MockOpenAIClient) SetDelay(delay time.Duration) {
	m.delay = delay
}

// MockRecommendationService implements the full recommendation workflow for testing
type MockRecommendationService struct {
	result    *services.RecommendationResult
	shouldErr bool
}

func NewMockRecommendationService() *MockRecommendationService {
	return &MockRecommendationService{
		result: &services.RecommendationResult{
			Response: TestRecommendResponse(),
			Stats: &services.RecommendationStats{
				StartTime:        time.Now().Add(-5 * time.Second),
				EndTime:          time.Now(),
				Duration:         5 * time.Second,
				SeedTrackCount:   10,
				KnownArtistCount: 5,
				LLMSuggestions:   8,
				FilteredCount:    3,
				EnrichedCount:    3,
				CacheHits:        2,
				CacheMisses:      1,
				APICallsMade:     5,
			},
		},
	}
}

func (m *MockRecommendationService) GenerateRecommendations(ctx context.Context, request models.RecommendRequest) (*services.RecommendationResult, error) {
	if m.shouldErr {
		return nil, errors.New("mock recommendation service error")
	}
	return m.result, nil
}

// SetResult configures the mock result
func (m *MockRecommendationService) SetResult(result *services.RecommendationResult) {
	m.result = result
}

// SetError configures the mock to return errors
func (m *MockRecommendationService) SetError(shouldErr bool) {
	m.shouldErr = shouldErr
}
