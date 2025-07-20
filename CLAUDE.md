# GoCommender Development Guide

## Architecture Overview

GoCommender is a music discovery backend with a **data-centric architecture** built around the Artist model:

- **Data Flow**: Plex tracks → MusicBrainz ID lookup → Multi-source enrichment → SQLite cache → LLM recommendations
- **Primary Key**: MusicBrainz ID (MBID) serves as the universal artist identifier across all APIs
- **Service Layer**: Clean separation between external APIs (MusicBrainz, Discogs, Last.fm, OpenAI) and business logic
- **Caching Strategy**: Intelligent TTL with background refresh (30 days for verified, 7 days for failed lookups)

## Critical Development Patterns

### 1. Data Model Design

The `models.Artist` struct uses **database/sql/driver compatibility** for complex types:

```go
// All custom types implement Value() and Scan() for SQLite JSON storage
type VerificationMap map[string]bool
type Genres []string
type ExternalURLs struct { ... }
```

This pattern enables storing complex Go types as JSON in SQLite without external dependencies.

### 2. API Client Pattern

All external clients follow the **rate-limited client pattern** (see `internal/services/musicbrainz.go`):
- Rate limiting via `time.Ticker`
- User-Agent headers for API compliance
- Structured error handling with retry logic
- Data transformation to internal models

### 3. Configuration Management

Uses Viper with **environment variable precedence**:
- Supports both `GOCOMMENDER_*` prefixed and standard env vars (e.g., both `PLEX_URL` and `GOCOMMENDER_PLEX_URL` work)
- Standard format (used in examples): `PLEX_URL`, `PLEX_TOKEN`, `OPENAI_API_KEY`
- Required validation: `PLEX_URL`, `PLEX_TOKEN`, `OPENAI_API_KEY`
- Optional APIs gracefully degrade when tokens missing

### 4. Frontend Architecture Pattern

The TypeScript frontend follows a **component-based architecture** with functional patterns:

```typescript
// All components export factory functions returning HTMLElement
export function createArtistCard(artist: Artist): HTMLElement
export function createRecommendationForm(onSubmit: SubmitHandler): HTMLElement

// Type-safe API client with error handling
const api = new ApiClient('/api')
const result = await api.recommend({ playlist: 'Favorites', count: 5 })

// State management with localStorage persistence
const state = createStateManager('gocommender-state')
state.set('selectedPlaylist', playlist)
```

This pattern enables DOM manipulation without React/Vue frameworks while maintaining type safety.

## Development Workflow

### Essential Commands

```bash
# Always run before claiming work complete
task build                    # Runs test + lint + build
gofmt -w .                    # Auto-format after code changes

# Development
task dev                      # Run backend development server
task frontend-dev             # Run frontend with hot-reload (in separate terminal)
task test                     # Unit tests only
task test-ci                  # Tests with coverage for CI

# Full-stack development
task build-all                # Build both backend and frontend

# Server modes for debugging
./build/gocommender -config-test    # Test config loading
./build/gocommender -init-db-only   # Initialize database only

# Analysis
go run llm-shared/utils/gofuncs/gofuncs.go -dir .  # Function analysis
```

### Project Structure Logic

```text
cmd/server/          # HTTP entry point with flag-based modes (-config-test, -init-db-only)
cmd/test-recommendations/  # End-to-end testing CLI tool
internal/models/     # Core data structures with SQL compatibility
internal/config/     # Viper-based config + database initialization
internal/services/   # External API clients + business logic
internal/api/        # HTTP handlers with CORS and middleware
internal/db/         # Database operations (artist.go, cache.go, refresh.go)
web/                 # TypeScript frontend with Vite build system
backlog/            # Implementation roadmap with completed checkboxes
```

## Implementation Progress (from backlog/)

✅ **Complete**: Steps 1-13 (All backend services, HTTP API, testing, deployment, containerization)  
✅ **Complete**: Step 14 (Frontend implementation with TypeScript + Vite)  
🎉 **Status**: Project is production-ready and fully functional end-to-end

## End-to-End Testing Available

The complete workflow is now testable from Plex to recommendations:

```bash
# Set required environment variables
export PLEX_URL="http://localhost:32400"
export PLEX_TOKEN="your-plex-token"
export OPENAI_API_KEY="your-openai-key"

# Optional API keys for better results
export DISCOGS_TOKEN="your-discogs-token"
export LASTFM_API_KEY="your-lastfm-key"

# Run end-to-end test (first build the CLI tool)
task build
./build/test-recommendations -playlist="My Favorites" -count=3 -verbose

# Or test the web frontend
task frontend-build && task dev  # Backend at :8080, frontend at :5173
```

## Key Technical Decisions

### Database Strategy

- **SQLite with modernc.org/sqlite** (CGO-free for deployment simplicity)
- **JSON columns** for complex types (verification status, genres, external URLs)
- **MBID as primary key** ensures data consistency across API sources

### Error Handling Philosophy

- **Graceful degradation**: Optional APIs (Discogs, Last.fm) fail silently
- **Retry logic**: Only for rate-limit errors, not data errors
- **Validation upfront**: Config validation prevents runtime failures

### External Dependencies

- **Minimal approach**: Standard library preferred (following llm-shared guidelines)
- **Viper for config**: Industry standard with env var support
- **Rate limiting**: Custom implementation to respect API limits

## Testing Strategy

- **Unit tests**: Focus on data transformation and JSON serialization
- **Integration tests**: Use `//go:build !ci` to skip in CI when APIs unavailable
- **Manual testing**: Use flag-based modes (`-config-test`, `-init-db-only`)

## Guidelines Reference

- **Tech stack guidelines**: `llm-shared/project_tech_stack.md`
- **Function analysis**: `go run llm-shared/utils/gofuncs/gofuncs.go -dir .`
- **Build system**: Taskfile.yml with test/lint dependencies

## Common Gotchas

1. **Always run `gofmt -w .`** after code changes (imports auto-adjust)
2. **Rate limiting**: All API clients must implement proper rate limiting
3. **JSON serialization**: Custom types need `Value()/Scan()` methods for database compatibility
4. **Configuration**: Environment variables override config files (Viper precedence)
5. **MBID handling**: Ensure MusicBrainz ID is preserved through all data transformations
6. **Flag-based testing**: Use `-config-test` and `-init-db-only` flags for debugging server initialization
7. **User-Agent headers**: All external API clients use "GoCommender/1.0" format for compliance
8. **Cache TTL strategy**: 30 days for verified artists, 7 days for failed lookups
9. **Build dependency**: `task build` must succeed before claiming any task complete
10. **Frontend development**: Use `task frontend-dev` for hot-reload during development
11. **API endpoints**: Backend runs on :8080, frontend dev server on :5173 (with proxy)
12. **Database location**: Default SQLite path is `./data/gocommender.db` (configurable via env)
13. **CORS handling**: API router includes CORS middleware for frontend integration
