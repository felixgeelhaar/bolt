# Hook v2 — field-aware interception design

The original `bolt.Hook` interface only sees the level and message:

```go
type Hook interface {
    Run(level Level, msg string) bool
}
```

That's enough for "increment a metric per log line" or "sample 1 of
100", but not for the things the multi-expert review identified as
critical: redaction, sensitive-content gating, cost accounting,
field-aware sampling. None of those work without seeing the fields.

This page documents the design we landed on and the trade-offs we
explicitly rejected.

## What shipped

```go
type EventHook interface {
    Run(e *Event, msg string) bool
}

func (l *Logger) AddEventHook(hook EventHook) *Logger

// Read-only accessors on *Event:
func (e *Event) Level() Level
func (e *Event) Buffer() []byte
func (e *Event) WalkFields(fn func(key, value []byte) bool) int
```

`EventHook` runs during `Msg()` after every legacy `Hook` succeeds.
Returning `false` suppresses the event. Hooks can also tag the event
by calling `e.Str(...)`, `e.Int(...)`, etc. — the existing field
methods continue to work.

## Why an alias rather than mutation primitive

Three options were on the table for "hooks that can change what
ships":

1. **Read-only `Buffer()` + `WalkFields`** (shipped). Hooks can
   inspect; the only state-changing operation available to them is
   adding a new field via the existing public API.
2. **Mutable `Buffer()`** that returns a writable slice. Rejected:
   the buffer aliases the in-flight log record at the moment hooks
   run. A hook stomping on it can corrupt the JSON shape, break the
   message append in `Msg`, or leak the previous record's tail into
   the next pool reuse. The contract we'd need to write — "you can
   mutate but only between offsets X and Y, only with valid JSON,
   never longer than original" — is not enforceable by the type
   system and would inevitably cause incidents.
3. **A `Redact(key, replacement)` helper** on `Event`. Rejected for
   v1: it commits us to a regex / glob substitution semantics that
   may not match what users want, and "shipping a sanitised variant"
   is more cleanly done by emitting two log records explicitly than
   by surgery on a partial buffer.

Option 1 keeps the contract honest: the hook can suppress (always
correct), inspect (always correct), or tag (uses existing
zero-alloc methods). For redaction in v1 the recommended pattern is
"suppress sensitive events; emit a redacted variant explicitly if
you want to ship something" — see `bolt/genai/hooks.go` for the
reference implementation.

## Order of execution

`Msg` runs hooks in this order:

1. Validate `message` length (reject > MaxValueLength).
2. Iterate `l.hooks` (legacy `Hook`). First `false` suppresses; the
   event is recycled, no handler write, EventHooks NOT called.
3. Iterate `l.eventHooks` (`EventHook`). First `false` suppresses.
4. Append `message` field, close the JSON object, emit a newline.
5. Call `handler.Write`.
6. Recycle the event (drop oversize buffers per `PoolBufferCap`).
7. If level was `FATAL`, call `exitFunc(1)`.

The legacy → field-aware order is deliberate: legacy hooks were
written without seeing fields, so they don't depend on what later
field-tagging hooks add. If you need the inverse (e.g. add a
correlation ID via EventHook then suppress with a legacy hook
based on the now-tagged record), invert the registration order or
combine into a single EventHook.

## Composition with the legacy `SampleHook`

`bolt.NewSampleHook(N)` is still the right tool for uniform 1-in-N
sampling — it's smaller and faster than walking fields. The
field-aware `AdaptiveSampler` in `bolt/genai` is for the case where
some classes of event must always pass (`gen_ai.*`, `error`, etc.)
and the rest can sample. Stack them:

```go
log.AddHook(bolt.NewSampleHook(10)).               // baseline 1/10
    AddEventHook(genai.NewAdaptiveSampler(100))   // override: keep gen_ai.*
```

Order matters per the rules above.

## Performance budget

Adding `EventHook` to the Msg path costs roughly 3 ns/op when no
event hooks are registered (a nil-check + range-over-empty-slice).
With one hook registered, the cost is dominated by whatever the
hook does — `WalkFields` is O(buffer size); a Str/Int tag from a
hook is the same cost as calling the method from the call site.

`BenchmarkZeroAllocation` is unaffected because the hook list is
nil for the benchmark logger.
