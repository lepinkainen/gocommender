package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"text/template"
	"time"

	"gocommender/internal/models"
)

// OpenAIClient handles OpenAI API interactions
type OpenAIClient struct {
	apiKey         string
	baseURL        string
	model          string
	templatePath   string
	httpClient     *http.Client
	promptTemplate *template.Template
	debug          bool
}

// OpenAIRequest represents the request structure for OpenAI API
type OpenAIRequest struct {
	Model          string                `json:"model"`
	Messages       []OpenAIMessage       `json:"messages"`
	Temperature    float64               `json:"temperature"`
	MaxTokens      int                   `json:"max_tokens"`
	ResponseFormat *OpenAIResponseFormat `json:"response_format,omitempty"`
}

// OpenAIMessage represents a single message in the conversation
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponseFormat specifies JSON response format
type OpenAIResponseFormat struct {
	Type string `json:"type"`
}

// OpenAIResponse represents the response from OpenAI API
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
	Error   *OpenAIError   `json:"error,omitempty"`
}

// OpenAIChoice represents a single choice in the response
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// OpenAIUsage represents token usage statistics
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIError represents an error from OpenAI API
type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// ArtistSuggestions represents the structured response from LLM
type ArtistSuggestions struct {
	Suggestions []string `json:"suggestions"`
	Reasoning   string   `json:"reasoning,omitempty"`
	Confidence  float64  `json:"confidence,omitempty"`
}

// PromptData contains all data needed for prompt template rendering
type PromptData struct {
	SeedTracks      []PromptTrack `json:"seed_tracks"`
	Genre           string        `json:"genre,omitempty"`
	PriorityArtists []string      `json:"priority_artists"`
	OtherArtists    []string      `json:"other_artists"`
	MaxResults      int           `json:"max_results"`
	SeedLimit       int           `json:"seed_limit"`
	ExclusionLimit  int           `json:"exclusion_limit"`
	TotalKnownCount int           `json:"total_known_count"`
	TotalTrackCount int           `json:"total_track_count"`
	HasMoreTracks   bool          `json:"has_more_tracks"`
	HasMoreArtists  bool          `json:"has_more_artists"`
}

// PromptTrack represents a track for template rendering
type PromptTrack struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Year   int    `json:"year"`
	Rating int    `json:"rating"`
	Stars  string `json:"stars"`
}

// NewOpenAIClient creates a new OpenAI API client
func NewOpenAIClient(apiKey, model, templatePath string, debug bool) (*OpenAIClient, error) {
	if model == "" {
		model = "gpt-4o"
	}

	if templatePath == "" {
		templatePath = "./prompts/openai_recommendation.tmpl"
	}

	client := &OpenAIClient{
		apiKey:       apiKey,
		baseURL:      "https://api.openai.com/v1",
		model:        model,
		templatePath: templatePath,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		debug: debug,
	}

	// Load and cache the template
	if err := client.loadTemplate(); err != nil {
		return nil, fmt.Errorf("failed to load prompt template: %w", err)
	}

	return client, nil
}

// loadTemplate loads the prompt template from file
func (c *OpenAIClient) loadTemplate() error {
	// Create template with custom functions
	tmpl := template.New("openai_recommendation.tmpl").Funcs(template.FuncMap{
		"sub": func(a, b int) int { return a - b },
		"len": func(s interface{}) int {
			switch v := s.(type) {
			case []PromptTrack:
				return len(v)
			case []string:
				return len(v)
			default:
				return 0
			}
		},
	})

	// Parse template from file
	var err error
	c.promptTemplate, err = tmpl.ParseFiles(c.templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template file %s: %w", c.templatePath, err)
	}

	return nil
}

// GetArtistRecommendations generates artist suggestions based on seed data
func (c *OpenAIClient) GetArtistRecommendations(seedTracks []models.PlexTrack,
	knownArtists []string,
	genre string,
	maxResults int) (*ArtistSuggestions, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured")
	}

	if maxResults <= 0 {
		maxResults = 5
	}

	prompt := c.buildRecommendationPrompt(seedTracks, knownArtists, genre, maxResults)

	if c.debug {
		slog.Debug("OpenAI request details",
			"model", c.model,
			"seed_tracks", len(seedTracks),
			"known_artists", len(knownArtists),
			"genre", genre,
			"max_results", maxResults,
			"prompt_content", prompt,
		)
	}

	request := OpenAIRequest{
		Model: c.model,
		Messages: []OpenAIMessage{
			{
				Role:    "system",
				Content: "You are a music discovery expert. Provide artist recommendations as valid JSON responses only.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   1000,
		ResponseFormat: &OpenAIResponseFormat{
			Type: "json_object",
		},
	}

	response, err := c.sendRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}

	content := response.Choices[0].Message.Content

	if c.debug {
		slog.Debug("OpenAI raw response",
			"status", response.Choices[0].FinishReason,
			"response_content", content,
		)
	}

	var suggestions ArtistSuggestions
	if err := json.Unmarshal([]byte(content), &suggestions); err != nil {
		if c.debug {
			slog.Debug("Failed to parse JSON response", "error", err)
		}
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if c.debug {
		slog.Debug("Parsed suggestions structure",
			"suggestions_count", len(suggestions.Suggestions),
			"suggestions", suggestions.Suggestions,
			"reasoning", suggestions.Reasoning,
			"confidence", suggestions.Confidence,
		)
	}

	// Validate suggestions
	if err := c.validateSuggestions(&suggestions, maxResults); err != nil {
		return nil, fmt.Errorf("invalid suggestions: %w", err)
	}

	return &suggestions, nil
}

// sendRequest sends the request to OpenAI API
func (c *OpenAIClient) sendRequest(request OpenAIRequest) (*OpenAIResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", response.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}

	return &response, nil
}

// buildRecommendationPrompt constructs the LLM prompt for artist recommendations using templates
func (c *OpenAIClient) buildRecommendationPrompt(seedTracks []models.PlexTrack,
	knownArtists []string,
	genre string,
	maxResults int) string {

	// Prepare template data
	data := c.preparePromptData(seedTracks, knownArtists, genre, maxResults)

	if c.debug {
		slog.Debug("Prepared prompt data",
			"seed_tracks_count", len(data.SeedTracks),
			"priority_artists_count", len(data.PriorityArtists),
			"other_artists_count", len(data.OtherArtists),
			"total_known_count", data.TotalKnownCount,
			"has_more_tracks", data.HasMoreTracks,
			"has_more_artists", data.HasMoreArtists,
		)
	}

	// Execute template
	var buf bytes.Buffer
	if err := c.promptTemplate.Execute(&buf, data); err != nil {
		if c.debug {
			slog.Debug("Template execution failed, using fallback prompt", "error", err)
		}
		// Fallback to basic prompt if template fails
		return fmt.Sprintf("I need %d artist recommendations based on my music taste. Please suggest artists I don't already know.", maxResults)
	}

	if c.debug {
		slog.Debug("Template execution successful")
	}

	return buf.String()
}

// preparePromptData prepares the data structure for template rendering
func (c *OpenAIClient) preparePromptData(seedTracks []models.PlexTrack, knownArtists []string, genre string, maxResults int) *PromptData {
	seedLimit := 20
	exclusionLimit := 100

	// Convert seed tracks to template format
	promptTracks := make([]PromptTrack, 0)
	for i, track := range seedTracks {
		if i >= seedLimit {
			break
		}
		promptTracks = append(promptTracks, PromptTrack{
			Title:  track.Title,
			Artist: track.Artist,
			Year:   track.Year,
			Rating: track.Rating,
			Stars:  strings.Repeat("★", track.Rating),
		})
	}

	// Separate priority and other artists
	seedArtists := extractSeedArtists(seedTracks)
	priorityArtists := make([]string, 0)
	otherArtists := make([]string, 0)

	for _, artist := range knownArtists {
		if containsArtist(seedArtists, artist) {
			priorityArtists = append(priorityArtists, artist)
		} else {
			otherArtists = append(otherArtists, artist)
		}
	}

	// Limit artists shown to avoid token overflow
	shown := 0
	limitedPriority := make([]string, 0)
	limitedOther := make([]string, 0)

	for _, artist := range priorityArtists {
		if shown >= exclusionLimit {
			break
		}
		limitedPriority = append(limitedPriority, artist)
		shown++
	}

	for _, artist := range otherArtists {
		if shown >= exclusionLimit {
			break
		}
		limitedOther = append(limitedOther, artist)
		shown++
	}

	return &PromptData{
		SeedTracks:      promptTracks,
		Genre:           genre,
		PriorityArtists: limitedPriority,
		OtherArtists:    limitedOther,
		MaxResults:      maxResults,
		SeedLimit:       seedLimit,
		ExclusionLimit:  exclusionLimit,
		TotalKnownCount: len(knownArtists),
		TotalTrackCount: len(seedTracks),
		HasMoreTracks:   len(seedTracks) > seedLimit,
		HasMoreArtists:  len(knownArtists) > exclusionLimit,
	}
}

// extractSeedArtists gets unique artists from seed tracks
func extractSeedArtists(tracks []models.PlexTrack) []string {
	seen := make(map[string]bool)
	artists := make([]string, 0)

	for _, track := range tracks {
		if track.Artist != "" && !seen[track.Artist] {
			seen[track.Artist] = true
			artists = append(artists, track.Artist)
		}
	}

	return artists
}

// containsArtist checks if a slice contains a string (case-insensitive)
func containsArtist(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

// validateSuggestions ensures LLM response meets requirements
func (c *OpenAIClient) validateSuggestions(suggestions *ArtistSuggestions, maxResults int) error {
	if c.debug {
		slog.Debug("Validating suggestions",
			"raw_suggestions_count", len(suggestions.Suggestions),
			"max_results", maxResults,
		)
	}

	if len(suggestions.Suggestions) == 0 {
		if c.debug {
			slog.Debug("Validation failed: no suggestions provided")
		}
		return fmt.Errorf("no suggestions provided")
	}

	if len(suggestions.Suggestions) > maxResults*2 {
		if c.debug {
			slog.Debug("Truncating suggestions",
				"original_count", len(suggestions.Suggestions),
				"truncated_to", maxResults,
			)
		}
		// Truncate if too many suggestions
		suggestions.Suggestions = suggestions.Suggestions[:maxResults]
	}

	// Remove duplicates and clean up names
	cleaned := make([]string, 0, len(suggestions.Suggestions))
	seen := make(map[string]bool)

	for _, artist := range suggestions.Suggestions {
		clean := cleanArtistName(artist)
		if clean == "" {
			continue
		}

		key := strings.ToLower(clean)
		if !seen[key] {
			seen[key] = true
			cleaned = append(cleaned, clean)
		}
	}

	suggestions.Suggestions = cleaned

	if c.debug {
		slog.Debug("Suggestions after cleaning",
			"cleaned_count", len(suggestions.Suggestions),
			"cleaned_suggestions", suggestions.Suggestions,
		)
	}

	if len(suggestions.Suggestions) == 0 {
		if c.debug {
			slog.Debug("Validation failed: no valid suggestions after cleaning")
		}
		return fmt.Errorf("no valid suggestions after cleaning")
	}

	return nil
}

// cleanArtistName removes extra characters and normalizes artist names
func cleanArtistName(name string) string {
	// Remove quotes, extra spaces, and normalize
	name = strings.Trim(name, `"'`)
	name = strings.TrimSpace(name)

	// Remove common prefixes that might indicate uncertainty
	prefixes := []string{"Maybe ", "Perhaps ", "Possibly ", "Consider "}
	for _, prefix := range prefixes {
		if after, found := strings.CutPrefix(name, prefix); found {
			name = strings.TrimSpace(after)
		}
	}

	// Validate the name isn't empty or too short
	if len(name) < 2 {
		return ""
	}

	return name
}

// FilterKnownArtists removes any suggestions that match known artists
func (c *OpenAIClient) FilterKnownArtists(suggestions []string, knownArtists []string) []string {
	filtered := make([]string, 0, len(suggestions))

	// Create lowercase map for faster lookups
	knownMap := make(map[string]bool)
	for _, artist := range knownArtists {
		knownMap[strings.ToLower(artist)] = true
	}

	for _, suggestion := range suggestions {
		if !knownMap[strings.ToLower(suggestion)] {
			// Additional fuzzy matching for similar names
			if !isSimilarToKnown(suggestion, knownArtists) {
				filtered = append(filtered, suggestion)
			}
		}
	}

	return filtered
}

// isSimilarToKnown checks for fuzzy matches against known artists
func isSimilarToKnown(suggestion string, knownArtists []string) bool {
	suggestionLower := strings.ToLower(suggestion)

	for _, known := range knownArtists {
		knownLower := strings.ToLower(known)

		// Check for substring matches (e.g., "The Beatles" vs "Beatles")
		if strings.Contains(suggestionLower, knownLower) ||
			strings.Contains(knownLower, suggestionLower) {
			return true
		}

		// Check for very similar names (simple Levenshtein-like check)
		if len(suggestionLower) > 3 && len(knownLower) > 3 {
			if similarity := calculateSimilarity(suggestionLower, knownLower); similarity > 0.8 {
				return true
			}
		}
	}

	return false
}

// calculateSimilarity returns a simple similarity score between two strings
func calculateSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}

	// Simple character overlap calculation
	aRunes := []rune(a)
	bRunes := []rune(b)

	common := 0
	maxLen := max(len(aRunes), len(bRunes))

	for i := 0; i < len(aRunes) && i < len(bRunes); i++ {
		if aRunes[i] == bRunes[i] {
			common++
		}
	}

	return float64(common) / float64(maxLen)
}
