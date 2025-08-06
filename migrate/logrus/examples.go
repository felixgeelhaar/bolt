// Package logrus provides examples demonstrating migration from Logrus to Bolt.
package logrus

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/felixgeelhaar/bolt/v2"
)

// ExampleBasicLogging demonstrates basic logging migration from Logrus to Bolt.
func ExampleBasicLogging() {
	// BEFORE: Logrus
	// log := logrus.New()
	// log.Info("Application started")
	// log.WithField("user", "john").Info("User logged in")
	// log.WithFields(logrus.Fields{
	//     "service": "auth",
	//     "action":  "login",
	// }).Info("Authentication successful")

	// AFTER: Bolt
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().Msg("Application started")
	logger.Info().Str("user", "john").Msg("User logged in")
	logger.Info().
		Str("service", "auth").
		Str("action", "login").
		Msg("Authentication successful")
}

// ExampleLoggerConfiguration demonstrates logger configuration migration.
func ExampleLoggerConfiguration() {
	// BEFORE: Logrus
	// logger := logrus.New()
	// logger.SetLevel(logrus.DebugLevel)
	// logger.SetFormatter(&logrus.JSONFormatter{})
	// logger.SetOutput(os.Stdout)

	// AFTER: Bolt
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.DEBUG)
	
	// For console output:
	// logger := bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG)
	
	// Use the logger
	logger.Debug().Str("component", "database").Msg("Connecting to database")
}

// ExampleErrorHandling demonstrates error handling migration.
func ExampleErrorHandling() {
	err := errors.New("connection failed")

	// BEFORE: Logrus
	// logger := logrus.New()
	// logger.WithError(err).Error("Database connection failed")
	// logger.WithFields(logrus.Fields{
	//     "host": "localhost",
	//     "port": 5432,
	// }).WithError(err).Error("Failed to connect to database")

	// AFTER: Bolt
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Error().Err(err).Msg("Database connection failed")
	logger.Error().
		Str("host", "localhost").
		Int("port", 5432).
		Err(err).
		Msg("Failed to connect to database")
}

// ExampleContextLogging demonstrates context-based logging migration.
func ExampleContextLogging() {
	ctx := context.Background()

	// BEFORE: Logrus
	// logger := logrus.New()
	// logger.WithContext(ctx).Info("Processing request")

	// AFTER: Bolt (automatic OpenTelemetry integration)
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	contextLogger := logger.Ctx(ctx)
	contextLogger.Info().Msg("Processing request")
}

// ExampleStructuredLogging demonstrates structured logging patterns.
func ExampleStructuredLogging() {
	userID := 12345
	sessionID := "sess_abc123"
	requestDuration := 250 * time.Millisecond

	// BEFORE: Logrus
	// logger := logrus.New()
	// logger.WithFields(logrus.Fields{
	//     "user_id":    userID,
	//     "session_id": sessionID,
	//     "duration":   requestDuration,
	//     "success":    true,
	// }).Info("Request completed")

	// AFTER: Bolt
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Int("user_id", userID).
		Str("session_id", sessionID).
		Dur("duration", requestDuration).
		Bool("success", true).
		Msg("Request completed")
}

// ExampleLevelBasedLogging demonstrates different log levels.
func ExampleLevelBasedLogging() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// BEFORE: Logrus
	// logger.Trace("Entering function")
	// logger.Debug("Debug information")
	// logger.Info("General information")
	// logger.Warn("Warning message")
	// logger.Error("Error occurred")
	// logger.Fatal("Fatal error") // This would exit the program

	// AFTER: Bolt
	logger.Trace().Msg("Entering function")
	logger.Debug().Msg("Debug information")
	logger.Info().Msg("General information")
	logger.Warn().Msg("Warning message")
	logger.Error().Msg("Error occurred")
	logger.Fatal().Msg("Fatal error") // This would exit the program
}

// ExampleFieldTypes demonstrates various field types migration.
func ExampleFieldTypes() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	timestamp := time.Now()
	duration := 100 * time.Millisecond
	byteData := []byte("binary data")

	// BEFORE: Logrus
	// logger.WithFields(logrus.Fields{
	//     "string_field": "value",
	//     "int_field":    42,
	//     "bool_field":   true,
	//     "time_field":   timestamp,
	//     "float_field":  3.14,
	//     "bytes_field":  byteData,
	// }).Info("Various field types")

	// AFTER: Bolt
	logger.Info().
		Str("string_field", "value").
		Int("int_field", 42).
		Bool("bool_field", true).
		Time("time_field", timestamp).
		Float64("float_field", 3.14).
		Dur("duration_field", duration).
		Bytes("bytes_field", byteData).
		Msg("Various field types")
}

// ExampleFormatterMigration demonstrates formatter migration strategies.
func ExampleFormatterMigration() {
	// BEFORE: Logrus with JSONFormatter
	// logger := logrus.New()
	// logger.SetFormatter(&logrus.JSONFormatter{
	//     TimestampFormat: time.RFC3339,
	//     PrettyPrint:     false,
	// })

	// AFTER: Bolt with JSON handler (automatically uses RFC3339 timestamps)
	jsonLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	jsonLogger.Info().Str("format", "json").Msg("Using JSON formatter")

	// BEFORE: Logrus with TextFormatter
	// logger.SetFormatter(&logrus.TextFormatter{
	//     FullTimestamp: true,
	//     ForceColors:   true,
	// })

	// AFTER: Bolt with Console handler (automatically colorized and timestamped)
	consoleLogger := bolt.New(bolt.NewConsoleHandler(os.Stdout))
	consoleLogger.Info().Str("format", "console").Msg("Using console formatter")
}

// ExampleHookMigration demonstrates how to handle Logrus hooks migration.
func ExampleHookMigration() {
	// BEFORE: Logrus with hooks
	// logger := logrus.New()
	// logger.AddHook(&MyCustomHook{})

	// AFTER: Bolt with custom handler or middleware pattern
	logger := bolt.New(&CustomHandler{
		handler: bolt.NewJSONHandler(os.Stdout),
	})
	logger.Info().Msg("Using custom handler for hook-like functionality")
}

// CustomHandler demonstrates how to implement hook-like functionality in Bolt.
type CustomHandler struct {
	handler bolt.Handler
}

// Write implements bolt.Handler interface and adds custom logic.
func (h *CustomHandler) Write(e *bolt.Event) error {
	// Custom logic that would have been in a Logrus hook
	// For example, adding extra fields, filtering, sending to external services, etc.
	
	// Call the underlying handler
	return h.handler.Write(e)
}

// ExamplePerformanceComparison demonstrates the performance improvement when migrating.
func ExamplePerformanceComparison() {
	// This example shows the equivalent code patterns and their relative performance

	// BEFORE: Logrus (slower, more allocations)
	// func logWithLogrus() {
	//     logger := logrus.New()
	//     logger.SetFormatter(&logrus.JSONFormatter{})
	//     logger.WithFields(logrus.Fields{
	//         "service": "api",
	//         "method":  "POST",
	//         "path":    "/users",
	//         "status":  200,
	//         "latency": 45.2,
	//     }).Info("Request completed")
	// }

	// AFTER: Bolt (faster, zero allocations)
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Str("service", "api").
		Str("method", "POST").
		Str("path", "/users").
		Int("status", 200).
		Float64("latency", 45.2).
		Msg("Request completed")

	// Bolt is approximately 2603% faster than Logrus in benchmarks
}

// ExampleMigrationSteps demonstrates a step-by-step migration approach.
func ExampleMigrationSteps() {
	// Step 1: Replace imports
	// OLD: import "github.com/sirupsen/logrus"
	// NEW: import "github.com/felixgeelhaar/bolt/v2"

	// Step 2: Replace logger creation
	// OLD: logger := logrus.New()
	// NEW: logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// Step 3: Replace WithFields with chaining
	// OLD: logger.WithFields(logrus.Fields{"key": "value"}).Info("message")
	// NEW: logger.Info().Str("key", "value").Msg("message")

	// Step 4: Update level setting
	// OLD: logger.SetLevel(logrus.DebugLevel)
	// NEW: logger = logger.SetLevel(bolt.DEBUG)

	// Step 5: Replace formatters with handlers
	// OLD: logger.SetFormatter(&logrus.JSONFormatter{})
	// NEW: Already handled in logger creation with bolt.NewJSONHandler()

	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.DEBUG)
	logger.Info().Str("migration", "completed").Msg("Successfully migrated from Logrus to Bolt")
}

// ExampleCompatibilityLayer demonstrates using the compatibility layer for gradual migration.
func ExampleCompatibilityLayer() {
	// Using the Logrus compatibility layer allows gradual migration
	// Import: "github.com/felixgeelhaar/bolt/v2/migrate/logrus"

	logger := New() // This creates a Logrus-compatible logger backed by Bolt
	
	// Use existing Logrus patterns
	logger.WithField("service", "auth").Info("Service started")
	logger.WithFields(Fields{
		"user_id": 123,
		"action":  "login",
	}).Info("User action")

	// Set levels and formatters as before
	logger.SetLevel(DebugLevel)
	logger.SetFormatter(&JSONFormatter{})

	// Benefits: Get Bolt performance with existing Logrus code
}

// ExampleFullMigrationWorkflow demonstrates a complete migration workflow.
func ExampleFullMigrationWorkflow() {
	// 1. Install Bolt
	// go get github.com/felixgeelhaar/bolt/v2

	// 2. Use compatibility layer first (gradual migration)
	// Replace: import "github.com/sirupsen/logrus"
	// With:    import logrus "github.com/felixgeelhaar/bolt/v2/migrate/logrus"

	// 3. Test compatibility - your existing code should work

	// 4. Gradually migrate to native Bolt API for better performance
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// 5. Use migration tools to automatically transform code
	// Run: go run github.com/felixgeelhaar/bolt/v2/migrate/cmd migrate --from=logrus --input=./src --output=./migrated

	// 6. Validate migration and run tests
	// 7. Deploy with significant performance improvements

	logger.Info().
		Str("migration_type", "logrus_to_bolt").
		Str("status", "completed").
		Float64("performance_gain", 26.03). // 2603% faster
		Int("allocations_reduced", 100).     // 100% reduction in allocations
		Msg("Migration completed successfully")
}