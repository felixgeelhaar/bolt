module github.com/felixgeelhaar/bolt/examples/microservices/grpc-interceptors

go 1.24.0

require (
	github.com/felixgeelhaar/bolt v1.2.1
	github.com/google/uuid v1.6.0
	google.golang.org/grpc v1.70.0
	google.golang.org/protobuf v1.36.6
)

require (
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241202173237-19429a94021a // indirect
)

// Local development - replace with actual module path in production
replace github.com/felixgeelhaar/bolt => ../../..
