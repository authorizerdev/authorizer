// Package gateway translates REST (`/v1/*`) calls into in-process gRPC
// method invocations using grpc-gateway. The resulting http.Handler is
// mounted into the existing Gin router so middleware (CORS, security
// headers, rate limit, logging) is shared.
//
// We deliberately use the *in-process* dialer style: the gateway dials the
// running grpc.Server via bufconn rather than making a real network hop.
// This avoids the latency and TLS-cert plumbing of a loopback gRPC call,
// and removes the need to keep two ports in sync.
package gateway

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/encoding/protojson"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/parsers"
)

// bufconn size; large enough that in-process gateway calls never block.
const bufSize = 1 << 20

// Handler builds an http.Handler that translates `/v1/*` REST calls into
// gRPC calls against the supplied in-process *grpc.Server. Returns the
// handler and a cleanup function the caller invokes at shutdown.
func Handler(ctx context.Context, grpcSrv *grpc.Server) (http.Handler, func(), error) {
	lis := bufconn.Listen(bufSize)

	// Serve gRPC over the bufconn in a goroutine; the existing TCP
	// listener (started by grpcsrv.Server.Run) is the public entry point —
	// this listener only carries in-process gateway traffic.
	go func() {
		_ = grpcSrv.Serve(lis)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		_ = lis.Close()
		return nil, nil, err
	}

	mux := runtime.NewServeMux(
		// Forward the original request's authorizer host URL to the gRPC
		// layer. The in-process bufconn call carries `:authority=bufconn`,
		// so without this the service layer would resolve the host as
		// "http://bufconn" and JWT issuer validation would reject every
		// token minted via the HTTP surface. parsers.GetHostFromRequest is
		// the same spoof-hardened resolution the gin path uses;
		// transport.MetaFromGRPC reads `x-authorizer-url` first.
		runtime.WithMetadata(func(_ context.Context, r *http.Request) metadata.MD {
			// x-authorizer-transport=rest lets transport.MetaFromGRPC tag the
			// request protocol as REST (vs a direct gRPC call) for audit logs
			// and the api-operations metric.
			return metadata.Pairs(
				"x-authorizer-url", parsers.GetHostFromRequest(r),
				"x-authorizer-transport", constants.ProtocolREST,
			)
		}),
		// Forward the custom admin-secret header to the gRPC layer. The default
		// matcher only forwards permanent headers (Authorization, Cookie), but
		// admin header-auth also accepts x-authorizer-admin-secret; without this
		// REST callers could only authenticate via the admin cookie. All other
		// headers fall through to the default matcher so existing behaviour is
		// unchanged.
		incomingHeaderMatcherOption(),
		// Promote the service layer's session/MFA cookies to real Set-Cookie
		// response headers. Handlers emit them via transport.ApplyToGRPC ->
		// grpc.SendHeader as `set-cookie` server metadata; grpc-gateway's
		// default outgoing matcher would rename that to `Grpc-Metadata-Set-Cookie`
		// (which browsers ignore), so login/signup/session/verify_* would set no
		// cookie over REST. Returning false for every other key also stops
		// internal gRPC metadata leaking as Grpc-Metadata-* response headers.
		runtime.WithOutgoingHeaderMatcher(func(key string) (string, bool) {
			if strings.EqualFold(key, "set-cookie") {
				return "Set-Cookie", true
			}
			return "", false
		}),
		// Consistent error envelope across the REST surface (see errorHandler).
		runtime.WithErrorHandler(errorHandler),
		// Preserve true HTTP routing statuses (e.g. 405 for a method mismatch
		// such as GET on a POST-only endpoint) instead of grpc-gateway's
		// default 405->501 remap. See routingErrorHandler.
		runtime.WithRoutingErrorHandler(routingErrorHandler),
		// Use snake_case proto field names (UseProtoNames=true) over the
		// camelCase default — keeps payloads aligned with the existing
		// GraphQL surface. DiscardUnknown tolerates older clients sending
		// fields that have since been removed.
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   true,
				EmitUnpopulated: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)
	if err := registerAll(ctx, mux, conn); err != nil {
		_ = conn.Close()
		_ = lis.Close()
		return nil, nil, err
	}

	cleanup := func() {
		_ = conn.Close()
		_ = lis.Close()
	}
	return mux, cleanup, nil
}

// errorHandler renders REST errors in a single, stable JSON envelope across
// every /v1 endpoint:
//
//	{"code": "invalid_argument", "message": "..."}
//
// instead of grpc-gateway's default `{"code": <number>, "message": ..., "details": []}`.
// The numeric-to-HTTP-status mapping still comes from runtime.HTTPStatusFromCode
// so e.g. InvalidArgument -> 400, Unauthenticated -> 401, PermissionDenied ->
// 403, NotFound -> 404, Internal -> 500. The `code` token is snake_case to
// match the rest of the API's naming.
func errorHandler(_ context.Context, _ *runtime.ServeMux, _ runtime.Marshaler, w http.ResponseWriter, _ *http.Request, err error) {
	st := status.Convert(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(runtime.HTTPStatusFromCode(st.Code()))
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    codeName(st.Code()),
		"message": st.Message(),
	})
}

// routingErrorHandler renders gateway routing failures (no matching route,
// or a method mismatch) using the same JSON envelope as errorHandler while
// keeping the correct HTTP status. grpc-gateway's default remaps 405 ->
// codes.Unimplemented -> 501, which is misleading for clients; here a GET on a
// POST-only path stays a 405.
func routingErrorHandler(_ context.Context, _ *runtime.ServeMux, _ runtime.Marshaler, w http.ResponseWriter, _ *http.Request, httpStatus int) {
	var code string
	switch httpStatus {
	case http.StatusMethodNotAllowed:
		code = "method_not_allowed"
	case http.StatusNotFound:
		code = "not_found"
	case http.StatusBadRequest:
		code = "invalid_argument"
	default:
		code = "internal"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    code,
		"message": http.StatusText(httpStatus),
	})
}

// codeName maps a gRPC status code onto a stable snake_case token for the REST
// error envelope. Unmapped codes fall back to "internal".
func codeName(c codes.Code) string {
	switch c {
	case codes.OK:
		return "ok"
	case codes.InvalidArgument:
		return "invalid_argument"
	case codes.Unauthenticated:
		return "unauthenticated"
	case codes.PermissionDenied:
		return "permission_denied"
	case codes.NotFound:
		return "not_found"
	case codes.AlreadyExists:
		return "already_exists"
	case codes.FailedPrecondition:
		return "failed_precondition"
	case codes.Unavailable:
		return "unavailable"
	case codes.Unimplemented:
		return "unimplemented"
	case codes.DeadlineExceeded:
		return "deadline_exceeded"
	case codes.ResourceExhausted:
		return "resource_exhausted"
	default:
		return "internal"
	}
}

func registerAll(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	// Public + admin surfaces share this gateway mux (one REST port serves
	// both). Both dial the same in-process gRPC server over the bufconn.
	if err := authorizerv1.RegisterAuthorizerServiceHandler(ctx, mux, conn); err != nil {
		return err
	}
	return authorizerv1.RegisterAuthorizerAdminServiceHandler(ctx, mux, conn)
}

// refusedForwardingHeaders is the set of request headers a client must never be
// able to put into gRPC metadata, keyed by textproto.CanonicalMIMEHeaderKey.
//
// Three of these are load-bearing today:
//
//   - X-Forwarded-For and X-Real-Ip decide the client IP used for rate
//     limiting, the admin-secret lockout and audit logs
//     (transport.clientIPFromGRPC reads them out of metadata).
//   - X-Authorizer-Url decides the host of emailed password-reset and
//     magic-link URLs and the `iss` claim of the tokens minted from them
//     (transport.hostURLFromGRPC).
//
// X-Forwarded-Host is NOT one of them: hostURLFromGRPC reads x-authorizer-url,
// and nothing in this repo reads x-forwarded-host. It is listed because
// runtime.AnnotateContext writes it from the incoming header, making it the one
// client-chosen value that does reach metadata — so the next reader looking for
// "the request host" would find it sitting there.
//
// The rest are the standard vendor and proxy spellings of the same two claims.
// Nothing in this repo reads them yet — gin's TrustedPlatform is unset, so even
// gin's own ClientIP ignores them. They are listed because the cost of the list
// is one map entry and the cost of omitting one is a silent spoof the next time
// a handler, an interceptor or a logging field reaches for "the client IP" or
// "the request host". A header a client can spell must not reach metadata at
// all, so the decision cannot be made wrongly later.
//
// Add a key here rather than at the call site: incomingHeaderMatcher and its
// test both read this map, so a new entry is exercised the moment it lands.
var refusedForwardingHeaders = map[string]struct{}{
	// Client-IP claims.
	"X-Forwarded-For":          {},
	"X-Real-Ip":                {},
	"X-Client-Ip":              {},
	"X-Cluster-Client-Ip":      {},
	"X-Originating-Ip":         {},
	"X-Original-Forwarded-For": {},
	"X-Remote-Ip":              {},
	"X-Remote-Addr":            {},
	"True-Client-Ip":           {},
	"Cf-Connecting-Ip":         {},
	"Cf-Connecting-Ipv6":       {},
	"Fastly-Client-Ip":         {},
	"Fly-Client-Ip":            {},
	"X-Azure-Clientip":         {},
	"X-Azure-Socketip":         {},
	"X-Appengine-User-Ip":      {},
	"X-Envoy-External-Address": {},

	// Host, scheme and port claims. These decide absolute URLs the server
	// emails or returns, so they poison links rather than labels.
	"X-Forwarded-Host":   {},
	"X-Authorizer-Url":   {},
	"X-Forwarded-Proto":  {},
	"X-Forwarded-Scheme": {},
	"X-Forwarded-Port":   {},
	"X-Forwarded-Server": {},
	"X-Original-Host":    {},
	"X-Original-Url":     {},
	"X-Rewrite-Url":      {},
	"X-Host":             {},

	// RFC 7239 and its pre-standard spelling carry both claims at once.
	"Forwarded":   {},
	"X-Forwarded": {},
}

// incomingHeaderMatcherOption builds the gateway's inbound header policy.
//
// Extracted as a named function so the test suite drives the REAL policy rather
// than a copy that can drift — the drift is the vulnerability here.
func incomingHeaderMatcherOption() runtime.ServeMuxOption {
	return runtime.WithIncomingHeaderMatcher(incomingHeaderMatcher(runtime.DefaultHeaderMatcher))
}

// incomingHeaderMatcher is the policy itself, over a fallback matcher.
//
// The fallback is a parameter for one reason. v2.33.0's DefaultHeaderMatcher
// already refuses every non-permanent header, and none of the keys below is a
// permanent header, so against that fallback the refusal list changes no byte of
// metadata — a test driving only it passes even with the list emptied.
// runtime.LegacyHeaderMatcher is the behaviour the list exists to survive and it
// still ships, so the test drives BOTH. Keep it that way: a policy whose test
// passes when the policy is deleted is not tested.
func incomingHeaderMatcher(fallback func(string) (string, bool)) func(string) (string, bool) {
	return func(key string) (string, bool) {

		if strings.EqualFold(key, "x-authorizer-admin-secret") {
			return key, true
		}
		// Refuse client-supplied forwarding headers outright. The full set is
		// refusedForwardingHeaders, declared above this function.
		//
		// A matcher that strips a "Grpc-Metadata-" prefix and forwards
		// whatever follows lets a REST client sending
		// `Grpc-Metadata-X-Forwarded-For: 1.2.3.4` get a real
		// `x-forwarded-for` metadata entry — placed BEFORE the authoritative
		// chain AnnotateContext appends afterwards. That smuggled entry
		// reinstated the exact spoof GHSA-93hc-xq3w-xw87 is about: the
		// admin-secret lockout bucketed on an attacker-chosen value again,
		// over REST, unauthenticated.
		//
		// That was DefaultHeaderMatcher's behaviour when this filter was
		// written. v2.33.0 moved it to runtime.LegacyHeaderMatcher and made the
		// default refuse every non-permanent header, so the vector is closed
		// upstream as well. This filter is now the lock that survives the
		// default changing back — which is why its test drives the legacy
		// matcher too, not only the one that ships.
		//
		// The raw `X-Forwarded-For` header is already safe (AnnotateContext
		// handles it itself and DefaultHeaderMatcher does not forward it,
		// since it is not a permanent header) — this closes the prefixed
		// spelling, which is the only way a client can reach the key.
		//
		// Belt and braces with transport.clientIPFromGRPC, which reads the
		// LAST value rather than the first for the same reason. Either alone
		// would close it; a header this load-bearing gets both.
		//
		// X-Authorizer-Url is here for the identical reason, one key over. It
		// decides the host of emailed password-reset and magic-link URLs and the
		// `iss` claim of the resulting tokens. The gin path pins it to --url
		// inside parsers.GetHostFromRequest; the gRPC path did not consult --url
		// at all, so the prefixed spelling reinstated host poisoning over REST.
		// transport.hostURLFromGRPC now prefers --url — this is its second lock.
		//
		// This does NOT affect the gateway's own x-authorizer-url value: that is
		// injected via runtime.WithMetadata (above), a separate option which
		// operates on the already-parsed request and never passes through this
		// matcher.
		if _, refused := refusedForwardingHeaders[textproto.CanonicalMIMEHeaderKey(
			strings.TrimPrefix(
				textproto.CanonicalMIMEHeaderKey(key), runtime.MetadataHeaderPrefix),
		)]; refused {
			return "", false
		}
		return fallback(key)
	}
}
