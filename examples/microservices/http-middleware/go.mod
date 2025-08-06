module github.com/felixgeelhaar/bolt/examples/microservices/http-middleware

go 1.23

require (
	github.com/felixgeelhaar/bolt/v2 v2.0.0
	github.com/google/uuid v1.6.0
)

// Local development - replace with actual module path in production
replace github.com/felixgeelhaar/bolt/v2 => ../../..