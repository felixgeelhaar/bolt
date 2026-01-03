// Package main demonstrates load balancer integration with Bolt logging.
// This example shows distributed request handling, health checks, circuit breaking,
// and comprehensive logging across multiple service instances.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand" // #nosec G404 - Using weak random for examples is acceptable
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/felixgeelhaar/bolt/v3"
	"github.com/google/uuid"
)

// HealthStatus represents the health state of a backend
type HealthStatus int

const (
	Healthy HealthStatus = iota
	Unhealthy
	Degraded
)

func (hs HealthStatus) String() string {
	switch hs {
	case Healthy:
		return "healthy"
	case Unhealthy:
		return "unhealthy"
	case Degraded:
		return "degraded"
	default:
		return "unknown"
	}
}

// Backend represents a backend server instance
type Backend struct {
	ID           string
	URL          *url.URL
	Proxy        *httputil.ReverseProxy
	Health       HealthStatus
	LastCheck    time.Time
	FailCount    int32
	RequestCount int64
	ResponseTime time.Duration
	mutex        sync.RWMutex
}

// LoadBalancer implements a round-robin load balancer with health checking
type LoadBalancer struct {
	backends []*Backend
	current  uint64
	logger   *bolt.Logger
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewLoadBalancer creates a new load balancer instance
func NewLoadBalancer(backendURLs []string) *LoadBalancer {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		SetLevel(bolt.INFO).
		With().
		Str("service", "load-balancer").
		Str("version", "v1.0.0").
		Str("component", "load_balancer").
		Logger()

	ctx, cancel := context.WithCancel(context.Background())

	lb := &LoadBalancer{
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize backends
	for i, backendURL := range backendURLs {
		parsedURL, err := url.Parse(backendURL)
		if err != nil {
			logger.Error().
				Str("backend_url", backendURL).
				Err(err).
				Msg("Failed to parse backend URL")
			continue
		}

		backendID := fmt.Sprintf("backend-%d", i+1)
		proxy := httputil.NewSingleHostReverseProxy(parsedURL)

		// Customize proxy director for logging
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)

			correlationID := req.Header.Get("X-Correlation-ID")
			if correlationID == "" {
				correlationID = uuid.New().String()
				req.Header.Set("X-Correlation-ID", correlationID)
			}

			req.Header.Set("X-Backend-ID", backendID)
			req.Header.Set("X-Load-Balancer", "bolt-lb-v1.0.0")

			logger.Debug().
				Str("correlation_id", correlationID).
				Str("backend_id", backendID).
				Str("backend_url", parsedURL.String()).
				Str("original_url", req.URL.String()).
				Msg("Request proxied to backend")
		}

		// Customize error handler
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			correlationID := r.Header.Get("X-Correlation-ID")
			backendID := r.Header.Get("X-Backend-ID")

			logger.Error().
				Str("correlation_id", correlationID).
				Str("backend_id", backendID).
				Str("backend_url", parsedURL.String()).
				Err(err).
				Msg("Backend request failed")

			// Mark backend as unhealthy
			lb.markBackendUnhealthy(backendID)

			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		}

		backend := Backend{
			ID:        backendID,
			URL:       parsedURL,
			Proxy:     proxy,
			Health:    Healthy,
			LastCheck: time.Now(),
		}

		lb.backends = append(lb.backends, &backend)

		logger.Info().
			Str("backend_id", backendID).
			Str("backend_url", parsedURL.String()).
			Msg("Backend registered")
	}

	// Start health checking
	go lb.startHealthChecking()

	return lb
}

// ServeHTTP implements the http.Handler interface
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	correlationID := getOrCreateCorrelationID(r)

	// Get next healthy backend
	backend := lb.getNextHealthyBackend()
	if backend == nil {
		lb.logger.Error().
			Str("correlation_id", correlationID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Msg("No healthy backends available")

		http.Error(w, "Service Unavailable - No Healthy Backends", http.StatusServiceUnavailable)
		return
	}

	// Log request routing
	lb.logger.Info().
		Str("correlation_id", correlationID).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("backend_id", backend.ID).
		Str("backend_url", backend.URL.String()).
		Str("client_ip", getClientIP(r)).
		Msg("Routing request to backend")

	// Increment backend request count
	atomic.AddInt64(&backend.RequestCount, 1)

	// Add response headers before proxying
	w.Header().Set("X-Load-Balancer", "bolt-lb-v1.0.0")
	w.Header().Set("X-Backend-ID", backend.ID)
	w.Header().Set("X-Correlation-ID", correlationID)

	// Proxy the request
	backend.Proxy.ServeHTTP(w, r)

	duration := time.Since(start)

	// Update backend response time
	backend.mutex.Lock()
	backend.ResponseTime = duration
	backend.mutex.Unlock()

	// Log request completion
	lb.logger.Info().
		Str("correlation_id", correlationID).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("backend_id", backend.ID).
		Dur("duration", duration).
		Float64("duration_ms", float64(duration.Nanoseconds())/1_000_000).
		Int64("backend_request_count", atomic.LoadInt64(&backend.RequestCount)).
		Msg("Request completed")
}

// getNextHealthyBackend returns the next healthy backend using round-robin
func (lb *LoadBalancer) getNextHealthyBackend() *Backend {
	attempts := len(lb.backends)

	for i := 0; i < attempts; i++ {
		idx := atomic.AddUint64(&lb.current, 1) % uint64(len(lb.backends))
		backend := lb.backends[idx]

		backend.mutex.RLock()
		isHealthy := backend.Health == Healthy
		backend.mutex.RUnlock()

		if isHealthy {
			return backend
		}
	}

	// No healthy backends found
	return nil
}

// markBackendUnhealthy marks a backend as unhealthy
func (lb *LoadBalancer) markBackendUnhealthy(backendID string) {
	for i := range lb.backends {
		if lb.backends[i].ID == backendID {
			lb.backends[i].mutex.Lock()
			lb.backends[i].Health = Unhealthy
			lb.backends[i].FailCount++
			lb.backends[i].mutex.Unlock()

			lb.logger.Warn().
				Str("backend_id", backendID).
				Str("backend_url", lb.backends[i].URL.String()).
				Int("fail_count", int(lb.backends[i].FailCount)).
				Msg("Backend marked as unhealthy")

			break
		}
	}
}

// startHealthChecking starts the health checking goroutine
func (lb *LoadBalancer) startHealthChecking() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	lb.logger.Info().Msg("Starting backend health checking")

	for {
		select {
		case <-lb.ctx.Done():
			lb.logger.Info().Msg("Health checking stopped")
			return
		case <-ticker.C:
			lb.performHealthChecks()
		}
	}
}

// performHealthChecks checks the health of all backends
func (lb *LoadBalancer) performHealthChecks() {
	for i := range lb.backends {
		go lb.checkBackendHealth(lb.backends[i])
	}
}

// checkBackendHealth performs a health check on a single backend
func (lb *LoadBalancer) checkBackendHealth(backend *Backend) {
	start := time.Now()
	healthCheckID := uuid.New().String()

	lb.logger.Debug().
		Str("health_check_id", healthCheckID).
		Str("backend_id", backend.ID).
		Str("backend_url", backend.URL.String()).
		Msg("Starting backend health check")

	// Create health check request
	healthURL := backend.URL.String() + "/health"
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		lb.logger.Error().
			Str("health_check_id", healthCheckID).
			Str("backend_id", backend.ID).
			Err(err).
			Msg("Failed to create health check request")
		return
	}

	req.Header.Set("User-Agent", "BoltLoadBalancer/1.0.0")
	req.Header.Set("X-Health-Check", "true")

	resp, err := client.Do(req)
	duration := time.Since(start)

	backend.mutex.Lock()
	defer backend.mutex.Unlock()

	backend.LastCheck = time.Now()
	previousHealth := backend.Health

	if err != nil {
		backend.Health = Unhealthy
		backend.FailCount++

		lb.logger.Warn().
			Str("health_check_id", healthCheckID).
			Str("backend_id", backend.ID).
			Str("backend_url", backend.URL.String()).
			Dur("duration", duration).
			Err(err).
			Int("fail_count", int(backend.FailCount)).
			Msg("Backend health check failed")

		return
	}

	defer resp.Body.Close()

	// Determine health based on status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if backend.Health == Unhealthy {
			lb.logger.Info().
				Str("health_check_id", healthCheckID).
				Str("backend_id", backend.ID).
				Str("backend_url", backend.URL.String()).
				Int("status_code", resp.StatusCode).
				Msg("Backend recovered - marking as healthy")
		}
		backend.Health = Healthy
		backend.FailCount = 0
	} else if resp.StatusCode >= 500 {
		backend.Health = Unhealthy
		backend.FailCount++
	} else {
		backend.Health = Degraded
	}

	// Log health check result
	logEvent := lb.logger.Debug()
	if previousHealth != backend.Health {
		logEvent = lb.logger.Info()
	}

	logEvent.
		Str("health_check_id", healthCheckID).
		Str("backend_id", backend.ID).
		Str("backend_url", backend.URL.String()).
		Int("status_code", resp.StatusCode).
		Dur("duration", duration).
		Float64("duration_ms", float64(duration.Nanoseconds())/1_000_000).
		Str("health_status", backend.Health.String()).
		Str("previous_health", previousHealth.String()).
		Int("fail_count", int(backend.FailCount)).
		Msg("Backend health check completed")
}

// Stats provides load balancer statistics
func (lb *LoadBalancer) Stats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_backends": len(lb.backends),
		"backends":       make([]map[string]interface{}, len(lb.backends)),
	}

	healthyCount := 0
	unhealthyCount := 0
	degradedCount := 0

	for i, backend := range lb.backends {
		backend.mutex.RLock()
		backendStats := map[string]interface{}{
			"id":            backend.ID,
			"url":           backend.URL.String(),
			"health_status": backend.Health.String(),
			"last_check":    backend.LastCheck.Format(time.RFC3339),
			"fail_count":    backend.FailCount,
			"request_count": atomic.LoadInt64(&backend.RequestCount),
			"response_time": backend.ResponseTime.String(),
		}
		backend.mutex.RUnlock()

		stats["backends"].([]map[string]interface{})[i] = backendStats

		switch backend.Health {
		case Healthy:
			healthyCount++
		case Unhealthy:
			unhealthyCount++
		case Degraded:
			degradedCount++
		}
	}

	stats["healthy_count"] = healthyCount
	stats["unhealthy_count"] = unhealthyCount
	stats["degraded_count"] = degradedCount

	return stats
}

// Application represents the main load balancer application
type Application struct {
	loadBalancer *LoadBalancer
	logger       *bolt.Logger
}

// NewApplication creates a new load balancer application
func NewApplication(backendURLs []string) *Application {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		SetLevel(bolt.INFO).
		With().
		Str("service", "load-balancer-app").
		Str("version", "v1.0.0").
		Logger()

	loadBalancer := NewLoadBalancer(backendURLs)

	return &Application{
		loadBalancer: loadBalancer,
		logger:       logger,
	}
}

// statsHandler provides load balancer statistics
func (app *Application) statsHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)

	stats := app.loadBalancer.Stats()

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("operation", "get_stats").
		Interface("stats", stats).
		Msg("Load balancer statistics requested")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Correlation-ID", correlationID)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{ // #nosec G104 - Example code
		"load_balancer_stats": stats,
		"timestamp":           time.Now().UTC().Format(time.RFC3339),
		"correlation_id":      correlationID,
	})
}

// healthHandler provides load balancer health check
func (app *Application) healthHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)

	stats := app.loadBalancer.Stats()
	healthyCount := stats["healthy_count"].(int)
	totalCount := stats["total_backends"].(int)

	isHealthy := healthyCount > 0
	status := "unhealthy"
	statusCode := http.StatusServiceUnavailable

	if isHealthy {
		status = "healthy"
		statusCode = http.StatusOK
	}

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("operation", "health_check").
		Bool("is_healthy", isHealthy).
		Int("healthy_backends", healthyCount).
		Int("total_backends", totalCount).
		Msg("Load balancer health check")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Correlation-ID", correlationID)
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           status,
		"healthy_backends": healthyCount,
		"total_backends":   totalCount,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
		"correlation_id":   correlationID,
	}); err != nil {
		app.logger.Error().Err(err).Msg("Failed to encode health response")
	}
}

// Utility functions
func getOrCreateCorrelationID(r *http.Request) string {
	if correlationID := r.Header.Get("X-Correlation-ID"); correlationID != "" {
		return correlationID
	}
	return uuid.New().String()
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// runBackend starts a simple backend server for testing
func runBackend(port int) {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		SetLevel(bolt.INFO).
		With().
		Str("service", "backend-server").
		Str("backend_id", fmt.Sprintf("backend-%d", port)).
		Int("port", port).
		Logger()

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		correlationID := getOrCreateCorrelationID(r)

		// Simulate occasional health check failures (5% chance)
		if rand.Intn(20) == 0 {
			logger.Warn().
				Str("correlation_id", correlationID).
				Int("port", port).
				Msg("Simulated health check failure")
			http.Error(w, "Health Check Failed", http.StatusInternalServerError)
			return
		}

		logger.Debug().
			Str("correlation_id", correlationID).
			Int("port", port).
			Msg("Health check OK")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"healthy","backend":"backend-%d","timestamp":"%s"}`,
			port, time.Now().UTC().Format(time.RFC3339))
	})

	// Main endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		correlationID := getOrCreateCorrelationID(r)
		backendID := r.Header.Get("X-Backend-ID")

		// Simulate processing time
		processingTime := time.Duration(rand.Intn(100)) * time.Millisecond
		time.Sleep(processingTime)

		logger.Info().
			Str("correlation_id", correlationID).
			Str("backend_id", backendID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Dur("processing_time", processingTime).
			Msg("Request processed by backend")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"message": "Hello from backend",
			"backend_id": "backend-%d",
			"port": %d,
			"processing_time_ms": %.3f,
			"correlation_id": "%s",
			"timestamp": "%s"
		}`, port, port, float64(processingTime.Nanoseconds())/1_000_000,
			correlationID, time.Now().UTC().Format(time.RFC3339))
	})

	addr := fmt.Sprintf(":%d", port)
	logger.Info().
		Str("address", addr).
		Msg("Starting backend server")

	// #nosec G114 - Example code, timeout handling demonstrated elsewhere
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Fatal().Err(err).Msg("Backend server failed")
	}
}

func main() {
	// Check if we should run as a backend server
	if len(os.Args) > 1 && os.Args[1] == "backend" {
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go backend <port>")
			os.Exit(1)
		}
		port, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("Invalid port: %v\n", err)
			os.Exit(1)
		}
		runBackend(port)
		return
	}

	// Run as load balancer
	backendURLs := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}

	// Override with environment variable if provided
	if backends := os.Getenv("BACKEND_URLS"); backends != "" {
		backendURLs = []string{backends}
		// Simple split by comma
		backendURLs = []string{}
		for _, url := range []string{backends} {
			backendURLs = append(backendURLs, url)
		}
	}

	app := NewApplication(backendURLs)

	// Setup routes
	mux := http.NewServeMux()

	// Load balancer management endpoints
	mux.HandleFunc("/lb/stats", app.statsHandler)
	mux.HandleFunc("/lb/health", app.healthHandler)

	// Default handler - proxy to backends
	mux.Handle("/", app.loadBalancer)

	port := getEnv("PORT", "8080")

	app.logger.Info().
		Str("port", port).
		Interface("backend_urls", backendURLs).
		Msg("Starting load balancer")

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		app.logger.Fatal().Err(err).Msg("Load balancer failed to start")
	}
}
