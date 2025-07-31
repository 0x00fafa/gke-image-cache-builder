# Multi-stage build for optimized container image
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gke-image-cache-builder ./cmd

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/gke-image-cache-builder .

# Copy the setup script
COPY --from=builder /app/scripts/setup-and-verify.sh ./scripts/

# Make script executable
RUN chmod +x ./scripts/setup-and-verify.sh

# Set the binary as entrypoint
ENTRYPOINT ["./gke-image-cache-builder"]
