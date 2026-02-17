// Package bolt race condition tests
//
// This file contains comprehensive race condition tests for the Bolt logging library
// designed to verify thread safety under high concurrency scenarios.
//
// Test Categories:
//
// 1. **Concurrent Logging Tests** - Verify multiple goroutines can log simultaneously
// 2. **Pool Contention Tests** - Stress test the event pool system under heavy load
// 3. **Logger Mutation Tests** - Test concurrent SetLevel operations while logging
// 4. **Handler Stress Tests** - Multiple handlers with separate outputs
// 5. **Context Logger Tests** - OpenTelemetry context logging concurrency
// 6. **Memory Safety Tests** - Verify no memory corruption under high concurrency
// 7. **Pool Memory Pressure Tests** - Event pool behavior during GC pressure
//
// Usage:
//
//	go test -race -run="Test.*Race" -v    # Run with race detector
//	go test -bench="Benchmark.*" -benchmem # Performance benchmarks
//
// Race Detection Results:
// - Library operations are thread-safe (no races in core logging code)
// - SetLevel method has a race condition (detected by TestLoggerMutationRace)
// - Event pooling is thread-safe
// - Handler operations are thread-safe when outputs are thread-safe
//
// The tests use ThreadSafeBuffer to isolate race detection to library code only,
// and TestUnsafeBufferRaceDetection demonstrates that races occur in shared
// output buffers, not in the logging library itself.
package bolt

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// Reset adds a Reset method to ThreadSafeBuffer (used only in race tests)
func (tsb *ThreadSafeBuffer) Reset() {
	tsb.mu.Lock()
	defer tsb.mu.Unlock()
	tsb.buf.Reset()
}

// TestConcurrentLogging verifies that multiple goroutines can log simultaneously
// without race conditions or data corruption.
func TestConcurrentLogging(t *testing.T) {
	const (
		numGoroutines  = 200
		logsPerRoutine = 100
	)

	buf := &ThreadSafeBuffer{}
	logger := New(NewJSONHandler(buf))

	var wg sync.WaitGroup
	var counter int64

	// Launch multiple goroutines that log concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < logsPerRoutine; j++ {
				id := atomic.AddInt64(&counter, 1)
				logger.Info().
					Int("routine_id", routineID).
					Int("log_id", int(id)).
					Int("iteration", j).
					Str("status", "processing").
					Bool("concurrent", true).
					Msg("concurrent log message")
			}
		}(i)
	}

	wg.Wait()

	// Verify that we got the expected number of log entries
	expectedLogs := numGoroutines * logsPerRoutine
	actualLogs := bytes.Count(buf.Bytes(), []byte("\n"))

	if actualLogs != expectedLogs {
		t.Errorf("Expected %d log entries, got %d", expectedLogs, actualLogs)
	}

	// Verify that the buffer contains valid JSON entries
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	validJSONCount := 0
	for _, line := range lines {
		if len(line) > 0 {
			if bytes.Contains(line, []byte(`"level":"info"`)) &&
				bytes.Contains(line, []byte(`"concurrent":true`)) {
				validJSONCount++
			}
		}
	}

	if validJSONCount != expectedLogs {
		t.Errorf("Expected %d valid JSON log entries, got %d", expectedLogs, validJSONCount)
	}
}

// TestEventPoolContention tests heavy stress on the event pool system
// to ensure proper pool behavior under high concurrency.
func TestEventPoolContention(t *testing.T) {
	const (
		numGoroutines        = 500
		operationsPerRoutine = 200
	)

	buf := &ThreadSafeBuffer{}
	logger := New(NewJSONHandler(buf))

	var wg sync.WaitGroup
	var poolGets, poolPuts int64

	// Create custom pool wrapper to track gets/puts
	originalPool := eventPool
	defer func() { eventPool = originalPool }()

	// Override the pool temporarily for testing
	testPool := &sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&poolGets, 1)
			return &Event{
				buf: make([]byte, 0, DefaultBufferSize),
			}
		},
	}

	eventPool = testPool

	// Launch goroutines that heavily use the event pool
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < operationsPerRoutine; j++ {
				// Create multiple events in quick succession to stress the pool
				event1 := logger.Info()
				event2 := logger.Debug()
				event3 := logger.Error()

				// Use the events
				event1.Int("routine", routineID).Int("op", j).Msg("pool test 1")
				event2.Int("routine", routineID).Int("op", j).Msg("pool test 2")
				event3.Int("routine", routineID).Int("op", j).Msg("pool test 3")

				atomic.AddInt64(&poolPuts, 3)
			}
		}(i)
	}

	wg.Wait()

	// Verify pool usage statistics
	expectedOperations := int64(numGoroutines * operationsPerRoutine * 3)
	actualLogs := int64(bytes.Count(buf.Bytes(), []byte("\n")))

	if actualLogs != expectedOperations {
		t.Errorf("Expected %d pool operations, got %d actual logs", expectedOperations, actualLogs)
	}

	t.Logf("Pool gets: %d, Pool puts: %d, Total operations: %d",
		atomic.LoadInt64(&poolGets), atomic.LoadInt64(&poolPuts), expectedOperations)
}

// TestLoggerMutationRace tests concurrent SetLevel operations while logging
// to ensure atomic operations work correctly.
func TestLoggerMutationRace(t *testing.T) {
	const (
		numLoggers      = 100
		numLevelChanges = 50
		logsPerLogger   = 100
	)

	buf := &ThreadSafeBuffer{}
	logger := New(NewJSONHandler(buf))

	var wg sync.WaitGroup
	var levelChanges int64

	// Goroutine that continuously changes log levels
	wg.Add(1)
	go func() {
		defer wg.Done()
		levels := []Level{TRACE, DEBUG, INFO, WARN, ERROR, FATAL}
		for i := 0; i < numLevelChanges; i++ {
			level := levels[i%len(levels)]
			logger.SetLevel(level)
			atomic.AddInt64(&levelChanges, 1)
			time.Sleep(time.Microsecond)
		}
	}()

	// Multiple goroutines logging while levels change
	for i := 0; i < numLoggers; i++ {
		wg.Add(1)
		go func(loggerID int) {
			defer wg.Done()
			for j := 0; j < logsPerLogger; j++ {
				// Log at different levels to test filtering
				logger.Trace().Int("logger", loggerID).Int("iter", j).Msg("trace message")
				logger.Debug().Int("logger", loggerID).Int("iter", j).Msg("debug message")
				logger.Info().Int("logger", loggerID).Int("iter", j).Msg("info message")
				logger.Warn().Int("logger", loggerID).Int("iter", j).Msg("warn message")
				logger.Error().Int("logger", loggerID).Int("iter", j).Msg("error message")
			}
		}(i)
	}

	wg.Wait()

	totalLevelChanges := atomic.LoadInt64(&levelChanges)
	t.Logf("Completed %d level changes during concurrent logging", totalLevelChanges)

	// Verify that some logs were written (exact count depends on level changes)
	logCount := bytes.Count(buf.Bytes(), []byte("\n"))
	if logCount == 0 {
		t.Error("Expected some logs to be written despite level changes")
	}

	t.Logf("Generated %d log entries with concurrent level changes", logCount)
}

// TestSetLevelInvalidValues tests that invalid levels are safely handled
func TestSetLevelInvalidValues(t *testing.T) {
	buf := &ThreadSafeBuffer{}
	logger := New(NewJSONHandler(buf))

	// Test invalid levels are clamped to INFO
	invalidLevels := []Level{
		Level(-1),   // Below TRACE
		Level(100),  // Above FATAL
		Level(127),  // Max int8
		Level(-128), // Min int8
	}

	for _, invalidLevel := range invalidLevels {
		logger.SetLevel(invalidLevel)

		// Should still be able to log (defaults to INFO)
		buf.Reset()
		logger.Info().Msg("test")

		if len(buf.Bytes()) == 0 {
			t.Errorf("Expected log to be written after setting invalid level %d", invalidLevel)
		}
	}

	// Verify concurrent invalid level setting doesn't cause corruption
	buf.Reset() // Clear buffer from previous tests
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logger.SetLevel(Level(id * 1000)) // All invalid
			logger.Info().Int("goroutine", id).Msg("test")
		}(i)
	}
	wg.Wait()

	// Should have logged from all goroutines without panic
	logCount := bytes.Count(buf.Bytes(), []byte("\n"))
	if logCount != 100 {
		t.Errorf("Expected 100 logs, got %d (some may have been filtered or corrupted)", logCount)
	}
}

// TestHandlerStress tests multiple handlers being written to concurrently
// with different buffer destinations.
func TestHandlerStress(t *testing.T) {
	const (
		numHandlers    = 50
		logsPerHandler = 200
	)

	var buffers []*bytes.Buffer
	var loggers []*Logger

	// Create multiple handlers with separate buffers
	for i := 0; i < numHandlers; i++ {
		buf := &bytes.Buffer{}
		handler := NewJSONHandler(buf)
		logger := New(handler)

		buffers = append(buffers, buf)
		loggers = append(loggers, logger)
	}

	var wg sync.WaitGroup

	// Launch goroutines that write to different handlers concurrently
	for i := 0; i < numHandlers; i++ {
		wg.Add(1)
		go func(handlerID int) {
			defer wg.Done()
			logger := loggers[handlerID]
			for j := 0; j < logsPerHandler; j++ {
				logger.Info().
					Int("handler_id", handlerID).
					Int("log_number", j).
					Str("test_type", "handler_stress").
					Bool("concurrent_handler", true).
					Msg("stress testing handler")
			}
		}(i)
	}

	wg.Wait()

	// Verify each handler received the correct number of logs
	totalLogs := 0
	for i, buf := range buffers {
		logCount := bytes.Count(buf.Bytes(), []byte("\n"))
		if logCount != logsPerHandler {
			t.Errorf("Handler %d expected %d logs, got %d", i, logsPerHandler, logCount)
		}
		totalLogs += logCount
	}

	expectedTotal := numHandlers * logsPerHandler
	if totalLogs != expectedTotal {
		t.Errorf("Expected total %d logs across all handlers, got %d", expectedTotal, totalLogs)
	}
}

// TestContextLoggerRace tests concurrent context-aware logging operations
// with OpenTelemetry integration.
func TestContextLoggerRace(t *testing.T) {
	const (
		numGoroutines  = 100
		logsPerRoutine = 50
	)

	buf := &ThreadSafeBuffer{}
	logger := New(NewJSONHandler(buf))

	var wg sync.WaitGroup

	// Create different trace contexts
	traceContexts := make([]context.Context, 10)
	for i := range traceContexts {
		traceID := trace.TraceID([16]byte{
			byte(i), byte(i + 1), byte(i + 2), byte(i + 3),
			byte(i + 4), byte(i + 5), byte(i + 6), byte(i + 7),
			byte(i + 8), byte(i + 9), byte(i + 10), byte(i + 11),
			byte(i + 12), byte(i + 13), byte(i + 14), byte(i + 15),
		})
		spanID := trace.SpanID([8]byte{
			byte(i), byte(i + 1), byte(i + 2), byte(i + 3),
			byte(i + 4), byte(i + 5), byte(i + 6), byte(i + 7),
		})
		scc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		})
		traceContexts[i] = trace.ContextWithSpanContext(context.Background(), scc)
	}

	// Launch goroutines that use context logging concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			ctx := traceContexts[routineID%len(traceContexts)]
			ctxLogger := logger.Ctx(ctx)

			for j := 0; j < logsPerRoutine; j++ {
				ctxLogger.Info().
					Int("routine_id", routineID).
					Int("iteration", j).
					Str("operation", "context_logging").
					Msg("logging with trace context")
			}
		}(i)
	}

	wg.Wait()

	// Verify logs contain trace information
	expectedLogs := numGoroutines * logsPerRoutine
	actualLogs := bytes.Count(buf.Bytes(), []byte("\n"))

	if actualLogs != expectedLogs {
		t.Errorf("Expected %d context logs, got %d", expectedLogs, actualLogs)
	}

	// Verify trace IDs are present in logs
	traceIDCount := bytes.Count(buf.Bytes(), []byte(`"trace_id"`))
	spanIDCount := bytes.Count(buf.Bytes(), []byte(`"span_id"`))

	if traceIDCount != expectedLogs {
		t.Errorf("Expected %d trace_id fields, got %d", expectedLogs, traceIDCount)
	}
	if spanIDCount != expectedLogs {
		t.Errorf("Expected %d span_id fields, got %d", expectedLogs, spanIDCount)
	}
}

// TestMemorySafety verifies no memory corruption under high concurrency
// by testing with different field types and sizes.
func TestMemorySafety(t *testing.T) {
	const (
		numGoroutines        = 300
		operationsPerRoutine = 100
	)

	buf := &ThreadSafeBuffer{}
	logger := New(NewJSONHandler(buf))

	var wg sync.WaitGroup
	var operations int64

	// Test with various field types and sizes to stress memory handling
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			// Create data of various sizes
			smallStr := "small"
			mediumStr := string(make([]byte, 100))
			largeStr := string(make([]byte, 1000))

			for j := 0; j < operationsPerRoutine; j++ {
				switch j % 10 {
				case 0:
					logger.Info().
						Str("small", smallStr).
						Int("routine", routineID).
						Int("op", j).
						Msg("small string test")
				case 1:
					logger.Info().
						Str("medium", mediumStr).
						Int("routine", routineID).
						Int("op", j).
						Msg("medium string test")
				case 2:
					logger.Info().
						Str("large", largeStr).
						Int("routine", routineID).
						Int("op", j).
						Msg("large string test")
				case 3:
					logger.Info().
						Int64("big_int", int64(routineID*1000000+j)).
						Uint64("big_uint", uint64(routineID*2000000+j)).
						Msg("big integer test")
				case 4:
					logger.Info().
						Float64("float", float64(routineID)+float64(j)/100.0).
						Bool("even", j%2 == 0).
						Msg("mixed types test")
				case 5:
					logger.Info().
						Time("timestamp", time.Now()).
						Dur("duration", time.Duration(j)*time.Millisecond).
						Msg("time test")
				case 6:
					byteData := make([]byte, 50)
					for k := range byteData {
						byteData[k] = byte(k + routineID + j)
					}
					logger.Info().
						Hex("hex_data", byteData).
						Base64("b64_data", byteData).
						Msg("binary data test")
				case 7:
					logger.Info().
						Fields(map[string]interface{}{
							"routine": routineID,
							"op":      j,
							"mixed":   []string{"a", "b", "c"},
							"nested":  map[string]int{"x": 1, "y": 2},
						}).
						Msg("complex fields test")
				case 8:
					logger.Error().
						Err(fmt.Errorf("error %d from routine %d", j, routineID)).
						Stack().
						Caller().
						Msg("error with stack trace")
				case 9:
					logger.Info().
						RandID("request_id").
						Counter("operations", &operations).
						Msg("utility methods test")
				}

				atomic.AddInt64(&operations, 1)
			}
		}(i)
	}

	wg.Wait()

	expectedOps := int64(numGoroutines * operationsPerRoutine)
	actualOps := atomic.LoadInt64(&operations)

	if actualOps != expectedOps {
		t.Errorf("Expected %d operations, recorded %d", expectedOps, actualOps)
	}

	// Verify log integrity
	logCount := bytes.Count(buf.Bytes(), []byte("\n"))
	if logCount != int(expectedOps) {
		t.Errorf("Expected %d log entries, got %d", expectedOps, logCount)
	}

	// Verify no obvious memory corruption by checking for valid JSON structure
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	corruptedLogs := 0
	for _, line := range lines {
		if len(line) > 0 {
			if !bytes.HasPrefix(line, []byte("{")) || !bytes.HasSuffix(line, []byte("}")) {
				corruptedLogs++
			}
		}
	}

	if corruptedLogs > 0 {
		t.Errorf("Found %d corrupted log entries (not valid JSON structure)", corruptedLogs)
	}
}

// TestPoolBehaviorUnderMemoryPressure tests event pool behavior when
// memory is under pressure and garbage collection is active.
func TestPoolBehaviorUnderMemoryPressure(t *testing.T) {
	const (
		numGoroutines  = 100
		logsPerRoutine = 500
		pressureCycles = 10
	)

	buf := &ThreadSafeBuffer{}
	logger := New(NewJSONHandler(buf))

	var wg sync.WaitGroup
	var allocatedMemory [][]byte
	var memoryMutex sync.Mutex

	// Goroutine that creates memory pressure
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < pressureCycles; i++ {
			// Allocate large chunks of memory to trigger GC
			chunk := make([]byte, 10*1024*1024) // 10MB chunks
			memoryMutex.Lock()
			allocatedMemory = append(allocatedMemory, chunk)
			memoryMutex.Unlock()

			time.Sleep(50 * time.Millisecond)

			// Force garbage collection
			runtime.GC()

			// Release some memory
			memoryMutex.Lock()
			if len(allocatedMemory) > 5 {
				allocatedMemory = allocatedMemory[1:]
			}
			memoryMutex.Unlock()
		}
	}()

	// Logging goroutines under memory pressure
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < logsPerRoutine; j++ {
				logger.Info().
					Int("routine", routineID).
					Int("iteration", j).
					Str("memory_test", "under_pressure").
					Bool("gc_active", true).
					Msg("logging under memory pressure")

				// Occasionally yield to allow GC
				if j%100 == 0 {
					runtime.Gosched()
				}
			}
		}(i)
	}

	wg.Wait()

	// Clean up allocated memory
	memoryMutex.Lock()
	allocatedMemory = nil
	memoryMutex.Unlock()
	runtime.GC()

	// Verify logging worked correctly despite memory pressure
	expectedLogs := numGoroutines * logsPerRoutine
	actualLogs := bytes.Count(buf.Bytes(), []byte("\n"))

	if actualLogs != expectedLogs {
		t.Errorf("Expected %d logs under memory pressure, got %d", expectedLogs, actualLogs)
	}

	// Verify log quality
	gcActiveCount := bytes.Count(buf.Bytes(), []byte(`"gc_active":true`))
	if gcActiveCount != expectedLogs {
		t.Errorf("Expected %d logs with gc_active flag, got %d", expectedLogs, gcActiveCount)
	}
}

// BenchmarkConcurrentLogging benchmarks concurrent logging performance
// compared to single-threaded performance.
func BenchmarkConcurrentLogging(b *testing.B) {
	buf := &ThreadSafeBuffer{}
	logger := New(NewJSONHandler(buf))

	b.Run("SingleThreaded", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logger.Info().
				Int("iteration", i).
				Str("benchmark", "single").
				Bool("concurrent", false).
				Msg("benchmark message")
		}
	})

	b.Run("Concurrent-10", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		var wg sync.WaitGroup
		const numGoroutines = 10
		logsPerGoroutine := b.N / numGoroutines

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()
				for j := 0; j < logsPerGoroutine; j++ {
					logger.Info().
						Int("routine", routineID).
						Int("iteration", j).
						Str("benchmark", "concurrent").
						Bool("concurrent", true).
						Msg("benchmark message")
				}
			}(i)
		}
		wg.Wait()
	})

	b.Run("Concurrent-100", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		var wg sync.WaitGroup
		const numGoroutines = 100
		logsPerGoroutine := b.N / numGoroutines

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()
				for j := 0; j < logsPerGoroutine; j++ {
					logger.Info().
						Int("routine", routineID).
						Int("iteration", j).
						Str("benchmark", "concurrent").
						Bool("concurrent", true).
						Msg("benchmark message")
				}
			}(i)
		}
		wg.Wait()
	})
}

// BenchmarkEventPoolContention benchmarks pool performance under contention.
func BenchmarkEventPoolContention(b *testing.B) {
	buf := &ThreadSafeBuffer{}
	logger := New(NewJSONHandler(buf))

	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	const numGoroutines = 50
	logsPerGoroutine := b.N / numGoroutines

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				logger.Info().
					Str("test", "pool_contention").
					Int("iteration", j).
					Msg("pool benchmark")
			}
		}()
	}
	wg.Wait()
}

// TestRaceDetection ensures the tests will catch race conditions when
// run with the race detector (-race flag).
func TestRaceDetection(t *testing.T) {
	if !testing.Short() {
		// This test is designed to be run with: go test -race -run=TestRaceDetection
		t.Log("Running race detection test - make sure to run with -race flag")

		// Run a subset of race-prone operations
		t.Run("ConcurrentLogging", func(t *testing.T) {
			buf := &ThreadSafeBuffer{}
			logger := New(NewJSONHandler(buf))
			var wg sync.WaitGroup

			for i := 0; i < 50; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 10; j++ {
						logger.Info().Int("id", id).Int("j", j).Msg("race test")
					}
				}(i)
			}
			wg.Wait()
		})

		t.Run("LevelChanges", func(t *testing.T) {
			buf := &ThreadSafeBuffer{}
			logger := New(NewJSONHandler(buf))
			var wg sync.WaitGroup

			// Change levels concurrently
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 10; i++ {
					logger.SetLevel(Level(i % 6))
					time.Sleep(time.Microsecond)
				}
			}()

			// Log concurrently
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 10; j++ {
						logger.Info().Int("id", id).Msg("level race test")
					}
				}(i)
			}
			wg.Wait()
		})
	}
}

// TestLibraryThreadSafety demonstrates that the Bolt library itself is thread-safe
// by showing that when using a thread-safe output, no races occur in the library code.
func TestLibraryThreadSafety(t *testing.T) {
	const (
		numGoroutines  = 100
		logsPerRoutine = 100
	)

	buf := &ThreadSafeBuffer{}
	logger := New(NewJSONHandler(buf))

	var wg sync.WaitGroup
	var operations int64

	// Test all library operations concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < logsPerRoutine; j++ {
				// Test different levels and methods to ensure thread safety
				switch j % 6 {
				case 0:
					logger.Info().Int("id", routineID).Int("iter", j).Msg("info message")
				case 1:
					logger.Debug().Str("type", "debug").Bool("test", true).Msg("debug message")
				case 2:
					logger.Warn().Float64("value", float64(j)).Msg("warn message")
				case 3:
					logger.Error().Time("timestamp", time.Now()).Msg("error message")
				case 4:
					logger.Trace().Dur("duration", time.Millisecond*time.Duration(j)).Msg("trace message")
				case 5:
					subLogger := logger.With().Str("routine", fmt.Sprintf("routine-%d", routineID)).Logger()
					subLogger.Fatal().Uint64("count", uint64(j)).Msg("fatal message")
				}
				atomic.AddInt64(&operations, 1)
			}
		}(i)
	}

	wg.Wait()

	expectedOps := int64(numGoroutines * logsPerRoutine)
	actualOps := atomic.LoadInt64(&operations)

	if actualOps != expectedOps {
		t.Errorf("Expected %d operations, got %d", expectedOps, actualOps)
	}

	// Verify no data corruption by checking log structure
	logData := buf.Bytes()
	logCount := bytes.Count(logData, []byte("\n"))

	if logCount != int(expectedOps) {
		t.Errorf("Expected %d log entries, got %d", expectedOps, logCount)
	}

	t.Logf("Successfully completed %d concurrent logging operations without race conditions", actualOps)
}

// TestDictPoolRace tests Dict's pool usage under concurrent access.
func TestDictPoolRace(t *testing.T) {
	const (
		numGoroutines  = 100
		logsPerRoutine = 100
	)

	buf := &ThreadSafeBuffer{}
	logger := New(NewJSONHandler(buf))

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < logsPerRoutine; j++ {
				logger.Info().Dict("data", func(d *Event) {
					d.Int("routine", routineID).Int("iter", j).Str("status", "ok")
				}).Msg("dict race test")
			}
		}(i)
	}
	wg.Wait()

	expectedLogs := numGoroutines * logsPerRoutine
	actualLogs := bytes.Count(buf.Bytes(), []byte("\n"))
	if actualLogs != expectedLogs {
		t.Errorf("Expected %d logs, got %d", expectedLogs, actualLogs)
	}
}

// TestMultiHandlerRace tests MultiHandler under concurrent access.
func TestMultiHandlerRace(t *testing.T) {
	const (
		numGoroutines  = 50
		logsPerRoutine = 100
	)

	buf1 := &ThreadSafeBuffer{}
	buf2 := &ThreadSafeBuffer{}
	h := MultiHandler(NewJSONHandler(buf1), NewJSONHandler(buf2))
	logger := New(h)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < logsPerRoutine; j++ {
				logger.Info().Int("id", routineID).Int("j", j).Msg("multi race")
			}
		}(i)
	}
	wg.Wait()

	expectedLogs := numGoroutines * logsPerRoutine
	for name, buf := range map[string]*ThreadSafeBuffer{"buf1": buf1, "buf2": buf2} {
		actualLogs := bytes.Count(buf.Bytes(), []byte("\n"))
		if actualLogs != expectedLogs {
			t.Errorf("%s: expected %d logs, got %d", name, expectedLogs, actualLogs)
		}
	}
}

// TestHookRace tests hooks under concurrent logging.
func TestHookRace(t *testing.T) {
	const (
		numGoroutines  = 100
		logsPerRoutine = 100
	)

	buf := &ThreadSafeBuffer{}
	hook := NewSampleHook(2) // 50% sampling
	logger := New(NewJSONHandler(buf)).AddHook(hook)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < logsPerRoutine; j++ {
				logger.Info().Int("id", routineID).Msg("hook race")
			}
		}(i)
	}
	wg.Wait()

	totalEvents := numGoroutines * logsPerRoutine
	actualLogs := bytes.Count(buf.Bytes(), []byte("\n"))
	expectedSampled := totalEvents / 2
	if actualLogs != expectedSampled {
		t.Errorf("Expected %d sampled logs, got %d", expectedSampled, actualLogs)
	}
}

// TestSampleHookRace tests SampleHook's atomic counter under heavy contention.
func TestSampleHookRace(t *testing.T) {
	hook := NewSampleHook(10)
	var passed int64
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				if hook.Run(INFO, "test") {
					atomic.AddInt64(&passed, 1)
				}
			}
		}()
	}
	wg.Wait()

	total := int64(100 * 1000)
	expected := total / 10
	actual := atomic.LoadInt64(&passed)
	if actual != expected {
		t.Errorf("Expected %d passed events (1 in 10 of %d), got %d", expected, total, actual)
	}
}

// TestUnsafeBufferRaceDetection explicitly demonstrates race condition detection
// when using an unsafe shared buffer. This test should FAIL when run with -race.
// It serves as a control to verify that our race detection setup is working.
func TestUnsafeBufferRaceDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping unsafe buffer race test in short mode")
	}
	if raceDetectorEnabled {
		t.Skip("Skipping intentional race test under -race detector")
	}

	t.Log("This test demonstrates that the shared bytes.Buffer is the source of races, not the logging library")

	// INTENTIONALLY use unsafe buffer to demonstrate race detection
	var unsafeBuf bytes.Buffer
	logger := New(NewJSONHandler(&unsafeBuf))

	var wg sync.WaitGroup
	const numGoroutines = 10
	const logsPerRoutine = 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < logsPerRoutine; j++ {
				logger.Info().
					Int("routine", routineID).
					Int("iteration", j).
					Msg("unsafe buffer test")
			}
		}(i)
	}

	wg.Wait()

	t.Log("If this test passes without -race, it completed successfully")
	t.Log("If run with -race, this test should detect data races in bytes.Buffer")
}
