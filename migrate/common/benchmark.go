// Package common provides shared utilities for all migration tools.
package common

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/felixgeelhaar/bolt"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

// BenchmarkResult holds the results of a logging library benchmark.
type BenchmarkResult struct {
	Library     string        `json:"library"`
	NsPerOp     int64         `json:"ns_per_op"`
	AllocsPerOp int64         `json:"allocs_per_op"`
	BytesPerOp  int64         `json:"bytes_per_op"`
	Duration    time.Duration `json:"duration"`
	Operations  int           `json:"operations"`
}

// BenchmarkSuite runs comprehensive benchmarks comparing different logging libraries.
type BenchmarkSuite struct {
	results []BenchmarkResult
	mu      sync.Mutex
}

// NewBenchmarkSuite creates a new benchmark suite.
func NewBenchmarkSuite() *BenchmarkSuite {
	return &BenchmarkSuite{
		results: make([]BenchmarkResult, 0),
	}
}

// RunComparison runs a comprehensive comparison of all logging libraries.
func (bs *BenchmarkSuite) RunComparison() []BenchmarkResult {
	// Setup benchmarks for each library
	benchmarks := []struct {
		name string
		fn   func(*testing.B)
	}{
		{"Bolt", bs.benchmarkBolt},
		{"Zerolog", bs.benchmarkZerolog},
		{"Zap", bs.benchmarkZap},
		{"Logrus", bs.benchmarkLogrus},
	}

	for _, bench := range benchmarks {
		result := testing.Benchmark(bench.fn)
		bs.mu.Lock()
		bs.results = append(bs.results, BenchmarkResult{
			Library:     bench.name,
			NsPerOp:     result.NsPerOp(),
			AllocsPerOp: result.AllocsPerOp(),
			BytesPerOp:  result.AllocedBytesPerOp(),
			Duration:    result.T,
			Operations:  result.N,
		})
		bs.mu.Unlock()
	}

	return bs.results
}

// benchmarkBolt benchmarks the Bolt logging library.
func (bs *BenchmarkSuite) benchmarkBolt(b *testing.B) {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	
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
}

// benchmarkZerolog benchmarks the Zerolog library.
func (bs *BenchmarkSuite) benchmarkZerolog(b *testing.B) {
	logger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)
	
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
}

// benchmarkZap benchmarks the Zap library.
func (bs *BenchmarkSuite) benchmarkZap(b *testing.B) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	logger, _ := config.Build()
	defer logger.Sync()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		logger.Info("User authenticated",
			zap.String("service", "auth"),
			zap.Int("user_id", 12345),
			zap.String("action", "login"),
			zap.Bool("success", true),
			zap.Float64("duration", 1.23),
		)
	}
}

// benchmarkLogrus benchmarks the Logrus library.
func (bs *BenchmarkSuite) benchmarkLogrus(b *testing.B) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		logger.WithFields(logrus.Fields{
			"service":  "auth",
			"user_id":  12345,
			"action":   "login",
			"success":  true,
			"duration": 1.23,
		}).Info("User authenticated")
	}
}

// PrintResults prints the benchmark results in a formatted table.
func (bs *BenchmarkSuite) PrintResults() {
	fmt.Println("\n=== Logging Library Performance Comparison ===")
	fmt.Printf("%-10s | %-12s | %-12s | %-12s | %-10s\n", 
		"Library", "ns/op", "allocs/op", "bytes/op", "ops")
	fmt.Println(strings.Repeat("-", 70))
	
	for _, result := range bs.results {
		fmt.Printf("%-10s | %-12d | %-12d | %-12d | %-10d\n",
			result.Library,
			result.NsPerOp,
			result.AllocsPerOp,
			result.BytesPerOp,
			result.Operations,
		)
	}
	
	// Calculate and display performance improvements
	if len(bs.results) > 1 {
		bolt := bs.results[0] // Assuming Bolt is first
		fmt.Println("\n=== Performance Improvements vs Bolt ===")
		for i := 1; i < len(bs.results); i++ {
			other := bs.results[i]
			improvement := float64(other.NsPerOp-bolt.NsPerOp) / float64(bolt.NsPerOp) * 100
			fmt.Printf("%s: %.1f%% %s\n", 
				other.Library, 
				abs(improvement),
				func() string {
					if improvement > 0 {
						return "slower"
					}
					return "faster"
				}(),
			)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// MemoryProfiler helps analyze memory usage patterns.
type MemoryProfiler struct {
	startMem runtime.MemStats
	endMem   runtime.MemStats
}

// Start begins memory profiling.
func (mp *MemoryProfiler) Start() {
	runtime.GC()
	runtime.ReadMemStats(&mp.startMem)
}

// Stop ends memory profiling and returns memory statistics.
func (mp *MemoryProfiler) Stop() (allocsDelta, bytesDelta uint64) {
	runtime.ReadMemStats(&mp.endMem)
	allocsDelta = mp.endMem.Mallocs - mp.startMem.Mallocs
	bytesDelta = mp.endMem.TotalAlloc - mp.startMem.TotalAlloc
	return
}

// LoadTester simulates concurrent logging load.
type LoadTester struct {
	loggerFunc func(context.Context, string, ...interface{})
	goroutines int
	operations int
}

// NewLoadTester creates a new load tester.
func NewLoadTester(loggerFunc func(context.Context, string, ...interface{}), goroutines, operations int) *LoadTester {
	return &LoadTester{
		loggerFunc: loggerFunc,
		goroutines: goroutines,
		operations: operations,
	}
}

// Run executes the load test.
func (lt *LoadTester) Run() time.Duration {
	var wg sync.WaitGroup
	ctx := context.Background()
	
	start := time.Now()
	
	for i := 0; i < lt.goroutines; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < lt.operations; j++ {
				lt.loggerFunc(ctx, "Load test message", 
					"worker", worker, 
					"operation", j,
					"timestamp", time.Now(),
				)
			}
		}(i)
	}
	
	wg.Wait()
	return time.Since(start)
}