// Package transport bridges between gRPC's incoming metadata / outgoing
// trailers and the service layer's transport-agnostic RequestMetadata /
// ResponseSideEffects.
//
// gRPC has no native cookie concept; cookies in ResponseSideEffects are
// serialised to a `Set-Cookie` trailer, which grpc-gateway then promotes
// into actual `Set-Cookie` response headers when the call comes in via REST.
// Pure-gRPC clients (server-to-server) typically don't need cookies and
// silently ignore them.
package transport

import (
	"context"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/authorizerdev/authorizer/internal/service"
)

// MetaFromGRPC builds a RequestMetadata from a gRPC context. Headers
// populated by grpc-gateway (`grpcgateway-*` prefix) are honored so the
// service sees the same host/IP/UA whether the call came via gRPC directly
// or via REST through the gateway.
func MetaFromGRPC(ctx context.Context) service.RequestMetadata {
	md, _ := metadata.FromIncomingContext(ctx)
	meta := service.RequestMetadata{
		HostURL:             firstHeader(md, "x-authorizer-url", "grpcgateway-x-authorizer-url"),
		IPAddress:           firstHeader(md, "x-forwarded-for", "grpcgateway-x-forwarded-for", "x-real-ip"),
		UserAgent:           firstHeader(md, "grpcgateway-user-agent", "user-agent"),
		AuthorizationHeader: firstHeader(md, "authorization", "grpcgateway-authorization"),
	}
	// Default the host URL when no header was set (pure-gRPC caller, no
	// proxy headers). The :authority pseudo-header is the gRPC equivalent
	// of Host; use it as a fallback.
	if meta.HostURL == "" {
		if authority := firstHeader(md, ":authority"); authority != "" {
			meta.HostURL = "http://" + authority
		}
	}
	return meta
}

// ApplyToGRPC writes the response side-effects to the outgoing gRPC stream:
// cookies become Set-Cookie metadata trailers. A nil receiver is a no-op.
func ApplyToGRPC(ctx context.Context, side *service.ResponseSideEffects) error {
	if side == nil || len(side.Cookies) == 0 {
		return nil
	}
	values := make([]string, 0, len(side.Cookies))
	for _, c := range side.Cookies {
		if c == nil {
			continue
		}
		values = append(values, c.String())
	}
	if len(values) == 0 {
		return nil
	}
	return grpc.SendHeader(ctx, metadata.Pairs(http.CanonicalHeaderKey("Set-Cookie"), values[0])) //nolint:staticcheck // only one cookie surfaces; multi-cookie comes with the gateway-aware wiring
}

func firstHeader(md metadata.MD, keys ...string) string {
	for _, k := range keys {
		if vs := md.Get(k); len(vs) > 0 {
			return vs[0]
		}
	}
	return ""
}
