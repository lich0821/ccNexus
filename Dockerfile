# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy module definition first for caching
COPY app/go.mod ./

# Download dependencies (go.sum will be generated after tidy)
RUN go mod download

# Copy source code
COPY app/ ./

# Ensure module graph is complete (generates go.sum)
RUN go mod tidy

# Build the headless API server (webui is embedded via go:embed)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o ccnexus-server ./cmd/server

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite-libs tzdata wget

WORKDIR /app

# Prepare data directory
RUN mkdir -p /data

# Copy binary from builder
COPY --from=builder /app/ccnexus-server /app/ccnexus-server

# Environment variables
ENV CCNEXUS_DATA_DIR=/data
ENV CCNEXUS_PORT=3000
ENV CCNEXUS_DB_PATH=/data/ccnexus.db

# Expose HTTP API port
EXPOSE 3000

# Health check targets the API health endpoint
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

# Volume for persistent data
VOLUME ["/data"]

# Run the headless server
ENTRYPOINT ["/app/ccnexus-server"]
