# REST API Example

This example demonstrates using Bolt in a production-grade REST API.

## Features

- **Structured Logging**: All requests/responses logged with structured data
- **Middleware Integration**: Logging and recovery middleware
- **Error Tracking**: Comprehensive error handling and logging
- **Performance Metrics**: Request duration and status tracking
- **OpenTelemetry Support**: Trace ID correlation (when available)
- **Graceful Shutdown**: Clean server shutdown with context

## Running the Example

```bash
cd examples/rest-api
go run main.go
```

The server will start on `http://localhost:8080`

## API Endpoints

### Health Check
```bash
curl http://localhost:8080/health
```

### Get User
```bash
curl http://localhost:8080/users/123
```

### Create User
```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"email":"jane@example.com","name":"Jane Doe"}'
```

### Metrics
```bash
curl http://localhost:8080/metrics
```

## Log Output Examples

### Successful Request
```json
{"level":"info","method":"GET","path":"/users/123","remote_addr":"127.0.0.1:52156","user_agent":"curl/7.64.1","message":"request started"}
{"level":"info","user_id":"123","user_email":"john@example.com","message":"user retrieved"}
{"level":"info","method":"GET","path":"/users/123","status":200,"duration":"10.5ms","bytes":89,"message":"request completed"}
```

### Error Request
```json
{"level":"info","method":"GET","path":"/users/999","remote_addr":"127.0.0.1:52157","user_agent":"curl/7.64.1","message":"request started"}
{"level":"warn","user_id":"999","error":"user not found","message":"user not found"}
{"level":"warn","method":"GET","path":"/users/999","status":404,"duration":"10.2ms","bytes":72,"message":"request completed"}
```

### User Creation
```json
{"level":"info","method":"POST","path":"/users","remote_addr":"127.0.0.1:52158","user_agent":"curl/7.64.1","message":"request started"}
{"level":"info","user_id":"user_1672531200","user_email":"jane@example.com","user_name":"Jane Doe","message":"user created"}
{"level":"info","method":"POST","path":"/users","status":201,"duration":"0.5ms","bytes":94,"message":"request completed"}
```

## Key Implementation Details

### Middleware Pattern
```go
func (api *API) LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Extract trace context
        ctx := r.Context()
        ctxLogger := api.logger.Ctx(ctx)

        // Log request
        ctxLogger.Info().
            Str("method", r.Method).
            Str("path", r.URL.Path).
            Msg("request started")

        // Process request
        next.ServeHTTP(w, r)

        // Log response
        duration := time.Since(start)
        ctxLogger.Info().
            Int("status", statusCode).
            Dur("duration", duration).
            Msg("request completed")
    })
}
```

### Error Handling
```go
if err != nil {
    api.logger.Warn().
        Str("user_id", userID).
        Str("error", err.Error()).
        Msg("user not found")

    w.WriteHeader(http.StatusNotFound)
    json.NewEncoder(w).Encode(ErrorResponse{
        Error: "not_found",
        Code: "USER_NOT_FOUND",
        Message: fmt.Sprintf("User %s not found", userID),
    })
    return
}
```

### OpenTelemetry Integration
```go
// Extract trace context
span := trace.SpanFromContext(ctx)
if span.SpanContext().IsValid() {
    ctxLogger = api.logger.Ctx(ctx)  // Automatic trace_id/span_id injection
}
```

## Performance Characteristics

- **Zero Allocations**: All logging operations use zero heap allocations
- **Sub-100ns Latency**: Logging adds <100ns overhead per request
- **Thread-Safe**: Safe for concurrent use across multiple goroutines
- **Minimal Memory**: Event pooling reduces GC pressure

## Production Deployment Tips

1. **Use JSON Handler**: Structured logs for log aggregation systems
2. **Set Log Levels**: Use environment variables to control verbosity
3. **Add Trace IDs**: Integrate with OpenTelemetry for distributed tracing
4. **Monitor Metrics**: Track log volume and error rates
5. **Rotate Logs**: Use external log rotation (logrotate, Docker, K8s)

## Extending the Example

### Add Database Logging
```go
func (api *API) queryDB(query string) error {
    start := time.Now()

    // Execute query
    err := db.Exec(query)

    api.logger.Info().
        Str("query", query).
        Dur("duration", time.Since(start)).
        Str("database", "postgres").
        Msg("query executed")

    return err
}
```

### Add Correlation IDs
```go
func (api *API) CorrelationMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        correlationID := r.Header.Get("X-Correlation-ID")
        if correlationID == "" {
            correlationID = generateUUID()
        }

        ctx := context.WithValue(r.Context(), "correlation_id", correlationID)
        w.Header().Set("X-Correlation-ID", correlationID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Add Rate Limiting
```go
func (api *API) RateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !api.rateLimiter.Allow() {
            api.logger.Warn().
                Str("remote_addr", r.RemoteAddr).
                Msg("rate limit exceeded")

            w.WriteHeader(http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```
