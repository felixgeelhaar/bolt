// Package zap provides compatibility layer and migration tools for Zap users.
package zap

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/felixgeelhaar/bolt"
	"go.uber.org/zap/zapcore"
)

// Logger provides a Zap-compatible API backed by Bolt for easier migration.
type Logger struct {
	bolt *bolt.Logger
}

// New creates a new Zap-compatible logger backed by Bolt.
func New(core zapcore.Core, options ...Option) *Logger {
	// For compatibility, we ignore the core and options for now
	// In a real migration, you might want to configure Bolt based on these
	return &Logger{
		bolt: bolt.New(bolt.NewJSONHandler(os.Stdout)),
	}
}

// NewProduction builds a sensible production logger.
func NewProduction(options ...Option) (*Logger, error) {
	return &Logger{
		bolt: bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO),
	}, nil
}

// NewDevelopment builds a development logger.
func NewDevelopment(options ...Option) (*Logger, error) {
	return &Logger{
		bolt: bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG),
	}, nil
}

// NewExample builds an example logger.
func NewExample(options ...Option) *Logger {
	return &Logger{
		bolt: bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG),
	}
}

// NewNop returns a no-op Logger.
func NewNop() *Logger {
	return &Logger{
		bolt: bolt.New(bolt.NewJSONHandler(io.Discard)),
	}
}

// Option configures a Logger.
type Option func(*Logger)

// Core methods

// Debug logs a message at DebugLevel.
func (l *Logger) Debug(msg string, fields ...Field) {
	event := l.bolt.Debug()
	for _, field := range fields {
		field.AddTo(event)
	}
	event.Msg(msg)
}

// Info logs a message at InfoLevel.
func (l *Logger) Info(msg string, fields ...Field) {
	event := l.bolt.Info()
	for _, field := range fields {
		field.AddTo(event)
	}
	event.Msg(msg)
}

// Warn logs a message at WarnLevel.
func (l *Logger) Warn(msg string, fields ...Field) {
	event := l.bolt.Warn()
	for _, field := range fields {
		field.AddTo(event)
	}
	event.Msg(msg)
}

// Error logs a message at ErrorLevel.
func (l *Logger) Error(msg string, fields ...Field) {
	event := l.bolt.Error()
	for _, field := range fields {
		field.AddTo(event)
	}
	event.Msg(msg)
}

// Fatal logs a message at FatalLevel.
func (l *Logger) Fatal(msg string, fields ...Field) {
	event := l.bolt.Fatal()
	for _, field := range fields {
		field.AddTo(event)
	}
	event.Msg(msg)
}

// Panic logs a message at PanicLevel.
func (l *Logger) Panic(msg string, fields ...Field) {
	// Bolt doesn't have panic level, use Fatal
	event := l.bolt.Fatal()
	for _, field := range fields {
		field.AddTo(event)
	}
	event.Msg(msg)
	panic(msg)
}

// With creates a child logger and adds structured context to it.
func (l *Logger) With(fields ...Field) *Logger {
	event := l.bolt.With()
	for _, field := range fields {
		field.AddTo(event)
	}
	return &Logger{
		bolt: event.Logger(),
	}
}

// WithContext creates a child logger with OpenTelemetry context.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		bolt: l.bolt.Ctx(ctx),
	}
}

// Sugar wraps the Logger to provide a more ergonomic, but slightly slower, API.
func (l *Logger) Sugar() *SugaredLogger {
	return &SugaredLogger{logger: l}
}

// Sync flushes any buffered log entries.
func (l *Logger) Sync() error {
	// Bolt doesn't buffer, so this is a no-op
	return nil
}

// SugaredLogger provides a more ergonomic API for casual logging.
type SugaredLogger struct {
	logger *Logger
}

// Debug uses fmt.Sprint to construct and log a message.
func (s *SugaredLogger) Debug(args ...interface{}) {
	s.logger.bolt.Debug().Msg(sprint(args...))
}

// Info uses fmt.Sprint to construct and log a message.
func (s *SugaredLogger) Info(args ...interface{}) {
	s.logger.bolt.Info().Msg(sprint(args...))
}

// Warn uses fmt.Sprint to construct and log a message.
func (s *SugaredLogger) Warn(args ...interface{}) {
	s.logger.bolt.Warn().Msg(sprint(args...))
}

// Error uses fmt.Sprint to construct and log a message.
func (s *SugaredLogger) Error(args ...interface{}) {
	s.logger.bolt.Error().Msg(sprint(args...))
}

// Fatal uses fmt.Sprint to construct and log a message, then calls os.Exit.
func (s *SugaredLogger) Fatal(args ...interface{}) {
	s.logger.bolt.Fatal().Msg(sprint(args...))
}

// Panic uses fmt.Sprint to construct and log a message, then panics.
func (s *SugaredLogger) Panic(args ...interface{}) {
	msg := sprint(args...)
	s.logger.bolt.Fatal().Msg(msg)
	panic(msg)
}

// Debugf uses fmt.Sprintf to log a templated message.
func (s *SugaredLogger) Debugf(template string, args ...interface{}) {
	s.logger.bolt.Debug().Printf(template, args...)
}

// Infof uses fmt.Sprintf to log a templated message.
func (s *SugaredLogger) Infof(template string, args ...interface{}) {
	s.logger.bolt.Info().Printf(template, args...)
}

// Warnf uses fmt.Sprintf to log a templated message.
func (s *SugaredLogger) Warnf(template string, args ...interface{}) {
	s.logger.bolt.Warn().Printf(template, args...)
}

// Errorf uses fmt.Sprintf to log a templated message.
func (s *SugaredLogger) Errorf(template string, args ...interface{}) {
	s.logger.bolt.Error().Printf(template, args...)
}

// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
func (s *SugaredLogger) Fatalf(template string, args ...interface{}) {
	s.logger.bolt.Fatal().Printf(template, args...)
}

// Panicf uses fmt.Sprintf to log a templated message, then panics.
func (s *SugaredLogger) Panicf(template string, args ...interface{}) {
	s.logger.bolt.Fatal().Printf(template, args...)
	panic(sprintf(template, args...))
}

// Debugw logs a message with some additional context.
func (s *SugaredLogger) Debugw(msg string, keysAndValues ...interface{}) {
	event := s.logger.bolt.Debug()
	s.addKeysAndValues(event, keysAndValues)
	event.Msg(msg)
}

// Infow logs a message with some additional context.
func (s *SugaredLogger) Infow(msg string, keysAndValues ...interface{}) {
	event := s.logger.bolt.Info()
	s.addKeysAndValues(event, keysAndValues)
	event.Msg(msg)
}

// Warnw logs a message with some additional context.
func (s *SugaredLogger) Warnw(msg string, keysAndValues ...interface{}) {
	event := s.logger.bolt.Warn()
	s.addKeysAndValues(event, keysAndValues)
	event.Msg(msg)
}

// Errorw logs a message with some additional context.
func (s *SugaredLogger) Errorw(msg string, keysAndValues ...interface{}) {
	event := s.logger.bolt.Error()
	s.addKeysAndValues(event, keysAndValues)
	event.Msg(msg)
}

// Fatalw logs a message with some additional context, then calls os.Exit.
func (s *SugaredLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	event := s.logger.bolt.Fatal()
	s.addKeysAndValues(event, keysAndValues)
	event.Msg(msg)
}

// Panicw logs a message with some additional context, then panics.
func (s *SugaredLogger) Panicw(msg string, keysAndValues ...interface{}) {
	event := s.logger.bolt.Fatal()
	s.addKeysAndValues(event, keysAndValues)
	event.Msg(msg)
	panic(msg)
}

// With creates a child logger with additional context.
func (s *SugaredLogger) With(args ...interface{}) *SugaredLogger {
	event := s.logger.bolt.With()
	s.addKeysAndValues(event, args)
	return &SugaredLogger{
		logger: &Logger{bolt: event.Logger()},
	}
}

// Sync flushes any buffered log entries.
func (s *SugaredLogger) Sync() error {
	return s.logger.Sync()
}

// addKeysAndValues adds key-value pairs to a Bolt event.
func (s *SugaredLogger) addKeysAndValues(event *bolt.Event, keysAndValues []interface{}) {
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := toString(keysAndValues[i])
			value := keysAndValues[i+1]
			event.Any(key, value)
		}
	}
}

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value interface{}
	Type  FieldType
}

// FieldType represents the type of a field.
type FieldType int

const (
	StringType FieldType = iota
	IntType
	Int64Type
	UintType
	Uint64Type
	BoolType
	Float64Type
	TimeType
	DurationType
	ErrorType
	AnyType
)

// AddTo adds the field to a Bolt event.
func (f Field) AddTo(event *bolt.Event) {
	switch f.Type {
	case StringType:
		if str, ok := f.Value.(string); ok {
			event.Str(f.Key, str)
		}
	case IntType:
		if i, ok := f.Value.(int); ok {
			event.Int(f.Key, i)
		}
	case Int64Type:
		if i, ok := f.Value.(int64); ok {
			event.Int64(f.Key, i)
		}
	case UintType:
		if u, ok := f.Value.(uint); ok {
			event.Uint(f.Key, u)
		}
	case Uint64Type:
		if u, ok := f.Value.(uint64); ok {
			event.Uint64(f.Key, u)
		}
	case BoolType:
		if b, ok := f.Value.(bool); ok {
			event.Bool(f.Key, b)
		}
	case Float64Type:
		if f64, ok := f.Value.(float64); ok {
			event.Float64(f.Key, f64)
		}
	case TimeType:
		if t, ok := f.Value.(time.Time); ok {
			event.Time(f.Key, t)
		}
	case DurationType:
		if d, ok := f.Value.(time.Duration); ok {
			event.Dur(f.Key, d)
		}
	case ErrorType:
		if err, ok := f.Value.(error); ok {
			event.Err(err)
		}
	default:
		event.Any(f.Key, f.Value)
	}
}

// Field constructors

// String constructs a field with the given key and value.
func String(key, val string) Field {
	return Field{Key: key, Value: val, Type: StringType}
}

// Int constructs a field with the given key and value.
func Int(key string, val int) Field {
	return Field{Key: key, Value: val, Type: IntType}
}

// Int64 constructs a field with the given key and value.
func Int64(key string, val int64) Field {
	return Field{Key: key, Value: val, Type: Int64Type}
}

// Uint constructs a field with the given key and value.
func Uint(key string, val uint) Field {
	return Field{Key: key, Value: val, Type: UintType}
}

// Uint64 constructs a field with the given key and value.
func Uint64(key string, val uint64) Field {
	return Field{Key: key, Value: val, Type: Uint64Type}
}

// Bool constructs a field with the given key and value.
func Bool(key string, val bool) Field {
	return Field{Key: key, Value: val, Type: BoolType}
}

// Float64 constructs a field with the given key and value.
func Float64(key string, val float64) Field {
	return Field{Key: key, Value: val, Type: Float64Type}
}

// Time constructs a field with the given key and value.
func Time(key string, val time.Time) Field {
	return Field{Key: key, Value: val, Type: TimeType}
}

// Duration constructs a field with the given key and value.
func Duration(key string, val time.Duration) Field {
	return Field{Key: key, Value: val, Type: DurationType}
}

// Error constructs a field that carries an error.
func Error(err error) Field {
	return Field{Key: "error", Value: err, Type: ErrorType}
}

// Any constructs a field with the given key and an arbitrary value.
func Any(key string, val interface{}) Field {
	return Field{Key: key, Value: val, Type: AnyType}
}

// Global logger functions for package-level usage
var globalLogger = &Logger{
	bolt: bolt.New(bolt.NewJSONHandler(os.Stdout)),
}

// L returns the global Logger.
func L() *Logger {
	return globalLogger
}

// S returns the global SugaredLogger.
func S() *SugaredLogger {
	return globalLogger.Sugar()
}

// ReplaceGlobals replaces the global Logger and SugaredLogger.
func ReplaceGlobals(logger *Logger) func() {
	prev := globalLogger
	globalLogger = logger
	return func() { globalLogger = prev }
}

// Config represents the configuration for a logger.
type Config struct {
	Level             string        `json:"level" yaml:"level"`
	Development       bool          `json:"development" yaml:"development"`
	DisableCaller     bool          `json:"disableCaller" yaml:"disableCaller"`
	DisableStacktrace bool          `json:"disableStacktrace" yaml:"disableStacktrace"`
	Sampling          *Sampling     `json:"sampling" yaml:"sampling"`
	Encoding          string        `json:"encoding" yaml:"encoding"`
	EncoderConfig     EncoderConfig `json:"encoderConfig" yaml:"encoderConfig"`
	OutputPaths       []string      `json:"outputPaths" yaml:"outputPaths"`
	ErrorOutputPaths  []string      `json:"errorOutputPaths" yaml:"errorOutputPaths"`
}

// Sampling configures log sampling.
type Sampling struct {
	Initial    int `json:"initial" yaml:"initial"`
	Thereafter int `json:"thereafter" yaml:"thereafter"`
}

// EncoderConfig configures the log encoder.
type EncoderConfig struct {
	MessageKey    string `json:"messageKey" yaml:"messageKey"`
	LevelKey      string `json:"levelKey" yaml:"levelKey"`
	TimeKey       string `json:"timeKey" yaml:"timeKey"`
	NameKey       string `json:"nameKey" yaml:"nameKey"`
	CallerKey     string `json:"callerKey" yaml:"callerKey"`
	StacktraceKey string `json:"stacktraceKey" yaml:"stacktraceKey"`
	LineEnding    string `json:"lineEnding" yaml:"lineEnding"`
}

// NewProductionConfig builds a sensible production logging configuration.
func NewProductionConfig() Config {
	return Config{
		Level:       "info",
		Development: false,
		Sampling: &Sampling{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding: "json",
		EncoderConfig: EncoderConfig{
			MessageKey:    "message",
			LevelKey:      "level",
			TimeKey:       "timestamp",
			CallerKey:     "caller",
			StacktraceKey: "stacktrace",
			LineEnding:    "\n",
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
}

// NewDevelopmentConfig builds a sensible development logging configuration.
func NewDevelopmentConfig() Config {
	return Config{
		Level:       "debug",
		Development: true,
		Encoding:    "console",
		EncoderConfig: EncoderConfig{
			MessageKey:    "message",
			LevelKey:      "level",
			TimeKey:       "timestamp",
			CallerKey:     "caller",
			StacktraceKey: "stacktrace",
			LineEnding:    "\n",
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
}

// Build constructs a logger from the configuration.
func (cfg Config) Build(opts ...Option) (*Logger, error) {
	var boltLogger *bolt.Logger

	// Choose handler based on encoding
	if cfg.Encoding == "console" {
		boltLogger = bolt.New(bolt.NewConsoleHandler(os.Stdout))
	} else {
		boltLogger = bolt.New(bolt.NewJSONHandler(os.Stdout))
	}

	// Set log level
	switch cfg.Level {
	case "debug":
		boltLogger = boltLogger.SetLevel(bolt.DEBUG)
	case "info":
		boltLogger = boltLogger.SetLevel(bolt.INFO)
	case "warn":
		boltLogger = boltLogger.SetLevel(bolt.WARN)
	case "error":
		boltLogger = boltLogger.SetLevel(bolt.ERROR)
	case "fatal":
		boltLogger = boltLogger.SetLevel(bolt.FATAL)
	default:
		boltLogger = boltLogger.SetLevel(bolt.INFO)
	}

	return &Logger{bolt: boltLogger}, nil
}

// Helper functions

func sprint(args ...interface{}) string {
	return fmt.Sprint(args...)
}

func sprintf(template string, args ...interface{}) string {
	return fmt.Sprintf(template, args...)
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
