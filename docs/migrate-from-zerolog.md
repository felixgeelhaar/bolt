# Migrating from zerolog to bolt

If you already write Go services with [zerolog], you'll be at home in bolt
within minutes — the chained-builder API is intentionally close. The big
differences to know up front:

| Concern | zerolog | bolt |
|---|---|---|
| First-class log/slog handler | indirect via 3rd-party | yes (`bolt.NewSlogHandler`) |
| Zero-allocation budget | yes | yes (and tighter on most field types) |
| OpenTelemetry trace/span injection | manual | built-in via `Logger.Ctx(ctx)` |
| `Fatal()` exits the process | yes | yes |
| Hooks see full event | yes | level + message only (richer hook API on the v2 roadmap) |
| Pre-set context | `logger.With().Str(…).Logger()` | `logger.With().Str(…).Logger()` (identical) |

[zerolog]: https://github.com/rs/zerolog

## TL;DR — the 30-second swap

```go
// zerolog
log := zerolog.New(os.Stdout).With().Timestamp().Logger()
log.Info().Str("user", uid).Int("status", 200).Msg("ok")

// bolt
log := bolt.New(bolt.NewJSONHandler(os.Stdout))
log.Info().Str("user", uid).Int("status", 200).Msg("ok")
```

The chain shape (`logger.Level().Field(…).Msg(…)`) is identical. Most
production codebases need nothing beyond an import-path swap and a
constructor change.

## API mapping

### Constructing a logger

| zerolog | bolt |
|---|---|
| `zerolog.New(w)` | `bolt.New(bolt.NewJSONHandler(w))` |
| `zerolog.New(zerolog.ConsoleWriter{Out: w})` | `bolt.New(bolt.NewConsoleHandler(w))` |
| `log.Logger.Level(zerolog.InfoLevel)` | `logger.SetLevel(bolt.LevelInfo)` |
| `log.Logger.With().Str("svc","api").Logger()` | `logger.With().Str("svc","api").Logger()` |

### Levels

| zerolog | bolt |
|---|---|
| `zerolog.TraceLevel` | `bolt.LevelTrace` (or `bolt.TRACE`) |
| `zerolog.DebugLevel` | `bolt.LevelDebug` |
| `zerolog.InfoLevel` | `bolt.LevelInfo` |
| `zerolog.WarnLevel` | `bolt.LevelWarn` |
| `zerolog.ErrorLevel` | `bolt.LevelError` |
| `zerolog.FatalLevel` | `bolt.LevelFatal` |

### Field methods

The common methods (`Str`, `Int`, `Int8`–`Int64`, `Uint*`, `Float64`, `Bool`,
`Bytes`, `Time`, `Dur`, `Err`, `Stringer`, `Any`, `Dict`, `Ints`, `Strs`,
`IPAddr`) have the same signature in both libraries. A few names differ:

| zerolog | bolt |
|---|---|
| `e.Hex(key, []byte)` | `e.Bytes(key, []byte)` (hex-encoded) |
| `e.RawJSON(key, []byte)` | not yet exposed — open an issue if you need it |
| `e.Embed(o zerolog.LogObjectMarshaler)` | use `e.Dict(key, func(d *bolt.Event){…})` |

### Context-aware logging

zerolog requires manual trace ID extraction. bolt has it built in:

```go
// zerolog
sub := log.With().
    Str("trace_id", trace.SpanFromContext(ctx).SpanContext().TraceID().String()).
    Logger()

// bolt
sub := log.Ctx(ctx)  // injects trace_id and span_id automatically
```

### Hooks and sampling

```go
// zerolog
log.Hook(zerolog.HookFunc(func(e *zerolog.Event, lvl zerolog.Level, msg string) {
    metrics.IncrementLogCounter(lvl.String())
}))

// bolt
log.AddHook(bolt.HookFunc(func(lvl bolt.Level, msg string) bool {
    metrics.IncrementLogCounter(lvl.String())
    return true
}))

// Built-in sampling: 1 in N
log.AddHook(bolt.NewSampleHook(100))
```

bolt's current `Hook` interface only exposes level + message. If you write
zerolog hooks that inspect fields (redaction, cost accounting, etc.), file
an issue — a richer Hook v2 interface is on the roadmap.

## Worked example: 30-line HTTP service

```go
// before — zerolog
package main

import (
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/hlog"
    "net/http"
    "os"
    "time"
)

func main() {
    log := zerolog.New(os.Stdout).With().Timestamp().Logger()

    h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        hlog.FromRequest(r).Info().
            Str("method", r.Method).
            Str("path", r.URL.Path).
            Msg("request")
    })

    chain := hlog.NewHandler(log)(
        hlog.RequestIDHandler("req_id", "X-Request-ID")(
            hlog.AccessHandler(func(r *http.Request, status, size int, dur time.Duration) {
                hlog.FromRequest(r).Info().
                    Int("status", status).
                    Dur("duration", dur).
                    Msg("done")
            })(h)))

    http.ListenAndServe(":8080", chain)
}
```

```go
// after — bolt
package main

import (
    "github.com/felixgeelhaar/bolt"
    "github.com/google/uuid"
    "net/http"
    "os"
    "time"
)

func main() {
    log := bolt.New(bolt.NewJSONHandler(os.Stdout))

    h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        rid := r.Header.Get("X-Request-ID")
        if rid == "" {
            rid = uuid.New().String()
        }
        start := time.Now()

        log.Info().
            Str("req_id", rid).
            Str("method", r.Method).
            Str("path", r.URL.Path).
            Msg("request")

        // … handler work …

        log.Info().
            Str("req_id", rid).
            Int("status", 200).
            Dur("duration", time.Since(start)).
            Msg("done")
    })

    http.ListenAndServe(":8080", h)
}
```

bolt does not ship its own middleware bundle — for a complete reference
HTTP setup including request IDs, panic recovery, and OTel propagation,
see `examples/microservices/http-middleware/`.

## When to **not** migrate

- **Your team uses `zerolog/log` global API extensively.** bolt has a
  default logger, but it's lighter than zerolog's `log` package. If you
  rely on `log.Output()`, custom writers tied to zerolog's globals, or
  niche features like `RawJSON`, the migration cost may exceed the
  benefit until the v2 roadmap items land.
- **You depend on hooks that inspect fields.** Wait for Hook v2.
- **You're already content.** Bolt's win over zerolog is single-digit
  nanoseconds on simple paths. Switch when you're already doing other
  work in the logging layer (slog adoption, OTel rollout) — not as a
  standalone exercise.
