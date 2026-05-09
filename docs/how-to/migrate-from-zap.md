# Migrating from zap to bolt

[zap]'s API is a different shape from bolt's: zap leans on typed
constructors (`zap.String("k", v)`) passed positionally, while bolt
chains them on the event (`event.Str("k", v)`). The migration is
mechanical but per-call-site.

| Concern | zap | bolt |
|---|---|---|
| API style | `logger.Info("msg", zap.String(…), zap.Int(…))` | `logger.Info().Str(…).Int(…).Msg("msg")` |
| Sugared logger | `SugaredLogger` (key/value pairs) | use `slog.New(bolt.NewSlogHandler(…))` |
| Allocations on hot path | 1 alloc/op (typed constructors) | 0 allocs/op |
| log/slog handler | external bridge | first-class (`bolt.NewSlogHandler`) |
| OpenTelemetry trace/span injection | manual | built-in via `Logger.Ctx(ctx)` |
| `Fatal()` exits | yes | yes |
| Field encoder customisation | rich (`zapcore.EncoderConfig`) | smaller surface — open an issue if you have a use case |

[zap]: https://github.com/uber-go/zap

## TL;DR — the 30-second swap

```go
// zap
logger, _ := zap.NewProduction()
logger.Info("ok",
    zap.String("user", uid),
    zap.Int("status", 200),
)

// bolt
logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
logger.Info().
    Str("user", uid).
    Int("status", 200).
    Msg("ok")
```

The Msg() call is mandatory in bolt — it's what flushes the event. If
you forget it, the event is silently dropped (the event pool reclaims
the buffer when GC runs). A `go vet` analyzer for this is on the
roadmap.

## API mapping

### Constructing a logger

| zap | bolt |
|---|---|
| `zap.NewProduction()` | `bolt.New(bolt.NewJSONHandler(os.Stdout))` |
| `zap.NewDevelopment()` | `bolt.New(bolt.NewConsoleHandler(os.Stdout))` |
| `logger.With(zap.String("svc","api"))` | `logger.With().Str("svc","api").Logger()` |
| `logger.Sugar()` | `slog.New(bolt.NewSlogHandler(os.Stdout, nil))` |

### Levels

| zap | bolt |
|---|---|
| `zapcore.DebugLevel` | `bolt.LevelDebug` |
| `zapcore.InfoLevel` | `bolt.LevelInfo` |
| `zapcore.WarnLevel` | `bolt.LevelWarn` |
| `zapcore.ErrorLevel` | `bolt.LevelError` |
| `zapcore.FatalLevel` | `bolt.LevelFatal` |

bolt has a `TRACE` / `LevelTrace` below DEBUG; zap does not.

### Field constructors → field methods

| zap | bolt |
|---|---|
| `zap.String("k", v)` | `.Str("k", v)` |
| `zap.Int("k", v)` | `.Int("k", v)` |
| `zap.Int8/16/32/64("k", v)` | `.Int8/16/32/64("k", v)` |
| `zap.Uint("k", v)` | `.Uint("k", v)` |
| `zap.Float64("k", v)` | `.Float64("k", v)` |
| `zap.Bool("k", v)` | `.Bool("k", v)` |
| `zap.Time("k", v)` | `.Time("k", v)` |
| `zap.Duration("k", v)` | `.Dur("k", v)` |
| `zap.Error(err)` | `.Err(err)` |
| `zap.Stringer("k", v)` | `.Stringer("k", v)` |
| `zap.Any("k", v)` | `.Any("k", v)` |
| `zap.Reflect("k", v)` | `.Any("k", v)` (uses `encoding/json` reflection) |
| `zap.Object("k", marshaler)` | `.Dict("k", func(d *bolt.Event){…})` |
| `zap.Array("k", arr)` | `.Ints("k", []int{…})` / `.Strs("k", []string{…})` |
| `zap.ByteString("k", v)` | `.Bytes("k", v)` |
| `zap.Stack("k")` | not yet exposed — open an issue if you need it |

### Sugared logger → slog

zap's `SugaredLogger` accepts loose key/value pairs; the equivalent in
bolt is to wrap `bolt.NewSlogHandler` in a stdlib `slog.Logger`:

```go
// zap
sugar := logger.Sugar()
sugar.Infow("ok", "user", uid, "status", 200)

// bolt — via slog
log := slog.New(bolt.NewSlogHandler(os.Stdout, nil))
log.Info("ok", "user", uid, "status", 200)
```

The slog handler passes the standard `testing/slogtest.TestHandler`
conformance suite, so anything the slog ecosystem (group nesting,
`LogValuer`, `WithAttrs` scoping) expects, works.

### Hooks

zap's `WithOptions(zap.Hooks(…))` callback receives a full
`zapcore.Entry`. bolt's current `Hook.Run(level, msg) bool` is
narrower; if your hooks read fields (redaction, sampling by attribute,
cost accounting), wait for Hook v2 or pre-process inputs before
calling `.Str(…)`.

### Sampling

| zap | bolt |
|---|---|
| `zap.WrapCore(zapcore.NewSamplerWithOptions(…))` | `logger.AddHook(bolt.NewSampleHook(N))` |

bolt's sampler keeps 1 of every N events at the same level. Adaptive,
field-aware sampling is on the roadmap.

## When to **not** migrate

- **You depend on `zapcore.EncoderConfig` customisation** (custom level
  text, custom timestamp formats per-call). bolt's encoder surface is
  smaller; open an issue with the use case before migrating.
- **You rely on `zap.Object` with rich `MarshalLogObject` types.** bolt's
  `Dict` is closure-based and works, but porting many marshalers can be
  busywork. Worth doing if you're already adopting slog.
- **You're satisfied with zap.** Bolt's allocation win on the hot path
  helps when logging is in a measurable pprof line item; for most
  services the difference is in the noise.
