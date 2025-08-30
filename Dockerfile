# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o monitor-agent ./cmd/monitor-agent

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S monitor && \
    adduser -u 1001 -S monitor -G monitor

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/monitor-agent .

# Copy migrations
COPY --from=builder /app/internal/database/migrations ./migrations

# Change ownership to non-root user
RUN chown -R monitor:monitor /app

# Switch to non-root user
USER monitor

# Expose port (if needed for health checks)
EXPOSE 8080

# Set environment variables
ENV GO_ENV=production

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ./monitor-agent health || exit 1

# Default command
ENTRYPOINT ["./monitor-agent"]
