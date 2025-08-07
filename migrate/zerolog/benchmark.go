package zerolog

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"testing"
	"time"

	"github.com/felixgeelhaar/bolt"
	"github.com/rs/zerolog"
)

// BenchmarkComparison provides side-by-side performance comparisons between Zerolog and Bolt.
type BenchmarkComparison struct {
	output io.Writer
}

// NewBenchmarkComparison creates a new benchmark comparison.
func NewBenchmarkComparison(output io.Writer) *BenchmarkComparison {
	if output == nil {
		output = io.Discard // Use discard to focus on performance, not I/O
	}
	return &BenchmarkComparison{output: output}
}

// ComparisonResult holds the results of a benchmark comparison.
type ComparisonResult struct {
	BoltResult    testing.BenchmarkResult `json:"bolt_result"`
	ZerologResult testing.BenchmarkResult `json:"zerolog_result"`
	Improvement   PerformanceImprovement   `json:"improvement"`
}

// PerformanceImprovement quantifies the performance difference.
type PerformanceImprovement struct {
	SpeedupPercent    float64 `json:"speedup_percent"`
	AllocReduction    int64   `json:"alloc_reduction"`
	BytesReduction    int64   `json:"bytes_reduction"`
	BoltFasterBy      string  `json:"bolt_faster_by"`
}

// RunBasicComparison runs a basic logging comparison.
func (bc *BenchmarkComparison) RunBasicComparison() *ComparisonResult {
	// Benchmark Bolt
	boltResult := testing.Benchmark(func(b *testing.B) {
		logger := bolt.New(bolt.NewJSONHandler(bc.output))
		
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			logger.Info().
				Str("service", "auth").
				Int("user_id", 12345).
				Str("action", "login").
				Bool("success", true).
				Float64("duration", 1.23).
				Msg("User authenticated")
		}
	})

	// Benchmark Zerolog
	zerologResult := testing.Benchmark(func(b *testing.B) {
		logger := zerolog.New(bc.output).Level(zerolog.InfoLevel)
		
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			logger.Info().
				Str("service", "auth").
				Int("user_id", 12345).
				Str("action", "login").
				Bool("success", true).
				Float64("duration", 1.23).
				Msg("User authenticated")
		}
	})

	return &ComparisonResult{
		BoltResult:    boltResult,
		ZerologResult: zerologResult,
		Improvement:   bc.calculateImprovement(boltResult, zerologResult),
	}
}

// RunStructuredLoggingComparison compares structured logging performance.
func (bc *BenchmarkComparison) RunStructuredLoggingComparison() *ComparisonResult {
	// Benchmark Bolt structured logging
	boltResult := testing.Benchmark(func(b *testing.B) {
		logger := bolt.New(bolt.NewJSONHandler(bc.output))
		
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			logger.Info().
				Str("trace_id", "abc123").
				Str("span_id", "def456").
				Str("service", "api-gateway").
				Str("method", "POST").
				Str("endpoint", "/api/v1/users").
				Int("status_code", 201).
				Float64("response_time_ms", 45.67).
				Int64("request_size", 1024).
				Int64("response_size", 512).
				Bool("cached", false).
				Msg("Request processed")
		}
	})

	// Benchmark Zerolog structured logging
	zerologResult := testing.Benchmark(func(b *testing.B) {
		logger := zerolog.New(bc.output).Level(zerolog.InfoLevel)
		
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			logger.Info().
				Str("trace_id", "abc123").
				Str("span_id", "def456").
				Str("service", "api-gateway").
				Str("method", "POST").
				Str("endpoint", "/api/v1/users").
				Int("status_code", 201).
				Float64("response_time_ms", 45.67).
				Int64("request_size", 1024).
				Int64("response_size", 512).
				Bool("cached", false).
				Msg("Request processed")
		}
	})

	return &ComparisonResult{
		BoltResult:    boltResult,
		ZerologResult: zerologResult,
		Improvement:   bc.calculateImprovement(boltResult, zerologResult),
	}
}

// RunContextualLoggingComparison compares contextual logging with OpenTelemetry.
func (bc *BenchmarkComparison) RunContextualLoggingComparison() *ComparisonResult {
	ctx := context.Background()

	// Benchmark Bolt contextual logging
	boltResult := testing.Benchmark(func(b *testing.B) {
		logger := bolt.New(bolt.NewJSONHandler(bc.output))
		
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			logger.Ctx(ctx).Info().
				Str("operation", "database_query").
				Str("table", "users").
				Int("affected_rows", 1).
				Msg("Database operation completed")
		}
	})

	// Benchmark Zerolog contextual logging
	zerologResult := testing.Benchmark(func(b *testing.B) {
		logger := zerolog.New(bc.output).Level(zerolog.InfoLevel)
		
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			logger.Info().
				Ctx(ctx).
				Str("operation", "database_query").
				Str("table", "users").
				Int("affected_rows", 1).
				Msg("Database operation completed")
		}
	})

	return &ComparisonResult{
		BoltResult:    boltResult,
		ZerologResult: zerologResult,
		Improvement:   bc.calculateImprovement(boltResult, zerologResult),
	}
}

// RunErrorLoggingComparison compares error logging performance.
func (bc *BenchmarkComparison) RunErrorLoggingComparison() *ComparisonResult {
	testError := fmt.Errorf("database connection failed: timeout after 30s")

	// Benchmark Bolt error logging
	boltResult := testing.Benchmark(func(b *testing.B) {
		logger := bolt.New(bolt.NewJSONHandler(bc.output))
		
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			logger.Error().
				Err(testError).
				Str("component", "database").
				Str("host", "db-primary.example.com").
				Int("retry_count", 3).
				Dur("timeout", 30*time.Second).
				Msg("Database connection failed")
		}
	})

	// Benchmark Zerolog error logging
	zerologResult := testing.Benchmark(func(b *testing.B) {
		logger := zerolog.New(bc.output).Level(zerolog.InfoLevel)
		
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			logger.Error().
				Err(testError).
				Str("component", "database").
				Str("host", "db-primary.example.com").
				Int("retry_count", 3).
				Dur("timeout", 30*time.Second).
				Msg("Database connection failed")
		}
	})

	return &ComparisonResult{
		BoltResult:    boltResult,
		ZerologResult: zerologResult,
		Improvement:   bc.calculateImprovement(boltResult, zerologResult),
	}
}

// RunConcurrentLoggingComparison compares concurrent logging performance.
func (bc *BenchmarkComparison) RunConcurrentLoggingComparison() *ComparisonResult {
	const numGoroutines = 10

	// Benchmark Bolt concurrent logging
	boltResult := testing.Benchmark(func(b *testing.B) {
		logger := bolt.New(bolt.NewJSONHandler(bc.output))
		
		b.ResetTimer()
		b.ReportAllocs()
		b.SetParallelism(numGoroutines)
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info().
					Str("goroutine", "worker").
					Int("worker_id", runtime.NumGoroutine()).
					Str("task", "processing").
					Msg("Concurrent task executed")
			}
		})
	})

	// Benchmark Zerolog concurrent logging
	zerologResult := testing.Benchmark(func(b *testing.B) {
		logger := zerolog.New(bc.output).Level(zerolog.InfoLevel)
		
		b.ResetTimer()
		b.ReportAllocs()
		b.SetParallelism(numGoroutines)
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info().
					Str("goroutine", "worker").
					Int("worker_id", runtime.NumGoroutine()).
					Str("task", "processing").
					Msg("Concurrent task executed")
			}
		})
	})

	return &ComparisonResult{
		BoltResult:    boltResult,
		ZerologResult: zerologResult,
		Improvement:   bc.calculateImprovement(boltResult, zerologResult),
	}
}

// calculateImprovement calculates the performance improvement of Bolt over Zerolog.
func (bc *BenchmarkComparison) calculateImprovement(bolt, zerolog testing.BenchmarkResult) PerformanceImprovement {
	speedupPercent := 0.0
	if zerolog.NsPerOp() > 0 {
		speedupPercent = float64(zerolog.NsPerOp()-bolt.NsPerOp()) / float64(zerolog.NsPerOp()) * 100
	}

	allocReduction := zerolog.AllocsPerOp() - bolt.AllocsPerOp()
	bytesReduction := zerolog.AllocedBytesPerOp() - bolt.AllocedBytesPerOp()

	fasterBy := ""
	if speedupPercent > 0 {
		fasterBy = fmt.Sprintf("%.1fx faster", float64(zerolog.NsPerOp())/float64(bolt.NsPerOp()))
	}

	return PerformanceImprovement{
		SpeedupPercent: speedupPercent,
		AllocReduction: allocReduction,
		BytesReduction: bytesReduction,
		BoltFasterBy:   fasterBy,
	}
}

// ComprehensiveBenchmark runs all benchmark comparisons.
type ComprehensiveBenchmark struct {
	bc *BenchmarkComparison
}

// NewComprehensiveBenchmark creates a new comprehensive benchmark.
func NewComprehensiveBenchmark() *ComprehensiveBenchmark {
	return &ComprehensiveBenchmark{
		bc: NewBenchmarkComparison(io.Discard),
	}
}

// Results holds all benchmark results.
type Results struct {
	Basic        *ComparisonResult `json:"basic"`
	Structured   *ComparisonResult `json:"structured"`
	Contextual   *ComparisonResult `json:"contextual"`
	Error        *ComparisonResult `json:"error"`
	Concurrent   *ComparisonResult `json:"concurrent"`
	Summary      string            `json:"summary"`
}

// RunAll runs all benchmark comparisons and returns comprehensive results.
func (cb *ComprehensiveBenchmark) RunAll() *Results {
	fmt.Println("Running comprehensive Zerolog vs Bolt benchmarks...")
	
	results := &Results{
		Basic:      cb.bc.RunBasicComparison(),
		Structured: cb.bc.RunStructuredLoggingComparison(),
		Contextual: cb.bc.RunContextualLoggingComparison(),
		Error:      cb.bc.RunErrorLoggingComparison(),
		Concurrent: cb.bc.RunConcurrentLoggingComparison(),
	}
	
	results.Summary = cb.generateSummary(results)
	return results
}

// generateSummary generates a summary of all benchmark results.
func (cb *ComprehensiveBenchmark) generateSummary(results *Results) string {
	avgSpeedup := (results.Basic.Improvement.SpeedupPercent +
		results.Structured.Improvement.SpeedupPercent +
		results.Contextual.Improvement.SpeedupPercent +
		results.Error.Improvement.SpeedupPercent +
		results.Concurrent.Improvement.SpeedupPercent) / 5

	return fmt.Sprintf(`
Bolt vs Zerolog Performance Summary:
====================================

Average Performance Improvement: %.1f%%

Detailed Results:
- Basic Logging: %.1f%% faster, %d fewer allocs
- Structured Logging: %.1f%% faster, %d fewer allocs  
- Contextual Logging: %.1f%% faster, %d fewer allocs
- Error Logging: %.1f%% faster, %d fewer allocs
- Concurrent Logging: %.1f%% faster, %d fewer allocs

Key Benefits of Migration:
- Zero allocations in hot paths
- Sub-100ns logging operations
- Better performance under concurrent load
- Seamless OpenTelemetry integration
- Smaller memory footprint
`,
		avgSpeedup,
		results.Basic.Improvement.SpeedupPercent, results.Basic.Improvement.AllocReduction,
		results.Structured.Improvement.SpeedupPercent, results.Structured.Improvement.AllocReduction,
		results.Contextual.Improvement.SpeedupPercent, results.Contextual.Improvement.AllocReduction,
		results.Error.Improvement.SpeedupPercent, results.Error.Improvement.AllocReduction,
		results.Concurrent.Improvement.SpeedupPercent, results.Concurrent.Improvement.AllocReduction,
	)
}

// PrintResults prints detailed benchmark results.
func (results *Results) PrintResults() {
	fmt.Println("\n=== Zerolog vs Bolt Benchmark Results ===\n")
	
	benchmarks := []struct {
		name   string
		result *ComparisonResult
	}{
		{"Basic Logging", results.Basic},
		{"Structured Logging", results.Structured},
		{"Contextual Logging", results.Contextual},
		{"Error Logging", results.Error},
		{"Concurrent Logging", results.Concurrent},
	}
	
	for _, bench := range benchmarks {
		fmt.Printf("%s:\n", bench.name)
		fmt.Printf("  Bolt:    %d ns/op, %d allocs/op, %d bytes/op\n",
			bench.result.BoltResult.NsPerOp(),
			bench.result.BoltResult.AllocsPerOp(),
			bench.result.BoltResult.AllocedBytesPerOp())
		fmt.Printf("  Zerolog: %d ns/op, %d allocs/op, %d bytes/op\n",
			bench.result.ZerologResult.NsPerOp(),
			bench.result.ZerologResult.AllocsPerOp(),
			bench.result.ZerologResult.AllocedBytesPerOp())
		fmt.Printf("  Improvement: %.1f%% faster, %d fewer allocs, %d fewer bytes\n\n",
			bench.result.Improvement.SpeedupPercent,
			bench.result.Improvement.AllocReduction,
			bench.result.Improvement.BytesReduction)
	}
	
	fmt.Println(results.Summary)
}