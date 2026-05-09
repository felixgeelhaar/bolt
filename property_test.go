// Property-based tests using pgregory.net/rapid.
//
// The expert quality review flagged that the existing string-contains
// assertions and the 120s/week fuzz schedule miss whole classes of
// JSON-encoding bugs. This file adds property tests that the encoder
// must satisfy for ANY input — not just hand-picked strings.
//
// All properties here are roundtrip / structural invariants. They run
// fast (a few seconds total) so they sit in the regular test suite,
// not behind a build tag.
package bolt

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// TestRapid_StrFieldRoundtrip — for any (key, value) pair, the encoded
// log line must be valid JSON whose decoded fields equal the input.
// Catches: bad JSON escaping, lost code points, control-char handling,
// and the string-contains ambiguity that handcrafted assertions hide.
func TestRapid_StrFieldRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.StringMatching(`[a-zA-Z_][a-zA-Z0-9_]{0,30}`).Draw(t, "key")
		value := rapid.String().Draw(t, "value")

		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))
		logger.Info().Str(key, value).Msg("test")

		var got map[string]any
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("invalid JSON %q: %v (key=%q value=%q)", buf.String(), err, key, value)
		}
		if got[key] != value {
			t.Errorf("roundtrip mismatch: key=%q got=%q want=%q", key, got[key], value)
		}
		if got["message"] != "test" {
			t.Errorf("message lost: got %v", got["message"])
		}
		if got["level"] != "info" {
			t.Errorf("level lost: got %v", got["level"])
		}
	})
}

// TestRapid_MessageRoundtrip — Msg(s) must encode the message such that
// decoding the log line produces s exactly. The previous fuzz inputs
// were 3-arg primitives; this property exercises arbitrary UTF-8 input
// including surrogates, control chars, and embedded JSON-special chars.
func TestRapid_MessageRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		msg := rapid.String().Draw(t, "msg")

		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))
		logger.Info().Msg(msg)

		var got map[string]any
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("invalid JSON %q for msg=%q: %v", buf.String(), msg, err)
		}
		if got["message"] != msg {
			t.Errorf("message roundtrip: got=%q want=%q", got["message"], msg)
		}
	})
}

// TestRapid_MultipleStrFieldsRoundtrip — chains of Str() calls must
// produce a single valid JSON object whose decoded fields match.
// Catches missing/extra commas in encoder, key-value pair drift.
func TestRapid_MultipleStrFieldsRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Use distinct keys so the property is unambiguous; otherwise
		// later fields would overwrite earlier ones in the map.
		nFields := rapid.IntRange(1, 8).Draw(t, "n")
		keys := make([]string, nFields)
		values := make([]string, nFields)
		seen := map[string]bool{}
		for i := 0; i < nFields; i++ {
			for {
				k := rapid.StringMatching(`[a-zA-Z_][a-zA-Z0-9_]{0,15}`).Draw(t, "key")
				if !seen[k] {
					seen[k] = true
					keys[i] = k
					break
				}
			}
			values[i] = rapid.String().Draw(t, "value")
		}

		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))
		ev := logger.Info()
		for i := range keys {
			ev = ev.Str(keys[i], values[i])
		}
		ev.Msg("test")

		var got map[string]any
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("invalid JSON %q: %v", buf.String(), err)
		}
		for i := range keys {
			if got[keys[i]] != values[i] {
				t.Errorf("field[%d]: key=%q got=%q want=%q", i, keys[i], got[keys[i]], values[i])
			}
		}
	})
}

// TestRapid_IntFieldRoundtrip — any int produces a JSON number whose
// decoded value matches the input.
func TestRapid_IntFieldRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.StringMatching(`[a-zA-Z_][a-zA-Z0-9_]{0,30}`).Draw(t, "key")
		value := rapid.Int().Draw(t, "value")

		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))
		logger.Info().Int(key, value).Msg("test")

		var got map[string]any
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("invalid JSON %q: %v (key=%q value=%d)", buf.String(), err, key, value)
		}
		// json.Unmarshal decodes JSON numbers into float64 unless
		// json.Number is requested.
		if f, ok := got[key].(float64); !ok {
			t.Errorf("Int field decoded as %T, want float64", got[key])
		} else if int(f) != value {
			t.Errorf("Int roundtrip: got=%v want=%d", f, value)
		}
	})
}

// TestRapid_OutputIsAlwaysValidJSON — the strongest invariant:
// regardless of inputs, NewJSONHandler must always produce a valid JSON
// object terminated by exactly one newline. Failure here indicates an
// encoder bug that could break log shippers, parsers, or downstream
// systems doing line-by-line ingest.
func TestRapid_OutputIsAlwaysValidJSON(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Mix of field types and values.
		nFields := rapid.IntRange(0, 6).Draw(t, "n")
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))
		ev := logger.Info()
		seen := map[string]bool{}
		for i := 0; i < nFields; i++ {
			var k string
			for {
				k = rapid.StringMatching(`[a-zA-Z_][a-zA-Z0-9_]{0,15}`).Draw(t, "key")
				if !seen[k] {
					seen[k] = true
					break
				}
			}
			switch rapid.IntRange(0, 3).Draw(t, "kind") {
			case 0:
				ev = ev.Str(k, rapid.String().Draw(t, "v"))
			case 1:
				ev = ev.Int(k, rapid.Int().Draw(t, "v"))
			case 2:
				ev = ev.Bool(k, rapid.Bool().Draw(t, "v"))
			case 3:
				// Float64: NaN/Inf are encoded as JSON strings, so the
				// output is still valid JSON.
				ev = ev.Float64(k, rapid.Float64().Draw(t, "v"))
			}
		}
		ev.Msg(rapid.String().Draw(t, "msg"))

		out := buf.String()
		if !strings.HasSuffix(out, "\n") {
			t.Errorf("output not newline-terminated: %q", out)
		}
		var got map[string]any
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("invalid JSON %q: %v", out, err)
		}
		if got["level"] != "info" {
			t.Errorf("level missing or wrong: %v", got["level"])
		}
	})
}
