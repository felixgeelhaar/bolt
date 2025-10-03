# Bolt

<div align="center">
  <img src="assets/bolt_logo.png" alt="Bolt Logo" width="300"/>
  
  [![Build Status](https://github.com/felixgeelhaar/bolt/actions/workflows/ci.yml/badge.svg)](https://github.com/felixgeelhaar/bolt/actions/workflows/ci.yml)
  [![codecov](https://codecov.io/gh/felixgeelhaar/bolt/branch/main/graph/badge.svg)](https://codecov.io/gh/felixgeelhaar/bolt)
  [![Go Version](https://img.shields.io/badge/go-%3E%3D1.19-blue.svg)](https://golang.org/)
  [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
  [![Go Report Card](https://goreportcard.com/badge/github.com/felixgeelhaar/bolt)](https://goreportcard.com/report/github.com/felixgeelhaar/bolt)
  [![Performance](https://img.shields.io/badge/performance-70ns%2Fop%20%7C%200%20allocs-brightgreen.svg)](#performance)
  [![GitHub Pages](https://img.shields.io/badge/docs-GitHub%20Pages-blue?logo=github)](https://felixgeelhaar.github.io/bolt/)
</div>

## ⚡ Zero-Allocation Structured Logging

Bolt is a high-performance, zero-allocation structured logging library for Go that delivers exceptional speed without compromising on features. Built from the ground up for modern applications that demand both performance and observability. Live benchmarks update automatically.

### 🚀 Performance First

| Library | Operation | ns/op | Allocations | Performance Advantage |
|---------|-----------|-------|-------------|----------------------|
| **Bolt v1.2.2** | Simple Log | **70** | **0** | **🏆 Industry Leading** |
| **Bolt v1.2.2** | Float64 | **66** | **0** | **✅ Zero Allocs** |
| **Bolt v1.2.2** | Complex Event | **258** | **0** | **✅ Zero Allocs** |
| Zerolog | Enabled | 175.4 | 0 | 60% slower |
| Zap | Enabled | 189.7 | 1 | 63% slower |
| Logrus | Enabled | 2,847 | 23 | 98% slower |

*Latest benchmarks on Apple M1 - v1.2.2 with race condition fixes and enhanced default logger*

## ✨ Features

- **🔥 Zero Allocations**: Achieved through intelligent event pooling, buffer reuse, and custom formatters
- **⚡ Ultra-Fast**: 70ns/op for simple logs, 66ns for Float64, 258ns for complex events
- **🏗️ Structured Logging**: Rich, type-safe field support (Int8/16/32/64, Uint, Float64, Bool, Str, etc.)
- **🔍 OpenTelemetry Integration**: Automatic trace and span ID injection
- **🎨 Multiple Outputs**: JSON for production, colorized console for development
- **🧩 Extensible**: Custom handlers and formatters
- **📦 Minimal Dependencies**: Lightweight core with optional OpenTelemetry
- **🛡️ Type Safe**: Strongly typed field methods prevent runtime errors
- **🔒 Security**: Input validation, JSON injection prevention, buffer limits

## 📦 Installation

```bash
go get github.com/felixgeelhaar/bolt
```

## 🏃 Quick Start

### Basic Usage

```go
package main

import (
    "os"
    "github.com/felixgeelhaar/bolt"
)

func main() {
    // Create a JSON logger for production
    logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    
    // Simple logging
    logger.Info().Str("service", "api").Int("port", 8080).Msg("Server starting")
    
    // Error logging with context
    if err := connectDatabase(); err != nil {
        logger.Error().
            Err(err).
            Str("component", "database").
            Msg("Failed to connect to database")
    }
}
```

### Advanced Features

```go
// Context-aware logging with OpenTelemetry
contextLogger := logger.Ctx(ctx) // Automatically includes trace/span IDs

// Structured logging with rich types
logger.Info().
    Str("user_id", "12345").
    Int("request_size", 1024).
    Bool("authenticated", true).
    Float64("processing_time", 0.234).
    Time("timestamp", time.Now()).
    Dur("timeout", 30*time.Second).
    Any("metadata", map[string]interface{}{"region": "us-east-1"}).
    Msg("Request processed")

// Create loggers with persistent context
userLogger := logger.With().
    Str("user_id", "12345").
    Str("session_id", "abc-def-ghi").
    Logger()

userLogger.Info().Msg("User action logged") // Always includes user_id and session_id
```

### Console Output for Development

```go
// Pretty console output for development
logger := bolt.New(bolt.NewConsoleHandler(os.Stdout))

logger.Info().
    Str("env", "development").
    Int("workers", 4).
    Msg("Application initialized")

// Output: [2024-01-15T10:30:45Z] INFO Application initialized env=development workers=4
```

## 🏗️ Architecture Insights

### Zero-Allocation Design

Bolt achieves zero allocations through several key innovations:

1. **Event Pooling**: Reuses event objects via `sync.Pool`
2. **Buffer Management**: Pre-allocated buffers with intelligent growth
3. **Direct Serialization**: Numbers and primitives written directly to buffers
4. **String Interning**: Efficient string handling without unnecessary copies

### Performance Optimizations

- **Fast Number Conversion**: Custom integer-to-string functions optimized for common cases
- **Allocation-Free RFC3339**: Custom timestamp formatting without `time.Format()` allocations
- **Intelligent Buffering**: Buffers sized to minimize reallocations for typical log entries
- **Branch Prediction**: Code structured to optimize for common execution paths

## 🎯 Production Usage

### Environment-Based Configuration

```go
// Automatic format selection based on environment
logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

// Set via environment variables:
// BOLT_LEVEL=info
// BOLT_FORMAT=json (production) or console (development)
```

### OpenTelemetry Integration

```go
import (
    "context"
    "go.opentelemetry.io/otel"
    "github.com/felixgeelhaar/bolt"
)

func handleRequest(ctx context.Context) {
    // Trace and span IDs automatically included
    logger := baseLogger.Ctx(ctx)
    
    logger.Info().
        Str("operation", "user.create").
        Msg("Processing user creation")
        
    // Logs will include:
    // {"level":"info","trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7","operation":"user.create","message":"Processing user creation"}
}
```

## 📊 Benchmarks

Run the included benchmarks to see Bolt's performance on your system:

```bash
go test -bench=. -benchmem ./...
```

### Sample Results (v1.2.2)

```
BenchmarkZeroAllocation-8      12,783,381    89.3 ns/op     0 B/op    0 allocs/op
BenchmarkFloat64Precision-8    19,987,876    60.1 ns/op     0 B/op    0 allocs/op
BenchmarkConsoleHandler-8       2,092,983   477.0 ns/op     0 B/op   ~10 allocs/op
BenchmarkDefaultLogger-8       15,075,518    82.6 ns/op     0 B/op    0 allocs/op
```

**Note**: ConsoleHandler currently has ~10 allocations due to string→bytes conversions in JSON parsing. This is a known limitation documented for v1.2.2 and will be addressed in v2.0 architecture redesign.

## 🛡️ Security Features

Bolt includes multiple security features to protect against common logging vulnerabilities:

### Input Validation & Sanitization

```go
// Automatic input validation prevents log injection attacks
logger.Info().
    Str("user_input", userProvidedData).  // Automatically JSON-escaped
    Msg("User data logged safely")

// Built-in size limits prevent resource exhaustion
// - Keys: max 256 characters
// - Values: max 64KB
// - Total buffer: max 1MB per log entry
```

### Thread Safety

```go
// All operations are thread-safe with atomic operations
var logger = bolt.New(bolt.NewJSONHandler(os.Stdout))

// Safe to use across multiple goroutines
go func() {
    logger.SetLevel(bolt.DEBUG) // Thread-safe level changes
}()

go func() {
    logger.Info().Msg("Concurrent logging") // Safe concurrent access
}()
```

### Error Handling

```go
// Comprehensive error handling with custom error handlers
logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
    SetErrorHandler(func(err error) {
        // Custom error handling logic
        fmt.Fprintf(os.Stderr, "Logging error: %v\n", err)
    })
```

### Security Best Practices

- **No eval() or injection vectors**: All data is properly escaped during JSON serialization
- **Memory safety**: Buffer size limits prevent unbounded memory usage
- **Structured output**: JSON format prevents log format injection
- **Controlled serialization**: Type-safe field methods prevent data corruption

## 🔧 Custom Handlers

Extend Bolt with custom output formats:

```go
type CustomHandler struct {
    output io.Writer
}

func (h *CustomHandler) Write(e *bolt.Event) error {
    // Custom formatting logic
    formatted := customFormat(e)
    _, err := h.output.Write(formatted)
    return err
}

logger := bolt.New(&CustomHandler{output: os.Stdout})
```

## 🔍 Troubleshooting

### Common Issues and Solutions

#### Performance Issues

**Symptom**: Logging is slower than expected
```bash
# Check if you're in debug mode accidentally
echo $BOLT_LEVEL  # Should be 'info' or 'warn' for production

# Run benchmarks to compare
go test -bench=BenchmarkZeroAllocation -benchmem
```

**Symptom**: Memory usage is high
```go
// Ensure you're calling Msg() to complete log entries
logger.Info().Str("key", "value")  // ❌ Event not completed
logger.Info().Str("key", "value").Msg("message")  // ✅ Proper completion

// Check for event leaks in error handling
if err != nil {
    // ❌ This leaks events if err is always nil
    logger.Error().Err(err).Msg("error occurred")  
}

if err != nil {
    // ✅ Proper conditional logging
    logger.Error().Err(err).Msg("error occurred")
}
```

#### Thread Safety Issues

**Symptom**: Race conditions detected
```bash
# Run tests with race detector
go test -race ./...

# The library itself is thread-safe, but output destinations may not be
# Use thread-safe output for concurrent scenarios
```

**Solution**: Use thread-safe outputs
```go
// ❌ bytes.Buffer is not thread-safe
var buf bytes.Buffer
logger := bolt.New(bolt.NewJSONHandler(&buf))

// ✅ Use thread-safe alternatives
type SafeBuffer struct {
    buf bytes.Buffer
    mu  sync.Mutex
}

func (sb *SafeBuffer) Write(p []byte) (n int, err error) {
    sb.mu.Lock()
    defer sb.mu.Unlock()
    return sb.buf.Write(p)
}
```

#### Configuration Issues

**Symptom**: Logs not appearing
```go
// Check log level configuration
logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
logger.SetLevel(bolt.ERROR)  // Will suppress Info/Debug logs

logger.Debug().Msg("Debug message")  // Won't appear
logger.Error().Msg("Error message")  // Will appear
```

**Symptom**: Wrong output format
```bash
# Check environment variables
echo $BOLT_FORMAT  # Should be 'json' or 'console'
echo $BOLT_LEVEL   # Should be valid level name

# Override with code if needed
logger := bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG)
```

#### Integration Issues

**Symptom**: OpenTelemetry traces not appearing
```go
// Ensure context contains valid span
span := trace.SpanFromContext(ctx)
if !span.SpanContext().IsValid() {
    // No active span in context
    logger.Info().Msg("No trace context")
}

// Use context-aware logger
ctxLogger := logger.Ctx(ctx)
ctxLogger.Info().Msg("With trace context")
```

#### Performance Debugging

```bash
# Profile memory usage
go test -bench=BenchmarkZeroAllocation -memprofile=mem.prof
go tool pprof mem.prof

# Profile CPU usage  
go test -bench=BenchmarkZeroAllocation -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Check for allocations
go test -bench=. -benchmem | grep allocs
```

### Getting Help

1. **Check the documentation**: Review API documentation and examples
2. **Run diagnostics**: Use built-in benchmarks and race detection
3. **Community support**: Open GitHub issues with minimal reproduction cases
4. **Security issues**: Follow responsible disclosure in [SECURITY.md](SECURITY.md)

## ⚠️ Limitations & Considerations

### When Bolt is Ideal

✅ **Perfect for:**
- High-throughput APIs (>10k req/sec)
- Microservices with strict latency requirements
- Applications requiring GC-friendly logging
- Production systems where performance is critical
- Containerized/cloud-native deployments

### Known Limitations

❌ **Not suitable for:**
- **Real-time systems with <10µs latency requirements** - Bolt's 60-300ns overhead may be significant
- **Extreme memory constraints (<1MB heap)** - Event pool requires ~8-64KB overhead
- **Non-Go applications** - Bolt is Go-specific
- **Legacy log parsers** - Requires JSON-compatible log processors

### Performance Characteristics

| Scenario | Overhead | Notes |
|----------|----------|-------|
| Simple log (3 fields) | ~70ns | 0 allocations |
| Complex log (10 fields) | ~300ns | 0 allocations |
| Console handler | ~150ns | 12 allocations (acceptable for dev) |
| HTTP request logging | ~0.01% CPU | Negligible impact |
| High throughput (100k/sec) | <1% CPU | Scales linearly |

**Memory Profile:**
- Event pool: 8-64KB (steady state)
- Per-event: ~336 bytes (pooled, reused)
- Production API (50k req/sec): ~2-4MB total

See [PERFORMANCE.md](PERFORMANCE.md) for detailed benchmarks and optimization guide.

## 🚀 Deployment Guide

### Production Deployment

#### 1. Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bolt-app
spec:
  template:
    spec:
      containers:
      - name: app
        image: myapp:latest
        env:
        - name: LOG_FORMAT
          value: "json"
        - name: LOG_LEVEL
          value: "info"
        - name: OTEL_EXPORTER_OTLP_ENDPOINT
          value: "http://otel-collector:4317"
```

📖 **Full example**: See [examples/kubernetes/](examples/kubernetes/)

#### 2. Cloud Platforms

**AWS Lambda:**
```go
logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).With().
    Str("function", os.Getenv("AWS_LAMBDA_FUNCTION_NAME")).
    Str("region", os.Getenv("AWS_REGION")).
    Logger()
```

**Google Cloud Run:**
```go
logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).With().
    Str("service", os.Getenv("K_SERVICE")).
    Str("revision", os.Getenv("K_REVISION")).
    Logger()
```

**Azure Functions:**
```go
logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).With().
    Str("function", os.Getenv("WEBSITE_SITE_NAME")).
    Logger()
```

📖 **Cloud guides**: [AWS](docs/cloud/aws.md) | [GCP](docs/cloud/gcp.md) | [Azure](docs/cloud/azure.md)

#### 3. Framework Integration

**Gin:**
```go
r.Use(BoltLogger(logger))
```

**Echo:**
```go
e.Use(BoltLoggerWithTracing(logger))
```

**Fiber:**
```go
app.Use(BoltLogger(logger))
```

**Chi:**
```go
r.Use(BoltLogger(logger))
```

📖 **Integration guides**: [Gin](docs/integrations/gin.md) | [Echo](docs/integrations/echo.md) | [Fiber](docs/integrations/fiber.md) | [Chi](docs/integrations/chi.md)

### Production Checklist

- [ ] Use JSON handler (zero allocations)
- [ ] Set appropriate log level (info/warn for production)
- [ ] Configure OpenTelemetry for distributed tracing
- [ ] Set up log aggregation (Fluentd, Fluent Bit, etc.)
- [ ] Enable monitoring (Prometheus, Grafana)
- [ ] Configure health checks and metrics endpoints
- [ ] Test graceful shutdown and log flushing
- [ ] Validate security (no sensitive data in logs)
- [ ] Set resource limits (memory, CPU)
- [ ] Enable alerting on error rates

📖 **Production examples**: See [examples/](examples/) directory

## 📚 Documentation

### Core Documentation
- [📖 **Full Documentation**](docs/README.md) - Complete documentation index
- [🚀 **Development Guide**](docs/DEVELOPMENT.md) - Setup and development workflow
- [🏢 **Enterprise Guide**](docs/ENTERPRISE.md) - Enterprise deployment and scaling
- [⚡ **Performance Guide**](PERFORMANCE.md) - Honest benchmarks and optimization
- [🔧 **Troubleshooting**](docs/TROUBLESHOOTING.md) - Common issues and solutions

### Integration & Deployment
- [☁️ **Cloud Platforms**](docs/cloud/) - AWS, GCP, Azure integration guides
- [🔌 **Framework Integration**](docs/integrations/) - Gin, Echo, Fiber, Chi guides
- [📊 **Monitoring**](examples/monitoring/) - Prometheus, Grafana, DataDog setup
- [🎯 **Production Examples**](examples/) - REST API, gRPC, Batch processing, K8s

### Community Guidelines
- [🤝 **Contributing**](CONTRIBUTING.md) - How to contribute to Bolt
- [🛡️ **Security Policy**](SECURITY.md) - Security vulnerability reporting
- [📜 **Code of Conduct**](CODE_OF_CONDUCT.md) - Community standards

## 🤝 Contributing

We welcome contributions! **Please fork the repository** and submit pull requests from your fork.

### Quick Start for Contributors

1. **Fork** this repository on GitHub
2. **Clone** your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/bolt.git
   cd bolt
   ```
3. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```
4. **Make your changes** and ensure tests pass:
   ```bash
   go test ./...
   go test -bench=. -benchmem -tags=bench
   ```
5. **Submit a pull request** from your fork

📖 **Detailed guidelines**: See [CONTRIBUTING.md](CONTRIBUTING.md) for complete contribution workflow, coding standards, and performance requirements.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🎖️ Recognition

Bolt draws inspiration from excellent logging libraries like Zerolog and Zap, while pushing the boundaries of what's possible in Go logging performance.

---

<div align="center">
  <img src="assets/bolt_logo.png" alt="Bolt Icon" width="64"/>
  
  **Built with ❤️ for high-performance Go applications**
</div>
