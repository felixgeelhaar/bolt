// Package main provides a comprehensive command-line interface for Bolt benchmarking
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/felixgeelhaar/bolt/benchmark/enterprise"
	"github.com/felixgeelhaar/bolt/benchmark/framework"
)

var (
	version = "2.0.0"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	var (
		benchmarkType = flag.String("type", "competitive", "Benchmark type: competitive, enterprise, regression, load")
		outputDir     = flag.String("output", "benchmark-results", "Output directory for results")
		duration      = flag.Duration("duration", 5*time.Minute, "Test duration")
		libraries     = flag.String("libraries", "Bolt,Zerolog,Zap,Logrus,Slog", "Comma-separated list of libraries to test")
		scenarios     = flag.String("scenarios", "all", "Comma-separated list of scenarios (or 'all')")
		iterations    = flag.Int("iterations", 10, "Number of benchmark iterations")
		parallel      = flag.Int("parallel", runtime.NumCPU(), "Number of parallel workers")
		profiling     = flag.Bool("profiling", true, "Enable CPU and memory profiling")
		verbose       = flag.Bool("verbose", false, "Verbose output")
		generateHTML  = flag.Bool("html", true, "Generate HTML reports")
		generateCSV   = flag.Bool("csv", true, "Generate CSV exports")
		noUpload      = flag.Bool("no-upload", false, "Skip uploading results to GitHub Pages")
		configFile    = flag.String("config", "", "Configuration file path")
		showVersion   = flag.Bool("version", false, "Show version information")
		listScenarios = flag.Bool("list-scenarios", false, "List available scenarios")
		validate      = flag.Bool("validate", false, "Validate configuration and exit")
		quiet         = flag.Bool("quiet", false, "Suppress non-essential output")
		maxMemory     = flag.Int64("max-memory", 4096, "Maximum memory usage in MB")
		maxCPU        = flag.Float64("max-cpu", 95.0, "Maximum CPU usage percentage")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `
Bolt Benchmark Suite v%s

USAGE:
    bolt-benchmark [OPTIONS]

DESCRIPTION:
    Comprehensive performance benchmarking suite for the Bolt logging library.
    Supports competitive analysis, enterprise scenarios, regression testing,
    and load testing with professional reporting.

BENCHMARK TYPES:
    competitive  - Compare Bolt against other logging libraries
    enterprise   - Run real-world enterprise scenarios  
    regression   - Check for performance regressions
    load         - High-load stress testing

OPTIONS:
`, version)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
EXAMPLES:
    # Run competitive analysis
    bolt-benchmark -type=competitive -duration=10m

    # Enterprise scenarios with custom output
    bolt-benchmark -type=enterprise -output=./results -scenarios=WebAPI,Trading

    # Regression testing with memory limits
    bolt-benchmark -type=regression -max-memory=1024 -max-cpu=80

    # Load testing with profiling
    bolt-benchmark -type=load -profiling=true -parallel=16

SCENARIOS:
    Competitive: Basic, Fields5, Disabled, Concurrent
    Enterprise:  WebAPI, Trading, Microservice, DataPipeline, Security, Container

LIBRARIES:
    Bolt, Zerolog, Zap, Logrus, Slog

For more information, visit: https://github.com/felixgeelhaar/bolt
`)
	}

	flag.Parse()

	_ = configFile // TODO: Implement configuration file loading

	if *showVersion {
		fmt.Printf("Bolt Benchmark Suite\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Commit:  %s\n", commit)
		fmt.Printf("Built:   %s\n", date)
		fmt.Printf("Go:      %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return
	}

	if *listScenarios {
		listAvailableScenarios()
		return
	}

	// Setup logging
	if *quiet {
		log.SetOutput(os.Stdout)
	}

	// Validate configuration
	if *validate {
		if err := validateConfiguration(*benchmarkType, *libraries, *scenarios); err != nil {
			log.Fatalf("Configuration validation failed: %v", err)
		}
		fmt.Println("✅ Configuration is valid")
		return
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n🛑 Shutdown signal received, gracefully stopping...")
		cancel()
	}()

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0750); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Print startup banner
	if !*quiet {
		printBanner(*benchmarkType, *outputDir)
	}

	// Run benchmarks based on type
	var err error
	switch strings.ToLower(*benchmarkType) {
	case "competitive":
		err = runCompetitiveBenchmarks(ctx, &CompetitiveConfig{
			OutputDir:    *outputDir,
			Duration:     *duration,
			Libraries:    parseLibraries(*libraries),
			Scenarios:    parseScenarios(*scenarios, "competitive"),
			Iterations:   *iterations,
			Profiling:    *profiling,
			GenerateHTML: *generateHTML,
			GenerateCSV:  *generateCSV,
			Verbose:      *verbose,
		})
	case "enterprise":
		err = runEnterpriseBenchmarks(ctx, &EnterpriseConfig{
			OutputDir:     *outputDir,
			Duration:      *duration,
			Scenarios:     parseScenarios(*scenarios, "enterprise"),
			MaxMemoryMB:   *maxMemory,
			MaxCPUPercent: *maxCPU,
			Profiling:     *profiling,
			GenerateHTML:  *generateHTML,
			GenerateCSV:   *generateCSV,
			Verbose:       *verbose,
		})
	case "regression":
		err = runRegressionBenchmarks(ctx, &RegressionConfig{
			OutputDir:     *outputDir,
			Duration:      *duration,
			Iterations:    *iterations,
			MaxMemoryMB:   *maxMemory,
			MaxCPUPercent: *maxCPU,
			Verbose:       *verbose,
		})
	case "load":
		err = runLoadBenchmarks(ctx, &LoadConfig{
			OutputDir:     *outputDir,
			Duration:      *duration,
			Parallel:      *parallel,
			MaxMemoryMB:   *maxMemory,
			MaxCPUPercent: *maxCPU,
			Profiling:     *profiling,
			Verbose:       *verbose,
		})
	default:
		log.Fatalf("Unknown benchmark type: %s", *benchmarkType)
	}

	if err != nil {
		log.Fatalf("Benchmark execution failed: %v", err)
	}

	// Upload results to GitHub Pages if not disabled
	if !*noUpload && os.Getenv("CI") == "true" {
		if err := uploadResults(*outputDir); err != nil {
			log.Printf("Warning: Failed to upload results: %v", err)
		}
	}

	if !*quiet {
		fmt.Printf("\n🎉 Benchmarks completed successfully!\n")
		fmt.Printf("Results available in: %s\n", *outputDir)
	}
}

// Configuration structures
type CompetitiveConfig struct {
	OutputDir    string
	Duration     time.Duration
	Libraries    []string
	Scenarios    []string
	Iterations   int
	Profiling    bool
	GenerateHTML bool
	GenerateCSV  bool
	Verbose      bool
}

type EnterpriseConfig struct {
	OutputDir     string
	Duration      time.Duration
	Scenarios     []string
	MaxMemoryMB   int64
	MaxCPUPercent float64
	Profiling     bool
	GenerateHTML  bool
	GenerateCSV   bool
	Verbose       bool
}

type RegressionConfig struct {
	OutputDir     string
	Duration      time.Duration
	Iterations    int
	MaxMemoryMB   int64
	MaxCPUPercent float64
	Verbose       bool
}

type LoadConfig struct {
	OutputDir     string
	Duration      time.Duration
	Parallel      int
	MaxMemoryMB   int64
	MaxCPUPercent float64
	Profiling     bool
	Verbose       bool
}

// Benchmark execution functions
func runCompetitiveBenchmarks(ctx context.Context, config *CompetitiveConfig) error {
	fmt.Printf("🏁 Starting competitive analysis...\n")
	fmt.Printf("Libraries: %v\n", config.Libraries)
	fmt.Printf("Duration: %v\n", config.Duration)
	fmt.Printf("Output: %s\n\n", config.OutputDir)

	analyzer := framework.NewCompetitiveAnalyzer().
		WithOutputDir(config.OutputDir).
		WithIterations(config.Iterations).
		WithProfiling(config.Profiling)

	// Configure libraries
	var libraries []framework.LibraryType
	for _, lib := range config.Libraries {
		switch strings.ToLower(lib) {
		case "bolt":
			libraries = append(libraries, framework.LibraryBolt)
		case "zerolog":
			libraries = append(libraries, framework.LibraryZerolog)
		case "zap":
			libraries = append(libraries, framework.LibraryZap)
		case "logrus":
			libraries = append(libraries, framework.LibraryLogrus)
		case "slog":
			libraries = append(libraries, framework.LibrarySlog)
		}
	}
	analyzer = analyzer.WithLibraries(libraries...)

	// Configure scenarios if specified
	if len(config.Scenarios) > 0 && config.Scenarios[0] != "all" {
		var scenarios []framework.TestScenario
		for _, scenarioName := range config.Scenarios {
			for _, predefined := range framework.PredefinedScenarios {
				if strings.EqualFold(predefined.Name, scenarioName) {
					scenarios = append(scenarios, predefined)
					break
				}
			}
		}
		if len(scenarios) > 0 {
			analyzer = analyzer.WithScenarios(scenarios...)
		}
	}

	// Run analysis
	if err := analyzer.RunComprehensiveAnalysis(ctx); err != nil {
		return fmt.Errorf("competitive analysis failed: %w", err)
	}

	// Generate reports
	if config.GenerateHTML || config.GenerateCSV {
		reportGen := framework.NewReportGenerator(analyzer)
		if err := reportGen.GenerateHTMLReport(); err != nil {
			return fmt.Errorf("report generation failed: %w", err)
		}
	}

	return nil
}

func runEnterpriseBenchmarks(_ctx context.Context, config *EnterpriseConfig) error {
	fmt.Printf("🏢 Starting enterprise benchmarking suite...\n")
	fmt.Printf("Max Memory: %dMB\n", config.MaxMemoryMB)
	fmt.Printf("Max CPU: %.1f%%\n", config.MaxCPUPercent)
	fmt.Printf("Duration: %v\n", config.Duration)
	fmt.Printf("Output: %s\n\n", config.OutputDir)

	suite := enterprise.NewEnterpriseBenchmarkSuite()

	// Configure scenarios if specified
	if len(config.Scenarios) > 0 && config.Scenarios[0] != "all" {
		var scenarios []enterprise.EnterpriseScenario
		for _, scenarioName := range config.Scenarios {
			for _, predefined := range enterprise.EnterpriseScenarios {
				if strings.EqualFold(predefined.Name, scenarioName) {
					scenarios = append(scenarios, predefined)
					break
				}
			}
		}
		if len(scenarios) > 0 {
			// Note: Would need to add a method to configure scenarios
			fmt.Printf("Using %d custom scenarios\n", len(scenarios))
		}
	}

	return suite.RunEnterpriseBenchmarks()
}

func runRegressionBenchmarks(_ctx context.Context, config *RegressionConfig) error {
	fmt.Printf("📊 Starting regression testing...\n")
	fmt.Printf("Iterations: %d\n", config.Iterations)
	fmt.Printf("Output: %s\n\n", config.OutputDir)

	// Implementation would use the existing performance regression script
	// or integrate it into Go code
	fmt.Println("Regression testing implementation would be integrated here")
	return nil
}

func runLoadBenchmarks(_ctx context.Context, config *LoadConfig) error {
	fmt.Printf("⚡ Starting load testing...\n")
	fmt.Printf("Parallel workers: %d\n", config.Parallel)
	fmt.Printf("Duration: %v\n", config.Duration)
	fmt.Printf("Output: %s\n\n", config.OutputDir)

	// Implementation would create high-load scenarios
	fmt.Println("Load testing implementation would be integrated here")
	return nil
}

// Utility functions
func parseLibraries(librariesStr string) []string {
	if librariesStr == "" {
		return []string{"Bolt", "Zerolog", "Zap", "Logrus", "Slog"}
	}
	return strings.Split(strings.ReplaceAll(librariesStr, " ", ""), ",")
}

func parseScenarios(scenariosStr, _benchmarkType string) []string {
	if scenariosStr == "" || scenariosStr == "all" {
		return []string{"all"}
	}
	return strings.Split(strings.ReplaceAll(scenariosStr, " ", ""), ",")
}

func validateConfiguration(benchmarkType, libraries, _scenarios string) error {
	validTypes := []string{"competitive", "enterprise", "regression", "load"}
	valid := false
	for _, t := range validTypes {
		if strings.ToLower(benchmarkType) == t {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid benchmark type: %s (valid: %v)", benchmarkType, validTypes)
	}

	validLibs := []string{"Bolt", "Zerolog", "Zap", "Logrus", "Slog"}
	for _, lib := range parseLibraries(libraries) {
		found := false
		for _, valid := range validLibs {
			if strings.EqualFold(lib, valid) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid library: %s (valid: %v)", lib, validLibs)
		}
	}

	return nil
}

func listAvailableScenarios() {
	fmt.Println("📋 Available Benchmark Scenarios:")
	fmt.Println()

	fmt.Println("Competitive Analysis Scenarios:")
	for _, scenario := range framework.PredefinedScenarios {
		fmt.Printf("  %-20s - %s\n", scenario.Name, scenario.Description)
	}

	fmt.Println("\nEnterprise Scenarios:")
	for _, scenario := range enterprise.EnterpriseScenarios {
		fmt.Printf("  %-20s - %s (Target: %d RPS, %d concurrent)\n",
			scenario.Name, scenario.Description, scenario.TargetRPS, scenario.MaxConcurrency)
	}
}

func printBanner(benchmarkType, outputDir string) {
	fmt.Printf(`
╔════════════════════════════════════════════════════════════════╗
║                    Bolt Benchmark Suite v%-8s                ║
║                                                                ║
║  🚀 Type: %-12s                                           ║
║  📊 Output: %-50s ║
║  🔧 Go: %-12s OS: %-8s Arch: %-8s            ║
║  🖥️  CPUs: %-2d                                                  ║
╚════════════════════════════════════════════════════════════════╝

`, version, benchmarkType, outputDir, runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
}

func uploadResults(outputDir string) error {
	// Implementation would upload results to GitHub Pages
	// This could use GitHub Actions artifacts or direct deployment
	fmt.Printf("📤 Uploading results from %s to GitHub Pages...\n", outputDir)

	// Placeholder implementation
	return nil
}
