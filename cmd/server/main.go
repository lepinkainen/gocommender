package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"gocommender/internal/api"
	"gocommender/internal/config"
	"gocommender/internal/db"
	"gocommender/internal/services"
)

func main() {
	var configTest = flag.Bool("config-test", false, "Test configuration loading and exit")
	var initDBOnly = flag.Bool("init-db-only", false, "Initialize database and exit")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if *configTest {
		fmt.Println("✅ Configuration loaded successfully")
		fmt.Printf("Plex URL: %s\n", cfg.Plex.URL)
		fmt.Printf("Server: %s:%s\n", cfg.Server.Host, cfg.Server.Port)
		fmt.Printf("Database: %s\n", cfg.Database.Path)
		return
	}

	// Initialize database
	database, err := config.InitDatabase(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	if *initDBOnly {
		fmt.Println("✅ Database initialized successfully")
		return
	}

	// Initialize services
	cacheManager := db.NewCacheManager(database)
	enrichmentService := services.NewEnrichmentService(
		cfg.External.DiscogsToken,
		cfg.External.LastFMAPIKey,
		"", // Last.fm secret not used
	)
	plexClient := services.NewPlexClient(cfg.Plex.URL, cfg.Plex.Token)
	openaiClient, err := services.NewOpenAIClient(
		cfg.OpenAI.APIKey,
		cfg.OpenAI.Model,
		"prompts/openai_recommendation.tmpl",
		false, // debug
	)
	if err != nil {
		log.Fatalf("Failed to initialize OpenAI client: %v", err)
	}
	recommendationService := services.NewRecommendationService(
		plexClient,
		openaiClient,
		enrichmentService,
	)

	// Create API server
	apiServer := api.NewServer(
		recommendationService,
		enrichmentService,
		plexClient,
		cacheManager,
	)

	// Start HTTP server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("🚀 GoCommender API server starting on %s", addr)

	if err := http.ListenAndServe(addr, apiServer); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
