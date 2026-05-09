# OpenTelemetry observability example

The flagship integration example: an HTTP service wired to OpenTelemetry
tracing (OTLP exporter), Prometheus metrics, and bolt structured logs
correlated by trace and span ID.

## Run it

```bash
cd examples/observability/opentelemetry
go mod tidy
go run .
```

The server listens on `:8080` and exports traces to `localhost:4318`
via OTLP/HTTP. To see traces, run a collector that accepts that
endpoint (Jaeger, Tempo, OTel Collector, etc.). Set `OTLP_ENDPOINT`
to override.

## What it shows

- `bolt.NewJSONHandler` writing structured logs to stdout
- `Logger.With()` to bind service-level fields once (`service`, `version`)
- Per-request correlation via the existing trace and span IDs from the
  active OTel context, attached as `trace_id` and `span_id` log fields
- Counter / histogram / gauge metrics on the same handler chain
- Honest error path: an `/error` endpoint emits both a recorded error
  span and an `Error` log line so log+trace correlation can be seen
  end-to-end

## Endpoints

| Path | What |
|---|---|
| `/` | Records a span, sleeps a bit, logs request lifecycle |
| `/users` | Calls a child span "db_query_users" with a 10% simulated failure rate |
| `/error` | Always returns 500 with a recorded span error and an Error log |
| `/health` | Liveness probe |
| `/metrics` | Prometheus scrape endpoint |

## Where bolt fits

bolt is the log layer; OTel is the trace + metric layer. The example
treats the two as complementary — logs carry the same correlation IDs
the traces do, so a log shipper like Loki + Grafana can jump from a log
line to the trace it belongs to (or back) without bespoke glue.

For the API mapping if you're migrating from a different logger, see
[`docs/migrate-from-zerolog.md`](../../../docs/migrate-from-zerolog.md),
[`docs/migrate-from-zap.md`](../../../docs/migrate-from-zap.md), or
[`docs/migrate-from-slog.md`](../../../docs/migrate-from-slog.md).
