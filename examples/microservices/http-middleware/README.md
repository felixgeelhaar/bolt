# HTTP Middleware with Bolt Logging

This example demonstrates enterprise-grade HTTP middleware integration with Bolt logging, featuring correlation IDs, structured logging, performance metrics, and error handling.

## Features

- **Correlation ID Management** - Automatic generation and propagation
- **Request/Response Logging** - Complete HTTP transaction logging
- **Performance Metrics** - Response time and throughput tracking
- **Error Handling** - Centralized error logging with panic recovery
- **Structured JSON Output** - Machine-readable log format
- **Zero-Allocation Logging** - High-performance structured logging

## Quick Start

```bash
# Install dependencies
go mod tidy

# Run the service
go run main.go

# In another terminal, test the endpoints
curl -H "X-Correlation-ID: test-123" http://localhost:8080/users
curl -X POST http://localhost:8080/users
curl http://localhost:8080/health
curl "http://localhost:8080/orders?simulate_error=true"
```

## Architecture

### Middleware Chain

The middleware is applied in a specific order for optimal functionality:

1. **Error Handling Middleware** - Catches panics and logs errors
2. **Correlation ID Middleware** - Manages request correlation
3. **Metrics Middleware** - Records performance data
4. **Business Logic Handlers** - Application-specific logic

```go
handler := service.errorHandlingMiddleware(
    service.correlationIDMiddleware(
        service.metricsMiddleware(mux),
    ),
)
```

### Logging Structure

Each log entry includes standardized fields:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "service": "user-api",
  "version": "v1.0.0",
  "environment": "production",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/users",
  "status_code": 200,
  "duration": "15ms",
  "message": "HTTP request completed"
}
```

## Endpoints

### Health Check
```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### User Management
```bash
# Get users
curl http://localhost:8080/users

# Create user
curl -X POST http://localhost:8080/users

# With custom correlation ID
curl -H "X-Correlation-ID: my-trace-123" http://localhost:8080/users
```

### Error Simulation
```bash
# Trigger error condition
curl "http://localhost:8080/orders?simulate_error=true"
```

## Configuration

### Environment Variables

- `PORT` - Server port (default: 8080)
- `ENVIRONMENT` - Deployment environment (development/staging/production)
- `LOG_LEVEL` - Logging level (debug/info/warn/error)

### Production Configuration

```bash
export ENVIRONMENT=production
export PORT=8080
export LOG_LEVEL=info
go run main.go
```

## Log Output Examples

### Successful Request
```json
{
  "timestamp": "2024-01-15T10:30:00.123Z",
  "level": "info",
  "service": "user-api",
  "version": "v1.0.0",
  "environment": "production",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/users",
  "remote_addr": "127.0.0.1:56789",
  "user_agent": "curl/7.68.0",
  "content_length": 0,
  "message": "HTTP request started"
}
```

### Error Response
```json
{
  "timestamp": "2024-01-15T10:30:00.456Z",
  "level": "error",
  "service": "user-api",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "operation": "get_orders",
  "error_type": "database_connection",
  "message": "Failed to connect to database"
}
```

### Performance Metrics
```json
{
  "timestamp": "2024-01-15T10:30:00.789Z",
  "level": "info",
  "service": "user-api",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "metric_type": "performance",
  "endpoint": "/users",
  "response_time": "15.234ms",
  "response_time_ns": 15234567,
  "response_time_ms": 15.234,
  "message": "Performance metric recorded"
}
```

## Testing

```bash
# Run tests
go test -v

# Run with race detection
go test -race -v

# Benchmark performance
go test -bench=. -benchmem

# Load testing with curl
for i in {1..100}; do
  curl -s "http://localhost:8080/users" > /dev/null &
done
wait
```

## Docker Support

```dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8080
CMD ["./main"]
```

Build and run:
```bash
docker build -t http-middleware-demo .
docker run -p 8080:8080 -e ENVIRONMENT=production http-middleware-demo
```

## Integration Patterns

### Service-to-Service Communication

```go
client := &http.Client{Timeout: 5 * time.Second}

req, _ := http.NewRequest("GET", "http://downstream-service/api", nil)
req.Header.Set("X-Correlation-ID", correlationID)

resp, err := client.Do(req)
if err != nil {
    logger.Error().
        Str("correlation_id", correlationID).
        Err(err).
        Str("service", "downstream-service").
        Msg("Service call failed")
}
```

### Database Integration

```go
func (s *Service) queryDatabase(ctx context.Context, query string) error {
    correlationID := ctx.Value("correlation_id").(string)
    start := time.Now()
    
    // Execute query
    rows, err := s.db.QueryContext(ctx, query)
    duration := time.Since(start)
    
    s.logger.Info().
        Str("correlation_id", correlationID).
        Str("operation", "database_query").
        Str("query", query).
        Dur("duration", duration).
        Err(err).
        Msg("Database operation completed")
    
    return err
}
```

### Metrics Collection

```go
// Custom metrics middleware
func (s *Service) metricsCollector(endpoint string, duration time.Duration, statusCode int) {
    s.logger.Info().
        Str("metric_type", "http_request").
        Str("endpoint", endpoint).
        Int("status_code", statusCode).
        Dur("duration", duration).
        Float64("requests_per_second", 1.0/duration.Seconds()).
        Msg("Request metrics")
}
```

## Production Considerations

### Performance
- **Zero allocations** in logging hot paths
- **Efficient JSON serialization** without reflection
- **Minimal middleware overhead** (<1μs per request)
- **Connection pooling** for downstream services

### Monitoring
- **Structured logs** compatible with log aggregation systems
- **Correlation ID propagation** across service boundaries  
- **Performance metrics** for SLA monitoring
- **Error rate tracking** for alerting

### Security
- **Input validation** in middleware
- **Request size limits** to prevent DoS
- **Timeout configuration** for all operations
- **Sensitive data masking** in logs

## Related Examples

- [gRPC Interceptors](../grpc-interceptors/) - gRPC logging patterns
- [Service Mesh](../service-mesh/) - Istio/Linkerd integration
- [Circuit Breaker](../circuit-breaker/) - Fault tolerance patterns
- [Observability Stack](../../observability/complete-stack/) - Full monitoring setup