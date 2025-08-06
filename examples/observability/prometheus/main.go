// Package main demonstrates Prometheus metrics integration with Bolt logging.
// This example shows how to collect and expose metrics while maintaining
// structured logging with correlation between metrics and logs.
package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/felixgeelhaar/bolt"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics registry
type MetricsCollector struct {
	// HTTP metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec

	// Application metrics
	activeConnections    prometheus.Gauge
	databaseConnections  *prometheus.GaugeVec
	cacheHits           *prometheus.CounterVec
	businessMetrics     *prometheus.CounterVec
	processingQueue     prometheus.Gauge

	// Error metrics
	errorTotal      *prometheus.CounterVec
	panicTotal      prometheus.Counter
	timeoutTotal    *prometheus.CounterVec

	// Custom metrics
	customGauge     *prometheus.GaugeVec
	customHistogram *prometheus.HistogramVec
}

// NewMetricsCollector creates and registers all Prometheus metrics
func NewMetricsCollector() *MetricsCollector {
	collector := &MetricsCollector{
		// HTTP metrics
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "endpoint"},
		),
		httpRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),
		httpResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint", "status_code"},
		),

		// Application metrics
		activeConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_connections",
				Help: "Number of active connections",
			},
		),
		databaseConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "database_connections",
				Help: "Number of database connections",
			},
			[]string{"state", "database"},
		),
		cacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_operations_total",
				Help: "Total cache operations",
			},
			[]string{"operation", "result"},
		),
		businessMetrics: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "business_events_total",
				Help: "Total business events",
			},
			[]string{"event_type", "status"},
		),
		processingQueue: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "processing_queue_size",
				Help: "Number of items in processing queue",
			},
		),

		// Error metrics
		errorTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "errors_total",
				Help: "Total number of errors",
			},
			[]string{"type", "component", "severity"},
		),
		panicTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "panics_total",
				Help: "Total number of panics",
			},
		),
		timeoutTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "timeouts_total",
				Help: "Total number of timeouts",
			},
			[]string{"operation", "component"},
		),

		// Custom metrics
		customGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "bolt_demo_custom_gauge",
				Help: "Custom gauge metric for demonstration",
			},
			[]string{"label1", "label2"},
		),
		customHistogram: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "bolt_demo_custom_duration_seconds",
				Help:    "Custom duration histogram for demonstration",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "component"},
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		collector.httpRequestsTotal,
		collector.httpRequestDuration,
		collector.httpRequestSize,
		collector.httpResponseSize,
		collector.activeConnections,
		collector.databaseConnections,
		collector.cacheHits,
		collector.businessMetrics,
		collector.processingQueue,
		collector.errorTotal,
		collector.panicTotal,
		collector.timeoutTotal,
		collector.customGauge,
		collector.customHistogram,
	)

	return collector
}

// Application represents the main application
type Application struct {
	logger  *bolt.Logger
	metrics *MetricsCollector
}

// NewApplication creates a new application instance
func NewApplication() *Application {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		With().
		Str("service", "prometheus-demo").
		Str("version", "v1.0.0").
		Logger()

	metrics := NewMetricsCollector()

	return &Application{
		logger:  logger,
		metrics: metrics,
	}
}

// Middleware for metrics collection
func (app *Application) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		correlationID := getOrCreateCorrelationID(r)

		// Increment active connections
		app.metrics.activeConnections.Inc()
		defer app.metrics.activeConnections.Dec()

		// Record request size
		if r.ContentLength > 0 {
			app.metrics.httpRequestSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(r.ContentLength))
		}

		// Wrap response writer to capture response size and status
		wrapper := &responseWriterWithMetrics{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		app.logger.Info().
			Str("correlation_id", correlationID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Int64("request_size", r.ContentLength).
			Msg("HTTP request started")

		// Process request
		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)

		// Record metrics
		app.metrics.httpRequestsTotal.WithLabelValues(
			r.Method,
			r.URL.Path,
			strconv.Itoa(wrapper.statusCode),
		).Inc()

		app.metrics.httpRequestDuration.WithLabelValues(
			r.Method,
			r.URL.Path,
		).Observe(duration.Seconds())

		app.metrics.httpResponseSize.WithLabelValues(
			r.Method,
			r.URL.Path,
			strconv.Itoa(wrapper.statusCode),
		).Observe(float64(wrapper.responseSize))

		// Log completion with metrics correlation
		logEvent := app.logger.Info()
		if wrapper.statusCode >= 400 {
			logEvent = app.logger.Warn()
		}
		if wrapper.statusCode >= 500 {
			logEvent = app.logger.Error()
		}

		logEvent.
			Str("correlation_id", correlationID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status_code", wrapper.statusCode).
			Dur("duration", duration).
			Float64("duration_seconds", duration.Seconds()).
			Int("response_size", wrapper.responseSize).
			Int64("request_size", r.ContentLength).
			Msg("HTTP request completed")
	})
}

type responseWriterWithMetrics struct {
	http.ResponseWriter
	statusCode   int
	responseSize int
}

func (rw *responseWriterWithMetrics) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterWithMetrics) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.responseSize += size
	return size, err
}

// HTTP Handlers with metrics and logging

func (app *Application) rootHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)

	// Business metrics
	app.metrics.businessMetrics.WithLabelValues("page_view", "success").Inc()

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("operation", "root_handler").
		Msg("Processing root request")

	response := fmt.Sprintf(`{
		"message": "Bolt Prometheus Demo",
		"correlation_id": "%s",
		"timestamp": "%s",
		"metrics_available": "/metrics"
	}`, correlationID, time.Now().UTC().Format(time.RFC3339))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Correlation-ID", correlationID)
	fmt.Fprint(w, response)
}

func (app *Application) usersHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)
	start := time.Now()

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("operation", "list_users").
		Msg("Processing users request")

	// Simulate database operation with metrics
	users, err := app.fetchUsersWithMetrics(r.Context(), correlationID)
	if err != nil {
		// Record error metrics
		app.metrics.errorTotal.WithLabelValues("database", "user_service", "high").Inc()

		app.logger.Error().
			Str("correlation_id", correlationID).
			Err(err).
			Msg("Failed to fetch users")

		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Record business metrics
	app.metrics.businessMetrics.WithLabelValues("users_listed", "success").Inc()
	app.metrics.customGauge.WithLabelValues("users", "active").Set(float64(len(users)))

	duration := time.Since(start)
	app.metrics.customHistogram.WithLabelValues("fetch_users", "database").Observe(duration.Seconds())

	app.logger.Info().
		Str("correlation_id", correlationID).
		Int("users_count", len(users)).
		Dur("fetch_duration", duration).
		Float64("fetch_duration_seconds", duration.Seconds()).
		Msg("Users fetched successfully")

	response := fmt.Sprintf(`{
		"users": [
			{"id": 1, "name": "John Doe", "email": "john@example.com"},
			{"id": 2, "name": "Jane Smith", "email": "jane@example.com"}
		],
		"total": %d,
		"correlation_id": "%s",
		"processing_time_seconds": %.6f
	}`, len(users), correlationID, duration.Seconds())

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Correlation-ID", correlationID)
	fmt.Fprint(w, response)
}

func (app *Application) cacheHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)

	// Simulate cache operations with metrics
	cacheKey := r.URL.Query().Get("key")
	if cacheKey == "" {
		cacheKey = "default"
	}

	// Simulate cache hit/miss (70% hit rate)
	isHit := rand.Intn(10) < 7
	
	if isHit {
		app.metrics.cacheHits.WithLabelValues("get", "hit").Inc()
		app.logger.Info().
			Str("correlation_id", correlationID).
			Str("cache_key", cacheKey).
			Str("cache_result", "hit").
			Msg("Cache hit")
	} else {
		app.metrics.cacheHits.WithLabelValues("get", "miss").Inc()
		app.logger.Info().
			Str("correlation_id", correlationID).
			Str("cache_key", cacheKey).
			Str("cache_result", "miss").
			Msg("Cache miss")
	}

	response := fmt.Sprintf(`{
		"cache_key": "%s",
		"result": "%s",
		"correlation_id": "%s",
		"timestamp": "%s"
	}`, cacheKey, 
		map[bool]string{true: "hit", false: "miss"}[isHit],
		correlationID, 
		time.Now().UTC().Format(time.RFC3339))

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, response)
}

func (app *Application) errorHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)

	// Record error metrics
	app.metrics.errorTotal.WithLabelValues("simulation", "error_handler", "medium").Inc()

	app.logger.Error().
		Str("correlation_id", correlationID).
		Str("error_type", "simulated").
		Msg("Simulated error for metrics demonstration")

	http.Error(w, "Simulated Error", http.StatusInternalServerError)
}

func (app *Application) panicHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)

	// This will be caught by panic recovery middleware
	app.logger.Warn().
		Str("correlation_id", correlationID).
		Msg("About to trigger panic for metrics demonstration")

	panic("simulated panic for metrics demonstration")
}

func (app *Application) slowHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)
	start := time.Now()

	// Simulate slow operation
	delay := time.Duration(rand.Intn(2000)+500) * time.Millisecond
	
	app.logger.Info().
		Str("correlation_id", correlationID).
		Dur("simulated_delay", delay).
		Msg("Starting slow operation")

	time.Sleep(delay)

	// Record custom metrics for slow operations
	app.metrics.customHistogram.WithLabelValues("slow_operation", "handler").Observe(delay.Seconds())

	duration := time.Since(start)
	app.logger.Info().
		Str("correlation_id", correlationID).
		Dur("total_duration", duration).
		Float64("duration_seconds", duration.Seconds()).
		Msg("Slow operation completed")

	response := fmt.Sprintf(`{
		"message": "Slow operation completed",
		"delay_seconds": %.3f,
		"correlation_id": "%s",
		"timestamp": "%s"
	}`, delay.Seconds(), correlationID, time.Now().UTC().Format(time.RFC3339))

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, response)
}

// Panic recovery middleware with metrics
func (app *Application) panicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				correlationID := getOrCreateCorrelationID(r)

				// Record panic metrics
				app.metrics.panicTotal.Inc()
				app.metrics.errorTotal.WithLabelValues("panic", "middleware", "critical").Inc()

				app.logger.Error().
					Str("correlation_id", correlationID).
					Interface("panic", err).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Msg("Panic recovered")

				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// Database simulation with metrics
func (app *Application) fetchUsersWithMetrics(ctx context.Context, correlationID string) ([]map[string]interface{}, error) {
	start := time.Now()

	// Simulate database connection metrics
	app.metrics.databaseConnections.WithLabelValues("active", "users_db").Set(5)
	app.metrics.databaseConnections.WithLabelValues("idle", "users_db").Set(10)

	// Simulate query processing
	app.metrics.processingQueue.Set(3) // Items in queue
	defer app.metrics.processingQueue.Set(0) // Queue processed

	// Simulate database operation time
	queryTime := time.Duration(rand.Intn(100)) * time.Millisecond
	time.Sleep(queryTime)

	// Simulate occasional database errors (5% chance)
	if rand.Intn(20) == 0 {
		app.metrics.errorTotal.WithLabelValues("database", "query", "high").Inc()
		app.metrics.timeoutTotal.WithLabelValues("database_query", "users_service").Inc()
		
		return nil, fmt.Errorf("database timeout after %v", queryTime)
	}

	// Record successful database metrics
	app.metrics.customHistogram.WithLabelValues("database_query", "users").Observe(time.Since(start).Seconds())

	users := []map[string]interface{}{
		{"id": 1, "name": "John Doe", "email": "john@example.com"},
		{"id": 2, "name": "Jane Smith", "email": "jane@example.com"},
	}

	app.logger.Debug().
		Str("correlation_id", correlationID).
		Int("users_returned", len(users)).
		Dur("query_duration", time.Since(start)).
		Msg("Database query completed")

	return users, nil
}

// Background metrics updater
func (app *Application) startMetricsUpdater() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			// Update some example metrics periodically
			app.metrics.customGauge.WithLabelValues("system", "cpu").Set(rand.Float64() * 100)
			app.metrics.customGauge.WithLabelValues("system", "memory").Set(rand.Float64() * 100)
			app.metrics.processingQueue.Set(float64(rand.Intn(10)))

			app.logger.Debug().
				Float64("cpu_usage", rand.Float64()*100).
				Float64("memory_usage", rand.Float64()*100).
				Int("queue_size", rand.Intn(10)).
				Msg("Metrics updated")
		}
	}()
}

// Health check with metrics
func (app *Application) healthHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)

	// Health check metrics
	app.metrics.businessMetrics.WithLabelValues("health_check", "success").Inc()

	app.logger.Info().
		Str("correlation_id", correlationID).
		Msg("Health check performed")

	response := fmt.Sprintf(`{
		"status": "healthy",
		"timestamp": "%s",
		"correlation_id": "%s",
		"metrics_endpoint": "/metrics"
	}`, time.Now().UTC().Format(time.RFC3339), correlationID)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, response)
}

// Utility functions
func getOrCreateCorrelationID(r *http.Request) string {
	if correlationID := r.Header.Get("X-Correlation-ID"); correlationID != "" {
		return correlationID
	}
	return uuid.New().String()
}

func main() {
	app := NewApplication()

	// Start background metrics updater
	app.startMetricsUpdater()

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Wrap all handlers with middleware
	handler := app.panicRecoveryMiddleware(app.metricsMiddleware(mux))

	// Register handlers
	mux.HandleFunc("/", app.rootHandler)
	mux.HandleFunc("/users", app.usersHandler)
	mux.HandleFunc("/cache", app.cacheHandler)
	mux.HandleFunc("/error", app.errorHandler)
	mux.HandleFunc("/panic", app.panicHandler)
	mux.HandleFunc("/slow", app.slowHandler)
	mux.HandleFunc("/health", app.healthHandler)

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	port := getEnv("PORT", "8080")

	app.logger.Info().
		Str("port", port).
		Str("metrics_endpoint", "/metrics").
		Msg("Starting Prometheus demo server")

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		app.logger.Fatal().Err(err).Msg("Server failed to start")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}