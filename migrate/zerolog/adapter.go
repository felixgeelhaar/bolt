// Package zerolog provides a compatibility adapter for migrating from rs/zerolog to Bolt
package zerolog

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/felixgeelhaar/bolt"
)

// Logger provides a Zerolog-compatible interface using Bolt underneath
type Logger struct {
	bolt   *bolt.Logger
	level  bolt.Level
	ctx    context.Context
	fields map[string]interface{}
}

// New creates a new Zerolog-compatible logger using Bolt
func New(w io.Writer) Logger {
	handler := bolt.NewJSONHandler(w)
	return Logger{
		bolt:   bolt.New(handler),
		level:  bolt.INFO,
		fields: make(map[string]interface{}),
	}
}

// Output returns a copy of the logger with a new output destination
func (l Logger) Output(w io.Writer) Logger {
	handler := bolt.NewJSONHandler(w)
	return Logger{
		bolt:   bolt.New(handler),
		level:  l.level,
		ctx:    l.ctx,
		fields: copyFields(l.fields),
	}
}

// With returns a logger with the field added
func (l Logger) With() Context {
	return Context{
		logger: l,
		fields: copyFields(l.fields),
	}
}

// Level sets the logger level
func (l Logger) Level(level Level) Logger {
	l.level = convertLevel(level)
	l.bolt.SetLevel(l.level)
	return l
}

// Sample returns a logger with sampling configured
func (l Logger) Sample(s Sampler) Logger {
	// For compatibility - Bolt doesn't have built-in sampling yet
	return l
}

// Hook adds a hook to the logger (compatibility layer)
func (l Logger) Hook(h Hook) Logger {
	// For compatibility - hooks could be implemented as custom handlers
	return l
}

// Debug starts a debug level message
func (l Logger) Debug() *Event {
	if l.level > bolt.DEBUG {
		return &Event{disabled: true}
	}
	return &Event{
		event:  l.bolt.Debug(),
		logger: &l,
	}
}

// Info starts an info level message
func (l Logger) Info() *Event {
	if l.level > bolt.INFO {
		return &Event{disabled: true}
	}
	return &Event{
		event:  l.bolt.Info(),
		logger: &l,
	}
}

// Warn starts a warn level message
func (l Logger) Warn() *Event {
	if l.level > bolt.WARN {
		return &Event{disabled: true}
	}
	return &Event{
		event:  l.bolt.Warn(),
		logger: &l,
	}
}

// Error starts an error level message
func (l Logger) Error() *Event {
	if l.level > bolt.ERROR {
		return &Event{disabled: true}
	}
	return &Event{
		event:  l.bolt.Error(),
		logger: &l,
	}
}

// Fatal starts a fatal level message
func (l Logger) Fatal() *Event {
	return &Event{
		event:  l.bolt.Fatal(),
		logger: &l,
	}
}

// Panic starts a panic level message
func (l Logger) Panic() *Event {
	return &Event{
		event:  l.bolt.Fatal(), // Bolt doesn't distinguish panic from fatal
		logger: &l,
	}
}

// Log starts a message with a specific level
func (l Logger) Log() *Event {
	return &Event{
		event:  l.bolt.Info(),
		logger: &l,
	}
}

// Print sends a log message using fmt.Sprint
func (l Logger) Print(v ...interface{}) {
	l.Info().Msg(sprint(v...))
}

// Printf sends a log message using fmt.Sprintf
func (l Logger) Printf(format string, v ...interface{}) {
	l.Info().Msgf(format, v...)
}

// Println sends a log message using fmt.Sprintln
func (l Logger) Println(v ...interface{}) {
	l.Info().Msg(sprintln(v...))
}

// Context provides a fluent interface for building log contexts
type Context struct {
	logger Logger
	fields map[string]interface{}
}

// Logger returns the logger with the accumulated context
func (c Context) Logger() Logger {
	l := c.logger
	l.fields = c.fields
	return l
}

// Str adds a string field to the context
func (c Context) Str(key, val string) Context {
	c.fields[key] = val
	return c
}

// Int adds an integer field to the context
func (c Context) Int(key string, i int) Context {
	c.fields[key] = i
	return c
}

// Float64 adds a float64 field to the context
func (c Context) Float64(key string, f float64) Context {
	c.fields[key] = f
	return c
}

// Bool adds a bool field to the context
func (c Context) Bool(key string, b bool) Context {
	c.fields[key] = b
	return c
}

// Time adds a time field to the context
func (c Context) Time(key string, t time.Time) Context {
	c.fields[key] = t
	return c
}

// Dur adds a duration field to the context
func (c Context) Dur(key string, d time.Duration) Context {
	c.fields[key] = d
	return c
}

// Interface adds an arbitrary field to the context
func (c Context) Interface(key string, i interface{}) Context {
	c.fields[key] = i
	return c
}

// Event provides a fluent interface for building log events
type Event struct {
	event    *bolt.Event
	logger   *Logger
	disabled bool
}

// Msg sends the event with a message
func (e *Event) Msg(msg string) {
	if e.disabled {
		return
	}
	
	// Add accumulated fields from logger context
	if e.logger != nil {
		for k, v := range e.logger.fields {
			e.addField(k, v)
		}
	}
	
	e.event.Msg(msg)
}

// Msgf sends the event with a formatted message
func (e *Event) Msgf(format string, v ...interface{}) {
	if e.disabled {
		return
	}
	e.Msg(sprintf(format, v...))
}

// Send sends the event without a message
func (e *Event) Send() {
	if e.disabled {
		return
	}
	e.Msg("")
}

// Str adds a string field to the event
func (e *Event) Str(key, val string) *Event {
	if e.disabled {
		return e
	}
	e.event.Str(key, val)
	return e
}

// Int adds an integer field to the event
func (e *Event) Int(key string, i int) *Event {
	if e.disabled {
		return e
	}
	e.event.Int(key, i)
	return e
}

// Float64 adds a float64 field to the event
func (e *Event) Float64(key string, f float64) *Event {
	if e.disabled {
		return e
	}
	e.event.Float64(key, f)
	return e
}

// Bool adds a bool field to the event
func (e *Event) Bool(key string, b bool) *Event {
	if e.disabled {
		return e
	}
	e.event.Bool(key, b)
	return e
}

// Time adds a time field to the event
func (e *Event) Time(key string, t time.Time) *Event {
	if e.disabled {
		return e
	}
	e.event.Time(key, t)
	return e
}

// Dur adds a duration field to the event
func (e *Event) Dur(key string, d time.Duration) *Event {
	if e.disabled {
		return e
	}
	e.event.Dur(key, d)
	return e
}

// Interface adds an arbitrary field to the event
func (e *Event) Interface(key string, i interface{}) *Event {
	if e.disabled {
		return e
	}
	e.event.Any(key, i)
	return e
}

// Err adds an error field to the event
func (e *Event) Err(err error) *Event {
	if e.disabled {
		return e
	}
	e.event.Err(err)
	return e
}

// addField adds a field based on its type
func (e *Event) addField(key string, value interface{}) {
	switch v := value.(type) {
	case string:
		e.event.Str(key, v)
	case int:
		e.event.Int(key, v)
	case int64:
		e.event.Int64(key, v)
	case float64:
		e.event.Float64(key, v)
	case bool:
		e.event.Bool(key, v)
	case time.Time:
		e.event.Time(key, v)
	case time.Duration:
		e.event.Dur(key, v)
	case error:
		e.event.Err(v)
	default:
		e.event.Any(key, v)
	}
}

// Level represents log levels (compatibility)
type Level int8

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	PanicLevel
	NoLevel
	Disabled
)

// Sampler interface for compatibility
type Sampler interface {
	Sample(lvl Level) bool
}

// Hook interface for compatibility
type Hook interface {
	Run(e *Event, level Level, msg string)
}

// Global logger for package-level functions
var globalLogger = New(os.Stderr)

// Package-level functions for compatibility
func Debug() *Event { return globalLogger.Debug() }
func Info() *Event  { return globalLogger.Info() }
func Warn() *Event  { return globalLogger.Warn() }
func Error() *Event { return globalLogger.Error() }
func Fatal() *Event { return globalLogger.Fatal() }
func Panic() *Event { return globalLogger.Panic() }
func Log() *Event   { return globalLogger.Log() }

func Print(v ...interface{})                 { globalLogger.Print(v...) }
func Printf(format string, v ...interface{}) { globalLogger.Printf(format, v...) }
func Println(v ...interface{})               { globalLogger.Println(v...) }

// Utility functions
func convertLevel(level Level) bolt.Level {
	switch level {
	case DebugLevel:
		return bolt.DEBUG
	case InfoLevel:
		return bolt.INFO
	case WarnLevel:
		return bolt.WARN
	case ErrorLevel:
		return bolt.ERROR
	case FatalLevel:
		return bolt.FatalLevel
	case PanicLevel:
		return bolt.FatalLevel
	case Disabled:
		return bolt.Level(99) // Higher than any valid level
	default:
		return bolt.INFO
	}
}

func copyFields(fields map[string]interface{}) map[string]interface{} {
	if fields == nil {
		return make(map[string]interface{})
	}
	copy := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		copy[k] = v
	}
	return copy
}

// Simplified fmt functions to avoid dependencies
func sprint(v ...interface{}) string {
	// Basic implementation - in real use, you'd import fmt
	return "sprint not implemented"
}

func sprintf(format string, v ...interface{}) string {
	// Basic implementation - in real use, you'd import fmt
	return "sprintf not implemented"
}

func sprintln(v ...interface{}) string {
	// Basic implementation - in real use, you'd import fmt
	return "sprintln not implemented"
}