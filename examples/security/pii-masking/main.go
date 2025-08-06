// Package main demonstrates PII masking and data protection with Bolt logging.
// This example shows how to handle sensitive data in logs while maintaining
// compliance with GDPR, CCPA, HIPAA, and other privacy regulations.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/felixgeelhaar/bolt/v2"
	"github.com/google/uuid"
)

// PIIMaskingLevel defines the level of masking to apply
type PIIMaskingLevel int

const (
	MaskingNone PIIMaskingLevel = iota
	MaskingPartial
	MaskingComplete
	MaskingHash
)

// PIIClassification defines the type of PII data
type PIIClassification string

const (
	PIIEmail        PIIClassification = "email"
	PIIPhone        PIIClassification = "phone"
	PIISSn          PIIClassification = "ssn"
	PIICreditCard   PIIClassification = "credit_card"
	PIIName         PIIClassification = "name"
	PIIAddress      PIIClassification = "address"
	PIIBankAccount  PIIClassification = "bank_account"
	PIIIPAddress    PIIClassification = "ip_address"
	PIIDateOfBirth  PIIClassification = "date_of_birth"
	PIIDriversLic   PIIClassification = "drivers_license"
	PIIPassport     PIIClassification = "passport"
	PIIMedicalID    PIIClassification = "medical_id"
	PIIGeneric      PIIClassification = "generic"
)

// PIIMasker handles PII data masking and redaction
type PIIMasker struct {
	logger     bolt.Logger
	patterns   map[PIIClassification]*regexp.Regexp
	maskingMap map[PIIClassification]PIIMaskingLevel
}

// NewPIIMasker creates a new PII masking instance
func NewPIIMasker() *PIIMasker {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		Level(bolt.InfoLevel).
		With().
		Str("component", "pii_masker").
		Logger()

	patterns := map[PIIClassification]*regexp.Regexp{
		PIIEmail:       regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
		PIIPhone:       regexp.MustCompile(`\b(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}\b`),
		PIISSn:         regexp.MustCompile(`\b\d{3}-?\d{2}-?\d{4}\b`),
		PIICreditCard:  regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`),
		PIIBankAccount: regexp.MustCompile(`\b\d{8,17}\b`),
		PIIIPAddress:   regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
		PIIDriversLic:  regexp.MustCompile(`\b[A-Z]\d{8}\b`),
		PIIPassport:    regexp.MustCompile(`\b[A-Z]\d{8}\b`),
		PIIMedicalID:   regexp.MustCompile(`\b\d{10}\b`),
	}

	// Configure default masking levels
	maskingMap := map[PIIClassification]PIIMaskingLevel{
		PIIEmail:       MaskingPartial,
		PIIPhone:       MaskingPartial,
		PIISSn:         MaskingComplete,
		PIICreditCard:  MaskingComplete,
		PIIName:        MaskingPartial,
		PIIAddress:     MaskingPartial,
		PIIBankAccount: MaskingComplete,
		PIIIPAddress:   MaskingPartial,
		PIIDateOfBirth: MaskingPartial,
		PIIDriversLic:  MaskingComplete,
		PIIPassport:    MaskingComplete,
		PIIMedicalID:   MaskingComplete,
		PIIGeneric:     MaskingPartial,
	}

	return &PIIMasker{
		logger:     logger,
		patterns:   patterns,
		maskingMap: maskingMap,
	}
}

// MaskString applies PII masking to a string
func (pm *PIIMasker) MaskString(input string) string {
	masked := input

	for classification, pattern := range pm.patterns {
		if pattern.MatchString(masked) {
			level := pm.maskingMap[classification]
			matches := pattern.FindAllString(masked, -1)
			
			for _, match := range matches {
				maskedValue := pm.applyMasking(match, classification, level)
				masked = strings.ReplaceAll(masked, match, maskedValue)
				
				pm.logger.Debug().
					Str("classification", string(classification)).
					Str("masking_level", pm.getMaskingLevelName(level)).
					Str("original_length", fmt.Sprintf("%d", len(match))).
					Str("masked_length", fmt.Sprintf("%d", len(maskedValue))).
					Msg("PII data masked")
			}
		}
	}

	return masked
}

// MaskStruct applies PII masking to struct fields
func (pm *PIIMasker) MaskStruct(input interface{}) interface{} {
	return pm.maskStructValue(reflect.ValueOf(input))
}

func (pm *PIIMasker) maskStructValue(v reflect.Value) interface{} {
	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.String:
		return pm.MaskString(v.String())
	
	case reflect.Struct:
		result := make(map[string]interface{})
		t := v.Type()
		
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldType := t.Field(i)
			
			if !field.CanInterface() {
				continue
			}
			
			fieldName := fieldType.Name
			
			// Check if field should be completely redacted
			if pm.shouldRedactField(fieldName) {
				result[fieldName] = "[REDACTED]"
				continue
			}
			
			result[fieldName] = pm.maskStructValue(field)
		}
		return result
	
	case reflect.Map:
		result := make(map[string]interface{})
		for _, key := range v.MapKeys() {
			keyStr := key.String()
			value := v.MapIndex(key)
			
			if pm.shouldRedactField(keyStr) {
				result[keyStr] = "[REDACTED]"
			} else {
				result[keyStr] = pm.maskStructValue(value)
			}
		}
		return result
	
	case reflect.Slice, reflect.Array:
		var result []interface{}
		for i := 0; i < v.Len(); i++ {
			result = append(result, pm.maskStructValue(v.Index(i)))
		}
		return result
	
	default:
		return v.Interface()
	}
}

// applyMasking applies the specified masking level to a value
func (pm *PIIMasker) applyMasking(value string, classification PIIClassification, level PIIMaskingLevel) string {
	switch level {
	case MaskingNone:
		return value
	
	case MaskingPartial:
		return pm.partialMask(value, classification)
	
	case MaskingComplete:
		return "[MASKED]"
	
	case MaskingHash:
		return fmt.Sprintf("[HASH:%x]", []byte(value)[:4])
	
	default:
		return "[UNKNOWN_MASKING]"
	}
}

// partialMask applies partial masking based on PII type
func (pm *PIIMasker) partialMask(value string, classification PIIClassification) string {
	switch classification {
	case PIIEmail:
		parts := strings.Split(value, "@")
		if len(parts) == 2 {
			username := parts[0]
			domain := parts[1]
			if len(username) > 2 {
				maskedUsername := username[:1] + strings.Repeat("*", len(username)-2) + username[len(username)-1:]
				return maskedUsername + "@" + domain
			}
		}
		return "***@" + parts[1]
	
	case PIIPhone:
		cleaned := regexp.MustCompile(`\D`).ReplaceAllString(value, "")
		if len(cleaned) >= 10 {
			return cleaned[:3] + "-***-" + cleaned[len(cleaned)-4:]
		}
		return "***-***-" + cleaned[len(cleaned)-4:]
	
	case PIIName:
		parts := strings.Fields(value)
		if len(parts) > 0 {
			masked := make([]string, len(parts))
			for i, part := range parts {
				if len(part) > 1 {
					masked[i] = part[:1] + strings.Repeat("*", len(part)-1)
				} else {
					masked[i] = "*"
				}
			}
			return strings.Join(masked, " ")
		}
		return "***"
	
	case PIIAddress:
		// Mask everything except first word and last word
		parts := strings.Fields(value)
		if len(parts) > 2 {
			masked := []string{parts[0]}
			for i := 1; i < len(parts)-1; i++ {
				masked = append(masked, "***")
			}
			masked = append(masked, parts[len(parts)-1])
			return strings.Join(masked, " ")
		}
		return "*** " + parts[len(parts)-1]
	
	case PIIIPAddress:
		parts := strings.Split(value, ".")
		if len(parts) == 4 {
			return parts[0] + ".***.***.***"
		}
		return "***.***.***"
	
	default:
		// Generic partial masking
		if len(value) <= 4 {
			return strings.Repeat("*", len(value))
		}
		return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
	}
}

// shouldRedactField determines if a field should be completely redacted
func (pm *PIIMasker) shouldRedactField(fieldName string) bool {
	sensitiveFields := []string{
		"password", "secret", "token", "key", "auth",
		"ssn", "social_security", "tax_id", "passport_number",
		"credit_card", "bank_account", "routing_number",
		"medical_record", "patient_id", "diagnosis",
	}

	fieldLower := strings.ToLower(fieldName)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(fieldLower, sensitive) {
			return true
		}
	}
	return false
}

func (pm *PIIMasker) getMaskingLevelName(level PIIMaskingLevel) string {
	switch level {
	case MaskingNone:
		return "none"
	case MaskingPartial:
		return "partial"
	case MaskingComplete:
		return "complete"
	case MaskingHash:
		return "hash"
	default:
		return "unknown"
	}
}

// Application represents the main application
type Application struct {
	logger    bolt.Logger
	piiMasker *PIIMasker
}

// User represents a user with potential PII data
type User struct {
	ID           string `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	SSN          string `json:"ssn"`
	Address      string `json:"address"`
	CreditCard   string `json:"credit_card"`
	BankAccount  string `json:"bank_account"`
	DateOfBirth  string `json:"date_of_birth"`
	MedicalID    string `json:"medical_id"`
	DriversLic   string `json:"drivers_license"`
}

// NewApplication creates a new application with PII masking
func NewApplication() *Application {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
		Level(bolt.InfoLevel).
		With().
		Str("service", "pii-masking-demo").
		Str("version", "v1.0.0").
		Str("compliance", "GDPR,CCPA,HIPAA").
		Logger()

	piiMasker := NewPIIMasker()

	return &Application{
		logger:    logger,
		piiMasker: piiMasker,
	}
}

// Middleware for PII-safe logging
func (app *Application) piiLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		correlationID := getOrCreateCorrelationID(r)

		// Mask sensitive headers
		safeHeaders := make(map[string]string)
		for name, values := range r.Header {
			if len(values) > 0 {
				safeHeaders[name] = app.piiMasker.MaskString(values[0])
			}
		}

		// Mask URL parameters that might contain PII
		safeQuery := app.piiMasker.MaskString(r.URL.RawQuery)

		app.logger.Info().
			Str("correlation_id", correlationID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("query", safeQuery).
			Interface("headers", safeHeaders).
			Str("remote_addr", app.piiMasker.MaskString(r.RemoteAddr)).
			Str("user_agent", app.piiMasker.MaskString(r.UserAgent())).
			Msg("Request started (PII-masked)")

		// Wrap response writer
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)

		app.logger.Info().
			Str("correlation_id", correlationID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status_code", wrapper.statusCode).
			Dur("duration", duration).
			Msg("Request completed (PII-safe)")
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// User creation handler with PII masking
func (app *Application) createUserHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		app.logger.Error().
			Str("correlation_id", correlationID).
			Err(err).
			Msg("Failed to decode user data")
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Log user creation with PII masking
	maskedUser := app.piiMasker.MaskStruct(user)
	
	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("operation", "create_user").
		Interface("user_data", maskedUser).
		Msg("User creation attempt")

	// Simulate user creation logic
	user.ID = uuid.New().String()

	// Log successful creation (still masked)
	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("operation", "create_user").
		Str("user_id", user.ID).
		Interface("created_user", app.piiMasker.MaskStruct(user)).
		Msg("User created successfully")

	// Return response (also mask sensitive data in response)
	maskedResponse := app.piiMasker.MaskStruct(user)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "created",
		"user":          maskedResponse,
		"correlation_id": correlationID,
	})
}

// User retrieval handler with PII masking
func (app *Application) getUserHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)
	userID := r.URL.Query().Get("user_id")

	// Simulate user retrieval
	user := User{
		ID:           userID,
		FirstName:    "John",
		LastName:     "Doe",
		Email:        "john.doe@example.com",
		Phone:        "555-123-4567",
		SSN:          "123-45-6789",
		Address:      "123 Main St, Anytown, CA 90210",
		CreditCard:   "4532-1234-5678-9012",
		BankAccount:  "987654321",
		DateOfBirth:  "1980-01-01",
		MedicalID:    "MED1234567",
		DriversLic:   "D1234567",
	}

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("operation", "get_user").
		Str("user_id", userID).
		Interface("user_data", app.piiMasker.MaskStruct(user)).
		Msg("User data retrieved")

	// Return masked user data
	maskedUser := app.piiMasker.MaskStruct(user)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":           maskedUser,
		"correlation_id": correlationID,
	})
}

// Search handler demonstrating PII masking in search operations
func (app *Application) searchUsersHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)
	query := r.URL.Query().Get("q")

	// Mask the search query for logging
	maskedQuery := app.piiMasker.MaskString(query)

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("operation", "search_users").
		Str("search_query", maskedQuery).
		Msg("User search performed")

	// Simulate search results
	users := []User{
		{
			ID:        "1",
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john.doe@example.com",
			Phone:     "555-123-4567",
		},
		{
			ID:        "2",
			FirstName: "Jane",
			LastName:  "Smith",
			Email:     "jane.smith@example.com",
			Phone:     "555-987-6543",
		},
	}

	// Mask all user data in results
	var maskedUsers []interface{}
	for _, user := range users {
		maskedUsers = append(maskedUsers, app.piiMasker.MaskStruct(user))
	}

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("operation", "search_users").
		Int("results_count", len(users)).
		Interface("results", maskedUsers).
		Msg("Search completed")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":          maskedQuery,
		"results":        maskedUsers,
		"total":          len(users),
		"correlation_id": correlationID,
	})
}

// Error handler demonstrating PII masking in error logs
func (app *Application) errorHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)

	// Simulate an error with potentially sensitive information
	sensitiveError := fmt.Errorf("database error: failed to update user john.doe@example.com (SSN: 123-45-6789) at 192.168.1.100")
	
	// Mask PII in error messages
	maskedError := app.piiMasker.MaskString(sensitiveError.Error())

	app.logger.Error().
		Str("correlation_id", correlationID).
		Str("error_message", maskedError).
		Str("operation", "simulate_error").
		Msg("Error occurred with PII masking")

	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// Configuration handler to show masking settings
func (app *Application) configHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)

	config := map[string]interface{}{
		"masking_levels": map[string]string{
			"email":        "partial",
			"phone":        "partial", 
			"ssn":          "complete",
			"credit_card":  "complete",
			"bank_account": "complete",
			"name":         "partial",
			"address":      "partial",
		},
		"compliance_frameworks": []string{"GDPR", "CCPA", "HIPAA"},
		"redacted_fields": []string{"password", "secret", "token", "key"},
	}

	app.logger.Info().
		Str("correlation_id", correlationID).
		Str("operation", "get_pii_config").
		Interface("config", config).
		Msg("PII masking configuration retrieved")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pii_masking_config": config,
		"correlation_id":     correlationID,
	})
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

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Apply PII-safe logging middleware
	handler := app.piiLoggingMiddleware(mux)

	// Register handlers
	mux.HandleFunc("/users/create", app.createUserHandler)
	mux.HandleFunc("/users/get", app.getUserHandler)
	mux.HandleFunc("/users/search", app.searchUsersHandler)
	mux.HandleFunc("/error", app.errorHandler)
	mux.HandleFunc("/config/pii", app.configHandler)

	port := getEnv("PORT", "8080")

	app.logger.Info().
		Str("port", port).
		Str("compliance", "GDPR,CCPA,HIPAA").
		Bool("pii_masking_enabled", true).
		Msg("Starting PII masking demo server")

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
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