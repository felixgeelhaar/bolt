# Logma

<div align="center">
  <img src="misc/logo.svg" alt="Logma Logo" width="300"/>
  
  [![Go Version](https://img.shields.io/badge/go-%3E%3D1.19-blue.svg)](https://golang.org/)
  [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
  [![Go Report Card](https://goreportcard.com/badge/github.com/felixgeelhaar/logma)](https://goreportcard.com/report/github.com/felixgeelhaar/logma)
  [![Performance](https://img.shields.io/badge/performance-127ns%2Fop%20%7C%200%20allocs-brightgreen.svg)](#performance)
</div>

## ⚡ Zero-Allocation Structured Logging

Logma is a high-performance, zero-allocation structured logging library for Go that delivers exceptional speed without compromising on features. Built from the ground up for modern applications that demand both performance and observability.

### 🚀 Performance First

| Library | Operation | ns/op | Allocations | Performance Advantage |
|---------|-----------|-------|-------------|----------------------|
| **Logma** | Disabled | **85.2** | **0** | **14% faster than Zerolog** |
| **Logma** | Enabled | **127.1** | **0** | **27% faster than Zerolog** |
| Zerolog | Disabled | 99.3 | 0 | - |
| Zerolog | Enabled | 175.4 | 0 | - |
| Zap | Disabled | 104.2 | 0 | - |
| Zap | Enabled | 189.7 | 1 | - |
| Logrus | Enabled | 2,847 | 23 | - |

*Benchmarks performed on Apple M1 Pro with Go 1.21*

## ✨ Features

- **🔥 Zero Allocations**: Achieved through intelligent event pooling and buffer reuse
- **⚡ Ultra-Fast**: 127ns/op for enabled logs, 85ns for disabled
- **🏗️ Structured Logging**: Rich, type-safe field support with JSON output
- **🔍 OpenTelemetry Integration**: Automatic trace and span ID injection
- **🎨 Multiple Outputs**: JSON for production, colorized console for development
- **🧩 Extensible**: Custom handlers and formatters
- **📦 Zero Dependencies**: Lightweight with minimal external dependencies
- **🛡️ Type Safe**: Strongly typed field methods prevent runtime errors

## 📦 Installation

```bash
go get github.com/felixgeelhaar/logma
```

## 🏃 Quick Start

### Basic Usage

```go
package main

import (
    "os"
    "github.com/felixgeelhaar/logma"
)

func main() {
    // Create a JSON logger for production
    logger := logma.New(logma.NewJSONHandler(os.Stdout))
    
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
logger := logma.New(logma.NewConsoleHandler(os.Stdout))

logger.Info().
    Str("env", "development").
    Int("workers", 4).
    Msg("Application initialized")

// Output: [2024-01-15T10:30:45Z] INFO Application initialized env=development workers=4
```

## 🏗️ Architecture Insights

### Zero-Allocation Design

Logma achieves zero allocations through several key innovations:

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
logger := logma.New(logma.NewJSONHandler(os.Stdout))

// Set via environment variables:
// LOGMA_LEVEL=info
// LOGMA_FORMAT=json (production) or console (development)
```

### OpenTelemetry Integration

```go
import (
    "context"
    "go.opentelemetry.io/otel"
    "github.com/felixgeelhaar/logma"
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

Run the included benchmarks to see Logma's performance on your system:

```bash
go test -bench=. -benchmem ./...
```

### Sample Results

```
BenchmarkLogmaDisabled-10       14,129,394    85.2 ns/op     0 B/op    0 allocs/op
BenchmarkLogmaEnabled-10         7,864,321   127.1 ns/op     0 B/op    0 allocs/op
BenchmarkZerologDisabled-10     12,077,472    99.3 ns/op     0 B/op    0 allocs/op
BenchmarkZerologEnabled-10       5,698,320   175.4 ns/op     0 B/op    0 allocs/op
```

## 🔧 Custom Handlers

Extend Logma with custom output formats:

```go
type CustomHandler struct {
    output io.Writer
}

func (h *CustomHandler) Write(e *logma.Event) error {
    // Custom formatting logic
    formatted := customFormat(e)
    _, err := h.output.Write(formatted)
    return err
}

logger := logma.New(&CustomHandler{output: os.Stdout})
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
git clone https://github.com/felixgeelhaar/logma.git
cd logma
go mod tidy
go test ./...
go test -bench=. -benchmem
```

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🎖️ Recognition

Logma draws inspiration from excellent logging libraries like Zerolog and Zap, while pushing the boundaries of what's possible in Go logging performance.

---

<div align="center">
  <img src="misc/logo_icon.svg" alt="Logma Icon" width="64"/>
  
  **Built with ❤️ for high-performance Go applications**
</div>