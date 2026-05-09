# Field types reference

Canonical reference is [pkg.go.dev][godoc]. This page is the short
in-repo lookup, sorted by category.

[godoc]: https://pkg.go.dev/github.com/felixgeelhaar/bolt

All field methods on `*Event` return `*Event` for chaining. They are
zero-allocation unless flagged otherwise. Every method validates its
key and is a no-op when the event was created at a level filtered out
by the logger.

## Strings

| Method | What | Notes |
|---|---|---|
| `Str(key, value string)` | Plain string field | Standard escape handling per RFC 8259 |
| `Stringer(key string, val fmt.Stringer)` | Value from `String()` | nil-safe — emits `null` |
| `Bytes(key string, value []byte)` | Raw bytes as a string | Allocates via `string(value)` |
| `Hex(key string, value []byte)` | Hex-encoded | |
| `Base64(key string, value []byte)` | Standard base64-encoded | |
| `RandID(key string)` | Random 8-byte ID, hex-encoded | Uses `crypto/rand` |

## Numbers

| Method | What |
|---|---|
| `Int(key string, value int)` | |
| `Int8 / Int16 / Int32 / Int64` | Width-specific |
| `Uint(key string, value uint)` | |
| `Uint8 / Uint16 / Uint32 / Uint64` | Width-specific |
| `Float64(key string, value float64)` | Round-trip via `strconv.AppendFloat('g', -1)` (matches `encoding/json`); NaN/Inf emitted as JSON strings `"NaN"` / `"+Inf"` / `"-Inf"` |
| `Counter(key string, counter *int64)` | Atomic load, emits as int |

## Booleans

| Method | What |
|---|---|
| `Bool(key string, value bool)` | |

## Time and duration

| Method | What |
|---|---|
| `Time(key string, value time.Time)` | RFC 3339 with nanosecond precision (trailing zeros trimmed) |
| `Dur(key string, value time.Duration)` | Nanoseconds as int |
| `Timestamp()` | Adds `timestamp` field with current time |

## Networks

| Method | What |
|---|---|
| `IPAddr(key string, ip net.IP)` | Dotted-decimal for IPv4, colon-hex for IPv6; nil emits `null` |

## Composites

| Method | What |
|---|---|
| `Any(key string, value interface{})` | `encoding/json` reflection — convenient, allocates |
| `Interface(key, value)` | Alias for `Any` |
| `Fields(map[string]interface{})` | Bulk add via map; iteration order non-deterministic |
| `Ints(key string, values []int)` | JSON array, zero-alloc |
| `Strs(key string, values []string)` | JSON array, zero-alloc |
| `Dict(key string, fn func(d *Event))` | Nested object built by closure |

## Diagnostics

| Method | What |
|---|---|
| `Err(err error)` | Adds `error` field with `err.Error()`; nil-safe (no field added) |
| `Stack()` | 64KB stack trace |
| `Caller()` | `file:line` of caller |
| `CallerSkip(skip int)` | `file:line` of caller plus `skip` frames |

## Terminators

| Method | What |
|---|---|
| `Msg(message string)` | Adds `message` field, ships record, recycles buffer |
| `Send()` | Equivalent to `Msg("")` |
| `Printf(format string, args ...interface{})` | `fmt.Sprintf` then `Msg` |

Forgetting a terminator silently drops the event — the buffer is
reclaimed by the pool only when GC walks it. A `go vet` analyser is
on the roadmap.

## Hot-path discipline

If an absolute zero-allocation guarantee matters:

- Stick to typed methods (`Str`, `Int`, `Bool`, `Float64`, `Time`,
  `Dur`, `Ints`, `Strs`, `IPAddr`, `Dict`).
- Avoid `Any` / `Fields` / `Bytes` / `Stack` / `Caller*` (each
  allocates).
- The encoding budget is verified by `BenchmarkZeroAllocation` in CI;
  the [benchstat workflow](../../.github/workflows/pr-checks.yml)
  fails the PR on statistically significant regressions.
