package services

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gocommender/internal/models"
)

// EnrichmentService orchestrates artist data enrichment from multiple sources
type EnrichmentService struct {
	musicbrainz *MusicBrainzClient
	discogs     *DiscogsClient
	lastfm      *LastFMClient
}

// EnrichmentOptions configures the enrichment process
type EnrichmentOptions struct {
	ForceUpdate    bool     // Force update even if recently cached
	SourcePriority []string // Order of sources for data precedence
}

// NewEnrichmentService creates a new enrichment service with all clients
func NewEnrichmentService(discogsToken, lastfmAPIKey, lastfmSecret string) *EnrichmentService {
	return &EnrichmentService{
		musicbrainz: NewMusicBrainzClient(),
		discogs:     NewDiscogsClient(discogsToken),
		lastfm:      NewLastFMClient(lastfmAPIKey, lastfmSecret),
	}
}

// EnrichArtistByName performs full artist enrichment starting from just a name
func (s *EnrichmentService) EnrichArtistByName(name string, options *EnrichmentOptions) (*models.Artist, error) {
	if options == nil {
		options = &EnrichmentOptions{
			SourcePriority: []string{"musicbrainz", "discogs", "lastfm"},
		}
	}

	// Start with MusicBrainz to get the MBID and basic data
	mbArtist, err := s.musicbrainz.SearchArtist(name)
	if err != nil {
		return nil, fmt.Errorf("failed to find artist in MusicBrainz: %w", err)
	}

	// Convert to our internal model
	artist := mbArtist.ToArtistModel()

	// Enrich with additional sources
	if err := s.EnrichExistingArtist(artist, options); err != nil {
		log.Printf("Warning: enrichment failed for %s: %v", name, err)
		// Don't return error - we have basic data from MusicBrainz
	}

	return artist, nil
}

// EnrichArtistByMBID performs full artist enrichment starting from an MBID
func (s *EnrichmentService) EnrichArtistByMBID(mbid string, options *EnrichmentOptions) (*models.Artist, error) {
	if options == nil {
		options = &EnrichmentOptions{
			SourcePriority: []string{"musicbrainz", "discogs", "lastfm"},
		}
	}

	// Get detailed data from MusicBrainz
	mbArtist, err := s.musicbrainz.GetArtistByMBID(mbid)
	if err != nil {
		return nil, fmt.Errorf("failed to get artist from MusicBrainz: %w", err)
	}

	// Convert to our internal model
	artist := mbArtist.ToArtistModel()

	// Enrich with additional sources
	if err := s.EnrichExistingArtist(artist, options); err != nil {
		log.Printf("Warning: enrichment failed for MBID %s: %v", mbid, err)
		// Don't return error - we have basic data from MusicBrainz
	}

	return artist, nil
}

// EnrichExistingArtist enriches an existing artist model with additional sources
func (s *EnrichmentService) EnrichExistingArtist(artist *models.Artist, options *EnrichmentOptions) error {
	if options == nil {
		options = &EnrichmentOptions{
			SourcePriority: []string{"discogs", "lastfm"},
		}
	}

	// Check if we need to update based on cache expiry
	if !options.ForceUpdate && time.Now().Before(artist.CacheExpiry) {
		log.Printf("Artist %s is still cached, skipping enrichment", artist.Name)
		return nil
	}

	// Enrich with each source according to priority
	var enrichmentErrors []string

	for _, source := range options.SourcePriority {
		switch source {
		case "discogs":
			if err := s.enrichWithDiscogs(artist); err != nil {
				enrichmentErrors = append(enrichmentErrors, fmt.Sprintf("discogs: %v", err))
			}
		case "lastfm":
			if err := s.enrichWithLastFM(artist); err != nil {
				enrichmentErrors = append(enrichmentErrors, fmt.Sprintf("lastfm: %v", err))
			}
		case "musicbrainz":
			// Skip if already have MusicBrainz data (MBID is required)
			if artist.MBID == "" {
				enrichmentErrors = append(enrichmentErrors, "musicbrainz: MBID required")
			}
		}
	}

	// Update cache expiry
	artist.LastUpdated = time.Now()
	if hasSuccessfulVerification(artist) {
		artist.CacheExpiry = time.Now().Add(30 * 24 * time.Hour) // 30 days for verified
	} else {
		artist.CacheExpiry = time.Now().Add(7 * 24 * time.Hour) // 7 days for failed lookups
	}

	// Return combined errors if any
	if len(enrichmentErrors) > 0 {
		return fmt.Errorf("enrichment partial failures: %s", strings.Join(enrichmentErrors, "; "))
	}

	return nil
}

// enrichWithDiscogs enriches artist with Discogs data
func (s *EnrichmentService) enrichWithDiscogs(artist *models.Artist) error {
	if s.discogs == nil {
		return fmt.Errorf("discogs client not initialized")
	}

	return s.discogs.EnrichArtist(artist)
}

// enrichWithLastFM enriches artist with Last.fm data
func (s *EnrichmentService) enrichWithLastFM(artist *models.Artist) error {
	if s.lastfm == nil {
		return fmt.Errorf("lastfm client not initialized")
	}

	return s.lastfm.EnrichArtist(artist)
}

// hasSuccessfulVerification checks if artist has at least one successful verification
func hasSuccessfulVerification(artist *models.Artist) bool {
	if artist.Verified == nil {
		return false
	}

	for _, verified := range artist.Verified {
		if verified {
			return true
		}
	}

	return false
}

// GetEnrichmentStatus returns the verification status for an artist
func (s *EnrichmentService) GetEnrichmentStatus(artist *models.Artist) map[string]interface{} {
	status := map[string]interface{}{
		"mbid":         artist.MBID,
		"name":         artist.Name,
		"last_updated": artist.LastUpdated,
		"cache_expiry": artist.CacheExpiry,
		"verified":     artist.Verified,
		"sources":      map[string]bool{},
	}

	// Check which sources have data
	sources := status["sources"].(map[string]bool)

	if artist.MBID != "" {
		sources["musicbrainz"] = true
	}

	if artist.ExternalURLs.Discogs != "" {
		sources["discogs"] = true
	}

	if artist.ExternalURLs.LastFM != "" {
		sources["lastfm"] = true
	}

	// Data completeness indicators
	status["has_description"] = artist.Description != ""
	status["has_image"] = artist.ImageURL != ""
	status["has_genres"] = len(artist.Genres) > 0
	status["has_country"] = artist.Country != ""

	return status
}

// ValidateEnrichmentConfig validates that required API credentials are available
func (s *EnrichmentService) ValidateEnrichmentConfig() map[string]bool {
	config := map[string]bool{
		"musicbrainz": s.musicbrainz != nil,
		"discogs":     s.discogs != nil && s.discogs.token != "",
		"lastfm":      s.lastfm != nil && s.lastfm.apiKey != "",
	}

	return config
}

// Close gracefully shuts down all clients
func (s *EnrichmentService) Close() {
	if s.musicbrainz != nil {
		s.musicbrainz.Close()
	}
	if s.discogs != nil {
		s.discogs.Close()
	}
	if s.lastfm != nil {
		s.lastfm.Close()
	}
}
