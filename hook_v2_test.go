package bolt

import (
	"bytes"
	"strings"
	"sync/atomic"
	"testing"
)

// fieldCounter is an EventHook that walks fields and accumulates a count.
// Used to verify WalkFields visits every encoded field once.
type fieldCounter struct {
	visited int64
}

func (h *fieldCounter) Run(e *Event, _ string) bool {
	count := e.WalkFields(func(_, _ []byte) bool { return true })
	atomic.AddInt64(&h.visited, int64(count))
	return true
}

// redactingHook is an EventHook that suppresses any event containing a
// field whose key matches one of the configured deny entries. Used to
// demonstrate the AI-review use case (sensitive content gating).
type redactingHook struct {
	deny []string
}

func (h *redactingHook) Run(e *Event, _ string) bool {
	allow := true
	e.WalkFields(func(key, _ []byte) bool {
		for _, d := range h.deny {
			if string(key) == d {
				allow = false
				return false
			}
		}
		return true
	})
	return allow
}

// taggingHook adds a constant field to every event. Demonstrates that
// EventHook may mutate via the public Str/Int/etc methods.
type taggingHook struct{}

func (h *taggingHook) Run(e *Event, _ string) bool {
	e.Str("tenant", "acme-corp")
	return true
}

// loggingLevelHook captures the level reported by Event.Level for assertion.
type loggingLevelHook struct {
	last Level
}

func (h *loggingLevelHook) Run(e *Event, _ string) bool {
	h.last = e.Level()
	return true
}

func TestEventHook_RunsAndSeesFields(t *testing.T) {
	var buf bytes.Buffer
	counter := &fieldCounter{}
	logger := New(NewJSONHandler(&buf)).AddEventHook(counter)

	logger.Info().Str("user", "alice").Int("status", 200).Bool("ok", true).Msg("done")

	if got := atomic.LoadInt64(&counter.visited); got != 4 {
		// 4 fields: level, user, status, ok
		t.Errorf("WalkFields count = %d, want 4", got)
	}
	if !strings.Contains(buf.String(), `"message":"done"`) {
		t.Errorf("event missing from output: %q", buf.String())
	}
}

func TestEventHook_SuppressesEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf)).AddEventHook(&redactingHook{deny: []string{"password"}})

	logger.Info().Str("user", "alice").Msg("login ok")
	if got := buf.String(); !strings.Contains(got, `"user":"alice"`) {
		t.Errorf("non-sensitive event was wrongly suppressed: %q", got)
	}

	buf.Reset()
	logger.Info().Str("user", "alice").Str("password", "secret").Msg("login")
	if got := buf.String(); got != "" {
		t.Errorf("event with sensitive field should have been suppressed, got %q", got)
	}
}

func TestEventHook_AddsField(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf)).AddEventHook(&taggingHook{})

	logger.Info().Str("path", "/api").Msg("request")
	out := buf.String()
	if !strings.Contains(out, `"tenant":"acme-corp"`) {
		t.Errorf("tagging hook field missing from output: %q", out)
	}
	if !strings.Contains(out, `"path":"/api"`) {
		t.Errorf("original field missing from output: %q", out)
	}
}

func TestEventHook_LevelAccessor(t *testing.T) {
	var buf bytes.Buffer
	hook := &loggingLevelHook{}
	logger := New(NewJSONHandler(&buf)).AddEventHook(hook)

	logger.Warn().Msg("warning")
	if hook.last != WARN {
		t.Errorf("hook saw level %v, want WARN", hook.last)
	}

	logger.Error().Msg("err")
	if hook.last != ERROR {
		t.Errorf("hook saw level %v, want ERROR", hook.last)
	}
}

func TestEventHook_BufferAccessorReturnsAlias(t *testing.T) {
	var buf bytes.Buffer
	var snapshot []byte
	captureHook := EventHookFunc(func(e *Event, _ string) bool {
		// Buffer returns a slice that aliases the event's internal buffer.
		// Caller is documented to copy if retention is needed.
		raw := e.Buffer()
		snapshot = make([]byte, len(raw))
		copy(snapshot, raw)
		return true
	})
	logger := New(NewJSONHandler(&buf)).AddEventHook(captureHook)

	logger.Info().Str("k", "v").Msg("ok")
	if !bytes.HasPrefix(snapshot, []byte(`{"level":"info"`)) {
		t.Errorf("snapshot prefix wrong: %q", snapshot)
	}
	if !bytes.Contains(snapshot, []byte(`"k":"v"`)) {
		t.Errorf("snapshot missing field: %q", snapshot)
	}
	// Per the contract, message is NOT yet in the buffer when EventHooks run.
	if bytes.Contains(snapshot, []byte(`"message"`)) {
		t.Errorf("snapshot should not contain message yet: %q", snapshot)
	}
}

func TestEventHook_LegacyHookSuppressionShortCircuits(t *testing.T) {
	var buf bytes.Buffer
	eventHookCalled := int64(0)
	logger := New(NewJSONHandler(&buf)).
		AddHook(HookFunc(func(_ Level, _ string) bool { return false })).
		AddEventHook(EventHookFunc(func(_ *Event, _ string) bool {
			atomic.AddInt64(&eventHookCalled, 1)
			return true
		}))

	logger.Info().Msg("dropped by legacy hook")
	if got := atomic.LoadInt64(&eventHookCalled); got != 0 {
		t.Errorf("EventHook should not run after legacy Hook suppresses; got %d invocations", got)
	}
	if buf.Len() != 0 {
		t.Errorf("event should have been suppressed; got %q", buf.String())
	}
}

func TestEventHook_CoexistsWithLegacyHook(t *testing.T) {
	var buf bytes.Buffer
	legacyCount := int64(0)
	eventCount := int64(0)
	logger := New(NewJSONHandler(&buf)).
		AddHook(HookFunc(func(_ Level, _ string) bool {
			atomic.AddInt64(&legacyCount, 1)
			return true
		})).
		AddEventHook(EventHookFunc(func(_ *Event, _ string) bool {
			atomic.AddInt64(&eventCount, 1)
			return true
		}))

	logger.Info().Msg("first")
	logger.Info().Msg("second")

	if got := atomic.LoadInt64(&legacyCount); got != 2 {
		t.Errorf("legacy hook calls = %d, want 2", got)
	}
	if got := atomic.LoadInt64(&eventCount); got != 2 {
		t.Errorf("event hook calls = %d, want 2", got)
	}
}

// HookFunc is a function-typed adapter for [Hook]. Defined here in the
// test file rather than the public API; promote to public if user demand
// emerges.
type HookFunc func(level Level, msg string) bool

func (f HookFunc) Run(level Level, msg string) bool { return f(level, msg) }

// EventHookFunc is a function-typed adapter for [EventHook]. Test-only.
type EventHookFunc func(e *Event, msg string) bool

func (f EventHookFunc) Run(e *Event, msg string) bool { return f(e, msg) }
