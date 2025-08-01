# Multi-stage build for optimized container image with smart entrypoint
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary with embedded scripts
# CGO_ENABLED=0 ensures static linking
# -ldflags "-w -s" strips debug info for smaller binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags "-w -s -X main.version=${VERSION:-2.0.0} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.gitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)" \
    -o gke-image-cache-builder ./cmd

# Verify the binary is static
RUN ldd gke-image-cache-builder 2>&1 | grep -q "not a dynamic executable" || (echo "Binary is not static!" && exit 1)

# Final stage - minimal runtime image with smart entrypoint
FROM alpine:3.18

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    bash \
    curl \
    jq \
    && update-ca-certificates

# Create non-root user for security
RUN addgroup -g 1001 -S gke && \
    adduser -u 1001 -S gke -G gke

# Set working directory
WORKDIR /app

# Copy the static binary from builder stage
COPY --from=builder /app/gke-image-cache-builder .

# Copy smart entrypoint script
COPY entrypoint.sh /entrypoint.sh

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Create directories for mounted volumes
RUN mkdir -p /app/configs /app/output && \
    chown -R gke:gke /app

# Set ownership and permissions
RUN chown gke:gke /app/gke-image-cache-builder && \
    chmod +x /app/gke-image-cache-builder && \
    chmod +x /entrypoint.sh

# Switch to non-root user
USER gke

# Set environment variables
ENV PATH="/app:${PATH}"
ENV TZ=UTC

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/gke-image-cache-builder", "--version"]

# Use smart entrypoint
ENTRYPOINT ["/entrypoint.sh"]

# Default to interactive mode (will be handled by entrypoint)
CMD ["interactive"]

# Metadata labels
LABEL maintainer="GKE Image Cache Builder Team" \
      org.opencontainers.image.title="GKE Image Cache Builder" \
      org.opencontainers.image.description="Build container image cache disks for GKE node acceleration" \
      org.opencontainers.image.url="https://github.com/0x00fafa/gke-image-cache-builder" \
      org.opencontainers.image.source="https://github.com/0x00fafa/gke-image-cache-builder" \
      org.opencontainers.image.vendor="GKE Image Cache Builder" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.documentation="https://github.com/0x00fafa/gke-image-cache-builder#docker-usage"
