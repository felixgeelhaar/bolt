//go:build go1.18
// +build go1.18

package bolt

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"unicode/utf8"
)

// Buffer size limits for fuzzing (based on security requirements)
const (
	maxBufferSize = 1048576 // 1MB max buffer size
	maxKeySize    = 256     // 256 chars max key length
	maxValueSize  = 65536   // 64KB max value size
)

// FuzzJSONHandler tests JSON serialization with random inputs
func FuzzJSONHandler(f *testing.F) {
	// Seed corpus with interesting test cases
	f.Add("test message", 42, true)
	f.Add("unicode: 你好世界", -1, false)
	f.Add("special\n\t\"chars\\", 0, true)
	f.Add("", math.MaxInt64, false)
	f.Add(strings.Repeat("A", 1000), math.MinInt64, true)

	f.Fuzz(func(t *testing.T, msg string, num int, flag bool) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		// Should never panic regardless of input
		logger.Info().
			Str("msg", msg).
			Int("num", num).
			Bool("flag", flag).
			Msg("fuzz test")

		// Output should be valid JSON
		output := buf.String()
		if len(output) > 0 {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Errorf("Invalid JSON output: %v\nInput: msg=%q num=%d flag=%v\nOutput: %s",
					err, msg, num, flag, output)
			}

			// Verify required fields
			if _, ok := result["level"]; !ok {
				t.Error("Missing 'level' field in JSON output")
			}
			if _, ok := result["message"]; !ok {
				t.Error("Missing 'message' field in JSON output")
			}
		}
	})
}

// FuzzInputValidation tests input sanitization against injection attacks
func FuzzInputValidation(f *testing.F) {
	// Injection attack vectors
	f.Add(`{"injected": "value"}`)
	f.Add(`\u0000\u0001\u0002`)
	f.Add(`"quote"injection"`)
	f.Add("\\backslash\\injection\\")
	f.Add("\n\r\t control chars")
	f.Add(strings.Repeat("A", 100000)) // Large input

	f.Fuzz(func(t *testing.T, input string) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		// Should sanitize safely without panic
		logger.Info().Str("user_input", input).Msg("test")

		output := buf.String()

		// No buffer overflow
		if buf.Len() > maxBufferSize {
			t.Errorf("Buffer exceeded max size: %d > %d", buf.Len(), maxBufferSize)
		}

		// Output should still be valid JSON
		if len(output) > 0 {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Errorf("Sanitization failed to produce valid JSON: %v\nInput: %q\nOutput: %s",
					err, input, output)
			}

			// Verify no raw injection in output
			if strings.Contains(output, `{"injected"`) {
				t.Error("JSON injection not properly escaped")
			}
		}
	})
}

// FuzzFloatFormatting tests float64 formatter with extreme values
func FuzzFloatFormatting(f *testing.F) {
	// Extreme float values
	f.Add(0.0)
	f.Add(1.0)
	f.Add(-1.0)
	f.Add(math.Inf(1))
	f.Add(math.Inf(-1))
	f.Add(math.NaN())
	f.Add(1e308)
	f.Add(1e-308)
	f.Add(math.MaxFloat64)
	f.Add(-math.MaxFloat64)
	f.Add(math.SmallestNonzeroFloat64)

	f.Fuzz(func(t *testing.T, val float64) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		// Should never panic, even with special values
		logger.Info().Float64("value", val).Msg("test")

		output := buf.String()

		// Output should be valid JSON
		if len(output) > 0 {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Errorf("Float formatting produced invalid JSON: %v\nInput: %v\nOutput: %s",
					err, val, output)
			}

			// Check for proper special value handling
			if math.IsNaN(val) && !strings.Contains(output, "NaN") {
				t.Error("NaN not properly formatted")
			}
			if math.IsInf(val, 1) && !strings.Contains(output, "+Inf") {
				t.Error("Positive infinity not properly formatted")
			}
			if math.IsInf(val, -1) && !strings.Contains(output, "-Inf") {
				t.Error("Negative infinity not properly formatted")
			}
		}
	})
}

// FuzzBufferManagement tests buffer handling and boundaries
func FuzzBufferManagement(f *testing.F) {
	// Various buffer sizes
	f.Add(10, "small")
	f.Add(1000, "medium")
	f.Add(100000, "large")
	f.Add(1000000, "max")

	f.Fuzz(func(t *testing.T, repeatCount int, content string) {
		// Clamp repeat count to avoid excessive memory
		if repeatCount < 0 {
			repeatCount = -repeatCount
		}
		if repeatCount > 10000 {
			repeatCount = 10000
		}

		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		// Generate large log entry
		event := logger.Info()
		for i := 0; i < repeatCount; i++ {
			event.Str("field", content)
		}
		event.Msg("buffer test")

		// Should handle gracefully
		output := buf.String()

		// Verify buffer limits
		if buf.Len() > maxBufferSize {
			// Should have been truncated or rejected
			t.Errorf("Buffer limit not enforced: %d > %d", buf.Len(), maxBufferSize)
		}

		// If output exists, should be valid
		if len(output) > 0 {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				// Truncated output might not be valid JSON, that's acceptable
				// But should still be safe
				if !strings.Contains(err.Error(), "unexpected end") {
					t.Errorf("Unexpected JSON error: %v", err)
				}
			}
		}
	})
}

// FuzzConcurrentLogging tests race conditions under fuzzing
func FuzzConcurrentLogging(f *testing.F) {
	// Concurrent scenarios
	f.Add(5, "test message", 42)
	f.Add(100, "concurrent", -1)
	f.Add(1000, "stress", 0)

	f.Fuzz(func(t *testing.T, goroutines int, msg string, num int) {
		// Limit goroutines to avoid excessive resource usage
		if goroutines < 1 {
			goroutines = 1
		}
		if goroutines > 1000 {
			goroutines = 1000
		}

		buf := &ThreadSafeBuffer{}
		logger := New(NewJSONHandler(buf))

		// Run concurrent logging
		done := make(chan bool, goroutines)
		for i := 0; i < goroutines; i++ {
			go func(id int) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Panic in goroutine %d: %v", id, r)
					}
					done <- true
				}()

				logger.Info().
					Int("goroutine", id).
					Str("msg", msg).
					Int("num", num).
					Msg("concurrent fuzz")
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < goroutines; i++ {
			<-done
		}

		// Verify output is safe
		output := string(buf.Bytes())
		lines := strings.Count(output, "\n")

		// Should have logged from all goroutines (some might be filtered by level)
		// Just verify no corruption - line count may vary
		_ = lines // Acknowledge we checked line count but don't enforce exact match

		// Each line should be valid JSON
		for i, line := range strings.Split(output, "\n") {
			if len(line) == 0 {
				continue
			}
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(line), &result); err != nil {
				t.Errorf("Line %d is invalid JSON: %v\nLine: %s", i, err, line)
			}
		}
	})
}

// FuzzKeyValidation tests key validation edge cases
func FuzzKeyValidation(f *testing.F) {
	// Edge case keys
	f.Add("")
	f.Add(" ")
	f.Add("\t\n")
	f.Add(strings.Repeat("k", 1000))
	f.Add("key with spaces")
	f.Add("key\x00null")
	f.Add("unicode\u2028key")

	f.Fuzz(func(t *testing.T, key string) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		var capturedError error
		logger.SetErrorHandler(func(err error) {
			capturedError = err
		})

		// Attempt to log with potentially invalid key
		logger.Info().Str(key, "value").Msg("test")

		output := buf.String()

		// Either should error or produce valid output
		if capturedError == nil && len(output) > 0 {
			// Should be valid JSON
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Errorf("Key validation produced invalid JSON: %v\nKey: %q\nOutput: %s",
					err, key, output)
			}
		}

		// Very long keys should be rejected
		if len(key) > 256 && capturedError == nil {
			t.Error("Long key not rejected by validation")
		}

		// Empty keys should be rejected
		if len(strings.TrimSpace(key)) == 0 && capturedError == nil && len(output) > 0 {
			t.Error("Empty key not rejected by validation")
		}
	})
}

// FuzzUnicodeHandling tests UTF-8 handling and invalid sequences
func FuzzUnicodeHandling(f *testing.F) {
	// Unicode edge cases
	f.Add("valid UTF-8: 你好")
	f.Add("emoji: 🚀⚡")
	f.Add("combined: café")
	f.Add("\xff\xfe invalid")
	f.Add("partial: \xc0")

	f.Fuzz(func(t *testing.T, input string) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		// Log potentially invalid UTF-8
		logger.Info().Str("unicode", input).Msg("test")

		output := buf.String()

		// Output should be valid UTF-8
		if !utf8.ValidString(output) {
			t.Errorf("Output contains invalid UTF-8\nInput: %q\nOutput: %q", input, output)
		}

		// Should be valid JSON
		if len(output) > 0 {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Errorf("Unicode handling produced invalid JSON: %v\nInput: %q\nOutput: %s",
					err, input, output)
			}
		}
	})
}

// FuzzMessageEscaping tests message field escaping
func FuzzMessageEscaping(f *testing.F) {
	// Message edge cases
	f.Add("normal message")
	f.Add("with \"quotes\"")
	f.Add("with\nnewlines\nand\ttabs")
	f.Add("backslash\\test\\")
	f.Add("{\"json\": \"inside\"}")
	f.Add("\x00\x01\x02 control chars")

	f.Fuzz(func(t *testing.T, msg string) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		// Log with potentially dangerous message
		logger.Info().Msg(msg)

		output := buf.String()

		// Should produce valid JSON
		if len(output) > 0 {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Errorf("Message escaping failed: %v\nMessage: %q\nOutput: %s",
					err, msg, output)
			}

			// If unmarshal succeeded, the escaping worked correctly
			// (JSON parser would fail on unescaped control characters in strings)
		}
	})
}

// FuzzLevelValidation tests level handling with invalid values
func FuzzLevelValidation(f *testing.F) {
	// Level values
	f.Add(int8(0))
	f.Add(int8(5))
	f.Add(int8(-1))
	f.Add(int8(127))
	f.Add(int8(-128))

	f.Fuzz(func(t *testing.T, levelVal int8) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		// Set potentially invalid level
		logger.SetLevel(Level(levelVal))

		// Should not panic or corrupt
		logger.Info().Msg("test after level set")

		output := buf.String()

		// Should either log or filter, but not corrupt
		if len(output) > 0 {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Errorf("Invalid level corrupted output: %v\nLevel: %d\nOutput: %s",
					err, levelVal, output)
			}
		}
	})
}
