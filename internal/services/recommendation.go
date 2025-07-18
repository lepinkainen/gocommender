package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"gocommender/internal/models"
)

// RecommendationService orchestrates the recommendation workflow
type RecommendationService struct {
	plexClient        *PlexClient
	openaiClient      *OpenAIClient
	enrichmentService *EnrichmentService
}

// RecommendationResult contains the complete recommendation result
type RecommendationResult struct {
	Response *models.RecommendResponse
	Stats    *RecommendationStats
}

// RecommendationStats tracks performance metrics
type RecommendationStats struct {
	StartTime        time.Time     `json:"start_time"`
	EndTime          time.Time     `json:"end_time"`
	Duration         time.Duration `json:"duration"`
	SeedTrackCount   int           `json:"seed_track_count"`
	KnownArtistCount int           `json:"known_artist_count"`
	LLMSuggestions   int           `json:"llm_suggestions"`
	FilteredCount    int           `json:"filtered_count"`
	EnrichedCount    int           `json:"enriched_count"`
	CacheHits        int           `json:"cache_hits"`
	CacheMisses      int           `json:"cache_misses"`
	APICallsMade     int           `json:"api_calls_made"`
	Errors           []string      `json:"errors,omitempty"`
}

// EnrichmentStats tracks enrichment performance
type EnrichmentStats struct {
	CacheHits    int      `json:"cache_hits"`
	CacheMisses  int      `json:"cache_misses"`
	APICallsMade int      `json:"api_calls_made"`
	Errors       []string `json:"errors"`
}

// NewRecommendationService creates a new recommendation service
func NewRecommendationService(plex *PlexClient, openai *OpenAIClient, enrichment *EnrichmentService) *RecommendationService {
	return &RecommendationService{
		plexClient:        plex,
		openaiClient:      openai,
		enrichmentService: enrichment,
	}
}

// GenerateRecommendations performs the complete recommendation workflow
func (s *RecommendationService) GenerateRecommendations(ctx context.Context, request models.RecommendRequest) (*RecommendationResult, error) {
	stats := &RecommendationStats{
		StartTime: time.Now(),
		Errors:    make([]string, 0),
	}

	// Set defaults
	if request.MaxResults <= 0 {
		request.MaxResults = 5
	}

	// Step 1: Get seed tracks from Plex playlist
	log.Printf("Fetching seed tracks from playlist: %s", request.PlaylistName)
	seedTracks, err := s.getHighRatedTracks(request.PlaylistName)
	if err != nil {
		return nil, fmt.Errorf("failed to get seed tracks: %w", err)
	}
	stats.SeedTrackCount = len(seedTracks)

	if len(seedTracks) == 0 {
		return nil, fmt.Errorf("no high-rated tracks found in playlist '%s'", request.PlaylistName)
	}

	// Step 2: Get known artists from Plex library
	log.Printf("Fetching known artists from Plex library")
	knownArtists, err := s.plexClient.GetAllArtists()
	if err != nil {
		stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to get known artists: %v", err))
		// Continue with empty list rather than fail
		knownArtists = make([]string, 0)
	}
	stats.KnownArtistCount = len(knownArtists)

	// Step 3: Generate LLM suggestions
	log.Printf("Generating LLM suggestions for %d seed tracks", len(seedTracks))
	genre := ""
	if request.Genre != nil {
		genre = *request.Genre
	}
	suggestions, err := s.openaiClient.GetArtistRecommendations(
		seedTracks, knownArtists, genre, request.MaxResults*2) // Request more to allow for filtering
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM suggestions: %w", err)
	}
	stats.LLMSuggestions = len(suggestions.Suggestions)
	stats.APICallsMade++

	// Step 4: Filter suggestions against known artists
	log.Printf("Filtering %d suggestions against known artists", len(suggestions.Suggestions))
	filtered := s.openaiClient.FilterKnownArtists(suggestions.Suggestions, knownArtists)
	stats.FilteredCount = len(filtered)

	if len(filtered) == 0 {
		return nil, fmt.Errorf("all LLM suggestions were filtered out as known artists")
	}

	// Limit to requested number
	if len(filtered) > request.MaxResults {
		filtered = filtered[:request.MaxResults]
	}

	// Step 5: Enrich artist data with metadata
	log.Printf("Enriching %d filtered suggestions", len(filtered))
	enrichedArtists, enrichStats := s.enrichArtistSuggestions(ctx, filtered)
	stats.EnrichedCount = len(enrichedArtists)
	stats.CacheHits += enrichStats.CacheHits
	stats.CacheMisses += enrichStats.CacheMisses
	stats.APICallsMade += enrichStats.APICallsMade
	stats.Errors = append(stats.Errors, enrichStats.Errors...)

	// Step 6: Build response
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	response := &models.RecommendResponse{
		Status:      "success",
		RequestID:   generateRequestID(),
		Suggestions: enrichedArtists,
		Metadata: models.RecommendMetadata{
			SeedTrackCount:   stats.SeedTrackCount,
			KnownArtistCount: stats.KnownArtistCount,
			ProcessingTime:   stats.Duration.String(),
			CacheHits:        stats.CacheHits,
			APICallsMade:     stats.APICallsMade,
			GeneratedAt:      time.Now(),
		},
	}

	result := &RecommendationResult{
		Response: response,
		Stats:    stats,
	}

	log.Printf("Recommendation complete: %d suggestions in %v",
		len(enrichedArtists), stats.Duration)

	return result, nil
}

// getHighRatedTracks retrieves high-rated tracks from the specified playlist
func (s *RecommendationService) getHighRatedTracks(playlistName string) ([]models.PlexTrack, error) {
	// Get high-rated tracks (7+ rating)
	tracks, err := s.plexClient.GetHighRatedTracks(playlistName, 7)
	if err != nil {
		return nil, err
	}

	// If no high-rated tracks, try with lower threshold
	if len(tracks) == 0 {
		log.Printf("No tracks with 7+ rating, trying 5+ rating")
		tracks, err = s.plexClient.GetHighRatedTracks(playlistName, 5)
		if err != nil {
			return nil, err
		}
	}

	// If still no tracks, get all tracks from playlist
	if len(tracks) == 0 {
		log.Printf("No rated tracks found, using all tracks from playlist")
		tracks, err = s.plexClient.GetPlaylistTracks(playlistName)
		if err != nil {
			return nil, err
		}
	}

	return tracks, nil
}

// enrichArtistSuggestions enriches artist suggestions with metadata
func (s *RecommendationService) enrichArtistSuggestions(ctx context.Context, artistNames []string) ([]models.Artist, *EnrichmentStats) {
	stats := &EnrichmentStats{
		Errors: make([]string, 0),
	}

	enriched := make([]models.Artist, 0, len(artistNames))

	// Process each artist sequentially for now (can be optimized later)
	for _, name := range artistNames {
		log.Printf("Enriching artist: %s", name)

		// Use enrichment service to get full artist data
		artist, err := s.enrichmentService.EnrichArtistByName(name, nil)
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to enrich %s: %v", name, err))
			stats.CacheMisses++
			continue
		}

		// Check if this was a cache hit by looking at verification status
		if hasAnyVerification(artist.Verified) {
			stats.CacheHits++
		} else {
			stats.CacheMisses++
		}

		stats.APICallsMade++ // Simplified - enrichment service makes multiple calls
		enriched = append(enriched, *artist)
	}

	return enriched, stats
}

// hasAnyVerification checks if an artist has been verified by any service
func hasAnyVerification(verified models.VerificationMap) bool {
	for _, isVerified := range verified {
		if isVerified {
			return true
		}
	}
	return false
}

// generateRequestID creates a unique request identifier
func generateRequestID() string {
	return fmt.Sprintf("rec_%d", time.Now().UnixNano())
}
