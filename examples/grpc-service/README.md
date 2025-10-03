# gRPC Microservice Example

This example demonstrates using Bolt in a production gRPC microservice.

## Features

- **gRPC Interceptors**: Unary and streaming interceptor support
- **Request/Response Logging**: Comprehensive RPC call tracking
- **Error Handling**: gRPC status code mapping to log levels
- **Performance Metrics**: Duration and throughput tracking
- **OpenTelemetry Integration**: Automatic trace ID propagation
- **Panic Recovery**: Graceful panic handling with logging
- **Graceful Shutdown**: Clean server shutdown

## Running the Example

```bash
cd examples/grpc-service
go run main.go
```

The server will start on `localhost:9090`

## Log Output Examples

### Successful RPC Call
```json
{"level":"info","method":"/user.UserService/GetUser","user_agent":"grpc-go/1.50.0","message":"grpc request started"}
{"level":"info","user_id":"123","message":"getting user"}
{"level":"info","user_id":"123","user_email":"john@example.com","message":"user retrieved"}
{"level":"info","method":"/user.UserService/GetUser","code":"OK","duration":"10.5ms","message":"grpc request completed"}
```

### Error RPC Call
```json
{"level":"info","method":"/user.UserService/GetUser","user_agent":"grpc-go/1.50.0","message":"grpc request started"}
{"level":"info","user_id":"999","message":"getting user"}
{"level":"warn","user_id":"999","error":"user not found","message":"user not found"}
{"level":"warn","method":"/user.UserService/GetUser","code":"NotFound","error":"user not found","duration":"10.2ms","message":"grpc request failed"}
```

### Stream Logging
```json
{"level":"info","method":"/user.UserService/ListUsers","is_client_stream":false,"is_server_stream":true,"message":"grpc stream started"}
{"level":"info","method":"/user.UserService/ListUsers","duration":"125.3ms","message":"grpc stream completed"}
```

## Key Implementation Details

### Unary Interceptor
```go
func UnaryLoggingInterceptor(logger *bolt.Logger) grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        start := time.Now()

        // Log request
        logger.Info().
            Str("method", info.FullMethod).
            Msg("grpc request started")

        // Handle request
        resp, err := handler(ctx, req)

        // Log response with error handling
        duration := time.Since(start)
        if err != nil {
            st, _ := status.FromError(err)
            logger.Warn().
                Str("code", st.Code().String()).
                Dur("duration", duration).
                Msg("grpc request failed")
        }

        return resp, err
    }
}
```

### Recovery Interceptor
```go
func RecoveryInterceptor(logger *bolt.Logger) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
        defer func() {
            if r := recover(); r != nil {
                logger.Error().
                    Str("method", info.FullMethod).
                    Str("panic", fmt.Sprintf("%v", r)).
                    Msg("panic recovered")

                err = status.Error(codes.Internal, "internal server error")
            }
        }()
        return handler(ctx, req)
    }
}
```

### Context Logger with Tracing
```go
func (s *Server) getContextLogger(ctx context.Context) *bolt.Logger {
    span := trace.SpanFromContext(ctx)
    if span.SpanContext().IsValid() {
        return s.logger.Ctx(ctx)  // Automatic trace_id/span_id
    }
    return s.logger
}
```

## gRPC Status Code to Log Level Mapping

| gRPC Code | Log Level | Example |
|-----------|-----------|---------|
| OK | Info | Successful operation |
| InvalidArgument | Warn | Validation error |
| NotFound | Warn | Resource not found |
| Internal | Error | Server error |
| Unavailable | Error | Service down |

## Performance Characteristics

- **Zero Allocations**: All logging uses zero heap allocations
- **Sub-100ns Overhead**: Logging adds minimal latency
- **Thread-Safe**: Safe for concurrent gRPC handlers
- **Minimal Impact**: <1% overhead on RPC calls

## Production Deployment

### With OpenTelemetry
```go
import (
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

grpcServer := grpc.NewServer(
    grpc.StatsHandler(otelgrpc.NewServerHandler()),
    grpc.ChainUnaryInterceptor(
        RecoveryInterceptor(logger),
        UnaryLoggingInterceptor(logger),
    ),
)
```

### With Health Checks
```go
import (
    "google.golang.org/grpc/health"
    healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

healthServer := health.NewServer()
healthpb.RegisterHealthServer(grpcServer, healthServer)
healthServer.SetServingStatus("user.UserService", healthpb.HealthCheckResponse_SERVING)
```

### With Reflection
```go
import "google.golang.org/grpc/reflection"

reflection.Register(grpcServer)
```

## Extending the Example

### Add Client Interceptor
```go
func UnaryClientLoggingInterceptor(logger *bolt.Logger) grpc.UnaryClientInterceptor {
    return func(
        ctx context.Context,
        method string,
        req, reply interface{},
        cc *grpc.ClientConn,
        invoker grpc.UnaryInvoker,
        opts ...grpc.CallOption,
    ) error {
        start := time.Now()

        logger.Info().
            Str("method", method).
            Str("target", cc.Target()).
            Msg("grpc client request")

        err := invoker(ctx, method, req, reply, cc, opts...)

        logger.Info().
            Str("method", method).
            Dur("duration", time.Since(start)).
            Msg("grpc client response")

        return err
    }
}
```

### Add Rate Limiting
```go
func RateLimitInterceptor(limiter *rate.Limiter, logger *bolt.Logger) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        if !limiter.Allow() {
            logger.Warn().
                Str("method", info.FullMethod).
                Msg("rate limit exceeded")

            return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
        }
        return handler(ctx, req)
    }
}
```

### Add Request Validation
```go
func ValidationInterceptor(logger *bolt.Logger) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // Validate request
        if validator, ok := req.(interface{ Validate() error }); ok {
            if err := validator.Validate(); err != nil {
                logger.Warn().
                    Str("method", info.FullMethod).
                    Str("error", err.Error()).
                    Msg("validation failed")

                return nil, status.Error(codes.InvalidArgument, err.Error())
            }
        }
        return handler(ctx, req)
    }
}
```

## Monitoring Integration

### Prometheus Metrics
```go
import "github.com/grpc-ecosystem/go-grpc-prometheus"

grpcServer := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        grpc_prometheus.UnaryServerInterceptor,
        UnaryLoggingInterceptor(logger),
    ),
)

grpc_prometheus.Register(grpcServer)
```

### Custom Metrics Logging
```go
func (s *Server) logMetrics() {
    ticker := time.NewTicker(60 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        s.logger.Info().
            Int64("requests_total", requestCounter.Value()).
            Int64("errors_total", errorCounter.Value()).
            Float64("avg_duration_ms", avgDuration.Value()).
            Msg("service metrics")
    }
}
```
