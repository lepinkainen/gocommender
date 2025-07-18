# syntax=docker/dockerfile:1.4

# Build stage
FROM golang:1.24-alpine AS builder

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
ARG TARGETARCH

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
RUN mkdir -p /app/data && chown gocommender:gocommender /app/data

# Copy binary from builder stage
COPY --from=builder /app/gocommender .

# Copy prompts directory
COPY --from=builder /app/prompts ./prompts

# Note: Configuration is provided via environment variables at runtime

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