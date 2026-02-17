package bolt_test

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"

	"github.com/felixgeelhaar/bolt"
)

func ExampleNew() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().Str("service", "api").Msg("started")
}

func ExampleNew_console() {
	logger := bolt.New(bolt.NewConsoleHandler(os.Stdout))
	logger.Info().Str("env", "development").Msg("ready")
}

func ExampleLogger_Info() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Str("method", "GET").
		Int("status", 200).
		Msg("request handled")
}

func ExampleLogger_Error() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Error().
		Str("component", "database").
		Msg("connection failed")
}

func ExampleLogger_With() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// Create a child logger with persistent context fields
	reqLogger := logger.With().
		Str("request_id", "abc-123").
		Str("user_id", "user-456").
		Logger()

	// All subsequent logs include request_id and user_id
	reqLogger.Info().Msg("processing request")
	reqLogger.Info().Int("items", 42).Msg("query complete")
}

func ExampleLogger_SetLevel() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// Only warn and above will be logged
	logger.SetLevel(bolt.WARN)

	logger.Info().Msg("this will be suppressed")
	logger.Warn().Msg("this will appear")
}

func ExampleNewSlogHandler() {
	// Use Bolt as a backend for the standard slog package
	handler := bolt.NewSlogHandler(os.Stdout, nil)
	logger := slog.New(handler)

	logger.Info("request handled",
		"method", "GET",
		"status", 200,
		"path", "/api/users",
	)
}

func ExampleNewSlogHandler_withLevel() {
	handler := bolt.NewSlogHandler(os.Stdout, &bolt.SlogHandlerOptions{
		Level: slog.LevelWarn,
	})
	logger := slog.New(handler)

	logger.Info("filtered out") // suppressed
	logger.Warn("visible")      // appears
}

func ExampleEvent_Str() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Str("user", "alice").
		Str("action", "login").
		Msg("user authenticated")
}

func ExampleEvent_Int() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Int("port", 8080).
		Int("workers", 4).
		Msg("server configured")
}

func ExampleEvent_Float64() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Float64("latency_ms", 12.345).
		Float64("cpu_percent", 42.7).
		Msg("metrics collected")
}

func ExampleEvent_Bool() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().
		Bool("authenticated", true).
		Bool("admin", false).
		Msg("access check")
}

func ExampleEvent_Err() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	err := os.ErrNotExist
	logger.Error().
		Err(err).
		Str("path", "/tmp/missing.txt").
		Msg("file not found")
}

// --- Feature 1: Uint8/Uint16/Uint32 ---

func ExampleEvent_Uint8() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().Uint8("ttl", 64).Msg("packet info")
	// Output: {"level":"info","ttl":64,"message":"packet info"}
}

func ExampleEvent_Uint16() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().Uint16("port", 8080).Msg("listening")
	// Output: {"level":"info","port":8080,"message":"listening"}
}

func ExampleEvent_Uint32() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().Uint32("sequence", 4294967295).Msg("max seq")
	// Output: {"level":"info","sequence":4294967295,"message":"max seq"}
}

// --- Feature 2: CallerSkip ---

func ExampleEvent_CallerSkip() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// In a wrapper function, use CallerSkip(1) to report the caller of the wrapper
	logWrapper := func(msg string) {
		logger.Info().CallerSkip(1).Msg(msg)
	}
	_ = logWrapper // demonstrates usage pattern
}

// --- Feature 3: Stringer ---

func ExampleEvent_Stringer() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().Stringer("addr", net.IPv4(127, 0, 0, 1)).Msg("connected")
	// Output: {"level":"info","addr":"127.0.0.1","message":"connected"}
}

func ExampleEvent_Stringer_nil() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().Stringer("val", nil).Msg("no value")
	// Output: {"level":"info","val":null,"message":"no value"}
}

// --- Feature 4: Ints / Strs ---

func ExampleEvent_Ints() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().Ints("ids", []int{10, 20, 30}).Msg("batch")
	// Output: {"level":"info","ids":[10,20,30],"message":"batch"}
}

func ExampleEvent_Strs() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().Strs("tags", []string{"go", "fast", "bolt"}).Msg("tagged")
	// Output: {"level":"info","tags":["go","fast","bolt"],"message":"tagged"}
}

// --- Feature 5: IPAddr ---

func ExampleEvent_IPAddr() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().IPAddr("client", net.IPv4(192, 168, 1, 100)).Msg("request")
	// Output: {"level":"info","client":"192.168.1.100","message":"request"}
}

// --- Feature 6: Dict ---

func ExampleEvent_Dict() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	logger.Info().Dict("user", func(d *bolt.Event) {
		d.Str("name", "alice").Int("age", 30)
	}).Msg("profile")
	// Output: {"level":"info","user":{"name":"alice","age":30},"message":"profile"}
}

// --- Feature 7: MultiHandler ---

func ExampleMultiHandler() {
	// Write logs to both stdout and a file (or any other writer)
	h := bolt.MultiHandler(
		bolt.NewJSONHandler(os.Stdout),
		bolt.NewJSONHandler(os.Stderr),
	)
	logger := bolt.New(h)
	_ = logger // demonstrates construction
}

// --- Feature 8: Hook / AddHook ---

// counterHook counts the number of log events processed.
type counterHook struct{ count int }

func (h *counterHook) Run(_ bolt.Level, _ string) bool {
	h.count++
	return true
}

func ExampleLogger_AddHook() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	hook := &counterHook{}
	logger.AddHook(hook)

	logger.Info().Msg("first")
	logger.Info().Msg("second")
	fmt.Println("events:", hook.count)
	// Output:
	// {"level":"info","message":"first"}
	// {"level":"info","message":"second"}
	// events: 2
}

// --- Feature 9: SampleHook ---

func ExampleNewSampleHook() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

	// Log only 1 out of every 100 events
	logger.AddHook(bolt.NewSampleHook(100))

	for i := 0; i < 200; i++ {
		logger.Info().Int("i", i).Msg("sampled")
	}
}

// --- Feature 10: NewLevelWriter ---

func ExampleNewLevelWriter() {
	logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
	w := bolt.NewLevelWriter(logger, bolt.ERROR)

	// Use with the standard log package
	stdlog := log.New(w, "", 0)
	stdlog.Print("something failed")
	// Output: {"level":"error","message":"something failed"}
}
