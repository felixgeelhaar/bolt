// Console handler allocation benchmark
package bolt

import (
	"testing"
)

// discardWriter implements io.Writer but discards all writes
type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// BenchmarkConsoleHandlerAllocation verifies zero allocations in ConsoleHandler
func BenchmarkConsoleHandlerAllocation(b *testing.B) {
	handler := NewConsoleHandler(&discardWriter{})
	logger := New(handler)

	b.Run("Simple", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logger.Info().Str("key", "value").Msg("test message")
		}
	})

	b.Run("Complex", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logger.Info().
				Str("service", "api").
				Int("user_id", 12345).
				Bool("success", true).
				Float64("latency", 123.456).
				Msg("request completed")
		}
	})
}

// BenchmarkConsoleVsJSON compares performance
func BenchmarkConsoleVsJSON(b *testing.B) {
	discard := &discardWriter{}

	b.Run("Console", func(b *testing.B) {
		logger := New(NewConsoleHandler(discard))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logger.Info().Str("key", "value").Int("num", 42).Msg("test")
		}
	})

	b.Run("JSON", func(b *testing.B) {
		logger := New(NewJSONHandler(discard))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			logger.Info().Str("key", "value").Int("num", 42).Msg("test")
		}
	})
}
