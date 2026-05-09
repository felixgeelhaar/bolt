module github.com/felixgeelhaar/bolt/benchmarks

go 1.25.0

require (
	github.com/felixgeelhaar/bolt v0.0.0-00010101000000-000000000000
	github.com/rs/zerolog v1.34.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	go.opentelemetry.io/otel v1.42.0 // indirect
	go.opentelemetry.io/otel/trace v1.42.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/term v0.41.0 // indirect
)

// Use local parent module for development
replace github.com/felixgeelhaar/bolt => ../
