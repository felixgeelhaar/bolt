// Package main implements a Kubernetes-ready application with Bolt logging.
// This example demonstrates production deployment patterns including health checks,
// graceful shutdown, configuration management, and observability integration.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/felixgeelhaar/bolt"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Application represents the main application
type Application struct {
	logger bolt.Logger
	config *Config
	server *http.Server
}

// Config holds application configuration
type Config struct {
	Port        string
	MetricsPort string
	Environment string
	LogLevel    string
	PodName     string
	PodIP       string
	NodeName    string
	Namespace   string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		MetricsPort: getEnv("METRICS_PORT", "8081"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		PodName:     getEnv("POD_NAME", "unknown"),
		PodIP:       getEnv("POD_IP", "unknown"),
		NodeName:    getEnv("NODE_NAME", "unknown"),
		Namespace:   getEnv("POD_NAMESPACE", "default"),
	}
}

// NewApplication creates a new application instance
func NewApplication() *Application {
	config := LoadConfig()

	// Configure structured logging with Kubernetes context
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		Level(parseLogLevel(config.LogLevel)).
		With().
		Str("service", "bolt-demo-app").
		Str("version", "v1.0.0").
		Str("environment", config.Environment).
		Str("pod_name", config.PodName).
		Str("pod_ip", config.PodIP).
		Str("node_name", config.NodeName).
		Str("namespace", config.Namespace).
		Logger()

	return &Application{
		logger: logger,
		config: config,
	}
}

// Health check handlers
func (app *Application) healthLiveHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := r.Header.Get("X-Correlation-ID")
	if correlationID == "" {
		correlationID = uuid.New().String()
	}

	app.logger.Debug().
		Str("correlation_id", correlationID).
		Str("endpoint", "/health/live").
		Str("remote_addr", r.RemoteAddr).
		Msg("Liveness probe check")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"alive","timestamp":"%s","pod":"%s"}`, 
		time.Now().UTC().Format(time.RFC3339), app.config.PodName)
}

func (app *Application) healthReadyHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := r.Header.Get("X-Correlation-ID")
	if correlationID == "" {
		correlationID = uuid.New().String()
	}

	// Simulate readiness checks (database, cache, external services)
	ready := app.checkReadiness()

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("endpoint", "/health/ready").
		Bool("ready", ready).
		Str("pod_name", app.config.PodName).
		Msg("Readiness probe check")

	w.Header().Set("Content-Type", "application/json")
	
	if ready {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ready","timestamp":"%s","pod":"%s"}`, 
			time.Now().UTC().Format(time.RFC3339), app.config.PodName)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"not_ready","timestamp":"%s","pod":"%s"}`, 
			time.Now().UTC().Format(time.RFC3339), app.config.PodName)
	}
}

func (app *Application) healthStartupHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := r.Header.Get("X-Correlation-ID")
	if correlationID == "" {
		correlationID = uuid.New().String()
	}

	// Simulate startup checks
	started := app.checkStartup()

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("endpoint", "/health/startup").
		Bool("started", started).
		Str("pod_name", app.config.PodName).
		Msg("Startup probe check")

	w.Header().Set("Content-Type", "application/json")
	
	if started {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"started","timestamp":"%s","pod":"%s"}`, 
			time.Now().UTC().Format(time.RFC3339), app.config.PodName)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"starting","timestamp":"%s","pod":"%s"}`, 
			time.Now().UTC().Format(time.RFC3339), app.config.PodName)
	}
}

// Application handlers
func (app *Application) rootHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := r.Header.Get("X-Correlation-ID")
	if correlationID == "" {
		correlationID = uuid.New().String()
	}
	w.Header().Set("X-Correlation-ID", correlationID)

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("user_agent", r.UserAgent()).
		Str("pod_name", app.config.PodName).
		Msg("Request received")

	response := map[string]interface{}{
		"message":        "Bolt Kubernetes Demo Application",
		"version":        "v1.0.0",
		"environment":    app.config.Environment,
		"pod_name":       app.config.PodName,
		"pod_ip":         app.config.PodIP,
		"node_name":      app.config.NodeName,
		"namespace":      app.config.Namespace,
		"correlation_id": correlationID,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	// Simulate JSON response
	fmt.Fprintf(w, `{
		"message": "%s",
		"version": "%s",
		"environment": "%s",
		"pod_name": "%s",
		"pod_ip": "%s",
		"node_name": "%s",
		"namespace": "%s",
		"correlation_id": "%s",
		"timestamp": "%s"
	}`,
		response["message"],
		response["version"],
		response["environment"],
		response["pod_name"],
		response["pod_ip"],
		response["node_name"],
		response["namespace"],
		response["correlation_id"],
		response["timestamp"],
	)
}

func (app *Application) apiUsersHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	correlationID := r.Header.Get("X-Correlation-ID")
	if correlationID == "" {
		correlationID = uuid.New().String()
	}
	w.Header().Set("X-Correlation-ID", correlationID)

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("operation", "list_users").
		Str("pod_name", app.config.PodName).
		Msg("Processing user list request")

	// Simulate database query
	time.Sleep(10 * time.Millisecond)
	
	duration := time.Since(start)

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("operation", "list_users").
		Int("users_returned", 25).
		Dur("duration", duration).
		Float64("duration_ms", float64(duration.Nanoseconds())/1_000_000).
		Str("pod_name", app.config.PodName).
		Msg("User list request completed")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"users": [
			{"id": 1, "name": "John Doe", "email": "john@example.com"},
			{"id": 2, "name": "Jane Smith", "email": "jane@example.com"}
		],
		"total": 25,
		"pod_name": "%s",
		"correlation_id": "%s",
		"processing_time_ms": %.3f
	}`, app.config.PodName, correlationID, float64(duration.Nanoseconds())/1_000_000)
}

// Middleware
func (app *Application) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		r.Header.Set("X-Correlation-ID", correlationID)

		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		app.logger.Info().
			Str("correlation_id", correlationID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Str("pod_name", app.config.PodName).
			Msg("HTTP request started")

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)

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
			Float64("duration_ms", float64(duration.Nanoseconds())/1_000_000).
			Int("response_size", wrapper.size).
			Str("pod_name", app.config.PodName).
			Msg("HTTP request completed")
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// Health check implementations
func (app *Application) checkReadiness() bool {
	// Simulate readiness checks
	// - Database connectivity
	// - External service availability
	// - Cache connectivity
	return true
}

func (app *Application) checkStartup() bool {
	// Simulate startup checks
	// - Configuration loaded
	// - Database migrations
	// - Cache warmed up
	return true
}

// Setup HTTP routes
func (app *Application) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/health/live", app.healthLiveHandler)
	mux.HandleFunc("/health/ready", app.healthReadyHandler)
	mux.HandleFunc("/health/startup", app.healthStartupHandler)

	// Application endpoints
	mux.HandleFunc("/", app.rootHandler)
	mux.HandleFunc("/api/users", app.apiUsersHandler)

	return mux
}

// Setup metrics server
func (app *Application) startMetricsServer() {
	// Create Prometheus metrics
	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status", "pod"},
	)
	
	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "pod"},
	)

	prometheus.MustRegister(requestsTotal, requestDuration)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	
	metricsServer := &http.Server{
		Addr:    ":" + app.config.MetricsPort,
		Handler: metricsMux,
	}

	app.logger.Info().
		Str("port", app.config.MetricsPort).
		Str("pod_name", app.config.PodName).
		Msg("Starting metrics server")

	go func() {
		if err := metricsServer.ListenAndServe(); err != http.ErrServerClosed {
			app.logger.Error().
				Err(err).
				Str("port", app.config.MetricsPort).
				Msg("Metrics server failed")
		}
	}()
}

// Start application
func (app *Application) Start() error {
	app.logger.Info().
		Str("version", "v1.0.0").
		Str("environment", app.config.Environment).
		Str("pod_name", app.config.PodName).
		Str("pod_ip", app.config.PodIP).
		Str("node_name", app.config.NodeName).
		Str("namespace", app.config.Namespace).
		Msg("Starting application")

	// Start metrics server
	app.startMetricsServer()

	// Setup main server
	handler := app.loggingMiddleware(app.setupRoutes())
	app.server = &http.Server{
		Addr:           ":" + app.config.Port,
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	app.logger.Info().
		Str("port", app.config.Port).
		Str("pod_name", app.config.PodName).
		Msg("Starting HTTP server")

	return app.server.ListenAndServe()
}

// Graceful shutdown
func (app *Application) Shutdown(ctx context.Context) error {
	app.logger.Info().
		Str("pod_name", app.config.PodName).
		Msg("Initiating graceful shutdown")

	if app.server != nil {
		return app.server.Shutdown(ctx)
	}
	return nil
}

// Utility functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseLogLevel(level string) bolt.Level {
	switch level {
	case "debug":
		return bolt.DebugLevel
	case "info":
		return bolt.InfoLevel
	case "warn":
		return bolt.WarnLevel
	case "error":
		return bolt.ErrorLevel
	default:
		return bolt.InfoLevel
	}
}

func main() {
	app := NewApplication()

	// Handle shutdown gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		app.logger.Info().
			Str("signal", sig.String()).
			Str("pod_name", app.config.PodName).
			Msg("Received shutdown signal")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := app.Shutdown(ctx); err != nil {
			app.logger.Error().
				Err(err).
				Str("pod_name", app.config.PodName).
				Msg("Error during shutdown")
			os.Exit(1)
		}

		app.logger.Info().
			Str("pod_name", app.config.PodName).
			Msg("Application shutdown complete")
		os.Exit(0)
	}()

	// Start application
	if err := app.Start(); err != http.ErrServerClosed {
		app.logger.Fatal().
			Err(err).
			Str("pod_name", app.config.PodName).
			Msg("Application failed to start")
	}
}