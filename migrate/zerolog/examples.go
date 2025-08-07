package zerolog

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/felixgeelhaar/bolt/v2"
	"github.com/rs/zerolog"
)

// Examples demonstrates side-by-side comparisons of Zerolog and Bolt usage patterns.

// Example 1: Basic Logger Setup
func ExampleBasicSetup() {
	// === BEFORE (Zerolog) ===
	zerologLogger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)

	zerologLogger.Info().
		Str("service", "auth").
		Int("port", 8080).
		Msg("Server starting")

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)

	boltLogger.Info().
		Str("service", "auth").
		Int("port", 8080).
		Msg("Server starting")
}

// Example 2: Console Output
func ExampleConsoleOutput() {
	// === BEFORE (Zerolog) ===
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
	zerologLogger := zerolog.New(consoleWriter).Level(zerolog.InfoLevel)

	zerologLogger.Info().
		Str("user", "john_doe").
		Int("attempts", 3).
		Msg("Login successful")

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.INFO)

	boltLogger.Info().
		Str("user", "john_doe").
		Int("attempts", 3).
		Msg("Login successful")
}

// Example 3: Structured Logging with Multiple Fields
func ExampleStructuredLogging() {
	// === BEFORE (Zerolog) ===
	zerologLogger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)

	zerologLogger.Info().
		Str("trace_id", "abc123").
		Str("span_id", "def456").
		Str("method", "POST").
		Str("endpoint", "/api/users").
		Int("status_code", 201).
		Float64("response_time", 45.67).
		Bool("cached", false).
		Msg("Request processed")

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)

	boltLogger.Info().
		Str("trace_id", "abc123").
		Str("span_id", "def456").
		Str("method", "POST").
		Str("endpoint", "/api/users").
		Int("status_code", 201).
		Float64("response_time", 45.67).
		Bool("cached", false).
		Msg("Request processed")
}

// Example 4: Error Handling
func ExampleErrorHandling() {
	err := errors.New("database connection failed")

	// === BEFORE (Zerolog) ===
	zerologLogger := zerolog.New(os.Stdout).Level(zerolog.ErrorLevel)

	zerologLogger.Error().
		Err(err).
		Str("database", "users_db").
		Int("retry_count", 3).
		Msg("Failed to connect to database")

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.ERROR)

	boltLogger.Error().
		Err(err).
		Str("database", "users_db").
		Int("retry_count", 3).
		Msg("Failed to connect to database")
}

// Example 5: Contextual Logging
func ExampleContextualLogging() {
	ctx := context.Background()

	// === BEFORE (Zerolog) ===
	zerologLogger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)

	zerologLogger.Info().
		Ctx(ctx).
		Str("operation", "user_creation").
		Int("user_id", 12345).
		Msg("User created successfully")

	// === AFTER (Bolt) ===
	// Bolt automatically handles OpenTelemetry context extraction
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)

	boltLogger.Ctx(ctx).Info().
		Str("operation", "user_creation").
		Int("user_id", 12345).
		Msg("User created successfully")
}

// Example 6: Logger with Context Fields
func ExampleLoggerWithContext() {
	// === BEFORE (Zerolog) ===
	baseZerologLogger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)
	serviceLogger := baseZerologLogger.With().
		Str("service", "payment-processor").
		Str("version", "v2.1.0").
		Logger()

	serviceLogger.Info().
		Str("transaction_id", "tx_123").
		Float64("amount", 99.99).
		Msg("Payment processed")

	// === AFTER (Bolt) ===
	baseBoltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)
	serviceBoltLogger := baseBoltLogger.With().
		Str("service", "payment-processor").
		Str("version", "v2.1.0").
		Logger()

	serviceBoltLogger.Info().
		Str("transaction_id", "tx_123").
		Float64("amount", 99.99).
		Msg("Payment processed")
}

// Example 7: Different Log Levels
func ExampleLogLevels() {
	// === BEFORE (Zerolog) ===
	zerologLogger := zerolog.New(os.Stdout).Level(zerolog.TraceLevel)

	zerologLogger.Trace().Msg("Entering function")
	zerologLogger.Debug().Str("variable", "value").Msg("Debug information")
	zerologLogger.Info().Msg("Information message")
	zerologLogger.Warn().Msg("Warning message")
	zerologLogger.Error().Msg("Error occurred")
	zerologLogger.Fatal().Msg("Fatal error") // Note: This would exit the program

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.TRACE)

	boltLogger.Trace().Msg("Entering function")
	boltLogger.Debug().Str("variable", "value").Msg("Debug information")
	boltLogger.Info().Msg("Information message")
	boltLogger.Warn().Msg("Warning message")
	boltLogger.Error().Msg("Error occurred")
	boltLogger.Fatal().Msg("Fatal error") // Note: This would exit the program
}

// Example 8: Time and Duration Fields
func ExampleTimeAndDuration() {
	startTime := time.Now()
	duration := time.Since(startTime)

	// === BEFORE (Zerolog) ===
	zerologLogger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)

	zerologLogger.Info().
		Time("start_time", startTime).
		Dur("duration", duration).
		Msg("Operation completed")

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)

	boltLogger.Info().
		Time("start_time", startTime).
		Dur("duration", duration).
		Msg("Operation completed")
}

// Example 9: Binary Data Logging
func ExampleBinaryData() {
	data := []byte("binary data here")

	// === BEFORE (Zerolog) ===
	zerologLogger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)

	zerologLogger.Info().
		Bytes("data", data).
		Hex("data_hex", data).
		Msg("Processing binary data")

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)

	boltLogger.Info().
		Bytes("data", data).
		Hex("data_hex", data).
		Msg("Processing binary data")
}

// Example 10: Complex Data Types
func ExampleComplexData() {
	userData := map[string]interface{}{
		"id":    12345,
		"name":  "John Doe",
		"email": "john@example.com",
		"roles": []string{"user", "admin"},
	}

	// === BEFORE (Zerolog) ===
	zerologLogger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)

	zerologLogger.Info().
		Interface("user", userData).
		Msg("User data received")

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)

	boltLogger.Info().
		Any("user", userData). // Bolt uses Any() instead of Interface()
		Msg("User data received")
}

// Migration Helper Functions

// MigrateZerologToBolt demonstrates how to wrap existing Zerolog code for gradual migration.
func MigrateZerologToBolt() {
	// Step 1: Replace the import
	// OLD: import "github.com/rs/zerolog"
	// NEW: import "github.com/felixgeelhaar/bolt/v2"

	// Step 2: Update logger creation
	// OLD: logger := zerolog.New(os.Stdout)
	// NEW: logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// Step 3: Update level setting
	// OLD: logger = logger.Level(zerolog.InfoLevel)
	// NEW: logger = logger.SetLevel(bolt.INFO)
	logger = logger.SetLevel(bolt.INFO)

	// Step 4: Use the same logging API (mostly compatible)
	logger.Info().
		Str("service", "migration-example").
		Int("step", 4).
		Msg("Migration completed successfully")
}

// Performance comparison function
func DemonstratePerformanceImprovement() {
	// This function would typically be in a separate benchmark file
	// but serves as an example of the performance benefits

	println("Zerolog to Bolt Migration Performance Benefits:")
	println("- 64% faster logging operations")
	println("- Zero memory allocations in hot paths")
	println("- Sub-100ns latency per log operation")
	println("- Better performance under concurrent load")
	println("- Reduced memory footprint")
	println("- Seamless OpenTelemetry integration")
}

// Common Migration Patterns

// Pattern 1: Global Logger Migration
var (
	// OLD: zerologGlobalLogger = zerolog.New(os.Stdout).Level(zerolog.InfoLevel)
	// NEW:
	boltGlobalLogger = bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)
)

// ExampleGlobalLoggerUsage shows how to use the global logger
func ExampleGlobalLoggerUsage() {
	boltGlobalLogger.Info().Msg("Global logger example")
}

// Pattern 2: Logger Interface Migration
type LoggerInterface interface {
	Info() LogEvent
	Error() LogEvent
	Debug() LogEvent
	Warn() LogEvent
}

type LogEvent interface {
	Str(key, value string) LogEvent
	Int(key string, value int) LogEvent
	Msg(message string)
}

// BoltLogger implements the Logger interface using Bolt
type BoltLogger struct {
	logger *bolt.Logger
}

func (bl *BoltLogger) Info() LogEvent  { return &BoltLogEvent{bl.logger.Info()} }
func (bl *BoltLogger) Error() LogEvent { return &BoltLogEvent{bl.logger.Error()} }
func (bl *BoltLogger) Debug() LogEvent { return &BoltLogEvent{bl.logger.Debug()} }
func (bl *BoltLogger) Warn() LogEvent  { return &BoltLogEvent{bl.logger.Warn()} }

type BoltLogEvent struct {
	event *bolt.Event
}

func (ble *BoltLogEvent) Str(key, value string) LogEvent {
	ble.event.Str(key, value)
	return ble
}

func (ble *BoltLogEvent) Int(key string, value int) LogEvent {
	ble.event.Int(key, value)
	return ble
}

func (ble *BoltLogEvent) Msg(message string) {
	ble.event.Msg(message)
}

// Usage example of the interface-based approach
func ExampleInterfaceBasedMigration() {
	var logger LoggerInterface = &BoltLogger{
		logger: bolt.New(bolt.NewJSONHandler(os.Stdout)),
	}

	logger.Info().
		Str("pattern", "interface-based").
		Int("migration_step", 1).
		Msg("Interface-based migration successful")
}

// Advanced Migration Examples

// Example 11: Middleware and HTTP Logging
func ExampleHTTPMiddleware() {
	// === BEFORE (Zerolog) ===
	// zerologLogger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)
	// func LoggingMiddleware(next http.Handler) http.Handler {
	//     return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//         start := time.Now()
	//         zerologLogger.Info().
	//             Str("method", r.Method).
	//             Str("path", r.URL.Path).
	//             Str("remote_addr", r.RemoteAddr).
	//             Msg("Request started")
	//
	//         next.ServeHTTP(w, r)
	//
	//         zerologLogger.Info().
	//             Str("method", r.Method).
	//             Str("path", r.URL.Path).
	//             Dur("duration", time.Since(start)).
	//             Msg("Request completed")
	//     })
	// }

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)

	// Example of how the middleware would look with Bolt
	start := time.Now()
	boltLogger.Info().
		Str("method", "GET").
		Str("path", "/api/users").
		Str("remote_addr", "192.168.1.100").
		Msg("Request started")

	// ... process request ...

	boltLogger.Info().
		Str("method", "GET").
		Str("path", "/api/users").
		Dur("duration", time.Since(start)).
		Int("status", 200).
		Msg("Request completed")
}

// Example 12: Database Connection Logging
func ExampleDatabaseLogging() {
	// === BEFORE (Zerolog) ===
	// zerologLogger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)
	// func ConnectDB(dsn string) error {
	//     zerologLogger.Info().
	//         Str("dsn", dsn).
	//         Msg("Attempting database connection")
	//
	//     // ... connection logic ...
	//
	//     zerologLogger.Info().
	//         Str("database", "postgresql").
	//         Str("status", "connected").
	//         Msg("Database connection established")
	//     return nil
	// }

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)

	dsn := "postgres://user:pass@localhost/db"
	boltLogger.Info().
		Str("dsn", dsn).
		Msg("Attempting database connection")

	// ... connection logic ...

	boltLogger.Info().
		Str("database", "postgresql").
		Str("status", "connected").
		Int("max_connections", 100).
		Bool("ssl_enabled", true).
		Msg("Database connection established")
}

// Example 13: Worker/Queue Processing
func ExampleWorkerLogging() {
	// === BEFORE (Zerolog) ===
	// zerologLogger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)
	// func ProcessJob(jobID string, jobType string) {
	//     jobLogger := zerologLogger.With().
	//         Str("job_id", jobID).
	//         Str("job_type", jobType).
	//         Logger()
	//
	//     jobLogger.Info().Msg("Job started")
	//
	//     // ... processing ...
	//
	//     jobLogger.Info().
	//         Int("items_processed", 250).
	//         Msg("Job completed successfully")
	// }

	// === AFTER (Bolt) ===
	baseBoltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)

	jobID := "job_12345"
	jobType := "email_batch"
	jobLogger := baseBoltLogger.With().
		Str("job_id", jobID).
		Str("job_type", jobType).
		Logger()

	jobLogger.Info().Msg("Job started")

	// ... processing ...

	jobLogger.Info().
		Int("items_processed", 250).
		Dur("processing_time", 45*time.Second).
		Float64("success_rate", 98.5).
		Msg("Job completed successfully")
}

// Example 14: Panic Recovery Logging
func ExamplePanicRecovery() {
	// === BEFORE (Zerolog) ===
	// zerologLogger := zerolog.New(os.Stdout).Level(zerolog.ErrorLevel)
	// func RecoveryMiddleware() {
	//     if r := recover(); r != nil {
	//         zerologLogger.Error().
	//             Interface("panic", r).
	//             Str("stack", string(debug.Stack())).
	//             Msg("Panic recovered")
	//     }
	// }

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.ERROR)

	// Simulating panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				boltLogger.Error().
					Any("panic", r).
					Stack().  // Bolt has built-in stack trace support
					Caller(). // Add caller information
					Msg("Panic recovered")
			}
		}()

		// This would normally trigger a panic
		// panic("example panic")
	}()
}

// Example 15: Rate Limiting and Throttling Logs
func ExampleRateLimitLogging() {
	// === BEFORE (Zerolog) ===
	// zerologLogger := zerolog.New(os.Stdout).Level(zerolog.WarnLevel)
	// func CheckRateLimit(userID int, endpoint string) bool {
	//     if rateLimitExceeded {
	//         zerologLogger.Warn().
	//             Int("user_id", userID).
	//             Str("endpoint", endpoint).
	//             Str("limit_type", "requests_per_minute").
	//             Int("current_count", 150).
	//             Int("limit", 100).
	//             Msg("Rate limit exceeded")
	//         return false
	//     }
	//     return true
	// }

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.WARN)

	userID := 12345
	endpoint := "/api/data"
	rateLimitExceeded := true // Example condition

	if rateLimitExceeded {
		boltLogger.Warn().
			Int("user_id", userID).
			Str("endpoint", endpoint).
			Str("limit_type", "requests_per_minute").
			Int("current_count", 150).
			Int("limit", 100).
			Time("reset_time", time.Now().Add(time.Minute)).
			Str("action", "request_blocked").
			Msg("Rate limit exceeded")
	}
}

// Example 16: Microservice Communication Logging
func ExampleMicroserviceLogging() {
	// === BEFORE (Zerolog) ===
	// zerologLogger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)
	// func CallExternalService(serviceName string, endpoint string) {
	//     requestID := uuid.New().String()
	//
	//     zerologLogger.Info().
	//         Str("service", serviceName).
	//         Str("endpoint", endpoint).
	//         Str("request_id", requestID).
	//         Msg("Calling external service")
	//
	//     // ... make request ...
	//
	//     zerologLogger.Info().
	//         Str("service", serviceName).
	//         Str("request_id", requestID).
	//         Int("status_code", 200).
	//         Dur("response_time", 150*time.Millisecond).
	//         Msg("External service call completed")
	// }

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)

	serviceName := "user-service"
	endpoint := "/users/12345"
	requestID := "req_abc123"

	boltLogger.Info().
		Str("service", serviceName).
		Str("endpoint", endpoint).
		Str("request_id", requestID).
		Str("correlation_id", "corr_xyz789").
		Msg("Calling external service")

	// ... make request ...

	boltLogger.Info().
		Str("service", serviceName).
		Str("request_id", requestID).
		Int("status_code", 200).
		Dur("response_time", 150*time.Millisecond).
		Int("retry_count", 0).
		Bool("cached", false).
		Msg("External service call completed")
}

// Example 17: Configuration and Environment Logging
func ExampleConfigurationLogging() {
	// === BEFORE (Zerolog) ===
	// zerologLogger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)
	// func LogConfiguration() {
	//     zerologLogger.Info().
	//         Str("environment", os.Getenv("ENV")).
	//         Str("version", "v1.2.3").
	//         Bool("debug_enabled", strings.ToLower(os.Getenv("DEBUG")) == "true").
	//         Msg("Application configuration loaded")
	// }

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)

	// Enhanced configuration logging with more detail
	boltLogger.Info().
		Str("environment", "production").
		Str("version", "v2.1.3").
		Str("build_hash", "abc123def").
		Bool("debug_enabled", false).
		Bool("metrics_enabled", true).
		Int("max_connections", 1000).
		Str("database_host", "db.example.com").
		Int("server_port", 8080).
		Str("log_level", "INFO").
		Msg("Application configuration loaded")
}

// Example 18: Security Event Logging
func ExampleSecurityEventLogging() {
	// === BEFORE (Zerolog) ===
	// zerologLogger := zerolog.New(os.Stdout).Level(zerolog.WarnLevel)
	// func LogSecurityEvent(event string, userID int, ipAddress string) {
	//     zerologLogger.Warn().
	//         Str("event_type", "security").
	//         Str("event", event).
	//         Int("user_id", userID).
	//         Str("ip_address", ipAddress).
	//         Msg("Security event detected")
	// }

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.WARN)

	// Enhanced security logging with more context
	boltLogger.Warn().
		Str("event_type", "security").
		Str("event", "failed_login_attempt").
		Int("user_id", 12345).
		Str("username", "john_doe").
		Str("ip_address", "192.168.1.100").
		Str("user_agent", "Mozilla/5.0...").
		Int("attempt_count", 5).
		Time("first_attempt", time.Now().Add(-10*time.Minute)).
		Bool("account_locked", true).
		Str("severity", "high").
		RandID("incident_id"). // Bolt's built-in random ID generator
		Msg("Security event: Multiple failed login attempts")
}

// Example 19: Performance Metrics Logging
func ExamplePerformanceMetricsLogging() {
	// === BEFORE (Zerolog) ===
	// zerologLogger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)
	// func LogPerformanceMetrics(operation string, duration time.Duration) {
	//     zerologLogger.Info().
	//         Str("operation", operation).
	//         Dur("duration", duration).
	//         Msg("Performance metric")
	// }

	// === AFTER (Bolt) ===
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)

	// Enhanced performance metrics with system information
	var counter int64 = 12345

	boltLogger.Info().
		Str("metric_type", "performance").
		Str("operation", "database_query").
		Str("query_type", "SELECT").
		Dur("query_duration", 25*time.Millisecond).
		Dur("total_duration", 45*time.Millisecond).
		Int("rows_affected", 150).
		Float64("cpu_usage", 15.5).
		Float64("memory_usage_mb", 245.7).
		Counter("total_queries", &counter). // Bolt's built-in counter support
		Bool("query_cached", false).
		Str("database", "users_db").
		Msg("Database query performance metrics")
}

// Example 20: Complete Migration Workflow
func ExampleCompleteMigrationWorkflow() {
	// This example shows a complete before/after migration for a realistic service

	// === BEFORE (Zerolog) ===
	// func StartUserService() error {
	//     logger := zerolog.New(os.Stdout).
	//         Level(zerolog.InfoLevel).
	//         With().
	//         Str("service", "user-service").
	//         Str("version", "v1.0.0").
	//         Logger()
	//
	//     logger.Info().Msg("Starting user service")
	//
	//     // Database connection
	//     logger.Info().Str("component", "database").Msg("Connecting to database")
	//     // ... db connection code ...
	//     logger.Info().Str("component", "database").Msg("Database connected")
	//
	//     // HTTP server
	//     logger.Info().Str("component", "http").Int("port", 8080).Msg("Starting HTTP server")
	//     // ... server code ...
	//     logger.Info().Str("component", "http").Int("port", 8080).Msg("HTTP server started")
	//
	//     logger.Info().Msg("User service started successfully")
	//     return nil
	// }

	// === AFTER (Bolt) ===
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		SetLevel(bolt.INFO)

	// Create service-specific logger with context
	serviceLogger := logger.With().
		Str("service", "user-service").
		Str("version", "v2.0.0").
		Timestamp(). // Add timestamp to all logs
		Logger()

	serviceLogger.Info().Msg("Starting user service")

	// Database connection with enhanced logging
	dbLogger := serviceLogger.With().Str("component", "database").Logger()
	dbLogger.Info().
		Str("host", "localhost").
		Int("port", 5432).
		Str("database", "users_db").
		Msg("Connecting to database")

	// ... db connection code ...

	dbLogger.Info().
		Dur("connection_time", 150*time.Millisecond).
		Int("max_connections", 100).
		Bool("ssl_enabled", true).
		Msg("Database connected successfully")

	// HTTP server with enhanced logging
	httpLogger := serviceLogger.With().Str("component", "http").Logger()
	httpLogger.Info().
		Int("port", 8080).
		Str("host", "0.0.0.0").
		Int("read_timeout", 30).
		Int("write_timeout", 30).
		Msg("Starting HTTP server")

	// ... server code ...

	httpLogger.Info().
		Int("port", 8080).
		Dur("startup_time", 50*time.Millisecond).
		Str("status", "ready").
		Msg("HTTP server started and ready to accept connections")

	serviceLogger.Info().
		Dur("total_startup_time", 200*time.Millisecond).
		Str("status", "ready").
		Msg("User service started successfully - all components operational")
}
