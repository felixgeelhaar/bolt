//go:build bench

package bolt

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/slog"
)

func BenchmarkBolt(b *testing.B) {
	logger := New(NewJSONHandler(&bytes.Buffer{}))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info().Str("foo", "bar").Int("baz", 123).Msg("hello world")
	}
}

func BenchmarkBolt5Fields(b *testing.B) {
	logger := New(NewJSONHandler(&bytes.Buffer{}))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info().Str("f1", "v1").Str("f2", "v2").Str("f3", "v3").Str("f4", "v4").Str("f5", "v5").Msg("hello world")
	}
}

func BenchmarkBoltDisabled(b *testing.B) {
	logger := New(NewJSONHandler(&bytes.Buffer{})).SetLevel(FATAL) // Set level to FATAL to disable INFO logs
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info().Str("foo", "bar").Int("baz", 123).Msg("hello world")
	}
}

func BenchmarkZerolog(b *testing.B) {
	logger := zerolog.New(&bytes.Buffer{})
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info().Str("foo", "bar").Int("baz", 123).Msg("hello world")
	}
}

func BenchmarkZerolog5Fields(b *testing.B) {
	logger := zerolog.New(&bytes.Buffer{})
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info().Str("f1", "v1").Str("f2", "v2").Str("f3", "v3").Str("f4", "v4").Str("f5", "v5").Msg("hello world")
	}
}

func BenchmarkZerologDisabled(b *testing.B) {
	logger := zerolog.New(&bytes.Buffer{}).Level(zerolog.FatalLevel)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info().Str("foo", "bar").Int("baz", 123).Msg("hello world")
	}
}

func BenchmarkZap(b *testing.B) {
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(&bytes.Buffer{}),
		zapcore.InfoLevel,
	)
	logger := zap.New(core)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info("hello world", zap.String("foo", "bar"), zap.Int("baz", 123))
	}
}

func BenchmarkZap5Fields(b *testing.B) {
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(&bytes.Buffer{}),
		zapcore.InfoLevel,
	)
	logger := zap.New(core)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info("hello world", zap.String("f1", "v1"), zap.String("f2", "v2"), zap.String("f3", "v3"), zap.String("f4", "v4"), zap.String("f5", "v5"))
	}
}

func BenchmarkZapDisabled(b *testing.B) {
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(&bytes.Buffer{}),
		zapcore.FatalLevel,
	)
	logger := zap.New(core)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info("hello world", zap.String("foo", "bar"), zap.Int("baz", 123))
	}
}

func BenchmarkSlog(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info("hello world", "foo", "bar", "baz", 123)
	}
}

func BenchmarkSlog5Fields(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info("hello world", "f1", "v1", "f2", "v2", "f3", "v3", "f4", "v4", "f5", "v5")
	}
}

func BenchmarkSlogDisabled(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelError}))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info("hello world", "foo", "bar", "baz", 123)
	}
}
