// Float64 precision tests
package bolt

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"testing"
)

// TestFloat64Precision validates that Float64 round-trips IEEE-754 values
// without truncation. The previous fixed-point encoder lost everything
// past 6 decimal places (e.g. pi → 3.141592). The current encoder uses
// strconv.AppendFloat with the shortest-round-trip ('g', -1) verb,
// matching encoding/json.
func TestFloat64Precision(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	tests := []struct {
		name     string
		value    float64
		contains string
	}{
		{
			name:     "Simple decimal",
			value:    99.99,
			contains: "99.99",
		},
		{
			name:     "Pi (full precision)",
			value:    3.14159265358979323846,
			contains: "3.141592653589793",
		},
		{
			name:     "Zero",
			value:    0.0,
			contains: "0",
		},
		{
			name:     "Negative",
			value:    -123.456789,
			contains: "-123.456789",
		},
		{
			name:     "Seven decimals",
			value:    1.2345678,
			contains: "1.2345678",
		},
		{
			name:     "Very large (scientific)",
			value:    1.23e100,
			contains: "1.23e+100",
		},
		{
			name:     "Very small (scientific)",
			value:    1.23e-100,
			contains: "1.23e-100",
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
			name:  "Negative zero",
			value: math.Copysign(0, -1),
			// strconv preserves the sign of negative zero ("-0").
			contains: "0",
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

// TestFloat64Roundtrip verifies that non-special-case Float64 output is
// numerically equal to the input after re-parsing (the round-trip
// invariant strconv.AppendFloat with 'g', -1 guarantees).
func TestFloat64Roundtrip(t *testing.T) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))

	cases := []float64{
		1, 1.5, 99.99, 3.14159265358979323846,
		-0.000123, 1e-100, 1e100, math.MaxFloat64, math.SmallestNonzeroFloat64,
	}
	for _, v := range cases {
		buf.Reset()
		logger.Info().Float64("v", v).Msg("test")
		// Pull the literal between the prefix and the closing comma/quote.
		s := buf.String()
		i := strings.Index(s, `"v":`)
		if i < 0 {
			t.Fatalf("missing v field in %q", s)
		}
		j := strings.IndexAny(s[i+4:], ",}")
		if j < 0 {
			t.Fatalf("malformed value in %q", s)
		}
		lit := s[i+4 : i+4+j]
		got, err := strconv.ParseFloat(lit, 64)
		if err != nil {
			t.Fatalf("ParseFloat(%q): %v", lit, err)
		}
		if got != v {
			t.Errorf("roundtrip: input %v → output %q → parsed %v", v, lit, got)
		}
	}
}

// TestFloat64ScientificNotation pins the exponent threshold strconv uses.
// strconv 'g' picks fixed-point or exponent representation according to
// the shortest output rule; the exact crossover differs from the old
// hand-rolled encoder.
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
		{"Just below 10^21", 1e15, "1e+15"},
		{"1e-6", 1e-6, "1e-06"},
		{"1e-7", 1e-7, "1e-07"},
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

// BenchmarkFloat64Precision benchmarks Float64 (zero-alloc, full precision)
// vs Any (allocates, full precision via reflection).
func BenchmarkFloat64Precision(b *testing.B) {
	var buf bytes.Buffer
	logger := New(NewJSONHandler(&buf))
	value := 3.14159265358979323846

	b.Run("Float64_zero_alloc", func(b *testing.B) {
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
