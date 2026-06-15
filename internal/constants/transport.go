package constants

// Transport protocol identifiers. Recorded on RequestMetadata and surfaced in
// audit logs (audit.Event.Protocol → audit metadata) and the
// authorizer_api_operations_total metric so each performed operation is
// attributable to the protocol it came in on. Low-cardinality; safe as metric
// label values.
const (
	// ProtocolGraphQL marks an operation served via the GraphQL endpoint.
	ProtocolGraphQL = "graphql"
	// ProtocolGRPC marks an operation served via a direct gRPC call.
	ProtocolGRPC = "grpc"
	// ProtocolREST marks an operation served via the REST (grpc-gateway) surface.
	ProtocolREST = "rest"
)
