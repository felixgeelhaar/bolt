module github.com/felixgeelhaar/bolt/examples/microservices/http-middleware

go 1.23.0

toolchain go1.24.6

require (
	github.com/felixgeelhaar/bolt v2.0.0+incompatible
	github.com/google/uuid v1.6.0
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/exp v0.0.0-20250711185948-6ae5c78190dc // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/term v0.33.0 // indirect
)

// Local development - replace with actual module path in production
replace github.com/felixgeelhaar/bolt => ../../..
