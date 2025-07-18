# GoCommender

A Go rewrite of PlexCommender - a music discovery backend that integrates Plex, external music APIs, and LLMs to recommend new artists based on listening habits.

## Features

- **Plex Integration**: Extract high-rated tracks and known artists from Plex library
- **LLM Recommendations**: Use OpenAI to suggest new artists based on listening patterns
- **Multi-Source Verification**: Verify and enrich artist data from MusicBrainz, Discogs, and Last.fm
- **Intelligent Caching**: SQLite-based caching with TTL and background refresh
- **REST API**: HTTP API ready for web UI integration

## Quick Start

### Option 1: Docker (Recommended)

1. Pull and run the latest container:
   ```bash
   # Pull from GitHub Container Registry
   docker pull ghcr.io/your-username/gocommender:latest
   
   # Run with environment file
   docker run -d \
     --name gocommender \
     -p 8080:8080 \
     --env-file .env \
     -v gocommender-data:/data \
     ghcr.io/your-username/gocommender:latest
   ```

2. Or use Docker Compose:
   ```bash
   # Copy environment template
   cp .env.example .env
   # Edit .env with your API keys and Plex configuration
   
   # Deploy with Docker Compose
   docker-compose up -d
   ```

### Option 2: Local Build

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
- Go 1.24.5+
- [Task](https://taskfile.dev) for build automation
- Docker (for containerized deployment)

### Commands
- `task build` - Build the server
- `task build-versioned` - Build with version information
- `task test` - Run tests
- `task test-ci` - Run tests with coverage
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
- `GET /api/info` - Detailed API and build information
- `GET /api/artists/{mbid}` - Get cached artist details
- `GET /api/plex/playlists` - List Plex playlists
- `GET /api/cache/stats` - Cache performance statistics

## Container Features

- **Multi-architecture support**: linux/amd64, linux/arm64
- **Non-root user**: Runs as user ID 1001 for security
- **Health checks**: Built-in health check endpoint at `/api/health`
- **Optimized size**: Multi-stage build for minimal image size
- **Security scanning**: Automated vulnerability scanning in CI/CD
- **Version information**: Built-in version and build info via `/api/info`

### Container Usage Examples

#### Docker Run
```bash
# Basic run with required environment variables
docker run -d \
  --name gocommender \
  -p 8080:8080 \
  -e PLEX_URL=http://your-plex-server:32400 \
  -e PLEX_TOKEN=your_plex_token \
  -e OPENAI_API_KEY=your_openai_key \
  -v gocommender-data:/data \
  ghcr.io/your-username/gocommender:latest

# With all optional APIs
docker run -d \
  --name gocommender \
  -p 8080:8080 \
  -e PLEX_URL=http://your-plex-server:32400 \
  -e PLEX_TOKEN=your_plex_token \
  -e OPENAI_API_KEY=your_openai_key \
  -e DISCOGS_TOKEN=your_discogs_token \
  -e LASTFM_API_KEY=your_lastfm_key \
  -v gocommender-data:/data \
  ghcr.io/your-username/gocommender:latest
```

#### Docker Compose
```yaml
version: "3.8"

services:
  gocommender:
    image: ghcr.io/your-username/gocommender:latest
    ports:
      - "8080:8080"
    environment:
      - PLEX_URL=${PLEX_URL}
      - PLEX_TOKEN=${PLEX_TOKEN}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - DISCOGS_TOKEN=${DISCOGS_TOKEN}
      - LASTFM_API_KEY=${LASTFM_API_KEY}
      - GOCOMMENDER_DATABASE_PATH=/data/gocommender.db
    volumes:
      - gocommender-data:/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  gocommender-data:
```

#### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gocommender
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gocommender
  template:
    metadata:
      labels:
        app: gocommender
    spec:
      containers:
      - name: gocommender
        image: ghcr.io/your-username/gocommender:latest
        ports:
        - containerPort: 8080
        env:
        - name: PLEX_URL
          valueFrom:
            secretKeyRef:
              name: gocommender-secrets
              key: plex-url
        - name: PLEX_TOKEN
          valueFrom:
            secretKeyRef:
              name: gocommender-secrets
              key: plex-token
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: gocommender-secrets
              key: openai-key
        volumeMounts:
        - name: data
          mountPath: /data
        livenessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: gocommender-data
```

## Version Information

The application includes build-time version information accessible via:

- Command line: `./gocommender -version`
- API endpoint: `GET /api/info`
- Health check: `GET /api/health`

Version information includes:
- Application version
- Git commit hash
- Build timestamp
- Go version
- Platform (OS/architecture)

## License

MIT License