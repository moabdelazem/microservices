# Stage 1: Build the Go application
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod and sum files first (for better caching)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy the source code
COPY . .

# Build the application with optimizations
# CGO_ENABLED=0 for static binary
# -ldflags="-w -s" to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/tasks-service \
    ./cmd/api/main.go

# Stage 2: Create minimal runtime image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/tasks-service /app/tasks-service

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port (default 3002)
EXPOSE 3002

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-3002}/health || exit 1

# Run the application
CMD ["/app/tasks-service"]