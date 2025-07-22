# Multi-stage Dockerfile for IPTW
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    gcc \
    musl-dev \
    git \
    ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -a -installsuffix cgo \
    -o iptw \
    ./cmd/iptw

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh iptw

# Set working directory
WORKDIR /home/iptw

# Copy binary from builder stage
COPY --from=builder /app/iptw /usr/local/bin/iptw

# Copy configuration and documentation
COPY --from=builder /app/config ./config/
COPY --from=builder /app/README.md ./
COPY --from=builder /app/SERVICE.md ./

# Change ownership
RUN chown -R iptw:iptw /home/iptw

# Switch to non-root user
USER iptw

# Expose any required ports (adjust as needed)
# EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD iptw --version || exit 1

# Default command
ENTRYPOINT ["/usr/local/bin/iptw"]
CMD ["--help"]
