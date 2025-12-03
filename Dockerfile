# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for fetching dependencies
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o pako-tts ./cmd/server

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS calls to ElevenLabs API
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -g '' appuser

# Create audio cache directory
RUN mkdir -p /app/audio_cache && chown appuser:appuser /app/audio_cache

# Copy binary from builder
COPY --from=builder /app/pako-tts .

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

# Run the binary
CMD ["./pako-tts"]
