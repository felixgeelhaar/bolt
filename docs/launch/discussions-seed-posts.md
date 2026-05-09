# GitHub Discussions seed posts

Drafts for the seed Discussion threads recommended by the GTM
review. The maintainer posts these manually after enabling the
Discussions tab. This file keeps the copy under version control so
edits are visible and contributors can suggest tweaks.

## How to use

1. Open the repository's [Discussions tab](https://github.com/felixgeelhaar/bolt/discussions).
2. For each section below, create a new Discussion in the matching
   category (Q&A or Show and tell as noted), pasting the title +
   body.
3. Update this file with links to the posted threads once they're
   live.

---

## 1. (Q&A) "When should I pick bolt over `log/slog`?"

**Category**: Q&A

**Body**:

```markdown
This is the question every evaluator asks first. Short answer:

**Pick `log/slog`** if:
- Your service is starting fresh and you want stdlib for the long run.
- Logging doesn't show up as a top-10 line in your pprof.
- You're not using OpenTelemetry, or you have a pre-existing trace/span injection helper you're happy with.

**Pick bolt** if:
- You want the slog API but logging is in your hot path. bolt's `slog.Handler` keeps the slog ergonomics with zero allocations on the encoding path.
- You want first-class OpenTelemetry integration: `Logger.Ctx(ctx)` automatically attaches `trace_id` and `span_id`. With slog you'd write that helper yourself.
- You're already on zerolog or zap and the chained-builder API is in your team's muscle memory; bolt's chained API is intentionally close to zerolog's.

This is the question the README also leads with — the table at <https://github.com/felixgeelhaar/bolt#why-bolt-vs-slog--zerolog--zap>.

Edge cases worth flagging in this thread:
- Custom levels (between Info and Warn etc.) — slog supports `slog.Level(N)`; bolt doesn't first-class them.
- Custom encoders (different timestamp format per log) — slog more flexible via `ReplaceAttr`; bolt's encoder is intentionally narrow.

Reply with your situation if you're not sure which side you fall on; happy to talk through specific cases.
```

---

## 2. (Q&A) "Migrating from zerolog: gotchas?"

**Category**: Q&A

**Body**:

```markdown
The migration guide at [docs/how-to/migrate-from-zerolog.md](https://github.com/felixgeelhaar/bolt/blob/main/docs/how-to/migrate-from-zerolog.md) covers the API mapping. Real-world gotchas the guide doesn't yet have first-hand reports on:

- Hooks. zerolog's `Hook` sees the event; bolt v1's `Hook` only sees level + message. For redaction or per-event metric extraction, use `EventHook` (added in v1.4) — see [Hook v2 design notes](https://github.com/felixgeelhaar/bolt/blob/main/docs/explanation/hook-v2.md).
- `RawJSON` is not yet in the bolt API. If you used it for embedding pre-marshalled JSON, current workaround is `Bytes` (copies) or open an issue.
- Float precision. bolt's `Float64` uses `strconv.AppendFloat('g', -1)` matching `encoding/json`. zerolog had a different default precision for some sub-cases. If you assert exact JSON output in tests, expect to update fixtures.
- Sub-loggers. Both libraries use `Logger.With()...Logger()`; chain shape is identical.

If you've done the migration, please reply with what you hit. The guide gets better with first-hand reports.
```

---

## 3. (Show and tell) "OTel correlation patterns we ship in bolt"

**Category**: Show and tell

**Body**:

```markdown
The flagship example at [`examples/observability/opentelemetry/`](https://github.com/felixgeelhaar/bolt/tree/main/examples/observability/opentelemetry) wires bolt to OTel tracing + Prometheus metrics. The pattern that did the most for our debugging experience: `Logger.Ctx(ctx)` automatically attaches `trace_id` and `span_id` so log shippers (Loki, Datadog, etc.) can jump from a log line to the trace it belongs to without bespoke glue.

```go
log := bolt.New(bolt.NewJSONHandler(os.Stdout))

http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    ctx, span := tracer.Start(r.Context(), "handler")
    defer span.End()

    // Every log line in this scope auto-includes trace_id + span_id
    sub := log.Ctx(ctx)
    sub.Info().Str("path", r.URL.Path).Msg("handling")
    // ... rest of handler
    sub.Info().Int("status", 200).Msg("done")
})
```

Reply with your OTel + bolt patterns — particularly interesting if you're using a non-default propagator (W3C Baggage, OT-shim, etc.).
```

---

## 4. (Q&A) "Benchmarking bolt: methodology"

**Category**: Q&A

**Body**:

```markdown
Two related questions show up regularly: "what numbers should I expect?" and "how do you guard against perf regressions?"

**Expected numbers** depend heavily on hardware. The CI gate
`BenchmarkZeroAllocation` requires 0 allocs/op for the simple
`logger.Info().Str(...).Msg(...)` chain. Wall-clock numbers I've
measured locally:

- M-series Mac: ~60-80 ns/op
- ubuntu-latest GHA runner: ~150-300 ns/op (shared infra noise)

**Regression guard**: the [pr-checks workflow](https://github.com/felixgeelhaar/bolt/blob/main/.github/workflows/pr-checks.yml) runs `benchstat -delta-test=utest` (Mann-Whitney U-test, alpha=0.05) comparing baseline (target branch) vs PR. The job fails only on statistically significant regressions, not on routine ±50% noise on shared runners. The previous wall-clock thresholds in `performance_test.go` were retired because they produced false positives that got muted into noise — the worst kind of test.

If you've done your own benchmark and the numbers surprise you (in either direction), share the methodology and I'll try to reproduce.
```

---

## 5. (Show and tell) "Hook v2 use cases"

**Category**: Show and tell

**Body**:

```markdown
The new `EventHook` interface (v1.4+) sees the `*Event` mid-build, so you can:

- Suppress events containing sensitive keys (see `bolt/genai.NewRedactHook`)
- Sample low-importance events but always-keep `gen_ai.*` and `error` levels (see `bolt/genai.NewAdaptiveSampler`)
- Tag events with tenant / region / build identifiers
- Extract per-event metrics (token counts, latency buckets) into Prometheus

What are you using it for? If your hook is generic, consider PRing it into the `bolt/genai` sub-module so others can use it.

Bad use cases — covered in the [Hook v2 design notes](https://github.com/felixgeelhaar/bolt/blob/main/docs/explanation/hook-v2.md):

- In-place buffer mutation. The `Buffer()` accessor is read-only; mutation corrupts the in-flight record.
- Surrogate-pair-aware redaction of UTF-8 prompts. Doable in a hook but expensive; better placed in the OTel collector.
- Anything that needs to call back into the same logger. Will deadlock the handler mutex; use a separate fire-and-forget channel.
```
