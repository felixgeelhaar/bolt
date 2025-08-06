# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with the Bolt logging library.

## Table of Contents

- [Performance Issues](#performance-issues)
- [Thread Safety and Concurrency](#thread-safety-and-concurrency)
- [Configuration Problems](#configuration-problems)
- [Integration Issues](#integration-issues)
- [Memory and Resource Usage](#memory-and-resource-usage)
- [Output and Formatting](#output-and-formatting)
- [Debugging Tools and Techniques](#debugging-tools-and-techniques)

## Performance Issues

### Symptom: Logging Performance is Slower Than Expected

**Diagnosis:**
```bash
# Check current performance vs baseline
go test -bench=BenchmarkZeroAllocation -benchmem -count=5
```

**Common Causes & Solutions:**

#### 1. Debug Mode in Production
```go
// ❌ Problem: Debug level in production
logger.SetLevel(bolt.DEBUG) // Too verbose for production

// ✅ Solution: Use appropriate production levels
logger.SetLevel(bolt.WARN)  // Only warnings and errors
logger.SetLevel(bolt.ERROR) // Only errors in high-performance scenarios
```

#### 2. Inefficient Field Usage
```go
// ❌ Problem: Using Any() for primitive types
logger.Info().
    Any("count", 42).        // Slow - uses reflection
    Any("enabled", true).    // Slow - uses reflection
    Msg("status")

// ✅ Solution: Use type-specific methods
logger.Info().
    Int("count", 42).        // Fast - direct serialization
    Bool("enabled", true).   // Fast - direct serialization
    Msg("status")
```

#### 3. Expensive String Formatting
```go
// ❌ Problem: Pre-formatting expensive strings
expensiveString := fmt.Sprintf("complex: %+v", largeStruct)
logger.Info().Str("data", expensiveString).Msg("test")

// ✅ Solution: Conditional formatting
if logger.GetLevel() <= bolt.INFO {
    logger.Info().Str("data", fmt.Sprintf("complex: %+v", largeStruct)).Msg("test")
}

// ✅ Better: Use lazy evaluation or structured logging
logger.Info().Any("data", largeStruct).Msg("test") // Only when needed
```

#### 4. Event Lifecycle Issues
```go
// ❌ Problem: Not completing events
func problematic() {
    event := logger.Info().Str("key", "value")
    // Event never completed - can't be reused!
    if condition {
        return // Event leaked
    }
    event.Msg("message")
}

// ✅ Solution: Always complete events
func correct() {
    if !condition {
        logger.Info().Str("key", "value").Msg("message")
    }
}
```

### Symptom: High Memory Usage

**Diagnosis:**
```bash
# Profile memory usage
go test -bench=BenchmarkZeroAllocation -memprofile=mem.prof
go tool pprof mem.prof

# Check for memory leaks
go test -run=TestMemoryLeaks -v -timeout=30s
```

**Solutions:**

#### Context Logger Accumulation
```go
// ❌ Problem: Large context data accumulating
func badPattern() {
    baseLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    
    for _, item := range hugeDataSet {
        // Creates context logger with large data
        itemLogger := baseLogger.With().
            Any("huge_item", item). // Keeps entire item in memory
            Logger()
        
        // Use itemLogger multiple times...
        itemLogger.Info().Msg("processing")
        // itemLogger retains reference to huge_item
    }
}

// ✅ Solution: Minimal context, add data per event
func goodPattern() {
    baseLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    
    for _, item := range hugeDataSet {
        // Minimal context
        itemLogger := baseLogger.With().
            Str("item_id", item.ID).
            Logger()
        
        // Add data only when needed
        itemLogger.Info().
            Int("size", item.Size).
            Str("status", item.Status).
            Msg("processing")
    }
}
```

## Thread Safety and Concurrency

### Symptom: Race Conditions Detected

**Diagnosis:**
```bash
# Run with race detector
go test -race ./...
go run -race your-app.go
```

#### 1. Output Buffer Race Conditions
```go
// ❌ Problem: Unsafe output buffer
var unsafeBuf bytes.Buffer
logger := bolt.New(bolt.NewJSONHandler(&unsafeBuf))

// Multiple goroutines writing to same buffer
for i := 0; i < 10; i++ {
    go func(id int) {
        logger.Info().Int("id", id).Msg("concurrent") // RACE!
    }(i)
}

// ✅ Solution: Thread-safe buffer
type SafeBuffer struct {
    buf bytes.Buffer
    mu  sync.Mutex
}

func (sb *SafeBuffer) Write(p []byte) (n int, err error) {
    sb.mu.Lock()
    defer sb.mu.Unlock()
    return sb.buf.Write(p)
}

safeBuf := &SafeBuffer{}
logger := bolt.New(bolt.NewJSONHandler(safeBuf))
```

#### 2. File Output Race Conditions
```go
// ❌ Problem: Multiple loggers writing to same file
file, _ := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
logger1 := bolt.New(bolt.NewJSONHandler(file))
logger2 := bolt.New(bolt.NewJSONHandler(file)) // Same file!

// ✅ Solution: Single logger instance or synchronized access
var globalLogger = bolt.New(bolt.NewJSONHandler(file))

// OR: Use channel for synchronized writes
type ChannelHandler struct {
    ch chan []byte
}

func (h *ChannelHandler) Write(e *bolt.Event) error {
    buf := make([]byte, len(e.buf))
    copy(buf, e.buf)
    h.ch <- buf
    return nil
}
```

### Symptom: Logger Level Race Conditions (Fixed in v2.0.1+)

This issue was resolved with atomic operations in recent versions. If you're still experiencing it:

```bash
# Check your Bolt version
go list -m github.com/felixgeelhaar/bolt

# Update to latest version
go get github.com/felixgeelhaar/bolt@latest
```

**Verification:**
```go
// This should now be race-free
go func() {
    logger.SetLevel(bolt.DEBUG) // Atomic operation
}()

go func() {
    logger.Info().Msg("Concurrent logging") // Safe
}()
```

## Configuration Problems

### Symptom: Logs Not Appearing

**Diagnosis Steps:**
1. Check log level configuration
2. Verify handler setup
3. Confirm output destination
4. Test with explicit configuration

```go
func diagnoseMissingLogs() {
    // 1. Check current level
    logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    logger.SetLevel(bolt.ERROR) // Only errors will appear
    
    logger.Debug().Msg("This won't appear")  // Below ERROR level
    logger.Info().Msg("This won't appear")   // Below ERROR level
    logger.Error().Msg("This WILL appear")   // At ERROR level
    
    // 2. Test with Debug level
    logger.SetLevel(bolt.DEBUG)
    logger.Debug().Msg("Now this will appear")
}
```

**Environment Variable Issues:**
```bash
# Check environment variables
echo $BOLT_LEVEL   # Should be: trace, debug, info, warn, error, fatal
echo $BOLT_FORMAT  # Should be: json, console

# Test with explicit values
export BOLT_LEVEL=debug
export BOLT_FORMAT=console
go run your-app.go

# Clear variables to test defaults
unset BOLT_LEVEL BOLT_FORMAT
```

### Symptom: Wrong Output Format

```go
// Force specific format regardless of environment
func explicitFormat() {
    // Always JSON
    jsonLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    
    // Always console
    consoleLogger := bolt.New(bolt.NewConsoleHandler(os.Stdout))
    
    // Environment-aware with fallback
    var handler bolt.Handler
    if os.Getenv("FORCE_JSON") == "true" {
        handler = bolt.NewJSONHandler(os.Stdout)
    } else {
        handler = bolt.NewConsoleHandler(os.Stdout)
    }
    logger := bolt.New(handler)
}
```

## Integration Issues

### Symptom: OpenTelemetry Traces Not Appearing

**Diagnosis:**
```go
func diagnoseTracing(ctx context.Context) {
    // Check if context has valid span
    span := trace.SpanFromContext(ctx)
    if !span.SpanContext().IsValid() {
        fmt.Println("No valid span in context")
        return
    }
    
    fmt.Printf("Trace ID: %s\n", span.SpanContext().TraceID().String())
    fmt.Printf("Span ID: %s\n", span.SpanContext().SpanID().String())
    
    // Use context-aware logger
    ctxLogger := logger.Ctx(ctx)
    ctxLogger.Info().Msg("This should include trace IDs")
}
```

**Common Solutions:**
```go
// 1. Ensure OpenTelemetry is properly initialized
func setupTracing() context.Context {
    tracer := otel.Tracer("your-service")
    ctx, span := tracer.Start(context.Background(), "operation")
    defer span.End()
    
    // Use ctx with logger
    logger.Ctx(ctx).Info().Msg("Traced operation")
    return ctx
}

// 2. Manual trace ID injection if needed
func manualTraceIDs(ctx context.Context) {
    span := trace.SpanFromContext(ctx)
    if span.SpanContext().IsValid() {
        logger.Info().
            Str("trace_id", span.SpanContext().TraceID().String()).
            Str("span_id", span.SpanContext().SpanID().String()).
            Msg("Manual trace injection")
    }
}
```

### Symptom: Custom Handler Not Working

```go
// Example debugging a custom handler
type DebuggingHandler struct {
    wrapped bolt.Handler
}

func (h *DebuggingHandler) Write(e *bolt.Event) error {
    fmt.Printf("Handler received: %s", string(e.buf))
    return h.wrapped.Write(e)
}

// Usage
originalHandler := bolt.NewJSONHandler(os.Stdout)
debuggingHandler := &DebuggingHandler{wrapped: originalHandler}
logger := bolt.New(debuggingHandler)
```

## Memory and Resource Usage

### Symptom: Event Pool Inefficiency

**Diagnosis:**
```go
// Monitor pool usage
func monitorPoolUsage() {
    var poolGets, poolPuts int64
    
    // Custom monitoring (for debugging only)
    originalNew := eventPool.New
    eventPool.New = func() interface{} {
        atomic.AddInt64(&poolGets, 1)
        return originalNew()
    }
    
    // Run your logging operations
    for i := 0; i < 1000; i++ {
        logger.Info().Int("i", i).Msg("test")
        atomic.AddInt64(&poolPuts, 1)
    }
    
    fmt.Printf("Pool gets: %d, puts: %d\n", poolGets, poolPuts)
}
```

### Symptom: Buffer Growth Issues

```go
// Monitor buffer sizes
func monitorBufferSizes(logger *bolt.Logger) {
    // This is for debugging - don't use in production
    logger.Info().Str("test", strings.Repeat("x", 1000)).Msg("buffer test")
    
    // Check if buffers are growing excessively
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    fmt.Printf("Heap size: %d KB\n", m.HeapAlloc/1024)
}
```

## Output and Formatting

### Symptom: Malformed JSON Output

**Causes & Solutions:**

#### 1. Concurrent Writes to Unsafe Buffer
```go
// ❌ Problem: Multiple goroutines writing to bytes.Buffer
var buf bytes.Buffer
logger := bolt.New(bolt.NewJSONHandler(&buf))

// ✅ Solution: Use thread-safe output or single goroutine
```

#### 2. Incomplete Events
```go
// ❌ Problem: Events not properly completed
func incomplete() {
    event := logger.Info().Str("key", "value")
    // Missing Msg() call - event not completed
}

// ✅ Solution: Always complete events
func complete() {
    logger.Info().Str("key", "value").Msg("message")
}
```

#### 3. Handler Errors
```go
// Custom handler that might fail
type FailingHandler struct{}

func (h *FailingHandler) Write(e *bolt.Event) error {
    return errors.New("handler failed")
}

// Solution: Add error handling
logger := bolt.New(&FailingHandler{}).
    SetErrorHandler(func(err error) {
        fmt.Fprintf(os.Stderr, "Logger error: %v\n", err)
    })
```

### Symptom: Missing Fields in Output

```go
// Common field issues
func fieldTroubleshooting() {
    // Field not appearing due to level filtering
    logger.SetLevel(bolt.ERROR)
    logger.Debug().Str("debug_field", "value").Msg("debug") // Won't appear
    
    // Field truncated due to size limits
    largeValue := strings.Repeat("x", 100000) // > 64KB limit
    logger.Info().Str("large", largeValue).Msg("test") // Value truncated
    
    // Field with invalid key
    logger.Info().Str("key\x00\n", "value").Msg("test") // Key rejected
}
```

## Debugging Tools and Techniques

### Performance Profiling

```bash
# CPU profiling
go test -bench=BenchmarkYourCode -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Memory profiling
go test -bench=BenchmarkYourCode -memprofile=mem.prof
go tool pprof mem.prof

# Block profiling for concurrency issues
go test -bench=BenchmarkYourCode -blockprofile=block.prof
go tool pprof block.prof
```

### Race Detection

```bash
# Run with race detector
go run -race your-app.go
go test -race ./...

# Build with race detector
go build -race -o app-race your-app.go
./app-race
```

### Memory Leak Detection

```go
func detectMemoryLeaks(t *testing.T) {
    var before, after runtime.MemStats
    
    runtime.GC()
    runtime.ReadMemStats(&before)
    
    // Run your logging code
    for i := 0; i < 10000; i++ {
        logger.Info().Int("i", i).Msg("leak test")
    }
    
    runtime.GC()
    runtime.ReadMemStats(&after)
    
    if after.HeapAlloc > before.HeapAlloc+1024*1024 { // 1MB threshold
        t.Errorf("Potential memory leak: %d -> %d bytes", 
            before.HeapAlloc, after.HeapAlloc)
    }
}
```

### Custom Diagnostics

```go
// Diagnostic logger wrapper
type DiagnosticLogger struct {
    *bolt.Logger
    eventCount  int64
    errorCount  int64
    totalBytes  int64
}

func (dl *DiagnosticLogger) Info() *bolt.Event {
    atomic.AddInt64(&dl.eventCount, 1)
    return dl.Logger.Info()
}

func (dl *DiagnosticLogger) Stats() (events, errors, bytes int64) {
    return atomic.LoadInt64(&dl.eventCount),
           atomic.LoadInt64(&dl.errorCount),
           atomic.LoadInt64(&dl.totalBytes)
}

// Usage
diagLogger := &DiagnosticLogger{
    Logger: bolt.New(bolt.NewJSONHandler(os.Stdout)),
}

// Use diagLogger for logging...

// Check stats
events, errors, bytes := diagLogger.Stats()
fmt.Printf("Events: %d, Errors: %d, Bytes: %d\n", events, errors, bytes)
```

### Environment Debugging

```go
func debugEnvironment() {
    fmt.Printf("BOLT_LEVEL: %s\n", os.Getenv("BOLT_LEVEL"))
    fmt.Printf("BOLT_FORMAT: %s\n", os.Getenv("BOLT_FORMAT"))
    
    // Test different configurations
    configs := []struct {
        level  string
        format string
    }{
        {"debug", "json"},
        {"info", "console"},
        {"error", "json"},
    }
    
    for _, cfg := range configs {
        os.Setenv("BOLT_LEVEL", cfg.level)
        os.Setenv("BOLT_FORMAT", cfg.format)
        
        // Reinitialize with new environment
        logger := setupLogger()
        logger.Info().Str("config", fmt.Sprintf("%s/%s", cfg.level, cfg.format)).Msg("test")
    }
}
```

## Getting Help

### Information to Provide

When reporting issues, please include:

1. **Bolt version**: `go list -m github.com/felixgeelhaar/bolt`
2. **Go version**: `go version`
3. **Operating system**: `uname -a` (Linux/Mac) or `systeminfo` (Windows)
4. **Minimal reproduction code**
5. **Expected vs actual behavior**
6. **Error messages or logs**
7. **Performance measurements** (if performance issue)

### Diagnostic Commands

```bash
# Comprehensive diagnostics
go version
go list -m github.com/felixgeelhaar/bolt
go env GOOS GOARCH
echo "BOLT_LEVEL=$BOLT_LEVEL BOLT_FORMAT=$BOLT_FORMAT"

# Test basic functionality
go run -c 'package main
import "github.com/felixgeelhaar/bolt"
import "os"
func main() {
    logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    logger.Info().Str("test", "diagnostic").Msg("basic test")
}'

# Performance test
go test -bench=BenchmarkZeroAllocation -benchmem -count=3

# Race detection test
go test -race -run=TestConcurrent -v
```

### Community Resources

- **GitHub Issues**: [Report bugs](https://github.com/felixgeelhaar/bolt/issues)
- **GitHub Discussions**: [General questions](https://github.com/felixgeelhaar/bolt/discussions)
- **Documentation**: [README.md](README.md)
- **Security Issues**: [SECURITY.md](SECURITY.md)

---

If this troubleshooting guide doesn't resolve your issue, please open a GitHub issue with the diagnostic information above.