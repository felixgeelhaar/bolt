// Package zerolog provides compatibility layer and migration tools for Zerolog users.
package zerolog

import (
	"fmt"
	"os"
	"strings"

	"github.com/felixgeelhaar/bolt"
)

// ParseLevel converts a level string into a zerolog Level value.
func ParseLevel(levelStr string) (Level, error) {
	switch strings.ToLower(levelStr) {
	case "trace":
		return TraceLevel, nil
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	case "fatal":
		return FatalLevel, nil
	case "panic":
		return PanicLevel, nil
	case "disabled":
		return Disabled, nil
	default:
		return InfoLevel, fmt.Errorf("unknown level string: '%s', defaulting to info", levelStr)
	}
}

// SetGlobalLevel sets the global log level.
func SetGlobalLevel(l Level) {
	GlobalLevel = l
}

// Console writer for human-friendly console output
type ConsoleWriter struct {
	Out        *os.File
	NoColor    bool
	TimeFormat string
}

// Write implements the io.Writer interface.
func (c ConsoleWriter) Write(p []byte) (int, error) {
	if c.Out == nil {
		c.Out = os.Stderr
	}
	return c.Out.Write(p)
}

// FormatLevel returns a formatted level string with appropriate color coding.
func (c ConsoleWriter) FormatLevel(level Level) string {
	if c.NoColor {
		return fmt.Sprintf("%-5s", level.String())
	}
	
	// Color codes for different levels
	switch level {
	case DebugLevel:
		return fmt.Sprintf("\x1b[36m%-5s\x1b[0m", "DEBUG") // Cyan
	case InfoLevel:
		return fmt.Sprintf("\x1b[34m%-5s\x1b[0m", "INFO")  // Blue
	case WarnLevel:
		return fmt.Sprintf("\x1b[33m%-5s\x1b[0m", "WARN")  // Yellow
	case ErrorLevel:
		return fmt.Sprintf("\x1b[31m%-5s\x1b[0m", "ERROR") // Red
	case FatalLevel:
		return fmt.Sprintf("\x1b[35m%-5s\x1b[0m", "FATAL") // Magenta
	case PanicLevel:
		return fmt.Sprintf("\x1b[35m%-5s\x1b[0m", "PANIC") // Magenta
	default:
		return fmt.Sprintf("%-5s", level.String())
	}
}

// Multi writer allows writing to multiple outputs simultaneously
type MultiLevelWriter map[Level][]Writer

// Write implements the LevelWriter interface.
func (m MultiLevelWriter) Write(level Level, p []byte) (int, error) {
	writers, ok := m[level]
	if !ok {
		return len(p), nil // No writers for this level
	}
	
	var lastN int
	var lastErr error
	for _, w := range writers {
		n, err := w.Write(p)
		lastN = n
		if err != nil {
			lastErr = err
		}
	}
	return lastN, lastErr
}

// Writer interface for compatibility.
type Writer interface {
	Write([]byte) (int, error)
}

// LevelWriter interface for level-specific writing.
type LevelWriter interface {
	Write(Level, []byte) (int, error)
}

// Additional utility functions

// Dict creates a new Object for structured logging.
func Dict() *Object {
	return &Object{
		fields: make(map[string]interface{}),
	}
}

// Arr creates a new Array for structured logging.
func Arr() *Array {
	return &Array{
		items: make([]interface{}, 0),
	}
}

// Ctx extracts the logger from the context if present.
func Ctx(ctx interface{}) Logger {
	// For basic compatibility, return global logger
	// In practice, you would extract from context
	return New(bolt.NewJSONHandler(os.Stdout))
}

// Array provides array marshaling for structured logging.
type Array struct {
	items []interface{}
}

// Append appends the val to the array.
func (a *Array) Append(val interface{}) *Array {
	a.items = append(a.items, val)
	return a
}

// Str appends the val as a string to the array.
func (a *Array) Str(val string) *Array {
	return a.Append(val)
}

// Int appends the val as an int to the array.
func (a *Array) Int(val int) *Array {
	return a.Append(val)
}

// Bool appends the val as a bool to the array.
func (a *Array) Bool(val bool) *Array {
	return a.Append(val)
}

// Object allows for complex object marshaling.
type Object struct {
	fields map[string]interface{}
}

// Str adds a string field.
func (o *Object) Str(key, val string) *Object {
	if o.fields == nil {
		o.fields = make(map[string]interface{})
	}
	o.fields[key] = val
	return o
}

// Int adds an int field.
func (o *Object) Int(key string, val int) *Object {
	if o.fields == nil {
		o.fields = make(map[string]interface{})
	}
	o.fields[key] = val
	return o
}

// Bool adds a bool field.
func (o *Object) Bool(key string, val bool) *Object {
	if o.fields == nil {
		o.fields = make(map[string]interface{})
	}
	o.fields[key] = val
	return o
}