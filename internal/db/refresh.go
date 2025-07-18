package db

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gocommender/internal/models"
)

// RefreshService manages background refresh of expired cache entries
type RefreshService struct {
	cacheManager *CacheManager
	config       RefreshConfig

	// Internal state
	running bool
	stopCh  chan struct{}
	mu      sync.RWMutex
}

// RefreshConfig defines refresh behavior
type RefreshConfig struct {
	Interval        time.Duration // How often to check for expired entries
	BatchSize       int           // Number of artists to refresh per batch
	MaxConcurrency  int           // Maximum number of concurrent refresh operations
	CleanupInterval time.Duration // How often to cleanup very old expired entries
	CleanupMaxAge   time.Duration // Age threshold for cleanup
}

// DefaultRefreshConfig returns sensible default configuration
func DefaultRefreshConfig() RefreshConfig {
	return RefreshConfig{
		Interval:        5 * time.Minute,    // Check every 5 minutes
		BatchSize:       10,                 // Process 10 artists at a time
		MaxConcurrency:  3,                  // Max 3 concurrent API calls
		CleanupInterval: 24 * time.Hour,     // Cleanup daily
		CleanupMaxAge:   7 * 24 * time.Hour, // Delete entries expired for 7+ days
	}
}

// RefreshFunc defines the signature for artist refresh functions
type RefreshFunc func(ctx context.Context, artist models.Artist) (*models.Artist, error)

// NewRefreshService creates a new background refresh service
func NewRefreshService(cacheManager *CacheManager, config RefreshConfig) *RefreshService {
	return &RefreshService{
		cacheManager: cacheManager,
		config:       config,
		stopCh:       make(chan struct{}),
	}
}

// Start begins the background refresh service
func (rs *RefreshService) Start(ctx context.Context, refreshFunc RefreshFunc) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.running {
		return fmt.Errorf("refresh service is already running")
	}

	rs.running = true

	// Start refresh ticker
	refreshTicker := time.NewTicker(rs.config.Interval)
	defer refreshTicker.Stop()

	// Start cleanup ticker
	cleanupTicker := time.NewTicker(rs.config.CleanupInterval)
	defer cleanupTicker.Stop()

	log.Printf("Background refresh service started (interval: %v, batch: %d)",
		rs.config.Interval, rs.config.BatchSize)

	for {
		select {
		case <-ctx.Done():
			rs.running = false
			return ctx.Err()

		case <-rs.stopCh:
			rs.running = false
			return nil

		case <-refreshTicker.C:
			if err := rs.refreshExpiredBatch(ctx, refreshFunc); err != nil {
				log.Printf("Refresh batch error: %v", err)
			}

		case <-cleanupTicker.C:
			if err := rs.cleanupOldEntries(); err != nil {
				log.Printf("Cleanup error: %v", err)
			}
		}
	}
}

// Stop gracefully stops the refresh service
func (rs *RefreshService) Stop() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.running {
		close(rs.stopCh)
	}
}

// IsRunning returns whether the service is currently running
func (rs *RefreshService) IsRunning() bool {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.running
}

// refreshExpiredBatch processes a batch of expired artists
func (rs *RefreshService) refreshExpiredBatch(ctx context.Context, refreshFunc RefreshFunc) error {
	// Get expired artists
	expiredArtists, err := rs.cacheManager.RefreshExpiredArtists(rs.config.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to get expired artists: %w", err)
	}

	if len(expiredArtists) == 0 {
		return nil // Nothing to refresh
	}

	log.Printf("Refreshing %d expired artists", len(expiredArtists))

	// Process with limited concurrency
	return rs.processArtistsBatch(ctx, expiredArtists, refreshFunc)
}

// processArtistsBatch processes artists with controlled concurrency
func (rs *RefreshService) processArtistsBatch(ctx context.Context, artists []models.Artist, refreshFunc RefreshFunc) error {
	semaphore := make(chan struct{}, rs.config.MaxConcurrency)
	var wg sync.WaitGroup
	errors := make(chan error, len(artists))

	for _, artist := range artists {
		wg.Add(1)
		go func(a models.Artist) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				errors <- ctx.Err()
				return
			}

			if err := rs.refreshSingleArtist(ctx, a, refreshFunc); err != nil {
				errors <- fmt.Errorf("failed to refresh artist %s: %w", a.MBID, err)
			}
		}(artist)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Collect any errors
	var refreshErrors []error
	for err := range errors {
		if err != nil {
			refreshErrors = append(refreshErrors, err)
		}
	}

	if len(refreshErrors) > 0 {
		return fmt.Errorf("refresh completed with %d errors: %v", len(refreshErrors), refreshErrors[0])
	}

	return nil
}

// refreshSingleArtist refreshes a single artist's data
func (rs *RefreshService) refreshSingleArtist(ctx context.Context, artist models.Artist, refreshFunc RefreshFunc) error {
	// Call the provided refresh function
	refreshedArtist, err := refreshFunc(ctx, artist)
	if err != nil {
		// Even if refresh fails, update cache expiry to avoid constant retries
		newExpiry := time.Now().Add(DefaultCacheConfig().UnverifiedTTL)
		if updateErr := rs.cacheManager.UpdateCacheExpiry(artist.MBID, newExpiry); updateErr != nil {
			log.Printf("Failed to update cache expiry for failed refresh of %s: %v", artist.MBID, updateErr)
		}
		return fmt.Errorf("refresh function failed: %w", err)
	}

	// Cache the refreshed data
	if refreshedArtist != nil {
		cacheConfig := DefaultCacheConfig()
		if err := rs.cacheManager.CacheArtist(refreshedArtist, cacheConfig); err != nil {
			return fmt.Errorf("failed to cache refreshed artist: %w", err)
		}
		log.Printf("Successfully refreshed artist: %s (%s)", refreshedArtist.Name, refreshedArtist.MBID)
	}

	return nil
}

// cleanupOldEntries removes very old expired entries to keep database size manageable
func (rs *RefreshService) cleanupOldEntries() error {
	deleted, err := rs.cacheManager.CleanupExpiredEntries(rs.config.CleanupMaxAge)
	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	if deleted > 0 {
		log.Printf("Cleaned up %d old expired cache entries", deleted)
	}

	return nil
}

// GetRefreshStats returns statistics about the refresh service
func (rs *RefreshService) GetRefreshStats() (RefreshStats, error) {
	cacheStats, err := rs.cacheManager.GetCacheStats()
	if err != nil {
		return RefreshStats{}, err
	}

	return RefreshStats{
		Running:     rs.IsRunning(),
		CacheStats:  cacheStats,
		BatchSize:   rs.config.BatchSize,
		Interval:    rs.config.Interval,
		LastCleanup: time.Now(), // TODO: Track actual last cleanup time
	}, nil
}

// RefreshStats represents refresh service statistics
type RefreshStats struct {
	Running     bool          `json:"running"`
	CacheStats  CacheStats    `json:"cache_stats"`
	BatchSize   int           `json:"batch_size"`
	Interval    time.Duration `json:"interval"`
	LastCleanup time.Time     `json:"last_cleanup"`
}
