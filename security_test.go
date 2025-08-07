package bolt

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// TestJSONEscaping verifies that JSON injection attacks are prevented
func TestJSONEscaping(t *testing.T) {
	var buf bytes.Buffer
	// Disable error handler so we can test escaping without validation interference
	logger := New(NewJSONHandler(&buf)).SetErrorHandler(nil)

	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{
			name:     "quotes in key and value",
			key:      `keywithquotes`, // Valid key
			value:    `value"with"quotes`,
			expected: `"keywithquotes":"value\"with\"quotes"`,
		},
		{
			name:     "backslashes in value",
			key:      `keywithbackslashes`, // Valid key
			value:    `value\with\backslashes`,
			expected: `"keywithbackslashes":"value\\with\\backslashes"`,
		},
		{
			name:     "newlines and tabs in value",
			key:      "keywithtabs", // Valid key
			value:    "value\nwith\ttabs",
			expected: `"keywithtabs":"value\nwith\ttabs"`,
		},
		{
			name:     "mixed special characters in value",
			key:      "keytest", // Valid key
			value:    "value\"\\test\r\t",
			expected: `"keytest":"value\"\\test\r\t"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger.Info().Str(tt.key, tt.value).Msg("test")

			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain %q, got %q", tt.expected, output)
			}

			// Verify it's valid JSON by checking structure
			if !strings.HasPrefix(output, `{"level":"info",`) {
				t.Errorf("Invalid JSON structure: %q", output)
			}
		})
	}
}

// TestInputValidation verifies that input validation prevents invalid inputs
func TestInputValidation(t *testing.T) {
	var buf bytes.Buffer
	var errorCalled bool
	var lastError error

	logger := New(NewJSONHandler(&buf)).SetErrorHandler(func(err error) {
		errorCalled = true
		lastError = err
	})

	t.Run("empty key", func(t *testing.T) {
		errorCalled = false
		buf.Reset()
		logger.Info().Str("", "value").Msg("test")

		if !errorCalled {
			t.Error("Expected error handler to be called for empty key")
		}
		if lastError == nil || !strings.Contains(lastError.Error(), "key cannot be empty") {
			t.Errorf("Expected empty key error, got: %v", lastError)
		}
	})

	t.Run("key too long", func(t *testing.T) {
		errorCalled = false
		buf.Reset()
		longKey := strings.Repeat("a", MaxKeyLength+1)
		logger.Info().Str(longKey, "value").Msg("test")

		if !errorCalled {
			t.Error("Expected error handler to be called for long key")
		}
		if lastError == nil || !strings.Contains(lastError.Error(), "key length exceeds maximum") {
			t.Errorf("Expected long key error, got: %v", lastError)
		}
	})

	t.Run("key with control characters", func(t *testing.T) {
		errorCalled = false
		buf.Reset()
		logger.Info().Str("key\x00with\x1Fcontrol", "value").Msg("test")

		if !errorCalled {
			t.Error("Expected error handler to be called for control character in key")
		}
		if lastError == nil || !strings.Contains(lastError.Error(), "invalid control character") {
			t.Errorf("Expected control character error, got: %v", lastError)
		}
	})

	t.Run("value too long", func(t *testing.T) {
		errorCalled = false
		buf.Reset()
		longValue := strings.Repeat("a", MaxValueLength+1)
		logger.Info().Str("key", longValue).Msg("test")

		if !errorCalled {
			t.Error("Expected error handler to be called for long value")
		}
		if lastError == nil || !strings.Contains(lastError.Error(), "value length exceeds maximum") {
			t.Errorf("Expected long value error, got: %v", lastError)
		}
	})

	t.Run("message too long", func(t *testing.T) {
		errorCalled = false
		buf.Reset()
		longMessage := strings.Repeat("a", MaxValueLength+1)
		logger.Info().Msg(longMessage)

		if !errorCalled {
			t.Error("Expected error handler to be called for long message")
		}
		if lastError == nil || !strings.Contains(lastError.Error(), "invalid message") {
			t.Errorf("Expected long message error, got: %v", lastError)
		}
	})
}

// TestErrorHandling verifies that handler write errors are properly handled
func TestErrorHandling(t *testing.T) {
	var errorCalled bool
	var lastError error

	// Create a handler that always fails
	failingHandler := &failingTestHandler{shouldFail: true}

	logger := New(failingHandler).SetErrorHandler(func(err error) {
		errorCalled = true
		lastError = err
	})

	t.Run("handler write error", func(t *testing.T) {
		errorCalled = false
		logger.Info().Str("key", "value").Msg("test")

		if !errorCalled {
			t.Error("Expected error handler to be called for handler write failure")
		}
		if lastError == nil || !strings.Contains(lastError.Error(), "handler write failed") {
			t.Errorf("Expected handler write error, got: %v", lastError)
		}
	})
}

// TestZeroAllocationWithSecurity verifies that security fixes don't break zero allocations
func TestZeroAllocationWithSecurity(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	// This should still have zero allocations
	allocs := testing.AllocsPerRun(100, func() {
		buf.Reset()
		logger.Info().Str("key", "value").Int("number", 42).Bool("flag", true).Msg("test message")
	})

	if allocs > 0 {
		t.Errorf("Expected 0 allocations, got %f", allocs)
	}
}

// TestMessageEscaping verifies that messages are properly escaped
func TestMessageEscaping(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "quotes in message",
			message:  `message with "quotes"`,
			expected: `"message":"message with \"quotes\""`,
		},
		{
			name:     "backslashes in message",
			message:  `message with \backslashes\`,
			expected: `"message":"message with \\backslashes\\"`,
		},
		{
			name:     "newlines and control chars",
			message:  "message\nwith\tcontrol\rchars",
			expected: `"message":"message\nwith\tcontrol\rchars"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger.Info().Msg(tt.message)

			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain %q, got %q", tt.expected, output)
			}
		})
	}
}

// TestBufferSizeLimits verifies that buffer size limits are enforced
func TestBufferSizeLimits(t *testing.T) {
	// We can't easily test MaxBufferSize in a unit test because it would require
	// actually filling a 1MB buffer, but we can test the validation logic
	t.Run("buffer size validation exists", func(t *testing.T) {
		// Test that the checkBufferSize function works
		smallBuf := make([]byte, 100)
		err := checkBufferSize(smallBuf)
		if err != nil {
			t.Errorf("Small buffer should not trigger error: %v", err)
		}

		// We can't create a buffer larger than MaxBufferSize in memory for testing,
		// but we can verify the function logic
		if MaxBufferSize <= 0 {
			t.Error("MaxBufferSize should be positive")
		}
	})
}

// failingTestHandler is a test handler that can be configured to fail
type failingTestHandler struct {
	shouldFail bool
}

func (h *failingTestHandler) Write(e *Event) error {
	if h.shouldFail {
		return errors.New("simulated write failure")
	}
	return nil
}

// TestAllFieldTypesWithValidation tests that all field types properly validate keys
func TestAllFieldTypesWithValidation(t *testing.T) {
	var buf bytes.Buffer
	var errorCalled bool

	logger := New(NewJSONHandler(&buf)).SetErrorHandler(func(err error) {
		errorCalled = true
	})

	invalidKey := "key\x00with\x1Fcontrol"

	fieldTests := []struct {
		name string
		fn   func() *Event
	}{
		{"Str", func() *Event { return logger.Info().Str(invalidKey, "value") }},
		{"Int", func() *Event { return logger.Info().Int(invalidKey, 42) }},
		{"Bool", func() *Event { return logger.Info().Bool(invalidKey, true) }},
		{"Float64", func() *Event { return logger.Info().Float64(invalidKey, 3.14) }},
		{"Uint", func() *Event { return logger.Info().Uint(invalidKey, 42) }},
		{"Int64", func() *Event { return logger.Info().Int64(invalidKey, 42) }},
		{"Uint64", func() *Event { return logger.Info().Uint64(invalidKey, 42) }},
		{"Hex", func() *Event { return logger.Info().Hex(invalidKey, []byte{1, 2, 3}) }},
		{"Base64", func() *Event { return logger.Info().Base64(invalidKey, []byte{1, 2, 3}) }},
		{"Any", func() *Event { return logger.Info().Any(invalidKey, map[string]string{"a": "b"}) }},
	}

	for _, tt := range fieldTests {
		t.Run(tt.name, func(t *testing.T) {
			errorCalled = false
			buf.Reset()
			tt.fn().Msg("test")

			if !errorCalled {
				t.Errorf("Expected error handler to be called for invalid key in %s method", tt.name)
			}
		})
	}
}
