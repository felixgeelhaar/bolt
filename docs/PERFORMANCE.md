# Performance Tuning Guide

This guide provides detailed information on optimizing Bolt's performance for various use cases and deployment scenarios.

## Table of Contents

- [Performance Fundamentals](#performance-fundamentals)
- [Benchmarking](#benchmarking)
- [Zero-Allocation Techniques](#zero-allocation-techniques)
- [Configuration Optimization](#configuration-optimization)
- [Production Deployment](#production-deployment)
- [Monitoring and Profiling](#monitoring-and-profiling)
- [Platform-Specific Optimizations](#platform-specific-optimizations)
- [Common Performance Pitfalls](#common-performance-pitfalls)

## Performance Fundamentals

### Current Performance Metrics

Bolt delivers industry-leading performance with the following characteristics:

| Configuration | ns/op | allocs/op | B/op | Performance Gain |
|---------------|-------|-----------|------|------------------|
| **Bolt Disabled** | **85.2** | **0** | **0** | **14% vs Zerolog** |
| **Bolt Enabled** | **127.1** | **0** | **0** | **27% vs Zerolog** |
| Zerolog Disabled | 99.3 | 0 | 0 | Baseline |
| Zerolog Enabled | 175.4 | 0 | 0 | Baseline |
| Zap Enabled | 189.7 | 1 | 0 | -33% vs Bolt |
| Logrus Enabled | 2,847 | 23 | 5,512 | -2,141% vs Bolt |

### Architecture Overview

Bolt's performance comes from several key architectural decisions:

1. **Event Pooling**: Reuses event objects via `sync.Pool`
2. **Direct Buffer Manipulation**: Bypasses intermediate allocations
3. **Custom Serialization**: Specialized functions for common data types
4. **Atomic Operations**: Thread-safe operations with minimal overhead
5. **Branch Prediction Optimization**: Code structured for common paths

## Benchmarking

### Running Performance Tests

```bash
# Basic benchmarks
go test -bench=BenchmarkZeroAllocation -benchmem

# Extended benchmark suite
go test -bench=. -benchmem -count=5

# Memory profiling
go test -bench=BenchmarkZeroAllocation -memprofile=mem.prof
go tool pprof mem.prof

# CPU profiling
go test -bench=BenchmarkZeroAllocation -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

### Custom Benchmark Creation

```go
func BenchmarkYourUsage(b *testing.B) {
    buf := &bytes.Buffer{}
    logger := bolt.New(bolt.NewJSONHandler(buf))
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        logger.Info().
            Str("service", "api").
            Int("user_id", i).
            Float64("response_time", 0.123).
            Bool("success", true).
            Msg("Request processed")
    }
    
    b.SetBytes(int64(buf.Len()))
}
```

### Comparative Benchmarking

```go
// Compare against your current logging solution
func BenchmarkCurrentLogger(b *testing.B) {
    // Your existing logger setup
    for i := 0; i < b.N; i++ {
        // Your logging code
    }
}

func BenchmarkBoltEquivalent(b *testing.B) {
    buf := &bytes.Buffer{}
    logger := bolt.New(bolt.NewJSONHandler(buf))
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        logger.Info().
            // Equivalent logging operations
            Msg("equivalent message")
    }
}
```

## Zero-Allocation Techniques

### Event Lifecycle Management

```go
// ✅ Proper event completion
logger.Info().Str("key", "value").Msg("message")

// ❌ Event leak - never completes
event := logger.Info().Str("key", "value")
// Event never returned to pool

// ✅ Conditional logging without leaks
if shouldLog {
    logger.Info().Str("key", "value").Msg("message")
}

// ❌ Creates unnecessary events
event := logger.Info()
if shouldLog {
    event.Str("key", "value").Msg("message")
}
```

### Buffer Reuse Strategies

```go
// ✅ Let Bolt manage buffer lifecycle
logger.Info().
    Str("field1", value1).
    Str("field2", value2).
    Msg("message")

// ❌ Don't try to manage buffers manually
var buf []byte
// Manual buffer management defeats pooling
```

### Field Method Optimization

```go
// ✅ Use specific type methods for best performance
logger.Info().
    Int("count", 42).           // Faster than Any()
    Bool("success", true).      // Faster than Any()
    Str("service", "api").      // Faster than Any()
    Msg("request completed")

// ❌ Generic Any() method is slower
logger.Info().
    Any("count", 42).          // Slower, uses reflection
    Any("success", true).      // Slower, uses reflection
    Any("service", "api").     // Slower, uses reflection
    Msg("request completed")
```

## Configuration Optimization

### Log Level Configuration

```go
// Production configuration
logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
logger.SetLevel(bolt.WARN)  // Only log warnings and errors

// Development configuration
logger.SetLevel(bolt.DEBUG)  // All log levels

// Environment-based configuration
func setupLogger() *bolt.Logger {
    logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    
    switch os.Getenv("ENV") {
    case "production":
        logger.SetLevel(bolt.ERROR)
    case "staging":
        logger.SetLevel(bolt.WARN)
    default:
        logger.SetLevel(bolt.DEBUG)
    }
    
    return logger
}
```

### Handler Selection

```go
// Production: JSON for structured logs
logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

// Development: Console for readability
logger := bolt.New(bolt.NewConsoleHandler(os.Stdout))

// High-performance: Custom handler with minimal formatting
type FastHandler struct {
    out io.Writer
}

func (h *FastHandler) Write(e *bolt.Event) error {
    // Minimal processing for maximum speed
    _, err := h.out.Write(e.buf)
    return err
}
```

### Context Logger Optimization

```go
// ✅ Create context loggers once, reuse many times
userLogger := baseLogger.With().
    Str("user_id", userID).
    Str("session_id", sessionID).
    Logger()

// Reuse throughout request lifecycle
userLogger.Info().Msg("Request started")
userLogger.Debug().Msg("Processing data")
userLogger.Info().Msg("Request completed")

// ❌ Don't recreate context loggers repeatedly
for i := 0; i < n; i++ {
    logger.With().Str("user_id", userID).Logger().Info().Msg("iteration")
    // Creates new logger each time
}
```

## Production Deployment

### Memory Management

```go
// Monitor event pool efficiency
var poolStats struct {
    gets  int64
    puts  int64
    news  int64
}

// Custom pool monitoring (development only)
func monitorPool() {
    originalPool := eventPool
    eventPool = &sync.Pool{
        New: func() interface{} {
            atomic.AddInt64(&poolStats.news, 1)
            return originalPool.New()
        },
    }
}
```

### High-Throughput Configuration

```go
// Optimized for high-volume logging
func setupHighThroughputLogger(output io.Writer) *bolt.Logger {
    // Use buffered writer for batch I/O
    bufferedOutput := bufio.NewWriterSize(output, 64*1024)
    
    logger := bolt.New(bolt.NewJSONHandler(bufferedOutput))
    
    // Periodic flush
    go func() {
        ticker := time.NewTicker(100 * time.Millisecond)
        for range ticker.C {
            bufferedOutput.Flush()
        }
    }()
    
    return logger
}
```

### Container Deployment

```dockerfile
# Dockerfile optimizations for logging performance
FROM golang:1.21-alpine AS builder

# Build with optimizations
ENV GOOS=linux GOARCH=amd64 CGO_ENABLED=0
RUN go build -ldflags="-w -s" -o app ./cmd/app

FROM alpine:latest
# Minimal logging overhead
ENV BOLT_LEVEL=warn
ENV BOLT_FORMAT=json
```

## Monitoring and Profiling

### Runtime Performance Monitoring

```go
func setupPerformanceMonitoring(logger *bolt.Logger) {
    // Log performance metrics periodically
    go func() {
        var m runtime.MemStats
        ticker := time.NewTicker(30 * time.Second)
        
        for range ticker.C {
            runtime.ReadMemStats(&m)
            
            logger.Info().
                Uint64("heap_alloc", m.HeapAlloc).
                Uint64("heap_sys", m.HeapSys).
                Uint32("num_gc", m.NumGC).
                Float64("gc_cpu_fraction", m.GCCPUFraction).
                Msg("runtime stats")
        }
    }()
}
```

### Custom Metrics Collection

```go
type PerformanceLogger struct {
    *bolt.Logger
    totalLogs    int64
    totalBytes   int64
    startTime    time.Time
}

func NewPerformanceLogger(logger *bolt.Logger) *PerformanceLogger {
    return &PerformanceLogger{
        Logger:    logger,
        startTime: time.Now(),
    }
}

func (pl *PerformanceLogger) Info() *bolt.Event {
    atomic.AddInt64(&pl.totalLogs, 1)
    return pl.Logger.Info()
}

func (pl *PerformanceLogger) Stats() (logsPerSecond, bytesPerSecond float64) {
    elapsed := time.Since(pl.startTime).Seconds()
    logs := atomic.LoadInt64(&pl.totalLogs)
    bytes := atomic.LoadInt64(&pl.totalBytes)
    
    return float64(logs) / elapsed, float64(bytes) / elapsed
}
```

### Profiling Integration

```go
// Production profiling setup
func setupProfiling(logger *bolt.Logger) {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    
    // Periodic performance snapshots
    go func() {
        ticker := time.NewTicker(5 * time.Minute)
        for range ticker.C {
            // Capture heap profile
            f, err := os.Create("heap.prof")
            if err != nil {
                continue
            }
            pprof.WriteHeapProfile(f)
            f.Close()
            
            logger.Info().Str("profile", "heap.prof").Msg("heap profile captured")
        }
    }()
}
```

## Platform-Specific Optimizations

### Linux Optimizations

```go
// Linux-specific I/O optimizations
func setupLinuxOptimizations() {
    // Use splice() syscall for zero-copy I/O when possible
    // This is handled automatically by Go's runtime
    
    // For file logging, use O_APPEND for better performance
    file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        log.Fatal(err)
    }
    
    logger := bolt.New(bolt.NewJSONHandler(file))
}
```

### macOS Optimizations

```go
// macOS-specific optimizations
func setupMacOSOptimizations() {
    // Use unified logging system integration when appropriate
    if runtime.GOOS == "darwin" {
        // Consider os_log integration for system-level logging
    }
}
```

### Windows Optimizations

```go
// Windows-specific optimizations
func setupWindowsOptimizations() {
    if runtime.GOOS == "windows" {
        // Use Windows Event Log for system integration
        // Consider ETW (Event Tracing for Windows) for high-performance scenarios
    }
}
```

## Common Performance Pitfalls

### Event Lifecycle Mismanagement

```go
// ❌ Don't store events for later use
var storedEvent *bolt.Event

func bad() {
    storedEvent = logger.Info().Str("key", "value")
    // Event is never completed and can't be reused
}

// ✅ Complete events immediately
func good() {
    logger.Info().Str("key", "value").Msg("message")
    // Event returned to pool immediately
}
```

### Inefficient String Formatting

```go
// ❌ Don't format strings before passing to logger
logger.Info().Str("formatted", fmt.Sprintf("value: %d", value)).Msg("message")

// ✅ Use specific field types
logger.Info().Int("value", value).Msg("message")

// ✅ For complex formatting, use lazy evaluation
logger.Info().
    Str("complex", func() string {
        if logger.GetLevel() <= bolt.INFO {
            return expensiveFormatting(data)
        }
        return ""
    }()).
    Msg("message")
```

### Synchronous I/O Blocking

```go
// ❌ Synchronous I/O can block logging
func badHandler(output io.Writer) bolt.Handler {
    return bolt.NewJSONHandler(output) // May block on slow outputs
}

// ✅ Use buffered or asynchronous I/O
func goodHandler(output io.Writer) bolt.Handler {
    buffered := bufio.NewWriter(output)
    
    // Periodic flush in background
    go func() {
        ticker := time.NewTicker(100 * time.Millisecond)
        for range ticker.C {
            buffered.Flush()
        }
    }()
    
    return bolt.NewJSONHandler(buffered)
}
```

### Memory Leak Patterns

```go
// ❌ Large context loggers accumulate memory
func badContextLogging() {
    baseLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    
    for _, item := range hugeDataSet {
        // Creates large context that persists
        itemLogger := baseLogger.With().
            Any("huge_data", item). // Large object kept in context
            Logger()
        
        itemLogger.Info().Msg("processing")
        // itemLogger retains reference to huge_data
    }
}

// ✅ Use minimal context, add data per log entry
func goodContextLogging() {
    baseLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    
    for _, item := range hugeDataSet {
        // Minimal context
        itemLogger := baseLogger.With().
            Str("item_id", item.ID).
            Logger()
        
        // Add data only when needed
        itemLogger.Info().
            Int("size", item.Size).  // Only what's needed
            Msg("processing")
    }
}
```

## Performance Validation

### Continuous Performance Testing

```go
// Automated performance regression detection
func TestPerformanceRegression(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping performance test in short mode")
    }
    
    buf := &bytes.Buffer{}
    logger := bolt.New(bolt.NewJSONHandler(buf))
    
    // Baseline performance expectations
    const (
        maxNsPerOp    = 200  // Max 200ns per operation
        maxAllocsPerOp = 0   // Zero allocations required
    )
    
    result := testing.Benchmark(func(b *testing.B) {
        b.ResetTimer()
        b.ReportAllocs()
        
        for i := 0; i < b.N; i++ {
            logger.Info().
                Str("service", "api").
                Int("request_id", i).
                Msg("performance test")
        }
    })
    
    nsPerOp := result.NsPerOp()
    allocsPerOp := float64(result.AllocsPerOp())
    
    if nsPerOp > maxNsPerOp {
        t.Errorf("Performance regression: %d ns/op > %d ns/op", nsPerOp, maxNsPerOp)
    }
    
    if allocsPerOp > maxAllocsPerOp {
        t.Errorf("Allocation regression: %.1f allocs/op > %d allocs/op", allocsPerOp, maxAllocsPerOp)
    }
    
    t.Logf("Performance: %d ns/op, %.1f allocs/op", nsPerOp, allocsPerOp)
}
```

### Production Performance Monitoring

```go
// Real-world performance tracking
type LoggerMetrics struct {
    logger        *bolt.Logger
    totalLogs     int64
    totalLatency  int64
    maxLatency    int64
}

func (lm *LoggerMetrics) Info() *bolt.Event {
    start := time.Now()
    event := lm.logger.Info()
    
    // Track performance
    latency := time.Since(start).Nanoseconds()
    atomic.AddInt64(&lm.totalLogs, 1)
    atomic.AddInt64(&lm.totalLatency, latency)
    
    // Update max latency
    for {
        current := atomic.LoadInt64(&lm.maxLatency)
        if latency <= current || atomic.CompareAndSwapInt64(&lm.maxLatency, current, latency) {
            break
        }
    }
    
    return event
}

func (lm *LoggerMetrics) Stats() (avgNs, maxNs int64) {
    total := atomic.LoadInt64(&lm.totalLogs)
    if total == 0 {
        return 0, 0
    }
    
    totalLatency := atomic.LoadInt64(&lm.totalLatency)
    maxLatency := atomic.LoadInt64(&lm.maxLatency)
    
    return totalLatency / total, maxLatency
}
```

This performance guide provides comprehensive optimization strategies for Bolt. Regular benchmarking and monitoring ensure optimal performance across different deployment scenarios and usage patterns.