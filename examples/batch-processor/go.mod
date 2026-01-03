module github.com/felixgeelhaar/bolt/examples/batch-processor

go 1.24.6

require github.com/felixgeelhaar/bolt/v3 v3.0.0

require (
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/term v0.33.0 // indirect
)

replace github.com/felixgeelhaar/bolt/v3 => ../../
