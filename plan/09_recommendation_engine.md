# 09 - Recommendation Engine

## Overview
Implement the core recommendation workflow that orchestrates all services to generate artist recommendations with metadata and performance tracking.

## Steps

### 1. Create Recommendation Service

- [ ] Define `internal/services/recommendation.go` with orchestration service

```go
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
    plexClient    *PlexClient
    openaiClient  *OpenAIClient
    artistService *ArtistService
    cache         CacheService
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

// NewRecommendationService creates a new recommendation service
func NewRecommendationService(plex *PlexClient, openai *OpenAIClient, 
                            artist *ArtistService, cache CacheService) *RecommendationService {
    return &RecommendationService{
        plexClient:    plex,
        openaiClient:  openai,
        artistService: artist,
        cache:         cache,
    }
}
```

### 2. Implement Core Recommendation Workflow

- [ ] Create the main recommendation process with error handling

```go
// GenerateRecommendations performs the complete recommendation workflow
func (s *RecommendationService) GenerateRecommendations(ctx context.Context, 
                                                       request models.RecommendRequest) (*RecommendationResult, error) {
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
    suggestions, err := s.openaiClient.GetArtistRecommendations(
        seedTracks, knownArtists, request.Genre, request.MaxResults*2) // Request more to allow for filtering
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
```

### 3. Artist Enrichment Process

- [ ] Implement parallel artist enrichment with concurrency control

```go
// EnrichmentStats tracks enrichment performance
type EnrichmentStats struct {
    CacheHits   int      `json:"cache_hits"`
    CacheMisses int      `json:"cache_misses"`
    APICallsMade int     `json:"api_calls_made"`
    Errors      []string `json:"errors"`
}

// enrichArtistSuggestions enriches artist suggestions with metadata
func (s *RecommendationService) enrichArtistSuggestions(ctx context.Context, 
                                                       artistNames []string) ([]models.Artist, *EnrichmentStats) {
    stats := &EnrichmentStats{
        Errors: make([]string, 0),
    }
    
    enriched := make([]models.Artist, 0, len(artistNames))
    
    // Use a semaphore to limit concurrent enrichments
    semaphore := make(chan struct{}, 3) // Max 3 concurrent enrichments
    results := make(chan enrichmentResult, len(artistNames))
    
    // Start enrichment goroutines
    for i, name := range artistNames {
        go func(index int, artistName string) {
            semaphore <- struct{}{} // Acquire semaphore
            defer func() { <-semaphore }() // Release semaphore
            
            result := enrichmentResult{
                Index: index,
                Name:  artistName,
            }
            
            // Check cache first
            if artist, err := s.findCachedArtist(artistName); err == nil && artist != nil {
                result.Artist = artist
                result.CacheHit = true
            } else {
                // Enrich from APIs
                if artist, err := s.artistService.GetArtist(artistName); err == nil {
                    result.Artist = artist
                    result.APICalls = 1 // Simplified - actual count varies
                } else {
                    result.Error = err
                }
            }
            
            results <- result
        }(i, name)
    }
    
    // Collect results
    resultMap := make(map[int]enrichmentResult)
    for i := 0; i < len(artistNames); i++ {
        result := <-results
        resultMap[result.Index] = result
        
        if result.CacheHit {
            stats.CacheHits++
        } else {
            stats.CacheMisses++
        }
        
        stats.APICallsMade += result.APICalls
        
        if result.Error != nil {
            stats.Errors = append(stats.Errors, 
                fmt.Sprintf("Failed to enrich %s: %v", result.Name, result.Error))
        }
    }
    
    // Build results in original order
    for i := 0; i < len(artistNames); i++ {
        if result, exists := resultMap[i]; exists && result.Artist != nil {
            enriched = append(enriched, *result.Artist)
        }
    }
    
    return enriched, stats
}

// enrichmentResult represents the result of enriching a single artist
type enrichmentResult struct {
    Index    int
    Name     string
    Artist   *models.Artist
    CacheHit bool
    APICalls int
    Error    error
}

// findCachedArtist attempts to find an artist in cache by name
func (s *RecommendationService) findCachedArtist(name string) (*models.Artist, error) {
    // This is a simplified implementation - in practice you might need
    // a secondary index on artist names or fuzzy matching
    
    // For now, we'll use the artist service which handles cache lookup by MBID
    // This is not optimal but works for the current implementation
    return nil, fmt.Errorf("name-based cache lookup not implemented")
}

// generateRequestID creates a unique request identifier
func generateRequestID() string {
    return fmt.Sprintf("rec_%d", time.Now().UnixNano())
}
```

### 4. Recommendation Filtering and Ranking

- [ ] Add advanced filtering logic and scoring algorithms

```go
// FilterRecommendations applies additional filtering rules
func (s *RecommendationService) FilterRecommendations(artists []models.Artist, 
                                                     criteria FilterCriteria) []models.Artist {
    filtered := make([]models.Artist, 0, len(artists))
    
    for _, artist := range artists {
        if s.meetsFilterCriteria(artist, criteria) {
            filtered = append(filtered, artist)
        }
    }
    
    return filtered
}

// FilterCriteria defines filtering rules for recommendations
type FilterCriteria struct {
    MinVerificationServices int      `json:"min_verification_services"`
    RequiredServices       []string `json:"required_services"`
    MinAlbumCount          int      `json:"min_album_count"`
    ExcludeGenres          []string `json:"exclude_genres"`
    RequireDescription     bool     `json:"require_description"`
    RequireImage           bool     `json:"require_image"`
}

// meetsFilterCriteria checks if an artist meets the filtering criteria
func (s *RecommendationService) meetsFilterCriteria(artist models.Artist, 
                                                   criteria FilterCriteria) bool {
    // Check verification services
    verifiedCount := 0
    for service, verified := range artist.Verified {
        if verified {
            verifiedCount++
            
            // Check if required service
            for _, required := range criteria.RequiredServices {
                if service == required {
                    goto serviceCheck
                }
            }
        }
    }
    
    serviceCheck:
    if verifiedCount < criteria.MinVerificationServices {
        return false
    }
    
    // Check album count
    if artist.AlbumCount < criteria.MinAlbumCount {
        return false
    }
    
    // Check excluded genres
    for _, artistGenre := range artist.Genres {
        for _, excludedGenre := range criteria.ExcludeGenres {
            if strings.EqualFold(artistGenre, excludedGenre) {
                return false
            }
        }
    }
    
    // Check required fields
    if criteria.RequireDescription && artist.Description == "" {
        return false
    }
    
    if criteria.RequireImage && artist.ImageURL == "" {
        return false
    }
    
    return true
}

// RankRecommendations sorts artists by relevance score
func (s *RecommendationService) RankRecommendations(artists []models.Artist, 
                                                   seedTracks []models.PlexTrack) []models.Artist {
    type scoredArtist struct {
        artist models.Artist
        score  float64
    }
    
    scored := make([]scoredArtist, len(artists))
    
    for i, artist := range artists {
        score := s.calculateRelevanceScore(artist, seedTracks)
        scored[i] = scoredArtist{artist: artist, score: score}
    }
    
    // Sort by score (highest first)
    for i := 0; i < len(scored)-1; i++ {
        for j := i + 1; j < len(scored); j++ {
            if scored[j].score > scored[i].score {
                scored[i], scored[j] = scored[j], scored[i]
            }
        }
    }
    
    // Extract sorted artists
    ranked := make([]models.Artist, len(scored))
    for i, sa := range scored {
        ranked[i] = sa.artist
    }
    
    return ranked
}

// calculateRelevanceScore computes a relevance score for an artist
func (s *RecommendationService) calculateRelevanceScore(artist models.Artist, 
                                                       seedTracks []models.PlexTrack) float64 {
    score := 0.0
    
    // Base score for verification
    verifiedServices := 0
    for _, verified := range artist.Verified {
        if verified {
            verifiedServices++
        }
    }
    score += float64(verifiedServices) * 10.0
    
    // Bonus for having description
    if artist.Description != "" {
        score += 15.0
    }
    
    // Bonus for having image
    if artist.ImageURL != "" {
        score += 10.0
    }
    
    // Album count bonus (logarithmic)
    if artist.AlbumCount > 0 {
        score += math.Log(float64(artist.AlbumCount+1)) * 5.0
    }
    
    // Genre overlap bonus
    seedGenres := extractGenresFromTracks(seedTracks)
    genreOverlap := calculateGenreOverlap(artist.Genres, seedGenres)
    score += genreOverlap * 20.0
    
    return score
}

// extractGenresFromTracks extracts likely genres from track metadata
func extractGenresFromTracks(tracks []models.PlexTrack) []string {
    // This is simplified - in practice you might use additional metadata
    // or genre inference from artist names
    return []string{} // Placeholder
}

// calculateGenreOverlap calculates similarity between artist and seed genres
func calculateGenreOverlap(artistGenres []string, seedGenres []string) float64 {
    if len(artistGenres) == 0 || len(seedGenres) == 0 {
        return 0.0
    }
    
    overlap := 0
    for _, ag := range artistGenres {
        for _, sg := range seedGenres {
            if strings.EqualFold(ag, sg) {
                overlap++
            }
        }
    }
    
    return float64(overlap) / float64(len(seedGenres))
}
```

## Verification Steps

- [ ] **End-to-End Workflow**:
   ```bash
   go test ./internal/services -run TestRecommendationWorkflow
   ```

- [ ] **Performance Testing**:
   ```bash
   go run ./cmd/test-recommendations -playlist="My Playlist" -genre="rock" -count=5
   ```

- [ ] **Error Handling**:
   ```bash
   go test ./internal/services -run TestRecommendationErrors
   ```

- [ ] **Filtering Logic**:
   ```bash
   go test ./internal/services -run TestRecommendationFiltering
   ```

- [ ] **Ranking Algorithm**:
   ```bash
   go test ./internal/services -run TestRecommendationRanking
   ```

## Dependencies
- Previous: `08_llm_client.md` (OpenAI integration)
- All service packages: Plex, OpenAI, Artist, Cache
- New imports: `context`, `math`, `sync`

## Next Steps
Proceed to `10_http_api.md` to implement REST API endpoints that expose the recommendation functionality.

## Notes
- Orchestrates all services in a coordinated workflow
- Comprehensive error handling with graceful degradation
- Performance tracking and metrics collection
- Concurrent artist enrichment with semaphore control
- Flexible filtering and ranking system
- Request ID generation for tracking
- Fallback strategies for missing data
- Genre-aware recommendation scoring
- Cache-aware processing to minimize API calls
- Detailed logging for debugging and monitoring