# Bolt Roadmap

This file is the public, dated commitment for what's planned in bolt.
The full task graph (with dependencies, owners, and status transitions)
lives in `.roady/spec.yaml`. This document is the human-readable view
and is updated when priorities change.

bolt's compatibility promise: **v1.x will not break the public API.**
Anything in this roadmap that would break callers will land behind a
new constructor / opt-in flag, or be deferred to a v2 cycle with a
documented migration script and tag retraction policy. Decisions like
the slog group-nesting fix, which technically changed JSON output
shape for the slog handler, are called out as breaking in the
`CHANGELOG`.

---

## Themes

The active themes for the current cycle:

1. **Trust** — every example builds; flagship CI gates are strict;
   every CHANGELOG entry is honest about behaviour changes.
2. **Positioning** — bolt is "the zero-alloc slog handler with
   first-class OpenTelemetry," not "the fastest Go logger" — that
   framing wins single-digit-nanosecond comparisons but loses the
   adoption argument against `log/slog`.
3. **Ergonomics for migrators** — first-class migration paths from
   zerolog, zap, and slog (see `docs/migrate-from-*.md`).
4. **Production correctness** — the engineering review surfaced a
   handful of incident-grade footguns; closing them is non-negotiable
   before we invest in new surface area.
5. **Supply-chain hardness** — a logging library is imported by every
   service, so SLSA-3, signed artefacts, and SBOM are not optional.

## Status legend

- ✅ — shipped (linked to release tag where applicable)
- 🚧 — in progress on a tracked branch
- 🟡 — planned for the current quarter
- ⚪ — backlog; sequence not yet committed
- ❌ — explicitly dropped from scope (rationale linked)

---

## P0 — Correctness & Trust

These are the items the multi-expert review flagged as
incident-blocking or trust-blocking. They must ship before any P1+
investment.

| Status | Item | Notes |
|---|---|---|
| ✅ | OTel example compiles, strict CI gate enforces every example builds | PR #48 |
| ✅ | `Logger.Fatal()` calls `os.Exit(1)` after the record is written | PR #48 |
| ✅ | `JSONHandler` and `ConsoleHandler` serialize writes via `sync.Mutex` | PR #48 |
| ✅ | Event pool drops oversized buffers (`PoolBufferCap = 8 KB`) | PR #48 |
| ✅ | `SlogHandler` passes `testing/slogtest.TestHandler` | PR #48 (breaking shape change for previous dotted-key consumers) |

## P1 — Positioning & DX

| Status | Item | Notes |
|---|---|---|
| 🚧 | README rewrite — lead with slog+OTel positioning, fold deployment / troubleshooting / limitations to docs site | reduces ~730 LOC README to <250 |
| ✅ | Migration guides — `docs/migrate-from-{zerolog,zap,slog}.md` | this PR |
| ✅ | Truthful `examples/README.md` | this PR |
| ✅ | `ROADMAP.md` published | you are here |
| ✅ | slog-style level aliases (`bolt.LevelInfo`, …) | this PR |

## P2 — Engineering Hygiene

| Status | Item | Notes |
|---|---|---|
| ✅ | Split `bolt.go` (~2000 LOC) into `event.go`, `logger.go`, `handler.go`, `encode.go` | done across PRs #55, #57, #58 — code-motion only; no API change; bolt.go is now 508 lines |
| ✅ | Hook v2 — `EventHook` interface receives `*Event` so hooks can inspect fields, suppress, or tag (redaction, cost accounting, sensitive-content gating) | additive; existing `Hook` keeps working |
| ✅ | benchstat-driven perf regression in CI | done in #51; replaces flaky wall-clock asserts in `performance_test.go` |
| ✅ | Replace custom `appendFloat64` with `strconv.AppendFloat` | done in #50; preserves full IEEE-754 precision instead of 6-decimal truncation |
| ✅ | Property tests via `pgregory.net/rapid` for JSON escaping | done in #53; invariant: `decode(encode(x)) == x` for any string |
| ⚪ | Mutation testing on hot paths (gremlins, nightly, gate ≥70% on critical funcs) | scheduled, not per-PR |

## P3 — Ecosystem & Trust

| Status | Item | Notes |
|---|---|---|
| ✅ | `bolt/genai` sub-module — own go.mod tracking OTel GenAI semconv 1:1 | done in #60 (Call/ToolCall) and #64 (RedactHook + AdaptiveSampler) |
| ✅ | SLSA-3 release pipeline — re-enable goreleaser checksums, add cosign signing, syft SBOM, slsa-github-generator provenance | done in #62 |
| ✅ | `ADOPTERS.md` + governance disclosure in `SECURITY.md` | done in #52 |
| ✅ | Diataxis docs split | tutorial / how-to / reference / explanation in `docs/`; GH Pages renderer choice deferred to a separate PR |
| ⚪ | OSS-Fuzz onboarding | external maintainer action — submit project + Dockerfile + build.sh per OSS-Fuzz Go template |

## P4 — Discovery & Growth

| Status | Item | Notes |
|---|---|---|
| ⚪ | "From zerolog to bolt" comparison blog post | one real HTTP service, before/after pprof + benchstat, honest tradeoffs |
| ⚪ | Awesome-Go submission under Logging | |
| ⚪ | Seed 5–10 GitHub Discussions threads — migration FAQs, when-not-to-use, OTel patterns, benchmarking methodology | |

---

## Explicitly out of scope (for now)

- **A built-in tokenizer or LLM pricing table.** The AI review concluded
  that pricing tables drift faster than a logging library can release;
  cost attribution belongs in the OTel collector or a server-side tool
  (Langfuse, Phoenix, Braintrust). bolt will expose richer hooks; users
  plug in.
- **PII detection (regex/ML) in core.** False positives at 60 ns/op
  break the perf story. Hook surface is the right extension point.
- **Logger globals beyond the existing default logger.** A second
  ergonomics layer competes with the slog API for no clear gain.
- **Custom encoder configuration on the level of `zapcore.EncoderConfig`.**
  bolt's encoder is intentionally narrow. If you need per-call timestamp
  formats or custom level text, file an issue with the use case before
  forking.

## How to influence this roadmap

- **File an issue** describing the use case. Concrete user requests
  beat speculative scaffolding for prioritisation.
- **Open a discussion** if the scope is fuzzy. The "out of scope"
  list is a starting point, not a religion.
- **PR welcome** for any P2/P3 item — coordinate via an issue first
  so we don't race.
