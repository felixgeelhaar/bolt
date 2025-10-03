// Package bolt performance regression tests
//
// This file contains performance regression tests to ensure Bolt maintains
// its zero-allocation and sub-100ns performance characteristics.
package bolt

import (
	"bytes"
	"testing"
	"time"
)

// Performance thresholds for regression detection
// Note: UTF-8 validation adds ~40ns overhead but provides critical security
const (
	MaxLatencyNs      = 120 // Maximum acceptable latency in nanoseconds (with UTF-8 validation)
	MaxAllocsPerOp    = 0   // Maximum allocations per operation (zero-allocation guarantee)
	MaxBytesPerOp     = 200 // Maximum bytes per operation (buffer overhead)
	Float64MaxLatency = 250 // Float64 is slightly slower due to precision formatting
)

// TestPerformanceRegression ensures core performance metrics don't regress
func TestPerformanceRegression(t *testing.T) {
	t.Run("Zero Allocations", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		result := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				buf.Reset()
				logger.Info().Str("hello", "world").Msg("test")
			}
		})

		if result.AllocsPerOp() > MaxAllocsPerOp {
			t.Errorf("Allocation regression detected: got %d allocs/op, want <= %d",
				result.AllocsPerOp(), MaxAllocsPerOp)
		}

		if result.NsPerOp() > MaxLatencyNs {
			t.Errorf("Latency regression detected: got %d ns/op, want <= %d",
				result.NsPerOp(), MaxLatencyNs)
		}

		if result.AllocedBytesPerOp() > MaxBytesPerOp {
			t.Errorf("Memory regression detected: got %d bytes/op, want <= %d",
				result.AllocedBytesPerOp(), MaxBytesPerOp)
		}

		t.Logf("Performance: %d ns/op, %d B/op, %d allocs/op",
			result.NsPerOp(), result.AllocedBytesPerOp(), result.AllocsPerOp())
	})

	t.Run("Float64 Zero Allocations", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		result := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				buf.Reset()
				logger.Info().Float64("value", 99.99).Msg("test")
			}
		})

		if result.AllocsPerOp() > MaxAllocsPerOp {
			t.Errorf("Float64 allocation regression detected: got %d allocs/op, want <= %d",
				result.AllocsPerOp(), MaxAllocsPerOp)
		}

		if result.NsPerOp() > Float64MaxLatency {
			t.Errorf("Float64 latency regression detected: got %d ns/op, want <= %d",
				result.NsPerOp(), Float64MaxLatency)
		}

		t.Logf("Float64 Performance: %d ns/op, %d B/op, %d allocs/op",
			result.NsPerOp(), result.AllocedBytesPerOp(), result.AllocsPerOp())
	})

	t.Run("Complex Event Zero Allocations", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		result := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				buf.Reset()
				logger.Info().
					Str("service", "api").
					Int("user_id", 12345).
					Bool("authenticated", true).
					Float64("latency", 123.456).
					Dur("timeout", 30*time.Second).
					Msg("request processed")
			}
		})

		if result.AllocsPerOp() > MaxAllocsPerOp {
			t.Errorf("Complex event allocation regression: got %d allocs/op, want <= %d",
				result.AllocsPerOp(), MaxAllocsPerOp)
		}

		// Complex events may be slightly slower but still under threshold
		maxComplexLatency := int64(350) // Allow up to 350ns for complex events (multiple fields + UTF-8 validation)
		if result.NsPerOp() > maxComplexLatency {
			t.Errorf("Complex event latency regression: got %d ns/op, want <= %d",
				result.NsPerOp(), maxComplexLatency)
		}

		t.Logf("Complex Event Performance: %d ns/op, %d B/op, %d allocs/op",
			result.NsPerOp(), result.AllocedBytesPerOp(), result.AllocsPerOp())
	})

	t.Run("All Field Types Zero Allocations", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		// Test each field type for zero allocations
		fieldTypes := map[string]func(){
			"Int":     func() { logger.Info().Int("key", 42).Msg("test") },
			"Int8":    func() { logger.Info().Int8("key", 42).Msg("test") },
			"Int16":   func() { logger.Info().Int16("key", 42).Msg("test") },
			"Int32":   func() { logger.Info().Int32("key", 42).Msg("test") },
			"Int64":   func() { logger.Info().Int64("key", 42).Msg("test") },
			"Uint":    func() { logger.Info().Uint("key", 42).Msg("test") },
			"Uint64":  func() { logger.Info().Uint64("key", 42).Msg("test") },
			"Bool":    func() { logger.Info().Bool("key", true).Msg("test") },
			"Float64": func() { logger.Info().Float64("key", 3.14).Msg("test") },
			"Str":     func() { logger.Info().Str("key", "value").Msg("test") },
			"Dur":     func() { logger.Info().Dur("key", time.Second).Msg("test") },
		}

		for name, fn := range fieldTypes {
			t.Run(name, func(t *testing.T) {
				result := testing.Benchmark(func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						buf.Reset()
						fn()
					}
				})

				if result.AllocsPerOp() > MaxAllocsPerOp {
					t.Errorf("%s allocation regression: got %d allocs/op, want <= %d",
						name, result.AllocsPerOp(), MaxAllocsPerOp)
				}

				t.Logf("%s Performance: %d ns/op, %d B/op, %d allocs/op",
					name, result.NsPerOp(), result.AllocedBytesPerOp(), result.AllocsPerOp())
			})
		}
	})
}

// TestPerformanceComparison provides baseline comparisons with stdlib
func TestPerformanceComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance comparison in short mode")
	}

	t.Run("Bolt vs Printf", func(t *testing.T) {
		var boltBuf, printfBuf bytes.Buffer
		logger := New(NewJSONHandler(&boltBuf))

		boltResult := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				boltBuf.Reset()
				logger.Info().Str("key", "value").Int("number", 42).Msg("test")
			}
		})

		printfResult := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				printfBuf.Reset()
				printfBuf.WriteString("level=info key=value number=42 message=test\n")
			}
		})

		speedup := float64(printfResult.NsPerOp()) / float64(boltResult.NsPerOp())
		t.Logf("Bolt: %d ns/op, %d allocs/op", boltResult.NsPerOp(), boltResult.AllocsPerOp())
		t.Logf("Printf: %d ns/op, %d allocs/op", printfResult.NsPerOp(), printfResult.AllocsPerOp())
		t.Logf("Speedup: %.2fx", speedup)
	})
}

// BenchmarkFloat64Formatting benchmarks the custom float64 formatter
func BenchmarkFloat64Formatting(b *testing.B) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	b.Run("SimpleFloat", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Float64("value", 99.99).Msg("test")
		}
	})

	b.Run("LargeFloat", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Float64("value", 123456789.123456).Msg("test")
		}
	})

	b.Run("SmallFloat", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Float64("value", 0.000001).Msg("test")
		}
	})

	b.Run("ScientificNotation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Float64("value", 1.23e15).Msg("test")
		}
	})
}

// BenchmarkNewFieldMethods benchmarks newly added field methods
func BenchmarkNewFieldMethods(b *testing.B) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	b.Run("Int32", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Int32("value", 42).Msg("test")
		}
	})

	b.Run("Int16", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Int16("value", 42).Msg("test")
		}
	})

	b.Run("Int8", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Int8("value", 42).Msg("test")
		}
	})
}

// BenchmarkEdgeCases benchmarks performance with edge case inputs
func BenchmarkEdgeCases(b *testing.B) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	b.Run("EmptyMessage", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Msg("")
		}
	})

	b.Run("LongMessage", func(b *testing.B) {
		longMsg := string(make([]byte, 1000))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Msg(longMsg)
		}
	})

	b.Run("UnicodeMessage", func(b *testing.B) {
		unicodeMsg := "Hello 世界 🚀 مرحبا"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Msg(unicodeMsg)
		}
	})

	b.Run("SpecialCharacters", func(b *testing.B) {
		specialMsg := "test\n\r\t\"\\"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Msg(specialMsg)
		}
	})

	b.Run("MinInt64", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Int64("value", -9223372036854775808).Msg("test")
		}
	})

	b.Run("MaxInt64", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Int64("value", 9223372036854775807).Msg("test")
		}
	})
}

// BenchmarkHandlers benchmarks different handler types
func BenchmarkHandlers(b *testing.B) {
	var buf bytes.Buffer

	b.Run("JSONHandler", func(b *testing.B) {
		logger := New(NewJSONHandler(&buf))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Str("key", "value").Int("num", 42).Msg("test")
		}
	})

	b.Run("ConsoleHandler", func(b *testing.B) {
		logger := New(NewConsoleHandler(&buf))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Str("key", "value").Int("num", 42).Msg("test")
		}
	})
}

// BenchmarkLogLevels benchmarks different log levels
func BenchmarkLogLevels(b *testing.B) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	b.Run("Trace", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Trace().Msg("trace message")
		}
	})

	b.Run("Debug", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Debug().Msg("debug message")
		}
	})

	b.Run("Info", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Msg("info message")
		}
	})

	b.Run("Warn", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Warn().Msg("warn message")
		}
	})

	b.Run("Error", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Error().Msg("error message")
		}
	})
}

// BenchmarkParallelPerformance benchmarks parallel logging performance
func BenchmarkParallelPerformance(b *testing.B) {
	b.Run("Serial", func(b *testing.B) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Int("id", i).Msg("serial")
		}
	})

	b.Run("ParallelPerGoroutine", func(b *testing.B) {
		// Note: Each goroutine needs its own buffer for thread safety
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			var buf bytes.Buffer
			logger := New(NewJSONHandler(&buf))
			id := 0
			for pb.Next() {
				buf.Reset()
				logger.Info().Int("id", id).Msg("parallel")
				id++
			}
		})
	})
}

// BenchmarkFieldChaining benchmarks field chaining patterns
func BenchmarkFieldChaining(b *testing.B) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	b.Run("SingleField", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Str("key", "value").Msg("test")
		}
	})

	b.Run("TwoFields", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().
				Str("key1", "value1").
				Str("key2", "value2").
				Msg("test")
		}
	})

	b.Run("FiveFields", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().
				Str("key1", "value1").
				Str("key2", "value2").
				Str("key3", "value3").
				Str("key4", "value4").
				Str("key5", "value5").
				Msg("test")
		}
	})

	b.Run("TenFields", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().
				Str("key1", "value1").
				Str("key2", "value2").
				Str("key3", "value3").
				Str("key4", "value4").
				Str("key5", "value5").
				Str("key6", "value6").
				Str("key7", "value7").
				Str("key8", "value8").
				Str("key9", "value9").
				Str("key10", "value10").
				Msg("test")
		}
	})
}

// BenchmarkProductionPatterns benchmarks common production usage patterns
func BenchmarkProductionPatterns(b *testing.B) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	b.Run("HTTPRequest", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().
				Str("method", "GET").
				Str("path", "/api/users").
				Int("status", 200).
				Dur("latency", 45*time.Millisecond).
				Str("ip", "192.168.1.1").
				Msg("request completed")
		}
	})

	b.Run("DatabaseQuery", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().
				Str("query", "SELECT * FROM users WHERE id = ?").
				Int("rows", 1).
				Dur("duration", 12*time.Millisecond).
				Str("database", "postgres").
				Msg("query executed")
		}
	})

	b.Run("ErrorWithContext", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Error().
				Str("error", "connection refused").
				Str("service", "redis").
				Int("retry_count", 3).
				Str("user_id", "12345").
				Msg("service unavailable")
		}
	})

	b.Run("MetricsLog", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().
				Str("metric", "requests_total").
				Float64("value", 12345.67).
				Str("endpoint", "/api/data").
				Timestamp().
				Msg("metric recorded")
		}
	})
}

// BenchmarkMemoryPressure benchmarks behavior under memory pressure
func BenchmarkMemoryPressure(b *testing.B) {
	b.Run("SmallBuffer", func(b *testing.B) {
		buf := bytes.NewBuffer(make([]byte, 0, 128))
		logger := New(NewJSONHandler(buf))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Str("key", "value").Msg("test")
		}
	})

	b.Run("MediumBuffer", func(b *testing.B) {
		buf := bytes.NewBuffer(make([]byte, 0, 1024))
		logger := New(NewJSONHandler(buf))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Str("key", "value").Msg("test")
		}
	})

	b.Run("LargeBuffer", func(b *testing.B) {
		buf := bytes.NewBuffer(make([]byte, 0, 8192))
		logger := New(NewJSONHandler(buf))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Str("key", "value").Msg("test")
		}
	})
}
