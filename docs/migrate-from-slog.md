# Migrating from `log/slog` to bolt

If you're already using the standard library's [log/slog], you don't
have to leave it. bolt ships a `slog.Handler` implementation
(`bolt.NewSlogHandler`) that gives you bolt's zero-allocation JSON
encoding without changing a single call site.

[log/slog]: https://pkg.go.dev/log/slog

## Path 1 — keep the slog API, swap the handler (recommended)

This is the lowest-risk path. Every `slog.Logger` call site stays the
same; only the handler construction changes.

```go
// before
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

// after — bolt-backed slog
logger := slog.New(bolt.NewSlogHandler(os.Stdout, nil))

// every existing call still works
logger.Info("ok", "user", uid, "status", 200)
logger.WithGroup("request").Info("done", "method", "POST", "status", 201)
```

The bolt `slog.Handler` passes the standard
`testing/slogtest.TestHandler` conformance suite:

- `WithGroup` produces correctly nested JSON objects (not dotted keys).
- `WithAttrs` is scoped to whichever group was active when it was called.
- Empty groups are omitted from the output.
- Empty-key attrs are ignored.
- `slog.LogValuer` values are resolved before encoding.

### Configuring the handler

```go
opts := &bolt.SlogHandlerOptions{
    Level:     slog.LevelDebug,
    AddSource: true,
}
logger := slog.New(bolt.NewSlogHandler(os.Stdout, opts))
```

### Performance

bolt's slog handler keeps the encoding budget close to the bolt-native
path. Independent benchmarks (run `go test -bench=BenchmarkSlog -benchmem`)
typically show:

- 0 allocations per record on the common path
- ~80–110 ns/op for a record with 3–5 fields

Compare against `slog.NewJSONHandler` which usually allocates 2–4× more.

## Path 2 — adopt bolt's chained API directly

If you want the slimmest possible call sites and don't mind moving call
sites, bolt's chained API is faster (zero allocations, no
`slog.Attr` heap traffic).

```go
// slog (varargs of slog.Attr — heap traffic on every call)
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("ok",
    slog.String("user", uid),
    slog.Int("status", 200),
)

// bolt — chained, zero alloc
logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
logger.Info().
    Str("user", uid).
    Int("status", 200).
    Msg("ok")
```

The mapping is straightforward — bolt has equivalents for every common
slog kind:

| slog | bolt |
|---|---|
| `slog.String("k", v)` | `.Str("k", v)` |
| `slog.Int("k", v)` | `.Int("k", v)` |
| `slog.Bool("k", v)` | `.Bool("k", v)` |
| `slog.Float64("k", v)` | `.Float64("k", v)` |
| `slog.Time("k", v)` | `.Time("k", v)` |
| `slog.Duration("k", v)` | `.Dur("k", v)` |
| `slog.Group("k", a, b, c)` | `.Dict("k", func(d *bolt.Event){…})` |
| `slog.Any("k", v)` | `.Any("k", v)` |

### Levels

`slog.Level` is an `int`; bolt's `Level` is also an `int`-based custom
type. The constants line up:

| slog | bolt |
|---|---|
| `slog.LevelDebug` | `bolt.LevelDebug` |
| `slog.LevelInfo` | `bolt.LevelInfo` |
| `slog.LevelWarn` | `bolt.LevelWarn` |
| `slog.LevelError` | `bolt.LevelError` |
| (none) | `bolt.LevelTrace` (below Debug) |
| (none) | `bolt.LevelFatal` (calls `os.Exit(1)`) |

Custom levels (e.g. `slog.Level(1)` for "Info+1") are not first-class in
bolt — file an issue if you depend on them.

### Groups → Dict

slog's `WithGroup` and `slog.Group` both nest attrs. In bolt's chained
API the equivalent is `Dict`:

```go
// slog
logger.WithGroup("request").Info("done",
    slog.String("method", "POST"),
    slog.Int("status", 201),
)

// bolt — chained
logger.Info().
    Dict("request", func(d *bolt.Event) {
        d.Str("method", "POST").Int("status", 201)
    }).
    Msg("done")
```

If you need *handler-level* group scoping (e.g. "every record from this
logger nests under `request`"), keep using the slog API
(`slog.New(bolt.NewSlogHandler(…)).WithGroup(…)`).

## Hybrid

Mix freely. The chained API is for hot paths; the slog API is for
existing code, libraries that take a `*slog.Logger`, and any place where
you want stdlib portability.

```go
boltJSON := bolt.NewJSONHandler(os.Stdout)
fast    := bolt.New(boltJSON)                    // chained, hot path
ergo    := slog.New(bolt.NewSlogHandler(os.Stdout, nil)) // ergonomic
```

Both write to the same destination (or different ones), both share
bolt's encoder.
