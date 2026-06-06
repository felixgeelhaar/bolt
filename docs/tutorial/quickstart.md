# Five-minute first log

> **Goal**: by the end of this tutorial you'll have an HTTP handler
> emitting structured JSON logs with correlation IDs and OTel trace
> attachment, running locally.

This is a learning-oriented walkthrough. If you already know what
you're doing and need a recipe, jump to the [how-to](../how-to/) or
the [reference](../reference/) instead.

## Prerequisites

- Go 1.24 or newer
- A terminal

## Step 1: install bolt

```bash
mkdir bolt-tutorial && cd bolt-tutorial
go mod init example.com/bolt-tutorial
go get go.klarlabs.de/bolt
```

## Step 2: a one-line first log

Save this as `main.go`:

```go
package main

import (
    "os"

    "go.klarlabs.de/bolt"
)

func main() {
    log := bolt.New(bolt.NewJSONHandler(os.Stdout))
    log.Info().Str("hello", "world").Msg("first log")
}
```

Run it:

```bash
go run .
```

You should see:

```json
{"level":"info","hello":"world","message":"first log"}
```

That's the chained API. Every call between `Info()` and `Msg(...)`
adds a field; `Msg(...)` is the terminator that ships the record. If
you forget `Msg`, the record is silently dropped — a `go vet`
analyser is on the roadmap to catch that.

## Step 3: a sub-logger with bound context

Most production services want every log line to carry the same
service-wide fields (service name, version, deploy environment).
Build a sub-logger once and reuse it:

```go
log := bolt.New(bolt.NewJSONHandler(os.Stdout))

apiLog := log.With().
    Str("service", "api").
    Str("version", "v1.2.3").
    Logger()

apiLog.Info().Str("path", "/").Msg("request")
apiLog.Info().Str("path", "/users").Msg("request")
```

Output:

```json
{"level":"info","service":"api","version":"v1.2.3","path":"/","message":"request"}
{"level":"info","service":"api","version":"v1.2.3","path":"/users","message":"request"}
```

## Step 4: an HTTP handler with correlation

Now a real handler. Save this as `main.go`:

```go
package main

import (
    "fmt"
    "net/http"
    "os"
    "time"

    "go.klarlabs.de/bolt"
    "github.com/google/uuid"
)

func main() {
    log := bolt.New(bolt.NewJSONHandler(os.Stdout)).
        With().Str("service", "tutorial").Logger()

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

        fmt.Fprintln(w, "hello")

        log.Info().
            Str("req_id", rid).
            Int("status", 200).
            Dur("duration", time.Since(start)).
            Msg("done")
    })

    log.Info().Str("addr", ":8080").Msg("listening")
    _ = http.ListenAndServe(":8080", nil)
}
```

```bash
go get github.com/google/uuid
go run .
```

In another shell:

```bash
curl -s localhost:8080/
```

You'll see two log lines per request — start and done — sharing the
same `req_id` so you can grep them together later.

## Step 5: drop into `slog` (optional)

If your project already uses `log/slog`, you don't have to change
your call sites — just swap the handler:

```go
import "log/slog"

slog.SetDefault(slog.New(bolt.NewSlogHandler(os.Stdout, nil)))
slog.Info("ok", "user", "alice", "status", 200)
```

The bolt slog handler passes the standard
`testing/slogtest.TestHandler` conformance suite so anything that
works with `slog.NewJSONHandler` works with `bolt.NewSlogHandler`.

## What next

- [How-to: migrate an existing zerolog/zap codebase](../how-to/) —
  mapping tables and worked examples.
- [Reference: every field type](../reference/field-types.md) — the
  full list of zero-alloc builders.
- [Explanation: why zero allocations](../explanation/zero-alloc-rationale.md)
  — the design choice underneath everything bolt does.
- The runnable [`examples/observability/opentelemetry/`](../../examples/observability/opentelemetry/)
  example wires bolt to OpenTelemetry tracing + Prometheus metrics
  for a more realistic deployment shape.
