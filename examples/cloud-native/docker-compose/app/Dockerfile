# Multi-stage build for Kubernetes-ready application
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags '-w -s -extldflags "-static"' \
    -o main .

# Final stage - minimal image
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN adduser -D -s /bin/sh appuser

# Set working directory
WORKDIR /app

# Copy ca-certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/main .

# Change ownership to appuser
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Set environment variables
ENV PORT=8080
ENV METRICS_PORT=8081
ENV ENVIRONMENT=production
ENV LOG_LEVEL=info
ENV TZ=UTC

# Expose ports
EXPOSE 8080 8081

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8080/health/live || exit 1

# Labels for metadata
LABEL maintainer="Bolt Team" \
      version="1.0.0" \
      description="Bolt Kubernetes Demo Application" \
      org.opencontainers.image.source="https://github.com/felixgeelhaar/bolt" \
      org.opencontainers.image.documentation="https://github.com/felixgeelhaar/bolt/blob/main/examples/README.md"

# Run the application
ENTRYPOINT ["./main"]