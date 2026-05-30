// Package transport bridges between gRPC's incoming metadata / outgoing
// trailers and the service layer's transport-agnostic RequestMetadata /
// ResponseSideEffects.
//
// gRPC has no native cookie concept; cookies in ResponseSideEffects are
// serialised to `Set-Cookie` metadata entries. grpc-gateway promotes those
// into real `Set-Cookie` response headers when the call came in via REST.
// Pure-gRPC clients can read them via the response trailers or ignore them.
package transport

import (
	"context"
	"net/http"
	"strings"

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
		Cookies:             cookiesFromMetadata(md),
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

// ApplyToGRPC writes the response side-effects to the outgoing gRPC stream.
// Every cookie becomes its own `Set-Cookie` metadata entry — preserving
// multi-cookie responses (e.g. host-scoped + domain-scoped session pair).
// grpc-gateway promotes the metadata back to real `Set-Cookie` HTTP headers.
// A nil receiver is a no-op.
func ApplyToGRPC(ctx context.Context, side *service.ResponseSideEffects) error {
	if side == nil || len(side.Cookies) == 0 {
		return nil
	}
	// grpc-gateway honours the per-RPC `Set-Cookie` metadata when prefixed
	// `Grpc-Metadata-Set-Cookie` or under the canonical header. Use
	// metadata.Pairs equivalents: same key, repeated values.
	header := http.CanonicalHeaderKey("Set-Cookie")
	md := metadata.MD{}
	for _, c := range side.Cookies {
		if c == nil {
			continue
		}
		md.Append(header, c.String())
	}
	if len(md) == 0 {
		return nil
	}
	return grpc.SendHeader(ctx, md)
}

func firstHeader(md metadata.MD, keys ...string) string {
	for _, k := range keys {
		if vs := md.Get(k); len(vs) > 0 {
			return vs[0]
		}
	}
	return ""
}

// cookiesFromMetadata parses Cookie header(s) supplied via gRPC metadata.
// grpc-gateway forwards browser cookies as the `grpcgateway-cookie` key;
// pure-gRPC clients can set `cookie` directly. Multiple Cookie headers are
// concatenated (semicolon-separated per RFC 6265).
func cookiesFromMetadata(md metadata.MD) []*http.Cookie {
	var raw []string
	raw = append(raw, md.Get("grpcgateway-cookie")...)
	raw = append(raw, md.Get("cookie")...)
	if len(raw) == 0 {
		return nil
	}
	// http.Request.Cookies parses the Cookie header for us. Synthesize a
	// minimal request rather than re-implementing the cookie grammar.
	req := &http.Request{Header: http.Header{}}
	for _, line := range raw {
		// One header may contain multiple cookies separated by "; ".
		// http.Header.Add preserves the line; cookies are parsed downstream.
		req.Header.Add("Cookie", strings.TrimSpace(line))
	}
	return req.Cookies()
}
