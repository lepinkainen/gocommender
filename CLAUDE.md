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
- `GOCOMMENDER_*` prefix for all env vars
- Required validation: `PLEX_URL`, `PLEX_TOKEN`, `OPENAI_API_KEY`
- Optional APIs gracefully degrade when tokens missing

## Development Workflow

### Essential Commands

```bash
# Always run before claiming work complete
task build                    # Runs test + lint + build
gofmt -w .                    # Auto-format after code changes

# Development
task dev                      # Run development server
task test                     # Unit tests only
task test-ci                  # Tests with coverage for CI

# Analysis
go run llm-shared/utils/gofuncs/gofuncs.go -dir .  # Function analysis
```

### Project Structure Logic

```
cmd/server/          # HTTP entry point with flag-based modes (-config-test, -init-db-only)
internal/models/     # Core data structures with SQL compatibility
internal/config/     # Viper-based config + database initialization
internal/services/   # External API clients + business logic
internal/api/        # HTTP handlers (not yet implemented)
plan/               # Implementation roadmap with checkboxes
```

## Implementation Progress (from plan/)

✅ **Complete**: Project setup, data models, configuration, MusicBrainz integration  
🚧 **Next**: External APIs (Discogs, Last.fm), caching layer, Plex integration  
📋 **Future**: LLM client, recommendation engine, HTTP API

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