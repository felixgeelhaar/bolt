package bolt

import (
	"os"
	"testing"
)

func TestDefaultLogger_FormatEnvVar(t *testing.T) {
	t.Run("json format", func(t *testing.T) {
		os.Setenv("BOLT_FORMAT", "json")
		initDefaultLogger()

		if _, ok := defaultLogger.handler.(*JSONHandler); !ok {
			t.Errorf("Expected JSONHandler, got %T", defaultLogger.handler)
		}
	})

	t.Run("console format", func(t *testing.T) {
		os.Setenv("BOLT_FORMAT", "console")
		initDefaultLogger()

		if _, ok := defaultLogger.handler.(*ConsoleHandler); !ok {
			t.Errorf("Expected ConsoleHandler, got %T", defaultLogger.handler)
		}
	})

	// Unset the env var to avoid affecting other tests.
	os.Unsetenv("BOLT_FORMAT")
	// Restore the original isTerminal function
	isTerminal = isatty
}

func TestDefaultLogger_Isatty(t *testing.T) {
	// Unset env var to ensure we test the isatty logic
	os.Unsetenv("BOLT_FORMAT")

	originalIsTerminal := isTerminal
	defer func() { isTerminal = originalIsTerminal }()

	t.Run("isatty true", func(t *testing.T) {
		// Mock isatty to return true
		isTerminal = func(*os.File) bool { return true }
		initDefaultLogger()

		if _, ok := defaultLogger.handler.(*ConsoleHandler); !ok {
			t.Errorf("Expected ConsoleHandler when isatty is true, got %T", defaultLogger.handler)
		}
	})

	t.Run("isatty false", func(t *testing.T) {
		// Mock isatty to return false
		isTerminal = func(*os.File) bool { return false }
		initDefaultLogger()

		if _, ok := defaultLogger.handler.(*JSONHandler); !ok {
			t.Errorf("Expected JSONHandler when isatty is false, got %T", defaultLogger.handler)
		}
	})
}

func TestDefaultLogger_LevelEnvVar(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected Level
	}{
		{name: "debug level", envValue: "debug", expected: DEBUG},
		{name: "info level", envValue: "info", expected: INFO},
		{name: "warn level", envValue: "warn", expected: WARN},
		{name: "error level", envValue: "error", expected: ERROR},
		{name: "fatal level", envValue: "fatal", expected: FATAL},
		{name: "trace level", envValue: "trace", expected: TRACE},
		{name: "unknown level defaults to info", envValue: "foo", expected: INFO},
		{name: "empty level defaults to info", envValue: "", expected: INFO},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("BOLT_LEVEL", tt.envValue)
			initDefaultLogger()

			if defaultLogger.level != tt.expected {
				t.Errorf("Expected level %s, got %s", tt.expected, defaultLogger.level)
			}
		})
	}
	// Unset the env var to avoid affecting other tests.
	os.Unsetenv("BOLT_LEVEL")
}
