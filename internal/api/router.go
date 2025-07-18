package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"gocommender/internal/db"
	"gocommender/internal/models"
	"gocommender/internal/services"
)

// Server holds the HTTP server and dependencies
type Server struct {
	mux                   *http.ServeMux
	recommendationService *services.RecommendationService
	enrichmentService     *services.EnrichmentService
	plexClient            *services.PlexClient
	cacheManager          *db.CacheManager
}

// NewServer creates a new HTTP server with all routes configured
func NewServer(recommendationService *services.RecommendationService,
	enrichmentService *services.EnrichmentService,
	plexClient *services.PlexClient,
	cacheManager *db.CacheManager) *Server {

	server := &Server{
		mux:                   http.NewServeMux(),
		recommendationService: recommendationService,
		enrichmentService:     enrichmentService,
		plexClient:            plexClient,
		cacheManager:          cacheManager,
	}

	server.setupRoutes()
	return server
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Add CORS middleware
	s.corsMiddleware(s.loggingMiddleware(s.mux)).ServeHTTP(w, r)
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health and info endpoints
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/info", s.handleInfo)

	// Recommendation endpoints
	s.mux.HandleFunc("/api/recommend", s.handleRecommend)

	// Artist endpoints
	s.mux.HandleFunc("/api/artists/", s.handleArtist) // Path with trailing slash for ID capture

	// Plex endpoints
	s.mux.HandleFunc("/api/plex/playlists", s.handlePlexPlaylists)
	s.mux.HandleFunc("/api/plex/test", s.handlePlexTest)

	// Cache endpoints
	s.mux.HandleFunc("/api/cache/stats", s.handleCacheStats)
	s.mux.HandleFunc("/api/cache/clear", s.handleCacheClear)

	// Static route for testing
	s.mux.HandleFunc("/", s.handleRoot)
}

// handleHealth provides service health information
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":    "ok",
		"service":   "gocommender",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	}

	// Test database connection
	if stats, err := s.cacheManager.GetCacheStats(); err == nil {
		health["database"] = map[string]interface{}{
			"status":          "connected",
			"total_entries":   stats.Total,
			"valid_entries":   stats.Valid,
			"expired_entries": stats.Expired,
		}
	} else {
		health["database"] = map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	}

	// Test Plex connection
	if err := s.plexClient.TestConnection(); err == nil {
		health["plex"] = map[string]string{"status": "connected"}
	} else {
		health["plex"] = map[string]string{
			"status": "error",
			"error":  err.Error(),
		}
	}

	writeJSONResponse(w, health, http.StatusOK)
}

// handleRecommend generates artist recommendations
func (s *Server) handleRecommend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request models.RecommendRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeErrorResponse(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	// Validate request
	if request.PlaylistName == "" {
		writeErrorResponse(w, "playlist_name is required", http.StatusBadRequest)
		return
	}

	if request.MaxResults <= 0 {
		request.MaxResults = 5 // Default
	}

	if request.MaxResults > 20 {
		request.MaxResults = 20 // Limit
	}

	// Generate recommendations
	ctx := r.Context()
	result, err := s.recommendationService.GenerateRecommendations(ctx, request)
	if err != nil {
		log.Printf("Recommendation error: %v", err)
		writeErrorResponse(w, fmt.Sprintf("Failed to generate recommendations: %v", err),
			http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, result.Response, http.StatusOK)
}

// handleArtist retrieves artist information by MBID
func (s *Server) handleArtist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract MBID from path: /api/artists/{mbid}
	path := strings.TrimPrefix(r.URL.Path, "/api/artists/")
	if path == "" {
		writeErrorResponse(w, "Artist MBID required", http.StatusBadRequest)
		return
	}

	// Validate MBID format (basic UUID validation)
	if !isValidMBID(path) {
		writeErrorResponse(w, "Invalid MBID format", http.StatusBadRequest)
		return
	}

	artist, needsFetch, err := s.cacheManager.GetOrFetchArtist(path)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeErrorResponse(w, "Artist not found", http.StatusNotFound)
			return
		}
		log.Printf("Artist lookup error: %v", err)
		writeErrorResponse(w, "Failed to retrieve artist", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"artist":      artist,
		"needs_fetch": needsFetch,
	}

	writeJSONResponse(w, response, http.StatusOK)
}

// handlePlexPlaylists lists available Plex playlists
func (s *Server) handlePlexPlaylists(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	playlists, err := s.plexClient.GetPlaylists()
	if err != nil {
		log.Printf("Plex playlists error: %v", err)
		writeErrorResponse(w, "Failed to retrieve playlists", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"playlists": playlists,
		"count":     len(playlists),
	}

	writeJSONResponse(w, response, http.StatusOK)
}

// handlePlexTest tests Plex connection and returns server info
func (s *Server) handlePlexTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.plexClient.TestConnection(); err != nil {
		writeErrorResponse(w, fmt.Sprintf("Plex connection failed: %v", err),
			http.StatusServiceUnavailable)
		return
	}

	serverInfo, err := s.plexClient.GetServerInfo()
	if err != nil {
		log.Printf("Plex server info error: %v", err)
		serverInfo = map[string]string{"status": "connected"}
	}

	response := map[string]interface{}{
		"status": "connected",
		"server": serverInfo,
	}

	writeJSONResponse(w, response, http.StatusOK)
}

// handleCacheStats returns cache performance statistics
func (s *Server) handleCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := s.cacheManager.GetCacheStats()
	if err != nil {
		log.Printf("Cache stats error: %v", err)
		writeErrorResponse(w, "Failed to retrieve cache stats", http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, stats, http.StatusOK)
}

// handleCacheClear clears expired cache entries
func (s *Server) handleCacheClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters for clear type
	clearType := r.URL.Query().Get("type")
	if clearType == "" {
		clearType = "expired" // Default
	}

	switch clearType {
	case "expired":
		// Clear entries that have been expired for more than 24 hours
		count, err := s.cacheManager.CleanupExpiredEntries(24 * time.Hour)
		if err != nil {
			log.Printf("Cache clear error: %v", err)
			writeErrorResponse(w, "Failed to clear expired entries", http.StatusInternalServerError)
			return
		}
		response := map[string]interface{}{
			"status":  "success",
			"message": fmt.Sprintf("Cleared %d expired cache entries", count),
			"type":    "expired",
		}
		writeJSONResponse(w, response, http.StatusOK)
	case "all":
		// This would require additional implementation in cache service
		writeErrorResponse(w, "Clear all not implemented", http.StatusNotImplemented)
		return
	default:
		writeErrorResponse(w, "Invalid clear type (use 'expired' or 'all')", http.StatusBadRequest)
		return
	}
}

// handleRoot provides API information
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeErrorResponse(w, "Not found", http.StatusNotFound)
		return
	}

	info := map[string]interface{}{
		"service":     "GoCommender API",
		"version":     "1.0.0",
		"description": "Music discovery backend using Plex, LLMs, and external APIs",
		"endpoints": map[string]string{
			"POST /api/recommend":     "Generate artist recommendations",
			"GET /api/artists/{mbid}": "Get artist information by MusicBrainz ID",
			"GET /api/health":         "Service health check",
			"GET /api/plex/playlists": "List Plex playlists",
			"GET /api/plex/test":      "Test Plex connection",
			"GET /api/cache/stats":    "Cache performance statistics",
			"POST /api/cache/clear":   "Clear cache entries",
		},
	}

	writeJSONResponse(w, info, http.StatusOK)
}

// handleInfo provides detailed API information
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := map[string]interface{}{
		"service":     "GoCommender",
		"version":     "1.0.0",
		"description": "AI-powered music discovery backend",
		"features": []string{
			"Plex playlist analysis",
			"LLM-based recommendations",
			"Multi-source artist verification",
			"Intelligent caching",
			"RESTful API",
		},
		"data_sources": []string{
			"MusicBrainz",
			"Discogs",
			"Last.fm",
			"OpenAI",
		},
	}

	writeJSONResponse(w, info, http.StatusOK)
}

// corsMiddleware adds CORS headers for web UI compatibility
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		log.Printf("%s %s %d %v %s", r.Method, r.URL.Path, rw.statusCode,
			duration, r.UserAgent())
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// writeJSONResponse writes a JSON response with proper headers
func writeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

// writeErrorResponse writes a standardized error response
func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	errorResponse := map[string]interface{}{
		"error":     message,
		"status":    "error",
		"timestamp": time.Now().UTC(),
	}

	writeJSONResponse(w, errorResponse, statusCode)
}

// isValidMBID performs basic MBID format validation
func isValidMBID(mbid string) bool {
	// Basic UUID format check: 8-4-4-4-12 characters
	if len(mbid) != 36 {
		return false
	}

	// Check for hyphens in correct positions
	if mbid[8] != '-' || mbid[13] != '-' || mbid[18] != '-' || mbid[23] != '-' {
		return false
	}

	// Check that all other characters are alphanumeric
	for i, r := range mbid {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue // Skip hyphens
		}
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}

	return true
}
