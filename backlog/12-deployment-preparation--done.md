# 12 - Deployment Preparation

## Overview
Set up CI/CD pipeline, containerization, and deployment configurations following llm-shared guidelines with GitHub Actions and Docker support.

## Steps

### 1. GitHub Actions CI/CD Pipeline

- [ ] Create `.github/workflows/ci.yml` with comprehensive CI/CD pipeline

```yaml
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

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
        go-version: '1.21'
    
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
    
    - name: Run tests
      run: task test-ci
    
    - name: Build
      run: task build-ci
    
    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
        flags: unittests
        name: codecov-umbrella

  build-docker:
    runs-on: ubuntu-latest
    needs: test
    
    steps:
    - uses: actions/checkout@v4
      with:
        submodules: recursive
    
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v2
    
    - name: Login to GitHub Container Registry
      if: github.event_name != 'pull_request'
      uses: docker/login-action@v2
      with:
        registry: ghcr.io
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}
    
    - name: Extract metadata
      id: meta
      uses: docker/metadata-action@v4
      with:
        images: ghcr.io/${{ github.repository }}
        tags: |
          type=ref,event=branch
          type=ref,event=pr
          type=sha,prefix={{branch}}-
          type=raw,value=latest,enable={{is_default_branch}}
    
    - name: Build and push
      uses: docker/build-push-action@v4
      with:
        context: .
        push: ${{ github.event_name != 'pull_request' }}
        tags: ${{ steps.meta.outputs.tags }}
        labels: ${{ steps.meta.outputs.labels }}
        cache-from: type=gha
        cache-to: type=gha,mode=max

  security:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v4
      with:
        submodules: recursive
    
    - name: Run Gosec Security Scanner
      uses: securecodewarrior/github-action-gosec@master
      with:
        args: '-severity medium ./...'
    
    - name: Run Trivy vulnerability scanner
      uses: aquasecurity/trivy-action@master
      with:
        scan-type: 'fs'
        scan-ref: '.'
        format: 'sarif'
        output: 'trivy-results.sarif'
    
    - name: Upload Trivy scan results to GitHub Security tab
      uses: github/codeql-action/upload-sarif@v2
      if: always()
      with:
        sarif_file: 'trivy-results.sarif'
```

### 2. Docker Configuration

- [ ] Create `Dockerfile` with multi-stage build and security best practices

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gocommender ./cmd/server

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S gocommender && \
    adduser -u 1001 -S gocommender -G gocommender

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/gocommender .

# Copy configuration files
COPY --from=builder /app/.env.example .env.example

# Change ownership
RUN chown -R gocommender:gocommender /app

# Switch to non-root user
USER gocommender

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

# Run the application
CMD ["./gocommender"]
```

### 3. Docker Compose for Development

- [ ] Create `docker-compose.yml` for local development and testing

```yaml
version: '3.8'

services:
  gocommender:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PLEX_URL=${PLEX_URL}
      - PLEX_TOKEN=${PLEX_TOKEN}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - DISCOGS_TOKEN=${DISCOGS_TOKEN}
      - LASTFM_API_KEY=${LASTFM_API_KEY}
      - DATABASE_PATH=/data/gocommender.db
      - HOST=0.0.0.0
      - PORT=8080
    volumes:
      - ./data:/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  # Optional: Add a reverse proxy for production
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - gocommender
    restart: unless-stopped
    profiles:
      - production

volumes:
  data:
    driver: local
```

### 4. Kubernetes Deployment

- [ ] Create `k8s/deployment.yaml` with production-ready Kubernetes manifests

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gocommender
  labels:
    app: gocommender
spec:
  replicas: 2
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
        image: ghcr.io/your-org/gocommender:latest
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
              key: openai-api-key
        - name: DISCOGS_TOKEN
          valueFrom:
            secretKeyRef:
              name: gocommender-secrets
              key: discogs-token
              optional: true
        - name: LASTFM_API_KEY
          valueFrom:
            secretKeyRef:
              name: gocommender-secrets
              key: lastfm-api-key
              optional: true
        - name: DATABASE_PATH
          value: "/data/gocommender.db"
        - name: HOST
          value: "0.0.0.0"
        - name: PORT
          value: "8080"
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
          periodSeconds: 5
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: gocommender-data

---
apiVersion: v1
kind: Service
metadata:
  name: gocommender-service
spec:
  selector:
    app: gocommender
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: gocommender-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi

---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gocommender-ingress
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - api.gocommender.example.com
    secretName: gocommender-tls
  rules:
  - host: api.gocommender.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: gocommender-service
            port:
              number: 80
```

### 5. Environment Configuration

- [ ] Create multiple environment files for different deployment stages

`environments/.env.development`:
```env
# Development environment
PLEX_URL=http://localhost:32400
PLEX_TOKEN=your_dev_token
OPENAI_API_KEY=your_dev_openai_key
DISCOGS_TOKEN=your_dev_discogs_token
LASTFM_API_KEY=your_dev_lastfm_key
DATABASE_PATH=./data/dev.db
HOST=localhost
PORT=8080
CACHE_TTL_SUCCESS=1h
CACHE_TTL_FAILURE=10m
```

`environments/.env.staging`:
```env
# Staging environment
PLEX_URL=https://staging-plex.example.com
PLEX_TOKEN=${STAGING_PLEX_TOKEN}
OPENAI_API_KEY=${STAGING_OPENAI_API_KEY}
DISCOGS_TOKEN=${STAGING_DISCOGS_TOKEN}
LASTFM_API_KEY=${STAGING_LASTFM_API_KEY}
DATABASE_PATH=/data/staging.db
HOST=0.0.0.0
PORT=8080
CACHE_TTL_SUCCESS=12h
CACHE_TTL_FAILURE=1h
```

`environments/.env.production`:
```env
# Production environment
PLEX_URL=${PRODUCTION_PLEX_URL}
PLEX_TOKEN=${PRODUCTION_PLEX_TOKEN}
OPENAI_API_KEY=${PRODUCTION_OPENAI_API_KEY}
DISCOGS_TOKEN=${PRODUCTION_DISCOGS_TOKEN}
LASTFM_API_KEY=${PRODUCTION_LASTFM_API_KEY}
DATABASE_PATH=/data/production.db
HOST=0.0.0.0
PORT=8080
CACHE_TTL_SUCCESS=720h
CACHE_TTL_FAILURE=168h
```

### 6. Monitoring and Observability

- [ ] Create `monitoring/prometheus.yml` and add metrics endpoint

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'gocommender'
    static_configs:
      - targets: ['gocommender:8080']
    metrics_path: /api/metrics
    scrape_interval: 30s

  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']
```

Add metrics endpoint to API (update `internal/api/router.go`):

```go
// Add to setupRoutes()
s.mux.HandleFunc("/api/metrics", s.handleMetrics)

// handleMetrics provides Prometheus metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    stats, err := s.cache.GetStats()
    if err != nil {
        writeErrorResponse(w, "Failed to get metrics", http.StatusInternalServerError)
        return
    }
    
    // Simple metrics format (in production, use Prometheus client library)
    metrics := fmt.Sprintf(`
# HELP gocommender_cache_entries Total cache entries
# TYPE gocommender_cache_entries gauge
gocommender_cache_entries %d

# HELP gocommender_cache_hit_rate Cache hit rate
# TYPE gocommender_cache_hit_rate gauge
gocommender_cache_hit_rate %.2f

# HELP gocommender_cache_expired_entries Expired cache entries
# TYPE gocommender_cache_expired_entries gauge
gocommender_cache_expired_entries %d
`, stats.TotalEntries, stats.HitRate, stats.ExpiredEntries)
    
    w.Header().Set("Content-Type", "text/plain")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(metrics))
}
```

### 7. Deployment Scripts

- [ ] Create `scripts/deploy.sh` for automated deployment

```bash
#!/bin/bash
set -e

# Deployment script for GoCommender

ENVIRONMENT=${1:-staging}
VERSION=${2:-latest}

echo "Deploying GoCommender version $VERSION to $ENVIRONMENT"

# Validate environment
case $ENVIRONMENT in
  development|staging|production)
    echo "Deploying to $ENVIRONMENT environment"
    ;;
  *)
    echo "Invalid environment: $ENVIRONMENT"
    echo "Usage: $0 <environment> [version]"
    echo "Environments: development, staging, production"
    exit 1
    ;;
esac

# Load environment variables
if [ -f "environments/.env.$ENVIRONMENT" ]; then
    set -a
    source "environments/.env.$ENVIRONMENT"
    set +a
else
    echo "Environment file not found: environments/.env.$ENVIRONMENT"
    exit 1
fi

# Docker deployment
if command -v docker-compose &> /dev/null; then
    echo "Deploying with Docker Compose..."
    
    # Pull latest image
    docker pull "ghcr.io/your-org/gocommender:$VERSION"
    
    # Update docker-compose with version
    sed -i.bak "s|image: ghcr.io/your-org/gocommender:.*|image: ghcr.io/your-org/gocommender:$VERSION|" docker-compose.yml
    
    # Deploy
    docker-compose down
    docker-compose up -d
    
    # Wait for health check
    echo "Waiting for service to be healthy..."
    timeout 60 bash -c 'until curl -f http://localhost:8080/api/health; do sleep 5; done'
    
    echo "Deployment successful!"
    
# Kubernetes deployment
elif command -v kubectl &> /dev/null; then
    echo "Deploying with Kubernetes..."
    
    # Update image in deployment
    kubectl set image deployment/gocommender gocommender="ghcr.io/your-org/gocommender:$VERSION"
    
    # Wait for rollout
    kubectl rollout status deployment/gocommender --timeout=300s
    
    echo "Deployment successful!"
else
    echo "Neither docker-compose nor kubectl found"
    exit 1
fi

# Run post-deployment checks
echo "Running post-deployment checks..."
./scripts/health-check.sh

echo "Deployment complete!"
```

### 8. Health Check Script

- [ ] Create `scripts/health-check.sh` for post-deployment validation

```bash
#!/bin/bash
set -e

# Health check script for GoCommender

BASE_URL=${1:-http://localhost:8080}
MAX_RETRIES=${2:-10}
RETRY_DELAY=${3:-5}

echo "Performing health checks for $BASE_URL"

# Function to check endpoint
check_endpoint() {
    local endpoint=$1
    local expected_status=${2:-200}
    
    echo -n "Checking $endpoint... "
    
    for i in $(seq 1 $MAX_RETRIES); do
        response=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL$endpoint" || echo "000")
        
        if [ "$response" = "$expected_status" ]; then
            echo "✓ OK ($response)"
            return 0
        fi
        
        if [ $i -lt $MAX_RETRIES ]; then
            echo -n "retrying ($i/$MAX_RETRIES)... "
            sleep $RETRY_DELAY
        fi
    done
    
    echo "✗ FAILED ($response)"
    return 1
}

# Check critical endpoints
check_endpoint "/api/health"
check_endpoint "/api/info"
check_endpoint "/api/cache/stats"

# Check Plex integration (may fail if not configured)
if check_endpoint "/api/plex/test" || [ "$?" = "1" ]; then
    echo "Note: Plex test endpoint check completed (may require configuration)"
fi

echo "Health checks completed successfully!"
```

## Verification Steps

- [ ] **Local Build**:
   ```bash
   task build
   ./build/gocommender -config-test
   ```

- [ ] **Docker Build**:
   ```bash
   docker build -t gocommender:local .
   docker run -p 8080:8080 gocommender:local
   ```

- [ ] **Docker Compose**:
   ```bash
   docker-compose up -d
   curl http://localhost:8080/api/health
   ```

- [ ] **Deployment Script**:
   ```bash
   chmod +x scripts/deploy.sh scripts/health-check.sh
   ./scripts/deploy.sh development
   ```

- [ ] **CI Pipeline**:
   ```bash
   # Triggered automatically on git push
   git add .
   git commit -m "Add deployment configuration"
   git push
   ```

## Dependencies
- Previous: `11_testing_strategy.md` (Testing infrastructure)
- Docker for containerization
- GitHub Actions for CI/CD
- Kubernetes for orchestration (optional)

## Next Steps
This completes the full GoCommender implementation plan. The system is now ready for development following the step-by-step implementation of each plan file.

## Notes
- Multi-stage Docker build for optimal image size
- Non-root container user for security
- Health checks and monitoring integration
- Environment-specific configurations
- Automated CI/CD pipeline with security scanning
- Kubernetes deployment with proper resource limits
- Prometheus metrics endpoint for observability
- Comprehensive deployment and health check scripts
- Proper secret management for sensitive data
- Container registry integration with GitHub Packages
- Rollback capabilities through versioned deployments