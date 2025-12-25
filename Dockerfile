# Build stage
FROM golang:1.24 AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y git make gcc

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o /app/bin/meeting-assistant ./cmd/api/main.go

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/meeting-assistant .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Create logs and recordings directories
RUN mkdir -p /app/logs /tmp/recordings && chown -R appuser:appuser /app /tmp/recordings

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the application (migrations will run automatically on startup)
CMD ["./meeting-assistant"]
