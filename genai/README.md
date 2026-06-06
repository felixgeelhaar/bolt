# bolt/genai

`bolt/genai` is a thin annotator on top of [`bolt`](../) that emits the
[OpenTelemetry GenAI semantic-convention][otel-gen-ai] field names for
LLM calls and tool/function invocations.

[otel-gen-ai]: https://opentelemetry.io/docs/specs/semconv/gen-ai/

## Why a separate sub-module

bolt v1.x freezes its public API. The OTel GenAI semconv is still
moving — keeping this package in its own `go.mod` lets it track
semconv updates without dragging the bolt core through breaking
releases.

It also keeps the bolt core small. The AI review's strongest signal:
LLM tokens, pricing tables, and PII redaction belong in OTel
collectors and downstream tools (Langfuse, Phoenix, Braintrust), not
in a logging primitive. This package only does the *naming* — emit
the right keys with the right shapes so those downstream tools can
ingest bolt logs without a translation layer.

## What it is and isn't

- ✅ Field-name normalisation per the OTel GenAI semconv
- ✅ Skips zero-valued fields, computes `total_tokens` automatically
- ✅ Composes naturally with the bolt chained API
- ❌ No tokenizer (counts must come from the provider response)
- ❌ No pricing table (use the OTel collector / downstream tools)
- ❌ No PII redaction (use a [`bolt.EventHook`](../README.md#hooks-and-sampling))

## Install

```bash
go get go.klarlabs.de/bolt/genai
```

The package re-exports nothing — `bolt` itself is your logger entry
point. `genai.Call` and `genai.ToolCall` add fields to existing
`*bolt.Event` chains.

## Quick start

```go
package main

import (
    "os"
    "time"

    "go.klarlabs.de/bolt"
    "go.klarlabs.de/bolt/genai"
)

func main() {
    log := bolt.New(bolt.NewJSONHandler(os.Stdout))

    genai.Call(log.Info(), genai.CallInfo{
        System:        "openai",
        Operation:     "chat",
        RequestModel:  "gpt-4o",
        ResponseModel: "gpt-4o-2024-08-06",
        InputTokens:   142,
        OutputTokens:  87,
        Latency:       450 * time.Millisecond,
        ResponseID:    "chatcmpl-abc123",
        FinishReason:  "stop",
    }).Msg("chat call complete")
}
```

Output:

```json
{
  "level": "info",
  "gen_ai.system": "openai",
  "gen_ai.operation.name": "chat",
  "gen_ai.request.model": "gpt-4o",
  "gen_ai.response.model": "gpt-4o-2024-08-06",
  "gen_ai.usage.input_tokens": 142,
  "gen_ai.usage.output_tokens": 87,
  "gen_ai.usage.total_tokens": 229,
  "gen_ai.client.operation.duration": 450000000,
  "gen_ai.response.id": "chatcmpl-abc123",
  "gen_ai.response.finish_reasons": ["stop"],
  "message": "chat call complete"
}
```

## Mixing with regular bolt fields

```go
e := log.Info().
    Str("user_id", "u-123").
    Str("tenant", "acme")
genai.Call(e, genai.CallInfo{
    System:    "openai",
    Operation: "chat",
}).Msg("user-initiated chat")
```

## Tool calls

```go
genai.ToolCall(log.Info(), genai.ToolCallInfo{
    Name:         "search_web",
    CallID:       "call_xyz",
    ArgsLength:   142,   // bytes; we never log the body, only the size
    ResultLength: 8192,
    Duration:     1200 * time.Millisecond,
}).Msg("tool call complete")
```

`genai.ToolCallInfo` deliberately captures **lengths** rather than the
arguments / result body. Tool calls routinely contain user-supplied
data — logging the body unconditionally is a PII trap. Bring your own
redaction hook (`bolt.AddEventHook`) if you want the contents.

## Pre-built EventHooks

Two hooks ship with this package for the most common AI workload
concerns identified in the multi-expert review.

### Redaction (deny-list suppression)

```go
log := bolt.New(bolt.NewJSONHandler(os.Stdout)).
    AddEventHook(genai.NewRedactHook())  // uses genai.DefaultDenyKeys

// Drops any event with a "prompt", "completion", "messages", "input",
// "output", "api_key", "token", or "authorization" field.
```

Pass a custom list to override the defaults:

```go
log.AddEventHook(genai.NewRedactHook("internal_secret", "session_id"))
```

The hook **suppresses** the event entirely — it does not redact in
place (bolt's `EventHook` contract is read-only). If you need a
sanitised variant of the event to ship anyway, log it explicitly
before the sensitive event reaches the hook.

### Adaptive sampling (always keep gen_ai + errors)

```go
log := bolt.New(bolt.NewJSONHandler(os.Stdout)).
    AddEventHook(genai.NewAdaptiveSampler(100))

// Keeps every event whose level is >= ERROR, every event whose fields
// include any "gen_ai.*" key, and 1 of every 100 other events.
```

Token-stream debug logs are typically 10–100× the volume of the
structured GenAI breadcrumbs that downstream tools (Langfuse / Phoenix
/ Braintrust) actually want. `AdaptiveSampler` lets you sample the
noise without losing the breadcrumbs.

The defaults are configurable on the returned struct:

```go
s := genai.NewAdaptiveSampler(100)
s.AlwaysKeepLevel = bolt.LevelWarn   // also keep WARN and above
s.AlwaysKeepPrefixes = []string{"gen_ai.", "internal."}
log.AddEventHook(s)
```

## Compatibility

The field names match the OTel GenAI semconv as of the latest stable.
Tools that consume OTel-shaped logs (Langfuse, Arize Phoenix,
Braintrust, OTel Collector with the GenAI processor) ingest bolt logs
emitted via this package directly.

## Roadmap (this package)

- `genai.Step` for agent multi-step trajectories — deferred until the
  OTel agent semconv stabilises
- Tighter typing for `FinishReason` (currently a `string`; semconv
  defines a small enum)
