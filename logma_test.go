package logma

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func BenchmarkZeroAllocation(b *testing.B) {
	logger := New(NewJSONHandler(&bytes.Buffer{}))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info().Str("hello", "world").Msg("test")
	}
}

func TestJSONHandler(t *testing.T) {
	var buf bytes.Buffer

	// Create a configuration without timestamps for predictable test output
	config := DefaultEncoderConfig()
	config.IncludeTime = false
	logger := New(NewJSONHandlerWithConfig(&buf, config))

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

	t.Run("log with stack trace", func(t *testing.T) {
		buf.Reset()
		logger.Error().Stack().Msg("error with stack")
		// We expect a stack trace, so we'll check for a common pattern.
		expectedRegex := `"stack":"(.|\n)*logma_test.go`
		if !regexp.MustCompile(expectedRegex).MatchString(buf.String()) {
			t.Errorf("Expected log output to contain stack trace, got %q", buf.String())
		}
	})

	t.Run("log with bytes field", func(t *testing.T) {
		buf.Reset()
		data := []byte("hello world")
		logger.Info().Bytes("data", data).Msg("binary data")
		expected := `{"level":"info","data":"aGVsbG8gd29ybGQ=","message":"binary data"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected log output %q, got %q", expected, buf.String())
		}
	})

	t.Run("log with hex field", func(t *testing.T) {
		buf.Reset()
		data := []byte{0x12, 0x34, 0xab, 0xcd}
		logger.Info().Hex("hash", data).Msg("hex data")
		expected := `{"level":"info","hash":"1234abcd","message":"hex data"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected log output %q, got %q", expected, buf.String())
		}
	})

	t.Run("log with multi-writer", func(t *testing.T) {
		buf.Reset()
		var buf1, buf2 bytes.Buffer
		multiWriter := NewMultiWriter(&buf1, &buf2)
		multiConfig := DefaultEncoderConfig()
		multiConfig.IncludeTime = false
		multiLogger := New(NewJSONHandlerWithConfig(multiWriter, multiConfig))
		multiLogger.Info().Msg("multi-writer test")
		expected := `{"level":"info","message":"multi-writer test"}` + "\n"
		if buf1.String() != expected {
			t.Errorf("Expected buf1 to contain %q, got %q", expected, buf1.String())
		}
		if buf2.String() != expected {
			t.Errorf("Expected buf2 to contain %q, got %q", expected, buf2.String())
		}
	})
	
	t.Run("test context-aware logging", func(t *testing.T) {
		buf.Reset()
		
		// Register a context extractor
		RegisterContextExtractor(DefaultRequestIDExtractor)
		
		// Create context with request ID
		ctx := context.WithValue(context.Background(), RequestIDKey, "req-123")
		
		logger.WithContext(ctx).Info().Msg("processing request")
		
		// Should contain request_id field
		result := buf.String()
		if !strings.Contains(result, `"request_id":"req-123"`) {
			t.Errorf("Expected log to contain request_id, got %q", result)
		}
	})
	
	t.Run("test error with stack", func(t *testing.T) {
		buf.Reset()
		err := errors.New("test error")
		
		logger.Error().ErrorWithStack(err).Msg("error occurred")
		
		result := buf.String()
		if !strings.Contains(result, `"error":"test error"`) {
			t.Errorf("Expected log to contain error field, got %q", result)
		}
	})
	
	t.Run("test sampling", func(t *testing.T) {
		var samplingBuf bytes.Buffer
		
		// Create logger with sampling that logs every 2nd entry
		samplingConfig := DefaultEncoderConfig()
		samplingConfig.IncludeTime = false
		sampledLogger := New(NewJSONHandlerWithConfig(&samplingBuf, samplingConfig)).SetSampler(NewFixedSampler(2))
		
		sampledLogger.Info().Msg("first")  // Should be dropped
		sampledLogger.Info().Msg("second") // Should be logged
		sampledLogger.Info().Msg("third")  // Should be dropped
		sampledLogger.Info().Msg("fourth") // Should be logged
		
		lines := strings.Split(strings.TrimSpace(samplingBuf.String()), "\n")
		if len(lines) != 2 {
			t.Errorf("Expected 2 log lines due to sampling, got %d: %v", len(lines), lines)
		}
	})
	
	t.Run("test panic and dpanic levels", func(t *testing.T) {
		buf.Reset()
		
		// Test DPanic doesn't panic in non-development mode
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("DPanic should not panic in production mode")
				}
			}()
			logger.DPanic().Msg("dpanic test")
		}()
		
		// Test Panic does panic
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Panic should panic")
				}
			}()
			logger.Panic().Msg("panic test")
		}()
	})
	
	t.Run("test advanced customization", func(t *testing.T) {
		var customBuf bytes.Buffer
		
		// Create custom configuration
		config := EncoderConfig{
			TimeKey:       "ts",
			LevelKey:      "lvl", 
			MessageKey:    "msg",
			ErrorKey:      "err",
			StacktraceKey: "stack",
			CallerKey:     "source",
			TimeFormat:    "2006-01-02",
			IncludeTime:   true,
			IncludeCaller: true,
		}
		
		customLogger := New(NewJSONHandlerWithConfig(&customBuf, config))
		customLogger.Info().Str("test", "value").Msg("custom format")
		
		result := customBuf.String()
		
		// Should contain custom keys
		if !strings.Contains(result, `"lvl":"info"`) {
			t.Errorf("Expected custom level key 'lvl', got %q", result)
		}
		if !strings.Contains(result, `"msg":"custom format"`) {
			t.Errorf("Expected custom message key 'msg', got %q", result)
		}
		if !strings.Contains(result, `"ts":"`) {
			t.Errorf("Expected custom time key 'ts', got %q", result)
		}
		if !strings.Contains(result, `"source":"`) {
			t.Errorf("Expected caller info with key 'source', got %q", result)
		}
	})
}
