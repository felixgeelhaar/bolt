package logma

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func BenchmarkZeroAllocation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Info().Str("hello", "world").Msg("test")
	}
}

func TestJSONHandler(t *testing.T) {
	var buf bytes.Buffer

	logger := New(NewJSONHandler(&buf))

	t.Run("simple log", func(t *testing.T) {
		buf.Reset()
		logger.Info().Str("foo", "bar").Msg("hello world")
		expected := `{"level":"info","foo":"bar","message":"hello world"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected log output %q, got %q", expected, buf.String())
		}
	})

	t.Run("log with error", func(t *testing.T) {
		buf.Reset()
		logger.Error().Err(errors.New("a wild error appeared")).Msg("something went wrong")
		expected := `{"level":"error","error":"a wild error appeared","message":"something went wrong"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected log output %q, got %q", expected, buf.String())
		}
	})

	t.Run("sub-logger with context", func(t *testing.T) {
		buf.Reset()
		subLogger := logger.With().Str("request_id", "123").Logger()
		subLogger.Info().Msg("processing request")
		expected := `{"level":"info","request_id":"123","message":"processing request"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected log output %q, got %q", expected, buf.String())
		}
	})

	t.Run("context with OpenTelemetry trace", func(t *testing.T) {
		buf.Reset()

		// Create a mock OpenTelemetry trace context
		traceID := trace.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
		spanID := trace.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})
		scc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		})
		ctx := trace.ContextWithSpanContext(context.Background(), scc)

		logger.Ctx(ctx).Info().Msg("doing work inside a trace")

		expected := fmt.Sprintf(`{"level":"info","trace_id":"%s","span_id":"%s","message":"doing work inside a trace"}`+"\n",
			traceID.String(), spanID.String())
		if buf.String() != expected {
			t.Errorf("Expected log output %q, got %q", expected, buf.String())
		}
	})

	t.Run("log with bool field", func(t *testing.T) {
		buf.Reset()
		logger.Info().Bool("is_active", true).Msg("user status")
		expected := `{"level":"info","is_active":true,"message":"user status"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected log output %q, got %q", expected, buf.String())
		}
	})

	t.Run("log with float64 field", func(t *testing.T) {
		buf.Reset()
		logger.Info().Float64("price", 99.99).Msg("item price")
		expected := `{"level":"info","price":99.99,"message":"item price"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected log output %q, got %q", expected, buf.String())
		}
	})

	t.Run("log with time field", func(t *testing.T) {
		buf.Reset()
		eventTime := time.Date(2025, time.July, 13, 15, 30, 0, 0, time.UTC)
		logger.Info().Time("event_time", eventTime).Msg("event occurred")
		expected := `{"level":"info","event_time":"2025-07-13T15:30:00Z","message":"event occurred"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected log output %q, got %q", expected, buf.String())
		}
	})

	t.Run("log with duration field", func(t *testing.T) {
		buf.Reset()
		d := 5 * time.Second
		logger.Info().Dur("duration", d).Msg("operation took")
		expected := `{"level":"info","duration":5000000000,"message":"operation took"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected log output %q, got %q", expected, buf.String())
		}
	})

	t.Run("log with uint field", func(t *testing.T) {
		buf.Reset()
		logger.Info().Uint("count", 12345).Msg("item count")
		expected := `{"level":"info","count":12345,"message":"item count"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected log output %q, got %q", expected, buf.String())
		}
	})

	t.Run("log with any field", func(t *testing.T) {
		buf.Reset()
		type User struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		user := User{Name: "John Doe", Age: 30}
		logger.Info().Any("user", user).Msg("user info")
		expected := `{"level":"info","user":{"name":"John Doe","age":30},"message":"user info"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected log output %q, got %q", expected, buf.String())
		}
	})
}

func TestConsoleHandler(t *testing.T) {
	var buf bytes.Buffer

	logger := New(NewConsoleHandler(&buf))

	t.Run("simple log", func(t *testing.T) {
		buf.Reset()
		logger.Info().Str("foo", "bar").Msg("hello world")
		// Expected output will include ANSI color codes and a human-readable format.
		// We'll use a regex to match the dynamic parts like timestamp.
		expectedRegex := `^\x1b\[34minfo\x1b\[0m\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z\] hello world foo=bar\n$`
		if !regexp.MustCompile(expectedRegex).MatchString(buf.String()) {
			t.Errorf("Expected log output to match regex %q, got %q", expectedRegex, buf.String())
		}
	})
}
