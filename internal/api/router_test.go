package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// createTestServer creates a Server instance with test buildInfo for testing
func createTestServer() *Server {
	buildInfo := &BuildInfo{
		Version:   "test",
		Commit:    "test-commit",
		BuildDate: "test-date",
		GoVersion: "go1.24.5",
		Platform:  "test/amd64",
	}
	server := &Server{
		mux:       http.NewServeMux(),
		buildInfo: buildInfo,
	}
	server.setupRoutes()
	return server
}

func TestValidateMBID(t *testing.T) {
	tests := []struct {
		name     string
		mbid     string
		expected bool
	}{
		{"valid MBID", "b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d", true},
		{"invalid length", "b10bbbfc-cf9e-42e0-be17", false},
		{"invalid format no hyphens", "b10bbbfccf9e42e0be17e2c3e1d2600d", false},
		{"invalid characters", "g10bbbfc-cf9e-42e0-be17-e2c3e1d2600d", false},
		{"empty string", "", false},
		{"uppercase valid", "B10BBBFC-CF9E-42E0-BE17-E2C3E1D2600D", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidMBID(tt.mbid)
			if result != tt.expected {
				t.Errorf("isValidMBID(%q) = %v, want %v", tt.mbid, result, tt.expected)
			}
		})
	}
}

func TestWriteJSONResponse(t *testing.T) {
	// Test successful JSON encoding
	w := httptest.NewRecorder()
	data := map[string]string{"message": "test"}

	writeJSONResponse(w, data, http.StatusOK)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "test" {
		t.Errorf("Expected message 'test', got %s", response["message"])
	}
}

func TestWriteErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()

	writeErrorResponse(w, "Test error", http.StatusBadRequest)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["error"] != "Test error" {
		t.Errorf("Expected error 'Test error', got %s", response["error"])
	}

	if response["status"] != "error" {
		t.Errorf("Expected status 'error', got %s", response["status"])
	}
}

func TestHandleRoot(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["service"] != "GoCommender API" {
		t.Errorf("Expected service 'GoCommender API', got %s", response["service"])
	}
}

func TestHandleInfo(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest("GET", "/api/info", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["service"] != "GoCommender" {
		t.Errorf("Expected service 'GoCommender', got %s", response["service"])
	}

	features, ok := response["features"].([]interface{})
	if !ok || len(features) == 0 {
		t.Error("Expected features array to be present and non-empty")
	}
}

func TestHandleRecommendMethodNotAllowed(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest("GET", "/api/recommend", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHandleRecommendInvalidJSON(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest("POST", "/api/recommend", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleRecommendMissingPlaylistName(t *testing.T) {
	server := createTestServer()

	requestBody := map[string]interface{}{
		"max_results": 5,
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/recommend", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["error"] != "playlist_name is required" {
		t.Errorf("Expected error about playlist_name, got %s", response["error"])
	}
}

func TestHandleArtistInvalidMBID(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest("GET", "/api/artists/invalid-mbid", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleArtistMissingMBID(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest("GET", "/api/artists/", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCorsMiddleware(t *testing.T) {
	server := createTestServer()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with CORS middleware
	corsHandler := server.corsMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected Access-Control-Allow-Origin header to be set")
	}

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header to be set")
	}
}

func TestCorsPreflightRequest(t *testing.T) {
	server := createTestServer()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := server.corsMiddleware(testHandler)

	req := httptest.NewRequest("OPTIONS", "/api/recommend", nil)
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for preflight request, got %d", http.StatusOK, w.Code)
	}
}
