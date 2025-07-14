package logma

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	oteltrace "go.opentelemetry.io/otel/trace"
)

// Constants for buffer sizes and configuration
const (
	// DefaultBufferSize is the initial buffer size for events - increased to reduce reallocations
	DefaultBufferSize = 2048
	// StackTraceBufferSize is the buffer size for stack traces
	StackTraceBufferSize = 64 * 1024 // 64KB
	// DefaultFilePermissions for log files
	DefaultFilePermissions = 0644
)

// Fast number conversion helpers to avoid allocations
var digits = "0123456789"

// appendInt appends an integer to the buffer without allocations
func appendInt(buf []byte, i int) []byte {
	if i == 0 {
		return append(buf, '0')
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

// appendLevel appends a level string to the buffer without allocations
func appendLevel(buf []byte, level Level) []byte {
	switch level {
	case TRACE:
		return append(buf, "trace"...)
	case DEBUG:
		return append(buf, "debug"...)
	case INFO:
		return append(buf, "info"...)
	case WARN:
		return append(buf, "warn"...)
	case ERROR:
		return append(buf, "error"...)
	case DPANIC:
		return append(buf, "dpanic"...)
	case PANIC:
		return append(buf, "panic"...)
	case FATAL:
		return append(buf, "fatal"...)
	default:
		return buf
	}
}

// appendRFC3339Time appends RFC3339 formatted time to buffer without allocations
func appendRFC3339Time(buf []byte, t time.Time) []byte {
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	nsec := t.Nanosecond()
	
	// Format: 2006-01-02T15:04:05.999999999Z07:00
	buf = appendUint(buf, uint64(year))
	buf = append(buf, '-')
	if month < 10 {
		buf = append(buf, '0')
	}
	buf = appendUint(buf, uint64(month))
	buf = append(buf, '-')
	if day < 10 {
		buf = append(buf, '0')
	}
	buf = appendUint(buf, uint64(day))
	buf = append(buf, 'T')
	if hour < 10 {
		buf = append(buf, '0')
	}
	buf = appendUint(buf, uint64(hour))
	buf = append(buf, ':')
	if min < 10 {
		buf = append(buf, '0')
	}
	buf = appendUint(buf, uint64(min))
	buf = append(buf, ':')
	if sec < 10 {
		buf = append(buf, '0')
	}
	buf = appendUint(buf, uint64(sec))
	
	// Add nanoseconds if present (simplified to avoid allocations)
	if nsec != 0 {
		buf = append(buf, '.')
		// Convert to microseconds for simpler formatting (6 digits)
		microsec := nsec / 1000
		if microsec < 100000 {
			buf = append(buf, '0')
		}
		if microsec < 10000 {
			buf = append(buf, '0')
		}
		if microsec < 1000 {
			buf = append(buf, '0')
		}
		if microsec < 100 {
			buf = append(buf, '0')
		}
		if microsec < 10 {
			buf = append(buf, '0')
		}
		buf = appendUint(buf, uint64(microsec))
	}
	
	// Add timezone
	_, offset := t.Zone()
	if offset == 0 {
		buf = append(buf, 'Z')
	} else {
		// Convert offset to hours and minutes
		if offset < 0 {
			buf = append(buf, '-')
			offset = -offset
		} else {
			buf = append(buf, '+')
		}
		hours := offset / 3600
		minutes := (offset % 3600) / 60
		
		if hours < 10 {
			buf = append(buf, '0')
		}
		buf = appendUint(buf, uint64(hours))
		buf = append(buf, ':')
		if minutes < 10 {
			buf = append(buf, '0')
		}
		buf = appendUint(buf, uint64(minutes))
	}
	
	return buf
}

// ContextExtractor is a function that extracts key-value pairs from a context.
type ContextExtractor func(ctx context.Context) map[string]interface{}

// contextExtractors holds registered context extractors.
var (
	contextExtractors   []ContextExtractor
	contextExtractorsMu sync.RWMutex
)

// Level defines the logging level.
type Level int8

// String returns the string representation of the level.
func (l Level) String() string {
	switch l {
	case TRACE:
		return "trace"
	case DEBUG:
		return "debug"
	case INFO:
		return "info"
	case WARN:
		return "warn"
	case ERROR:
		return "error"
	case DPANIC:
		return "dpanic"
	case PANIC:
		return "panic"
	case FATAL:
		return "fatal"
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
	DPANIC
	PANIC
	FATAL
)

// Handler processes a log event and writes it to an output.
type Handler interface {
	// Write handles the log event, writing it to its destination.
	// The handler is responsible for returning the event's buffer to the pool.
	Write(e *Event) error
}

// Sampler determines whether a log entry should be logged.
type Sampler interface {
	// Sample returns true if the log entry should be logged.
	Sample() bool
}

// FixedSampler samples every Nth log entry.
type FixedSampler struct {
	counter uint64
	nth     uint64
}

// NewFixedSampler creates a sampler that logs every nth entry.
func NewFixedSampler(n uint64) *FixedSampler {
	return &FixedSampler{nth: n}
}

// Sample implements the Sampler interface.
func (s *FixedSampler) Sample() bool {
	return atomic.AddUint64(&s.counter, 1)%s.nth == 0
}

// RandomSampler samples log entries randomly based on probability.
type RandomSampler struct {
	probability float64
}

// NewRandomSampler creates a sampler that logs entries with given probability (0.0 to 1.0).
func NewRandomSampler(probability float64) *RandomSampler {
	return &RandomSampler{probability: probability}
}

// Sample implements the Sampler interface.
func (s *RandomSampler) Sample() bool {
	return rand.Float64() < s.probability
}

// LevelSampler applies different sampling rates based on log level.
type LevelSampler struct {
	samplers map[Level]Sampler
}

// NewLevelSampler creates a level-based sampler.
func NewLevelSampler(samplers map[Level]Sampler) *LevelSampler {
	return &LevelSampler{samplers: samplers}
}

// Sample implements the Sampler interface.
func (s *LevelSampler) Sample(level Level) bool {
	if sampler, exists := s.samplers[level]; exists {
		return sampler.Sample()
	}
	return true // Default to logging if no sampler for this level
}

// Logger is the main logging interface.
type Logger struct {
	handler     Handler
	level       Level
	context     []byte   // Pre-formatted context fields for this logger instance.
	sampler     Sampler  // Optional sampler for this logger
	levelSampler *LevelSampler // Optional level-based sampler
}

// New creates a new logger with the given handler.
// The logger will have the default log level (INFO) until SetLevel is called.
func New(handler Handler) *Logger {
	return &Logger{handler: handler}
}

// Event represents a single log message - now the primary type (eliminating wrapper allocation).
type Event struct {
	buf   []byte // The raw buffer for building the log line.
	level Level
	l     *Logger // Logger reference for access to handler, context, etc.
}

// Global pool for Event objects.
var eventPool = &sync.Pool{
	New: func() interface{} {
		return &Event{
			buf: make([]byte, 0, DefaultBufferSize),
		}
	},
}

// event is now an alias for backward compatibility (if needed)
type event = Event

// appendEscapedString appends a JSON-escaped string to the buffer.
func appendEscapedString(buf []byte, s string) []byte {
	for _, r := range s {
		switch r {
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
			if r < 32 {
				buf = append(buf, '\\', 'u', '0', '0')
				hex := "0123456789abcdef"
				buf = append(buf, hex[r>>4], hex[r&0xF])
			} else {
				buf = append(buf, string(r)...)
			}
		}
	}
	return buf
}

// With creates a new Event with the current logger's context.
func (l *Logger) With() *Event {
	return &Event{buf: append([]byte{}, l.context...), level: l.level, l: l}
}

// Logger returns a new Logger with the event's fields as context.
func (e *Event) Logger() *Logger {
	// Remove the leading comma if present
	contextBuf := e.buf
	if len(contextBuf) > 0 && contextBuf[0] == ',' {
		contextBuf = contextBuf[1:]
	}
	return &Logger{handler: e.l.handler, level: e.l.level, context: contextBuf}
}

// RegisterContextExtractor registers a context extractor function.
// These functions will be called when using WithContext to extract additional fields.
// This function is thread-safe.
func RegisterContextExtractor(extractor ContextExtractor) {
	contextExtractorsMu.Lock()
	defer contextExtractorsMu.Unlock()
	contextExtractors = append(contextExtractors, extractor)
}

// WithContext creates a new logger with fields extracted from the context.
// This includes OpenTelemetry trace/span IDs and any registered context extractors.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	event := l.With()
	
	// Extract OpenTelemetry trace/span IDs
	span := oteltrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		event = event.Str("trace_id", span.SpanContext().TraceID().String()).
			       Str("span_id", span.SpanContext().SpanID().String())
	}
	
	// Extract fields from registered context extractors (with read lock)
	contextExtractorsMu.RLock()
	extractorsCopy := make([]ContextExtractor, len(contextExtractors))
	copy(extractorsCopy, contextExtractors)
	contextExtractorsMu.RUnlock()
	
	for _, extractor := range extractorsCopy {
		fields := extractor(ctx)
		for key, value := range fields {
			event = event.Any(key, value)
		}
	}
	
	return event.Logger()
}

// Ctx automatically includes OpenTelemetry trace/span IDs if present.
// Returns a new Logger with trace_id and span_id fields if the context contains valid OpenTelemetry span data.
// Deprecated: Use WithContext instead for more comprehensive context support.
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
	// Fast path for disabled levels - avoid all allocations
	if level < l.level {
		return nil
	}
	
	// Check sampling - do this early to avoid allocations
	if l.levelSampler != nil {
		if !l.levelSampler.Sample(level) {
			return nil
		}
	} else if l.sampler != nil {
		if !l.sampler.Sample() {
			return nil
		}
	}

	e := eventPool.Get().(*Event)
	e.level = level
	e.l = l

	e.buf = append(e.buf, '{') // Always start with '{'

	// Get encoder config from handler if it's a JSONHandler
	var config EncoderConfig
	if jsonHandler, ok := l.handler.(*JSONHandler); ok {
		config = jsonHandler.config
	} else {
		config = DefaultEncoderConfig()
	}

	// Add timestamp if enabled
	if config.IncludeTime {
		e.buf = append(e.buf, '"')
		e.buf = append(e.buf, config.TimeKey...)
		e.buf = append(e.buf, `":"`...)
		if config.TimeFormat == time.RFC3339 || config.TimeFormat == time.RFC3339Nano {
			// Use allocation-free RFC3339 formatting
			e.buf = appendRFC3339Time(e.buf, time.Now())
		} else {
			// Fallback to standard formatting (may allocate)
			e.buf = append(e.buf, time.Now().Format(config.TimeFormat)...)
		}
		e.buf = append(e.buf, '"')
		e.buf = append(e.buf, ',')
	}

	// Add level
	e.buf = append(e.buf, '"')
	e.buf = append(e.buf, config.LevelKey...)
	e.buf = append(e.buf, `":"`...)
	e.buf = appendLevel(e.buf, level)
	e.buf = append(e.buf, '"')

	// Add caller info if enabled
	if config.IncludeCaller {
		if _, file, line, ok := runtime.Caller(3); ok {
			e.buf = append(e.buf, ',')
			e.buf = append(e.buf, '"')
			e.buf = append(e.buf, config.CallerKey...)
			e.buf = append(e.buf, `":"`...)
			// Extract just the filename for brevity
			if idx := strings.LastIndex(file, "/"); idx >= 0 {
				file = file[idx+1:]
			}
			e.buf = append(e.buf, file...)
			e.buf = append(e.buf, ':')
			e.buf = appendInt(e.buf, line)
			e.buf = append(e.buf, '"')
		}
	}

	// Add logger context if present
	if len(l.context) > 0 {
		e.buf = append(e.buf, ',') // Add comma before context
		e.buf = append(e.buf, l.context...)
	}
	return e
}


// writeFieldStart writes the start of a field (comma and quoted key)
func (e *Event) writeFieldStart(key string) {
	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = append(e.buf, key...)
	e.buf = append(e.buf, `":`...)
}

// writeStringValue writes a quoted string value with escaping if needed
func (e *Event) writeStringValue(value string) {
	e.buf = append(e.buf, '"')
	if needsEscaping(value) {
		e.buf = appendEscapedString(e.buf, value)
	} else {
		e.buf = append(e.buf, value...)
	}
	e.buf = append(e.buf, '"')
}

// writeRawValue writes an unquoted value (for numbers, booleans)
func (e *Event) writeRawValue(value []byte) {
	e.buf = append(e.buf, value...)
}

// noopEvent is a shared instance for disabled log levels to avoid allocations
var noopEvent = &Event{buf: nil, level: 0, l: nil}

// Debug starts a new message with the DEBUG level.
func (l *Logger) Debug() *Event {
	e := l.log(DEBUG)
	if e == nil {
		return noopEvent // Return shared no-op Event
	}
	return e
}

// Trace starts a new message with the TRACE level.
func (l *Logger) Trace() *Event {
	e := l.log(TRACE)
	if e == nil {
		return noopEvent // Return shared no-op Event
	}
	return e
}

// Info starts a new message with the INFO level.
func (l *Logger) Info() *Event {
	e := l.log(INFO)
	if e == nil {
		return noopEvent // Return shared no-op Event
	}
	return e
}

// Warn starts a new message with the WARN level.
func (l *Logger) Warn() *Event {
	e := l.log(WARN)
	if e == nil {
		return noopEvent // Return shared no-op Event
	}
	return e
}

// Error starts a new message with the ERROR level.
func (l *Logger) Error() *Event {
	e := l.log(ERROR)
	if e == nil {
		return noopEvent // Return shared no-op Event
	}
	return e
}

// DPanic starts a new message with the DPANIC level.
// DPanic stands for "development panic" - in development mode, it panics after logging.
func (l *Logger) DPanic() *Event {
	e := l.log(DPANIC)
	if e == nil {
		return noopEvent // Return shared no-op Event
	}
	return e
}

// Panic starts a new message with the PANIC level.
// After logging, it calls panic().
func (l *Logger) Panic() *Event {
	e := l.log(PANIC)
	if e == nil {
		return noopEvent // Return shared no-op Event
	}
	return e
}

// Fatal starts a new message with the FATAL level.
func (l *Logger) Fatal() *Event {
	e := l.log(FATAL)
	if e == nil {
		return noopEvent // Return shared no-op Event
	}
	return e
}

// Str adds a string field to the event.
func (e *Event) Str(key, value string) *Event {
	if e.buf == nil {
		return e
	}
	e.writeFieldStart(key)
	e.writeStringValue(value)
	return e
}

// needsEscaping checks if a string contains characters that need JSON escaping
// Optimized for better performance while staying simple and safe
func needsEscaping(s string) bool {
	// Optimized loop with reduced bounds checking
	for i := 0; i < len(s); i++ {
		c := s[i]
		// Combine checks for efficiency: control chars (0-31), quote (34), backslash (92)
		if c < 32 || c == '"' || c == '\\' {
			return true
		}
	}
	return false
}

// Int adds an integer field to the event.
func (e *Event) Int(key string, value int) *Event {
	if e.buf == nil {
		return e
	}
	e.writeFieldStart(key)
	e.buf = appendInt(e.buf, value)
	return e
}

// Bool adds a boolean field to the event.
func (e *Event) Bool(key string, value bool) *Event {
	if e.buf == nil {
		return e
	}
	e.writeFieldStart(key)
	e.buf = appendBool(e.buf, value)
	return e
}

// Float64 adds a float64 field to the event.
func (e *Event) Float64(key string, value float64) *Event {
	if e.buf == nil {
		return e
	}
	e.writeFieldStart(key)
	e.writeRawValue([]byte(strconv.FormatFloat(value, 'f', -1, 64)))
	return e
}

// Time adds a time.Time field to the event.
func (e *Event) Time(key string, value time.Time) *Event {
	if e.buf == nil {
		return e
	}
	e.writeFieldStart(key)
	e.writeStringValue(value.Format(time.RFC3339))
	return e
}

// Dur adds a time.Duration field to the event.
func (e *Event) Dur(key string, value time.Duration) *Event {
	if e.buf == nil {
		return e
	}
	e.writeFieldStart(key)
	e.buf = appendUint(e.buf, uint64(value))
	return e
}

// Uint adds an unsigned integer field to the event.
func (e *Event) Uint(key string, value uint) *Event {
	if e.buf == nil {
		return e
	}
	e.writeFieldStart(key)
	e.buf = appendUint(e.buf, uint64(value))
	return e
}

// Bytes adds a byte slice field to the event (base64 encoded).
func (e *Event) Bytes(key string, value []byte) *Event {
	if e.buf == nil {
		return e
	}
	e.writeFieldStart(key)
	
	// Calculate required space and grow buffer efficiently
	encodedLen := base64.StdEncoding.EncodedLen(len(value))
	requiredCap := len(e.buf) + encodedLen + 2 // +2 for quotes
	
	// Grow buffer once if needed
	if cap(e.buf) < requiredCap {
		newBuf := make([]byte, len(e.buf), requiredCap*2) // Grow with some extra space
		copy(newBuf, e.buf)
		e.buf = newBuf
	}
	
	e.buf = append(e.buf, '"')
	start := len(e.buf)
	e.buf = e.buf[:start+encodedLen] // Extend slice to include encoded data
	base64.StdEncoding.Encode(e.buf[start:], value)
	e.buf = append(e.buf, '"')
	return e
}

// Hex adds a hexadecimal string field to the event.
func (e *Event) Hex(key string, value []byte) *Event {
	if e.buf == nil {
		return e
	}
	e.writeFieldStart(key)
	
	// Calculate required space and grow buffer efficiently
	hexLen := hex.EncodedLen(len(value))
	requiredCap := len(e.buf) + hexLen + 2 // +2 for quotes
	
	// Grow buffer once if needed
	if cap(e.buf) < requiredCap {
		newBuf := make([]byte, len(e.buf), requiredCap*2) // Grow with some extra space
		copy(newBuf, e.buf)
		e.buf = newBuf
	}
	
	e.buf = append(e.buf, '"')
	start := len(e.buf)
	e.buf = e.buf[:start+hexLen] // Extend slice to include hex data
	hex.Encode(e.buf[start:], value)
	e.buf = append(e.buf, '"')
	return e
}

// Stack adds a stack trace field to the event.
func (e *Event) Stack() *Event {
	if e.buf == nil {
		return e
	}
	buf := make([]byte, StackTraceBufferSize)
	n := runtime.Stack(buf, false)
	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = append(e.buf, []byte("stack")...)
	e.buf = append(e.buf, `":"`...)
	// Escape newlines and quotes in the stack trace
	marshaledStack := []byte(strconv.Quote(string(buf[:n])))
	e.buf = append(e.buf, marshaledStack[1:len(marshaledStack)-1]...)
	e.buf = append(e.buf, '"')
	return e
}

// Any adds an arbitrary interface{} field to the event (marshaled to JSON).
func (e *Event) Any(key string, value interface{}) *Event {
	if e.buf == nil {
		return e
	}
	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = append(e.buf, key...)
	e.buf = append(e.buf, `":`...)
	marshaledValue, err := json.Marshal(value)
	if err != nil {
		// Handle error, perhaps log it or append a string representation of the error
		e.buf = append(e.buf, []byte(fmt.Sprintf(`"!ERROR: %v!"`, err))...)
	} else {
		e.buf = append(e.buf, marshaledValue...)
	}
	return e
}

// Err adds an error field, handling nil errors gracefully.
func (e *Event) Err(err error) *Event {
	if e.buf == nil {
		return e
	}
	if err != nil {
		e.Str("error", err.Error())
	}
	return e
}

// ErrorWithStack adds an error field with stack trace information.
func (e *Event) ErrorWithStack(err error) *Event {
	if e.buf == nil {
		return e
	}
	if err != nil {
		e.Str("error", err.Error())
		
		// Try to extract stack trace from error if it implements interface
		if stackTracer, ok := err.(interface{ StackTrace() []uintptr }); ok {
			e.Any("error_stack", formatStackTrace(stackTracer.StackTrace()))
		} else {
			// Fallback to runtime stack
			e.Stack()
		}
		
		// Handle wrapped errors
		if wrapper, ok := err.(interface{ Unwrap() error }); ok {
			var causes []string
			for unwrapped := wrapper.Unwrap(); unwrapped != nil; {
				causes = append(causes, unwrapped.Error())
				if nextWrapper, ok := unwrapped.(interface{ Unwrap() error }); ok {
					unwrapped = nextWrapper.Unwrap()
				} else {
					break
				}
			}
			if len(causes) > 0 {
				e.Any("error_causes", causes)
			}
		}
	}
	return e
}

// formatStackTrace formats a stack trace for logging
func formatStackTrace(stack []uintptr) []string {
	var traces []string
	for _, pc := range stack {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			file, line := fn.FileLine(pc)
			traces = append(traces, fmt.Sprintf("%s:%d %s", file, line, fn.Name()))
		}
	}
	return traces
}

// Msg sends the event to the handler for processing.
// This is always the final method in the chain.
func (e *Event) Msg(message string) {
	if e.buf == nil {
		return // No-op for disabled events
	}
	
	level := e.level
	
	// Get encoder config for message key
	var messageKey string = "message"
	if jsonHandler, ok := e.l.handler.(*JSONHandler); ok {
		messageKey = jsonHandler.config.MessageKey
	}
	
	// Add message
	e.buf = append(e.buf, ',')
	e.buf = append(e.buf, '"')
	e.buf = append(e.buf, messageKey...)
	e.buf = append(e.buf, `":"`...)
	e.buf = append(e.buf, message...)
	e.buf = append(e.buf, '"')

	// Finalize JSON and add newline
	e.buf = append(e.buf, '}')
	e.buf = append(e.buf, '\n')

	// Pass the event to the handler.
	e.l.handler.Write(e)

	// Reset the buffer and put the event back into the pool.
	e.buf = e.buf[:0]
	eventPool.Put(e)
	
	// Handle special levels that require additional actions
	switch level {
	case PANIC:
		panic(message)
	case DPANIC:
		// Only panic in development mode - check if we're in debug mode
		if isDevelopmentMode() {
			panic(message)
		}
	case FATAL:
		os.Exit(1)
	}
}

// isDevelopmentMode checks if we're running in development mode
func isDevelopmentMode() bool {
	// Simple heuristic: check if we're running with go run or in a test
	return os.Getenv("CGO_ENABLED") != "" || 
		   os.Getenv("GOROOT") != "" || 
		   os.Getenv("GOCACHE") != ""
}

// EncoderConfig allows customization of JSON output format.
type EncoderConfig struct {
	TimeKey       string // Field name for timestamp (default: "time")
	LevelKey      string // Field name for log level (default: "level") 
	MessageKey    string // Field name for message (default: "message")
	ErrorKey      string // Field name for errors (default: "error")
	StacktraceKey string // Field name for stack traces (default: "stacktrace")
	CallerKey     string // Field name for caller info (default: "caller")
	TimeFormat    string // Time format (default: RFC3339)
	IncludeTime   bool   // Whether to include timestamp (default: true)
	IncludeCaller bool   // Whether to include caller info (default: false)
}

// DefaultEncoderConfig returns a default encoder configuration.
func DefaultEncoderConfig() EncoderConfig {
	return EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		MessageKey:    "message", 
		ErrorKey:      "error",
		StacktraceKey: "stacktrace",
		CallerKey:     "caller",
		TimeFormat:    time.RFC3339,
		IncludeTime:   true,
		IncludeCaller: false,
	}
}

// JSONHandler formats logs as JSON.
type JSONHandler struct {
	out    io.Writer
	config EncoderConfig
}

// NewJSONHandler creates a new JSON handler with default configuration.
func NewJSONHandler(out io.Writer) *JSONHandler {
	return &JSONHandler{
		out:    out,
		config: DefaultEncoderConfig(),
	}
}

// NewJSONHandlerWithConfig creates a new JSON handler with custom configuration.
func NewJSONHandlerWithConfig(out io.Writer, config EncoderConfig) *JSONHandler {
	return &JSONHandler{
		out:    out,
		config: config,
	}
}

// Write handles the log event.
func (h *JSONHandler) Write(e *Event) error {
	if _, err := h.out.Write(e.buf); err != nil {
		return fmt.Errorf("failed to write log event: %w", err)
	}
	return nil
}

// ConsoleHandler formats logs for human-readable console output.
type ConsoleHandler struct {
	out io.Writer
}

// NewConsoleHandler creates a new ConsoleHandler.
func NewConsoleHandler(out io.Writer) *ConsoleHandler {
	return &ConsoleHandler{out: out}
}

// Write handles the log event.
func (h *ConsoleHandler) Write(e *Event) error {
	var data map[string]interface{}
	if err := json.Unmarshal(e.buf, &data); err != nil {
		return fmt.Errorf("failed to unmarshal event buffer: %w", err)
	}

	level, _ := data["level"].(string)
	message, _ := data["message"].(string)

	// Get color for the level
	color := getColorForLevel(level)

	// Format timestamp
	timestamp := time.Now().Format("2006-01-02T15:04:05Z")

	// Write level and timestamp
	if _, err := h.out.Write([]byte(fmt.Sprintf("%s%s\x1b[0m[%s] ", color, level, timestamp))); err != nil {
		return fmt.Errorf("failed to write level and timestamp: %w", err)
	}

	// Write message
	if _, err := h.out.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	// Write fields
	for k, v := range data {
		if k != "level" && k != "message" {
			if _, err := h.out.Write([]byte(fmt.Sprintf(" %s=%v", k, v))); err != nil {
				return fmt.Errorf("failed to write field %s: %w", k, err)
			}
		}
	}
	if _, err := h.out.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

func getColorForLevel(level string) string {
	switch level {
	case "info":
		return "\x1b[34m" // Blue
	case "warn":
		return "\x1b[33m" // Yellow
	case "error", "fatal":
		return "\x1b[31m" // Red
	case "debug", "trace":
		return "\x1b[90m" // Bright Black (Gray)
	default:
		return "\x1b[0m" // Reset
	}
}

// WriteSyncer wraps an io.Writer and provides synchronization capabilities.
type WriteSyncer interface {
	io.Writer
	Sync() error
}

// BufferedWriteSyncer wraps a WriteSyncer with buffering for better performance.
type BufferedWriteSyncer struct {
	writer WriteSyncer
	buf    []byte
	mutex  sync.Mutex
}

// NewBufferedWriteSyncer creates a new buffered write syncer.
func NewBufferedWriteSyncer(writer WriteSyncer, bufferSize int) *BufferedWriteSyncer {
	return &BufferedWriteSyncer{
		writer: writer,
		buf:    make([]byte, 0, bufferSize),
	}
}

// Write writes data to the buffer.
func (bws *BufferedWriteSyncer) Write(p []byte) (n int, err error) {
	bws.mutex.Lock()
	defer bws.mutex.Unlock()
	
	// If buffer would overflow, flush first
	if len(bws.buf)+len(p) > cap(bws.buf) {
		if err := bws.flushLocked(); err != nil {
			return 0, err
		}
	}
	
	bws.buf = append(bws.buf, p...)
	return len(p), nil
}

// Sync flushes the buffer and syncs the underlying writer.
func (bws *BufferedWriteSyncer) Sync() error {
	bws.mutex.Lock()
	defer bws.mutex.Unlock()
	
	if err := bws.flushLocked(); err != nil {
		return err
	}
	return bws.writer.Sync()
}

// flushLocked flushes the buffer without locking (caller must hold mutex).
func (bws *BufferedWriteSyncer) flushLocked() error {
	if len(bws.buf) == 0 {
		return nil
	}
	
	_, err := bws.writer.Write(bws.buf)
	bws.buf = bws.buf[:0]
	return err
}

// FileWriteSyncer implements WriteSyncer for files.
type FileWriteSyncer struct {
	file *os.File
}

// NewFileWriteSyncer creates a new file write syncer.
func NewFileWriteSyncer(filename string) (*FileWriteSyncer, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, DefaultFilePermissions)
	if err != nil {
		return nil, err
	}
	return &FileWriteSyncer{file: file}, nil
}

// Write writes data to the file.
func (fws *FileWriteSyncer) Write(p []byte) (n int, err error) {
	return fws.file.Write(p)
}

// Sync syncs the file.
func (fws *FileWriteSyncer) Sync() error {
	return fws.file.Sync()
}

// Close closes the file.
func (fws *FileWriteSyncer) Close() error {
	return fws.file.Close()
}

// RotatingFileWriteSyncer implements WriteSyncer with file rotation.
type RotatingFileWriteSyncer struct {
	filename    string
	maxSize     int64 // Maximum file size in bytes
	maxFiles    int   // Maximum number of files to keep
	currentFile *os.File
	currentSize int64
	mutex       sync.Mutex
}

// NewRotatingFileWriteSyncer creates a new rotating file write syncer.
func NewRotatingFileWriteSyncer(filename string, maxSize int64, maxFiles int) (*RotatingFileWriteSyncer, error) {
	rws := &RotatingFileWriteSyncer{
		filename: filename,
		maxSize:  maxSize,
		maxFiles: maxFiles,
	}
	
	if err := rws.openFile(); err != nil {
		return nil, err
	}
	
	return rws, nil
}

// Write writes data to the current file, rotating if necessary.
func (rws *RotatingFileWriteSyncer) Write(p []byte) (n int, err error) {
	rws.mutex.Lock()
	defer rws.mutex.Unlock()
	
	// Check if rotation is needed
	if rws.currentSize+int64(len(p)) > rws.maxSize {
		if err := rws.rotateLocked(); err != nil {
			return 0, err
		}
	}
	
	n, err = rws.currentFile.Write(p)
	rws.currentSize += int64(n)
	return n, err
}

// Sync syncs the current file.
func (rws *RotatingFileWriteSyncer) Sync() error {
	rws.mutex.Lock()
	defer rws.mutex.Unlock()
	
	if rws.currentFile == nil {
		return nil
	}
	return rws.currentFile.Sync()
}

// Close closes the current file.
func (rws *RotatingFileWriteSyncer) Close() error {
	rws.mutex.Lock()
	defer rws.mutex.Unlock()
	
	if rws.currentFile == nil {
		return nil
	}
	return rws.currentFile.Close()
}

// openFile opens a new log file.
func (rws *RotatingFileWriteSyncer) openFile() error {
	file, err := os.OpenFile(rws.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, DefaultFilePermissions)
	if err != nil {
		return err
	}
	
	// Get current file size
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	
	rws.currentFile = file
	rws.currentSize = stat.Size()
	return nil
}

// rotateLocked rotates the log files (caller must hold mutex).
func (rws *RotatingFileWriteSyncer) rotateLocked() error {
	if rws.currentFile != nil {
		rws.currentFile.Close()
	}
	
	// Rotate existing files
	for i := rws.maxFiles - 1; i > 0; i-- {
		oldName := fmt.Sprintf("%s.%d", rws.filename, i)
		newName := fmt.Sprintf("%s.%d", rws.filename, i+1)
		
		if i == rws.maxFiles-1 {
			// Remove the oldest file
			os.Remove(newName)
		}
		
		// Rename if file exists
		if _, err := os.Stat(oldName); err == nil {
			os.Rename(oldName, newName)
		}
	}
	
	// Move current file to .1
	if _, err := os.Stat(rws.filename); err == nil {
		os.Rename(rws.filename, rws.filename+".1")
	}
	
	// Open new file
	return rws.openFile()
}

// MultiWriter is a writer that writes to multiple writers.
type MultiWriter struct {
	writers []io.Writer
}

// NewMultiWriter creates a new MultiWriter that writes to all provided writers.
func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

// Write writes data to all writers.
func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
	}
	return len(p), nil
}

// A default logger for package-level functions.
var defaultLogger *Logger

func init() {
	initDefaultLogger()
}

var isTerminal = isatty

// ParseLevel converts a string to a Level.
// Returns INFO level for unrecognized level strings.
func ParseLevel(levelStr string) Level {
	switch levelStr {
	case "trace":
		return TRACE
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	case "dpanic":
		return DPANIC
	case "panic":
		return PANIC
	case "fatal":
		return FATAL
	default:
		return INFO // Default to INFO if the level is not recognized
	}
}

// initDefaultLogger initializes the default logger based on environment variables.
func initDefaultLogger() {
	format := os.Getenv("LOGMA_FORMAT")
	if format == "" {
		if isTerminal(os.Stdout) {
			format = "console"
		} else {
			format = "json"
		}
	}

	level := ParseLevel(os.Getenv("LOGMA_LEVEL"))

	switch format {
	case "console":
		defaultLogger = New(NewConsoleHandler(os.Stdout)).SetLevel(level)
	default:
		// Default to JSON if the format is not specified or is "json"
		defaultLogger = New(NewJSONHandler(os.Stdout)).SetLevel(level)
	}
}

// SetLevel sets the logging level for the logger.
func (l *Logger) SetLevel(level Level) *Logger {
	l.level = level
	return l
}

// SetSampler sets a global sampler for this logger.
func (l *Logger) SetSampler(sampler Sampler) *Logger {
	l.sampler = sampler
	return l
}

// SetLevelSampler sets a level-based sampler for this logger.
func (l *Logger) SetLevelSampler(sampler *LevelSampler) *Logger {
	l.levelSampler = sampler
	return l
}

// Trace starts a new message with the TRACE level on the default logger.
func Trace() *Event {
	return defaultLogger.Trace()
}

// Debug starts a new message with the DEBUG level on the default logger.
func Debug() *Event {
	return defaultLogger.Debug()
}

// Info starts a new message with the INFO level on the default logger.
func Info() *Event {
	return defaultLogger.Info()
}

// Warn starts a new message with the WARN level on the default logger.
func Warn() *Event {
	return defaultLogger.Warn()
}

// Error starts a new message with the ERROR level on the default logger.
func Error() *Event {
	return defaultLogger.Error()
}

// DPanic starts a new message with the DPANIC level on the default logger.
func DPanic() *Event {
	return defaultLogger.DPanic()
}

// Panic starts a new message with the PANIC level on the default logger.
func Panic() *Event {
	return defaultLogger.Panic()
}

// Fatal starts a new message with the FATAL level on the default logger.
func Fatal() *Event {
	return defaultLogger.Fatal()
}

// WithContext returns a logger with context from the default logger.
func WithContext(ctx context.Context) *Logger {
	return defaultLogger.WithContext(ctx)
}

// Ctx returns a logger with context from the default logger.
// Deprecated: Use WithContext instead for more comprehensive context support.
func Ctx(ctx context.Context) *Logger {
	return defaultLogger.Ctx(ctx)
}

// Common context keys for extractors
type contextKey string

const (
	// RequestIDKey is a common context key for request IDs
	RequestIDKey contextKey = "request_id"
	// UserIDKey is a common context key for user IDs  
	UserIDKey contextKey = "user_id"
	// SessionIDKey is a common context key for session IDs
	SessionIDKey contextKey = "session_id"
)

// DefaultRequestIDExtractor extracts request_id from context if present
func DefaultRequestIDExtractor(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		fields["request_id"] = requestID
	}
	return fields
}

// DefaultUserIDExtractor extracts user_id from context if present
func DefaultUserIDExtractor(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})
	if userID := ctx.Value(UserIDKey); userID != nil {
		fields["user_id"] = userID
	}
	return fields
}
