# GoCommender Deployment Guide

## Quick Start with Docker

### Prerequisites
- Docker and Docker Compose installed
- API tokens for required services

### Environment Setup

1. Copy the environment template:
```bash
cp .env.example .env
```

2. Edit `.env` with your actual API tokens:
```bash
# Required
PLEX_URL=http://your-plex-server:32400
PLEX_TOKEN=your-plex-token
OPENAI_API_KEY=your-openai-key

# Optional (for enhanced metadata)
DISCOGS_TOKEN=your-discogs-token
LASTFM_API_KEY=your-lastfm-key
```

### Deploy with Docker Compose

```bash
# Build and start the service
docker-compose up -d

# Check logs
docker-compose logs -f

# Health check
curl http://localhost:8080/health
```

## Docker Deployment Options

### Option 1: Docker Compose (Recommended)

```bash
docker-compose up -d
```

This provides:
- Automatic container restart
- Volume persistence for database
- Health checks
- Environment variable management

### Option 2: Docker Run

```bash
# Build image
docker build -t gocommender .

# Run container
docker run -d \
  --name gocommender \
  -p 8080:8080 \
  -v gocommender_data:/app/data \
  -e GOCOMMENDER_PLEX_URL="$PLEX_URL" \
  -e GOCOMMENDER_PLEX_TOKEN="$PLEX_TOKEN" \
  -e GOCOMMENDER_OPENAI_API_KEY="$OPENAI_API_KEY" \
  gocommender
```

## Production Considerations

### Security
- Use secrets management for API tokens (Docker secrets, K8s secrets, etc.)
- Run containers as non-root user (already configured)
- Use HTTPS in production with reverse proxy
- Keep API tokens out of container images

### Performance
- Mount persistent volume for SQLite database
- Configure proper resource limits
- Use health checks for container orchestration
- Monitor API rate limits

### Scaling
- Single instance recommended (SQLite limitation)
- For high availability, consider:
  - Database migration to PostgreSQL
  - Stateless architecture with external cache
  - Load balancer with session affinity

## Kubernetes Deployment

### ConfigMap for Environment
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gocommender-config
data:
  GOCOMMENDER_SERVER_HOST: "0.0.0.0"
  GOCOMMENDER_SERVER_PORT: "8080"
  GOCOMMENDER_DATABASE_PATH: "/app/data/gocommender.db"
  GOCOMMENDER_LOG_LEVEL: "info"
```

### Secret for API Tokens
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gocommender-secrets
type: Opaque
stringData:
  GOCOMMENDER_PLEX_URL: "http://your-plex-server:32400"
  GOCOMMENDER_PLEX_TOKEN: "your-plex-token"
  GOCOMMENDER_OPENAI_API_KEY: "your-openai-key"
  GOCOMMENDER_DISCOGS_TOKEN: "your-discogs-token"
  GOCOMMENDER_LASTFM_API_KEY: "your-lastfm-key"
```

### Deployment
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
        image: gocommender:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: gocommender-config
        - secretRef:
            name: gocommender-secrets
        volumeMounts:
        - name: data
          mountPath: /app/data
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: gocommender-data
```

## Monitoring

### Health Endpoints
- `GET /health` - Application health status
- `GET /ready` - Readiness check (if implemented)

### Logs
- Application logs to stdout/stderr
- Structured JSON logging in production
- Log levels: debug, info, warn, error

### Metrics
- Consider adding Prometheus metrics endpoint
- Monitor API response times
- Track cache hit/miss ratios
- Monitor external API rate limits

## Troubleshooting

### Common Issues

1. **Container fails to start**
   ```bash
   # Check logs
   docker-compose logs gocommender
   
   # Verify environment variables
   docker-compose exec gocommender env | grep GOCOMMENDER
   ```

2. **Database connection issues**
   ```bash
   # Check data volume permissions
   docker-compose exec gocommender ls -la /app/data
   
   # Verify SQLite file creation
   docker-compose exec gocommender ls -la /app/data/gocommender.db
   ```

3. **API token issues**
   ```bash
   # Test configuration
   docker-compose exec gocommender ./gocommender -config-test
   ```

4. **Health check failures**
   ```bash
   # Manual health check
   curl -v http://localhost:8080/health
   
   # Check container status
   docker-compose ps
   ```

### Debug Mode

```bash
# Run with debug logging
GOCOMMENDER_LOG_LEVEL=debug docker-compose up
```

## Backup and Recovery

### Database Backup
```bash
# Create backup
docker-compose exec gocommender cp /app/data/gocommender.db /app/data/backup-$(date +%Y%m%d).db

# Copy to host
docker cp $(docker-compose ps -q gocommender):/app/data/backup-$(date +%Y%m%d).db ./backup.db
```

### Restore Database
```bash
# Copy backup to container
docker cp ./backup.db $(docker-compose ps -q gocommender):/app/data/gocommender.db

# Restart service
docker-compose restart gocommender
```