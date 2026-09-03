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
	"net/url"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// MetaFromGRPC builds a RequestMetadata from a gRPC context. Headers
// populated by grpc-gateway (`grpcgateway-*` prefix) are honored so the
// service sees the same host/IP/UA whether the call came via gRPC directly
// or via REST through the gateway.
func MetaFromGRPC(ctx context.Context) service.RequestMetadata {
	md, _ := metadata.FromIncomingContext(ctx)
	// Distinguish a REST (grpc-gateway) call from a direct gRPC call: the
	// gateway injects x-authorizer-transport=rest (see gateway.Handler's
	// WithMetadata). Absent that marker, the call arrived over pure gRPC.
	protocol := constants.ProtocolGRPC
	if firstHeader(md, "x-authorizer-transport") == constants.ProtocolREST {
		protocol = constants.ProtocolREST
	}
	meta := service.RequestMetadata{
		HostURL:             firstHeader(md, "x-authorizer-url", "grpcgateway-x-authorizer-url"),
		IPAddress:           clientIPFromGRPC(ctx, md),
		UserAgent:           firstHeader(md, "grpcgateway-user-agent", "user-agent"),
		AuthorizationHeader: firstHeader(md, "authorization", "grpcgateway-authorization"),
		Cookies:             cookiesFromMetadata(md),
		Protocol:            protocol,
	}
	// Default the host URL when no header was set (pure-gRPC caller, no
	// proxy headers). The :authority pseudo-header is the gRPC equivalent
	// of Host; use it as a fallback.
	if meta.HostURL == "" {
		if authority := firstHeader(md, ":authority"); authority != "" {
			meta.HostURL = "http://" + authority
		}
	}
	// Synthesize an *http.Request mirroring the extracted metadata. Several
	// migrated service methods (Profile, Permissions, Logout, Session,
	// ValidateSession) still hand a gin.Context shim to TokenProvider helpers
	// that read Request.Header / Request.Cookies(). Without a non-nil Request
	// those helpers dereference nil and panic. Building the request here keeps
	// the gRPC/REST path behaving exactly like the gin path.
	meta.Request = synthRequest(meta)

	// Carry the custom admin-secret header onto the synthesized request so the
	// admin-auth check (TokenProvider.IsSuperAdmin, reached via the gin shim in
	// service.requireSuperAdmin) sees it identically over pure gRPC and over
	// REST. The REST gateway forwards it after gateway.WithIncomingHeaderMatcher
	// allows it; pure-gRPC callers set it as metadata directly.
	if adminSecret := firstHeader(md, "x-authorizer-admin-secret", "grpcgateway-x-authorizer-admin-secret"); adminSecret != "" {
		meta.Request.Header.Set("x-authorizer-admin-secret", adminSecret)
	}
	return meta
}

// bufconnNetwork is the Network() reported by google.golang.org/grpc/test/bufconn.
// The REST surface reaches the gRPC handlers over an in-process bufconn (see
// package gateway), so this is how a gateway-originated call is told apart from
// a real network gRPC call. It comes from the transport, not from the caller,
// which is what makes it safe to branch on: a remote gRPC client cannot present
// a bufconn peer no matter what metadata it sends.
const bufconnNetwork = "bufconn"

// clientIPFromGRPC resolves the caller's address for a gRPC or REST call.
//
// Forwarded metadata is a CLAIM the caller makes about itself. gRPC metadata is
// trivially settable by any client — `x-forwarded-for` is not a privileged key —
// so believing it unconditionally reproduced GHSA-93hc-xq3w-xw87 on this
// surface: the admin-secret lockout (reached over gRPC and REST via
// service.requireSuperAdmin) buckets on this value, and a caller that picks its
// own bucket is never throttled.
//
// The resolution is delegated to utils.GetIP so gRPC, REST and plain HTTP share
// ONE trusted-proxy rule. All this function does is establish the unforgeable
// peer address to hand it:
//
//   - Direct gRPC: peer.Addr is the connection's real remote address. Forwarded
//     metadata is honoured only when that peer is a configured trusted proxy.
//   - REST via the in-process gateway: peer.Addr is "bufconn" and says nothing
//     about the client. grpc-gateway's AnnotateContext appends the real HTTP
//     RemoteAddr to the RIGHT of any client-supplied X-Forwarded-For, so the
//     rightmost entry of that chain is the true HTTP peer and everything to its
//     left is the ordinary forwarded chain. Splitting it back apart hands
//     utils.GetIP exactly the shape it would have seen on the gin path, so a
//     REST call and the equivalent direct HTTP call resolve identically.
func clientIPFromGRPC(ctx context.Context, md metadata.MD) string {
	// LAST value, not first. A metadata key can hold several values, and
	// grpc-gateway appends its own authoritative chain (client XFF + the real
	// HTTP RemoteAddr) AFTER any entry a client smuggled in via the
	// "Grpc-Metadata-X-Forwarded-For" spelling, which DefaultHeaderMatcher
	// forwards with the prefix stripped. Reading the first value therefore
	// handed an unauthenticated REST caller its own choice of client IP and
	// reinstated GHSA-93hc-xq3w-xw87 on this surface. gateway.Handler now also
	// refuses those keys at the boundary; this is the second lock.
	forwarded := lastHeader(md, "x-forwarded-for", "grpcgateway-x-forwarded-for")
	realIP := lastHeader(md, "x-real-ip", "grpcgateway-x-real-ip")

	peerAddr := ""
	viaGateway := false
	if pr, ok := peer.FromContext(ctx); ok && pr.Addr != nil {
		if pr.Addr.Network() == bufconnNetwork {
			viaGateway = true
		} else {
			peerAddr = pr.Addr.String()
		}
	}

	if viaGateway {
		items := strings.Split(forwarded, ",")
		if n := len(items); n > 0 && strings.TrimSpace(items[n-1]) != "" {
			peerAddr = strings.TrimSpace(items[n-1])
			forwarded = strings.Join(items[:n-1], ",")
		} else {
			// No chain at all. Two ways to get here, both safe:
			//
			//   - The MCP surface, which also serves over an in-process bufconn
			//     (internal/mcp) but forwards ONLY the Authorization header —
			//     deliberately, see Server.stampAuth — so it never carries a
			//     forwarded chain. Its callers therefore have no resolvable
			//     address and record an empty one. That is not a regression in
			//     kind: before this change the same calls recorded the literal
			//     string "bufconn". Nothing is spoofable either way, because a
			//     remote MCP client's headers never cross the bridge, and the
			//     admin-secret lockout is unreachable over MCP (its interceptor
			//     accepts bearer tokens only).
			//   - A gateway call whose HTTP RemoteAddr was empty.
			//
			// Either way there is nothing anchoring the metadata to a
			// connection, so no caller-supplied value may be promoted.
			forwarded, realIP = "", ""
		}
	}

	r := &http.Request{Header: http.Header{}, RemoteAddr: peerAddr}
	if forwarded != "" {
		r.Header.Set("X-Forwarded-For", forwarded)
	}
	if realIP != "" {
		r.Header.Set("X-Real-Ip", realIP)
	}
	return utils.GetIP(r)
}

// synthRequest reconstructs a minimal *http.Request from the transport-neutral
// RequestMetadata so the legacy gin-shim helpers (which read Header, Cookies,
// Host, and RemoteAddr) work identically over gRPC/REST and direct HTTP.
func synthRequest(meta service.RequestMetadata) *http.Request {
	req := &http.Request{
		Header: http.Header{},
		URL:    &url.URL{},
	}
	if meta.AuthorizationHeader != "" {
		req.Header.Set("Authorization", meta.AuthorizationHeader)
	}
	if meta.UserAgent != "" {
		req.Header.Set("User-Agent", meta.UserAgent)
	}
	if meta.IPAddress != "" {
		req.Header.Set("X-Forwarded-For", meta.IPAddress)
		req.RemoteAddr = meta.IPAddress
	}
	if meta.HostURL != "" {
		req.Header.Set("X-Authorizer-URL", meta.HostURL)
		if u, err := url.Parse(meta.HostURL); err == nil && u.Host != "" {
			req.Host = u.Host
		}
	}
	for _, c := range meta.Cookies {
		if c != nil {
			req.AddCookie(c)
		}
	}
	return req
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

// lastHeader returns the LAST value of the first key present.
//
// firstHeader's semantics are wrong for any key a client can also inject: a
// metadata key holds a list, and the value this server appends itself lands at
// the end. For the forwarding headers that distinction is the security
// boundary — see clientIPFromGRPC.
func lastHeader(md metadata.MD, keys ...string) string {
	for _, k := range keys {
		if vs := md.Get(k); len(vs) > 0 {
			return vs[len(vs)-1]
		}
	}
	return ""
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
