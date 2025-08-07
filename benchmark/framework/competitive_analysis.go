// Package framework provides comprehensive benchmarking tools for competitive analysis
// of logging libraries with statistical significance and enterprise-grade validation.
package framework

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/felixgeelhaar/bolt/v2"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/slog"
)

// LibraryType represents different logging libraries for benchmarking
type LibraryType int

const (
	LibraryBolt LibraryType = iota
	LibraryZerolog
	LibraryZap
	LibraryLogrus
	LibrarySlog
)

func (lt LibraryType) String() string {
	switch lt {
	case LibraryBolt:
		return "Bolt"
	case LibraryZerolog:
		return "Zerolog"
	case LibraryZap:
		return "Zap"
	case LibraryLogrus:
		return "Logrus"
	case LibrarySlog:
		return "Slog"
	default:
		return "Unknown"
	}
}

// BenchmarkResult holds comprehensive performance metrics for a single benchmark run
type BenchmarkResult struct {
	Library        string        `json:"library"`
	TestName       string        `json:"test_name"`
	NsPerOp        float64       `json:"ns_per_op"`
	AllocsPerOp    float64       `json:"allocs_per_op"`
	BytesPerOp     float64       `json:"bytes_per_op"`
	MBPerSec       float64       `json:"mb_per_sec"`
	Duration       time.Duration `json:"duration"`
	Operations     int           `json:"operations"`
	Iterations     int           `json:"iterations"`
	CPUs           int           `json:"cpus"`
	GoroutineCount int           `json:"goroutine_count"`

	// Statistical metrics
	NsPerOpStdDev      float64            `json:"ns_per_op_stddev"`
	NsPerOpMin         float64            `json:"ns_per_op_min"`
	NsPerOpMax         float64            `json:"ns_per_op_max"`
	NsPerOpPercentiles map[string]float64 `json:"ns_per_op_percentiles"`

	// System metrics
	GoVersion string    `json:"go_version"`
	GOOS      string    `json:"goos"`
	GOARCH    string    `json:"goarch"`
	Timestamp time.Time `json:"timestamp"`

	// Memory analysis
	HeapAllocs   uint64        `json:"heap_allocs"`
	HeapSys      uint64        `json:"heap_sys"`
	GCCount      uint32        `json:"gc_count"`
	GCPauseTotal time.Duration `json:"gc_pause_total"`
}

// TestScenario defines different testing scenarios for realistic benchmarking
type TestScenario struct {
	Name        string
	Description string
	Fields      int
	Concurrency int
	MessageSize string // small, medium, large
	LogLevel    string
	Duration    time.Duration
}

// PredefinedScenarios provides realistic logging scenarios for benchmarking
var PredefinedScenarios = []TestScenario{
	{
		Name:        "WebAPI",
		Description: "Typical web API logging with request/response data",
		Fields:      5,
		Concurrency: 10,
		MessageSize: "medium",
		LogLevel:    "info",
		Duration:    5 * time.Second,
	},
	{
		Name:        "Microservice",
		Description: "High-frequency microservice internal logging",
		Fields:      3,
		Concurrency: 50,
		MessageSize: "small",
		LogLevel:    "debug",
		Duration:    10 * time.Second,
	},
	{
		Name:        "HighFrequencyTrading",
		Description: "Ultra-low latency trading system logging",
		Fields:      2,
		Concurrency: 1,
		MessageSize: "small",
		LogLevel:    "info",
		Duration:    30 * time.Second,
	},
	{
		Name:        "DataPipeline",
		Description: "Batch processing with detailed logging",
		Fields:      10,
		Concurrency: 20,
		MessageSize: "large",
		LogLevel:    "info",
		Duration:    15 * time.Second,
	},
	{
		Name:        "ContainerOrchestration",
		Description: "Container runtime logging with metadata",
		Fields:      8,
		Concurrency: 100,
		MessageSize: "medium",
		LogLevel:    "warn",
		Duration:    20 * time.Second,
	},
}

// CompetitiveAnalyzer provides comprehensive analysis of logging library performance
type CompetitiveAnalyzer struct {
	libraries []LibraryType
	scenarios []TestScenario
	results   sync.Map // map[string][]BenchmarkResult
	baseline  LibraryType

	// Configuration
	iterations      int
	warmupDuration  time.Duration
	enableProfiling bool
	enableGCMetrics bool

	// Output
	outputDir string
	buffer    *bytes.Buffer
}

// NewCompetitiveAnalyzer creates a new analyzer with default configuration
func NewCompetitiveAnalyzer() *CompetitiveAnalyzer {
	return &CompetitiveAnalyzer{
		libraries: []LibraryType{
			LibraryBolt,
			LibraryZerolog,
			LibraryZap,
			LibraryLogrus,
			LibrarySlog,
		},
		scenarios:       PredefinedScenarios,
		baseline:        LibraryBolt,
		iterations:      10,
		warmupDuration:  2 * time.Second,
		enableProfiling: true,
		enableGCMetrics: true,
		outputDir:       "benchmark-results",
		buffer:          bytes.NewBuffer(make([]byte, 0, 1024*1024)), // 1MB buffer
	}
}

// WithLibraries configures which libraries to benchmark
func (ca *CompetitiveAnalyzer) WithLibraries(libraries ...LibraryType) *CompetitiveAnalyzer {
	ca.libraries = libraries
	return ca
}

// WithScenarios configures which scenarios to test
func (ca *CompetitiveAnalyzer) WithScenarios(scenarios ...TestScenario) *CompetitiveAnalyzer {
	ca.scenarios = scenarios
	return ca
}

// WithBaseline sets the baseline library for comparisons
func (ca *CompetitiveAnalyzer) WithBaseline(baseline LibraryType) *CompetitiveAnalyzer {
	ca.baseline = baseline
	return ca
}

// WithIterations sets the number of benchmark iterations
func (ca *CompetitiveAnalyzer) WithIterations(iterations int) *CompetitiveAnalyzer {
	ca.iterations = iterations
	return ca
}

// WithProfiling enables or disables profiling
func (ca *CompetitiveAnalyzer) WithProfiling(enable bool) *CompetitiveAnalyzer {
	ca.enableProfiling = enable
	return ca
}

// WithOutputDir sets the output directory for results
func (ca *CompetitiveAnalyzer) WithOutputDir(dir string) *CompetitiveAnalyzer {
	ca.outputDir = dir
	return ca
}

// RunComprehensiveAnalysis executes all configured benchmarks and generates analysis
func (ca *CompetitiveAnalyzer) RunComprehensiveAnalysis(ctx context.Context) error {
	// Ensure output directory exists
	if err := os.MkdirAll(ca.outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	fmt.Printf("🚀 Starting comprehensive competitive analysis\n")
	fmt.Printf("Libraries: %v\n", ca.libraryNames())
	fmt.Printf("Scenarios: %v\n", ca.scenarioNames())
	fmt.Printf("Iterations per test: %d\n", ca.iterations)
	fmt.Printf("Output directory: %s\n\n", ca.outputDir)

	// Run benchmarks for each combination
	totalTests := len(ca.libraries) * len(ca.scenarios)
	currentTest := 0

	for _, library := range ca.libraries {
		for _, scenario := range ca.scenarios {
			currentTest++
			fmt.Printf("[%d/%d] Testing %s with %s scenario...\n",
				currentTest, totalTests, library, scenario.Name)

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			results, err := ca.runScenarioBenchmark(library, scenario)
			if err != nil {
				fmt.Printf("❌ Failed to run benchmark: %v\n", err)
				continue
			}

			key := fmt.Sprintf("%s-%s", library, scenario.Name)
			ca.results.Store(key, results)

			fmt.Printf("✅ Completed: %s - %s (%.2fns/op, %d allocs/op)\n",
				library, scenario.Name,
				results[len(results)-1].NsPerOp,
				int(results[len(results)-1].AllocsPerOp))
		}
	}

	fmt.Printf("\n📊 Analysis complete! Generating reports...\n")
	return ca.generateAnalysisReports()
}

// runScenarioBenchmark runs a benchmark for a specific library and scenario
func (ca *CompetitiveAnalyzer) runScenarioBenchmark(library LibraryType, scenario TestScenario) ([]BenchmarkResult, error) {
	var results []BenchmarkResult

	// Setup logger for the specific library
	logger, err := ca.setupLogger(library)
	if err != nil {
		return nil, fmt.Errorf("failed to setup logger: %w", err)
	}

	// Create benchmark function
	benchmarkFunc := ca.createBenchmarkFunc(library, logger, scenario)

	// Run multiple iterations for statistical significance
	for i := 0; i < ca.iterations; i++ {
		result := testing.Benchmark(benchmarkFunc)

		// Capture additional metrics
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		benchResult := BenchmarkResult{
			Library:        library.String(),
			TestName:       scenario.Name,
			NsPerOp:        float64(result.NsPerOp()),
			AllocsPerOp:    float64(result.AllocsPerOp()),
			BytesPerOp:     float64(result.AllocedBytesPerOp()),
			MBPerSec:       float64(result.AllocedBytesPerOp()) / 1024 / 1024,
			Duration:       result.T,
			Operations:     result.N,
			Iterations:     i + 1,
			CPUs:           runtime.NumCPU(),
			GoroutineCount: runtime.NumGoroutine(),
			GoVersion:      runtime.Version(),
			GOOS:           runtime.GOOS,
			GOARCH:         runtime.GOARCH,
			Timestamp:      time.Now(),
			HeapAllocs:     memStats.HeapAlloc,
			HeapSys:        memStats.HeapSys,
			GCCount:        memStats.NumGC,
			GCPauseTotal:   time.Duration(int64(memStats.PauseTotalNs)),
		}

		results = append(results, benchResult)

		// Brief pause between iterations to allow GC
		time.Sleep(100 * time.Millisecond)
		runtime.GC()
	}

	// Calculate statistical metrics
	ca.calculateStatistics(results)

	return results, nil
}

// setupLogger creates a configured logger for the specified library
func (ca *CompetitiveAnalyzer) setupLogger(library LibraryType) (interface{}, error) {
	// Reset buffer for consistent testing
	ca.buffer.Reset()

	switch library {
	case LibraryBolt:
		return bolt.New(bolt.NewJSONHandler(ca.buffer)), nil

	case LibraryZerolog:
		return zerolog.New(ca.buffer).Level(zerolog.InfoLevel), nil

	case LibraryZap:
		encoderCfg := zapcore.EncoderConfig{
			MessageKey:     "message",
			LevelKey:       "level",
			TimeKey:        "time",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
		}
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.AddSync(ca.buffer),
			zapcore.InfoLevel,
		)
		return zap.New(core), nil

	case LibraryLogrus:
		logger := logrus.New()
		logger.SetFormatter(&logrus.JSONFormatter{})
		logger.SetOutput(ca.buffer)
		logger.SetLevel(logrus.InfoLevel)
		return logger, nil

	case LibrarySlog:
		return slog.New(slog.NewJSONHandler(ca.buffer, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})), nil

	default:
		return nil, fmt.Errorf("unsupported library: %s", library)
	}
}

// createBenchmarkFunc creates a benchmark function for the given library and scenario
func (ca *CompetitiveAnalyzer) createBenchmarkFunc(library LibraryType, logger interface{}, scenario TestScenario) func(*testing.B) {
	return func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		// Warmup phase
		warmupEnd := time.Now().Add(ca.warmupDuration)
		for time.Now().Before(warmupEnd) {
			ca.logMessage(library, logger, scenario, 0)
		}

		b.ResetTimer()

		// Actual benchmark
		if scenario.Concurrency > 1 {
			// Concurrent benchmark
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					ca.logMessage(library, logger, scenario, b.N)
				}
			})
		} else {
			// Sequential benchmark
			for i := 0; i < b.N; i++ {
				ca.logMessage(library, logger, scenario, i)
			}
		}
	}
}

// logMessage writes a log message using the specified library and scenario parameters
func (ca *CompetitiveAnalyzer) logMessage(library LibraryType, logger interface{}, scenario TestScenario, iteration int) {
	message := ca.generateMessage(scenario.MessageSize)
	fields := ca.generateFields(scenario.Fields, iteration)

	switch library {
	case LibraryBolt:
		l := logger.(*bolt.Logger)
		event := l.Info()
		for k, v := range fields {
			switch val := v.(type) {
			case string:
				event = event.Str(k, val)
			case int:
				event = event.Int(k, val)
			case int64:
				event = event.Int64(k, val)
			case float64:
				event = event.Float64(k, val)
			case bool:
				event = event.Bool(k, val)
			case time.Time:
				event = event.Time(k, val)
			}
		}
		event.Msg(message)

	case LibraryZerolog:
		l := logger.(zerolog.Logger)
		event := l.Info()
		for k, v := range fields {
			switch val := v.(type) {
			case string:
				event = event.Str(k, val)
			case int:
				event = event.Int(k, val)
			case int64:
				event = event.Int64(k, val)
			case float64:
				event = event.Float64(k, val)
			case bool:
				event = event.Bool(k, val)
			case time.Time:
				event = event.Time(k, val)
			}
		}
		event.Msg(message)

	case LibraryZap:
		l := logger.(*zap.Logger)
		zapFields := make([]zap.Field, 0, len(fields))
		for k, v := range fields {
			switch val := v.(type) {
			case string:
				zapFields = append(zapFields, zap.String(k, val))
			case int:
				zapFields = append(zapFields, zap.Int(k, val))
			case int64:
				zapFields = append(zapFields, zap.Int64(k, val))
			case float64:
				zapFields = append(zapFields, zap.Float64(k, val))
			case bool:
				zapFields = append(zapFields, zap.Bool(k, val))
			case time.Time:
				zapFields = append(zapFields, zap.Time(k, val))
			}
		}
		l.Info(message, zapFields...)

	case LibraryLogrus:
		l := logger.(*logrus.Logger)
		l.WithFields(logrus.Fields(fields)).Info(message)

	case LibrarySlog:
		l := logger.(*slog.Logger)
		attrs := make([]slog.Attr, 0, len(fields))
		for k, v := range fields {
			attrs = append(attrs, slog.Any(k, v))
		}
		l.LogAttrs(context.Background(), slog.LevelInfo, message, attrs...)
	}
}

// generateMessage creates a message of the specified size
func (ca *CompetitiveAnalyzer) generateMessage(size string) string {
	switch size {
	case "small":
		return "Quick log message"
	case "medium":
		return "This is a medium-sized log message that contains more information and context"
	case "large":
		return "This is a large log message that contains extensive information, detailed context, error descriptions, stack traces, and other comprehensive data that would typically be found in enterprise logging scenarios where detailed information is crucial for debugging and monitoring purposes"
	default:
		return "Default log message"
	}
}

// generateFields creates a map of fields based on the specified count and iteration
func (ca *CompetitiveAnalyzer) generateFields(count int, iteration int) map[string]interface{} {
	fields := make(map[string]interface{})

	fieldTemplates := []struct {
		key   string
		value interface{}
	}{
		{"service", "user-service"},
		{"request_id", fmt.Sprintf("req_%d", iteration)},
		{"user_id", int64(12345 + iteration%1000)},
		{"action", "authenticate"},
		{"success", iteration%10 != 0},
		{"duration_ms", float64(100 + iteration%500)},
		{"ip_address", "192.168.1." + fmt.Sprintf("%d", 1+iteration%254)},
		{"timestamp", time.Now()},
		{"method", "POST"},
		{"status_code", 200 + iteration%400},
		{"user_agent", "Mozilla/5.0 (compatible; Benchmark)"},
		{"endpoint", "/api/v1/users/authenticate"},
		{"region", "us-west-2"},
		{"instance_id", "i-" + fmt.Sprintf("%08x", iteration)},
		{"correlation_id", fmt.Sprintf("corr_%d_%d", iteration, time.Now().UnixNano())},
	}

	for i := 0; i < count && i < len(fieldTemplates); i++ {
		fields[fieldTemplates[i].key] = fieldTemplates[i].value
	}

	return fields
}

// calculateStatistics computes statistical metrics for the benchmark results
func (ca *CompetitiveAnalyzer) calculateStatistics(results []BenchmarkResult) {
	if len(results) < 2 {
		return
	}

	// Extract ns/op values for statistical analysis
	values := make([]float64, len(results))
	for i, result := range results {
		values[i] = result.NsPerOp
	}

	sort.Float64s(values)

	// Calculate percentiles
	percentiles := map[string]float64{
		"p50": ca.percentile(values, 0.50),
		"p90": ca.percentile(values, 0.90),
		"p95": ca.percentile(values, 0.95),
		"p99": ca.percentile(values, 0.99),
	}

	// Calculate mean and standard deviation
	mean := ca.mean(values)
	stddev := ca.standardDeviation(values, mean)

	// Update last result with statistical data
	lastIdx := len(results) - 1
	results[lastIdx].NsPerOpStdDev = stddev
	results[lastIdx].NsPerOpMin = values[0]
	results[lastIdx].NsPerOpMax = values[len(values)-1]
	results[lastIdx].NsPerOpPercentiles = percentiles
}

// Helper statistical functions
func (ca *CompetitiveAnalyzer) mean(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (ca *CompetitiveAnalyzer) standardDeviation(values []float64, mean float64) float64 {
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	variance := sumSquares / float64(len(values)-1)
	return math.Sqrt(variance)
}

func (ca *CompetitiveAnalyzer) percentile(sortedValues []float64, p float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	if len(sortedValues) == 1 {
		return sortedValues[0]
	}

	index := p * float64(len(sortedValues)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sortedValues) {
		return sortedValues[len(sortedValues)-1]
	}

	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

// Helper functions for analysis output
func (ca *CompetitiveAnalyzer) libraryNames() []string {
	names := make([]string, len(ca.libraries))
	for i, lib := range ca.libraries {
		names[i] = lib.String()
	}
	return names
}

func (ca *CompetitiveAnalyzer) scenarioNames() []string {
	names := make([]string, len(ca.scenarios))
	for i, scenario := range ca.scenarios {
		names[i] = scenario.Name
	}
	return names
}

// generateAnalysisReports creates comprehensive analysis reports
func (ca *CompetitiveAnalyzer) generateAnalysisReports() error {
	// Implementation will be in the reporting module
	fmt.Printf("📈 Analysis reports will be generated in %s\n", ca.outputDir)
	return nil
}
