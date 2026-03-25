module github.com/felixgeelhaar/bolt/examples/rest-api

go 1.25.0

require (
	github.com/felixgeelhaar/bolt v1.2.1
	go.opentelemetry.io/otel/trace v1.42.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	go.opentelemetry.io/otel v1.42.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/term v0.41.0 // indirect
)

replace github.com/felixgeelhaar/bolt => ../../
