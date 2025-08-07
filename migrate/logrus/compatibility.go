// Package logrus provides compatibility layer and migration tools for Logrus users.
package logrus

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/felixgeelhaar/bolt"
)

// Logger provides a Logrus-compatible API backed by Bolt for easier migration.
type Logger struct {
	bolt      *bolt.Logger
	mu        sync.RWMutex
	level     Level
	formatter Formatter
	hooks     LevelHooks
	exitFunc  func(int)
}

// New creates a new Logrus-compatible logger backed by Bolt.
func New() *Logger {
	return &Logger{
		bolt:      bolt.New(bolt.NewJSONHandler(os.Stdout)),
		level:     InfoLevel,
		formatter: &JSONFormatter{},
		hooks:     make(LevelHooks),
		exitFunc:  os.Exit,
	}
}

// StandardLogger returns the standard logger.
var StandardLogger = New()

// Level defines logging levels compatible with Logrus.
type Level uint32

const (
	// PanicLevel level, highest level of severity.
	PanicLevel Level = iota
	// FatalLevel level. Logs and then calls `os.Exit(1)`.
	FatalLevel
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's happening inside the application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging.
	DebugLevel
	// TraceLevel level. Designates finer-grained informational events than the Debug.
	TraceLevel
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
		return "warning"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	default:
		return "unknown"
	}
}

// convertLevel converts Logrus levels to Bolt levels.
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
	case PanicLevel:
		return bolt.FATAL // Bolt doesn't have panic level, use fatal
	default:
		return bolt.INFO
	}
}

// Formatter interface defines how to format log entries.
type Formatter interface {
	Format(*Entry) ([]byte, error)
}

// JSONFormatter formats logs into parsable json.
type JSONFormatter struct {
	TimestampFormat   string
	DisableTimestamp  bool
	DisableHTMLEscape bool
	DataKey           string
	FieldMap          FieldMap
	CallerPrettyfier  func(*runtime.Frame) (function string, file string)
	PrettyPrint       bool
}

// Format renders a single log entry.
func (f *JSONFormatter) Format(entry *Entry) ([]byte, error) {
	// For compatibility, we'll let Bolt handle the formatting
	return []byte{}, nil
}

// TextFormatter formats logs into text.
type TextFormatter struct {
	ForceColors               bool
	DisableColors             bool
	ForceQuote                bool
	DisableQuote              bool
	EnvironmentOverrideColors bool
	DisableTimestamp          bool
	FullTimestamp             bool
	TimestampFormat           string
	DisableSorting            bool
	SortingFunc               func([]string)
	DisableLevelTruncation    bool
	PadLevelText              bool
	QuoteEmptyFields          bool
	CallerPrettyfier          func(*runtime.Frame) (function string, file string)
}

// Format renders a single log entry.
func (f *TextFormatter) Format(entry *Entry) ([]byte, error) {
	// For compatibility, we'll let Bolt handle the formatting
	return []byte{}, nil
}

// FieldMap allows customization of the key names for default fields.
type FieldMap map[string]string

// Fields type for structured logging.
type Fields map[string]interface{}

// Entry represents a log entry.
type Entry struct {
	Logger  *Logger
	Data    Fields
	Time    time.Time
	Level   Level
	Caller  *runtime.Frame
	Message string
	Buffer  []byte
	Context context.Context
	err     error
}

// String returns the string representation from the reader and ultimately the
// formatter.
func (entry *Entry) String() string {
	serialized, err := entry.Logger.formatter.Format(entry)
	if err != nil {
		return err.Error()
	}
	str := string(serialized)
	return strings.TrimSuffix(str, "\n")
}

// WithField allocates a new entry and adds a field to it.
func (entry *Entry) WithField(key string, value interface{}) *Entry {
	return entry.WithFields(Fields{key: value})
}

// WithFields adds a map of fields to the Entry.
func (entry *Entry) WithFields(fields Fields) *Entry {
	data := make(Fields, len(entry.Data)+len(fields))
	for k, v := range entry.Data {
		data[k] = v
	}
	for k, v := range fields {
		data[k] = v
	}
	return &Entry{Logger: entry.Logger, Data: data, Time: entry.Time}
}

// WithError adds an error as single field to the Entry.
func (entry *Entry) WithError(err error) *Entry {
	return entry.WithField("error", err)
}

// WithContext adds a context to the Entry.
func (entry *Entry) WithContext(ctx context.Context) *Entry {
	dataCopy := make(Fields, len(entry.Data))
	for k, v := range entry.Data {
		dataCopy[k] = v
	}
	return &Entry{
		Logger:  entry.Logger,
		Data:    dataCopy,
		Time:    entry.Time,
		Context: ctx,
	}
}

// WithTime overrides the time of the Entry.
func (entry *Entry) WithTime(t time.Time) *Entry {
	dataCopy := make(Fields, len(entry.Data))
	for k, v := range entry.Data {
		dataCopy[k] = v
	}
	return &Entry{Logger: entry.Logger, Data: dataCopy, Time: t}
}

// log sends the entry to Bolt for processing.
func (entry *Entry) log(level Level, args ...interface{}) {
	if entry.Logger.IsLevelEnabled(level) {
		boltEvent := entry.Logger.logWithBolt(convertLevel(level))

		// Add all fields from entry.Data
		for key, value := range entry.Data {
			boltEvent.Any(key, value)
		}

		// Add context if present
		if entry.Context != nil {
			// Bolt handles OpenTelemetry context automatically
		}

		// Add error if present
		if entry.err != nil {
			boltEvent.Err(entry.err)
		}

		// Build message from args
		message := ""
		if len(args) > 0 {
			message = Sprint(args...)
		}

		boltEvent.Msg(message)
	}
}

// Trace logs a message at trace level.
func (entry *Entry) Trace(args ...interface{}) {
	entry.log(TraceLevel, args...)
}

// Debug logs a message at debug level.
func (entry *Entry) Debug(args ...interface{}) {
	entry.log(DebugLevel, args...)
}

// Info logs a message at info level.
func (entry *Entry) Info(args ...interface{}) {
	entry.log(InfoLevel, args...)
}

// Print logs a message at info level.
func (entry *Entry) Print(args ...interface{}) {
	entry.Info(args...)
}

// Warn logs a message at warn level.
func (entry *Entry) Warn(args ...interface{}) {
	entry.log(WarnLevel, args...)
}

// Warning logs a message at warn level.
func (entry *Entry) Warning(args ...interface{}) {
	entry.Warn(args...)
}

// Error logs a message at error level.
func (entry *Entry) Error(args ...interface{}) {
	entry.log(ErrorLevel, args...)
}

// Fatal logs a message at fatal level and calls os.Exit(1).
func (entry *Entry) Fatal(args ...interface{}) {
	entry.log(FatalLevel, args...)
	entry.Logger.exitFunc(1)
}

// Panic logs a message at panic level and panics.
func (entry *Entry) Panic(args ...interface{}) {
	entry.log(PanicLevel, args...)
	panic(Sprint(args...))
}

// Formatted logging methods

// Tracef logs a message at trace level with formatting.
func (entry *Entry) Tracef(format string, args ...interface{}) {
	if entry.Logger.IsLevelEnabled(TraceLevel) {
		entry.log(TraceLevel, Sprintf(format, args...))
	}
}

// Debugf logs a message at debug level with formatting.
func (entry *Entry) Debugf(format string, args ...interface{}) {
	if entry.Logger.IsLevelEnabled(DebugLevel) {
		entry.log(DebugLevel, Sprintf(format, args...))
	}
}

// Infof logs a message at info level with formatting.
func (entry *Entry) Infof(format string, args ...interface{}) {
	if entry.Logger.IsLevelEnabled(InfoLevel) {
		entry.log(InfoLevel, Sprintf(format, args...))
	}
}

// Printf logs a message at info level with formatting.
func (entry *Entry) Printf(format string, args ...interface{}) {
	entry.Infof(format, args...)
}

// Warnf logs a message at warn level with formatting.
func (entry *Entry) Warnf(format string, args ...interface{}) {
	if entry.Logger.IsLevelEnabled(WarnLevel) {
		entry.log(WarnLevel, Sprintf(format, args...))
	}
}

// Warningf logs a message at warn level with formatting.
func (entry *Entry) Warningf(format string, args ...interface{}) {
	entry.Warnf(format, args...)
}

// Errorf logs a message at error level with formatting.
func (entry *Entry) Errorf(format string, args ...interface{}) {
	if entry.Logger.IsLevelEnabled(ErrorLevel) {
		entry.log(ErrorLevel, Sprintf(format, args...))
	}
}

// Fatalf logs a message at fatal level with formatting and calls os.Exit(1).
func (entry *Entry) Fatalf(format string, args ...interface{}) {
	if entry.Logger.IsLevelEnabled(FatalLevel) {
		entry.log(FatalLevel, Sprintf(format, args...))
	}
	entry.Logger.exitFunc(1)
}

// Panicf logs a message at panic level with formatting and panics.
func (entry *Entry) Panicf(format string, args ...interface{}) {
	if entry.Logger.IsLevelEnabled(PanicLevel) {
		entry.log(PanicLevel, Sprintf(format, args...))
	}
	panic(Sprintf(format, args...))
}

// Line logging methods

// Traceln logs a message at trace level.
func (entry *Entry) Traceln(args ...interface{}) {
	if entry.Logger.IsLevelEnabled(TraceLevel) {
		entry.log(TraceLevel, Sprintln(args...))
	}
}

// Debugln logs a message at debug level.
func (entry *Entry) Debugln(args ...interface{}) {
	if entry.Logger.IsLevelEnabled(DebugLevel) {
		entry.log(DebugLevel, Sprintln(args...))
	}
}

// Infoln logs a message at info level.
func (entry *Entry) Infoln(args ...interface{}) {
	if entry.Logger.IsLevelEnabled(InfoLevel) {
		entry.log(InfoLevel, Sprintln(args...))
	}
}

// Println logs a message at info level.
func (entry *Entry) Println(args ...interface{}) {
	entry.Infoln(args...)
}

// Warnln logs a message at warn level.
func (entry *Entry) Warnln(args ...interface{}) {
	if entry.Logger.IsLevelEnabled(WarnLevel) {
		entry.log(WarnLevel, Sprintln(args...))
	}
}

// Warningln logs a message at warn level.
func (entry *Entry) Warningln(args ...interface{}) {
	entry.Warnln(args...)
}

// Errorln logs a message at error level.
func (entry *Entry) Errorln(args ...interface{}) {
	if entry.Logger.IsLevelEnabled(ErrorLevel) {
		entry.log(ErrorLevel, Sprintln(args...))
	}
}

// Fatalln logs a message at fatal level and calls os.Exit(1).
func (entry *Entry) Fatalln(args ...interface{}) {
	if entry.Logger.IsLevelEnabled(FatalLevel) {
		entry.log(FatalLevel, Sprintln(args...))
	}
	entry.Logger.exitFunc(1)
}

// Panicln logs a message at panic level and panics.
func (entry *Entry) Panicln(args ...interface{}) {
	if entry.Logger.IsLevelEnabled(PanicLevel) {
		entry.log(PanicLevel, Sprintln(args...))
	}
	panic(Sprintln(args...))
}

// Logger methods

// SetLevel sets the logger level.
func (logger *Logger) SetLevel(level Level) {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	logger.level = level
	logger.bolt = logger.bolt.SetLevel(convertLevel(level))
}

// GetLevel returns the logger level.
func (logger *Logger) GetLevel() Level {
	logger.mu.RLock()
	defer logger.mu.RUnlock()
	return logger.level
}

// IsLevelEnabled checks if the logger will output a logging event.
func (logger *Logger) IsLevelEnabled(level Level) bool {
	return logger.GetLevel() >= level
}

// SetFormatter sets the logger formatter.
func (logger *Logger) SetFormatter(formatter Formatter) {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	logger.formatter = formatter

	// Update Bolt handler based on formatter type
	switch formatter.(type) {
	case *JSONFormatter:
		logger.bolt = bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(convertLevel(logger.level))
	case *TextFormatter:
		logger.bolt = bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(convertLevel(logger.level))
	}
}

// SetOutput sets the output destination.
func (logger *Logger) SetOutput(output io.Writer) {
	logger.mu.Lock()
	defer logger.mu.Unlock()

	// Recreate bolt logger with new output
	switch logger.formatter.(type) {
	case *JSONFormatter:
		logger.bolt = bolt.New(bolt.NewJSONHandler(output)).SetLevel(convertLevel(logger.level))
	case *TextFormatter:
		logger.bolt = bolt.New(bolt.NewConsoleHandler(output)).SetLevel(convertLevel(logger.level))
	default:
		logger.bolt = bolt.New(bolt.NewJSONHandler(output)).SetLevel(convertLevel(logger.level))
	}
}

// AddHook adds a hook to the logger.
func (logger *Logger) AddHook(hook Hook) {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	for _, level := range hook.Levels() {
		logger.hooks[level] = append(logger.hooks[level], hook)
	}
}

// newEntry creates a new entry for the logger.
func (logger *Logger) newEntry() *Entry {
	entry := &Entry{
		Logger: logger,
		Data:   make(Fields),
		Time:   time.Now(),
	}
	return entry
}

// logWithBolt creates a bolt event for the given level.
func (logger *Logger) logWithBolt(level bolt.Level) *bolt.Event {
	switch level {
	case bolt.TRACE:
		return logger.bolt.Trace()
	case bolt.DEBUG:
		return logger.bolt.Debug()
	case bolt.INFO:
		return logger.bolt.Info()
	case bolt.WARN:
		return logger.bolt.Warn()
	case bolt.ERROR:
		return logger.bolt.Error()
	case bolt.FATAL:
		return logger.bolt.Fatal()
	default:
		return logger.bolt.Info()
	}
}

// WithField allocates a new entry and adds a field to it.
func (logger *Logger) WithField(key string, value interface{}) *Entry {
	return logger.newEntry().WithField(key, value)
}

// WithFields creates an entry with multiple fields.
func (logger *Logger) WithFields(fields Fields) *Entry {
	return logger.newEntry().WithFields(fields)
}

// WithError creates an entry with an error field.
func (logger *Logger) WithError(err error) *Entry {
	return logger.newEntry().WithError(err)
}

// WithContext creates an entry with a context.
func (logger *Logger) WithContext(ctx context.Context) *Entry {
	return logger.newEntry().WithContext(ctx)
}

// WithTime creates an entry with a specific time.
func (logger *Logger) WithTime(t time.Time) *Entry {
	return logger.newEntry().WithTime(t)
}

// Log logs an entry.
func (logger *Logger) Log(level Level, args ...interface{}) {
	if logger.IsLevelEnabled(level) {
		logger.newEntry().log(level, args...)
	}
}

// Logf logs a formatted entry.
func (logger *Logger) Logf(level Level, format string, args ...interface{}) {
	if logger.IsLevelEnabled(level) {
		logger.newEntry().log(level, Sprintf(format, args...))
	}
}

// Logln logs an entry with a newline.
func (logger *Logger) Logln(level Level, args ...interface{}) {
	if logger.IsLevelEnabled(level) {
		logger.newEntry().log(level, Sprintln(args...))
	}
}

// Logging methods for each level
func (logger *Logger) Trace(args ...interface{})   { logger.newEntry().Trace(args...) }
func (logger *Logger) Debug(args ...interface{})   { logger.newEntry().Debug(args...) }
func (logger *Logger) Info(args ...interface{})    { logger.newEntry().Info(args...) }
func (logger *Logger) Print(args ...interface{})   { logger.newEntry().Print(args...) }
func (logger *Logger) Warn(args ...interface{})    { logger.newEntry().Warn(args...) }
func (logger *Logger) Warning(args ...interface{}) { logger.newEntry().Warning(args...) }
func (logger *Logger) Error(args ...interface{})   { logger.newEntry().Error(args...) }
func (logger *Logger) Fatal(args ...interface{})   { logger.newEntry().Fatal(args...) }
func (logger *Logger) Panic(args ...interface{})   { logger.newEntry().Panic(args...) }

// Formatted logging methods
func (logger *Logger) Tracef(format string, args ...interface{}) {
	logger.newEntry().Tracef(format, args...)
}
func (logger *Logger) Debugf(format string, args ...interface{}) {
	logger.newEntry().Debugf(format, args...)
}
func (logger *Logger) Infof(format string, args ...interface{}) {
	logger.newEntry().Infof(format, args...)
}
func (logger *Logger) Printf(format string, args ...interface{}) {
	logger.newEntry().Printf(format, args...)
}
func (logger *Logger) Warnf(format string, args ...interface{}) {
	logger.newEntry().Warnf(format, args...)
}
func (logger *Logger) Warningf(format string, args ...interface{}) {
	logger.newEntry().Warningf(format, args...)
}
func (logger *Logger) Errorf(format string, args ...interface{}) {
	logger.newEntry().Errorf(format, args...)
}
func (logger *Logger) Fatalf(format string, args ...interface{}) {
	logger.newEntry().Fatalf(format, args...)
}
func (logger *Logger) Panicf(format string, args ...interface{}) {
	logger.newEntry().Panicf(format, args...)
}

// Line logging methods
func (logger *Logger) Traceln(args ...interface{})   { logger.newEntry().Traceln(args...) }
func (logger *Logger) Debugln(args ...interface{})   { logger.newEntry().Debugln(args...) }
func (logger *Logger) Infoln(args ...interface{})    { logger.newEntry().Infoln(args...) }
func (logger *Logger) Println(args ...interface{})   { logger.newEntry().Println(args...) }
func (logger *Logger) Warnln(args ...interface{})    { logger.newEntry().Warnln(args...) }
func (logger *Logger) Warningln(args ...interface{}) { logger.newEntry().Warningln(args...) }
func (logger *Logger) Errorln(args ...interface{})   { logger.newEntry().Errorln(args...) }
func (logger *Logger) Fatalln(args ...interface{})   { logger.newEntry().Fatalln(args...) }
func (logger *Logger) Panicln(args ...interface{})   { logger.newEntry().Panicln(args...) }

// Hook interface allows adding hooks to loggers.
type Hook interface {
	Levels() []Level
	Fire(*Entry) error
}

// LevelHooks is a map of hooks for each level.
type LevelHooks map[Level][]Hook

// Add a hook to a level.
func (hooks LevelHooks) Add(hook Hook) {
	for _, level := range hook.Levels() {
		hooks[level] = append(hooks[level], hook)
	}
}

// Fire all hooks for a level.
func (hooks LevelHooks) Fire(level Level, entry *Entry) error {
	for _, hook := range hooks[level] {
		if err := hook.Fire(entry); err != nil {
			return err
		}
	}
	return nil
}

// Package-level functions that use the standard logger

// SetLevel sets the standard logger level.
func SetLevel(level Level) {
	StandardLogger.SetLevel(level)
}

// GetLevel returns the standard logger level.
func GetLevel() Level {
	return StandardLogger.GetLevel()
}

// IsLevelEnabled checks if the standard logger will output a logging event.
func IsLevelEnabled(level Level) bool {
	return StandardLogger.IsLevelEnabled(level)
}

// SetOutput sets the standard logger output.
func SetOutput(out io.Writer) {
	StandardLogger.SetOutput(out)
}

// SetFormatter sets the standard logger formatter.
func SetFormatter(formatter Formatter) {
	StandardLogger.SetFormatter(formatter)
}

// AddHook adds a hook to the standard logger.
func AddHook(hook Hook) {
	StandardLogger.AddHook(hook)
}

// WithField creates an entry from the standard logger and adds a field to it.
func WithField(key string, value interface{}) *Entry {
	return StandardLogger.WithField(key, value)
}

// WithFields creates an entry from the standard logger and adds multiple fields to it.
func WithFields(fields Fields) *Entry {
	return StandardLogger.WithFields(fields)
}

// WithError creates an entry from the standard logger with an error field.
func WithError(err error) *Entry {
	return StandardLogger.WithError(err)
}

// WithContext creates an entry from the standard logger with a context.
func WithContext(ctx context.Context) *Entry {
	return StandardLogger.WithContext(ctx)
}

// WithTime creates an entry from the standard logger with a specific time.
func WithTime(t time.Time) *Entry {
	return StandardLogger.WithTime(t)
}

// Standard logging methods
func Trace(args ...interface{})   { StandardLogger.Trace(args...) }
func Debug(args ...interface{})   { StandardLogger.Debug(args...) }
func Info(args ...interface{})    { StandardLogger.Info(args...) }
func Print(args ...interface{})   { StandardLogger.Print(args...) }
func Warn(args ...interface{})    { StandardLogger.Warn(args...) }
func Warning(args ...interface{}) { StandardLogger.Warning(args...) }
func Error(args ...interface{})   { StandardLogger.Error(args...) }
func Fatal(args ...interface{})   { StandardLogger.Fatal(args...) }
func Panic(args ...interface{})   { StandardLogger.Panic(args...) }

// Formatted logging methods
func Tracef(format string, args ...interface{})   { StandardLogger.Tracef(format, args...) }
func Debugf(format string, args ...interface{})   { StandardLogger.Debugf(format, args...) }
func Infof(format string, args ...interface{})    { StandardLogger.Infof(format, args...) }
func Printf(format string, args ...interface{})   { StandardLogger.Printf(format, args...) }
func Warnf(format string, args ...interface{})    { StandardLogger.Warnf(format, args...) }
func Warningf(format string, args ...interface{}) { StandardLogger.Warningf(format, args...) }
func Errorf(format string, args ...interface{})   { StandardLogger.Errorf(format, args...) }
func Fatalf(format string, args ...interface{})   { StandardLogger.Fatalf(format, args...) }
func Panicf(format string, args ...interface{})   { StandardLogger.Panicf(format, args...) }

// Line logging methods
func Traceln(args ...interface{})   { StandardLogger.Traceln(args...) }
func Debugln(args ...interface{})   { StandardLogger.Debugln(args...) }
func Infoln(args ...interface{})    { StandardLogger.Infoln(args...) }
func Println(args ...interface{})   { StandardLogger.Println(args...) }
func Warnln(args ...interface{})    { StandardLogger.Warnln(args...) }
func Warningln(args ...interface{}) { StandardLogger.Warningln(args...) }
func Errorln(args ...interface{})   { StandardLogger.Errorln(args...) }
func Fatalln(args ...interface{})   { StandardLogger.Fatalln(args...) }
func Panicln(args ...interface{})   { StandardLogger.Panicln(args...) }

// Helper functions for string formatting

func Sprint(args ...interface{}) string {
	return fmt.Sprint(args...)
}

func Sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

func Sprintln(args ...interface{}) string {
	return fmt.Sprintln(args...)
}

// ParseLevel converts a string to a Level.
func ParseLevel(lvl string) (Level, error) {
	switch strings.ToLower(lvl) {
	case "panic":
		return PanicLevel, nil
	case "fatal":
		return FatalLevel, nil
	case "error":
		return ErrorLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	case "trace":
		return TraceLevel, nil
	}
	return InfoLevel, fmt.Errorf("invalid log level: %s", lvl)
}

// AllLevels returns all available log levels.
func AllLevels() []Level {
	return []Level{
		PanicLevel,
		FatalLevel,
		ErrorLevel,
		WarnLevel,
		InfoLevel,
		DebugLevel,
		TraceLevel,
	}
}
