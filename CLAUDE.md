# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Logma is a high-performance, zero-allocation structured logging library for Go designed to solve the "Logger's Trilemma" by balancing performance, developer experience, and observability. The library achieves performance comparable to zerolog and zap while providing an intuitive, chainable API.

## Architecture

### Core Components

- **Event Pool System**: Uses `sync.Pool` for event object pooling to minimize allocations (logma.go:77-84)
- **Handler Interface**: Pluggable output handlers for different formats (logma.go:52-57)
  - `JSONHandler`: Structured JSON output for production
  - `ConsoleHandler`: Human-readable colored output for development
- **Logger Chain**: Fluent API with method chaining for structured field addition
- **Context Integration**: Built-in OpenTelemetry trace/span ID integration

### Key Design Patterns

- **Zero-allocation core**: Event pooling and buffer reuse minimize garbage collection
- **Conditional logging**: Disabled log levels have minimal overhead with shared no-op events
- **TTY detection**: Automatic format selection based on terminal detection
- **Cross-platform support**: Separate isatty implementations for Unix and Windows
- **Pluggable architecture**: Configurable handlers, samplers, and context extractors
- **Performance-first**: Optimizations for common cases (disabled logs, clean strings)

### Performance Architecture

The library uses several optimization techniques:
- Pre-allocated byte buffers with 500-byte initial capacity
- Direct byte manipulation for JSON construction
- Pool-based event object reuse
- Early exit for disabled log levels

## Development Commands

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestJSONHandler

# Test with race detection
go test -race ./...
```

### Benchmarking
```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific benchmark
go test -bench=BenchmarkLogma -benchmem

# Compare with other logging libraries
go test -bench=. -benchmem > benchmarks.txt
```

### Linting and Quality Checks
```bash
# Run Go vet
go vet ./...

# Run golint (if available)
golint ./...

# Format code
go fmt ./...
```

### Module Management
```bash
# Update dependencies
go mod tidy

# Verify dependencies
go mod verify

# Download dependencies
go mod download
```

## Configuration

### Environment Variables
- `LOGMA_FORMAT`: Set to `json` or `console` (auto-detected if unset via TTY detection)
- `LOGMA_LEVEL`: Set log level (`trace`, `debug`, `info`, `warn`, `error`, `fatal`)

### Handler Selection Logic
1. Check `LOGMA_FORMAT` environment variable
2. If unset, use TTY detection via `isatty()` function
3. TTY detected → ConsoleHandler (colored output)
4. No TTY → JSONHandler (structured output)

## Code Organization

### Main Files
- `logma.go`: Core logger implementation, event pooling, handlers
- `logma_test.go`: Comprehensive test suite covering all functionality
- `logma_bench_test.go`: Performance benchmarks against competing libraries
- `default_logger_test.go`: Tests for default logger initialization and environment variable handling
- `isatty_unix.go`/`isatty_windows.go`: Platform-specific TTY detection

### Testing Strategy
- Unit tests for all public APIs
- Benchmark tests against zerolog, zap, and slog
- Environment variable configuration testing
- OpenTelemetry integration testing
- Cross-platform TTY detection testing

## Performance Considerations

### Benchmark Results (Latest)
- **Enabled logs**: ~231ns/op, 239B/op, 2 allocs/op for basic logging with timestamps
- **Disabled logs**: 5.6ns/op, 0B/op, 0 allocs/op (matches Zerolog performance)
- **Competitive performance**: 2x faster than Zap, matches Zerolog for disabled logs
- **Zero-allocation optimization**: Shared no-op events for disabled levels

### Optimization Features
- **Shared no-op events** - zero allocations for disabled log levels
- **Event buffer pooling** - reuse of event objects and buffers
- **Fast path string escaping** - skip escaping for clean strings
- **Direct level string append** - avoid String() method overhead
- **Early sampling checks** - sampling decision before allocations

## Dependencies

### Core Dependencies
- `go.opentelemetry.io/otel/trace`: OpenTelemetry integration
- `golang.org/x/term`: TTY detection for Unix systems
- `golang.org/x/sys`: System-level operations

### Benchmark Dependencies
- `github.com/rs/zerolog`: Performance comparison
- `go.uber.org/zap`: Performance comparison
- `golang.org/x/exp/slog`: Performance comparison

## New Features (Latest Updates)

### Context-Aware Logging
- **`WithContext(ctx)`** method for automatic context field extraction
- **Pluggable context extractors** via `RegisterContextExtractor()`
- **Built-in extractors** for request_id, user_id, session_id
- **Enhanced OpenTelemetry integration** with automatic trace/span extraction

### Advanced Error Handling
- **`ErrorWithStack(err)`** method for detailed error logging with stack traces
- **Error chain unwrapping** for nested error analysis
- **Stack trace integration** compatible with errors implementing StackTrace interface

### Complete Log Level Support
- **All log levels**: Trace, Debug, Info, Warn, Error, DPanic, Panic, Fatal
- **Proper panic behavior**: Panic always panics, DPanic only in development mode
- **Package-level functions** for all log levels

### Built-in Sampling
- **`FixedSampler`**: logs every Nth entry (e.g., every 10th log)
- **`RandomSampler`**: probabilistic sampling (e.g., 10% of logs)
- **`LevelSampler`**: different sampling rates per log level
- **Zero-allocation sampling**: decision made before event creation

### Advanced Customization
- **`EncoderConfig`**: customizable field names and formats
- **`NewJSONHandlerWithConfig()`**: custom encoder configuration
- **Configurable timestamps**: custom time formats and optional inclusion
- **Caller information**: optional file/line number logging

### Output Syncers and File Handling
- **`WriteSyncer`** interface for reliable output
- **`BufferedWriteSyncer`**: buffered writing for performance
- **`FileWriteSyncer`**: file output with sync support
- **`RotatingFileWriteSyncer`**: automatic log rotation by size
- **`MultiWriter`**: simultaneous output to multiple destinations

## Common Development Patterns

### Adding New Field Types
1. Add method to `Event` struct in logma.go
2. Follow existing pattern: check for nil event, append comma, format field
3. Add comprehensive tests in logma_test.go
4. Consider performance implications and benchmark if needed

### Handler Development
1. Implement `Handler` interface
2. Handle event buffer correctly (caller responsible for pool return)
3. Add tests for new handler
4. Consider integration with environment variable configuration

### Context Extractor Development
1. Implement `ContextExtractor` function type
2. Return map[string]interface{} with extracted fields
3. Register with `RegisterContextExtractor()`
4. Test with `WithContext()` method

### Sampling Strategy Development
1. Implement `Sampler` interface with `Sample() bool` method
2. Use atomic operations for thread-safety if needed
3. Test with `SetSampler()` or `SetLevelSampler()`
4. Benchmark impact on performance

### Performance Testing
- Always run benchmarks when making changes to core logging path
- Compare against baseline measurements
- Test both enabled and disabled logging scenarios
- Use `-benchmem` flag to track allocation changes
- Test sampling performance impact