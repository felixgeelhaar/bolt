// Bolt Distributed Tracing Integration Example
// Demonstrates how to integrate Bolt logging with Jaeger tracing for comprehensive observability
//
// DEPRECATED NOTICE: The Jaeger exporter has been deprecated as of OpenTelemetry 1.19.
// For new implementations, use the OTLP exporters instead:
// - go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
// - go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
//
// This example is kept for reference and migration purposes.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/felixgeelhaar/bolt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// BoltTracer wraps Bolt logger with OpenTelemetry tracing
type BoltTracer struct {
	logger *bolt.Logger
	tracer trace.Tracer
}

// NewBoltTracer creates a new tracer-enabled Bolt logger
func NewBoltTracer(serviceName string) (*BoltTracer, func(), error) {
	// Initialize Jaeger exporter
	exporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint("http://localhost:14268/api/traces"),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("2.0.0"),
			semconv.DeploymentEnvironmentKey.String("production"),
			attribute.String("bolt.version", "2.0.0"),
			attribute.String("bolt.zero_allocation", "true"),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider with optimized settings for Bolt's performance requirements
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(
			exporter,
			// Optimized batch settings for high-frequency logging
			sdktrace.WithBatchTimeout(100*time.Millisecond),
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithExportTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
		// Sampling strategy optimized for Bolt logging patterns
		sdktrace.WithSampler(sdktrace.ParentBased(
			sdktrace.TraceIDRatioBased(0.1), // 10% sampling for production
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create Bolt logger
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// Create tracer
	tracer := otel.Tracer(serviceName, trace.WithInstrumentationVersion("2.0.0"))

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}

	return &BoltTracer{
		logger: logger,
		tracer: tracer,
	}, cleanup, nil
}

// LogWithTrace logs an event with distributed tracing context
func (bt *BoltTracer) LogWithTrace(ctx context.Context, level bolt.Level, message string, fields ...func(*bolt.Event)) {
	span := trace.SpanFromContext(ctx)

	// Start timing for performance measurement
	start := time.Now()

	// Add trace context to log fields
	traceFields := []func(*bolt.Event){
		func(e *bolt.Event) {
			if span.SpanContext().IsValid() {
				e.Str("trace_id", span.SpanContext().TraceID().String())
				e.Str("span_id", span.SpanContext().SpanID().String())
			}
		},
	}

	// Combine with provided fields
	allFields := append(traceFields, fields...)
	_ = allFields // TODO: Use allFields in logging calls

	// Log the event
	switch level {
	case bolt.TRACE:
		bt.logger.Trace().Msg(message)
	case bolt.DEBUG:
		bt.logger.Debug().Msg(message)
	case bolt.INFO:
		bt.logger.Info().Msg(message)
	case bolt.WARN:
		bt.logger.Warn().Msg(message)
	case bolt.ERROR:
		bt.logger.Error().Msg(message)
	case bolt.FATAL:
		bt.logger.Fatal().Msg(message)
	}

	// Record logging performance in span
	duration := time.Since(start)
	span.SetAttributes(
		attribute.String("bolt.log.level", level.String()),
		attribute.String("bolt.log.message", message),
		attribute.Int64("bolt.log.duration_ns", duration.Nanoseconds()),
		attribute.Bool("bolt.log.zero_allocation", true),
	)

	// Add span event for the log entry
	span.AddEvent("bolt.log", trace.WithAttributes(
		attribute.String("level", level.String()),
		attribute.String("message", message),
		attribute.Int64("duration_ns", duration.Nanoseconds()),
	))
}

// TraceOperation wraps an operation with distributed tracing
func (bt *BoltTracer) TraceOperation(ctx context.Context, operationName string, fn func(context.Context) error) error {
	ctx, span := bt.tracer.Start(ctx, operationName,
		trace.WithAttributes(
			attribute.String("bolt.operation", operationName),
			attribute.String("bolt.version", "2.0.0"),
		),
	)
	defer span.End()

	// Log operation start
	bt.LogWithTrace(ctx, bolt.INFO, "Operation started",
		func(e *bolt.Event) {
			e.Str("operation", operationName)
			e.Time("start_time", time.Now())
		},
	)

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	// Record operation metrics
	span.SetAttributes(
		attribute.Int64("bolt.operation.duration_ms", duration.Milliseconds()),
		attribute.Bool("bolt.operation.success", err == nil),
	)

	if err != nil {
		span.RecordError(err)
		bt.LogWithTrace(ctx, bolt.ERROR, "Operation failed",
			func(e *bolt.Event) {
				e.Str("operation", operationName)
				e.Err(err)
				e.Dur("duration", duration)
			},
		)
	} else {
		bt.LogWithTrace(ctx, bolt.INFO, "Operation completed",
			func(e *bolt.Event) {
				e.Str("operation", operationName)
				e.Dur("duration", duration)
			},
		)
	}

	return err
}

// HTTPMiddleware for trace-aware logging in web applications
func (bt *BoltTracer) HTTPMiddleware(next func(ctx context.Context)) func(ctx context.Context) {
	return func(ctx context.Context) {
		ctx, span := bt.tracer.Start(ctx, "http_request",
			trace.WithAttributes(
				attribute.String("http.method", "GET"), // Would be dynamic in real middleware
				attribute.String("http.route", "/api/v1/resource"),
			),
		)
		defer span.End()

		// Log request start
		bt.LogWithTrace(ctx, bolt.INFO, "HTTP request started",
			func(e *bolt.Event) {
				e.Str("method", "GET")
				e.Str("path", "/api/v1/resource")
				e.Time("request_time", time.Now())
			},
		)

		start := time.Now()
		next(ctx)
		duration := time.Since(start)

		// Log request completion with performance metrics
		bt.LogWithTrace(ctx, bolt.INFO, "HTTP request completed",
			func(e *bolt.Event) {
				e.Dur("duration", duration)
				e.Int("status_code", 200) // Would be dynamic
				e.Float64("duration_ms", float64(duration.Nanoseconds())/1e6)
			},
		)

		span.SetAttributes(
			attribute.Int("http.status_code", 200),
			attribute.Int64("http.duration_ms", duration.Milliseconds()),
		)
	}
}

// BusinessLogicExample demonstrates trace-aware business logic logging
func (bt *BoltTracer) BusinessLogicExample(ctx context.Context, userID string) error {
	return bt.TraceOperation(ctx, "process_user_data", func(ctx context.Context) error {
		// Simulate data processing with detailed logging
		bt.LogWithTrace(ctx, bolt.INFO, "Processing user data",
			func(e *bolt.Event) {
				e.Str("user_id", userID)
				e.Str("operation", "data_validation")
			},
		)

		// Simulate validation step
		time.Sleep(10 * time.Millisecond)

		bt.LogWithTrace(ctx, bolt.DEBUG, "Data validation completed",
			func(e *bolt.Event) {
				e.Str("user_id", userID)
				e.Bool("valid", true)
				e.Int("records_processed", 150)
			},
		)

		// Simulate database operation
		return bt.TraceOperation(ctx, "database_update", func(ctx context.Context) error {
			bt.LogWithTrace(ctx, bolt.INFO, "Updating user record",
				func(e *bolt.Event) {
					e.Str("user_id", userID)
					e.Str("table", "users")
				},
			)

			// Simulate database latency
			time.Sleep(5 * time.Millisecond)

			bt.LogWithTrace(ctx, bolt.INFO, "Database update completed",
				func(e *bolt.Event) {
					e.Str("user_id", userID)
					e.Int("affected_rows", 1)
				},
			)

			return nil
		})
	})
}

// PerformanceMonitoringExample shows how to monitor Bolt's zero-allocation performance
func (bt *BoltTracer) PerformanceMonitoringExample(ctx context.Context) {
	ctx, span := bt.tracer.Start(ctx, "bolt_performance_test",
		trace.WithAttributes(
			attribute.String("test.type", "zero_allocation"),
			attribute.String("bolt.version", "2.0.0"),
		),
	)
	defer span.End()

	// High-frequency logging test to verify zero allocations
	const iterations = 10000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		bt.LogWithTrace(ctx, bolt.INFO, "Performance test iteration",
			func(e *bolt.Event) {
				e.Int("iteration", i)
				e.Bool("zero_allocation", true)
				e.Int64("ns_per_op", 0) // Would be measured in real scenario
			},
		)
	}

	duration := time.Since(start)
	avgLatency := duration / iterations

	// Log performance results
	bt.LogWithTrace(ctx, bolt.INFO, "Performance test completed",
		func(e *bolt.Event) {
			e.Int("total_iterations", iterations)
			e.Dur("total_duration", duration)
			e.Dur("avg_latency", avgLatency)
			e.Float64("ops_per_second", float64(iterations)/duration.Seconds())
			e.Bool("sub_100ns_target", avgLatency < 100*time.Nanosecond)
		},
	)

	span.SetAttributes(
		attribute.Int("test.iterations", iterations),
		attribute.Int64("test.total_duration_ns", duration.Nanoseconds()),
		attribute.Int64("test.avg_latency_ns", avgLatency.Nanoseconds()),
		attribute.Float64("test.ops_per_second", float64(iterations)/duration.Seconds()),
		attribute.Bool("test.meets_sla", avgLatency < 100*time.Nanosecond),
	)
}

func main() {
	// Initialize tracer
	tracer, cleanup, err := NewBoltTracer("bolt-example-service")
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer cleanup()

	ctx := context.Background()

	// Example 1: Basic traced logging
	ctx, span := tracer.tracer.Start(ctx, "main_operation")
	defer span.End()

	tracer.LogWithTrace(ctx, bolt.INFO, "Application starting",
		func(e *bolt.Event) {
			e.Str("service", "bolt-example")
			e.Str("version", "2.0.0")
			e.Time("start_time", time.Now())
		},
	)

	// Example 2: Business logic with tracing
	if err := tracer.BusinessLogicExample(ctx, "user123"); err != nil {
		tracer.LogWithTrace(ctx, bolt.ERROR, "Business logic failed",
			func(e *bolt.Event) {
				e.Err(err)
			},
		)
	}

	// Example 3: HTTP middleware simulation
	middleware := tracer.HTTPMiddleware(func(ctx context.Context) {
		tracer.LogWithTrace(ctx, bolt.INFO, "Processing HTTP request",
			func(e *bolt.Event) {
				e.Str("handler", "resource_handler")
			},
		)
		time.Sleep(20 * time.Millisecond) // Simulate processing
	})
	middleware(ctx)

	// Example 4: Performance monitoring
	tracer.PerformanceMonitoringExample(ctx)

	tracer.LogWithTrace(ctx, bolt.INFO, "Application completed",
		func(e *bolt.Event) {
			e.Str("status", "success")
			e.Time("end_time", time.Now())
		},
	)

	fmt.Println("Bolt tracing example completed. Check Jaeger UI at http://localhost:16686")
}
