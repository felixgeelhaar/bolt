# Bolt Examples

Each subdirectory under `examples/` is a standalone Go module
(`go.mod` + `replace` directive pointing at the repo root) so you can
clone, `cd` into one, and `go run .` without polluting the bolt module
itself. CI builds every example listed below on every PR.

## Currently shipped

| Path | What it shows |
|---|---|
| [`rest-api/`](./rest-api/) | Plain HTTP service with structured access logs and request IDs |
| [`grpc-service/`](./grpc-service/) | gRPC server with structured request/response logging |
| [`microservices/http-middleware/`](./microservices/http-middleware/) | Middleware-style HTTP logging with correlation IDs |
| [`microservices/grpc-interceptors/`](./microservices/grpc-interceptors/) | gRPC unary/stream interceptors. **Note:** requires running `protoc` to regenerate stubs; not built in CI |
| [`observability/opentelemetry/`](./observability/opentelemetry/) | OTel tracing + Prometheus metrics + bolt log correlation. The flagship integration example |
| [`batch-processor/`](./batch-processor/) | Worker-pool / fan-out batch processing with bolt sampling |

Each one contains a `main.go` and (where relevant) supporting files.
Each directory is intentionally minimal — pull what you need, leave
the rest.

### Run any of them

```bash
git clone https://github.com/felixgeelhaar/bolt.git
cd bolt/examples/rest-api
go mod tidy
go run .
```

## Patterns the examples cover

- **Zero-allocation hot paths** — verified by `BenchmarkZeroAllocation`
  in the root module
- **Structured field types** — `Str`, `Int`, `Float64`, `Bool`, `Time`,
  `Dur`, `Bytes`, `Stringer`, `Dict`, `Ints`, `Strs`, `IPAddr`
- **OTel trace/span injection** — `Logger.Ctx(ctx)` automatically
  surfaces `trace_id` / `span_id`
- **slog interop** — `bolt.NewSlogHandler` plugs into a stdlib
  `*slog.Logger` (see `docs/migrate-from-slog.md`)
- **Hooks and sampling** — `bolt.NewSampleHook(N)` and the `Hook`
  interface

## Roadmap (not yet shipped)

The earlier version of this README listed dozens of "examples" that
did not exist. The full list is now tracked in `ROADMAP.md`. Examples
under consideration include service-mesh integration patterns, a
Kubernetes deployment walkthrough with the Diataxis docs site,
PII-redaction hook patterns once the Hook v2 interface lands, a
bolt/genai sub-module showing OTel GenAI semconv, and a benchstat-based
performance comparison harness. None of these directories exist today;
they will land alongside their respective spec changes and will appear
in this table once they do.

If you need an example that isn't here yet, **open an issue describing
the use case** — concrete requests beat speculative scaffolding for
prioritisation.
