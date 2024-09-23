module github.com/sqrtofpisquared/avalanche/clients/testClient

go 1.23.1

replace github.com/sqrtofpisquared/avalanche/avalanchecore => ../../avalanchecore

require github.com/sqrtofpisquared/avalanche/avalanchecore v0.0.0-00010101000000-000000000000

require (
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/net v0.29.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)
