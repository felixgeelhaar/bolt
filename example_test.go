package bolt_test

import (
	"log/slog"
	"os"

	"github.com/felixgeelhaar/bolt/v3"
)

func ExampleNew() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().Str("service", "api").Msg("started")
}

func ExampleNew_console() {
	logger := bolt.New(bolt.NewConsoleHandler(os.Stdout))
	logger.Info().Str("env", "development").Msg("ready")
}

func ExampleLogger_Info() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Str("method", "GET").
		Int("status", 200).
		Msg("request handled")
}

func ExampleLogger_Error() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Error().
		Str("component", "database").
		Msg("connection failed")
}

func ExampleLogger_With() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// Create a child logger with persistent context fields
	reqLogger := logger.With().
		Str("request_id", "abc-123").
		Str("user_id", "user-456").
		Logger()

	// All subsequent logs include request_id and user_id
	reqLogger.Info().Msg("processing request")
	reqLogger.Info().Int("items", 42).Msg("query complete")
}

func ExampleLogger_SetLevel() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// Only warn and above will be logged
	logger.SetLevel(bolt.WARN)

	logger.Info().Msg("this will be suppressed")
	logger.Warn().Msg("this will appear")
}

func ExampleNewSlogHandler() {
	// Use Bolt as a backend for the standard slog package
	handler := bolt.NewSlogHandler(os.Stdout, nil)
	logger := slog.New(handler)

	logger.Info("request handled",
		"method", "GET",
		"status", 200,
		"path", "/api/users",
	)
}

func ExampleNewSlogHandler_withLevel() {
	handler := bolt.NewSlogHandler(os.Stdout, &bolt.SlogHandlerOptions{
		Level: slog.LevelWarn,
	})
	logger := slog.New(handler)

	logger.Info("filtered out") // suppressed
	logger.Warn("visible")      // appears
}

func ExampleEvent_Str() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Str("user", "alice").
		Str("action", "login").
		Msg("user authenticated")
}

func ExampleEvent_Int() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Int("port", 8080).
		Int("workers", 4).
		Msg("server configured")
}

func ExampleEvent_Float64() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Float64("latency_ms", 12.345).
		Float64("cpu_percent", 42.7).
		Msg("metrics collected")
}

func ExampleEvent_Bool() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Bool("authenticated", true).
		Bool("admin", false).
		Msg("access check")
}

func ExampleEvent_Err() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	err := os.ErrNotExist
	logger.Error().
		Err(err).
		Str("path", "/tmp/missing.txt").
		Msg("file not found")
}
