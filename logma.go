package logma

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	oteltrace "go.opentelemetry.io/otel/trace"
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
	FATAL
)

// Handler processes a log event and writes it to an output.
type Handler interface {
	// Write handles the log event, writing it to its destination.
	// The handler is responsible for returning the event's buffer to the pool.
	Write(e *event) error
}

// Logger is the main logging interface.
type Logger struct {
	handler Handler
	level   Level
	context []byte // Pre-formatted context fields for this logger instance.
}

// New creates a new logger with the given handler.
func New(handler Handler) *Logger {
	return &Logger{handler: handler}
}

// event represents a single log message.
type event struct {
	buf   []byte // The raw buffer for building the log line.
	level Level
}

// Global pool for event objects.
var eventPool = &sync.Pool{
	New: func() interface{} {
		return &event{
			buf: make([]byte, 0, 500), // Start with a 500-byte buffer.
		}
	},
}

// Event is the public type for a log event.
type Event struct {
	e *event
	l *Logger
}

// With creates a new Event with the current logger's context.
func (l *Logger) With() *Event {
	return &Event{e: &event{buf: append([]byte{}, l.context...), level: l.level}, l: l}
}

// Logger returns a new Logger with the event's fields as context.
func (e *Event) Logger() *Logger {
	// Remove the leading comma if present
	contextBuf := e.e.buf
	if len(contextBuf) > 0 && contextBuf[0] == ',' {
		contextBuf = contextBuf[1:]
	}
	return &Logger{handler: e.l.handler, level: e.l.level, context: contextBuf}
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

func (l *Logger) log(level Level) *event {
	if level < l.level {
		return nil
	}

	e := eventPool.Get().(*event)
	e.level = level

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
		return &Event{e: &event{}, l: l} // Return a no-op Event
	}
	return &Event{e: e, l: l}
}

// Error starts a new message with the ERROR level.
func (l *Logger) Error() *Event {
	e := l.log(ERROR)
	if e == nil {
		return &Event{e: &event{}, l: l} // Return a no-op Event
	}
	return &Event{e: e, l: l}
}

// Str adds a string field to the event.
func (e *Event) Str(key, value string) *Event {
	if e.e == nil {
		return e
	}
	e.e.buf = append(e.e.buf, ',')
	e.e.buf = append(e.e.buf, '"')
	e.e.buf = append(e.e.buf, key...)
	e.e.buf = append(e.e.buf, `":"`...)
	e.e.buf = append(e.e.buf, value...)
	e.e.buf = append(e.e.buf, '"')
	return e
}

// Int adds an integer field to the event.
func (e *Event) Int(key string, value int) *Event {
	if e.e == nil {
		return e
	}
	e.e.buf = append(e.e.buf, ',')
	e.e.buf = append(e.e.buf, '"')
	e.e.buf = append(e.e.buf, key...)
	e.e.buf = append(e.e.buf, `":`...)
	e.e.buf = append(e.e.buf, []byte(strconv.Itoa(value))...)
	return e
}

// Err adds an error field, handling nil errors gracefully.
func (e *Event) Bool(key string, value bool) *Event {
	if e.e == nil {
		return e
	}
	e.e.buf = append(e.e.buf, ',')
	e.e.buf = append(e.e.buf, '"')
	e.e.buf = append(e.e.buf, key...)
	e.e.buf = append(e.e.buf, `":`...)
	e.e.buf = append(e.e.buf, []byte(strconv.FormatBool(value))...)
	return e
}

func (e *Event) Float64(key string, value float64) *Event {
	if e.e == nil {
		return e
	}
	e.e.buf = append(e.e.buf, ',')
	e.e.buf = append(e.e.buf, '"')
	e.e.buf = append(e.e.buf, key...)
	e.e.buf = append(e.e.buf, `":`...)
	e.e.buf = append(e.e.buf, []byte(strconv.FormatFloat(value, 'f', -1, 64))...)
	return e
}

func (e *Event) Time(key string, value time.Time) *Event {
	if e.e == nil {
		return e
	}
	e.e.buf = append(e.e.buf, ',')
	e.e.buf = append(e.e.buf, '"')
	e.e.buf = append(e.e.buf, key...)
	e.e.buf = append(e.e.buf, `":"`...)
	e.e.buf = append(e.e.buf, []byte(value.Format(time.RFC3339))...)
	e.e.buf = append(e.e.buf, '"')
	return e
}

func (e *Event) Dur(key string, value time.Duration) *Event {
	if e.e == nil {
		return e
	}
	e.e.buf = append(e.e.buf, ',')
	e.e.buf = append(e.e.buf, '"')
	e.e.buf = append(e.e.buf, key...)
	e.e.buf = append(e.e.buf, `":`...)
	e.e.buf = append(e.e.buf, []byte(strconv.FormatInt(int64(value), 10))...)
	return e
}

func (e *Event) Uint(key string, value uint) *Event {
	if e.e == nil {
		return e
	}
	e.e.buf = append(e.e.buf, ',')
	e.e.buf = append(e.e.buf, '"')
	e.e.buf = append(e.e.buf, key...)
	e.e.buf = append(e.e.buf, `":`...)
	e.e.buf = append(e.e.buf, []byte(strconv.FormatUint(uint64(value), 10))...)
	return e
}

func (e *Event) Any(key string, value interface{}) *Event {
	if e.e == nil {
		return e
	}
	e.e.buf = append(e.e.buf, ',')
	e.e.buf = append(e.e.buf, '"')
	e.e.buf = append(e.e.buf, key...)
	e.e.buf = append(e.e.buf, `":`...)
	marshaledValue, err := json.Marshal(value)
	if err != nil {
		// Handle error, perhaps log it or append a string representation of the error
		e.e.buf = append(e.e.buf, []byte(fmt.Sprintf("!ERROR: %v!", err))...)
	} else {
		e.e.buf = append(e.e.buf, marshaledValue...)
	}
	return e
}

func (e *Event) Err(err error) *Event {
	if e.e == nil {
		return e
	}
	if err != nil {
		e.Str("error", err.Error())
	}
	return e
}

// Msg sends the event to the handler for processing.
// This is always the final method in the chain.
func (e *Event) Msg(message string) {
	if e.e == nil {
		return // No-op for disabled events
	}
	// Add message
	e.e.buf = append(e.e.buf, `,"message":"`...)
	e.e.buf = append(e.e.buf, message...)
	e.e.buf = append(e.e.buf, '"')

	// Finalize JSON and add newline
	e.e.buf = append(e.e.buf, '}')
	e.e.buf = append(e.e.buf, '\n')

	// Pass the event to the handler.
	e.l.handler.Write(e.e)

	// Reset the buffer and put the event back into the pool.
	e.e.buf = e.e.buf[:0]
	eventPool.Put(e.e)
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
func (h *JSONHandler) Write(e *event) error {
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

// Write handles the log event.
func (h *ConsoleHandler) Write(e *event) error {
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
	h.out.Write([]byte(fmt.Sprintf("%s%s\x1b[0m[%s] ", color, level, timestamp)))

	// Write message
	h.out.Write([]byte(message))

	// Write fields
	for k, v := range data {
		if k != "level" && k != "message" {
			h.out.Write([]byte(fmt.Sprintf(" %s=%v", k, v)))
		}
	}
	h.out.Write([]byte("\n"))

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

// A default logger for package-level functions.
var defaultLogger *Logger

func init() {
	initDefaultLogger()
}

var isTerminal = isatty

// ParseLevel converts a string to a Level.
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
		defaultLogger = New(NewConsoleHandler(io.Discard)).SetLevel(level)
	default:
		// Default to JSON if the format is not specified or is "json"
		defaultLogger = New(NewJSONHandler(io.Discard)).SetLevel(level)
	}
}

// SetLevel sets the logging level for the logger.
func (l *Logger) SetLevel(level Level) *Logger {
	l.level = level
	return l
}

// Info starts a new message with the INFO level on the default logger.
func Info() *Event {
	return defaultLogger.Info()
}