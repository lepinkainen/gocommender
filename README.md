# GoCommender

A Go rewrite of PlexCommender - a music discovery backend that integrates Plex, external music APIs, and LLMs to recommend new artists based on listening habits.

## Features

- **Plex Integration**: Extract high-rated tracks and known artists from Plex library
- **LLM Recommendations**: Use OpenAI to suggest new artists based on listening patterns
- **Multi-Source Verification**: Verify and enrich artist data from MusicBrainz, Discogs, and Last.fm
- **Intelligent Caching**: SQLite-based caching with TTL and background refresh
- **REST API**: HTTP API ready for web UI integration

## Quick Start

1. Copy environment configuration:
   ```bash
   cp .env.example .env
   # Edit .env with your API keys and Plex configuration
   ```

2. Build and run:
   ```bash
   task build
   ./build/gocommender
   ```

## Development

### Requirements
- Go 1.21+
- [Task](https://taskfile.dev) for build automation

### Commands
- `task build` - Build the server
- `task test` - Run tests
- `task lint` - Run linter and formatter
- `task dev` - Run development server
- `task clean` - Clean build artifacts

### Project Structure
```
cmd/server/     - HTTP server entry point
internal/api/   - HTTP handlers and routing
internal/config/ - Configuration management
internal/models/ - Data structures
internal/services/ - Business logic
```

## API Endpoints

- `POST /api/recommend` - Get artist recommendations
- `GET /api/health` - Health check
- `GET /api/artists/{mbid}` - Get cached artist details

## License

MIT License