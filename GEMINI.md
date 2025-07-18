# GoCommender Project Guidelines for Gemini

This document provides essential information for the Gemini agent to effectively understand, navigate, and contribute to the GoCommender project.

## 1. Project Overview

GoCommender is a music discovery backend written in Go. It integrates with Plex to analyze user listening habits (high-rated tracks, known artists) and leverages Large Language Models (LLMs) (specifically OpenAI) to recommend new artists. It enriches artist data from multiple external sources (MusicBrainz, Discogs, Last.fm) and uses an intelligent SQLite-based caching layer. It exposes its functionality via a RESTful HTTP API.

## 2. Key Technologies

*   **Language**: Go (1.21+)
*   **Build Automation**: Task (Taskfile.yml)
*   **Configuration**: `github.com/spf13/viper`
*   **Database**: SQLite (`modernc.org/sqlite` - CGO-free)
*   **External APIs**:
    *   MusicBrainz
    *   Discogs
    *   Last.fm
    *   OpenAI
    *   Plex (direct HTTP API, XML parsing)
*   **Web Framework**: Standard `net/http` library
*   **Containerization**: Docker
*   **CI/CD**: GitHub Actions

## 3. Project Structure

The project follows a standard Go project layout:

*   `cmd/server/`: Main entry point for the HTTP server.
*   `internal/api/`: HTTP handlers and API routing.
*   `internal/config/`: Application configuration management.
*   `internal/models/`: Data structures and database schema.
*   `internal/services/`: Business logic, API clients, and core recommendation engine.
*   `pkg/`: Public library code (currently empty, reserved for future shared utilities).
*   `build/`: Directory for compiled binaries and build artifacts.
*   `plan/`: Project planning documentation (Markdown files).
*   `Taskfile.yml`: Task runner configuration for common development operations.
*   `.env.example`: Template for environment variables.

## 4. Build and Test Commands

Always use `task` for project operations to ensure consistency and proper dependency management.

*   **Build**: `task build` (also runs tests and linting)
*   **Test (Unit)**: `task test`
*   **Test (CI/Coverage)**: `task test-ci`
*   **Test (Integration)**: `task test-integration` (requires specific setup, see `Taskfile.yml`)
*   **Lint/Format**: `task lint`
*   **Clean**: `task clean`
*   **Run Development Server**: `task dev`

## 5. Important Conventions and Guidelines

*   **llm-shared Guidelines**: Adhere to the conventions and best practices outlined in the `llm-shared` project, particularly for Go project structure, configuration, and CI/CD.
*   **Error Handling**: Go's idiomatic error handling (returning `error` as the last return value). Custom error types are used for specific API errors (e.g., `PlexError`).
*   **JSON Serialization**: Custom types (e.g., `VerificationMap`, `ExternalURLs`, `Genres`) implement `driver.Valuer` and `sql.Scanner` for seamless JSON serialization to/from SQLite.
*   **Rate Limiting**: External API clients (MusicBrainz, Discogs, Last.fm, OpenAI) implement internal rate limiting to respect API provider policies.
*   **Configuration**: Configuration is loaded via `viper` from environment variables and optional YAML files. Required fields are validated at startup.
*   **Caching**: SQLite-based caching is used for enriched artist data with configurable TTLs and a background refresh worker.
*   **Plex Integration**: Direct HTTP API calls are used for Plex, requiring XML parsing.
*   **LLM Interaction**: OpenAI API is used with structured JSON responses and sophisticated prompt engineering, including exclusion lists.

## 6. Known Limitations/Considerations

*   **Artist Lookup by Name**: The `ArtistService.findArtistByName` method is currently a placeholder and not fully implemented for direct name-based cache lookups. The primary lookup is by MusicBrainz ID (MBID).
*   **Genre Extraction**: The `extractGenresFromTracks` function in the recommendation engine is a simplified placeholder. More sophisticated genre inference from track metadata or artist tags might be needed in the future.
*   **API Server Initialization**: The `cmd/server/main.go` might still contain some direct `http.HandleFunc` calls instead of fully leveraging the `api.NewServer` pattern for all routes. Ensure consistency if modifying API routes.
*   **Test Data**: Integration tests use in-memory SQLite databases for isolation. For more complex scenarios, consider persistent test databases.

## 7. Main API Endpoints

*   `POST /api/recommend`: Generate artist recommendations.
*   `GET /api/artists/{mbid}`: Retrieve detailed artist information by MusicBrainz ID.
*   `GET /api/health`: Service health check.
*   `GET /api/info`: Detailed service information, including build details.
*   `GET /api/plex/playlists`: List available Plex playlists.
*   `GET /api/plex/test`: Test Plex connection.
*   `GET /api/cache/stats`: Retrieve cache performance statistics.
*   `POST /api/cache/clear`: Clear expired cache entries.
