# gRPC interceptors example

A reference for adding bolt structured logs to a gRPC server through
unary and stream interceptors. Demonstrates request correlation,
duration measurement, error capture, and per-RPC log scoping.

## Status: requires proto regeneration

This example imports generated protobuf stubs (e.g.
`pb.UserServiceServer`) that are not committed to the repo. To run
it, regenerate the stubs first:

```bash
cd examples/microservices/grpc-interceptors
protoc --go_out=. --go-grpc_out=. user_service.proto
go mod tidy
go run .
```

This requirement is the reason the project's CI explicitly skips
building this directory. The `main.go` and `user_service.proto` are
intended as a starting template, not a turnkey demo.

## What the interceptors show

- A unary interceptor that:
  - generates a correlation ID per RPC
  - logs request start with method name, peer address, deadline
  - measures duration with `time.Since` and emits a structured log on
    completion or error
  - enriches the per-RPC logger via `Logger.With()` so any handler
    code that needs to log can do so without rewriting the chain
- A stream interceptor that wraps `grpc.ServerStream` to log Send/Recv
  framing events without changing the application protocol.

## Where bolt fits

The interceptors do not introduce a custom logger interface — they
take a `*bolt.Logger` directly. This keeps the call site small and
avoids the indirection that often grows around middleware.

For an HTTP-flavoured equivalent without proto-stub setup, see
[`../http-middleware`](../http-middleware/).
