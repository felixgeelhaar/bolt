module github.com/felixgeelhaar/bolt/benchmarks

go 1.23.0

require (
	github.com/felixgeelhaar/bolt/v3 v3.0.0
	github.com/rs/zerolog v1.34.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/term v0.33.0 // indirect
)

// Use local parent module for development
replace github.com/felixgeelhaar/bolt/v3 => ../
