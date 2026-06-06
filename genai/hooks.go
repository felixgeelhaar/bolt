// EventHook helpers for AI workloads.
//
// The bolt core ships a generic [bolt.EventHook] surface (see
// hook_v2_test.go in the parent module). This file provides two
// pre-built hooks for the common GenAI use cases the multi-expert
// review highlighted as "the two things that genuinely belong at the
// logger layer":
//
//   1. RedactKeys — drop events whose payload contains any of a
//      caller-supplied set of sensitive keys. The default deny list
//      covers prompt / completion / messages / api_key / token; pass a
//      custom list for project-specific concerns.
//
//   2. AdaptiveSample — sample low-importance events at a caller-set
//      rate, but always keep events with sticky keys (default:
//      `gen_ai.*` namespace and any error-level event). This makes
//      token-stream debug logs cheap without losing GenAI breadcrumbs.
//
// Both hooks compose freely with Logger.AddHook(bolt.NewSampleHook(N))
// for the simple uniform case. Compose by AddEventHook order: hooks
// run sequentially, first false suppresses.
package genai

import (
	"strings"
	"sync/atomic"

	"go.klarlabs.de/bolt"
)

// DefaultDenyKeys is the default sensitive-key list used by
// [NewRedactHook] when no override is provided. Matches the most
// common GenAI fields that should not be logged verbatim.
var DefaultDenyKeys = []string{
	"prompt",
	"completion",
	"messages",
	"input",
	"output",
	"api_key",
	"token",
	"authorization",
}

// RedactHook suppresses events whose buffer contains any field whose
// key matches a configured deny entry. Match is case-sensitive exact;
// callers wanting prefix or regex matching should write a custom hook
// against [bolt.EventHook] directly.
//
// RedactHook does NOT redact in place — bolt's EventHook contract is
// read-only. Suppression is the correct primitive for sensitive
// content; if a redacted variant should still ship, log it explicitly
// before the event reaches the hook.
type RedactHook struct {
	deny map[string]struct{}
}

// NewRedactHook returns a RedactHook that suppresses any event with
// at least one field key in keys. If keys is empty, [DefaultDenyKeys]
// is used.
func NewRedactHook(keys ...string) *RedactHook {
	if len(keys) == 0 {
		keys = DefaultDenyKeys
	}
	deny := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		deny[k] = struct{}{}
	}
	return &RedactHook{deny: deny}
}

// Run implements [bolt.EventHook].
func (h *RedactHook) Run(e *bolt.Event, _ string) bool {
	allow := true
	e.WalkFields(func(key, _ []byte) bool {
		if _, hit := h.deny[string(key)]; hit {
			allow = false
			return false // stop walking
		}
		return true
	})
	return allow
}

// AdaptiveSampler keeps 1 of every N low-importance events but always
// keeps events whose level matches AlwaysKeepLevel or whose buffer
// contains at least one field with a key matching AlwaysKeepPrefixes.
//
// Default behaviour ("always keep gen_ai.* and ERROR or higher") is a
// good fit for GenAI workloads where token-stream debug logs swamp
// the structured GenAI breadcrumbs that downstream tools care about.
type AdaptiveSampler struct {
	N                  uint32
	AlwaysKeepLevel    bolt.Level
	AlwaysKeepPrefixes []string

	counter uint32
}

// NewAdaptiveSampler returns an AdaptiveSampler keeping 1 of every n
// events whose level is below ERROR and whose fields don't match the
// `gen_ai.` prefix. Pass n == 0 or 1 for "keep everything".
func NewAdaptiveSampler(n uint32) *AdaptiveSampler {
	return &AdaptiveSampler{
		N:                  n,
		AlwaysKeepLevel:    bolt.LevelError,
		AlwaysKeepPrefixes: []string{"gen_ai."},
	}
}

// Run implements [bolt.EventHook].
func (s *AdaptiveSampler) Run(e *bolt.Event, _ string) bool {
	// Always-keep level wins immediately.
	if e.Level() >= s.AlwaysKeepLevel {
		return true
	}
	// Always-keep prefix scan. Walk fields once; if any key matches,
	// keep the event regardless of the sample counter.
	keep := false
	if len(s.AlwaysKeepPrefixes) > 0 {
		e.WalkFields(func(key, _ []byte) bool {
			k := string(key)
			for _, p := range s.AlwaysKeepPrefixes {
				if strings.HasPrefix(k, p) {
					keep = true
					return false
				}
			}
			return true
		})
		if keep {
			return true
		}
	}
	// Standard 1-in-N sampling.
	if s.N <= 1 {
		return true
	}
	c := atomic.AddUint32(&s.counter, 1)
	return c%s.N == 0
}
