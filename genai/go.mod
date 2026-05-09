module github.com/felixgeelhaar/bolt/genai

go 1.25.0

require github.com/felixgeelhaar/bolt v1.3.0

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/term v0.43.0 // indirect
)

// Local development — pin to the in-tree bolt module. CI consumers
// override this via `go work` or by removing the directive in their
// own checkouts.
replace github.com/felixgeelhaar/bolt => ../
