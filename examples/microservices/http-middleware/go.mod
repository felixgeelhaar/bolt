module github.com/felixgeelhaar/bolt/examples/microservices/http-middleware

go 1.23.0

toolchain go1.24.6

require (
	github.com/felixgeelhaar/bolt v1.2.1
	github.com/google/uuid v1.6.0
)

require (
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/term v0.33.0 // indirect
)

// Local development - replace with actual module path in production
replace github.com/felixgeelhaar/bolt => ../../..
