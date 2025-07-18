package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"

	"gocommender/internal/config"
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
	db, err := config.InitDatabase(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	if *initDBOnly {
		fmt.Println("✅ Database initialized successfully")
		return
	}

	// Set up HTTP routes
	http.HandleFunc("/api/health", healthHandler(db))

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("🚀 GoCommender server starting on %s", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func healthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Test database connection
		if err := db.Ping(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"error","service":"gocommender","error":"database unavailable"}`)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","service":"gocommender","database":"connected"}`)
	}
}
