# 13 - Container Publishing to GitHub Container Registry

## Overview

Set up automated container publishing to GitHub Container Registry (GHCR) with proper tagging, security scanning, and multi-architecture support.

## Steps

### 1. GitHub Container Registry Setup

- [ ] Configure repository settings and permissions for GHCR

```bash
# Enable GitHub Container Registry for the repository
# This is done through GitHub repository settings:
# Settings → General → Features → Container registry (enable)
```

Update `.github/workflows/ci.yml` to include container publishing:

```yaml
name: CI/CD Pipeline

on:
  push:
    branches: [main, develop]
    tags: ["v*"]
  pull_request:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          submodules: recursive

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: "1.21"

      - name: Install Task
        run: |
          sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin

      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-

      - name: Download dependencies
        run: go mod download

      - name: Run linter
        run: task lint

      - name: Run tests with coverage
        run: task test-ci

      - name: Build application
        run: task build-ci

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
          flags: unittests

  security-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          submodules: recursive

      - name: Run Gosec Security Scanner
        uses: securecodewarrior/github-action-gosec@master
        with:
          args: "-severity medium ./..."

      - name: Run Trivy vulnerability scanner on filesystem
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: "fs"
          scan-ref: "."
          format: "sarif"
          output: "trivy-results.sarif"

      - name: Upload Trivy scan results
        uses: github/codeql-action/upload-sarif@v2
        if: always()
        with:
          sarif_file: "trivy-results.sarif"

  build-and-publish:
    runs-on: ubuntu-latest
    needs: [test, security-scan]
    permissions:
      contents: read
      packages: write

    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          submodules: recursive

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=ref,event=branch
            type=ref,event=pr
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=semver,pattern={{major}}
            type=sha,prefix={{branch}}-
            type=raw,value=latest,enable={{is_default_branch}}
          labels: |
            org.opencontainers.image.title=GoCommender
            org.opencontainers.image.description=AI-powered music discovery backend
            org.opencontainers.image.vendor=${{ github.repository_owner }}
            org.opencontainers.image.licenses=MIT

      - name: Build and push container image
        uses: docker/build-push-action@v5
        with:
          context: .
          platforms: linux/amd64,linux/arm64
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            VERSION=${{ steps.meta.outputs.version }}
            COMMIT=${{ github.sha }}
            BUILD_DATE=${{ steps.meta.outputs.created }}

      - name: Run Trivy vulnerability scanner on image
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ steps.meta.outputs.version }}
          format: "sarif"
          output: "trivy-image-results.sarif"

      - name: Upload Trivy image scan results
        uses: github/codeql-action/upload-sarif@v2
        if: always()
        with:
          sarif_file: "trivy-image-results.sarif"

  deploy:
    runs-on: ubuntu-latest
    needs: build-and-publish
    if: github.ref == 'refs/heads/main'
    environment: production

    steps:
      - name: Deploy to production
        run: |
          echo "🚀 Deploying to production environment"
          echo "Image: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:latest"
          # Add actual deployment steps here
```

### 2. Enhanced Dockerfile with Multi-Stage Build

- [ ] Update `Dockerfile` for optimized container publishing with build args

```dockerfile
# syntax=docker/dockerfile:1.4

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build the application with version info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -a -installsuffix cgo \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -o gocommender ./cmd/server

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    curl \
    && update-ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S gocommender && \
    adduser -u 1001 -S gocommender -G gocommender

# Set working directory
WORKDIR /app

# Create data directory
RUN mkdir -p /data && chown gocommender:gocommender /data

# Copy binary from builder stage
COPY --from=builder /app/gocommender .

# Copy configuration files
COPY --from=builder /app/.env.example .

# Change ownership
RUN chown -R gocommender:gocommender /app

# Switch to non-root user
USER gocommender

# Expose port
EXPOSE 8080

# Add labels for better metadata
LABEL org.opencontainers.image.title="GoCommender" \
      org.opencontainers.image.description="AI-powered music discovery backend" \
      org.opencontainers.image.vendor="GoCommender Team" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.source="https://github.com/user/gocommender" \
      org.opencontainers.image.documentation="https://github.com/user/gocommender/blob/main/README.md"

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

# Run the application
ENTRYPOINT ["./gocommender"]
```

### 3. Version Information in Application

- [ ] Update `cmd/server/main.go` to include version information and build details

```go
package main

import (
    "database/sql"
    "flag"
    "fmt"
    "log"
    "net/http"
    "runtime"
    "time"

    "gocommender/internal/config"
)

// Build information (set by ldflags during build)
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
)

// BuildInfo contains application build information
type BuildInfo struct {
    Version   string `json:"version"`
    Commit    string `json:"commit"`
    BuildDate string `json:"build_date"`
    GoVersion string `json:"go_version"`
    Platform  string `json:"platform"`
}

func main() {
    var (
        configTest = flag.Bool("config-test", false, "Test configuration loading and exit")
        initDBOnly = flag.Bool("init-db-only", false, "Initialize database and exit")
        version    = flag.Bool("version", false, "Show version information and exit")
    )
    flag.Parse()

    if *version {
        showVersion()
        return
    }

    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    if *configTest {
        fmt.Println("✅ Configuration loaded successfully")
        fmt.Printf("Plex URL: %s\n", cfg.Plex.URL)
        fmt.Printf("Server: %s:%s\n", cfg.Server.Host, cfg.Server.Port)
        fmt.Printf("Database: %s\n", cfg.Database.Path)
        showVersion()
        return
    }

    // Initialize database
    db, err := config.InitDatabase(cfg.Database.Path)
    if err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    defer db.Close()

    if *initDBOnly {
        fmt.Println("✅ Database initialized successfully")
        return
    }

    // Set up HTTP routes and services (existing code)
    http.HandleFunc("/api/health", healthHandler(db))
    http.HandleFunc("/api/info", infoHandler())

    addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
    log.Printf("🚀 GoCommender v%s starting on %s", Version, addr)
    log.Printf("📝 Build: %s (%s)", Commit[:8], BuildDate)

    if err := http.ListenAndServe(addr, nil); err != nil {
        log.Fatal("Server failed to start:", err)
    }
}

func showVersion() {
    build := BuildInfo{
        Version:   Version,
        Commit:    Commit,
        BuildDate: BuildDate,
        GoVersion: runtime.Version(),
        Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
    }

    fmt.Printf("GoCommender %s\n", build.Version)
    fmt.Printf("Commit: %s\n", build.Commit)
    fmt.Printf("Built: %s\n", build.BuildDate)
    fmt.Printf("Go: %s\n", build.GoVersion)
    fmt.Printf("Platform: %s\n", build.Platform)
}

func infoHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        build := BuildInfo{
            Version:   Version,
            Commit:    Commit,
            BuildDate: BuildDate,
            GoVersion: runtime.Version(),
            Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
        }

        info := map[string]interface{}{
            "service":     "GoCommender",
            "description": "AI-powered music discovery backend",
            "build":       build,
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

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(info)
    }
}

// healthHandler remains the same...
```

### 4. Container Registry Configuration

- [ ] Create `.github/workflows/cleanup-packages.yml` for package cleanup

```yaml
name: Cleanup Container Registry

on:
  schedule:
    # Run every Sunday at 2 AM UTC
    - cron: "0 2 * * 0"
  workflow_dispatch:

jobs:
  cleanup:
    runs-on: ubuntu-latest
    permissions:
      packages: write

    steps:
      - name: Delete old container images
        uses: actions/delete-package-versions@v4
        with:
          package-name: "gocommender"
          package-type: "container"
          min-versions-to-keep: 10
          delete-only-untagged-versions: false
```

### 5. Container Usage Documentation

- [ ] Update `README.md` with container usage examples

````markdown
# GoCommender

## Quick Start with Container

### Pull from GitHub Container Registry

```bash
# Pull latest version
docker pull ghcr.io/your-username/gocommender:latest

# Pull specific version
docker pull ghcr.io/your-username/gocommender:v1.0.0
```
````

### Run Container

```bash
# Basic run
docker run -d \
  --name gocommender \
  -p 8080:8080 \
  -e PLEX_URL=http://your-plex-server:32400 \
  -e PLEX_TOKEN=your_plex_token \
  -e OPENAI_API_KEY=your_openai_key \
  -v gocommender-data:/data \
  ghcr.io/your-username/gocommender:latest

# With all environment variables
docker run -d \
  --name gocommender \
  -p 8080:8080 \
  --env-file .env \
  -v gocommender-data:/data \
  ghcr.io/your-username/gocommender:latest
```

### Docker Compose

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
      - DATABASE_PATH=/data/gocommender.db
    volumes:
      - gocommender-data:/data
    restart: unless-stopped

volumes:
  gocommender-data:
```

### Kubernetes Deployment

```bash
# Apply Kubernetes manifests
kubectl apply -f k8s/

# Or use Helm (if chart is created)
helm install gocommender ./helm/gocommender
```

## Container Features

- **Multi-architecture support**: linux/amd64, linux/arm64
- **Non-root user**: Runs as user ID 1001 for security
- **Health checks**: Built-in health check endpoint
- **Optimized size**: Multi-stage build for minimal image size
- **Security scanning**: Automated vulnerability scanning
- **Version information**: Built-in version and build info

### 6. Release Automation

- [ ] Create `.github/workflows/release.yml` for automated releases

```yaml
name: Release

on:
  release:
    types: [published]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          submodules: recursive

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=semver,pattern={{major}}
            type=raw,value=latest

      - name: Build and push release image
        uses: docker/build-push-action@v5
        with:
          context: .
          platforms: linux/amd64,linux/arm64
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          build-args: |
            VERSION=${{ github.event.release.tag_name }}
            COMMIT=${{ github.sha }}
            BUILD_DATE=${{ github.event.release.published_at }}

      - name: Update release with container info
        uses: actions/github-script@v7
        with:
          script: |
            const release = await github.rest.repos.getRelease({
              owner: context.repo.owner,
              repo: context.repo.repo,
              release_id: context.payload.release.id
            });

            const body = release.data.body + `\n\n## Container Image\n\n\`\`\`bash\ndocker pull ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.event.release.tag_name }}\n\`\`\``;

            await github.rest.repos.updateRelease({
              owner: context.repo.owner,
              repo: context.repo.repo,
              release_id: context.payload.release.id,
              body: body
            });
```

## Verification Steps

- [ ] **Local Container Build**:

  ```bash
  docker build -t gocommender:local .
  docker run -d -p 8080:8080 gocommender:local
  curl http://localhost:8080/api/info
  ```

- [ ] **Push to GHCR (after CI setup)**:

  ```bash
  git tag v1.0.0
  git push origin v1.0.0
  # Check GitHub Actions for build status
  ```

- [ ] **Pull and Test Published Image**:

  ```bash
  docker pull ghcr.io/your-username/gocommender:latest
  docker run -d -p 8080:8080 ghcr.io/your-username/gocommender:latest
  ```

- [ ] **Multi-architecture Verification**:

  ```bash
  docker manifest inspect ghcr.io/your-username/gocommender:latest
  ```

- [ ] **Security Scan Results**:

  ```bash
  # Check GitHub Security tab for Trivy scan results
  ```

## Dependencies

- Previous: `12_deployment_preparation.md` (Basic deployment setup)
- GitHub Container Registry enabled
- Repository permissions for packages

## Next Steps

This completes the GoCommender implementation plan. The application is now fully containerized and ready for production deployment through automated CI/CD pipelines.

## Notes

- **Multi-architecture builds**: Supports both AMD64 and ARM64 architectures
- **Security-first**: Non-root user, vulnerability scanning, and minimal attack surface
- **Automated publishing**: Every push to main and every release creates new container images
- **Version tracking**: Build information embedded in binary and exposed via API
- **Registry cleanup**: Automated cleanup of old container versions
- **Production ready**: Health checks, proper labeling, and optimized for container orchestration
- **Documentation**: Complete usage examples for Docker, Docker Compose, and Kubernetes
