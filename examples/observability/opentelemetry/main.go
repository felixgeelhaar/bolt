// Package main demonstrates OpenTelemetry integration with Bolt logging.
// This example shows distributed tracing, metrics collection, and structured logging
// with full observability stack integration.
//
// NOTE: This example includes both Jaeger (deprecated) and OTLP exporters
// for comparison. New implementations should use OTLP exporters.
package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/felixgeelhaar/bolt"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

// Application represents the main application with observability
type Application struct {
	logger bolt.Logger
	tracer trace.Tracer
	meter  metric.Meter

	// Metrics
	requestCounter     metric.Int64Counter
	requestDuration    metric.Float64Histogram
	activeConnections  metric.Int64UpDownCounter
	databaseOperations metric.Int64Counter
	errorCounter       metric.Int64Counter
}

// NewApplication creates a new application with full observability setup
func NewApplication() (*Application, error) {
	// Initialize tracing
	tracerProvider, err := initTracing()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}
	otel.SetTracerProvider(tracerProvider)

	// Initialize metrics
	meterProvider, err := initMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}
	otel.SetMeterProvider(meterProvider)

	// Initialize propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create logger with OpenTelemetry integration
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		Level(bolt.InfoLevel).
		With().
		Str("service", "otel-demo").
		Str("version", "v1.0.0").
		Logger()

	// Get tracer and meter
	tracer := otel.Tracer("bolt-otel-demo")
	meter := otel.Meter("bolt-otel-demo")

	// Create metrics
	requestCounter, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request counter: %w", err)
	}

	requestDuration, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create duration histogram: %w", err)
	}

	activeConnections, err := meter.Int64UpDownCounter(
		"active_connections",
		metric.WithDescription("Number of active connections"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection counter: %w", err)
	}

	databaseOperations, err := meter.Int64Counter(
		"database_operations_total",
		metric.WithDescription("Total number of database operations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create database counter: %w", err)
	}

	errorCounter, err := meter.Int64Counter(
		"errors_total",
		metric.WithDescription("Total number of errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create error counter: %w", err)
	}

	return &Application{
		logger:             logger,
		tracer:             tracer,
		meter:              meter,
		requestCounter:     requestCounter,
		requestDuration:    requestDuration,
		activeConnections:  activeConnections,
		databaseOperations: databaseOperations,
		errorCounter:       errorCounter,
	}, nil
}

// initTracing initializes OpenTelemetry tracing
func initTracing() (*tracesdk.TracerProvider, error) {
	// Create Jaeger exporter
	jaegerExporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(
		getEnv("JAEGER_ENDPOINT", "http://localhost:14268/api/traces"),
	)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create OTLP exporter (alternative to Jaeger)
	otlpExporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint(getEnv("OTLP_ENDPOINT", "http://localhost:4318")),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("bolt-otel-demo"),
			semconv.ServiceVersionKey.String("v1.0.0"),
			semconv.ServiceInstanceIDKey.String(uuid.New().String()),
			semconv.DeploymentEnvironmentKey.String(getEnv("ENVIRONMENT", "development")),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(jaegerExporter),
		tracesdk.WithBatcher(otlpExporter),
		tracesdk.WithResource(res),
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
	)

	return tp, nil
}

// initMetrics initializes OpenTelemetry metrics
func initMetrics() (*metricsdk.MeterProvider, error) {
	// Create Prometheus exporter
	prometheusExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("bolt-otel-demo"),
			semconv.ServiceVersionKey.String("v1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create meter provider
	mp := metricsdk.NewMeterProvider(
		metricsdk.WithReader(prometheusExporter),
		metricsdk.WithResource(res),
	)

	return mp, nil
}

// HTTP Handlers with full observability

func (app *Application) rootHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Start tracing span
	ctx, span := app.tracer.Start(ctx, "root_handler",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.user_agent", r.UserAgent()),
		),
	)
	defer span.End()

	// Extract correlation ID from context or generate new one
	correlationID := getOrCreateCorrelationID(ctx, r)
	span.SetAttributes(attribute.String("correlation_id", correlationID))

	// Start timing
	start := time.Now()

	// Increment active connections
	app.activeConnections.Add(ctx, 1,
		metric.WithAttributes(attribute.String("endpoint", "/")),
	)
	defer app.activeConnections.Add(ctx, -1,
		metric.WithAttributes(attribute.String("endpoint", "/")),
	)

	// Add trace and span IDs to logger context
	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("trace_id", traceID).
		Str("span_id", spanID).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("remote_addr", r.RemoteAddr).
		Msg("Processing root request")

	// Simulate some work
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

	// Create response
	response := map[string]interface{}{
		"message":        "Bolt OpenTelemetry Demo",
		"correlation_id": correlationID,
		"trace_id":       traceID,
		"span_id":        spanID,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}

	// Record metrics
	duration := time.Since(start)
	app.requestCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("endpoint", "/"),
			attribute.String("status", "200"),
		),
	)
	app.requestDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("endpoint", "/"),
		),
	)

	// Add span events
	span.AddEvent("response_prepared",
		trace.WithAttributes(
			attribute.Int("response_size", len(fmt.Sprintf("%v", response))),
		),
	)

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("trace_id", traceID).
		Str("span_id", spanID).
		Dur("duration", duration).
		Float64("duration_ms", float64(duration.Nanoseconds())/1_000_000).
		Msg("Root request completed")

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Correlation-ID", correlationID)
	w.Header().Set("X-Trace-ID", traceID)

	fmt.Fprintf(w, `{
		"message": "%s",
		"correlation_id": "%s",
		"trace_id": "%s",
		"span_id": "%s",
		"timestamp": "%s"
	}`,
		response["message"],
		response["correlation_id"],
		response["trace_id"],
		response["span_id"],
		response["timestamp"],
	)
}

func (app *Application) usersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Start tracing span
	ctx, span := app.tracer.Start(ctx, "users_handler",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("operation", "list_users"),
		),
	)
	defer span.End()

	correlationID := getOrCreateCorrelationID(ctx, r)
	span.SetAttributes(attribute.String("correlation_id", correlationID))

	start := time.Now()
	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("trace_id", traceID).
		Str("span_id", spanID).
		Str("operation", "list_users").
		Msg("Processing users request")

	// Simulate database operation with child span
	users, err := app.fetchUsersFromDatabase(ctx, correlationID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error", "true"))

		app.errorCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("operation", "list_users"),
				attribute.String("error_type", "database_error"),
			),
		)

		app.logger.Error().
			Str("correlation_id", correlationID).
			Str("trace_id", traceID).
			Str("span_id", spanID).
			Err(err).
			Msg("Failed to fetch users")

		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	duration := time.Since(start)

	// Record metrics
	app.requestCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("endpoint", "/users"),
			attribute.String("status", "200"),
		),
	)
	app.requestDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("endpoint", "/users"),
		),
	)

	// Add span attributes
	span.SetAttributes(
		attribute.Int("users.count", len(users)),
		attribute.String("database.operation", "SELECT"),
	)

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("trace_id", traceID).
		Str("span_id", spanID).
		Int("users_count", len(users)).
		Dur("duration", duration).
		Float64("duration_ms", float64(duration.Nanoseconds())/1_000_000).
		Msg("Users request completed")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Correlation-ID", correlationID)
	w.Header().Set("X-Trace-ID", traceID)

	fmt.Fprintf(w, `{
		"users": %s,
		"total": %d,
		"correlation_id": "%s",
		"trace_id": "%s",
		"processing_time_ms": %.3f
	}`,
		fmt.Sprintf("%v", users),
		len(users),
		correlationID,
		traceID,
		float64(duration.Nanoseconds())/1_000_000,
	)
}

func (app *Application) errorHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Start tracing span
	ctx, span := app.tracer.Start(ctx, "error_handler",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
		),
	)
	defer span.End()

	correlationID := getOrCreateCorrelationID(ctx, r)
	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()

	// Simulate an error
	err := fmt.Errorf("simulated error for demonstration")
	span.RecordError(err)
	span.SetAttributes(
		attribute.String("error", "true"),
		attribute.String("error.type", "simulation_error"),
	)

	// Record error metrics
	app.errorCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("endpoint", "/error"),
			attribute.String("error_type", "simulation_error"),
		),
	)

	app.logger.Error().
		Str("correlation_id", correlationID).
		Str("trace_id", traceID).
		Str("span_id", spanID).
		Err(err).
		Str("error_type", "simulation_error").
		Msg("Simulated error occurred")

	w.Header().Set("X-Correlation-ID", correlationID)
	w.Header().Set("X-Trace-ID", traceID)
	http.Error(w, "Simulated Error", http.StatusInternalServerError)
}

// Database simulation with tracing
func (app *Application) fetchUsersFromDatabase(ctx context.Context, correlationID string) ([]map[string]interface{}, error) {
	// Create child span for database operation
	ctx, span := app.tracer.Start(ctx, "db_query_users",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.statement", "SELECT * FROM users LIMIT 10"),
		),
	)
	defer span.End()

	start := time.Now()

	// Simulate database query time
	queryTime := time.Duration(rand.Intn(100)) * time.Millisecond
	time.Sleep(queryTime)

	// Record database operation metric
	app.databaseOperations.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", "SELECT"),
			attribute.String("table", "users"),
		),
	)

	// Simulate potential database error (10% chance)
	if rand.Intn(10) == 0 {
		err := fmt.Errorf("database connection timeout")
		span.RecordError(err)
		span.SetAttributes(attribute.String("error", "true"))

		app.logger.Error().
			Str("correlation_id", correlationID).
			Str("trace_id", span.SpanContext().TraceID().String()).
			Str("span_id", span.SpanContext().SpanID().String()).
			Err(err).
			Dur("query_duration", time.Since(start)).
			Msg("Database query failed")

		return nil, err
	}

	// Create mock users
	users := []map[string]interface{}{
		{"id": 1, "name": "John Doe", "email": "john@example.com"},
		{"id": 2, "name": "Jane Smith", "email": "jane@example.com"},
	}

	span.SetAttributes(
		attribute.Int("db.rows_affected", len(users)),
		attribute.String("db.status", "success"),
	)

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("trace_id", span.SpanContext().TraceID().String()).
		Str("span_id", span.SpanContext().SpanID().String()).
		Int("users_returned", len(users)).
		Dur("query_duration", time.Since(start)).
		Float64("query_duration_ms", float64(time.Since(start).Nanoseconds())/1_000_000).
		Msg("Database query completed")

	return users, nil
}

// Middleware with observability
func (app *Application) tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract tracing context from headers
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// Utility functions
func getOrCreateCorrelationID(ctx context.Context, r *http.Request) string {
	// Try to get from header first
	if correlationID := r.Header.Get("X-Correlation-ID"); correlationID != "" {
		return correlationID
	}

	// Try to get from span baggage
	if baggage := trace.SpanFromContext(ctx).SpanContext(); baggage.IsValid() {
		// Could extract from baggage if set
	}

	// Generate new correlation ID
	return uuid.New().String()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Health check handler
func (app *Application) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx, span := app.tracer.Start(ctx, "health_check")
	defer span.End()

	correlationID := getOrCreateCorrelationID(ctx, r)

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("trace_id", span.SpanContext().TraceID().String()).
		Msg("Health check performed")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s","correlation_id":"%s"}`,
		time.Now().UTC().Format(time.RFC3339), correlationID)
}

// Metrics handler
func (app *Application) metricsHandler() http.Handler {
	// Return Prometheus metrics handler
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This will be handled by the Prometheus exporter
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "# Metrics endpoint - use Prometheus exporter")
	})
}

func main() {
	// Initialize application
	app, err := NewApplication()
	if err != nil {
		fmt.Printf("Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Wrap all handlers with tracing middleware
	mux.Handle("/", app.tracingMiddleware(http.HandlerFunc(app.rootHandler)))
	mux.Handle("/users", app.tracingMiddleware(http.HandlerFunc(app.usersHandler)))
	mux.Handle("/error", app.tracingMiddleware(http.HandlerFunc(app.errorHandler)))
	mux.Handle("/health", app.tracingMiddleware(http.HandlerFunc(app.healthHandler)))
	mux.Handle("/metrics", app.metricsHandler())

	port := getEnv("PORT", "8080")

	app.logger.Info().
		Str("port", port).
		Str("service", "otel-demo").
		Msg("Starting server with OpenTelemetry integration")

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		app.logger.Fatal().Err(err).Msg("Server failed to start")
	}
}
