package genai_test

import (
	"bytes"
	"strings"
	"sync/atomic"
	"testing"

	"go.klarlabs.de/bolt"
	"go.klarlabs.de/bolt/genai"
)

func TestRedactHook_DefaultDenyList(t *testing.T) {
	var buf bytes.Buffer
	logger := bolt.New(bolt.NewJSONHandler(&buf)).
		AddEventHook(genai.NewRedactHook())

	logger.Info().Str("user", "alice").Msg("login ok")
	if !strings.Contains(buf.String(), `"user":"alice"`) {
		t.Errorf("non-sensitive event suppressed: %q", buf.String())
	}

	for _, key := range []string{"prompt", "completion", "messages", "input", "output", "api_key", "token", "authorization"} {
		buf.Reset()
		logger.Info().Str(key, "secret-content").Msg("ignored")
		if buf.Len() != 0 {
			t.Errorf("event with sensitive key %q should have been suppressed, got %q", key, buf.String())
		}
	}
}

func TestRedactHook_CustomDenyList(t *testing.T) {
	var buf bytes.Buffer
	logger := bolt.New(bolt.NewJSONHandler(&buf)).
		AddEventHook(genai.NewRedactHook("internal_secret", "session_id"))

	buf.Reset()
	// Default deny list keys are NOT suppressed when custom list passed.
	logger.Info().Str("prompt", "hello").Msg("ok")
	if !strings.Contains(buf.String(), `"prompt":"hello"`) {
		t.Errorf("custom deny list should not suppress 'prompt'; got %q", buf.String())
	}

	buf.Reset()
	logger.Info().Str("internal_secret", "x").Msg("blocked")
	if buf.Len() != 0 {
		t.Errorf("custom deny key should suppress; got %q", buf.String())
	}
}

func TestAdaptiveSampler_AlwaysKeepGenAIFields(t *testing.T) {
	var buf bytes.Buffer
	logger := bolt.New(bolt.NewJSONHandler(&buf)).
		AddEventHook(genai.NewAdaptiveSampler(1000))

	// 5 debug events with gen_ai.* — must all pass.
	for i := 0; i < 5; i++ {
		buf.Reset()
		logger.Debug().Str("gen_ai.system", "openai").Msg("trace")
		if buf.Len() == 0 {
			t.Errorf("iter %d: gen_ai-tagged debug should always pass, got suppressed", i)
		}
	}
}

func TestAdaptiveSampler_AlwaysKeepErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := bolt.New(bolt.NewJSONHandler(&buf)).
		AddEventHook(genai.NewAdaptiveSampler(1000))

	for i := 0; i < 5; i++ {
		buf.Reset()
		logger.Error().Str("user", "alice").Msg("boom")
		if buf.Len() == 0 {
			t.Errorf("iter %d: error-level event always-kept, got suppressed", i)
		}
	}
}

func TestAdaptiveSampler_LowImportanceSampled(t *testing.T) {
	var buf bytes.Buffer
	logger := bolt.New(bolt.NewJSONHandler(&buf)).
		AddEventHook(genai.NewAdaptiveSampler(10))

	const n = 1000
	emitted := 0
	for i := 0; i < n; i++ {
		buf.Reset()
		logger.Debug().Str("user", "alice").Msg("noise")
		if buf.Len() != 0 {
			emitted++
		}
	}
	// Counter wraps at every 10th event, so we expect ~100 to pass.
	if emitted < 70 || emitted > 130 {
		t.Errorf("AdaptiveSampler emitted %d/%d (expected ~100)", emitted, n)
	}
}

func TestAdaptiveSampler_NoSamplingWhenN0Or1(t *testing.T) {
	for _, n := range []uint32{0, 1} {
		var buf bytes.Buffer
		hook := genai.NewAdaptiveSampler(n)
		logger := bolt.New(bolt.NewJSONHandler(&buf)).AddEventHook(hook)

		emitted := 0
		for i := 0; i < 50; i++ {
			buf.Reset()
			logger.Debug().Str("user", "x").Msg("ok")
			if buf.Len() != 0 {
				emitted++
			}
		}
		if emitted != 50 {
			t.Errorf("N=%d should keep everything; got %d/50", n, emitted)
		}
	}
}

func TestAdaptiveSampler_CounterIsAtomic(t *testing.T) {
	hook := genai.NewAdaptiveSampler(7)
	var buf bytes.Buffer
	logger := bolt.New(bolt.NewJSONHandler(&buf)).AddEventHook(hook)

	const goroutines = 16
	const perGoroutine = 1000
	done := make(chan struct{})
	var emitted int64
	for g := 0; g < goroutines; g++ {
		go func() {
			defer func() { done <- struct{}{} }()
			local := bytes.NewBuffer(nil)
			ll := bolt.New(bolt.NewJSONHandler(local)).AddEventHook(hook)
			for i := 0; i < perGoroutine; i++ {
				local.Reset()
				ll.Debug().Str("user", "x").Msg("ok")
				if local.Len() != 0 {
					atomic.AddInt64(&emitted, 1)
				}
			}
		}()
	}
	for g := 0; g < goroutines; g++ {
		<-done
	}
	_ = logger // silence unused; the original logger is the regression check that AddEventHook with shared *AdaptiveSampler still works.

	total := goroutines * perGoroutine
	want := int64(total / 7)
	got := atomic.LoadInt64(&emitted)
	// Allow ±15% drift across 16 goroutines.
	if got < want*85/100 || got > want*115/100 {
		t.Errorf("concurrent sampler emitted %d/%d (expected ~%d, ±15%%)", got, total, want)
	}
}
