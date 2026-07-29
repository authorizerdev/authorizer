// Standalone Go module for the SDK-driven e2e-playground suite.
//
// It is a SEPARATE module (own go.mod) from the parent
// github.com/authorizerdev/authorizer repo on purpose: that keeps it out of the
// parent's `go build ./...` / CI, and — the whole point of this suite — it
// depends on the ACTUALLY-PUBLISHED authorizer-go SDK release via the module
// proxy (a real `require`, never a `replace` onto local source), so these tests
// exercise the same bytes a downstream integrator would `go get`.
module github.com/authorizerdev/authorizer/e2e-playground/sdk-tests/go

go 1.25.5

require (
	github.com/authorizerdev/authorizer-go/v2 v2.2.0-rc.4
	github.com/authorizerdev/authorizer-proto-go v0.1.0
	github.com/descope/virtualwebauthn v1.0.5
	github.com/pquerna/otp v1.5.0
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260415201107-50325440f8f2.1 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/fxamacker/cbor/v2 v2.9.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260523011958-0a33c5d7ca68 // indirect
	google.golang.org/grpc v1.81.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
