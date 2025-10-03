// Float64 precision tests
package bolt

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

// TestFloat64Precision documents and validates the 6-decimal precision limitation
func TestFloat64Precision(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	tests := []struct {
		name     string
		value    float64
		contains string // What the output should contain
	}{
		{
			name:     "Simple decimal",
			value:    99.99,
			contains: "99.989999", // 6 decimal precision
		},
		{
			name:     "Pi",
			value:    3.14159265358979323846,
			contains: "3.141592", // Truncated to 6 decimals
		},
		{
			name:     "Zero",
			value:    0.0,
			contains: "0",
		},
		{
			name:     "Negative",
			value:    -123.456789,
			contains: "-123.456789", // 6 decimals
		},
		{
			name:     "Very large (scientific)",
			value:    1.23e100,
			contains: "1.23e+100", // Scientific notation
		},
		{
			name:     "Very small (scientific)",
			value:    1.23e-100,
			contains: "1.23e-100", // Scientific notation
		},
		{
			name:     "NaN",
			value:    math.NaN(),
			contains: "NaN",
		},
		{
			name:     "Positive Infinity",
			value:    math.Inf(1),
			contains: "\"+Inf\"",
		},
		{
			name:     "Negative Infinity",
			value:    math.Inf(-1),
			contains: "\"-Inf\"",
		},
		{
			name:     "Negative zero",
			value:    math.Copysign(0, -1),
			contains: "0", // Does not preserve negative zero sign
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger.Info().Float64("value", tt.value).Msg("test")

			output := buf.String()
			if !strings.Contains(output, tt.contains) {
				t.Errorf("Expected output to contain %q, got %q", tt.contains, output)
			}
		})
	}
}

// TestFloat64VsAnyPrecision demonstrates precision difference
func TestFloat64VsAnyPrecision(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	value := 99.99

	// Float64 (6 decimal precision, zero-allocation)
	buf.Reset()
	logger.Info().Float64("fast", value).Msg("test")
	float64Output := buf.String()

	// Any (full precision, allocates)
	buf.Reset()
	logger.Info().Any("precise", value).Msg("test")
	anyOutput := buf.String()

	t.Logf("Float64 output: %s", float64Output)
	t.Logf("Any output:     %s", anyOutput)

	// Float64 should have 6-decimal representation
	if !strings.Contains(float64Output, "99.989999") {
		t.Errorf("Float64 should output 6-decimal precision, got: %s", float64Output)
	}

	// Any should have exact representation
	if !strings.Contains(anyOutput, "99.99") {
		t.Errorf("Any should preserve exact value, got: %s", anyOutput)
	}
}

// TestFloat64ScientificNotation tests scientific notation formatting
func TestFloat64ScientificNotation(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	scientificTests := []struct {
		name  string
		value float64
		want  string
	}{
		{"Large positive", 1e20, "1e+20"},
		{"Large negative", -1e20, "-1e+20"},
		{"Small positive", 1e-10, "1e-10"},
		{"Small negative", -1e-10, "-1e-10"},
		{"Exact threshold upper", 1e15, "1e+15"},
		{"Exact threshold lower", 1e-6, "0.000001"}, // Just above threshold, fixed-point
		{"Below threshold", 1e-7, "1e-7"},           // Below threshold, scientific
	}

	for _, tt := range scientificTests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger.Info().Float64("value", tt.value).Msg("test")

			output := buf.String()
			if !strings.Contains(output, tt.want) {
				t.Errorf("Expected %s to contain %q, got %q", tt.name, tt.want, output)
			}
		})
	}
}

// BenchmarkFloat64Precision benchmarks Float64 vs Any for precision tradeoff
func BenchmarkFloat64Precision(b *testing.B) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))
	value := 3.14159265358979323846

	b.Run("Float64_6decimals_zero_alloc", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Float64("pi", value).Msg("test")
		}
	})

	b.Run("Any_full_precision_allocates", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			logger.Info().Any("pi", value).Msg("test")
		}
	})
}
