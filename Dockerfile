# =============================================================================
# 12306 Train Ticket Monitor - Dockerfile
# =============================================================================
# Build:   docker build -t cn-rail-monitor:latest .
# Run:     docker run -d -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml:ro cn-rail-monitor:latest
# =============================================================================

# -----------------------------------------------------------------------------
# Build Stage
# -----------------------------------------------------------------------------
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cn-rail-monitor ./cmd

# -----------------------------------------------------------------------------
# Production Stage
# -----------------------------------------------------------------------------
FROM alpine:latest

# Install CA certificates for HTTPS
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user for security
RUN adduser -D -u 1000 appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/cn-rail-monitor .
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Default configuration file path
ENV CONFIG_PATH=/app/config.yaml

# Expose HTTP port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/cn-rail-monitor"]
CMD ["-config", "/app/config.yaml"]
