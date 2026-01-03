module github.com/felixgeelhaar/bolt/examples/microservices/grpc-interceptors

go 1.23

require (
	github.com/felixgeelhaar/bolt/v3 v3.0.0
	github.com/google/uuid v1.6.0
	google.golang.org/grpc v1.70.0
	google.golang.org/protobuf v1.36.0
)

require (
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
)

// Local development - replace with actual module path in production
replace github.com/felixgeelhaar/bolt/v3 => ../../..