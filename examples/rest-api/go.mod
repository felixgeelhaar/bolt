module github.com/felixgeelhaar/bolt/examples/rest-api

go 1.24.6

require (
	github.com/felixgeelhaar/bolt/v3 v3.0.0
	go.opentelemetry.io/otel/trace v1.38.0
)

require (
	go.opentelemetry.io/otel v1.38.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/term v0.33.0 // indirect
)

replace github.com/felixgeelhaar/bolt/v3 => ../../
