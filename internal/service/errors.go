package service

// Transport-agnostic typed errors. The service layer must not import gRPC or
// HTTP packages, yet callers across transports (gRPC, REST gateway, GraphQL)
// need to distinguish a client mistake (400) from an auth failure (401/403)
// from an internal fault (500). We express that intent with an ErrorKind and
// let each transport translate it:
//   - gRPC: interceptors.ErrorMap maps Kind -> codes.Code (and grpc-gateway
//     then maps the code -> HTTP status for the REST surface).
//   - GraphQL: http_handlers.kindToGraphQLCode maps Kind -> extensions.code
//     on the GraphQL error, alongside the unchanged message text, so clients
//     can switch on a stable code instead of matching message strings.
//
// Only client-facing (4xx-class) errors need to be wrapped with these
// constructors. Anything returned bare (storage failures, token-creation
// failures, etc.) is treated as internal by the mapper, which is the correct
// default.

// ErrorKind classifies a service error independently of any transport.
type ErrorKind int

const (
	// KindInternal is an unexpected server-side failure. Default for any
	// error not explicitly classified. Maps to gRPC Internal / HTTP 500.
	KindInternal ErrorKind = iota
	// KindInvalidArgument is a malformed or semantically invalid request.
	// Maps to gRPC InvalidArgument / HTTP 400.
	KindInvalidArgument
	// KindUnauthenticated is a missing or invalid credential/session.
	// Maps to gRPC Unauthenticated / HTTP 401.
	KindUnauthenticated
	// KindPermissionDenied is an authenticated caller lacking the required
	// permission. Maps to gRPC PermissionDenied / HTTP 403.
	KindPermissionDenied
	// KindNotFound is a referenced resource that does not exist.
	// Maps to gRPC NotFound / HTTP 404.
	KindNotFound
	// KindFailedPrecondition is a request that is well-formed but not
	// permitted in the server's current state (e.g. signup disabled).
	// Maps to gRPC FailedPrecondition / HTTP 400.
	KindFailedPrecondition
	// KindTooManyRequests is a request rejected because the caller exceeded
	// a rate/attempt limit (e.g. MFA verification locked after repeated
	// failures). Maps to gRPC ResourceExhausted / HTTP 429.
	KindTooManyRequests
)

// Error is a typed service error carrying a transport-neutral Kind alongside a
// human-readable message. It implements error and unwraps to any underlying
// cause so errors.Is/errors.As keep working.
type Error struct {
	Kind ErrorKind
	msg  string
	err  error
}

// Error returns the human-readable message. When constructed from an
// underlying error without an explicit message, it falls back to that error's
// text so existing message-based assertions keep passing.
func (e *Error) Error() string {
	if e.msg != "" {
		return e.msg
	}
	if e.err != nil {
		return e.err.Error()
	}
	return "service error"
}

// Unwrap exposes the underlying cause for errors.Is / errors.As.
func (e *Error) Unwrap() error { return e.err }

// InvalidArgument reports a malformed or semantically invalid request.
func InvalidArgument(msg string) error {
	return &Error{Kind: KindInvalidArgument, msg: msg}
}

// Unauthenticated reports a missing or invalid credential/session.
func Unauthenticated(msg string) error {
	return &Error{Kind: KindUnauthenticated, msg: msg}
}

// PermissionDenied reports an authenticated caller lacking a required permission.
func PermissionDenied(msg string) error {
	return &Error{Kind: KindPermissionDenied, msg: msg}
}

// NotFound reports a referenced resource that does not exist.
func NotFound(msg string) error {
	return &Error{Kind: KindNotFound, msg: msg}
}

// FailedPrecondition reports a well-formed request disallowed by current state.
func FailedPrecondition(msg string) error {
	return &Error{Kind: KindFailedPrecondition, msg: msg}
}

// TooManyRequests reports a request rejected for exceeding a rate/attempt limit.
func TooManyRequests(msg string) error {
	return &Error{Kind: KindTooManyRequests, msg: msg}
}
