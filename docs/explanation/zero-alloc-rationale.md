# Why zero allocations

Bolt's design centres on producing zero heap allocations on the hot
logging path. This page explains why that goal exists, what it costs,
and where it stops mattering.

## What "zero allocation" means here

For the typical chained call —

```go
log.Info().Str("user", uid).Int("status", 200).Msg("ok")
```

— the entire pipeline (event lease from pool, field encoding, JSON
serialisation, write to handler, event return to pool) produces zero
heap allocations as measured by `go test -benchmem`. The
`BenchmarkZeroAllocation` test in the parent module gates this, and
the [benchstat workflow](../../.github/workflows/pr-checks.yml) catches
statistical regressions across PRs.

A handful of methods explicitly trade allocation for ergonomics:
`Any` (reflection through `encoding/json`), `Fields` (map iteration),
`Bytes` (string-from-byte-slice copy), `Stack` and `Caller*` (frame
walking). They're documented at the call site.

## Why it matters

Three regimes where allocation cost dominates:

1. **High-throughput services** (>10k log records/sec/core). Each
   allocation triggers the GC's write barrier and contributes to
   pause time. A logger producing 5 allocations/log at 50k logs/sec
   adds 250k allocations/sec of GC pressure — typically the largest
   single source of allocator traffic in the process.
2. **Latency-sensitive services** (P99 budgets ≤ 10ms). GC pauses
   manifest as P99 spikes. Removing the logger from the allocator
   pipeline removes one of the easier sources of jitter.
3. **Sidecar / agent processes** with constrained memory budgets
   (~50 MB total RSS). Steady-state allocation rate compounds into
   working-set growth between collections; lower allocation rate
   means smaller live heap.

If your service is none of those — a CRUD API logging a few hundred
records per second on a host with 8 GB free — the allocation
difference between bolt and `log/slog` is in the noise. Don't pick a
logger on it; pick on the slog API match, the OTel correlation, or
the migration cost.

## What it costs

The chained-builder API has a known footgun: `Msg` is a
terminator that you must call. Forgetting it does not produce a
compile error; the event sits in the pool, drained by GC, and the
log line never reaches the handler. zerolog has the same pattern;
zap's typed constructors don't (because there's no chain), but pay
1 alloc/op via the typed-Field heap traffic.

A `go vet` analyser to catch missing terminators is tracked on the
roadmap. Until then, the cost is reviewer attention plus running
your tests.

The encoder is custom rather than `encoding/json`. That means:

- A small number of types are first-class (everything in the
  reference's "Field types" page); falling back to `Any` allocates.
- We don't get free schema evolution from struct tags; if you want a
  log line that mirrors a struct, you write the calls explicitly or
  pay the `Any` cost.

## Where the goal stops

Zero-alloc is a hot-path constraint, not a religion. Branch points
where bolt happily allocates:

- `slog.Handler` bridge: the slog API requires materialising
  `slog.Attr` values; we don't fight that.
- `MultiHandler`: copies the handler list at construction (one-time).
- Console formatting: parses the JSON buffer for level + message
  extraction; allocations bounded by the field count, but it's a
  development-time handler, not a production hot path.
- `Caller`/`CallerSkip`: `runtime.Caller` allocates a frame slice;
  `fmt.Sprintf` for `file:line` allocates the result string.

If you've hit a path where the budget breaks unexpectedly, check
`BenchmarkZeroAllocation` reproduces the issue on your machine and
file an issue with the benchstat output.
