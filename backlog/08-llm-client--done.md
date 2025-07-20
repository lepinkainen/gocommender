# 08 - LLM Client Integration

## Overview
Implement OpenAI API client for generating artist recommendations based on seed tracks and known artists, with structured JSON response parsing.

## Steps

### 1. Create OpenAI Client Structure

- [ ] Define `internal/services/openai.go` with client and request/response types

```go
package services

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"
    
    "gocommender/internal/models"
)

// OpenAIClient handles OpenAI API interactions
type OpenAIClient struct {
    apiKey     string
    baseURL    string
    model      string
    httpClient *http.Client
}

// OpenAIRequest represents the request structure for OpenAI API
type OpenAIRequest struct {
    Model       string                 `json:"model"`
    Messages    []OpenAIMessage        `json:"messages"`
    Temperature float64                `json:"temperature"`
    MaxTokens   int                    `json:"max_tokens"`
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
    ID      string           `json:"id"`
    Object  string           `json:"object"`
    Created int64            `json:"created"`
    Model   string           `json:"model"`
    Choices []OpenAIChoice   `json:"choices"`
    Usage   OpenAIUsage      `json:"usage"`
    Error   *OpenAIError     `json:"error,omitempty"`
}

// OpenAIChoice represents a single choice in the response
type OpenAIChoice struct {
    Index        int            `json:"index"`
    Message      OpenAIMessage  `json:"message"`
    FinishReason string         `json:"finish_reason"`
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
```

### 2. Implement Client Methods

- [ ] Create core OpenAI API functionality with structured JSON responses

```go
// NewOpenAIClient creates a new OpenAI API client
func NewOpenAIClient(apiKey, model string) *OpenAIClient {
    if model == "" {
        model = "gpt-4o"
    }
    
    return &OpenAIClient{
        apiKey:  apiKey,
        baseURL: "https://api.openai.com/v1",
        model:   model,
        httpClient: &http.Client{
            Timeout: 60 * time.Second,
        },
    }
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
    
    var suggestions ArtistSuggestions
    if err := json.Unmarshal([]byte(content), &suggestions); err != nil {
        return nil, fmt.Errorf("failed to parse LLM response: %w", err)
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
```

### 3. Prompt Engineering

- [ ] Create sophisticated prompt building with exclusion lists

```go
// buildRecommendationPrompt constructs the LLM prompt for artist recommendations
func (c *OpenAIClient) buildRecommendationPrompt(seedTracks []models.PlexTrack, 
                                                knownArtists []string, 
                                                genre string, 
                                                maxResults int) string {
    var prompt strings.Builder
    
    prompt.WriteString("# Music Discovery Task\n\n")
    prompt.WriteString("I need artist recommendations for music discovery. ")
    prompt.WriteString("You must suggest ONLY artists I don't already know.\n\n")
    
    // Add seed tracks section
    prompt.WriteString("## My High-Rated Tracks (Listening Profile):\n")
    seedLimit := 20 // Limit to avoid overwhelming the context
    for i, track := range seedTracks {
        if i >= seedLimit {
            prompt.WriteString(fmt.Sprintf("... and %d more tracks\n", len(seedTracks)-seedLimit))
            break
        }
        rating := strings.Repeat("★", track.Rating)
        prompt.WriteString(fmt.Sprintf("- \"%s\" by %s (%d) %s\n", 
                                      track.Title, track.Artist, track.Year, rating))
    }
    
    // Add genre focus if specified
    if genre != "" {
        prompt.WriteString(fmt.Sprintf("\n## Genre Focus: %s\n", genre))
        prompt.WriteString("Please focus recommendations within this genre while maintaining style similarity.\n")
    }
    
    // Add exclusion list
    prompt.WriteString("\n## CRITICAL: Artists to EXCLUDE (Already in My Collection):\n")
    prompt.WriteString("DO NOT suggest ANY of these artists - they are already known to me:\n\n")
    
    // Prioritize artists that appear in seed tracks
    seedArtists := extractSeedArtists(seedTracks)
    priorityArtists := make([]string, 0)
    otherArtists := make([]string, 0)
    
    for _, artist := range knownArtists {
        if contains(seedArtists, artist) {
            priorityArtists = append(priorityArtists, artist)
        } else {
            otherArtists = append(otherArtists, artist)
        }
    }
    
    // Show priority artists first, then a sample of others
    exclusionLimit := 100 // Limit to avoid token overflow
    shown := 0
    
    for _, artist := range priorityArtists {
        if shown >= exclusionLimit {
            break
        }
        prompt.WriteString(fmt.Sprintf("- %s\n", artist))
        shown++
    }
    
    for _, artist := range otherArtists {
        if shown >= exclusionLimit {
            break
        }
        prompt.WriteString(fmt.Sprintf("- %s\n", artist))
        shown++
    }
    
    if len(knownArtists) > exclusionLimit {
        prompt.WriteString(fmt.Sprintf("\n(Showing %d of %d total known artists - please avoid ALL variations and similar names)\n", 
                                      exclusionLimit, len(knownArtists)))
    }
    
    // Add requirements
    prompt.WriteString("\n## Requirements:\n")
    prompt.WriteString(fmt.Sprintf("1. Suggest exactly %d artists\n", maxResults))
    prompt.WriteString("2. Each artist must be COMPLETELY DIFFERENT from my known artists\n")
    prompt.WriteString("3. Artists must be real and have released official albums\n")
    prompt.WriteString("4. Focus on stylistic similarity to my high-rated tracks\n")
    prompt.WriteString("5. Double-check each suggestion against the exclusion list\n")
    prompt.WriteString("6. If unsure about an artist, choose someone else\n\n")
    
    // Add response format
    prompt.WriteString("## Response Format (JSON only):\n")
    prompt.WriteString("```json\n")
    prompt.WriteString("{\n")
    prompt.WriteString("  \"suggestions\": [\"Artist 1\", \"Artist 2\", \"Artist 3\", ...],\n")
    prompt.WriteString("  \"reasoning\": \"Brief explanation of recommendations\",\n")
    prompt.WriteString("  \"confidence\": 0.85\n")
    prompt.WriteString("}\n")
    prompt.WriteString("```\n\n")
    
    prompt.WriteString("Remember: The goal is MUSIC DISCOVERY - I want to find NEW artists!")
    
    return prompt.String()
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

// contains checks if a slice contains a string (case-insensitive)
func contains(slice []string, item string) bool {
    for _, s := range slice {
        if strings.EqualFold(s, item) {
            return true
        }
    }
    return false
}
```

### 4. Response Validation and Filtering

- [ ] Implement robust validation and filtering of LLM responses

```go
// validateSuggestions ensures LLM response meets requirements
func (c *OpenAIClient) validateSuggestions(suggestions *ArtistSuggestions, maxResults int) error {
    if len(suggestions.Suggestions) == 0 {
        return fmt.Errorf("no suggestions provided")
    }
    
    if len(suggestions.Suggestions) > maxResults*2 {
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
    
    if len(suggestions.Suggestions) == 0 {
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
        if strings.HasPrefix(name, prefix) {
            name = strings.TrimPrefix(name, prefix)
            name = strings.TrimSpace(name)
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
    maxLen := len(aRunes)
    if len(bRunes) > maxLen {
        maxLen = len(bRunes)
    }
    
    for i := 0; i < len(aRunes) && i < len(bRunes); i++ {
        if aRunes[i] == bRunes[i] {
            common++
        }
    }
    
    return float64(common) / float64(maxLen)
}
```

## Verification Steps

- [ ] **API Connection**:
   ```bash
   OPENAI_API_KEY=your_key go test ./internal/services -run TestOpenAIClient
   ```

- [ ] **Prompt Generation**:
   ```bash
   go test ./internal/services -run TestPromptBuilding
   ```

- [ ] **JSON Response Parsing**:
   ```bash
   go test ./internal/services -run TestResponseParsing
   ```

- [ ] **Artist Filtering**:
   ```bash
   go test ./internal/services -run TestArtistFiltering
   ```

- [ ] **End-to-End Recommendation**:
   ```bash
   go run ./cmd/test-llm -tracks="test_tracks.json" -known="test_artists.json"
   ```

## Dependencies
- Previous: `07_plex_integration.md` (Plex track data)
- New imports: `context`, `bytes`, `encoding/json`
- Models: `gocommender/internal/models`
- Environment: `OPENAI_API_KEY`

## Next Steps
Proceed to `09_recommendation_engine.md` to implement the core recommendation workflow combining all services.

## Notes
- Uses OpenAI's structured JSON response format for reliability
- Comprehensive prompt engineering with exclusion lists
- Fuzzy matching prevents suggesting similar artists
- Token usage tracking for cost monitoring
- Graceful error handling for API failures
- Configurable model selection (gpt-4o, gpt-3.5-turbo, etc.)
- Response validation ensures quality suggestions
- Artist name cleaning handles common LLM output issues
- Similarity checking prevents near-duplicate suggestions
- Context length management for large artist libraries