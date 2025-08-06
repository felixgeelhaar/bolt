// Package main demonstrates enterprise audit logging with Bolt.
// This example shows compliance-focused logging for regulatory requirements
// including SOX, GDPR, HIPAA, and other enterprise audit standards.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/felixgeelhaar/bolt"
	"github.com/google/uuid"
)

// AuditEvent represents a structured audit event
type AuditEvent struct {
	EventID       string    `json:"event_id"`
	Timestamp     time.Time `json:"timestamp"`
	EventType     string    `json:"event_type"`
	Action        string    `json:"action"`
	Resource      string    `json:"resource"`
	ResourceID    string    `json:"resource_id,omitempty"`
	UserID        string    `json:"user_id"`
	UserEmail     string    `json:"user_email,omitempty"`
	SessionID     string    `json:"session_id"`
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent"`
	Result        string    `json:"result"`
	ResultCode    int       `json:"result_code"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	DataChanged   bool      `json:"data_changed"`
	BeforeHash    string    `json:"before_hash,omitempty"`
	AfterHash     string    `json:"after_hash,omitempty"`
	ComplianceTag string    `json:"compliance_tag"`
	Severity      string    `json:"severity"`
	Category      string    `json:"category"`
}

// AuditLogger handles enterprise audit logging
type AuditLogger struct {
	logger         bolt.Logger
	sensitiveFields []string
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	// Configure structured logging for audit trail
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		Level(bolt.InfoLevel).
		With().
		Str("service", "audit-logger").
		Str("version", "v1.0.0").
		Str("environment", getEnv("ENVIRONMENT", "production")).
		Str("compliance_framework", "SOX,GDPR,HIPAA").
		Logger()

	return &AuditLogger{
		logger: logger,
		sensitiveFields: []string{
			"password", "ssn", "credit_card", "bank_account",
			"api_key", "token", "secret", "private_key",
		},
	}
}

// LogAuditEvent logs a structured audit event
func (al *AuditLogger) LogAuditEvent(event AuditEvent) {
	// Ensure event ID is set
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Log structured audit event
	al.logger.Info().
		Str("audit_event_id", event.EventID).
		Time("audit_timestamp", event.Timestamp).
		Str("audit_event_type", event.EventType).
		Str("audit_action", event.Action).
		Str("audit_resource", event.Resource).
		Str("audit_resource_id", event.ResourceID).
		Str("audit_user_id", event.UserID).
		Str("audit_user_email", event.UserEmail).
		Str("audit_session_id", event.SessionID).
		Str("audit_ip_address", event.IPAddress).
		Str("audit_user_agent", event.UserAgent).
		Str("audit_result", event.Result).
		Int("audit_result_code", event.ResultCode).
		Str("audit_error_message", event.ErrorMessage).
		Bool("audit_data_changed", event.DataChanged).
		Str("audit_before_hash", event.BeforeHash).
		Str("audit_after_hash", event.AfterHash).
		Str("audit_compliance_tag", event.ComplianceTag).
		Str("audit_severity", event.Severity).
		Str("audit_category", event.Category).
		Msg("Audit event recorded")
}

// Application represents the main application
type Application struct {
	auditLogger *AuditLogger
	logger      bolt.Logger
}

// NewApplication creates a new application with audit logging
func NewApplication() *Application {
	auditLogger := NewAuditLogger()

	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		Level(bolt.InfoLevel).
		With().
		Str("service", "audit-demo").
		Str("version", "v1.0.0").
		Logger()

	return &Application{
		auditLogger: auditLogger,
		logger:      logger,
	}
}

// Audit middleware for HTTP requests
func (app *Application) auditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		correlationID := getOrCreateCorrelationID(r)

		// Extract user information (in real app, from JWT/session)
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			userID = "anonymous"
		}
		userEmail := r.Header.Get("X-User-Email")
		sessionID := r.Header.Get("X-Session-ID")
		if sessionID == "" {
			sessionID = uuid.New().String()
		}

		// Wrap response writer to capture status
		wrapper := &auditResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Log request start audit event
		app.auditLogger.LogAuditEvent(AuditEvent{
			EventType:     "HTTP_REQUEST_START",
			Action:        r.Method + " " + r.URL.Path,
			Resource:      r.URL.Path,
			UserID:        userID,
			UserEmail:     userEmail,
			SessionID:     sessionID,
			IPAddress:     getClientIP(r),
			UserAgent:     r.UserAgent(),
			Result:        "IN_PROGRESS",
			ComplianceTag: "ACCESS_LOG",
			Severity:      "INFO",
			Category:      "ACCESS",
		})

		// Process request
		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)
		
		// Determine compliance severity based on status
		severity := "INFO"
		category := "ACCESS"
		complianceTag := "ACCESS_LOG"
		
		if wrapper.statusCode >= 400 {
			severity = "WARN"
			category = "ACCESS_DENIED"
			complianceTag = "SECURITY_EVENT"
		}
		if wrapper.statusCode >= 500 {
			severity = "ERROR"
			category = "SYSTEM_ERROR"
			complianceTag = "SYSTEM_FAILURE"
		}

		// Log completion audit event
		app.auditLogger.LogAuditEvent(AuditEvent{
			EventType:     "HTTP_REQUEST_COMPLETE",
			Action:        r.Method + " " + r.URL.Path,
			Resource:      r.URL.Path,
			UserID:        userID,
			UserEmail:     userEmail,
			SessionID:     sessionID,
			IPAddress:     getClientIP(r),
			UserAgent:     r.UserAgent(),
			Result:        getResultFromStatus(wrapper.statusCode),
			ResultCode:    wrapper.statusCode,
			ComplianceTag: complianceTag,
			Severity:      severity,
			Category:      category,
		})

		// Regular application logging
		app.logger.Info().
			Str("correlation_id", correlationID).
			Str("user_id", userID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", wrapper.statusCode).
			Dur("duration", duration).
			Msg("Request processed")
	})
}

type auditResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *auditResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Authentication audit handler
func (app *Application) loginHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)
	userEmail := r.FormValue("email")
	password := r.FormValue("password")
	
	// Simulate authentication
	success := userEmail != "" && password != ""
	
	result := "FAILURE"
	resultCode := http.StatusUnauthorized
	severity := "WARN"
	category := "AUTHENTICATION_FAILURE"
	
	if success {
		result = "SUCCESS"
		resultCode = http.StatusOK
		severity = "INFO"
		category = "AUTHENTICATION_SUCCESS"
	}

	// Generate session ID for successful logins
	sessionID := ""
	if success {
		sessionID = uuid.New().String()
	}

	// Audit login attempt
	app.auditLogger.LogAuditEvent(AuditEvent{
		EventType:     "USER_AUTHENTICATION",
		Action:        "LOGIN_ATTEMPT",
		Resource:      "/auth/login",
		UserEmail:     userEmail,
		SessionID:     sessionID,
		IPAddress:     getClientIP(r),
		UserAgent:     r.UserAgent(),
		Result:        result,
		ResultCode:    resultCode,
		ComplianceTag: "AUTHENTICATION_EVENT",
		Severity:      severity,
		Category:      category,
	})

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("user_email", maskSensitiveData(userEmail)).
		Str("result", result).
		Bool("success", success).
		Msg("Login attempt")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resultCode)
	
	if success {
		fmt.Fprintf(w, `{
			"status": "success", 
			"session_id": "%s",
			"correlation_id": "%s"
		}`, sessionID, correlationID)
	} else {
		fmt.Fprintf(w, `{
			"status": "failure",
			"message": "Invalid credentials",
			"correlation_id": "%s"
		}`, correlationID)
	}
}

// Data access audit handler
func (app *Application) getUserDataHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)
	userID := r.URL.Query().Get("user_id")
	requestorID := r.Header.Get("X-User-ID")
	sessionID := r.Header.Get("X-Session-ID")

	// Simulate authorization check
	authorized := requestorID == userID || requestorID == "admin"
	
	if !authorized {
		// Audit unauthorized access attempt
		app.auditLogger.LogAuditEvent(AuditEvent{
			EventType:     "DATA_ACCESS",
			Action:        "UNAUTHORIZED_ACCESS_ATTEMPT",
			Resource:      "USER_DATA",
			ResourceID:    userID,
			UserID:        requestorID,
			SessionID:     sessionID,
			IPAddress:     getClientIP(r),
			UserAgent:     r.UserAgent(),
			Result:        "DENIED",
			ResultCode:    http.StatusForbidden,
			ComplianceTag: "SECURITY_VIOLATION",
			Severity:      "HIGH",
			Category:      "UNAUTHORIZED_ACCESS",
		})

		app.logger.Warn().
			Str("correlation_id", correlationID).
			Str("requestor_id", requestorID).
			Str("target_user_id", userID).
			Msg("Unauthorized access attempt")

		http.Error(w, "Access Denied", http.StatusForbidden)
		return
	}

	// Simulate data retrieval
	userData := map[string]interface{}{
		"id":    userID,
		"name":  "John Doe",
		"email": "john@example.com",
		"role":  "user",
	}

	// Create data hash for integrity
	dataHash := createDataHash(userData)

	// Audit successful data access
	app.auditLogger.LogAuditEvent(AuditEvent{
		EventType:     "DATA_ACCESS",
		Action:        "VIEW_USER_DATA",
		Resource:      "USER_DATA",
		ResourceID:    userID,
		UserID:        requestorID,
		SessionID:     sessionID,
		IPAddress:     getClientIP(r),
		UserAgent:     r.UserAgent(),
		Result:        "SUCCESS",
		ResultCode:    http.StatusOK,
		DataChanged:   false,
		AfterHash:     dataHash,
		ComplianceTag: "DATA_ACCESS_LOG",
		Severity:      "INFO",
		Category:      "DATA_ACCESS",
	})

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("requestor_id", requestorID).
		Str("user_id", userID).
		Str("data_hash", dataHash).
		Msg("User data accessed")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"user": {
			"id": "%s",
			"name": "%s",
			"email": "%s",
			"role": "%s"
		},
		"correlation_id": "%s"
	}`, userID, userData["name"], userData["email"], userData["role"], correlationID)
}

// Data modification audit handler
func (app *Application) updateUserDataHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)
	userID := r.URL.Query().Get("user_id")
	requestorID := r.Header.Get("X-User-ID")
	sessionID := r.Header.Get("X-Session-ID")

	// Simulate getting current data
	beforeData := map[string]interface{}{
		"id":    userID,
		"name":  "John Doe",
		"email": "john@example.com",
		"role":  "user",
	}
	beforeHash := createDataHash(beforeData)

	// Simulate data update
	afterData := map[string]interface{}{
		"id":    userID,
		"name":  "John Smith", // Changed
		"email": "john.smith@example.com", // Changed
		"role":  "user",
	}
	afterHash := createDataHash(afterData)

	// Audit data modification
	app.auditLogger.LogAuditEvent(AuditEvent{
		EventType:     "DATA_MODIFICATION",
		Action:        "UPDATE_USER_DATA",
		Resource:      "USER_DATA",
		ResourceID:    userID,
		UserID:        requestorID,
		SessionID:     sessionID,
		IPAddress:     getClientIP(r),
		UserAgent:     r.UserAgent(),
		Result:        "SUCCESS",
		ResultCode:    http.StatusOK,
		DataChanged:   true,
		BeforeHash:    beforeHash,
		AfterHash:     afterHash,
		ComplianceTag: "DATA_CHANGE_LOG",
		Severity:      "MEDIUM",
		Category:      "DATA_MODIFICATION",
	})

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("user_id", userID).
		Str("requestor_id", requestorID).
		Str("before_hash", beforeHash).
		Str("after_hash", afterHash).
		Bool("data_changed", true).
		Msg("User data updated")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"status": "updated",
		"user_id": "%s",
		"before_hash": "%s",
		"after_hash": "%s",
		"correlation_id": "%s"
	}`, userID, beforeHash, afterHash, correlationID)
}

// Administrative action audit handler
func (app *Application) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)
	userID := r.URL.Query().Get("user_id")
	requestorID := r.Header.Get("X-User-ID")
	sessionID := r.Header.Get("X-Session-ID")

	// Simulate authorization check (only admins can delete)
	isAdmin := requestorID == "admin"
	
	if !isAdmin {
		// Audit unauthorized administrative attempt
		app.auditLogger.LogAuditEvent(AuditEvent{
			EventType:     "ADMINISTRATIVE_ACTION",
			Action:        "UNAUTHORIZED_DELETE_ATTEMPT",
			Resource:      "USER_DATA",
			ResourceID:    userID,
			UserID:        requestorID,
			SessionID:     sessionID,
			IPAddress:     getClientIP(r),
			UserAgent:     r.UserAgent(),
			Result:        "DENIED",
			ResultCode:    http.StatusForbidden,
			ComplianceTag: "SECURITY_VIOLATION",
			Severity:      "CRITICAL",
			Category:      "UNAUTHORIZED_ADMIN_ACTION",
		})

		http.Error(w, "Access Denied", http.StatusForbidden)
		return
	}

	// Get data before deletion for audit trail
	beforeData := map[string]interface{}{
		"id":    userID,
		"name":  "John Doe",
		"email": "john@example.com",
		"role":  "user",
	}
	beforeHash := createDataHash(beforeData)

	// Audit successful deletion
	app.auditLogger.LogAuditEvent(AuditEvent{
		EventType:     "DATA_DELETION",
		Action:        "DELETE_USER",
		Resource:      "USER_DATA",
		ResourceID:    userID,
		UserID:        requestorID,
		SessionID:     sessionID,
		IPAddress:     getClientIP(r),
		UserAgent:     r.UserAgent(),
		Result:        "SUCCESS",
		ResultCode:    http.StatusOK,
		DataChanged:   true,
		BeforeHash:    beforeHash,
		ComplianceTag: "DATA_DELETION_LOG",
		Severity:      "HIGH",
		Category:      "DATA_DELETION",
	})

	app.logger.Warn().
		Str("correlation_id", correlationID).
		Str("user_id", userID).
		Str("requestor_id", requestorID).
		Str("before_hash", beforeHash).
		Msg("User deleted")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"status": "deleted",
		"user_id": "%s",
		"before_hash": "%s",
		"correlation_id": "%s"
	}`, userID, beforeHash, correlationID)
}

// System events audit
func (app *Application) systemEventHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)
	eventType := r.URL.Query().Get("event")

	var auditEventType, action, severity, category string

	switch eventType {
	case "startup":
		auditEventType = "SYSTEM_EVENT"
		action = "APPLICATION_STARTUP"
		severity = "INFO"
		category = "SYSTEM_LIFECYCLE"
	case "shutdown":
		auditEventType = "SYSTEM_EVENT"
		action = "APPLICATION_SHUTDOWN"
		severity = "INFO"
		category = "SYSTEM_LIFECYCLE"
	case "config_change":
		auditEventType = "CONFIGURATION_CHANGE"
		action = "SYSTEM_CONFIGURATION_UPDATED"
		severity = "MEDIUM"
		category = "CONFIGURATION"
	default:
		auditEventType = "SYSTEM_EVENT"
		action = "UNKNOWN_SYSTEM_EVENT"
		severity = "WARN"
		category = "UNKNOWN"
	}

	// Audit system event
	app.auditLogger.LogAuditEvent(AuditEvent{
		EventType:     auditEventType,
		Action:        action,
		Resource:      "SYSTEM",
		UserID:        "SYSTEM",
		IPAddress:     getClientIP(r),
		Result:        "SUCCESS",
		ResultCode:    http.StatusOK,
		ComplianceTag: "SYSTEM_EVENT_LOG",
		Severity:      severity,
		Category:      category,
	})

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("system_event", eventType).
		Str("action", action).
		Msg("System event processed")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"event": "%s",
		"status": "logged",
		"correlation_id": "%s"
	}`, eventType, correlationID)
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
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	
	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

func getResultFromStatus(status int) string {
	if status >= 200 && status < 300 {
		return "SUCCESS"
	}
	if status >= 400 && status < 500 {
		return "CLIENT_ERROR"
	}
	if status >= 500 {
		return "SERVER_ERROR"
	}
	return "UNKNOWN"
}

func createDataHash(data interface{}) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%v", data)))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for brevity
}

func maskSensitiveData(data string) string {
	if len(data) <= 4 {
		return "***"
	}
	return data[:2] + "***" + data[len(data)-2:]
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	app := NewApplication()

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Apply audit middleware to all routes
	handler := app.auditMiddleware(mux)

	// Register handlers
	mux.HandleFunc("/auth/login", app.loginHandler)
	mux.HandleFunc("/users/data", app.getUserDataHandler)
	mux.HandleFunc("/users/update", app.updateUserDataHandler)
	mux.HandleFunc("/admin/delete-user", app.deleteUserHandler)
	mux.HandleFunc("/system/event", app.systemEventHandler)

	port := getEnv("PORT", "8080")

	app.logger.Info().
		Str("port", port).
		Str("compliance_frameworks", "SOX,GDPR,HIPAA").
		Msg("Starting audit logging demo server")

	// Log application startup as system event
	app.auditLogger.LogAuditEvent(AuditEvent{
		EventType:     "SYSTEM_EVENT",
		Action:        "APPLICATION_STARTUP",
		Resource:      "SYSTEM",
		UserID:        "SYSTEM",
		Result:        "SUCCESS",
		ComplianceTag: "SYSTEM_EVENT_LOG",
		Severity:      "INFO",
		Category:      "SYSTEM_LIFECYCLE",
	})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		// Log application shutdown as system event
		app.auditLogger.LogAuditEvent(AuditEvent{
			EventType:     "SYSTEM_EVENT",
			Action:        "APPLICATION_SHUTDOWN",
			Resource:      "SYSTEM",
			UserID:        "SYSTEM",
			Result:        "ERROR",
			ErrorMessage:  err.Error(),
			ComplianceTag: "SYSTEM_EVENT_LOG",
			Severity:      "ERROR",
			Category:      "SYSTEM_FAILURE",
		})

		app.logger.Fatal().Err(err).Msg("Server failed to start")
	}
}