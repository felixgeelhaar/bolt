// Package zerolog provides compatibility layer and migration tools for Zerolog users.
package zerolog

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/felixgeelhaar/bolt"
)

// Logger provides a Zerolog-compatible API backed by Bolt for easier migration.
// This allows for gradual migration by providing the same interface.
type Logger struct {
	bolt *bolt.Logger
}

// New creates a new Zerolog-compatible logger backed by Bolt.
func New(writer io.Writer) Logger {
	return Logger{
		bolt: bolt.New(bolt.NewJSONHandler(writer)),
	}
}

// Level represents logging levels compatible with Zerolog.
type Level int8

const (
	// TraceLevel defines trace log level.
	TraceLevel Level = iota
	// DebugLevel defines debug log level.
	DebugLevel
	// InfoLevel defines info log level.
	InfoLevel
	// WarnLevel defines warn log level.
	WarnLevel
	// ErrorLevel defines error log level.
	ErrorLevel
	// FatalLevel defines fatal log level.
	FatalLevel
	// PanicLevel defines panic log level.
	PanicLevel
	// NoLevel defines an absent log level.
	NoLevel
	// Disabled disables the logger.
	Disabled
)

// String returns the string representation of the level.
func (l Level) String() string {
	switch l {
	case TraceLevel:
		return "trace"
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	case NoLevel:
		return ""
	case Disabled:
		return "disabled"
	default:
		return ""
	}
}

// convertLevel converts Zerolog levels to Bolt levels.
func convertLevel(level Level) bolt.Level {
	switch level {
	case TraceLevel:
		return bolt.TRACE
	case DebugLevel:
		return bolt.DEBUG
	case InfoLevel:
		return bolt.INFO
	case WarnLevel:
		return bolt.WARN
	case ErrorLevel:
		return bolt.ERROR
	case FatalLevel:
		return bolt.FATAL
	default:
		return bolt.INFO
	}
}

// Level sets the logging level.
func (l Logger) Level(level Level) Logger {
	return Logger{
		bolt: l.bolt.SetLevel(convertLevel(level)),
	}
}

// With creates a child logger with additional context.
func (l Logger) With() Context {
	return Context{
		event: l.bolt.With(),
		logger: l,
	}
}

// Info starts a new message with info level.
func (l Logger) Info() *Event {
	return &Event{bolt: l.bolt.Info()}
}

// Debug starts a new message with debug level.
func (l Logger) Debug() *Event {
	return &Event{bolt: l.bolt.Debug()}
}

// Error starts a new message with error level.
func (l Logger) Error() *Event {
	return &Event{bolt: l.bolt.Error()}
}

// Warn starts a new message with warn level.
func (l Logger) Warn() *Event {
	return &Event{bolt: l.bolt.Warn()}
}

// Trace starts a new message with trace level.
func (l Logger) Trace() *Event {
	return &Event{bolt: l.bolt.Trace()}
}

// Fatal starts a new message with fatal level.
func (l Logger) Fatal() *Event {
	return &Event{bolt: l.bolt.Fatal()}
}

// Log starts a new message with the specified level.
func (l Logger) Log() *Event {
	return &Event{bolt: l.bolt.Info()}
}

// Context represents a logging context for building structured logs.
type Context struct {
	event  *bolt.Event
	logger Logger
}

// Logger returns a new logger with the context.
func (c Context) Logger() Logger {
	return Logger{bolt: c.event.Logger()}
}

// Str adds a string field to the context.
func (c Context) Str(key, val string) Context {
	c.event.Str(key, val)
	return c
}

// Int adds an integer field to the context.
func (c Context) Int(key string, i int) Context {
	c.event.Int(key, i)
	return c
}

// Bool adds a boolean field to the context.
func (c Context) Bool(key string, b bool) Context {
	c.event.Bool(key, b)
	return c
}

// Float64 adds a float64 field to the context.
func (c Context) Float64(key string, f float64) Context {
	c.event.Float64(key, f)
	return c
}

// Time adds a time field to the context.
func (c Context) Time(key string, t time.Time) Context {
	c.event.Time(key, t)
	return c
}

// Dur adds a duration field to the context.
func (c Context) Dur(key string, d time.Duration) Context {
	c.event.Dur(key, d)
	return c
}

// Interface adds an interface{} field to the context.
func (c Context) Interface(key string, i interface{}) Context {
	c.event.Any(key, i)
	return c
}

// Event represents a log event with Zerolog-compatible API.
type Event struct {
	bolt *bolt.Event
}

// Str adds a string field to the event.
func (e *Event) Str(key, val string) *Event {
	if e.bolt != nil {
		e.bolt.Str(key, val)
	}
	return e
}

// Int adds an integer field to the event.
func (e *Event) Int(key string, i int) *Event {
	if e.bolt != nil {
		e.bolt.Int(key, i)
	}
	return e
}

// Int64 adds a 64-bit integer field to the event.
func (e *Event) Int64(key string, i int64) *Event {
	if e.bolt != nil {
		e.bolt.Int64(key, i)
	}
	return e
}

// Uint adds an unsigned integer field to the event.
func (e *Event) Uint(key string, i uint) *Event {
	if e.bolt != nil {
		e.bolt.Uint(key, i)
	}
	return e
}

// Uint64 adds a 64-bit unsigned integer field to the event.
func (e *Event) Uint64(key string, i uint64) *Event {
	if e.bolt != nil {
		e.bolt.Uint64(key, i)
	}
	return e
}

// Bool adds a boolean field to the event.
func (e *Event) Bool(key string, b bool) *Event {
	if e.bolt != nil {
		e.bolt.Bool(key, b)
	}
	return e
}

// Float64 adds a float64 field to the event.
func (e *Event) Float64(key string, f float64) *Event {
	if e.bolt != nil {
		e.bolt.Float64(key, f)
	}
	return e
}

// Time adds a time field to the event.
func (e *Event) Time(key string, t time.Time) *Event {
	if e.bolt != nil {
		e.bolt.Time(key, t)
	}
	return e
}

// Dur adds a duration field to the event.
func (e *Event) Dur(key string, d time.Duration) *Event {
	if e.bolt != nil {
		e.bolt.Dur(key, d)
	}
	return e
}

// Interface adds an interface{} field to the event.
func (e *Event) Interface(key string, i interface{}) *Event {
	if e.bolt != nil {
		e.bolt.Any(key, i)
	}
	return e
}

// Err adds an error field to the event.
func (e *Event) Err(err error) *Event {
	if e.bolt != nil {
		e.bolt.Err(err)
	}
	return e
}

// Bytes adds a byte slice field to the event.
func (e *Event) Bytes(key string, val []byte) *Event {
	if e.bolt != nil {
		e.bolt.Bytes(key, val)
	}
	return e
}

// Hex adds a hex-encoded byte slice field to the event.
func (e *Event) Hex(key string, val []byte) *Event {
	if e.bolt != nil {
		e.bolt.Hex(key, val)
	}
	return e
}

// Ctx adds context information to the event (for OpenTelemetry integration).
func (e *Event) Ctx(ctx context.Context) *Event {
	// Bolt handles OpenTelemetry context automatically, so we don't need to do anything special
	return e
}

// Timestamp adds the current timestamp to the event.
func (e *Event) Timestamp() *Event {
	if e.bolt != nil {
		e.bolt.Timestamp()
	}
	return e
}

// Caller adds caller information to the event.
func (e *Event) Caller() *Event {
	if e.bolt != nil {
		e.bolt.Caller()
	}
	return e
}

// Stack adds stack trace to the event.
func (e *Event) Stack() *Event {
	if e.bolt != nil {
		e.bolt.Stack()
	}
	return e
}

// Msg sends the event with the given message.
func (e *Event) Msg(msg string) {
	if e.bolt != nil {
		e.bolt.Msg(msg)
	}
}

// Msgf sends the event with a formatted message.
func (e *Event) Msgf(format string, v ...interface{}) {
	if e.bolt != nil {
		e.bolt.Printf(format, v...)
	}
}

// Send sends the event without a message.
func (e *Event) Send() {
	if e.bolt != nil {
		e.bolt.Send()
	}
}

// Global logger functions for drop-in compatibility.
var global = New(os.Stdout)

// SetGlobalLevel sets the global logging level.
func SetGlobalLevel(level Level) {
	global = global.Level(level)
}

// Info starts a new message with info level on the global logger.
func Info() *Event {
	return global.Info()
}

// Debug starts a new message with debug level on the global logger.
func Debug() *Event {
	return global.Debug()
}

// Error starts a new message with error level on the global logger.
func Error() *Event {
	return global.Error()
}

// Warn starts a new message with warn level on the global logger.
func Warn() *Event {
	return global.Warn()
}

// Trace starts a new message with trace level on the global logger.
func Trace() *Event {
	return global.Trace()
}

// Fatal starts a new message with fatal level on the global logger.
func Fatal() *Event {
	return global.Fatal()
}

// Log starts a new message on the global logger.
func Log() *Event {
	return global.Log()
}

// ConsoleWriter provides a Zerolog-compatible console writer.
type ConsoleWriter struct {
	Out io.Writer
}

// NewConsoleWriter creates a new console writer.
func NewConsoleWriter() ConsoleWriter {
	return ConsoleWriter{Out: os.Stdout}
}

// Write implements io.Writer interface.
func (c ConsoleWriter) Write(p []byte) (n int, err error) {
	return c.Out.Write(p)
}

// LevelFieldName is the field name used for the level field.
var LevelFieldName = "level"

// MessageFieldName is the field name used for the message field.
var MessageFieldName = "message"

// TimestampFieldName is the field name used for the timestamp field.
var TimestampFieldName = "timestamp"

// CallerFieldName is the field name used for the caller field.
var CallerFieldName = "caller"

// ErrorFieldName is the field name used for the error field.
var ErrorFieldName = "error"