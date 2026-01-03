// REST API Example
//
// This example demonstrates using Bolt in a production-grade REST API with:
// - Structured request/response logging
// - Error handling and tracking
// - Performance metrics
// - Middleware integration
// - OpenTelemetry trace correlation
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/felixgeelhaar/bolt/v3"
	"go.opentelemetry.io/otel/trace"
)

// API represents our REST API server
type API struct {
	logger *bolt.Logger
	server *http.Server
}

// User represents a user in our system
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// LoggingMiddleware logs all HTTP requests with structured data
func (api *API) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Extract trace context if available
		ctx := r.Context()
		span := trace.SpanFromContext(ctx)

		// Create context logger with trace IDs
		var ctxLogger *bolt.Logger
		if span.SpanContext().IsValid() {
			ctxLogger = api.logger.Ctx(ctx)
		} else {
			ctxLogger = api.logger
		}

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		// Log request
		ctxLogger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Msg("request started")

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log response
		duration := time.Since(start)
		logger := ctxLogger.Info()
		if wrapped.statusCode >= 500 {
			logger = ctxLogger.Error()
		} else if wrapped.statusCode >= 400 {
			logger = ctxLogger.Warn()
		}

		logger.
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", wrapped.statusCode).
			Dur("duration", duration).
			Int("bytes", wrapped.bytesWritten).
			Msg("request completed")
	})
}

// RecoveryMiddleware recovers from panics and logs them
func (api *API) RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				api.logger.Error().
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Str("panic", fmt.Sprintf("%v", err)).
					Msg("panic recovered")

				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(ErrorResponse{
					Error:   "internal_server_error",
					Code:    "INTERNAL_ERROR",
					Message: "An internal error occurred",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// GetUserHandler handles GET /users/:id
func (api *API) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Path[len("/users/"):]

	// Simulate database lookup
	user, err := api.getUserFromDB(userID)
	if err != nil {
		api.logger.Warn().
			Str("user_id", userID).
			Str("error", err.Error()).
			Msg("user not found")

		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "not_found",
			Code:    "USER_NOT_FOUND",
			Message: fmt.Sprintf("User %s not found", userID),
		})
		return
	}

	api.logger.Info().
		Str("user_id", user.ID).
		Str("user_email", user.Email).
		Msg("user retrieved")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// CreateUserHandler handles POST /users
func (api *API) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		api.logger.Warn().
			Str("error", err.Error()).
			Msg("invalid request body")

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "bad_request",
			Code:    "INVALID_JSON",
			Message: "Invalid JSON in request body",
		})
		return
	}

	// Validate user
	if user.Email == "" || user.Name == "" {
		api.logger.Warn().
			Str("email", user.Email).
			Str("name", user.Name).
			Msg("validation failed")

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "validation_error",
			Code:    "INVALID_INPUT",
			Message: "Email and name are required",
		})
		return
	}

	// Create user
	user.ID = fmt.Sprintf("user_%d", time.Now().Unix())
	user.CreatedAt = time.Now()

	api.logger.Info().
		Str("user_id", user.ID).
		Str("user_email", user.Email).
		Str("user_name", user.Name).
		Msg("user created")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// HealthHandler handles GET /health
func (api *API) HealthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// MetricsHandler handles GET /metrics (simple metrics endpoint)
func (api *API) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"uptime_seconds": time.Since(time.Now().Add(-5 * time.Minute)).Seconds(),
		"requests_total": 1234,
		"errors_total":   42,
	}

	api.logger.Debug().
		Float64("uptime", metrics["uptime_seconds"].(float64)).
		Msg("metrics requested")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// getUserFromDB simulates database lookup
func (api *API) getUserFromDB(id string) (*User, error) {
	// Simulate DB latency
	time.Sleep(10 * time.Millisecond)

	if id == "123" {
		return &User{
			ID:        "123",
			Email:     "john@example.com",
			Name:      "John Doe",
			CreatedAt: time.Now().Add(-24 * time.Hour),
		}, nil
	}

	return nil, fmt.Errorf("user not found")
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

func main() {
	// Initialize logger
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Str("service", "rest-api").
		Str("version", "1.0.0").
		Msg("starting REST API server")

	// Create API instance
	api := &API{
		logger: logger,
	}

	// Setup routes with middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/health", api.HealthHandler)
	mux.HandleFunc("/metrics", api.MetricsHandler)
	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			api.GetUserHandler(w, r)
		} else if r.Method == http.MethodPost && r.URL.Path == "/users" {
			api.CreateUserHandler(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Wrap with middleware
	handler := api.RecoveryMiddleware(api.LoggingMiddleware(mux))

	// Create server
	api.server = &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info().
			Str("addr", api.server.Addr).
			Msg("server listening")

		if err := api.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().
				Str("error", err.Error()).
				Msg("server error")
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("shutting down server")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := api.server.Shutdown(ctx); err != nil {
		logger.Error().
			Str("error", err.Error()).
			Msg("server forced to shutdown")
	}

	logger.Info().Msg("server stopped")
}
