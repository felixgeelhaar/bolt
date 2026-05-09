module github.com/felixgeelhaar/bolt/examples/microservices/grpc-interceptors

go 1.25.0

require (
	github.com/felixgeelhaar/bolt v1.2.1
	github.com/google/uuid v1.6.0
	google.golang.org/grpc v1.81.0
	google.golang.org/protobuf v1.36.11
)

require (
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
)

// Local development - replace with actual module path in production
replace github.com/felixgeelhaar/bolt => ../../..
