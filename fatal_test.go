package bolt

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// init disables process termination for the test binary as a whole. Individual
// tests that exercise exit semantics restore exitFunc explicitly. Without this
// shim every test that emits a Fatal record (existing TestEndToEndLevelFiltering,
// race tests, etc.) would terminate the test process.
func init() {
	exitFunc = func(int) {}
}

// TestFatal_EmitsRecordBeforeExit asserts the FATAL record reaches the handler
// before exitFunc is invoked.
func TestFatal_EmitsRecordBeforeExit(t *testing.T) {
	var (
		buf      bytes.Buffer
		exitCode int
		called   bool
	)
	prev := exitFunc
	exitFunc = func(code int) {
		called = true
		exitCode = code
	}
	t.Cleanup(func() { exitFunc = prev })

	logger := New(NewJSONHandler(&buf))
	logger.Fatal().Str("reason", "boom").Msg("fatal message")

	if !called {
		t.Fatal("exitFunc was not invoked for FATAL event")
	}
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	out := buf.String()
	if !strings.Contains(out, `"message":"fatal message"`) {
		t.Errorf("FATAL record missing from output: %q", out)
	}
	if !strings.Contains(out, `"level":"fatal"`) {
		t.Errorf("FATAL level missing from output: %q", out)
	}
}

// TestFatal_TerminatesProcess verifies real os.Exit(1) semantics by spawning
// a subprocess that re-enables exitFunc and emits a Fatal event.
func TestFatal_TerminatesProcess(t *testing.T) {
	if os.Getenv("BOLT_FATAL_SUBPROCESS") == "1" {
		// Restore real exit and trigger Fatal.
		exitFunc = os.Exit
		logger := New(NewJSONHandler(os.Stdout))
		logger.Fatal().Msg("subprocess fatal")
		// Should be unreachable.
		os.Exit(0)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestFatal_TerminatesProcess", "-test.v")
	cmd.Env = append(os.Environ(), "BOLT_FATAL_SUBPROCESS=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("subprocess returned exit 0; expected exit 1. output:\n%s", out)
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("exit code = %d, want 1. output:\n%s", exitErr.ExitCode(), out)
	}
	if !strings.Contains(string(out), `"message":"subprocess fatal"`) {
		t.Errorf("subprocess output missing fatal record:\n%s", out)
	}
}
