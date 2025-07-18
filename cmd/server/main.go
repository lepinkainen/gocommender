package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"runtime"

	"gocommender/internal/api"
	"gocommender/internal/config"
	"gocommender/internal/db"
	"gocommender/internal/services"
)

// Build information (set by ldflags during build)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// BuildInfo contains application build information
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

func main() {
	var (
		configTest = flag.Bool("config-test", false, "Test configuration loading and exit")
		initDBOnly = flag.Bool("init-db-only", false, "Initialize database and exit")
		version    = flag.Bool("version", false, "Show version information and exit")
	)
	flag.Parse()

	if *version {
		showVersion()
		return
	}

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
		showVersion()
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

	// Create build info
	buildInfo := &api.BuildInfo{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	// Create API server
	apiServer := api.NewServer(
		recommendationService,
		enrichmentService,
		plexClient,
		cacheManager,
		buildInfo,
	)

	// Start HTTP server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("🚀 GoCommender v%s starting on %s", Version, addr)
	log.Printf("📝 Build: %s (%s)", getShortCommit(), BuildDate)

	if err := http.ListenAndServe(addr, apiServer); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func showVersion() {
	build := BuildInfo{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	fmt.Printf("GoCommender %s\n", build.Version)
	fmt.Printf("Commit: %s\n", build.Commit)
	fmt.Printf("Built: %s\n", build.BuildDate)
	fmt.Printf("Go: %s\n", build.GoVersion)
	fmt.Printf("Platform: %s\n", build.Platform)
}

func getShortCommit() string {
	if len(Commit) > 8 {
		return Commit[:8]
	}
	return Commit
}
