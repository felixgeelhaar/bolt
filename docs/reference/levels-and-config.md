# Levels, configuration, and environment overrides

## Levels

Six levels in increasing severity:

| Constant | slog-style alias | Usage |
|---|---|---|
| `bolt.TRACE` | `bolt.LevelTrace` | Verbose tracing; off by default |
| `bolt.DEBUG` | `bolt.LevelDebug` | Development-time diagnostics |
| `bolt.INFO`  | `bolt.LevelInfo`  | Normal operational events |
| `bolt.WARN`  | `bolt.LevelWarn`  | Recoverable abnormal conditions |
| `bolt.ERROR` | `bolt.LevelError` | Application-visible failures |
| `bolt.FATAL` | `bolt.LevelFatal` | Emit then `os.Exit(1)` |

The SCREAMING_CASE and slog-style aliases are interchangeable —
they're literally the same constants. Prefer `LevelInfo` etc. in new
code so it lines up with `log/slog`.

## Setting the level

```go
log := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.LevelInfo)
```

Level reads use `sync/atomic`, so `SetLevel` is safe to call
concurrently with logging — useful for runtime level toggles.

## Environment overrides

The default logger (the one accessed via package-level `bolt.Info()`,
`bolt.Error()`, etc.) honours two environment variables at init time:

| Variable | Values | Effect |
|---|---|---|
| `BOLT_LEVEL` | `trace` \| `debug` \| `info` \| `warn` \| `error` \| `fatal` | Sets the default logger's minimum level |
| `BOLT_FORMAT` | `json` \| `console` | Picks the default handler |

Custom loggers (created with `bolt.New(...)`) ignore these — they
take their handler and level from the explicit constructor calls.

## Fatal semantics

`logger.Fatal()` writes the record and then calls `os.Exit(1)`,
matching `zap`, `zerolog`, `logrus`, and `slog`. To exercise Fatal
code paths in tests without terminating the test binary, override
the unexported `exitFunc` package variable — see `fatal_test.go` in
the parent module for the pattern.

## Thread safety

- `JSONHandler`, `ConsoleHandler`, `SlogHandler`: serialise writes
  via `sync.Mutex` internally. A single handler instance is safe for
  concurrent use.
- `Logger`: safe for concurrent use; level changes are atomic.
- `Logger.AddHook` / `Logger.AddEventHook`: setup-time API. NOT safe
  to call concurrently with logging operations on the same logger.
- `Event`: NOT safe for concurrent use. Each `Logger.Info()` etc.
  returns a fresh event from a per-logger pool.
- Custom `Handler` implementations are responsible for their own
  synchronisation.

## Slog interop

`bolt.NewSlogHandler(out io.Writer, opts *bolt.SlogHandlerOptions)`
returns a `slog.Handler`. Options:

```go
opts := &bolt.SlogHandlerOptions{
    Level:     slog.LevelDebug,  // optional; defaults to slog.LevelInfo
    AddSource: true,             // optional; default false
}
slog.SetDefault(slog.New(bolt.NewSlogHandler(os.Stdout, opts)))
```

The handler passes `testing/slogtest.TestHandler` — `WithGroup`
nests as a JSON object (not dotted keys), `WithAttrs` is correctly
scoped to the group active at call time, empty groups are omitted,
empty-key attrs are dropped, and `LogValuer` is resolved.

## Stdlib `log` bridge

Adapt a `*bolt.Logger` to `io.Writer` for libraries that take the
stdlib `log.Logger`:

```go
w := bolt.NewLevelWriter(myLog, bolt.LevelError)
stdlog.New(w, "", 0).Print("legacy error")
// → bolt ERROR record
```
