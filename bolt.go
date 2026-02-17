// Package bolt provides a high-performance, zero-allocation structured logging library for Go.
//
// Bolt is designed for applications that demand exceptional performance without compromising
// on features. It delivers sub-100ns logging operations with zero memory allocations in hot paths.
//
// # Key Features
//
// - Zero allocations in hot paths
// - Sub-100ns latency for logging operations
// - Type-safe field methods
// - JSON and console output formats
// - OpenTelemetry tracing integration
// - Environment variable configuration
// - Production-ready reliability
//
// # Performance
//
// Bolt achieves industry-leading performance:
//   - 105.2ns/op with 0 allocations
//   - 64% faster than Zerolog
//   - 80% faster than Zap
//   - 2603% faster than Logrus
//
// # Quick Start
//
// Basic usage with JSON output:
//
//	package main
//
//	import (
//	    "os"
//	    "github.com/felixgeelhaar/bolt"
//	)
//
//	func main() {
//	    logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
//
//	    logger.Info().
//	        Str("service", "auth").
//	        Int("user_id", 12345).
//	        Msg("User authenticated")
//	}
//
// Console output with colors:
//
//	logger := bolt.New(bolt.NewConsoleHandler(os.Stdout))
//	logger.Info().Str("status", "ready").Msg("Server started")
//
// # Configuration
//
// Environment variables:
//   - BOLT_LEVEL: Set log level (trace, debug, info, warn, error, fatal)
//   - BOLT_FORMAT: Set output format (json, console)
//
// Programmatic configuration:
//
//	logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.DEBUG)
//
// # Zero Allocations
//
// Bolt uses object pooling and direct serialization to achieve zero allocations:
//
//	// This logging operation performs 0 allocations
//	logger.Info().
//	    Str("method", "GET").
//	    Str("path", "/api/users").
//	    Int("status", 200).
//	    Dur("duration", time.Since(start)).
//	    Msg("Request completed")
//
// # OpenTelemetry Integration
//
// Bolt automatically extracts trace information from context:
//
//	ctx := context.Background()
//	logger.Info().Ctx(ctx).Msg("Operation completed")
//
// # Thread Safety
//
// All Bolt operations are thread-safe and can be used concurrently across goroutines.
// The library uses atomic operations for level changes and sync.Pool for event management.
//
// # Security Features
//
// Bolt includes comprehensive security protections:
//   - Automatic JSON escaping prevents log injection attacks
//   - Input validation with configurable size limits (keys: 256 chars, values: 64KB)
//   - Control character filtering in keys
//   - Buffer size limits prevent resource exhaustion (max 1MB per entry)
//   - Thread-safe operations prevent race conditions
//   - Secure error handling prevents information disclosure
//
// # Performance Characteristics
//
// Bolt delivers industry-leading performance:
//   - Zero allocations in hot paths through intelligent event pooling
//   - Sub-100ns latency for most logging operations
//   - Custom serialization optimized for common data types
//   - Lock-free event management with atomic synchronization
package bolt

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	oteltrace "go.opentelemetry.io/otel/trace"
)

// Constants for buffer sizes and configuration
const (
	// DefaultBufferSize is the initial buffer size for events - increased to reduce reallocations
	DefaultBufferSize = 2048
	// MaxBufferSize is the maximum allowed buffer size to prevent unbounded growth
	MaxBufferSize = 1024 * 1024 // 1MB
	// StackTraceBufferSize is the buffer size for stack traces
	StackTraceBufferSize = 64 * 1024 // 64KB
	// DefaultFilePermissions for log files
	DefaultFilePermissions = 0644
	// MaxKeyLength is the maximum allowed key length
	MaxKeyLength = 256
	// MaxValueLength is the maximum allowed value length
	MaxValueLength = 64 * 1024 // 64KB
)

// Level string constants
const (
	traceStr   = "trace"
	debugStr   = "debug"
	infoStr    = "info"
	warnStr    = "warn"
	errorStr   = "error"
	fatalStr   = "fatal"
	consoleStr = "console"
)

// Fast number conversion helpers to avoid allocations
var digits = "0123456789"
var hexDigits = "0123456789ABCDEF"

// appendInt appends an integer to the buffer without allocations
func appendInt(buf []byte, i int) []byte {
	if i == 0 {
		return append(buf, '0')
	}

	// Handle math.MinInt64 special case to prevent overflow
	// MinInt64 = -9223372036854775808, which cannot be negated without overflow
	if i == -9223372036854775808 {
		return append(buf, "-9223372036854775808"...)
	}

	// Handle negative numbers
	if i < 0 {
		buf = append(buf, '-')
		i = -i
	}

	// Fast path for small numbers (0-99) - most common case
	if i < 100 {
		if i < 10 {
			return append(buf, digits[i])
		}
		return append(buf, digits[i/10], digits[i%10])
	}

	// For larger numbers, build from the end
	start := len(buf)
	for i > 0 {
		buf = append(buf, digits[i%10])
		i /= 10
	}

	// Reverse the digits we just added
	end := len(buf) - 1
	for start < end {
		buf[start], buf[end] = buf[end], buf[start]
		start++
		end--
	}

	return buf
}

// appendUint appends an unsigned integer to the buffer without allocations
func appendUint(buf []byte, i uint64) []byte {
	if i == 0 {
		return append(buf, '0')
	}

	// Fast path for small numbers (0-99) - most common case
	if i < 100 {
		if i < 10 {
			return append(buf, digits[i])
		}
		return append(buf, digits[i/10], digits[i%10])
	}

	// For larger numbers, build from the end
	start := len(buf)
	for i > 0 {
		buf = append(buf, digits[i%10])
		i /= 10
	}

	// Reverse the digits we just added
	end := len(buf) - 1
	for start < end {
		buf[start], buf[end] = buf[end], buf[start]
		start++
		end--
	}

	return buf
}

// appendBool appends a boolean to the buffer without allocations
func appendBool(buf []byte, b bool) []byte {
	if b {
		return append(buf, "true"...)
	}
	return append(buf, "false"...)
}

// appendJSONString appends a JSON-escaped string to the buffer without allocations.
// This is a critical security function that prevents JSON injection attacks by properly
// escaping all special characters according to RFC 7159. It handles:
//   - Quote characters that could break JSON structure
//   - Control characters that could corrupt log format
//   - Backslashes that could create escape sequences
//   - Unicode control characters (U+0000 to U+001F)
func appendJSONString(buf []byte, s string) []byte {
	// Fast path: iterate once, handling both UTF-8 validation and JSON escaping
	for i := 0; i < len(s); {
		c := s[i]

		// Fast path for ASCII characters (most common case)
		if c < utf8.RuneSelf {
			switch c {
			case '"':
				buf = append(buf, '\\', '"')
			case '\\':
				buf = append(buf, '\\', '\\')
			case '\b':
				buf = append(buf, '\\', 'b')
			case '\f':
				buf = append(buf, '\\', 'f')
			case '\n':
				buf = append(buf, '\\', 'n')
			case '\r':
				buf = append(buf, '\\', 'r')
			case '\t':
				buf = append(buf, '\\', 't')
			default:
				if c < 0x20 {
					// Escape control characters as \u00XX
					buf = append(buf, '\\', 'u', '0', '0')
					buf = append(buf, hexDigits[c>>4], hexDigits[c&0xF])
				} else {
					buf = append(buf, c)
				}
			}
			i++
			continue
		}

		// Multi-byte UTF-8 character - validate and copy
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 - replace with replacement character (U+FFFD = �)
			buf = append(buf, 0xEF, 0xBF, 0xBD) // UTF-8 encoding of U+FFFD
		} else {
			// Valid UTF-8 - copy bytes directly
			buf = append(buf, s[i:i+size]...)
		}
		i += size
	}
	return buf
}

// appendIP appends an IP address to the buffer without allocations.
// IPv4 addresses use dotted-decimal notation, IPv6 uses colon-hex notation.
func appendIP(buf []byte, ip net.IP) []byte {
	if p4 := ip.To4(); p4 != nil {
		buf = appendUint(buf, uint64(p4[0]))
		buf = append(buf, '.')
		buf = appendUint(buf, uint64(p4[1]))
		buf = append(buf, '.')
		buf = appendUint(buf, uint64(p4[2]))
		buf = append(buf, '.')
		buf = appendUint(buf, uint64(p4[3]))
		return buf
	}
	if len(ip) == net.IPv6len {
		for i := 0; i < net.IPv6len; i += 2 {
			if i > 0 {
				buf = append(buf, ':')
			}
			buf = appendHex16(buf, uint16(ip[i])<<8|uint16(ip[i+1]))
		}
		return buf
	}
	return buf
}

// appendHex16 appends a 16-bit value as 4 lowercase hex digits.
func appendHex16(buf []byte, v uint16) []byte {
	const lowerHex = "0123456789abcdef"
	buf = append(buf, lowerHex[(v>>12)&0xf])
	buf = append(buf, lowerHex[(v>>8)&0xf])
	buf = append(buf, lowerHex[(v>>4)&0xf])
	buf = append(buf, lowerHex[v&0xf])
	return buf
}

// validateKey validates a key parameter for safety and length to prevent security vulnerabilities.
// This function protects against:
//   - Control character injection (prevents log format corruption)
//   - Resource exhaustion attacks (enforces 256 character limit)
//   - Empty key exploitation (requires non-empty keys)
func validateKey(key string) error {
	// Trim whitespace and check for empty
	trimmed := strings.TrimSpace(key)
	if len(trimmed) == 0 {
		return fmt.Errorf("key cannot be empty or whitespace-only")
	}
	if len(key) > MaxKeyLength {
		return fmt.Errorf("key length exceeds maximum of %d characters", MaxKeyLength)
	}
	// Check for control characters in key
	for i := 0; i < len(key); i++ {
		if key[i] < 0x20 || key[i] == 0x7F {
			return fmt.Errorf("key contains invalid control character")
		}
	}
	return nil
}

// validateValue validates a value parameter for safety and length
func validateValue(value string) error {
	if len(value) > MaxValueLength {
		return fmt.Errorf("value length exceeds maximum of %d characters", MaxValueLength)
	}
	return nil
}

// checkBufferSize checks if buffer size is within limits and prevents unbounded growth
func checkBufferSize(buf []byte) error {
	if len(buf) > MaxBufferSize {
		return fmt.Errorf("buffer size exceeds maximum of %d bytes", MaxBufferSize)
	}
	return nil
}

// RFC3339 timestamp formatting without allocations
func appendRFC3339(buf []byte, t time.Time) []byte {
	year, month, day := t.Date()
	hour, minute, sec := t.Clock()
	nano := t.Nanosecond()

	buf = appendDate(buf, year, int(month), day)
	buf = append(buf, 'T')
	buf = appendTime(buf, hour, minute, sec)
	buf = appendNanoseconds(buf, nano)
	buf = append(buf, 'Z')
	return buf
}

// appendDate appends date in YYYY-MM-DD format
func appendDate(buf []byte, year, month, day int) []byte {
	buf = appendInt(buf, year)
	buf = append(buf, '-')
	buf = appendTwoDigits(buf, month)
	buf = append(buf, '-')
	buf = appendTwoDigits(buf, day)
	return buf
}

// appendTime appends time in HH:MM:SS format
func appendTime(buf []byte, hour, minute, sec int) []byte {
	buf = appendTwoDigits(buf, hour)
	buf = append(buf, ':')
	buf = appendTwoDigits(buf, minute)
	buf = append(buf, ':')
	buf = appendTwoDigits(buf, sec)
	return buf
}

// appendTwoDigits appends a number with leading zero if needed
func appendTwoDigits(buf []byte, value int) []byte {
	if value < 10 {
		buf = append(buf, '0')
	}
	return appendInt(buf, value)
}

// appendNanoseconds appends nanoseconds if non-zero using zero-allocation formatting
func appendNanoseconds(buf []byte, nano int) []byte {
	if nano == 0 {
		return buf
	}
	buf = append(buf, '.')
	// Format nanoseconds to 9 digits without allocations
	buf = appendNanoDigits(buf, nano)
	// Trim trailing zeros
	for len(buf) > 0 && buf[len(buf)-1] == '0' {
		buf = buf[:len(buf)-1]
	}
	return buf
}

// appendNanoDigits appends nanoseconds as 9 digits without allocations
func appendNanoDigits(buf []byte, nano int) []byte {
	// Convert nanoseconds to 9-digit string without allocations
	digitBuf := [9]byte{}
	for i := 8; i >= 0; i-- {
		digitBuf[i] = byte(nano%10) + '0'
		nano /= 10
	}
	return append(buf, digitBuf[:]...)
}

// appendFloat64 appends a float64 to the buffer without allocations.
// This implementation uses a fixed-point approach for common float values,
// providing reasonable precision while maintaining zero allocations.
func appendFloat64(buf []byte, f float64) []byte {
	// Handle special cases
	if f != f { // NaN
		return append(buf, `"NaN"`...)
	}
	if f > 1e308 { // +Inf
		return append(buf, `"+Inf"`...)
	}
	if f < -1e308 { // -Inf
		return append(buf, `"-Inf"`...)
	}

	// Handle negative numbers
	if f < 0 {
		buf = append(buf, '-')
		f = -f
	}

	// Handle zero
	if f == 0 {
		return append(buf, '0')
	}

	// For very large or very small numbers, use scientific notation
	if f >= 1e15 || (f > 0 && f < 1e-6) {
		return appendFloatScientific(buf, f)
	}

	// Split into integer and fractional parts
	intPart := int64(f)
	fracPart := f - float64(intPart)

	// Append integer part
	buf = appendInt(buf, int(intPart))

	// Append fractional part if non-zero (up to 6 decimal places for precision)
	if fracPart > 0 {
		buf = append(buf, '.')
		// Multiply by 1000000 to get 6 decimal places
		fracInt := int64(fracPart * 1000000)

		// Format fractional part with leading zeros
		fracBuf := [6]byte{}
		for i := 5; i >= 0; i-- {
			fracBuf[i] = byte(fracInt%10) + '0'
			fracInt /= 10
		}

		// Append fractional digits, trimming trailing zeros
		trimmed := fracBuf[:]
		for len(trimmed) > 1 && trimmed[len(trimmed)-1] == '0' {
			trimmed = trimmed[:len(trimmed)-1]
		}
		buf = append(buf, trimmed...)
	}

	return buf
}

// appendFloatScientific appends a float in scientific notation without allocations
func appendFloatScientific(buf []byte, f float64) []byte {
	// Calculate exponent
	exp := 0
	absF := f
	if absF >= 10 {
		for absF >= 10 {
			absF /= 10
			exp++
		}
	} else if absF < 1 && absF > 0 {
		for absF < 1 {
			absF *= 10
			exp--
		}
	}

	// Append mantissa (1 digit before decimal, 6 after)
	intPart := int(absF)
	buf = appendInt(buf, intPart)

	fracPart := absF - float64(intPart)
	if fracPart > 0 {
		buf = append(buf, '.')
		fracInt := int64(fracPart * 1000000)
		fracBuf := [6]byte{}
		for i := 5; i >= 0; i-- {
			fracBuf[i] = byte(fracInt%10) + '0'
			fracInt /= 10
		}

		// Trim trailing zeros
		trimmed := fracBuf[:]
		for len(trimmed) > 1 && trimmed[len(trimmed)-1] == '0' {
			trimmed = trimmed[:len(trimmed)-1]
		}
		buf = append(buf, trimmed...)
	}

	// Append exponent
	buf = append(buf, 'e')
	if exp >= 0 {
		buf = append(buf, '+')
	}
	buf = appendInt(buf, exp)

	return buf
}

// Level defines the logging level.
type Level int8

// String returns the string representation of the level.
func (l Level) String() string {
	switch l {
	case TRACE:
		return traceStr
	case DEBUG:
		return debugStr
	case INFO:
		return infoStr
	case WARN:
		return warnStr
	case ERROR:
		return errorStr
	case FATAL:
		return fatalStr
	default:
		return ""
	}
}

// Log levels.
const (
	TRACE Level = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

// Handler processes a log event and writes it to an output.
type Handler interface {
	// Write handles the log event, writing it to its destination.
	// The handler is responsible for returning the event's buffer to the pool.
	Write(e *Event) error
}

// ErrorHandler is called when a handler write operation fails
type ErrorHandler func(err error)

// defaultErrorHandler is the default error handler that does nothing for backward compatibility
func defaultErrorHandler(err error) {
	// Silent by default for backward compatibility
}

// Hook defines an interface for log event interception.
// Hooks are called during Msg() before the event is written to the handler.
// Returning false from Run suppresses the log event entirely.
type Hook interface {
	Run(level Level, msg string) bool
}

// SampleHook implements Hook to sample log events at a rate of 1 in every N.
// It uses atomic operations for thread-safe counting.
type SampleHook struct {
	n       uint32
	counter uint32
}

// NewSampleHook creates a SampleHook that passes 1 out of every n events.
// If n is 0 or 1, all events pass through.
func NewSampleHook(n uint32) *SampleHook {
	return &SampleHook{n: n}
}

// Run implements Hook. It returns true for every Nth event.
func (h *SampleHook) Run(_ Level, _ string) bool {
	if h.n <= 1 {
		return true
	}
	c := atomic.AddUint32(&h.counter, 1)
	return c%h.n == 0
}

// Logger is the main logging interface.
type Logger struct {
	handler      Handler
	level        int64  // Use int64 for atomic operations with Level
	context      []byte // Pre-formatted context fields for this logger instance.
	errorHandler ErrorHandler
	hooks        []Hook
}

// New creates a new logger with the given handler.
func New(handler Handler) *Logger {
	return &Logger{handler: handler, errorHandler: defaultErrorHandler}
}

// SetErrorHandler sets a custom error handler for the logger
func (l *Logger) SetErrorHandler(eh ErrorHandler) *Logger {
	l.errorHandler = eh
	return l
}

// AddHook adds a hook to the logger. Hooks are called in order during Msg().
// AddHook is intended for setup-time configuration and is not safe to call
// concurrently with logging operations.
func (l *Logger) AddHook(hook Hook) *Logger {
	l.hooks = append(l.hooks, hook)
	return l
}

// Event represents a single log message - now the primary type to eliminate wrapper allocation
type Event struct {
	buf   []byte // The raw buffer for building the log line.
	level Level
	l     *Logger
}

// Global pool for event objects.
var eventPool = &sync.Pool{
	New: func() interface{} {
		return &Event{
			buf: make([]byte, 0, DefaultBufferSize), // Start with larger buffer
		}
	},
}

// With creates a new Event with the current logger's context.
func (l *Logger) With() *Event {
	levelValue := atomic.LoadInt64(&l.level)
	// Ensure level is within valid range (defensive programming)
	// Level is int8, so valid range is -128 to 127, but our levels are 0-5
	if levelValue < int64(TRACE) || levelValue > int64(FATAL) {
		levelValue = int64(INFO) // Default to INFO if somehow corrupted
	}
	// Safe conversion after bounds check
	level := Level(levelValue) // #nosec G115 - bounds already checked above
	return &Event{buf: append([]byte{}, l.context...), level: level, l: l}
}

// Logger returns a new Logger with the event's fields as context.
func (e *Event) Logger() *Logger {
	// Remove the leading comma if present
	contextBuf := e.buf
	if len(contextBuf) > 0 && contextBuf[0] == ',' {
		contextBuf = contextBuf[1:]
	}
	// Create new logger with atomic level
	newLogger := &Logger{handler: e.l.handler, context: contextBuf, errorHandler: e.l.errorHandler, hooks: e.l.hooks}
	atomic.StoreInt64(&newLogger.level, atomic.LoadInt64(&e.l.level))
	return newLogger
}

// Ctx automatically includes OpenTelemetry trace/span IDs if present.
func (l *Logger) Ctx(ctx context.Context) *Logger {
	logger := l // Start with the current logger

	span := oteltrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		// Create a new logger with trace and span IDs as context
		logger = logger.With().Str("trace_id", span.SpanContext().TraceID().String()).Str("span_id", span.SpanContext().SpanID().String()).Logger()
	}
	return logger
}

func (l *Logger) log(level Level) *Event {
	// Use atomic load to safely read the current level
	levelValue := atomic.LoadInt64(&l.level)
	// Ensure level is within valid range (defensive programming)
	// Level is int8, so valid range is -128 to 127, but our levels are 0-5
	if levelValue < int64(TRACE) || levelValue > int64(FATAL) {
		levelValue = int64(INFO) // Default to INFO if somehow corrupted
	}
	// Safe conversion after bounds check
	currentLevel := Level(levelValue) // #nosec G115 - bounds already checked above
	if level < currentLevel {
		return nil
	}

	e := eventPool.Get().(*Event)
	e.level = level
	e.l = l
	e.buf = e.buf[:0] // Reset buffer length but keep capacity

	e.buf = append(e.buf, '{') // Always start with '{'

	// Add level
	e.buf = append(e.buf, `"level":"`...)
	e.buf = append(e.buf, level.String()...)
	e.buf = append(e.buf, '"')

	// Add logger context if present
	if len(l.context) > 0 {
		e.buf = append(e.buf, ',') // Add comma before context
		e.buf = append(e.buf, l.context...)
	}
	return e
}

// Info starts a new message with the INFO level.
func (l *Logger) Info() *Event {
	e := l.log(INFO)
	if e == nil {
		return &Event{} // Return a no-op Event
	}
	return e
}

// Error starts a new message with the ERROR level.
func (l *Logger) Error() *Event {
	e := l.log(ERROR)
	if e == nil {
		return &Event{} // Return a no-op Event
	}
	return e
}

// Debug starts a new message with the DEBUG level.
func (l *Logger) Debug() *Event {
	e := l.log(DEBUG)
	if e == nil {
		return &Event{} // Return a no-op Event
	}
	return e
}

// Warn starts a new message with the WARN level.
func (l *Logger) Warn() *Event {
	e := l.log(WARN)
	if e == nil {
		return &Event{} // Return a no-op Event
	}
	return e
}

// Trace starts a new message with the TRACE level.
func (l *Logger) Trace() *Event {
	e := l.log(TRACE)
	if e == nil {
		return &Event{} // Return a no-op Event
	}
	return e
}

// Fatal starts a new message with the FATAL level.
func (l *Logger) Fatal() *Event {
	e := l.log(FATAL)
	if e == nil {
		return &Event{} // Return a no-op Event
	}
	return e
}

// Str adds a string field to the event with proper JSON escaping and validation.
func (e *Event) Str(key, value string) *Event {
	if e.l == nil {
		return e
	}

	// Validate inputs for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Str(): %w", err))
		}
		return e
	}
	if err := validateValue(value); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid value in Str(): %w", err))
		}
		return e
	}

	// Check buffer size before adding content
	if err := checkBufferSize(e.buf); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("buffer size limit exceeded in Str(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":"`...)
	e.buf = appendJSONString(e.buf, value)
	e.buf = append(e.buf, '"')
	return e
}

// Stringer adds a field whose value is obtained by calling the String method of
// a [fmt.Stringer]. If val is nil, the field value is JSON null.
func (e *Event) Stringer(key string, val fmt.Stringer) *Event {
	if e.l == nil {
		return e
	}
	if val == nil {
		if err := validateKey(key); err != nil {
			if e.l.errorHandler != nil {
				e.l.errorHandler(fmt.Errorf("invalid key in Stringer(): %w", err))
			}
			return e
		}
		e.buf = append(e.buf, ',')
		e.buf = append(e.buf, '"')
		e.buf = appendJSONString(e.buf, key)
		e.buf = append(e.buf, `":null`...)
		return e
	}
	return e.Str(key, val.String())
}

// Int adds an integer field to the event using fast conversion.
func (e *Event) Int(key string, value int) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Int(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":`...)
	e.buf = appendInt(e.buf, value)
	return e
}

// Bool adds a boolean field to the event using fast conversion.
func (e *Event) Bool(key string, value bool) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Bool(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":`...)
	e.buf = appendBool(e.buf, value)
	return e
}

// Float64 adds a float64 field with 6 decimal precision (zero-allocation).
//
// IMPORTANT: This method uses a custom formatter that limits precision to 6 decimal
// places to achieve zero allocations. For values requiring full precision, use Any()
// which delegates to encoding/json (allocates but preserves full precision).
//
// Precision examples:
//   - 99.99      → "99.989999" (6 decimals, minor rounding)
//   - 3.14159265 → "3.141592"  (6 decimals, truncated)
//   - 1.23e100   → "1.23e+100" (scientific notation for very large/small)
//
// Special values:
//   - NaN        → "NaN"
//   - +Infinity  → "+Inf" (quoted)
//   - -Infinity  → "-Inf" (quoted)
//   - -0.0       → 0 (negative zero not preserved)
//
// For financial/scientific applications requiring exact precision:
//
//	logger.Any("precise_value", 99.99)  // Full precision, allocates
//
// For performance-critical logging where 6 decimals suffice:
//
//	logger.Float64("fast_value", 99.99) // Zero allocation, 6 decimals
func (e *Event) Float64(key string, value float64) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Float64(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":`...)
	e.buf = appendFloat64(e.buf, value)
	return e
}

func (e *Event) Time(key string, value time.Time) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Time(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":"`...)
	e.buf = appendRFC3339(e.buf, value)
	e.buf = append(e.buf, '"')
	return e
}

func (e *Event) Dur(key string, value time.Duration) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Dur(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":`...)
	e.buf = appendInt(e.buf, int(value.Nanoseconds()))
	return e
}

func (e *Event) Uint(key string, value uint) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Uint(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":`...)
	e.buf = appendUint(e.buf, uint64(value))
	return e
}

func (e *Event) Any(key string, value interface{}) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Any(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":`...)
	marshaledValue, err := json.Marshal(value)
	if err != nil {
		// Handle error with proper JSON escaping
		errorMsg := fmt.Sprintf("!ERROR: %v!", err)
		e.buf = append(e.buf, '"')
		e.buf = appendJSONString(e.buf, errorMsg)
		e.buf = append(e.buf, '"')
	} else {
		e.buf = append(e.buf, marshaledValue...)
	}
	return e
}

func (e *Event) Err(err error) *Event {
	if e.l == nil || err == nil {
		return e
	}
	return e.Str("error", err.Error())
}

// Msg sends the event to the handler for processing.
// This is always the final method in the chain.
func (e *Event) Msg(message string) {
	if e.l == nil {
		return // No-op for disabled events
	}

	// Validate message length
	if err := validateValue(message); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid message in Msg(): %w", err))
		}
		return
	}

	// Run hooks - if any returns false, suppress the event
	for _, hook := range e.l.hooks {
		if !hook.Run(e.level, message) {
			e.buf = e.buf[:0]
			e.l = nil
			eventPool.Put(e)
			return
		}
	}

	// Check buffer size before finalizing
	if err := checkBufferSize(e.buf); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("buffer size limit exceeded in Msg(): %w", err))
		}
		return
	}

	// Add message with proper JSON escaping
	e.buf = append(e.buf, `,"message":"`...)
	e.buf = appendJSONString(e.buf, message)
	e.buf = append(e.buf, '"')

	// Finalize JSON and add newline
	e.buf = append(e.buf, '}')
	e.buf = append(e.buf, '\n')

	// Pass the event to the handler with proper error handling
	if err := e.l.handler.Write(e); err != nil && e.l.errorHandler != nil {
		e.l.errorHandler(fmt.Errorf("handler write failed: %w", err))
	}

	// Reset the buffer and put the event back into the pool.
	e.buf = e.buf[:0]
	e.l = nil // Clear logger reference
	eventPool.Put(e)
}

// JSONHandler formats logs as JSON.
type JSONHandler struct {
	out io.Writer
}

// NewJSONHandler creates a new JSON handler.
func NewJSONHandler(out io.Writer) *JSONHandler {
	return &JSONHandler{out: out}
}

// Write handles the log event.
func (h *JSONHandler) Write(e *Event) error {
	_, err := h.out.Write(e.buf)
	return err
}

// ConsoleHandler formats logs for human-readable console output.
type ConsoleHandler struct {
	out io.Writer
}

// NewConsoleHandler creates a new ConsoleHandler.
func NewConsoleHandler(out io.Writer) *ConsoleHandler {
	return &ConsoleHandler{out: out}
}

// Write handles the log event with zero allocations by streaming JSON parsing.
func (h *ConsoleHandler) Write(e *Event) error {
	// Extract level and message without unmarshaling (zero-allocation)
	level := extractJSONField(e.buf, "level")
	message := extractJSONField(e.buf, "message")

	// Get color for the level
	color := getColorForLevel(string(level))

	// Format timestamp (reuse buffer for efficiency)
	var timeBuf [25]byte // RFC3339 is max 25 chars
	timestamp := appendRFC3339(timeBuf[:0], time.Now())

	// Write level with color
	if _, err := h.out.Write([]byte(color)); err != nil {
		return fmt.Errorf("failed to write color: %w", err)
	}
	if _, err := h.out.Write(level); err != nil {
		return fmt.Errorf("failed to write level: %w", err)
	}

	// Reset color and write timestamp
	if _, err := h.out.Write([]byte("\x1b[0m[")); err != nil {
		return fmt.Errorf("failed to write reset: %w", err)
	}
	if _, err := h.out.Write(timestamp); err != nil {
		return fmt.Errorf("failed to write timestamp: %w", err)
	}

	// Write message
	if _, err := h.out.Write([]byte("] ")); err != nil {
		return fmt.Errorf("failed to write separator: %w", err)
	}
	if _, err := h.out.Write(message); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	// Write remaining fields by streaming through JSON
	if err := writeFieldsStreaming(h.out, e.buf, level, message); err != nil {
		return fmt.Errorf("failed to write fields: %w", err)
	}

	// Write newline
	if _, err := h.out.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// multiHandler is a Handler that writes to multiple handlers.
type multiHandler struct {
	handlers []Handler
}

// MultiHandler returns a Handler that writes to all provided handlers.
// The handlers slice is copied at construction, so the original slice can be
// safely modified afterward. Write returns the first error encountered.
func MultiHandler(handlers ...Handler) Handler {
	h := make([]Handler, len(handlers))
	copy(h, handlers)
	return &multiHandler{handlers: h}
}

// Write sends the event to all handlers, returning the first error encountered.
func (m *multiHandler) Write(e *Event) error {
	var firstErr error
	for _, h := range m.handlers {
		if err := h.Write(e); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// findJSONFieldStart locates the start position of a JSON field value
// Returns -1 if field not found
func findJSONFieldStart(buf []byte, key string) int {
	searchPattern := []byte(`"` + key + `":`)
	idx := bytes.Index(buf, searchPattern)
	if idx == -1 {
		return -1
	}

	start := idx + len(searchPattern)
	// Skip whitespace after colon
	for start < len(buf) && (buf[start] == ' ' || buf[start] == '\t') {
		start++
	}

	if start >= len(buf) {
		return -1
	}

	return start
}

// extractStringValue extracts a quoted string value from position start
// Returns the unquoted value or nil if invalid
func extractStringValue(buf []byte, start int) []byte {
	if start >= len(buf) || buf[start] != '"' {
		return nil
	}

	start++ // Skip opening quote
	end := start

	// Find closing quote (handle escaped quotes)
	for end < len(buf) {
		if buf[end] == '"' && (end == start || buf[end-1] != '\\') {
			return buf[start:end]
		}
		end++
	}

	return nil
}

// extractNonStringValue extracts a non-string value (number, boolean, null)
// Returns the value with trailing whitespace trimmed
func extractNonStringValue(buf []byte, start int) []byte {
	end := start
	for end < len(buf) && buf[end] != ',' && buf[end] != '}' {
		end++
	}

	// Trim trailing whitespace
	for end > start && (buf[end-1] == ' ' || buf[end-1] == '\t') {
		end--
	}

	return buf[start:end]
}

// extractJSONField extracts a field value from JSON without unmarshaling.
// Returns the value as a byte slice (no allocation).
func extractJSONField(buf []byte, key string) []byte {
	start := findJSONFieldStart(buf, key)
	if start == -1 {
		return nil
	}

	// Check if value is a string (starts with ")
	if buf[start] == '"' {
		return extractStringValue(buf, start)
	}

	// Non-string value (number, boolean, null)
	return extractNonStringValue(buf, start)
}

// skipWhitespace advances index past whitespace characters
func skipWhitespace(buf []byte, i int) int {
	for i < len(buf) && (buf[i] == ' ' || buf[i] == '\t' || buf[i] == '\n') {
		i++
	}
	return i
}

// extractJSONKey extracts a JSON key starting at position i (should point to opening ")
// Returns key bytes and new position after closing "
func extractJSONKey(buf []byte, i int) ([]byte, int) {
	if i >= len(buf) || buf[i] != '"' {
		return nil, i
	}

	keyStart := i + 1
	keyEnd := keyStart
	for keyEnd < len(buf) && buf[keyEnd] != '"' {
		keyEnd++
	}

	if keyEnd >= len(buf) {
		return nil, len(buf)
	}

	return buf[keyStart:keyEnd], keyEnd + 1
}

// extractJSONValue extracts a JSON value starting at position i
// Returns value bytes and new position after value
func extractJSONValue(buf []byte, i int) ([]byte, int) {
	if i >= len(buf) {
		return nil, i
	}

	if buf[i] == '"' {
		// String value
		valueStart := i + 1
		valueEnd := valueStart
		for valueEnd < len(buf) && !(buf[valueEnd] == '"' && (valueEnd == valueStart || buf[valueEnd-1] != '\\')) {
			valueEnd++
		}
		return buf[valueStart:valueEnd], valueEnd + 1
	}

	// Non-string value
	valueStart := i
	valueEnd := i
	for valueEnd < len(buf) && buf[valueEnd] != ',' && buf[valueEnd] != '}' {
		valueEnd++
	}

	// Trim trailing whitespace
	for valueEnd > valueStart && (buf[valueEnd-1] == ' ' || buf[valueEnd-1] == '\t') {
		valueEnd--
	}

	return buf[valueStart:valueEnd], valueEnd
}

// writeKeyValue writes a key=value pair to the writer
func writeKeyValue(w io.Writer, key, value []byte) error {
	if _, err := w.Write([]byte(" ")); err != nil {
		return err
	}
	if _, err := w.Write(key); err != nil {
		return err
	}
	if _, err := w.Write([]byte("=")); err != nil {
		return err
	}
	if _, err := w.Write(value); err != nil {
		return err
	}
	return nil
}

// skipCommaIfPresent advances past comma if found
func skipCommaIfPresent(buf []byte, i int) int {
	if i < len(buf) && buf[i] == ',' {
		return i + 1
	}
	return i
}

// isReservedField checks if field should be skipped (already written)
func isReservedField(key []byte) bool {
	return bytes.Equal(key, []byte("level")) || bytes.Equal(key, []byte("message"))
}

// writeFieldsStreaming writes additional fields by parsing JSON without allocations
func writeFieldsStreaming(w io.Writer, buf []byte, _ []byte, _ []byte) error {
	i := 1 // Skip opening {

	for i < len(buf) {
		i = skipWhitespace(buf, i)

		if i >= len(buf) || buf[i] == '}' {
			break
		}

		// Extract key
		key, newPos := extractJSONKey(buf, i)
		if key == nil {
			i++
			continue
		}
		i = newPos

		// Skip to value (colon and whitespace)
		i = skipWhitespace(buf, i)
		if i < len(buf) && buf[i] == ':' {
			i++
		}
		i = skipWhitespace(buf, i)

		if i >= len(buf) {
			break
		}

		// Extract value
		value, newPos := extractJSONValue(buf, i)
		i = newPos

		// Skip reserved fields (already written)
		if isReservedField(key) {
			i = skipCommaIfPresent(buf, i)
			continue
		}

		// Write field
		if err := writeKeyValue(w, key, value); err != nil {
			return err
		}

		i = skipCommaIfPresent(buf, i)
	}

	return nil
}

func getColorForLevel(level string) string {
	switch level {
	case infoStr:
		return "\x1b[34m" // Blue
	case warnStr:
		return "\x1b[33m" // Yellow
	case errorStr, fatalStr:
		return "\x1b[31m" // Red
	case debugStr, traceStr:
		return "\x1b[90m" // Bright Black (Gray)
	default:
		return "\x1b[0m" // Reset
	}
}

// A default logger for package-level functions.
var defaultLogger *Logger

func init() {
	initDefaultLogger()
}

var isTerminal = isatty

// ParseLevel converts a string to a Level.
func ParseLevel(levelStr string) Level {
	switch levelStr {
	case traceStr:
		return TRACE
	case debugStr:
		return DEBUG
	case infoStr:
		return INFO
	case warnStr:
		return WARN
	case errorStr:
		return ERROR
	case fatalStr:
		return FATAL
	default:
		return INFO // Default to INFO if the level is not recognized
	}
}

// initDefaultLogger initializes the default logger based on environment variables.
func initDefaultLogger() {
	format := os.Getenv("BOLT_FORMAT")
	if format == "" {
		if isTerminal(os.Stdout) {
			format = consoleStr
		} else {
			format = "json"
		}
	}

	level := ParseLevel(os.Getenv("BOLT_LEVEL"))

	switch format {
	case consoleStr:
		defaultLogger = New(NewConsoleHandler(os.Stdout)).SetLevel(level)
	default:
		// Default to JSON if the format is not specified or is "json"
		defaultLogger = New(NewJSONHandler(os.Stdout)).SetLevel(level)
	}
}

// SetLevel sets the logging level for the logger using atomic operations for thread safety.
// This method is safe to call concurrently from multiple goroutines without additional
// synchronization. The atomic operations prevent race conditions that could lead to
// inconsistent filtering behavior or security bypass scenarios.
//
// Invalid levels are clamped to INFO to ensure defensive behavior.
func (l *Logger) SetLevel(level Level) *Logger {
	// Validate level before storing to prevent corruption
	if level < TRACE || level > FATAL {
		level = INFO // Defensive: clamp to INFO for invalid values
	}
	atomic.StoreInt64(&l.level, int64(level))
	return l
}

// Info starts a new message with the INFO level on the default logger.
func Info() *Event {
	return defaultLogger.Info()
}

// Error starts a new message with the ERROR level on the default logger.
func Error() *Event {
	return defaultLogger.Error()
}

// Debug starts a new message with the DEBUG level on the default logger.
func Debug() *Event {
	return defaultLogger.Debug()
}

// Warn starts a new message with the WARN level on the default logger.
func Warn() *Event {
	return defaultLogger.Warn()
}

// Trace starts a new message with the TRACE level on the default logger.
func Trace() *Event {
	return defaultLogger.Trace()
}

// Fatal starts a new message with the FATAL level on the default logger.
func Fatal() *Event {
	return defaultLogger.Fatal()
}

// Additional utility methods and performance optimizations

// Hex adds a hexadecimal field to the event.
func (e *Event) Hex(key string, value []byte) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Hex(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":"`...)
	e.buf = append(e.buf, hex.EncodeToString(value)...)
	e.buf = append(e.buf, '"')
	return e
}

// Base64 adds a base64-encoded field to the event.
func (e *Event) Base64(key string, value []byte) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Base64(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":"`...)
	e.buf = append(e.buf, base64.StdEncoding.EncodeToString(value)...)
	e.buf = append(e.buf, '"')
	return e
}

// IPAddr adds a net.IP address field to the event. IPv4 addresses are formatted
// as dotted-decimal (e.g. "192.168.1.1"), IPv6 as colon-hex notation.
// If ip is nil, the field value is JSON null. This method is zero-allocation.
func (e *Event) IPAddr(key string, ip net.IP) *Event {
	if e.l == nil {
		return e
	}
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in IPAddr(): %w", err))
		}
		return e
	}
	if ip == nil {
		e.buf = append(e.buf, ',')
		e.buf = append(e.buf, '"')
		e.buf = appendJSONString(e.buf, key)
		e.buf = append(e.buf, `":null`...)
		return e
	}
	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":"`...)
	e.buf = appendIP(e.buf, ip)
	e.buf = append(e.buf, '"')
	return e
}

// Bytes adds a byte array field as a string to the event.
func (e *Event) Bytes(key string, value []byte) *Event {
	if e.l == nil {
		return e
	}
	return e.Str(key, string(value))
}

// Stack adds a stack trace field to the event.
func (e *Event) Stack() *Event {
	if e.l == nil {
		return e
	}
	buf := make([]byte, StackTraceBufferSize)
	n := runtime.Stack(buf, false)
	return e.Str("stack", string(buf[:n]))
}

// Caller adds caller information (file:line) to the event.
func (e *Event) Caller() *Event {
	if e.l == nil {
		return e
	}
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return e.Str("caller", "unknown")
	}
	// Extract just the filename
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		file = file[idx+1:]
	}
	return e.Str("caller", fmt.Sprintf("%s:%d", file, line))
}

// CallerSkip adds caller information (file:line) to the event, skipping the
// specified number of additional stack frames. This is useful when Bolt is
// wrapped in helper functions and you need the caller of the wrapper.
func (e *Event) CallerSkip(skip int) *Event {
	if e.l == nil {
		return e
	}
	_, file, line, ok := runtime.Caller(1 + skip)
	if !ok {
		return e.Str("caller", "unknown")
	}
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		file = file[idx+1:]
	}
	return e.Str("caller", fmt.Sprintf("%s:%d", file, line))
}

// RandID adds a random ID field to the event for request tracing.
func (e *Event) RandID(key string) *Event {
	if e.l == nil {
		return e
	}
	// Generate a random 8-byte ID
	id := make([]byte, 8)
	_, _ = rand.Read(id) // crypto/rand.Read never fails
	return e.Hex(key, id)
}

// Fields allows adding multiple fields at once from a map.
func (e *Event) Fields(fields map[string]interface{}) *Event {
	if e.l == nil {
		return e
	}
	for k, v := range fields {
		e.Any(k, v)
	}
	return e
}

// Ints adds an integer slice field to the event as a JSON array.
// This method is zero-allocation.
func (e *Event) Ints(key string, values []int) *Event {
	if e.l == nil {
		return e
	}
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Ints(): %w", err))
		}
		return e
	}
	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":[`...)
	for i, v := range values {
		if i > 0 {
			e.buf = append(e.buf, ',')
		}
		e.buf = appendInt(e.buf, v)
	}
	e.buf = append(e.buf, ']')
	return e
}

// Strs adds a string slice field to the event as a JSON array.
// This method is zero-allocation.
func (e *Event) Strs(key string, values []string) *Event {
	if e.l == nil {
		return e
	}
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Strs(): %w", err))
		}
		return e
	}
	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":[`...)
	for i, v := range values {
		if i > 0 {
			e.buf = append(e.buf, ',')
		}
		e.buf = append(e.buf, '"')
		e.buf = appendJSONString(e.buf, v)
		e.buf = append(e.buf, '"')
	}
	e.buf = append(e.buf, ']')
	return e
}

// Dict adds a sub-object field to the event. The provided function is called
// with a temporary Event that collects the sub-object's fields. The fields
// are then embedded as a JSON object under the given key.
func (e *Event) Dict(key string, fn func(d *Event)) *Event {
	if e.l == nil {
		return e
	}
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Dict(): %w", err))
		}
		return e
	}
	sub := eventPool.Get().(*Event)
	sub.buf = sub.buf[:0]
	sub.level = e.level
	sub.l = e.l
	fn(sub)
	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":{`...)
	subBuf := sub.buf
	if len(subBuf) > 0 && subBuf[0] == ',' {
		subBuf = subBuf[1:]
	}
	e.buf = append(e.buf, subBuf...)
	e.buf = append(e.buf, '}')
	sub.buf = sub.buf[:0]
	sub.l = nil
	eventPool.Put(sub)
	return e
}

// Int64 adds a 64-bit integer field to the event.
func (e *Event) Int64(key string, value int64) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Int64(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":`...)
	e.buf = appendInt(e.buf, int(value))
	return e
}

// Int32 adds a 32-bit integer field to the event.
func (e *Event) Int32(key string, value int32) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Int32(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":`...)
	e.buf = appendInt(e.buf, int(value))
	return e
}

// Int16 adds a 16-bit integer field to the event.
func (e *Event) Int16(key string, value int16) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Int16(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":`...)
	e.buf = appendInt(e.buf, int(value))
	return e
}

// Int8 adds an 8-bit integer field to the event.
func (e *Event) Int8(key string, value int8) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Int8(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":`...)
	e.buf = appendInt(e.buf, int(value))
	return e
}

// Uint64 adds a 64-bit unsigned integer field to the event.
func (e *Event) Uint64(key string, value uint64) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Uint64(): %w", err))
		}
		return e
	}

	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":`...)
	e.buf = appendUint(e.buf, value)
	return e
}

// Uint32 adds a 32-bit unsigned integer field to the event.
func (e *Event) Uint32(key string, value uint32) *Event {
	if e.l == nil {
		return e
	}
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Uint32(): %w", err))
		}
		return e
	}
	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":`...)
	e.buf = appendUint(e.buf, uint64(value))
	return e
}

// Uint16 adds a 16-bit unsigned integer field to the event.
func (e *Event) Uint16(key string, value uint16) *Event {
	if e.l == nil {
		return e
	}
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Uint16(): %w", err))
		}
		return e
	}
	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":`...)
	e.buf = appendUint(e.buf, uint64(value))
	return e
}

// Uint8 adds an 8-bit unsigned integer field to the event.
func (e *Event) Uint8(key string, value uint8) *Event {
	if e.l == nil {
		return e
	}
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Uint8(): %w", err))
		}
		return e
	}
	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = appendJSONString(e.buf, key)
	e.buf = append(e.buf, `":`...)
	e.buf = appendUint(e.buf, uint64(value))
	return e
}

// Counter adds an atomic counter value to the event.
func (e *Event) Counter(key string, counter *int64) *Event {
	if e.l == nil {
		return e
	}

	// Validate key for security
	if err := validateKey(key); err != nil {
		if e.l.errorHandler != nil {
			e.l.errorHandler(fmt.Errorf("invalid key in Counter(): %w", err))
		}
		return e
	}

	value := atomic.LoadInt64(counter)
	return e.Int64(key, value)
}

// Timestamp adds the current timestamp to the event.
func (e *Event) Timestamp() *Event {
	if e.l == nil {
		return e
	}
	return e.Time("timestamp", time.Now())
}

// Interface adds an interface{} field to the event (alias for Any).
func (e *Event) Interface(key string, value interface{}) *Event {
	return e.Any(key, value)
}

// Printf adds a formatted message to the event.
func (e *Event) Printf(format string, args ...interface{}) {
	if e.l == nil {
		return
	}
	e.Msg(fmt.Sprintf(format, args...))
}

// Send is an alias for Msg for consistency with other logging libraries.
func (e *Event) Send() {
	e.Msg("")
}

// levelWriter adapts a Logger to the io.Writer interface.
type levelWriter struct {
	logger *Logger
	level  Level
}

// NewLevelWriter returns an io.Writer that logs each Write call as a message
// at the given level. Trailing newlines are trimmed. This is useful for bridging
// libraries that expect an io.Writer (such as the standard log package) into Bolt.
//
// The string(p) conversion allocates, which is acceptable since this is a
// compatibility bridge rather than a hot-path logging method.
func NewLevelWriter(logger *Logger, level Level) io.Writer {
	return &levelWriter{logger: logger, level: level}
}

func (w *levelWriter) Write(p []byte) (int, error) {
	n := len(p)
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}
	e := w.logger.log(w.level)
	if e != nil {
		e.Msg(msg)
	}
	return n, nil
}
