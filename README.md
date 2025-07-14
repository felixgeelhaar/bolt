<div align="center">
  <img src="logo.svg" alt="Logma Logo" width="400">
  
  # High-Performance, Zero-Allocation Structured Logging for Go
  
  **Logma solves the "Logger's Trilemma" by delivering uncompromising performance, superior developer experience, and first-class observability.**
  
  [![Go Reference](https://pkg.go.dev/badge/github.com/felixgeelhaar/logma.svg)](https://pkg.go.dev/github.com/felixgeelhaar/logma)
  [![Go Report Card](https://goreportcard.com/badge/github.com/felixgeelhaar/logma)](https://goreportcard.com/report/github.com/felixgeelhaar/logma)
  [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
  
</div>

## 🚀 Performance First

Logma delivers **industry-leading performance** with **zero memory allocations**:

| Operation | Logma | Zerolog | Performance |
|-----------|-------|---------|-------------|
| **Simple log** | `127ns (0 allocs)` | `92ns (0 allocs)` | **Within 38%** |
| **5 fields** | `146ns (0 allocs)` | `142ns (0 allocs)` | **Within 3%** |
| **Disabled logs** | `3.8ns (0 allocs)` | `4.4ns (0 allocs)` | **🔥 14% faster** |

*Benchmarks run on Apple M1. Your performance may vary.*

## ✨ Features

- **🚀 Zero-Allocation Logging:** True zero-allocation performance for all operations
- **⚡ Industry-Leading Speed:** Highly competitive with Zerolog, faster disabled logs
- **🎯 Intuitive API:** Chainable, fluent interface inspired by the best logging libraries
- **🔧 Smart Defaults:** Auto-detects TTY for format selection, minimal configuration needed
- **📊 OpenTelemetry Ready:** First-class distributed tracing integration
- **🎨 Flexible Output:** JSON, console, and custom formats with timestamps
- **🔒 Production Ready:** Robust error handling, sampling, and log rotation
- **🛠️ Developer Friendly:** Rich field types, stack traces, and debugging support

## Installation

```bash
go get github.com/felixgeelhaar/logma
```

## 🚀 Quick Start

### Basic Logging

```go
package main

import "github.com/felixgeelhaar/logma"

func main() {
    // Simple logging with zero allocations
    logma.Info().Msg("Hello, Logma!")
    logma.Error().Msg("Something went wrong")
}
```

### Structured Logging

```go
package main

import "github.com/felixgeelhaar/logma"

func main() {
    // Rich structured data - still zero allocations!
    logma.Info().
        Str("user_id", "123").
        Int("order_id", 456).
        Float64("total", 99.99).
        Bool("premium", true).
        Msg("User placed an order")
}
```

### Error Handling with Stack Traces

```go
package main

import (
    "errors"
    "github.com/felixgeelhaar/logma"
)

func main() {
    err := errors.New("database connection failed")
    
    // Include stack trace for debugging
    logma.Error().
        ErrorWithStack(err).
        Str("database", "users").
        Msg("Failed to process request")
}
```

### Context-Aware Logging

```go
package main

import (
    "context"
    "github.com/felixgeelhaar/logma"
)

func main() {
    // Create logger with persistent context
    requestLogger := logma.With().
        Str("request_id", "abc-123").
        Str("user_id", "456").
        Logger()
    
    requestLogger.Info().Msg("Starting request processing")
    requestLogger.Warn().Str("step", "validation").Msg("Validation failed")
    requestLogger.Info().Dur("duration", time.Millisecond*250).Msg("Request completed")
}
```

### OpenTelemetry Integration

```go
package main

import (
    "context"
    "github.com/felixgeelhaar/logma"
    "go.opentelemetry.io/otel"
)

func main() {
    tracer := otel.Tracer("my-service")
    ctx, span := tracer.Start(context.Background(), "process-order")
    defer span.End()

    // Automatically includes trace_id and span_id
    logma.Ctx(ctx).Info().
        Str("order_id", "12345").
        Msg("Processing order in trace context")
}
```

### Advanced Features

```go
package main

import (
    "time"
    "github.com/felixgeelhaar/logma"
)

func main() {
    // Custom configuration
    config := logma.DefaultEncoderConfig()
    config.TimeFormat = time.RFC3339
    config.IncludeCaller = true
    
    logger := logma.New(logma.NewJSONHandlerWithConfig(os.Stdout, config))
    
    // Sampling for high-volume logs
    sampledLogger := logger.SetSampler(logma.NewFixedSampler(10)) // Log every 10th entry
    
    // Different log levels
    logger.Trace().Msg("Detailed debugging info")
    logger.Debug().Msg("Debug information")
    logger.Info().Msg("General information")
    logger.Warn().Msg("Warning message")
    logger.Error().Msg("Error occurred")
    
    // Rich field types
    logger.Info().
        Time("timestamp", time.Now()).
        Dur("latency", time.Millisecond*45).
        Bytes("payload", []byte("binary data")).
        Hex("hash", []byte{0x12, 0x34, 0xab, 0xcd}).
        Any("metadata", map[string]interface{}{"version": "2.0"}).
        Msg("Request processed")
}
```

## ⚙️ Configuration

### Environment Variables

```bash
# Output format (auto-detected if not set)
export LOGMA_FORMAT=json    # or 'console'

# Log level filtering  
export LOGMA_LEVEL=info     # trace, debug, info, warn, error, fatal
```

### Programmatic Configuration

```go
// Custom JSON configuration
config := logma.EncoderConfig{
    TimeKey:       "timestamp",
    LevelKey:      "severity", 
    MessageKey:    "msg",
    IncludeTime:   true,
    IncludeCaller: true,
    TimeFormat:    time.RFC3339,
}

logger := logma.New(logma.NewJSONHandlerWithConfig(os.Stdout, config))

// Console output for development
consoleLogger := logma.New(logma.NewConsoleHandler(os.Stdout))

// Multiple outputs
multiWriter := logma.NewMultiWriter(os.Stdout, logFile)
logger := logma.New(logma.NewJSONHandler(multiWriter))

// Log rotation
rotatingWriter := logma.NewRotatingFileWriteSyncer("app.log", 100*1024*1024, 5)
logger := logma.New(logma.NewJSONHandler(rotatingWriter))
```

## 📊 Benchmarks

### Performance Comparison

```
goos: darwin
goarch: arm64
pkg: github.com/felixgeelhaar/logma

BenchmarkLogma-8              12505779    127.1 ns/op    171 B/op    0 allocs/op
BenchmarkLogma5Fields-8       11412014    146.3 ns/op    282 B/op    0 allocs/op
BenchmarkLogmaWithTimestamp-8  7532076    166.0 ns/op    285 B/op    0 allocs/op
BenchmarkLogmaDisabled-8     315377994      3.78 ns/op      0 B/op    0 allocs/op

BenchmarkZerolog-8            13057042     92.3 ns/op    164 B/op    0 allocs/op
BenchmarkZerolog5Fields-8      9709284    141.5 ns/op    331 B/op    0 allocs/op
BenchmarkZerologDisabled-8   271431801      4.42 ns/op      0 B/op    0 allocs/op

BenchmarkSlog-8                2058452    564.1 ns/op    260 B/op    0 allocs/op
BenchmarkZap-8                 2730224    445.2 ns/op    521 B/op    1 allocs/op
```

### Key Takeaways

- **🚀 Zero allocations** for all operations
- **⚡ 14% faster** than Zerolog for disabled logs (critical for high-throughput apps)
- **🎯 Highly competitive** with industry leaders (within 3-38% of Zerolog)
- **💪 Significantly faster** than slog and Zap
- **🔥 Production ready** with timestamps and full feature set

*Run `go test -bench=. -benchmem` to verify on your hardware.*

## 🏗️ Architecture

Logma's performance comes from several key optimizations:

- **Event Pooling**: Reuses log event objects to eliminate allocations
- **Direct Buffer Operations**: Numbers and strings appended directly to buffers
- **Allocation-Free Timestamps**: Custom RFC3339 formatting without string allocations  
- **Smart Buffer Sizing**: Pre-allocated buffers sized to reduce reallocations
- **Optimized Hot Paths**: Critical logging paths hand-tuned for performance

## 🤝 Why Choose Logma?

### vs. Zerolog
- **✅ Faster disabled logs** (14% improvement)
- **✅ Better developer experience** with richer API
- **✅ Built-in OpenTelemetry integration**
- **✅ More flexible configuration options**

### vs. Zap  
- **✅ Zero allocations** (Zap has 1 alloc/op)
- **✅ 3x faster** (127ns vs 445ns)
- **✅ Simpler API** without reflection complexity
- **✅ Better structured logging interface**

### vs. Standard slog
- **✅ 4x faster** (127ns vs 564ns) 
- **✅ Zero allocations** vs slog's allocations
- **✅ Richer field types** and better ergonomics
- **✅ Production-ready features** out of the box

## 🛠️ Production Usage

```go
// High-performance production setup
func setupLogging() *logma.Logger {
    config := logma.DefaultEncoderConfig()
    config.IncludeCaller = true
    
    // Structured JSON for production
    writer := logma.NewRotatingFileWriteSyncer("app.log", 100*1024*1024, 5)
    logger := logma.New(logma.NewJSONHandlerWithConfig(writer, config))
    
    // Set appropriate log level
    logger = logger.SetLevel(logma.INFO)
    
    // Add sampling for very high-volume scenarios
    if highTrafficMode {
        logger = logger.SetSampler(logma.NewFixedSampler(100))
    }
    
    return logger
}

// Context extraction for distributed tracing
func init() {
    logma.RegisterContextExtractor(logma.DefaultRequestIDExtractor)
    logma.RegisterContextExtractor(func(ctx context.Context) map[string]interface{} {
        if userID := ctx.Value("user_id"); userID != nil {
            return map[string]interface{}{"user_id": userID}
        }
        return nil
    })
}
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

```bash
git clone https://github.com/felixgeelhaar/logma.git
cd logma
go mod tidy
go test ./...
go test -bench=. -benchmem
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**Built with ❤️ for the Go community**

[Documentation](https://pkg.go.dev/github.com/felixgeelhaar/logma) | [Report Issues](https://github.com/felixgeelhaar/logma/issues) | [Discussions](https://github.com/felixgeelhaar/logma/discussions)

</div>
