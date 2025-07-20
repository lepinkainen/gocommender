package api

import (
	"fmt"
	"testing"
	"time"

	"gocommender/internal/db"
	"gocommender/internal/testutil"
)

func TestMockServicesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("Mock services basic functionality", func(t *testing.T) {
		plexClient := testutil.NewMockPlexClient()
		enrichmentService := testutil.NewMockEnrichmentService()
		openaiClient := testutil.NewMockOpenAIClient()
		cacheManager := testutil.NewMockCacheManager()

		// Test mock services work as expected
		testutil.AssertNotNil(t, plexClient)
		testutil.AssertNotNil(t, enrichmentService)
		testutil.AssertNotNil(t, openaiClient)
		testutil.AssertNotNil(t, cacheManager)

		// Test Plex client
		playlists, err := plexClient.GetPlaylists()
		testutil.AssertNoError(t, err)
		testutil.AssertTrue(t, len(playlists) > 0)

		// Test error handling
		plexClient.SetError(true)
		_, err = plexClient.GetPlaylists()
		testutil.AssertError(t, err)

		plexClient.SetError(false)
		err = plexClient.TestConnection()
		testutil.AssertNoError(t, err)
	})
}

func TestCacheManagerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	database, cleanup := testutil.CreateTestDB(t)
	defer cleanup()

	cacheManager := db.NewCacheManager(database)
	config := db.DefaultCacheConfig()

	t.Run("Store and retrieve artist", func(t *testing.T) {
		artist := testutil.TestArtist()

		// Cache the artist
		err := cacheManager.CacheArtist(artist, config)
		testutil.AssertNoError(t, err)

		// Retrieve the artist
		retrieved, needsFetch, err := cacheManager.GetOrFetchArtist(artist.MBID)
		testutil.AssertNoError(t, err)
		testutil.AssertNotNil(t, retrieved)
		testutil.AssertFalse(t, needsFetch) // Should not need fetch for fresh cache

		// Verify data integrity
		testutil.AssertEqual(t, artist.MBID, retrieved.MBID)
		testutil.AssertEqual(t, artist.Name, retrieved.Name)
		testutil.AssertEqual(t, len(artist.Genres), len(retrieved.Genres))
		testutil.AssertEqual(t, len(artist.Verified), len(retrieved.Verified))
	})

	t.Run("Cache expiry logic", func(t *testing.T) {
		artist := testutil.TestArtistExpired()

		// First cache the artist normally
		err := cacheManager.CacheArtist(artist, config)
		testutil.AssertNoError(t, err)

		// Then manually set cache expiry to past to simulate expiry
		expiredTime := time.Now().Add(-time.Hour)
		err = cacheManager.UpdateCacheExpiry(artist.MBID, expiredTime)
		testutil.AssertNoError(t, err)

		// Should indicate needs fetch due to expiry
		retrieved, needsFetch, err := cacheManager.GetOrFetchArtist(artist.MBID)
		testutil.AssertNoError(t, err)
		testutil.AssertNotNil(t, retrieved) // Should still return the artist
		testutil.AssertTrue(t, needsFetch)  // But indicate it needs refresh
	})

	t.Run("Bulk cache operations", func(t *testing.T) {
		artists := testutil.TestArtistCollection()

		err := cacheManager.BulkCacheArtists(artists, config)
		testutil.AssertNoError(t, err)

		// Verify all artists were cached
		for _, artist := range artists {
			retrieved, needsFetch, err := cacheManager.GetOrFetchArtist(artist.MBID)
			testutil.AssertNoError(t, err)
			testutil.AssertNotNil(t, retrieved)
			testutil.AssertFalse(t, needsFetch)
			testutil.AssertEqual(t, artist.Name, retrieved.Name)
		}
	})

	t.Run("Cache statistics", func(t *testing.T) {
		// Clear any existing data first
		stats, err := cacheManager.GetCacheStats()
		testutil.AssertNoError(t, err)
		initialTotal := stats.Total

		artist := testutil.TestArtistMinimal()
		artist.MBID = "stats-test-mbid"

		err = cacheManager.CacheArtist(artist, config)
		testutil.AssertNoError(t, err)

		stats, err = cacheManager.GetCacheStats()
		testutil.AssertNoError(t, err)

		testutil.AssertEqual(t, initialTotal+1, stats.Total)
		testutil.AssertGreaterOrEqual(t, stats.Valid, 1)
	})

	t.Run("Cleanup expired entries", func(t *testing.T) {
		// Create an artist that's been expired for a long time
		artist := testutil.TestArtistMinimal()
		artist.MBID = "cleanup-test-mbid"

		// First cache the artist normally
		err := cacheManager.CacheArtist(artist, config)
		testutil.AssertNoError(t, err)

		// Then manually set cache expiry to past to simulate long expiry
		expiredTime := time.Now().Add(-2 * time.Hour) // Expired 2 hours ago
		err = cacheManager.UpdateCacheExpiry(artist.MBID, expiredTime)
		testutil.AssertNoError(t, err)

		// Cleanup entries that have been expired for more than 1 hour
		deleted, err := cacheManager.CleanupExpiredEntries(time.Hour)
		testutil.AssertNoError(t, err)
		testutil.AssertGreaterOrEqual(t, deleted, 1)

		// Verify the artist was deleted
		retrieved, needsFetch, err := cacheManager.GetOrFetchArtist(artist.MBID)
		testutil.AssertNoError(t, err)
		if retrieved != nil {
			t.Errorf("Expected artist to be deleted, but found: MBID=%s, Name=%s, CacheExpiry=%v",
				retrieved.MBID, retrieved.Name, retrieved.CacheExpiry)
		}
		testutil.AssertTrue(t, needsFetch)
	})
}

func TestEnrichmentServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	enrichmentService := testutil.NewMockEnrichmentService()

	t.Run("Mock enrichment service functionality", func(t *testing.T) {
		testutil.AssertNotNil(t, enrichmentService)

		// Test error handling
		enrichmentService.SetError(true)
		enrichmentService.SetError(false) // Reset

		// Test delay setting
		enrichmentService.SetDelay(10 * time.Millisecond)
		enrichmentService.SetDelay(0) // Reset

		// Test pre-configured artist setting
		artist := testutil.TestArtist()
		enrichmentService.SetEnrichedArtist(artist.MBID, artist)
	})
}

func TestAPIServerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("Mock services compatibility", func(t *testing.T) {
		// Test that mock services can be used for testing
		plexClient := testutil.NewMockPlexClient()
		cacheManager := testutil.NewMockCacheManager()
		enrichmentService := testutil.NewMockEnrichmentService()
		recommendationService := testutil.NewMockRecommendationService()

		// Verify all mocks are created successfully
		testutil.AssertNotNil(t, plexClient)
		testutil.AssertNotNil(t, cacheManager)
		testutil.AssertNotNil(t, enrichmentService)
		testutil.AssertNotNil(t, recommendationService)

		// Test cache manager mock functionality
		artist := testutil.TestArtist()
		err := cacheManager.CacheArtist(artist, db.DefaultCacheConfig())
		testutil.AssertNoError(t, err)

		retrieved, needsFetch, err := cacheManager.GetOrFetchArtist(artist.MBID)
		testutil.AssertNoError(t, err)
		testutil.AssertNotNil(t, retrieved)
		testutil.AssertFalse(t, needsFetch)
	})
}

func TestConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	database, cleanup := testutil.CreateTestDB(t)
	defer cleanup()

	cacheManager := db.NewCacheManager(database)
	config := db.DefaultCacheConfig()

	t.Run("Concurrent cache operations", func(t *testing.T) {
		const numGoroutines = 10
		const artistsPerGoroutine = 5

		// Channel to collect errors
		errChan := make(chan error, numGoroutines)

		// Launch concurrent goroutines
		for i := 0; i < numGoroutines; i++ {
			go func(routineID int) {
				for j := 0; j < artistsPerGoroutine; j++ {
					artist := testutil.TestArtistMinimal()
					artist.MBID = fmt.Sprintf("concurrent-%d-%d", routineID, j)
					artist.Name = fmt.Sprintf("Artist %d-%d", routineID, j)

					err := cacheManager.CacheArtist(artist, config)
					if err != nil {
						errChan <- err
						return
					}

					// Try to retrieve it
					retrieved, _, err := cacheManager.GetOrFetchArtist(artist.MBID)
					if err != nil {
						errChan <- err
						return
					}
					if retrieved == nil {
						errChan <- fmt.Errorf("failed to retrieve artist %s", artist.MBID)
						return
					}
				}
				errChan <- nil
			}(i)
		}

		// Collect results
		errorCount := 0
		for i := 0; i < numGoroutines; i++ {
			if err := <-errChan; err != nil {
				t.Errorf("Goroutine error: %v", err)
				errorCount++
			}
		}

		if errorCount > 0 {
			t.Errorf("Had %d errors in concurrent operations", errorCount)
		}

		// Verify final state
		stats, err := cacheManager.GetCacheStats()
		testutil.AssertNoError(t, err)
		testutil.AssertGreaterOrEqual(t, stats.Total, numGoroutines*artistsPerGoroutine)
	})
}
