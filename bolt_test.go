package bolt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func BenchmarkZeroAllocation(b *testing.B) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info().Str("hello", "world").Msg("test")
	}
}

func TestJSONHandlerBasic(t *testing.T) {
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
}

func TestJSONHandlerOpenTelemetry(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

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
}

func TestJSONHandlerFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

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
		// Our custom formatter provides 6 decimal precision
		expected := `{"level":"info","price":99.989999,"message":"item price"}` + "\n"
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
		// We'll use a regex to match the dynamic parts like timestamp (with microseconds).
		expectedRegex := `^\x1b\[34minfo\x1b\[0m\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z\] hello world foo=bar\n$`
		if !regexp.MustCompile(expectedRegex).MatchString(buf.String()) {
			t.Errorf("Expected log output to match regex %q, got %q", expectedRegex, buf.String())
		}
	})
}

// --- Feature 1: Uint8/Uint16/Uint32 ---

func TestUintFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	t.Run("Uint8", func(t *testing.T) {
		buf.Reset()
		logger.Info().Uint8("val", 255).Msg("u8")
		expected := `{"level":"info","val":255,"message":"u8"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("Uint8 zero", func(t *testing.T) {
		buf.Reset()
		logger.Info().Uint8("val", 0).Msg("u8z")
		expected := `{"level":"info","val":0,"message":"u8z"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("Uint16", func(t *testing.T) {
		buf.Reset()
		logger.Info().Uint16("val", 65535).Msg("u16")
		expected := `{"level":"info","val":65535,"message":"u16"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("Uint32", func(t *testing.T) {
		buf.Reset()
		logger.Info().Uint32("val", 4294967295).Msg("u32")
		expected := `{"level":"info","val":4294967295,"message":"u32"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("disabled event", func(t *testing.T) {
		buf.Reset()
		logger.SetLevel(ERROR)
		defer logger.SetLevel(TRACE)
		logger.Info().Uint8("val", 1).Uint16("val", 2).Uint32("val", 3).Msg("skip")
		if buf.Len() != 0 {
			t.Errorf("Expected no output, got %q", buf.String())
		}
	})
}

// --- Feature 2: CallerSkip ---

func logHelper(logger *Logger) {
	logger.Info().CallerSkip(1).Msg("from helper")
}

func TestCallerSkip(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	t.Run("skip 0 reports this function", func(t *testing.T) {
		buf.Reset()
		logger.Info().CallerSkip(0).Msg("here")
		if !strings.Contains(buf.String(), `"caller":"bolt_test.go:`) {
			t.Errorf("Expected caller from bolt_test.go, got %q", buf.String())
		}
	})

	t.Run("skip 1 from helper reports test function", func(t *testing.T) {
		buf.Reset()
		logHelper(logger)
		if !strings.Contains(buf.String(), `"caller":"bolt_test.go:`) {
			t.Errorf("Expected caller from bolt_test.go, got %q", buf.String())
		}
	})

	t.Run("disabled event", func(t *testing.T) {
		buf.Reset()
		logger.SetLevel(ERROR)
		defer logger.SetLevel(TRACE)
		logger.Info().CallerSkip(0).Msg("skip")
		if buf.Len() != 0 {
			t.Errorf("Expected no output, got %q", buf.String())
		}
	})
}

// --- Feature 3: Stringer ---

type testStringer struct{ val string }

func (s *testStringer) String() string { return s.val }

func TestStringer(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	t.Run("non-nil stringer", func(t *testing.T) {
		buf.Reset()
		s := &testStringer{val: "hello"}
		logger.Info().Stringer("val", s).Msg("s")
		expected := `{"level":"info","val":"hello","message":"s"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("nil stringer", func(t *testing.T) {
		buf.Reset()
		logger.Info().Stringer("val", nil).Msg("s")
		expected := `{"level":"info","val":null,"message":"s"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("disabled event", func(t *testing.T) {
		buf.Reset()
		logger.SetLevel(ERROR)
		defer logger.SetLevel(TRACE)
		logger.Info().Stringer("val", &testStringer{val: "x"}).Msg("skip")
		if buf.Len() != 0 {
			t.Errorf("Expected no output, got %q", buf.String())
		}
	})
}

// --- Feature 4: Ints / Strs ---

func TestInts(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	t.Run("values", func(t *testing.T) {
		buf.Reset()
		logger.Info().Ints("ids", []int{1, 2, 3}).Msg("i")
		expected := `{"level":"info","ids":[1,2,3],"message":"i"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		buf.Reset()
		logger.Info().Ints("ids", []int{}).Msg("i")
		expected := `{"level":"info","ids":[],"message":"i"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("nil slice", func(t *testing.T) {
		buf.Reset()
		logger.Info().Ints("ids", nil).Msg("i")
		expected := `{"level":"info","ids":[],"message":"i"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("negative values", func(t *testing.T) {
		buf.Reset()
		logger.Info().Ints("ids", []int{-1, 0, 42}).Msg("i")
		expected := `{"level":"info","ids":[-1,0,42],"message":"i"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})
}

func TestStrs(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	t.Run("values", func(t *testing.T) {
		buf.Reset()
		logger.Info().Strs("tags", []string{"a", "b", "c"}).Msg("s")
		expected := `{"level":"info","tags":["a","b","c"],"message":"s"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		buf.Reset()
		logger.Info().Strs("tags", []string{}).Msg("s")
		expected := `{"level":"info","tags":[],"message":"s"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("nil slice", func(t *testing.T) {
		buf.Reset()
		logger.Info().Strs("tags", nil).Msg("s")
		expected := `{"level":"info","tags":[],"message":"s"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("escaping", func(t *testing.T) {
		buf.Reset()
		logger.Info().Strs("tags", []string{`a"b`, "c\\d"}).Msg("s")
		expected := `{"level":"info","tags":["a\"b","c\\d"],"message":"s"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})
}

// --- Feature 5: IPAddr ---

func TestIPAddr(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	t.Run("IPv4", func(t *testing.T) {
		buf.Reset()
		logger.Info().IPAddr("ip", net.IPv4(192, 168, 1, 1)).Msg("addr")
		expected := `{"level":"info","ip":"192.168.1.1","message":"addr"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("IPv4 loopback", func(t *testing.T) {
		buf.Reset()
		logger.Info().IPAddr("ip", net.IPv4(127, 0, 0, 1)).Msg("lo")
		expected := `{"level":"info","ip":"127.0.0.1","message":"lo"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("IPv6 loopback", func(t *testing.T) {
		buf.Reset()
		logger.Info().IPAddr("ip", net.IPv6loopback).Msg("lo6")
		expected := `{"level":"info","ip":"0000:0000:0000:0000:0000:0000:0000:0001","message":"lo6"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("nil IP", func(t *testing.T) {
		buf.Reset()
		logger.Info().IPAddr("ip", nil).Msg("nil")
		expected := `{"level":"info","ip":null,"message":"nil"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("disabled event", func(t *testing.T) {
		buf.Reset()
		logger.SetLevel(ERROR)
		defer logger.SetLevel(TRACE)
		logger.Info().IPAddr("ip", net.IPv4(1, 2, 3, 4)).Msg("skip")
		if buf.Len() != 0 {
			t.Errorf("Expected no output, got %q", buf.String())
		}
	})
}

// --- Feature 6: Dict ---

func TestDict(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	t.Run("basic dict", func(t *testing.T) {
		buf.Reset()
		logger.Info().Dict("user", func(d *Event) {
			d.Str("name", "alice").Int("age", 30)
		}).Msg("d")
		expected := `{"level":"info","user":{"name":"alice","age":30},"message":"d"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("empty dict", func(t *testing.T) {
		buf.Reset()
		logger.Info().Dict("empty", func(d *Event) {}).Msg("d")
		expected := `{"level":"info","empty":{},"message":"d"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("nested dict", func(t *testing.T) {
		buf.Reset()
		logger.Info().Dict("outer", func(d *Event) {
			d.Str("key", "val").Dict("inner", func(d2 *Event) {
				d2.Int("n", 1)
			})
		}).Msg("d")
		expected := `{"level":"info","outer":{"key":"val","inner":{"n":1}},"message":"d"}` + "\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("disabled event", func(t *testing.T) {
		buf.Reset()
		logger.SetLevel(ERROR)
		defer logger.SetLevel(TRACE)
		logger.Info().Dict("x", func(d *Event) {
			d.Str("a", "b")
		}).Msg("skip")
		if buf.Len() != 0 {
			t.Errorf("Expected no output, got %q", buf.String())
		}
	})
}

// --- Feature 7: MultiHandler ---

func TestMultiHandler(t *testing.T) {
	t.Run("fan-out to multiple handlers", func(t *testing.T) {
		var buf1, buf2 bytes.Buffer
		h := MultiHandler(NewJSONHandler(&buf1), NewJSONHandler(&buf2))
		logger := New(h)
		logger.Info().Str("k", "v").Msg("multi")
		expected := `{"level":"info","k":"v","message":"multi"}` + "\n"
		if buf1.String() != expected {
			t.Errorf("buf1: expected %q, got %q", expected, buf1.String())
		}
		if buf2.String() != expected {
			t.Errorf("buf2: expected %q, got %q", expected, buf2.String())
		}
	})

	t.Run("error from one handler", func(t *testing.T) {
		var buf bytes.Buffer
		failing := &failingHandler{err: errors.New("write failed")}
		h := MultiHandler(failing, NewJSONHandler(&buf))
		logger := New(h)

		var gotErr error
		logger.SetErrorHandler(func(err error) { gotErr = err })
		logger.Info().Msg("test")

		if gotErr == nil {
			t.Error("Expected error from MultiHandler")
		}
		// Second handler should still have received the event
		if !strings.Contains(buf.String(), `"message":"test"`) {
			t.Errorf("Expected second handler to receive event, got %q", buf.String())
		}
	})

	t.Run("empty handlers", func(t *testing.T) {
		h := MultiHandler()
		logger := New(h)
		// Should not panic
		logger.Info().Msg("no handlers")
	})
}

type failingHandler struct{ err error }

func (f *failingHandler) Write(_ *Event) error { return f.err }

// --- Feature 8: Hook + AddHook ---

type testHook struct {
	called  int
	lastLvl Level
	lastMsg string
	allow   bool
}

func (h *testHook) Run(level Level, msg string) bool {
	h.called++
	h.lastLvl = level
	h.lastMsg = msg
	return h.allow
}

func TestHook(t *testing.T) {
	t.Run("hook receives level and message", func(t *testing.T) {
		var buf bytes.Buffer
		hook := &testHook{allow: true}
		logger := New(NewJSONHandler(&buf)).AddHook(hook)

		logger.Info().Str("k", "v").Msg("hello hooks")

		if hook.called != 1 {
			t.Errorf("Expected hook called 1 time, got %d", hook.called)
		}
		if hook.lastLvl != INFO {
			t.Errorf("Expected INFO level, got %v", hook.lastLvl)
		}
		if hook.lastMsg != "hello hooks" {
			t.Errorf("Expected message 'hello hooks', got %q", hook.lastMsg)
		}
		if !strings.Contains(buf.String(), `"message":"hello hooks"`) {
			t.Errorf("Expected log output, got %q", buf.String())
		}
	})

	t.Run("hook suppresses event", func(t *testing.T) {
		var buf bytes.Buffer
		hook := &testHook{allow: false}
		logger := New(NewJSONHandler(&buf)).AddHook(hook)

		logger.Info().Msg("suppressed")

		if hook.called != 1 {
			t.Errorf("Expected hook called 1 time, got %d", hook.called)
		}
		if buf.Len() != 0 {
			t.Errorf("Expected no output, got %q", buf.String())
		}
	})

	t.Run("hook chain - first suppresses", func(t *testing.T) {
		var buf bytes.Buffer
		hook1 := &testHook{allow: false}
		hook2 := &testHook{allow: true}
		logger := New(NewJSONHandler(&buf)).AddHook(hook1).AddHook(hook2)

		logger.Info().Msg("chain")

		if hook1.called != 1 {
			t.Errorf("Expected hook1 called 1 time, got %d", hook1.called)
		}
		if hook2.called != 0 {
			t.Errorf("Expected hook2 not called, got %d", hook2.called)
		}
		if buf.Len() != 0 {
			t.Errorf("Expected no output, got %q", buf.String())
		}
	})

	t.Run("hooks inherited by child logger", func(t *testing.T) {
		var buf bytes.Buffer
		hook := &testHook{allow: true}
		parent := New(NewJSONHandler(&buf)).AddHook(hook)
		child := parent.With().Str("ctx", "val").Logger()

		child.Info().Msg("child msg")

		if hook.called != 1 {
			t.Errorf("Expected hook called from child, got %d", hook.called)
		}
		if !strings.Contains(buf.String(), `"ctx":"val"`) {
			t.Errorf("Expected context in output, got %q", buf.String())
		}
	})

	t.Run("disabled event does not trigger hooks", func(t *testing.T) {
		var buf bytes.Buffer
		hook := &testHook{allow: true}
		logger := New(NewJSONHandler(&buf)).AddHook(hook).SetLevel(ERROR)

		logger.Info().Msg("filtered")

		if hook.called != 0 {
			t.Errorf("Expected hook not called for filtered event, got %d", hook.called)
		}
	})
}

// --- Feature 9: SampleHook ---

func TestSampleHook(t *testing.T) {
	t.Run("sampling rate", func(t *testing.T) {
		var buf bytes.Buffer
		hook := NewSampleHook(10)
		logger := New(NewJSONHandler(&buf)).AddHook(hook)

		for i := 0; i < 100; i++ {
			logger.Info().Int("i", i).Msg("sample")
		}

		logCount := bytes.Count(buf.Bytes(), []byte("\n"))
		if logCount != 10 {
			t.Errorf("Expected 10 sampled logs (1 in 10), got %d", logCount)
		}
	})

	t.Run("n=0 passes all", func(t *testing.T) {
		var buf bytes.Buffer
		hook := NewSampleHook(0)
		logger := New(NewJSONHandler(&buf)).AddHook(hook)

		for i := 0; i < 10; i++ {
			logger.Info().Msg("all")
		}

		logCount := bytes.Count(buf.Bytes(), []byte("\n"))
		if logCount != 10 {
			t.Errorf("Expected all 10 logs with n=0, got %d", logCount)
		}
	})

	t.Run("n=1 passes all", func(t *testing.T) {
		var buf bytes.Buffer
		hook := NewSampleHook(1)
		logger := New(NewJSONHandler(&buf)).AddHook(hook)

		for i := 0; i < 10; i++ {
			logger.Info().Msg("all")
		}

		logCount := bytes.Count(buf.Bytes(), []byte("\n"))
		if logCount != 10 {
			t.Errorf("Expected all 10 logs with n=1, got %d", logCount)
		}
	})

	t.Run("concurrent sampling", func(t *testing.T) {
		buf := &ThreadSafeBuffer{}
		hook := NewSampleHook(5)
		logger := New(NewJSONHandler(buf)).AddHook(hook)

		var counter int64
		done := make(chan struct{})
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					logger.Info().Msg("concurrent")
					atomic.AddInt64(&counter, 1)
				}
				done <- struct{}{}
			}()
		}
		for i := 0; i < 10; i++ {
			<-done
		}

		total := atomic.LoadInt64(&counter)
		logCount := bytes.Count(buf.Bytes(), []byte("\n"))
		expectedSampled := int(total) / 5
		if logCount != expectedSampled {
			t.Errorf("Expected ~%d sampled logs, got %d (total events: %d)", expectedSampled, logCount, total)
		}
	})
}

// --- Feature 10: NewLevelWriter ---

func TestNewLevelWriter(t *testing.T) {
	t.Run("writes at specified level", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))
		w := NewLevelWriter(logger, WARN)

		_, _ = w.Write([]byte("warning message"))

		if !strings.Contains(buf.String(), `"level":"warn"`) {
			t.Errorf("Expected warn level, got %q", buf.String())
		}
		if !strings.Contains(buf.String(), `"message":"warning message"`) {
			t.Errorf("Expected message, got %q", buf.String())
		}
	})

	t.Run("trims trailing newline", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))
		w := NewLevelWriter(logger, INFO)

		_, _ = w.Write([]byte("trimmed\n"))

		if strings.Contains(buf.String(), `"message":"trimmed\n"`) {
			t.Errorf("Expected newline to be trimmed, got %q", buf.String())
		}
		if !strings.Contains(buf.String(), `"message":"trimmed"`) {
			t.Errorf("Expected trimmed message, got %q", buf.String())
		}
	})

	t.Run("respects level filtering", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf)).SetLevel(ERROR)
		w := NewLevelWriter(logger, INFO)

		n, err := w.Write([]byte("filtered"))

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if n != len("filtered") {
			t.Errorf("Expected n=%d, got %d", len("filtered"), n)
		}
		if buf.Len() != 0 {
			t.Errorf("Expected no output, got %q", buf.String())
		}
	})

	t.Run("stdlib log integration", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))
		w := NewLevelWriter(logger, ERROR)
		stdlog := log.New(w, "", 0)

		stdlog.Print("stdlib error")

		if !strings.Contains(buf.String(), `"level":"error"`) {
			t.Errorf("Expected error level, got %q", buf.String())
		}
		if !strings.Contains(buf.String(), `"message":"stdlib error"`) {
			t.Errorf("Expected message, got %q", buf.String())
		}
	})
}
