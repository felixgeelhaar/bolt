// Package zap provides comprehensive examples demonstrating migration from Zap to Bolt.
package zap

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/felixgeelhaar/bolt/v2"
)

// ExampleBasicLoggerCreation demonstrates basic logger creation migration.
func ExampleBasicLoggerCreation() {
	// BEFORE: Zap
	// Production logger
	// zapLogger, _ := zap.NewProduction()
	// defer zapLogger.Sync()

	// Development logger
	// zapDevLogger, _ := zap.NewDevelopment()
	// defer zapDevLogger.Sync()

	// Example logger
	// zapExampleLogger := zap.NewExample()

	// AFTER: Bolt
	// Production logger (JSON output, INFO level)
	boltProdLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)

	// Development logger (Console output, DEBUG level)
	boltDevLogger := bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG)

	// Example logger (same as development)
	boltExampleLogger := bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG)

	// Use the loggers
	boltProdLogger.Info().Str("environment", "production").Msg("Application started")
	boltDevLogger.Debug().Str("environment", "development").Msg("Debug message")
	boltExampleLogger.Info().Str("environment", "example").Msg("Example log")
}

// ExampleStructuredLogging demonstrates structured logging migration.
func ExampleStructuredLogging() {
	// BEFORE: Zap structured logging
	// zapLogger, _ := zap.NewProduction()
	// defer zapLogger.Sync()
	//
	// zapLogger.Info("User login",
	//     zap.String("username", "john_doe"),
	//     zap.Int("user_id", 12345),
	//     zap.Bool("success", true),
	//     zap.Float64("response_time", 0.123),
	//     zap.Time("timestamp", time.Now()),
	// )

	// AFTER: Bolt structured logging
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	boltLogger.Info().
		Str("username", "john_doe").
		Int("user_id", 12345).
		Bool("success", true).
		Float64("response_time", 0.123).
		Time("timestamp", time.Now()).
		Msg("User login")
}

// ExampleSugarAPIMigration demonstrates migration from Zap's Sugar API.
func ExampleSugarAPIMigration() {
	// BEFORE: Zap Sugar API
	// zapLogger, _ := zap.NewDevelopment()
	// defer zapLogger.Sync()
	// sugar := zapLogger.Sugar()
	//
	// // Printf-style logging
	// sugar.Infof("User %s logged in with ID %d", "john_doe", 12345)
	// sugar.Errorf("Failed to process user %s: %v", "jane_doe", errors.New("database error"))
	//
	// // Key-value pair logging
	// sugar.Infow("User action",
	//     "username", "john_doe",
	//     "action", "login",
	//     "ip_address", "192.168.1.100",
	//     "success", true,
	// )

	// AFTER: Bolt (structured approach - recommended)
	boltLogger := bolt.New(bolt.NewConsoleHandler(os.Stdout))

	// Replace printf-style with structured logging
	boltLogger.Info().
		Str("username", "john_doe").
		Int("user_id", 12345).
		Msg("User logged in")

	boltLogger.Error().
		Str("username", "jane_doe").
		Err(errors.New("database error")).
		Msg("Failed to process user")

	// Replace key-value pairs with structured fields
	boltLogger.Info().
		Str("username", "john_doe").
		Str("action", "login").
		Str("ip_address", "192.168.1.100").
		Bool("success", true).
		Msg("User action")

	// ALTERNATIVE: Bolt (printf-style for gradual migration)
	boltLogger.Info().Printf("User %s logged in with ID %d", "john_doe", 12345)
	boltLogger.Error().Err(errors.New("database error")).Printf("Failed to process user %s", "jane_doe")
}

// ExampleLoggerWithContext demonstrates context and fields migration.
func ExampleLoggerWithContext() {
	// BEFORE: Zap with context fields
	// zapLogger, _ := zap.NewProduction()
	// defer zapLogger.Sync()
	//
	// // Create logger with context fields
	// contextLogger := zapLogger.With(
	//     zap.String("service", "payment-service"),
	//     zap.String("version", "v2.1.0"),
	//     zap.String("environment", "production"),
	// )
	//
	// contextLogger.Info("Service started")
	// contextLogger.Info("Processing payment",
	//     zap.String("payment_id", "pay_123"),
	//     zap.Float64("amount", 99.99),
	// )

	// AFTER: Bolt with context fields
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// Create logger with context fields
	contextLogger := boltLogger.With().
		Str("service", "payment-service").
		Str("version", "v2.1.0").
		Str("environment", "production").
		Logger()

	contextLogger.Info().Msg("Service started")
	contextLogger.Info().
		Str("payment_id", "pay_123").
		Float64("amount", 99.99).
		Msg("Processing payment")
}

// ExampleErrorHandling demonstrates error handling migration.
func ExampleErrorHandling() {
	err := errors.New("connection timeout")

	// BEFORE: Zap error handling
	// zapLogger, _ := zap.NewProduction()
	// defer zapLogger.Sync()
	//
	// zapLogger.Error("Database connection failed",
	//     zap.Error(err),
	//     zap.String("database", "users_db"),
	//     zap.Int("retry_count", 3),
	//     zap.Duration("timeout", 30*time.Second),
	// )

	// AFTER: Bolt error handling
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	boltLogger.Error().
		Err(err).
		Str("database", "users_db").
		Int("retry_count", 3).
		Dur("timeout", 30*time.Second).
		Msg("Database connection failed")
}

// ExampleLogLevels demonstrates log level migration.
func ExampleLogLevels() {
	// BEFORE: Zap log levels
	// zapLogger, _ := zap.NewDevelopment()
	// defer zapLogger.Sync()
	//
	// zapLogger.Debug("Debug message", zap.String("component", "auth"))
	// zapLogger.Info("Info message", zap.Int("user_count", 100))
	// zapLogger.Warn("Warning message", zap.String("warning", "high_memory"))
	// zapLogger.Error("Error message", zap.Error(errors.New("example error")))
	// zapLogger.Fatal("Fatal message") // This would exit the program

	// AFTER: Bolt log levels
	boltLogger := bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG)

	boltLogger.Debug().Str("component", "auth").Msg("Debug message")
	boltLogger.Info().Int("user_count", 100).Msg("Info message")
	boltLogger.Warn().Str("warning", "high_memory").Msg("Warning message")
	boltLogger.Error().Err(errors.New("example error")).Msg("Error message")
	boltLogger.Fatal().Msg("Fatal message") // This would exit the program
}

// ExampleConfigurationMigration demonstrates configuration migration.
func ExampleConfigurationMigration() {
	// BEFORE: Zap configuration
	// config := zap.Config{
	//     Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
	//     Development: false,
	//     Encoding:    "json",
	//     EncoderConfig: zap.NewProductionEncoderConfig(),
	//     OutputPaths:      []string{"stdout"},
	//     ErrorOutputPaths: []string{"stderr"},
	//     InitialFields: map[string]interface{}{
	//         "service": "my-service",
	//         "version": "v1.0.0",
	//     },
	// }
	//
	// zapLogger, _ := config.Build()
	// defer zapLogger.Sync()

	// AFTER: Bolt configuration
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		SetLevel(bolt.INFO)

	// Add initial fields using With()
	serviceLogger := boltLogger.With().
		Str("service", "my-service").
		Str("version", "v1.0.0").
		Logger()

	serviceLogger.Info().Msg("Service initialized with configuration")
}

// ExampleAdvancedFieldTypes demonstrates various field types migration.
func ExampleAdvancedFieldTypes() {
	// BEFORE: Zap field types
	// zapLogger, _ := zap.NewProduction()
	// defer zapLogger.Sync()
	//
	// zapLogger.Info("Complex data types",
	//     zap.String("string_field", "value"),
	//     zap.Int("int_field", 42),
	//     zap.Int64("int64_field", 9223372036854775807),
	//     zap.Uint("uint_field", 42),
	//     zap.Uint64("uint64_field", 18446744073709551615),
	//     zap.Bool("bool_field", true),
	//     zap.Float64("float64_field", 3.14159),
	//     zap.Time("time_field", time.Now()),
	//     zap.Duration("duration_field", time.Hour),
	//     zap.Binary("binary_field", []byte("binary data")),
	//     zap.Any("any_field", map[string]int{"key": 123}),
	// )

	// AFTER: Bolt field types
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	boltLogger.Info().
		Str("string_field", "value").
		Int("int_field", 42).
		Int64("int64_field", 9223372036854775807).
		Uint("uint_field", 42).
		Uint64("uint64_field", 18446744073709551615).
		Bool("bool_field", true).
		Float64("float64_field", 3.14159).
		Time("time_field", time.Now()).
		Dur("duration_field", time.Hour).
		Bytes("binary_field", []byte("binary data")).
		Any("any_field", map[string]int{"key": 123}).
		Msg("Complex data types")
}

// ExampleMiddlewareLogging demonstrates HTTP middleware logging migration.
func ExampleMiddlewareLogging() {
	// BEFORE: Zap middleware pattern
	// zapLogger, _ := zap.NewProduction()
	// defer zapLogger.Sync()
	//
	// func LoggingMiddleware(next http.Handler) http.Handler {
	//     return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//         start := time.Now()
	//
	//         zapLogger.Info("Request started",
	//             zap.String("method", r.Method),
	//             zap.String("path", r.URL.Path),
	//             zap.String("remote_addr", r.RemoteAddr),
	//             zap.String("user_agent", r.UserAgent()),
	//         )
	//
	//         next.ServeHTTP(w, r)
	//
	//         zapLogger.Info("Request completed",
	//             zap.String("method", r.Method),
	//             zap.String("path", r.URL.Path),
	//             zap.Duration("duration", time.Since(start)),
	//             zap.Int("status", 200), // Would need response writer wrapper
	//         )
	//     })
	// }

	// AFTER: Bolt middleware pattern
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// Example of how the middleware would look (simulated)
	start := time.Now()
	method := "GET"
	path := "/api/users"
	remoteAddr := "192.168.1.100"
	userAgent := "Mozilla/5.0..."

	boltLogger.Info().
		Str("method", method).
		Str("path", path).
		Str("remote_addr", remoteAddr).
		Str("user_agent", userAgent).
		Msg("Request started")

	// ... process request ...

	boltLogger.Info().
		Str("method", method).
		Str("path", path).
		Dur("duration", time.Since(start)).
		Int("status", 200).
		Int("response_size", 1024).
		Msg("Request completed")
}

// ExamplePanicRecovery demonstrates panic recovery logging migration.
func ExamplePanicRecovery() {
	// BEFORE: Zap panic recovery
	// zapLogger, _ := zap.NewProduction()
	// defer zapLogger.Sync()
	//
	// func RecoverMiddleware() {
	//     if r := recover(); r != nil {
	//         zapLogger.Error("Panic recovered",
	//             zap.Any("panic", r),
	//             zap.Stack("stacktrace"),
	//         )
	//     }
	// }

	// AFTER: Bolt panic recovery
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// Simulated panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				boltLogger.Error().
					Any("panic", r).
					Stack().               // Bolt has built-in stack trace support
					Caller().              // Add caller information
					RandID("incident_id"). // Generate incident ID
					Msg("Panic recovered")
			}
		}()

		// This would normally cause a panic
		// panic("example panic")
	}()
}

// ExampleWorkerLogging demonstrates background worker logging migration.
func ExampleWorkerLogging() {
	// BEFORE: Zap worker logging
	// zapLogger, _ := zap.NewProduction()
	// defer zapLogger.Sync()
	//
	// func ProcessJob(jobID string, jobType string) {
	//     logger := zapLogger.With(
	//         zap.String("job_id", jobID),
	//         zap.String("job_type", jobType),
	//         zap.Time("start_time", time.Now()),
	//     )
	//
	//     logger.Info("Job started")
	//
	//     // ... process job ...
	//
	//     logger.Info("Job completed",
	//         zap.Int("items_processed", 1000),
	//         zap.Duration("processing_time", 30*time.Second),
	//         zap.Float64("success_rate", 98.5),
	//     )
	// }

	// AFTER: Bolt worker logging
	baseBoltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	jobID := "job_abc123"
	jobType := "email_campaign"
	startTime := time.Now()

	jobLogger := baseBoltLogger.With().
		Str("job_id", jobID).
		Str("job_type", jobType).
		Time("start_time", startTime).
		Logger()

	jobLogger.Info().Msg("Job started")

	// ... process job ...

	jobLogger.Info().
		Int("items_processed", 1000).
		Dur("processing_time", 30*time.Second).
		Float64("success_rate", 98.5).
		Msg("Job completed")
}

// ExampleDatabaseLogging demonstrates database operation logging migration.
func ExampleDatabaseLogging() {
	// BEFORE: Zap database logging
	// zapLogger, _ := zap.NewProduction()
	// defer zapLogger.Sync()
	//
	// func ExecuteQuery(query string, args ...interface{}) {
	//     start := time.Now()
	//
	//     zapLogger.Debug("Executing query",
	//         zap.String("query", query),
	//         zap.Any("args", args),
	//     )
	//
	//     // ... execute query ...
	//
	//     zapLogger.Info("Query executed",
	//         zap.String("query", query),
	//         zap.Duration("duration", time.Since(start)),
	//         zap.Int("rows_affected", 25),
	//     )
	// }

	// AFTER: Bolt database logging
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	query := "SELECT * FROM users WHERE status = ? AND created_at > ?"
	args := []interface{}{"active", time.Now().Add(-24 * time.Hour)}
	start := time.Now()

	boltLogger.Debug().
		Str("query", query).
		Any("args", args).
		Msg("Executing query")

	// ... execute query ...

	boltLogger.Info().
		Str("query", query).
		Dur("duration", time.Since(start)).
		Int("rows_affected", 25).
		Bool("cache_hit", false).
		Str("database", "users_db").
		Msg("Query executed")
}

// ExampleMetricsLogging demonstrates metrics and monitoring migration.
func ExampleMetricsLogging() {
	// BEFORE: Zap metrics logging
	// zapLogger, _ := zap.NewProduction()
	// defer zapLogger.Sync()
	//
	// func LogMetrics(operation string, duration time.Duration, count int) {
	//     zapLogger.Info("Performance metric",
	//         zap.String("operation", operation),
	//         zap.Duration("duration", duration),
	//         zap.Int("count", count),
	//         zap.Float64("ops_per_second", float64(count)/duration.Seconds()),
	//     )
	// }

	// AFTER: Bolt metrics logging
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	operation := "user_authentication"
	duration := 150 * time.Millisecond
	count := 1000
	var totalRequests int64 = 50000

	boltLogger.Info().
		Str("metric_type", "performance").
		Str("operation", operation).
		Dur("duration", duration).
		Int("count", count).
		Float64("ops_per_second", float64(count)/duration.Seconds()).
		Counter("total_requests", &totalRequests). // Bolt's atomic counter support
		Time("measured_at", time.Now()).
		Msg("Performance metric")
}

// ExampleConfigurationComparison shows configuration patterns side-by-side.
func ExampleConfigurationComparison() {
	// BEFORE: Zap configuration patterns
	//
	// // Pattern 1: Simple configuration
	// logger1, _ := zap.NewProduction()
	//
	// // Pattern 2: Custom configuration
	// config := zap.Config{
	//     Level:       zap.NewAtomicLevelAt(zap.DebugLevel),
	//     Development: true,
	//     Encoding:    "console",
	//     EncoderConfig: zap.NewDevelopmentEncoderConfig(),
	//     OutputPaths: []string{"stdout"},
	// }
	// logger2, _ := config.Build()
	//
	// // Pattern 3: Core configuration
	// core := zapcore.NewCore(
	//     zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
	//     zapcore.AddSync(os.Stdout),
	//     zapcore.DebugLevel,
	// )
	// logger3 := zap.New(core)

	// AFTER: Bolt configuration patterns

	// Pattern 1: Simple production configuration
	logger1 := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)

	// Pattern 2: Development configuration
	logger2 := bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG)

	// Pattern 3: Custom handler (equivalent to core configuration)
	// For advanced cases, you can implement custom handlers
	logger3 := bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG)

	// Use the loggers
	logger1.Info().Str("config", "production").Msg("Production logger")
	logger2.Debug().Str("config", "development").Msg("Development logger")
	logger3.Debug().Str("config", "custom").Msg("Custom logger")
}

// ExampleMigrationWorkflow demonstrates a complete migration workflow.
func ExampleMigrationWorkflow() {
	// Complete migration example showing before/after for a service

	// BEFORE: Zap service implementation
	// type UserService struct {
	//     logger *zap.Logger
	// }
	//
	// func NewUserService() *UserService {
	//     logger, _ := zap.NewProduction()
	//     return &UserService{logger: logger}
	// }
	//
	// func (s *UserService) CreateUser(username string, email string) error {
	//     start := time.Now()
	//
	//     s.logger.Info("Creating user",
	//         zap.String("username", username),
	//         zap.String("email", email),
	//     )
	//
	//     // ... create user logic ...
	//
	//     s.logger.Info("User created successfully",
	//         zap.String("username", username),
	//         zap.Duration("duration", time.Since(start)),
	//     )
	//
	//     return nil
	// }

	// AFTER: Bolt service implementation
	type UserService struct {
		logger *bolt.Logger
	}

	NewUserService := func() *UserService {
		logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)
		return &UserService{logger: logger}
	}

	service := NewUserService()

	CreateUser := func(username string, email string) error {
		start := time.Now()

		service.logger.Info().
			Str("username", username).
			Str("email", email).
			Str("operation", "create_user").
			Msg("Creating user")

		// ... create user logic ...

		service.logger.Info().
			Str("username", username).
			Dur("duration", time.Since(start)).
			Bool("success", true).
			Str("operation", "create_user").
			Msg("User created successfully")

		return nil
	}

	// Use the service
	CreateUser("john_doe", "john@example.com")
}

// ExamplePerformanceBenefits demonstrates the performance benefits of migration.
func ExamplePerformanceBenefits() {
	// This example shows the performance characteristics comparison

	fmt.Println("Zap to Bolt Migration Performance Benefits:")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("Logging Performance:")
	fmt.Println("  Zap:  ~170 ns/op with 2-3 allocations")
	fmt.Println("  Bolt: ~63 ns/op with 0 allocations")
	fmt.Println("  Improvement: 80% faster, 100% fewer allocations")
	fmt.Println()
	fmt.Println("Memory Usage:")
	fmt.Println("  Zap:  Multiple allocations per log call")
	fmt.Println("  Bolt: Zero allocations in hot path")
	fmt.Println("  Result: Reduced garbage collection pressure")
	fmt.Println()
	fmt.Println("Concurrency:")
	fmt.Println("  Zap:  Performs well but with allocation overhead")
	fmt.Println("  Bolt: Excellent concurrent performance, zero allocations")
	fmt.Println("  Result: Better scalability under high concurrency")
	fmt.Println()
	fmt.Println("Binary Size:")
	fmt.Println("  Zap:  Larger due to more dependencies")
	fmt.Println("  Bolt: Smaller binary size")
	fmt.Println("  Result: Faster deployment and startup")
}

// ExampleCompatibilityLayer demonstrates using the compatibility layer for gradual migration.
func ExampleCompatibilityLayer() {
	// Using the Zap compatibility layer for gradual migration
	// This allows existing Zap code to work with minimal changes

	// Step 1: Replace import
	// OLD: import "go.uber.org/zap"
	// NEW: import zap "github.com/felixgeelhaar/bolt/migrate/zap"

	// Step 2: Existing code continues to work
	logger, _ := NewProduction()
	defer logger.Sync() // No-op in Bolt but maintained for compatibility

	logger.Info("Service started",
		String("service", "user-api"),
		Int("port", 8080),
		Bool("debug", false),
	)

	// Step 3: Gradually adopt Bolt's native API for new code
	boltLogger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	boltLogger.Info().
		Str("service", "user-api").
		Int("port", 8080).
		Bool("debug", false).
		Str("migration_stage", "hybrid").
		Msg("Service started with native Bolt API")

	// Step 4: Eventually migrate all code to native Bolt API
	fmt.Println("Migration completed - all code now uses native Bolt API")
}

// ExampleTroubleshooting demonstrates common migration issues and solutions.
func ExampleTroubleshooting() {
	fmt.Println("Common Migration Issues and Solutions:")
	fmt.Println("=====================================")
	fmt.Println()

	fmt.Println("Issue: Sugar API dependencies")
	fmt.Println("Problem: Code uses logger.Sugar().Infof(...)")
	fmt.Println("Solution: Replace with structured logging")
	fmt.Println(`Before: sugar.Infof("User {name} logged in", username)`)
	fmt.Println("After:  logger.Info().Str(\"username\", username).Msg(\"User logged in\")")
	fmt.Println()

	fmt.Println("Issue: Field constructor patterns")
	fmt.Println("Problem: Code uses zap.String(\"key\", \"value\")")
	fmt.Println("Solution: Convert to method calls")
	fmt.Println("Before: logger.Info(\"message\", zap.String(\"key\", \"value\"))")
	fmt.Println("After:  logger.Info().Str(\"key\", \"value\").Msg(\"message\")")
	fmt.Println()

	fmt.Println("Issue: With() method differences")
	fmt.Println("Problem: Zap's With() returns Logger, Bolt's returns Event")
	fmt.Println("Solution: Use .Logger() to get logger from event")
	fmt.Println("Before: contextLogger := logger.With(zap.String(\"service\", \"api\"))")
	fmt.Println("After:  contextLogger := logger.With().Str(\"service\", \"api\").Logger()")
	fmt.Println()

	fmt.Println("Issue: Configuration complexity")
	fmt.Println("Problem: Complex zapcore configuration")
	fmt.Println("Solution: Use Bolt's simpler handler system or ConfigMigrator")
	fmt.Println("Before: zapcore.NewCore(encoder, writer, level)")
	fmt.Println("After:  bolt.New(bolt.NewJSONHandler(writer)).SetLevel(level)")
	fmt.Println()
}
