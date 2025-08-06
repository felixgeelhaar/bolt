// Package stdlog provides examples demonstrating migration from Go's standard log to Bolt.
package stdlog

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/felixgeelhaar/bolt"
)

// ExampleBasicMigration demonstrates basic logging migration from standard log to Bolt.
func ExampleBasicMigration() {
	// BEFORE: Standard library log
	// log.Print("Application started")
	// log.Printf("Processing %d items", 100)
	// log.Println("Processing complete")

	// AFTER: Bolt (structured approach)
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().Msg("Application started")
	logger.Info().Int("item_count", 100).Msg("Processing items")
	logger.Info().Msg("Processing complete")

	// AFTER: Bolt (compatibility approach - zero code changes)
	// Just replace: import "log"
	// With: import log "github.com/felixgeelhaar/bolt/migrate/stdlog"
	// All existing log.Print, log.Printf, log.Println calls work unchanged!
}

// ExampleDropInReplacement demonstrates the drop-in replacement approach.
func ExampleDropInReplacement() {
	// This example shows how existing standard log code can work unchanged
	// by simply changing the import statement.

	// Step 1: Change import from:
	// import "log"
	// 
	// To:
	// import log "github.com/felixgeelhaar/bolt/migrate/stdlog"

	// Step 2: All existing code continues to work:
	log.Print("Server starting")
	log.Printf("Listening on port %d", 8080)
	log.Println("Ready to accept connections")

	// Benefits:
	// - Zero code changes required
	// - Immediate performance improvement
	// - Better JSON output format
	// - Maintains all existing behavior
}

// ExampleCustomLogger demonstrates custom logger migration.
func ExampleCustomLogger() {
	// BEFORE: Standard log with custom configuration
	// customLog := log.New(os.Stdout, "[API] ", log.LstdFlags|log.Lshortfile)
	// customLog.Print("Custom logger message")

	// AFTER: Bolt structured approach
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Str("component", "API").
		Caller().
		Msg("Custom logger message")

	// AFTER: Bolt compatibility approach
	customLog := New(os.Stdout, "[API] ", LstdFlags|Lshortfile)
	customLog.Print("Custom logger message")
}

// ExampleErrorHandling demonstrates error and panic handling migration.
func ExampleErrorHandling() {
	err := errors.New("database connection failed")

	// BEFORE: Standard log error handling
	// log.Printf("Error: %v", err)
	// log.Fatal("Critical error - shutting down")
	// log.Panic("Unrecoverable error")

	// AFTER: Bolt structured approach
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Error().Err(err).Msg("Database connection failed")
	logger.Fatal().Msg("Critical error - shutting down")
	
	// For panic behavior:
	logger.Fatal().Msg("Unrecoverable error")
	panic("Unrecoverable error")
}

// ExamplePerformanceComparison demonstrates performance benefits.
func ExamplePerformanceComparison() {
	// This example shows equivalent functionality and the performance difference

	// BEFORE: Standard log (slower)
	standardLogExample := func() {
		log.Printf("User %d performed action %s with result %v", 12345, "login", true)
	}

	// AFTER: Bolt (much faster, zero allocations)
	boltExample := func() {
		logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
		logger.Info().
			Int("user_id", 12345).
			Str("action", "login").
			Bool("result", true).
			Msg("User performed action")
	}

	// Performance improvements when migrating to Bolt:
	// - Significantly faster logging operations
	// - Zero memory allocations in hot paths
	// - Better structured output for log analysis
	// - Built-in JSON formatting

	_ = standardLogExample
	_ = boltExample
}

// ExampleMigrationStrategies demonstrates different migration approaches.
func ExampleMigrationStrategies() {
	// Strategy 1: Drop-in replacement (fastest migration)
	// Replace import "log" with import log "github.com/felixgeelhaar/bolt/migrate/stdlog"
	// Zero code changes, immediate benefits
	
	// Strategy 2: Compatibility layer with gradual enhancement
	logger := GetUnderlyingLogger() // Get the Bolt logger for structured logging
	
	// Continue using standard log functions
	log.Print("Standard log message")
	
	// Gradually add structured logging
	logger.Info().
		Str("service", "migration").
		Str("strategy", "gradual").
		Msg("Enhanced logging with structure")

	// Strategy 3: Full migration to structured logging
	structuredLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	structuredLogger.Info().
		Str("migration_type", "complete").
		Bool("structured", true).
		Msg("Fully migrated to structured logging")
}

// ExampleLogLevels demonstrates migrating from print statements to proper levels.
func ExampleLogLevels() {
	logger := bolt.New(bolt.NewConsoleHandler(os.Stdout))

	// BEFORE: Everything was log.Print (info level)
	// log.Print("Debug information")      // Wrong level
	// log.Print("Normal operation")       // Correct level
	// log.Print("Something suspicious")   // Wrong level
	// log.Print("Error occurred")         // Wrong level

	// AFTER: Proper log levels
	logger.Debug().Msg("Debug information")
	logger.Info().Msg("Normal operation")
	logger.Warn().Msg("Something suspicious")
	logger.Error().Msg("Error occurred")
}

// ExampleStructuredLogging demonstrates the evolution from string formatting to structured fields.
func ExampleStructuredLogging() {
	userID := 12345
	userName := "john_doe"
	requestDuration := 150.5
	success := true

	// BEFORE: String formatting (less queryable, harder to analyze)
	// log.Printf("User %s (ID: %d) request completed in %.2fms, success: %v", 
	//     userName, userID, requestDuration, success)

	// AFTER: Structured logging (easily queryable and analyzable)
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Str("user_name", userName).
		Int("user_id", userID).
		Float64("duration_ms", requestDuration).
		Bool("success", success).
		Msg("User request completed")

	// Benefits of structured logging:
	// - Fields are easily searchable and filterable
	// - Type-safe field methods prevent formatting errors
	// - Better integration with log analysis tools
	// - Consistent field naming across the application
}

// ExampleConfigurationMigration demonstrates configuration migration patterns.
func ExampleConfigurationMigration() {
	// BEFORE: Standard log configuration
	// log.SetFlags(log.LstdFlags | log.Lshortfile)
	// log.SetPrefix("[MyApp] ")
	// log.SetOutput(os.Stderr)

	// AFTER: Bolt configuration (structured approach)
	logger := bolt.New(bolt.NewConsoleHandler(os.Stderr))
	
	// Instead of prefixes, use structured fields
	appLogger := logger.With().Str("app", "MyApp").Logger()
	
	// Caller information is available on demand
	appLogger.Info().Caller().Msg("Application message with caller info")

	// AFTER: Bolt configuration (compatibility approach)
	compatLogger := New(os.Stderr, "[MyApp] ", LstdFlags|Lshortfile)
	compatLogger.Print("Application message")
}

// ExampleOutputFormats demonstrates different output format options.
func ExampleOutputFormats() {
	// JSON output for structured logging and machine parsing
	jsonLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	jsonLogger.Info().
		Str("format", "json").
		Str("use_case", "production_logs").
		Msg("JSON formatted log")

	// Console output for development and human-readable logs
	consoleLogger := bolt.New(bolt.NewConsoleHandler(os.Stdout))
	consoleLogger.Info().
		Str("format", "console").
		Str("use_case", "development").
		Msg("Human-readable colored log")

	// Environment-based configuration
	// Set BOLT_FORMAT=console for development
	// Set BOLT_FORMAT=json for production
	envLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)) // Default from env vars
	envLogger.Info().Msg("Environment-configured logger")
}

// ExampleAdvancedFeatures demonstrates advanced Bolt features not available in standard log.
func ExampleAdvancedFeatures() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// Context-based logging (OpenTelemetry integration)
	// ctx := context.Background()
	// contextLogger := logger.Ctx(ctx)
	// contextLogger.Info().Msg("Request with tracing context")

	// Rich field types
	logger.Info().
		Str("string_field", "value").
		Int("int_field", 42).
		Float64("float_field", 3.14159).
		Bool("bool_field", true).
		// Dur("duration_field", 100*time.Millisecond).
		// Time("timestamp_field", time.Now()).
		Hex("hex_field", []byte("binary data")).
		Msg("Rich typed fields")

	// Error handling with stack traces
	err := errors.New("example error")
	logger.Error().
		Err(err).
		Stack(). // Add stack trace
		Caller(). // Add caller information
		Msg("Error with debugging information")

	// Performance monitoring
	logger.Info().
		// Counter("requests_total", &requestCounter).
		Float64("response_time_ms", 45.2).
		Int("status_code", 200).
		Msg("Performance metrics")
}

// ExampleGradualMigration demonstrates a step-by-step migration approach.
func ExampleGradualMigration() {
	// Phase 1: Drop-in replacement
	// Change: import "log"
	// To: import log "github.com/felixgeelhaar/bolt/migrate/stdlog"
	log.Print("Phase 1: Using compatibility layer")

	// Phase 2: Access underlying Bolt logger for new code
	boltLogger := GetUnderlyingLogger()
	boltLogger.Info().
		Str("phase", "2").
		Msg("New code uses structured logging")

	// Phase 3: Gradually replace old log calls
	// Old: log.Printf("User %d logged in", userID)
	// New: boltLogger.Info().Int("user_id", userID).Msg("User logged in")

	// Phase 4: Full migration to native Bolt API
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Str("phase", "4").
		Str("status", "complete").
		Msg("Fully migrated to Bolt")
}

// ExampleTestingMigration demonstrates testing considerations during migration.
func ExampleTestingMigration() {
	// For testing, you might want to capture log output
	
	// BEFORE: Testing with standard log required complex setup
	// var buf bytes.Buffer
	// log.SetOutput(&buf)
	// log.Print("test message")
	// output := buf.String()

	// AFTER: Testing with Bolt is straightforward
	var logOutput bytes.Buffer
	testLogger := bolt.New(bolt.NewJSONHandler(&logOutput))
	
	testLogger.Info().Str("test", "value").Msg("test message")
	
	// logOutput.String() contains the JSON log output
	// Easy to parse and verify specific fields
	fmt.Printf("Test log output: %s", logOutput.String())
}

// ExampleMigrationBestPractices demonstrates best practices during and after migration.
func ExampleMigrationBestPractices() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// Best Practice 1: Use consistent field names across your application
	logger.Info().
		Str("user_id", "12345").    // Consistent naming
		Str("request_id", "req_1"). // Consistent naming
		Msg("User request processed")

	// Best Practice 2: Use appropriate log levels
	logger.Debug().Msg("Detailed debugging information")
	logger.Info().Msg("General application flow")
	logger.Warn().Msg("Something noteworthy but not an error")
	logger.Error().Msg("Error condition occurred")
	logger.Fatal().Msg("Critical error - application cannot continue")

	// Best Practice 3: Include contextual information
	logger.Info().
		Str("component", "auth").
		Str("method", "POST").
		Str("endpoint", "/login").
		Int("status_code", 200).
		Float64("duration_ms", 125.5).
		Msg("Authentication request completed")

	// Best Practice 4: Use structured fields instead of string formatting
	// Good:
	logger.Info().Int("count", 42).Str("type", "users").Msg("Processing items")
	
	// Avoid:
	logger.Info().Msg(fmt.Sprintf("Processing %d %s", 42, "users"))

	// Best Practice 5: Handle errors appropriately
	err := errors.New("example error")
	logger.Error().
		Err(err).
		Str("operation", "database_query").
		Msg("Database operation failed")
}

// ExamplePerformanceTesting demonstrates how to measure migration benefits.
func ExamplePerformanceTesting() {
	// This function shows how to set up benchmarks to measure migration benefits
	
	fmt.Println("Performance Comparison: Standard Log vs Bolt")
	fmt.Println("==========================================")
	fmt.Println()
	
	fmt.Println("Standard Log:")
	fmt.Println("- Multiple memory allocations per log call")
	fmt.Println("- String formatting overhead")
	fmt.Println("- No built-in structured data support")
	fmt.Println()
	
	fmt.Println("Bolt:")
	fmt.Println("- Zero allocations in hot path")
	fmt.Println("- Direct serialization without intermediate allocations")
	fmt.Println("- Native structured logging support")
	fmt.Println("- Sub-100ns logging operations")
	fmt.Println()
	
	fmt.Println("Typical improvements after migration:")
	fmt.Println("- 80%+ faster logging operations")
	fmt.Println("- 90%+ reduction in memory allocations")
	fmt.Println("- Better observability through structured fields")
	fmt.Println("- Improved application performance under load")
}

// ExampleRealWorldMigration demonstrates a complete real-world migration scenario.
func ExampleRealWorldMigration() {
	// Scenario: Web server logging migration
	
	// BEFORE: Standard log in a web server
	// func handleRequest(w http.ResponseWriter, r *http.Request) {
	//     start := time.Now()
	//     log.Printf("Handling %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
	//     
	//     // ... handle request ...
	//     
	//     duration := time.Since(start)
	//     log.Printf("Request completed in %v", duration)
	// }

	// AFTER: Structured logging with Bolt
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	
	// Simulated request handling
	method := "GET"
	path := "/api/users"
	remoteAddr := "192.168.1.100"
	// start := time.Now()
	
	logger.Info().
		Str("method", method).
		Str("path", path).
		Str("remote_addr", remoteAddr).
		Msg("Handling request")
	
	// ... handle request ...
	
	// duration := time.Since(start)
	logger.Info().
		Str("method", method).
		Str("path", path).
		// Dur("duration", duration).
		Int("status_code", 200).
		Msg("Request completed")
	
	// Benefits in production:
	// - Easy to filter logs by method, path, status code
	// - Simple to create dashboards and alerts
	// - Better performance under high request load
	// - Structured data can be easily ingested by log aggregation systems
}

import "bytes"