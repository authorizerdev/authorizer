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
			return metadata.Pairs("x-authorizer-url", parsers.GetHostFromRequest(r))
		}),
		// Forward the custom admin-secret header to the gRPC layer. The default
		// matcher only forwards permanent headers (Authorization, Cookie), but
		// admin header-auth also accepts x-authorizer-admin-secret; without this
		// REST callers could only authenticate via the admin cookie. All other
		// headers fall through to the default matcher so existing behaviour is
		// unchanged.
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			if strings.EqualFold(key, "x-authorizer-admin-secret") {
				return key, true
			}
			return runtime.DefaultHeaderMatcher(key)
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
	// both). Both dial the same in-process gRPC server over the bufconn. The
	// admin registrar (RegisterAuthorizerAdminServiceHandler) is added in
	// Phase 1 once AuthorizerAdminService has its first HTTP-annotated RPC —
	// grpc-gateway only generates the registrar for services with annotations.
	return authorizerv1.RegisterAuthorizerServiceHandler(ctx, mux, conn)
}
