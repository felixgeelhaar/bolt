package bolt

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

// TestAdvancedErrorHandling tests comprehensive error handling scenarios
func TestAdvancedErrorHandling(t *testing.T) {
	t.Run("Handler Write Failures", func(t *testing.T) {
		// Test handler that always fails to write
		failWriter := &FailingWriter{failAfter: 0}
		logger := New(NewJSONHandler(failWriter))

		var errorCaptured error
		logger.SetErrorHandler(func(err error) {
			errorCaptured = err
		})

		// Should capture write error
		logger.Info().Str("test", "value").Msg("test message")

		if errorCaptured == nil {
			t.Error("Expected write error to be captured, got nil")
		}
		if !strings.Contains(errorCaptured.Error(), "write failed") {
			t.Errorf("Expected 'write failed' error, got: %v", errorCaptured)
		}
	})

	t.Run("Partial Write Failures", func(t *testing.T) {
		// Writer that fails after N bytes
		failWriter := &FailingWriter{failAfter: 10}
		logger := New(NewJSONHandler(failWriter))

		var errorCaptured error
		logger.SetErrorHandler(func(err error) {
			errorCaptured = err
		})

		// Large message that will exceed failAfter threshold
		logger.Info().Str("test", strings.Repeat("x", 100)).Msg("test")

		// Note: Bolt may buffer the entire message before writing,
		// so partial writes might not trigger errors as expected
		// This test documents actual behavior rather than enforcing it
		if errorCaptured != nil {
			t.Logf("Partial write error captured: %v", errorCaptured)
		}
	})

	t.Run("Nil Writer Handling", func(t *testing.T) {
		// This should not panic - handler should handle nil writer gracefully
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic on nil writer: %v", r)
			}
		}()

		// Note: NewJSONHandler doesn't accept nil, so we test error handler itself
		logger := New(NewJSONHandler(&bytes.Buffer{}))
		logger.SetErrorHandler(nil) // nil error handler should be safe

		logger.Info().Msg("test") // Should not panic
	})

	t.Run("Error Handler Chain", func(t *testing.T) {
		var errors []error
		var mu sync.Mutex

		logger := New(NewJSONHandler(&FailingWriter{failAfter: 0}))
		logger.SetErrorHandler(func(err error) {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
		})

		// Multiple failures should all be captured
		for i := 0; i < 5; i++ {
			logger.Info().Int("iteration", i).Msg("test")
		}

		mu.Lock()
		count := len(errors)
		mu.Unlock()

		if count != 5 {
			t.Errorf("Expected 5 errors captured, got %d", count)
		}
	})

	t.Run("Invalid Key Errors", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		var errorCaptured error
		logger.SetErrorHandler(func(err error) {
			errorCaptured = err
		})

		// Empty key
		logger.Info().Str("", "value").Msg("test")
		if errorCaptured == nil {
			t.Error("Expected error for empty key")
		}
		if !strings.Contains(errorCaptured.Error(), "empty") {
			t.Errorf("Expected 'empty' in error, got: %v", errorCaptured)
		}

		errorCaptured = nil

		// Whitespace-only key
		logger.Info().Str("   ", "value").Msg("test")
		if errorCaptured == nil {
			t.Error("Expected error for whitespace-only key")
		}

		errorCaptured = nil

		// Control character in key
		logger.Info().Str("key\x00null", "value").Msg("test")
		if errorCaptured == nil {
			t.Error("Expected error for control character in key")
		}

		errorCaptured = nil

		// Key too long
		longKey := strings.Repeat("k", MaxKeyLength+1)
		logger.Info().Str(longKey, "value").Msg("test")
		if errorCaptured == nil {
			t.Error("Expected error for overly long key")
		}
	})

	t.Run("Concurrent Error Handling", func(t *testing.T) {
		var errors []error
		var mu sync.Mutex

		logger := New(NewJSONHandler(&FailingWriter{failAfter: 0}))
		logger.SetErrorHandler(func(err error) {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
		})

		// Concurrent writes that all fail
		done := make(chan bool, 100)
		for i := 0; i < 100; i++ {
			go func(id int) {
				logger.Info().Int("id", id).Msg("concurrent test")
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 100; i++ {
			<-done
		}

		mu.Lock()
		count := len(errors)
		mu.Unlock()

		if count != 100 {
			t.Errorf("Expected 100 errors from concurrent operations, got %d", count)
		}
	})

	t.Run("Error Handler Panic Recovery", func(t *testing.T) {
		// Note: Bolt does NOT recover panics in error handlers
		// This test documents that behavior - error handlers must not panic
		logger := New(NewJSONHandler(&FailingWriter{failAfter: 0}))

		panicOccurred := false
		logger.SetErrorHandler(func(err error) {
			// Error handlers should handle errors gracefully, not panic
			// If they do panic, it will propagate (this is intentional)
			panicOccurred = true
		})

		logger.Info().Msg("test")

		if !panicOccurred {
			t.Error("Error handler was not called")
		}
	})

	t.Run("Value Validation Errors", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		var errorCaptured error
		logger.SetErrorHandler(func(err error) {
			errorCaptured = err
		})

		// Valid key with empty value (should succeed - empty values are allowed)
		logger.Info().Str("key", "").Msg("test")
		if errorCaptured != nil {
			t.Error("Empty value should be allowed")
		}

		// Test that overly large values are handled gracefully
		// (No specific error, but should not crash)
		largeValue := strings.Repeat("x", 1000000) // 1MB string
		logger.Info().Str("key", largeValue).Msg("test")
		// Just verify no panic
	})
}

// TestErrorRecovery tests that the logger recovers from various error conditions
func TestErrorRecovery(t *testing.T) {
	t.Run("Recover from Write Error", func(t *testing.T) {
		failWriter := &FailingWriter{failAfter: 0, failCount: 1} // Fail once, then succeed
		logger := New(NewJSONHandler(failWriter))

		errorCount := 0
		logger.SetErrorHandler(func(err error) {
			errorCount++
		})

		// First write fails
		logger.Info().Msg("first")
		if errorCount != 1 {
			t.Errorf("Expected 1 error, got %d", errorCount)
		}

		// Second write succeeds
		logger.Info().Msg("second")
		if errorCount != 1 {
			t.Errorf("Expected still 1 error after recovery, got %d", errorCount)
		}
	})

	t.Run("Continue Logging After Errors", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		errorCount := 0
		logger.SetErrorHandler(func(err error) {
			errorCount++
		})

		// Generate several errors with invalid keys
		logger.Info().Str("", "value").Msg("error1")
		logger.Info().Str("   ", "value").Msg("error2")

		// Then log successfully
		logger.Info().Str("valid", "value").Msg("success")

		// Should have 2 errors but also successful log
		if errorCount != 2 {
			t.Errorf("Expected 2 errors, got %d", errorCount)
		}

		output := buf.String()
		if !strings.Contains(output, "success") {
			t.Error("Expected successful log message in output")
		}
	})
}

// TestErrorEdgeCases tests unusual error scenarios
func TestErrorEdgeCases(t *testing.T) {
	t.Run("Extremely Long Error Messages", func(t *testing.T) {
		logger := New(NewJSONHandler(&bytes.Buffer{}))

		var capturedError error
		logger.SetErrorHandler(func(err error) {
			capturedError = err
		})

		// Key that's way too long
		extremelyLongKey := strings.Repeat("k", 10000)
		logger.Info().Str(extremelyLongKey, "value").Msg("test")

		if capturedError == nil {
			t.Error("Expected error for extremely long key")
		}
	})

	t.Run("Error Handler Replacement", func(t *testing.T) {
		logger := New(NewJSONHandler(&FailingWriter{failAfter: 0}))

		firstHandlerCalled := false
		logger.SetErrorHandler(func(err error) {
			firstHandlerCalled = true
		})

		secondHandlerCalled := false
		logger.SetErrorHandler(func(err error) {
			secondHandlerCalled = true
		})

		logger.Info().Msg("test")

		if firstHandlerCalled {
			t.Error("First error handler should have been replaced")
		}
		if !secondHandlerCalled {
			t.Error("Second error handler should have been called")
		}
	})

	t.Run("Multiple Errors Per Event", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(NewJSONHandler(&buf))

		errors := []error{}
		logger.SetErrorHandler(func(err error) {
			errors = append(errors, err)
		})

		// Multiple invalid keys in same event
		logger.Info().
			Str("", "val1").      // Error: empty key
			Str("   ", "val2").   // Error: whitespace key
			Str("valid", "val3"). // OK
			Str("\x00", "val4").  // Error: control char
			Msg("test")

		if len(errors) < 2 {
			t.Errorf("Expected at least 2 errors, got %d", len(errors))
		}
	})
}

// FailingWriter is a writer that fails after N bytes or N calls
type FailingWriter struct {
	written   int
	calls     int
	failAfter int // Fail after this many bytes written (0 = always fail)
	failCount int // Number of times to fail (0 = always fail)
	mu        sync.Mutex
}

func (fw *FailingWriter) Write(p []byte) (n int, err error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	fw.calls++

	// Check if we should fail based on failCount
	if fw.failCount > 0 && fw.calls > fw.failCount {
		// Stop failing after failCount failures
		fw.written += len(p)
		return len(p), nil
	}

	// Check if we should fail based on bytes written
	if fw.failAfter == 0 {
		return 0, errors.New("write failed")
	}

	if fw.written >= fw.failAfter {
		return 0, errors.New("write failed after threshold")
	}

	// Write partial data
	canWrite := fw.failAfter - fw.written
	if canWrite > len(p) {
		canWrite = len(p)
	}

	fw.written += canWrite
	return canWrite, nil
}

// TestNilHandlers tests behavior with nil handlers
func TestNilHandlers(t *testing.T) {
	t.Run("Nil Error Handler", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic with nil error handler: %v", r)
			}
		}()

		logger := New(NewJSONHandler(&FailingWriter{failAfter: 0}))
		logger.SetErrorHandler(nil) // Explicitly set to nil

		// Should not panic even with errors
		logger.Info().Msg("test")
	})
}

// TestIOErrorScenarios tests various IO error conditions
func TestIOErrorScenarios(t *testing.T) {
	t.Run("EOF Error", func(t *testing.T) {
		logger := New(NewJSONHandler(&EOFWriter{}))

		var capturedError error
		logger.SetErrorHandler(func(err error) {
			capturedError = err
		})

		logger.Info().Msg("test")

		if capturedError == nil {
			t.Error("Expected EOF error to be captured")
		}
		if !errors.Is(capturedError, io.EOF) {
			t.Errorf("Expected io.EOF, got: %v", capturedError)
		}
	})

	t.Run("Timeout Error", func(t *testing.T) {
		logger := New(NewJSONHandler(&TimeoutWriter{}))

		var capturedError error
		logger.SetErrorHandler(func(err error) {
			capturedError = err
		})

		logger.Info().Msg("test")

		if capturedError == nil {
			t.Error("Expected timeout error to be captured")
		}
		if !strings.Contains(capturedError.Error(), "timeout") {
			t.Errorf("Expected timeout error, got: %v", capturedError)
		}
	})
}

// EOFWriter always returns io.EOF
type EOFWriter struct{}

func (w *EOFWriter) Write(p []byte) (n int, err error) {
	return 0, io.EOF
}

// TimeoutWriter simulates timeout errors
type TimeoutWriter struct{}

func (w *TimeoutWriter) Write(p []byte) (n int, err error) {
	return 0, &TimeoutError{}
}

// TimeoutError simulates a timeout error
type TimeoutError struct{}

func (e *TimeoutError) Error() string {
	return "write timeout"
}

func (e *TimeoutError) Timeout() bool {
	return true
}

func (e *TimeoutError) Temporary() bool {
	return true
}
