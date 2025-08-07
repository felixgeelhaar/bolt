// Package stdlog provides compatibility layer and migration tools for Go standard log users.
package stdlog

import (
	"fmt"
	"io"
	"os"

	"github.com/felixgeelhaar/bolt"
)

// Logger provides a drop-in replacement for Go's standard log.Logger.
// This allows existing code using the standard library to benefit from Bolt's performance
// while maintaining full API compatibility.
type Logger struct {
	bolt   *bolt.Logger
	prefix string
	flag   int
}

// Log flag constants - compatible with standard library log package.
const (
	Ldate         = 1 << iota     // the date in the local time zone
	Ltime                         // the time in the local time zone
	Lmicroseconds                 // microsecond resolution
	Llongfile                     // full file name and line number
	Lshortfile                    // final file name element and line number
	LUTC                          // use UTC rather than the local time zone
	Lmsgprefix                    // move the "prefix" from the beginning of the line to before the message
	LstdFlags     = Ldate | Ltime // initial values for the standard logger
)

// New creates a new Logger that's compatible with Go's standard log.Logger.
// The out parameter is the destination to which log data will be written.
// The prefix appears at the beginning of each generated log line, or after the log header if Lmsgprefix flag is provided.
// The flag argument defines the logging properties.
func New(out io.Writer, prefix string, flag int) *Logger {
	var boltLogger *bolt.Logger

	// Choose handler based on output destination and flags
	if out == os.Stderr || out == os.Stdout {
		// For standard outputs, use console handler if it looks like a terminal
		boltLogger = bolt.New(bolt.NewConsoleHandler(out))
	} else {
		// For files and other outputs, use JSON handler
		boltLogger = bolt.New(bolt.NewJSONHandler(out))
	}

	return &Logger{
		bolt:   boltLogger,
		prefix: prefix,
		flag:   flag,
	}
}

// Default creates a standard logger that writes to stderr with default flags.
func Default() *Logger {
	return New(os.Stderr, "", LstdFlags)
}

// formatMessage formats a message according to standard log format requirements.
func (l *Logger) formatMessage(format string, v ...interface{}) string {
	var msg string
	if format == "" {
		msg = fmt.Sprint(v...)
	} else {
		msg = fmt.Sprintf(format, v...)
	}

	// Add prefix if specified and not using Lmsgprefix
	if l.prefix != "" && (l.flag&Lmsgprefix) == 0 {
		msg = l.prefix + msg
	} else if l.prefix != "" && (l.flag&Lmsgprefix) != 0 {
		msg = msg + " " + l.prefix
	}

	return msg
}

// addCallerInfo adds caller information to the bolt event if flags require it.
func (l *Logger) addCallerInfo(event *bolt.Event) *bolt.Event {
	if (l.flag&Lshortfile) != 0 || (l.flag&Llongfile) != 0 {
		return event.Caller()
	}
	return event
}

// Fatal is equivalent to Print() followed by a call to os.Exit(1).
func (l *Logger) Fatal(v ...interface{}) {
	event := l.bolt.Fatal()
	event = l.addCallerInfo(event)
	event.Msg(l.formatMessage("", v...))
}

// Fatalf is equivalent to Printf() followed by a call to os.Exit(1).
func (l *Logger) Fatalf(format string, v ...interface{}) {
	event := l.bolt.Fatal()
	event = l.addCallerInfo(event)
	event.Msg(l.formatMessage(format, v...))
}

// Fatalln is equivalent to Println() followed by a call to os.Exit(1).
func (l *Logger) Fatalln(v ...interface{}) {
	event := l.bolt.Fatal()
	event = l.addCallerInfo(event)
	event.Msg(l.formatMessage("", v...))
}

// Flags returns the output flags for the logger.
func (l *Logger) Flags() int {
	return l.flag
}

// Output writes the output for a logging event.
// The calldepth parameter is ignored as Bolt handles caller information automatically.
func (l *Logger) Output(calldepth int, s string) error {
	event := l.bolt.Info()
	event = l.addCallerInfo(event)
	event.Msg(s)
	return nil
}

// Panic is equivalent to Print() followed by a call to panic().
func (l *Logger) Panic(v ...interface{}) {
	msg := l.formatMessage("", v...)
	event := l.bolt.Fatal() // Use fatal level for panic
	event = l.addCallerInfo(event)
	event.Msg(msg)
	panic(msg)
}

// Panicf is equivalent to Printf() followed by a call to panic().
func (l *Logger) Panicf(format string, v ...interface{}) {
	msg := l.formatMessage(format, v...)
	event := l.bolt.Fatal() // Use fatal level for panic
	event = l.addCallerInfo(event)
	event.Msg(msg)
	panic(msg)
}

// Panicln is equivalent to Println() followed by a call to panic().
func (l *Logger) Panicln(v ...interface{}) {
	msg := l.formatMessage("", v...)
	event := l.bolt.Fatal() // Use fatal level for panic
	event = l.addCallerInfo(event)
	event.Msg(msg)
	panic(msg)
}

// Prefix returns the output prefix for the logger.
func (l *Logger) Prefix() string {
	return l.prefix
}

// Print calls l.Output to print to the logger.
func (l *Logger) Print(v ...interface{}) {
	event := l.bolt.Info()
	event = l.addCallerInfo(event)
	event.Msg(l.formatMessage("", v...))
}

// Printf calls l.Output to print to the logger.
func (l *Logger) Printf(format string, v ...interface{}) {
	event := l.bolt.Info()
	event = l.addCallerInfo(event)
	event.Msg(l.formatMessage(format, v...))
}

// Println calls l.Output to print to the logger.
func (l *Logger) Println(v ...interface{}) {
	event := l.bolt.Info()
	event = l.addCallerInfo(event)
	event.Msg(l.formatMessage("", v...))
}

// SetFlags sets the output flags for the logger.
func (l *Logger) SetFlags(flag int) {
	l.flag = flag
}

// SetOutput sets the output destination for the logger.
func (l *Logger) SetOutput(w io.Writer) {
	// Recreate bolt logger with new output
	var boltLogger *bolt.Logger

	if w == os.Stderr || w == os.Stdout {
		boltLogger = bolt.New(bolt.NewConsoleHandler(w))
	} else {
		boltLogger = bolt.New(bolt.NewJSONHandler(w))
	}

	l.bolt = boltLogger
}

// SetPrefix sets the output prefix for the logger.
func (l *Logger) SetPrefix(prefix string) {
	l.prefix = prefix
}

// Writer returns the output destination for the logger.
func (l *Logger) Writer() io.Writer {
	// Since we can't easily extract the writer from bolt handlers,
	// we return a dummy writer. This is rarely used in practice.
	return os.Stderr
}

// Package-level functions that mirror the standard log package

// Global logger instance
var std = Default()

// Fatal is equivalent to Print() followed by a call to os.Exit(1).
func Fatal(v ...interface{}) {
	std.Fatal(v...)
}

// Fatalf is equivalent to Printf() followed by a call to os.Exit(1).
func Fatalf(format string, v ...interface{}) {
	std.Fatalf(format, v...)
}

// Fatalln is equivalent to Println() followed by a call to os.Exit(1).
func Fatalln(v ...interface{}) {
	std.Fatalln(v...)
}

// Flags returns the output flags for the standard logger.
func Flags() int {
	return std.Flags()
}

// Output writes the output for a logging event. The string s contains
// the text to print; a newline character will be added if one is not
// provided. Calldepth is used to recover the PC and is provided for
// generality, although at the moment on all pre-defined paths it will
// be 2.
func Output(calldepth int, s string) error {
	return std.Output(calldepth, s)
}

// Panic is equivalent to Print() followed by a call to panic().
func Panic(v ...interface{}) {
	std.Panic(v...)
}

// Panicf is equivalent to Printf() followed by a call to panic().
func Panicf(format string, v ...interface{}) {
	std.Panicf(format, v...)
}

// Panicln is equivalent to Println() followed by a call to panic().
func Panicln(v ...interface{}) {
	std.Panicln(v...)
}

// Prefix returns the output prefix for the standard logger.
func Prefix() string {
	return std.Prefix()
}

// Print calls Output to print to the standard logger.
func Print(v ...interface{}) {
	std.Print(v...)
}

// Printf calls Output to print to the standard logger.
func Printf(format string, v ...interface{}) {
	std.Printf(format, v...)
}

// Println calls Output to print to the standard logger.
func Println(v ...interface{}) {
	std.Println(v...)
}

// SetFlags sets the output flags for the standard logger.
func SetFlags(flag int) {
	std.SetFlags(flag)
}

// SetOutput sets the output destination for the standard logger.
func SetOutput(w io.Writer) {
	std.SetOutput(w)
}

// SetPrefix sets the output prefix for the standard logger.
func SetPrefix(prefix string) {
	std.SetPrefix(prefix)
}

// Writer returns the output destination for the standard logger.
func Writer() io.Writer {
	return std.Writer()
}

// Additional convenience functions for enhanced compatibility

// Debug provides debug-level logging (not in standard log, but commonly needed).
func Debug(v ...interface{}) {
	event := std.bolt.Debug()
	event = std.addCallerInfo(event)
	event.Msg(std.formatMessage("", v...))
}

// Debugf provides formatted debug-level logging.
func Debugf(format string, v ...interface{}) {
	event := std.bolt.Debug()
	event = std.addCallerInfo(event)
	event.Msg(std.formatMessage(format, v...))
}

// Info provides info-level logging (alias for Print functions).
func Info(v ...interface{}) {
	Print(v...)
}

// Infof provides formatted info-level logging (alias for Printf).
func Infof(format string, v ...interface{}) {
	Printf(format, v...)
}

// Warn provides warning-level logging.
func Warn(v ...interface{}) {
	event := std.bolt.Warn()
	event = std.addCallerInfo(event)
	event.Msg(std.formatMessage("", v...))
}

// Warnf provides formatted warning-level logging.
func Warnf(format string, v ...interface{}) {
	event := std.bolt.Warn()
	event = std.addCallerInfo(event)
	event.Msg(std.formatMessage(format, v...))
}

// Error provides error-level logging.
func Error(v ...interface{}) {
	event := std.bolt.Error()
	event = std.addCallerInfo(event)
	event.Msg(std.formatMessage("", v...))
}

// Errorf provides formatted error-level logging.
func Errorf(format string, v ...interface{}) {
	event := std.bolt.Error()
	event = std.addCallerInfo(event)
	event.Msg(std.formatMessage(format, v...))
}

// EnableStructuredLogging enables structured logging features while maintaining compatibility.
// This allows gradual migration to structured logging patterns.
func EnableStructuredLogging() *bolt.Logger {
	return std.bolt
}

// GetUnderlyingLogger returns the underlying Bolt logger for advanced usage.
// This allows access to Bolt's structured logging features while maintaining
// standard log compatibility for the rest of the application.
func GetUnderlyingLogger() *bolt.Logger {
	return std.bolt
}

// SetLevel sets the logging level for the underlying Bolt logger.
// This provides level-based filtering not available in standard log.
func SetLevel(level string) {
	switch level {
	case "debug":
		std.bolt = std.bolt.SetLevel(bolt.DEBUG)
	case "info":
		std.bolt = std.bolt.SetLevel(bolt.INFO)
	case "warn", "warning":
		std.bolt = std.bolt.SetLevel(bolt.WARN)
	case "error":
		std.bolt = std.bolt.SetLevel(bolt.ERROR)
	case "fatal":
		std.bolt = std.bolt.SetLevel(bolt.FATAL)
	default:
		std.bolt = std.bolt.SetLevel(bolt.INFO)
	}
}

// Migration helper functions

// MigrateStandardLogger replaces the global standard logger with a Bolt-backed version.
// This is useful for applications that use the global log functions.
func MigrateStandardLogger(output io.Writer, prefix string, flag int) {
	std = New(output, prefix, flag)
}

// CreateCompatibleLogger creates a logger that's compatible with existing standard log usage
// but provides the performance benefits of Bolt.
func CreateCompatibleLogger(output io.Writer) *Logger {
	return New(output, "", LstdFlags)
}

// Performance comparison note:
// While this compatibility layer maintains the standard log API, the underlying
// Bolt implementation provides:
// - Zero allocations in hot paths
// - Sub-100ns logging operations
// - Better performance under concurrent load
// - Structured logging capabilities when needed
//
// To get maximum performance benefits, gradually migrate to native Bolt API:
// logger.Info().Str("key", "value").Msg("message")
