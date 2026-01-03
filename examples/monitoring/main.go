// Monitoring Integration Example
//
// This example demonstrates integrating Bolt with popular monitoring platforms:
// - Prometheus metrics
// - Grafana dashboards
// - DataDog integration
// - New Relic integration
// - Custom metrics exporters
package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/felixgeelhaar/bolt/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsCollector collects application metrics
type MetricsCollector struct {
	// Log metrics
	logCounter *prometheus.CounterVec
	logLatency *prometheus.HistogramVec

	// Application metrics
	requestCounter  *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	activeRequests  prometheus.Gauge
	errorCounter    *prometheus.CounterVec

	// Business metrics
	businessEvents *prometheus.CounterVec
	queueDepth     prometheus.Gauge
	cacheHitRate   *prometheus.CounterVec
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		logCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "bolt_logs_total",
				Help: "Total number of log messages by level",
			},
			[]string{"level"},
		),
		logLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "bolt_log_latency_seconds",
				Help:    "Log operation latency in seconds",
				Buckets: prometheus.ExponentialBuckets(0.000001, 2, 10), // 1µs to 1ms
			},
			[]string{"level"},
		),
		requestCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "app_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "app_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		activeRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "app_active_requests",
				Help: "Number of active HTTP requests",
			},
		),
		errorCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "app_errors_total",
				Help: "Total number of errors by type",
			},
			[]string{"type", "severity"},
		),
		businessEvents: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "app_business_events_total",
				Help: "Total number of business events",
			},
			[]string{"event_type", "status"},
		),
		queueDepth: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "app_queue_depth",
				Help: "Current queue depth",
			},
		),
		cacheHitRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "app_cache_operations_total",
				Help: "Cache operations by result",
			},
			[]string{"result"}, // hit or miss
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		mc.logCounter,
		mc.logLatency,
		mc.requestCounter,
		mc.requestDuration,
		mc.activeRequests,
		mc.errorCounter,
		mc.businessEvents,
		mc.queueDepth,
		mc.cacheHitRate,
	)

	return mc
}

// LoggingMiddleware wraps HTTP handler with logging and metrics
func LoggingMiddleware(logger *bolt.Logger, metrics *MetricsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Track active requests
			metrics.activeRequests.Inc()
			defer metrics.activeRequests.Dec()

			// Wrap response writer
			wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

			// Log request start
			logStart := time.Now()
			logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Msg("request started")
			metrics.logLatency.WithLabelValues("info").Observe(time.Since(logStart).Seconds())
			metrics.logCounter.WithLabelValues("info").Inc()

			// Process request
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)

			// Record request metrics
			metrics.requestCounter.WithLabelValues(
				r.Method,
				r.URL.Path,
				fmt.Sprintf("%d", wrapped.statusCode),
			).Inc()

			metrics.requestDuration.WithLabelValues(
				r.Method,
				r.URL.Path,
			).Observe(duration.Seconds())

			// Log request completion with appropriate level
			logStart = time.Now()
			level := "info"
			logEvent := logger.Info()

			if wrapped.statusCode >= 500 {
				logEvent = logger.Error()
				level = "error"
				metrics.errorCounter.WithLabelValues("http_error", "error").Inc()
			} else if wrapped.statusCode >= 400 {
				logEvent = logger.Warn()
				level = "warn"
				metrics.errorCounter.WithLabelValues("client_error", "warn").Inc()
			}

			logEvent.
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", wrapped.statusCode).
				Dur("duration", duration).
				Msg("request completed")

			metrics.logLatency.WithLabelValues(level).Observe(time.Since(logStart).Seconds())
			metrics.logCounter.WithLabelValues(level).Inc()
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// BusinessEventLogger logs business events with metrics
type BusinessEventLogger struct {
	logger  *bolt.Logger
	metrics *MetricsCollector
}

func NewBusinessEventLogger(logger *bolt.Logger, metrics *MetricsCollector) *BusinessEventLogger {
	return &BusinessEventLogger{
		logger:  logger,
		metrics: metrics,
	}
}

// LogOrderPlaced logs order placement event
func (bel *BusinessEventLogger) LogOrderPlaced(orderID string, userID string, amount float64) {
	start := time.Now()

	bel.logger.Info().
		Str("event_type", "order_placed").
		Str("order_id", orderID).
		Str("user_id", userID).
		Float64("amount", amount).
		Msg("order placed successfully")

	bel.metrics.logLatency.WithLabelValues("info").Observe(time.Since(start).Seconds())
	bel.metrics.logCounter.WithLabelValues("info").Inc()
	bel.metrics.businessEvents.WithLabelValues("order_placed", "success").Inc()
}

// LogPaymentProcessed logs payment processing
func (bel *BusinessEventLogger) LogPaymentProcessed(paymentID string, amount float64, success bool) {
	start := time.Now()

	status := "success"
	level := "info"
	logEvent := bel.logger.Info()

	if !success {
		status = "failed"
		level = "error"
		logEvent = bel.logger.Error()
		bel.metrics.errorCounter.WithLabelValues("payment_failure", "error").Inc()
	}

	logEvent.
		Str("event_type", "payment_processed").
		Str("payment_id", paymentID).
		Float64("amount", amount).
		Bool("success", success).
		Msg("payment processed")

	bel.metrics.logLatency.WithLabelValues(level).Observe(time.Since(start).Seconds())
	bel.metrics.logCounter.WithLabelValues(level).Inc()
	bel.metrics.businessEvents.WithLabelValues("payment_processed", status).Inc()
}

// CacheMonitor monitors cache operations
type CacheMonitor struct {
	logger    *bolt.Logger
	metrics   *MetricsCollector
	hitCount  atomic.Int64
	missCount atomic.Int64
}

func NewCacheMonitor(logger *bolt.Logger, metrics *MetricsCollector) *CacheMonitor {
	return &CacheMonitor{
		logger:  logger,
		metrics: metrics,
	}
}

// LogCacheHit logs cache hit
func (cm *CacheMonitor) LogCacheHit(key string) {
	cm.hitCount.Add(1)
	cm.metrics.cacheHitRate.WithLabelValues("hit").Inc()

	cm.logger.Debug().
		Str("key", key).
		Str("result", "hit").
		Msg("cache access")
}

// LogCacheMiss logs cache miss
func (cm *CacheMonitor) LogCacheMiss(key string) {
	cm.missCount.Add(1)
	cm.metrics.cacheHitRate.WithLabelValues("miss").Inc()

	cm.logger.Debug().
		Str("key", key).
		Str("result", "miss").
		Msg("cache access")
}

// ReportMetrics logs cache metrics periodically
func (cm *CacheMonitor) ReportMetrics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hits := cm.hitCount.Load()
			misses := cm.missCount.Load()
			total := hits + misses

			var hitRate float64
			if total > 0 {
				hitRate = float64(hits) / float64(total) * 100
			}

			cm.logger.Info().
				Int64("cache_hits", hits).
				Int64("cache_misses", misses).
				Float64("hit_rate_pct", hitRate).
				Msg("cache metrics")
		}
	}
}

// QueueMonitor monitors queue depth
type QueueMonitor struct {
	logger  *bolt.Logger
	metrics *MetricsCollector
	depth   atomic.Int64
}

func NewQueueMonitor(logger *bolt.Logger, metrics *MetricsCollector) *QueueMonitor {
	return &QueueMonitor{
		logger:  logger,
		metrics: metrics,
	}
}

// Enqueue adds item to queue
func (qm *QueueMonitor) Enqueue(itemID string) {
	depth := qm.depth.Add(1)
	qm.metrics.queueDepth.Set(float64(depth))

	qm.logger.Debug().
		Str("item_id", itemID).
		Int64("queue_depth", depth).
		Msg("item enqueued")
}

// Dequeue removes item from queue
func (qm *QueueMonitor) Dequeue(itemID string) {
	depth := qm.depth.Add(-1)
	qm.metrics.queueDepth.Set(float64(depth))

	qm.logger.Debug().
		Str("item_id", itemID).
		Int64("queue_depth", depth).
		Msg("item dequeued")
}

// MonitorQueueDepth logs queue depth periodically
func (qm *QueueMonitor) MonitorQueueDepth(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			depth := qm.depth.Load()

			level := qm.logger.Info()
			if depth > 100 {
				level = qm.logger.Warn()
			}

			level.
				Int64("queue_depth", depth).
				Msg("queue status")
		}
	}
}

// Simulator simulates application activity
type Simulator struct {
	businessLogger *BusinessEventLogger
	cacheMonitor   *CacheMonitor
	queueMonitor   *QueueMonitor
}

func NewSimulator(businessLogger *BusinessEventLogger, cacheMonitor *CacheMonitor, queueMonitor *QueueMonitor) *Simulator {
	return &Simulator{
		businessLogger: businessLogger,
		cacheMonitor:   cacheMonitor,
		queueMonitor:   queueMonitor,
	}
}

// Run simulates application activity
func (s *Simulator) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Simulate order placement (70% success rate)
			if rand.Float32() < 0.7 {
				orderID := fmt.Sprintf("order_%d", time.Now().Unix())
				userID := fmt.Sprintf("user_%d", rand.Intn(100))
				amount := 50.0 + rand.Float64()*450.0

				s.businessLogger.LogOrderPlaced(orderID, userID, amount)

				// Simulate payment
				paymentID := fmt.Sprintf("pay_%d", time.Now().Unix())
				success := rand.Float32() < 0.95 // 95% success rate
				s.businessLogger.LogPaymentProcessed(paymentID, amount, success)
			}

			// Simulate cache operations
			key := fmt.Sprintf("cache_key_%d", rand.Intn(50))
			if rand.Float32() < 0.7 { // 70% hit rate
				s.cacheMonitor.LogCacheHit(key)
			} else {
				s.cacheMonitor.LogCacheMiss(key)
			}

			// Simulate queue operations
			if rand.Float32() < 0.6 {
				itemID := fmt.Sprintf("item_%d", time.Now().UnixNano())
				s.queueMonitor.Enqueue(itemID)

				// Process after delay
				go func(id string) {
					time.Sleep(time.Duration(rand.Intn(5)+1) * time.Second)
					s.queueMonitor.Dequeue(id)
				}(itemID)
			}
		}
	}
}

func main() {
	// Initialize logger
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Str("service", "monitoring-example").
		Str("version", "1.0.0").
		Msg("starting monitoring integration example")

	// Initialize metrics collector
	metrics := NewMetricsCollector()

	// Initialize business event logger
	businessLogger := NewBusinessEventLogger(logger, metrics)

	// Initialize cache monitor
	cacheMonitor := NewCacheMonitor(logger, metrics)

	// Initialize queue monitor
	queueMonitor := NewQueueMonitor(logger, metrics)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background monitors
	go cacheMonitor.ReportMetrics(ctx)
	go queueMonitor.MonitorQueueDepth(ctx)

	// Start simulator
	simulator := NewSimulator(businessLogger, cacheMonitor, queueMonitor)
	go simulator.Run(ctx)

	// Setup HTTP server with middleware
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		logger.Debug().Msg("health check")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"healthy"}`)
	})

	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		// Simulate some work
		time.Sleep(time.Duration(10+rand.Intn(40)) * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]}`)
	})

	mux.HandleFunc("/api/error", func(w http.ResponseWriter, r *http.Request) {
		logger.Error().
			Str("error", "simulated error").
			Msg("error endpoint called")

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"error":"simulated error"}`)
	})

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Wrap with logging middleware
	handler := LoggingMiddleware(logger, metrics)(mux)

	// Start HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	go func() {
		logger.Info().
			Str("addr", server.Addr).
			Msg("http server started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().
				Str("error", err.Error()).
				Msg("http server error")
		}
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("shutting down server")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().
			Str("error", err.Error()).
			Msg("shutdown error")
	}

	logger.Info().Msg("server stopped")
}
