package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"gocommender/internal/config"
	"gocommender/internal/models"
	"gocommender/internal/services"
)

func main() {
	var playlist = flag.String("playlist", "", "Plex playlist name (required)")
	var genre = flag.String("genre", "", "Genre filter (optional)")
	var count = flag.Int("count", 5, "Number of recommendations")
	var verbose = flag.Bool("verbose", false, "Enable verbose logging")
	var debug = flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	if *playlist == "" {
		fmt.Printf("Usage: %s -playlist=\"My Playlist\" [-genre=\"rock\"] [-count=5] [-verbose] [-debug]\n", os.Args[0])
		fmt.Println("\nRequired environment variables:")
		fmt.Println("  PLEX_URL=http://localhost:32400")
		fmt.Println("  PLEX_TOKEN=your-plex-token")
		fmt.Println("  OPENAI_API_KEY=your-openai-key")
		fmt.Println("\nOptional environment variables:")
		fmt.Println("  DISCOGS_TOKEN=your-discogs-token")
		fmt.Println("  LASTFM_API_KEY=your-lastfm-key")
		os.Exit(1)
	}

	if !*verbose {
		log.SetOutput(os.Stderr) // Keep logs separate from JSON output
	}

	// Configure debug logging
	if *debug {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize services
	log.Println("Initializing services...")

	// Create Plex client
	plexClient := services.NewPlexClient(cfg.Plex.URL, cfg.Plex.Token)

	// Create OpenAI client
	openaiClient, err := services.NewOpenAIClient(cfg.OpenAI.APIKey, cfg.OpenAI.Model, cfg.OpenAI.PromptTemplatePath, *debug)
	if err != nil {
		log.Fatalf("Failed to create OpenAI client: %v", err)
	}

	// Create enrichment service
	enrichmentService := services.NewEnrichmentService(
		cfg.External.DiscogsToken,
		cfg.External.LastFMAPIKey,
		"", // LastFM secret not used in current implementation
	)

	// Create recommendation service
	recommendationService := services.NewRecommendationService(
		plexClient,
		openaiClient,
		enrichmentService,
	)

	// Test Plex connection
	log.Println("Testing Plex connection...")
	if err := plexClient.TestConnection(); err != nil {
		log.Fatalf("Failed to connect to Plex: %v", err)
	}

	// Build request
	request := models.RecommendRequest{
		PlaylistName: *playlist,
		MaxResults:   *count,
	}
	if *genre != "" {
		request.Genre = genre
	}

	// Generate recommendations
	log.Printf("Generating %d recommendations from playlist '%s'", *count, *playlist)
	if *genre != "" {
		log.Printf("Genre filter: %s", *genre)
	}

	ctx := context.Background()
	result, err := recommendationService.GenerateRecommendations(ctx, request)
	if err != nil {
		log.Fatalf("Failed to generate recommendations: %v", err)
	}

	// Output results as JSON
	output, err := json.MarshalIndent(result.Response, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal response: %v", err)
	}

	fmt.Println(string(output))

	// Print stats to stderr if verbose
	if *verbose {
		fmt.Fprintf(os.Stderr, "\n=== Performance Stats ===\n")
		fmt.Fprintf(os.Stderr, "Duration: %v\n", result.Stats.Duration)
		fmt.Fprintf(os.Stderr, "Seed tracks: %d\n", result.Stats.SeedTrackCount)
		fmt.Fprintf(os.Stderr, "Known artists: %d\n", result.Stats.KnownArtistCount)
		fmt.Fprintf(os.Stderr, "LLM suggestions: %d\n", result.Stats.LLMSuggestions)
		fmt.Fprintf(os.Stderr, "Filtered: %d\n", result.Stats.FilteredCount)
		fmt.Fprintf(os.Stderr, "Enriched: %d\n", result.Stats.EnrichedCount)
		fmt.Fprintf(os.Stderr, "API calls: %d\n", result.Stats.APICallsMade)
		fmt.Fprintf(os.Stderr, "Cache hits: %d\n", result.Stats.CacheHits)
		fmt.Fprintf(os.Stderr, "Cache misses: %d\n", result.Stats.CacheMisses)
		if len(result.Stats.Errors) > 0 {
			fmt.Fprintf(os.Stderr, "Errors: %d\n", len(result.Stats.Errors))
			for _, err := range result.Stats.Errors {
				fmt.Fprintf(os.Stderr, "  - %s\n", err)
			}
		}
	}
}
