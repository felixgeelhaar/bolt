// Package main demonstrates HTTP middleware integration with Bolt logging.
// This example shows how to implement structured logging for HTTP requests
// with correlation IDs, request/response tracing, and performance metrics.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/felixgeelhaar/bolt/v2"
	"github.com/google/uuid"
)

// Service represents a microservice with structured logging
type Service struct {
	logger bolt.Logger
	name   string
}

// NewService creates a new service instance with configured logging
func NewService(name string) *Service {
	// Configure logger for production JSON output
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		Level(bolt.InfoLevel).
		With().
		Str("service", name).
		Str("version", "v1.0.0").
		Str("environment", getEnv("ENVIRONMENT", "development")).
		Logger()

	return &Service{
		logger: logger,
		name:   name,
	}
}

// correlationIDMiddleware adds correlation IDs to requests and responses
func (s *Service) correlationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Get or generate correlation ID
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		
		// Add correlation ID to response headers
		w.Header().Set("X-Correlation-ID", correlationID)
		
		// Create request-scoped context with correlation ID
		ctx := context.WithValue(r.Context(), "correlation_id", correlationID)
		r = r.WithContext(ctx)
		
		// Create response writer wrapper to capture status code
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:    http.StatusOK,
		}
		
		// Log request start
		s.logger.Info().
			Str("correlation_id", correlationID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Int64("content_length", r.ContentLength).
			Msg("HTTP request started")
		
		// Process request
		next.ServeHTTP(wrapper, r)
		
		duration := time.Since(start)
		
		// Log request completion
		logEvent := s.logger.Info()
		if wrapper.statusCode >= 400 {
			logEvent = s.logger.Warn()
		}
		if wrapper.statusCode >= 500 {
			logEvent = s.logger.Error()
		}
		
		logEvent.
			Str("correlation_id", correlationID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status_code", wrapper.statusCode).
			Dur("duration", duration).
			Int("response_size", wrapper.bytesWritten).
			Float64("duration_ms", float64(duration.Nanoseconds())/1_000_000).
			Msg("HTTP request completed")
	})
}

// responseWriterWrapper captures response status code and size
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += size
	return size, err
}

// metricsMiddleware adds performance and business metrics logging
func (s *Service) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Process request
		next.ServeHTTP(w, r)
		
		duration := time.Since(start)
		correlationID := r.Context().Value("correlation_id").(string)
		
		// Log performance metrics
		s.logger.Info().
			Str("correlation_id", correlationID).
			Str("metric_type", "performance").
			Str("endpoint", r.URL.Path).
			Dur("response_time", duration).
			Int64("response_time_ns", duration.Nanoseconds()).
			Float64("response_time_ms", float64(duration.Nanoseconds())/1_000_000).
			Msg("Performance metric recorded")
	})
}

// errorHandlingMiddleware provides centralized error handling and logging
func (s *Service) errorHandlingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				correlationID := r.Context().Value("correlation_id").(string)
				
				// Log panic with full context
				s.logger.Error().
					Str("correlation_id", correlationID).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Interface("panic", err).
					Msg("HTTP request panicked")
				
				// Return 500 error
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		
		next.ServeHTTP(w, r)
	})
}

// Business logic handlers

func (s *Service) healthHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := r.Context().Value("correlation_id").(string)
	
	// Simulate health check logic
	healthy := true
	
	s.logger.Info().
		Str("correlation_id", correlationID).
		Bool("healthy", healthy).
		Msg("Health check performed")
	
	if healthy {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"unhealthy","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))
	}
}

func (s *Service) usersHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := r.Context().Value("correlation_id").(string)
	
	switch r.Method {
	case http.MethodGet:
		s.handleGetUsers(w, r, correlationID)
	case http.MethodPost:
		s.handleCreateUser(w, r, correlationID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Service) handleGetUsers(w http.ResponseWriter, r *http.Request, correlationID string) {
	// Simulate database query
	start := time.Now()
	
	// Simulate some processing time
	time.Sleep(10 * time.Millisecond)
	
	queryDuration := time.Since(start)
	
	s.logger.Info().
		Str("correlation_id", correlationID).
		Str("operation", "get_users").
		Str("query", "SELECT * FROM users").
		Dur("query_duration", queryDuration).
		Int("records_returned", 25).
		Msg("Database query executed")
	
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"users":[{"id":1,"name":"John Doe"},{"id":2,"name":"Jane Smith"}],"total":25,"correlation_id":"%s"}`, correlationID)
}

func (s *Service) handleCreateUser(w http.ResponseWriter, r *http.Request, correlationID string) {
	// Simulate user creation
	start := time.Now()
	
	// Simulate validation and database insert
	time.Sleep(5 * time.Millisecond)
	
	operationDuration := time.Since(start)
	userID := 123
	
	s.logger.Info().
		Str("correlation_id", correlationID).
		Str("operation", "create_user").
		Int("user_id", userID).
		Dur("operation_duration", operationDuration).
		Int64("content_length", r.ContentLength).
		Msg("User created successfully")
	
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id":%d,"status":"created","correlation_id":"%s"}`, userID, correlationID)
}

func (s *Service) ordersHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := r.Context().Value("correlation_id").(string)
	
	// Simulate an error condition
	if r.URL.Query().Get("simulate_error") == "true" {
		s.logger.Error().
			Str("correlation_id", correlationID).
			Str("operation", "get_orders").
			Str("error_type", "database_connection").
			Msg("Failed to connect to database")
		
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"Database connection failed","correlation_id":"%s"}`, correlationID)
		return
	}
	
	// Simulate successful order retrieval
	s.logger.Info().
		Str("correlation_id", correlationID).
		Str("operation", "get_orders").
		Int("orders_count", 42).
		Msg("Orders retrieved successfully")
	
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"orders":[],"total":42,"correlation_id":"%s"}`, correlationID)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Create service instance
	service := NewService("user-api")
	
	// Create HTTP server with middleware chain
	mux := http.NewServeMux()
	
	// Register handlers
	mux.HandleFunc("/health", service.healthHandler)
	mux.HandleFunc("/users", service.usersHandler)
	mux.HandleFunc("/orders", service.ordersHandler)
	
	// Apply middleware chain (order matters: error handling → correlation → metrics → handler)
	handler := service.errorHandlingMiddleware(
		service.correlationIDMiddleware(
			service.metricsMiddleware(mux),
		),
	)
	
	port := getEnv("PORT", "8080")
	
	service.logger.Info().
		Str("port", port).
		Str("service", service.name).
		Msg("Starting HTTP server")
	
	// Configure server
	server := &http.Server{
		Addr:           ":" + port,
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}
	
	// Start server
	if err := server.ListenAndServe(); err != nil {
		service.logger.Fatal().
			Err(err).
			Str("port", port).
			Msg("Failed to start HTTP server")
	}
}