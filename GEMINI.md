# GoCommender Project Guidelines for Gemini

This document guides the Gemini agent in understanding and contributing to the GoCommender project.

## 1. AI Agent Guidelines

*   **General Conventions**: Adhere to the guidelines in `llm-shared/project_tech_stack.md` for universal development practices.
*   **GitHub Workflow**: Refer to `llm-shared/GITHUB.md` for managing GitHub issues and pull requests.
*   **Go Specifics**: Consult `llm-shared/languages/go.md` for Go-specific libraries, tools, and conventions.

## 2. Project Overview & Architecture

GoCommender is a Go-based music discovery backend. It integrates with Plex to analyze listening habits, uses OpenAI for artist recommendations, and enriches artist data from MusicBrainz, Discogs, and Last.fm. Data is cached in SQLite. Functionality is exposed via a RESTful HTTP API.

**Key Data Flows:**
1.  Plex (user listening data) -> GoCommender (analysis)
2.  GoCommender (artist data) -> OpenAI (recommendations)
3.  GoCommender (artist data) -> MusicBrainz, Discogs, Last.fm (enrichment)
4.  GoCommender (enriched data) -> SQLite (caching)
5.  HTTP API -> GoCommender (requests)

## 3. Key Technologies

*   **Language**: Go (1.21+)
*   **Build Automation**: Task (`Taskfile.yml`)
*   **Configuration**: `github.com/spf13/viper` (environment variables, YAML files)
*   **Database**: SQLite (`modernc.org/sqlite` - CGO-free)
*   **External APIs**: MusicBrainz, Discogs, Last.fm, OpenAI, Plex (direct HTTP API, XML parsing)
*   **Web Framework**: Standard `net/http` library

## 4. Project Structure

Follows standard Go layout:
*   `cmd/server/`: HTTP server entry point.
*   `internal/api/`: HTTP handlers and routing.
*   `internal/config/`: Configuration management.
*   `internal/models/`: Data structures, database schema.
*   `internal/services/`: Business logic, API clients, recommendation engine.
*   `Taskfile.yml`: Defines common development operations.

## 5. Developer Workflows (using `task`)

Always use `task` for project operations.
*   **Build & Lint & Test**: `task build`
*   **Unit Tests**: `task test`
*   **CI/Coverage Tests**: `task test-ci`
*   **Integration Tests**: `task test-integration` (requires specific setup)
*   **Lint/Format**: `task lint`
*   **Clean**: `task clean`
*   **Run Dev Server**: `task dev`

## 6. Important Conventions & Patterns

*   **Error Handling**: Use Go's idiomatic error handling (`error` as last return value). Custom error types like `PlexError` are used for specific API errors.
*   **JSON Serialization to SQLite**: Custom types (`VerificationMap`, `ExternalURLs`, `Genres`) implement `driver.Valuer` and `sql.Scanner` for seamless JSON persistence in SQLite. See `internal/models/artist.go` for examples.
*   **External API Rate Limiting**: All external API clients (MusicBrainz, Discogs, Last.fm, OpenAI) implement internal rate limiting. Do not bypass this.
*   **Configuration Validation**: Required configuration fields are validated at startup.
*   **SQLite Caching**: `internal/db/cache.go` manages SQLite-based caching with configurable TTLs and background refresh.
*   **Plex Integration**: Direct HTTP API calls and XML parsing are used. See `internal/services/plex.go`.
*   **LLM Interaction**: OpenAI API is used with structured JSON responses and prompt engineering, including exclusion lists. See `prompts/openai_recommendation.tmpl` and `internal/services/openai.go`.

## 7. Known Limitations/Considerations

*   **Artist Lookup by Name**: `ArtistService.findArtistByName` is a placeholder; primary lookup is by MusicBrainz ID (MBID).
*   **Genre Extraction**: `extractGenresFromTracks` is simplified.
*   **API Server Initialization**: Be consistent with `api.NewServer` pattern for routes in `cmd/server/main.go`.
*   **Test Data**: Integration tests use in-memory SQLite.

## 8. Main API Endpoints

*   `POST /api/recommend`
*   `GET /api/artists/{mbid}`
*   `GET /api/health`
*   `GET /api/info`
*   `GET /api/plex/playlists`
*   `GET /api/plex/test`
*   `GET /api/cache/stats`
*   `POST /api/cache/clear`