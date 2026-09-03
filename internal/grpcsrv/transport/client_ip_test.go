package transport

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/authorizerdev/authorizer/internal/utils"
)

// bufconnAddr mimics google.golang.org/grpc/test/bufconn's addr, which is what
// peer.Addr reports for a REST call arriving through the in-process gateway.
type bufconnAddr struct{}

func (bufconnAddr) Network() string { return "bufconn" }
func (bufconnAddr) String() string  { return "bufconn" }

// grpcCtx builds an incoming gRPC context with the given peer address and
// metadata pairs.
func grpcCtx(addr net.Addr, kv map[string]string) context.Context {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(kv))
	if addr != nil {
		ctx = peer.NewContext(ctx, &peer.Peer{Addr: addr})
	}
	return ctx
}

func tcp(t *testing.T, s string) net.Addr {
	t.Helper()
	a, err := net.ResolveTCPAddr("tcp", s)
	require.NoError(t, err)
	return a
}

// TestClientIPFromGRPC_DirectCallerCannotForge is the gRPC half of the
// regression for GHSA-93hc-xq3w-xw87. gRPC metadata is fully caller-controlled,
// so `x-forwarded-for` must not be believed from an untrusted peer — otherwise
// the admin-secret lockout (reached over gRPC via requireSuperAdmin) buckets on
// a value the attacker picks.
func TestClientIPFromGRPC_DirectCallerCannotForge(t *testing.T) {
	require.NoError(t, utils.SetTrustedProxies(nil))

	got := MetaFromGRPC(grpcCtx(tcp(t, "198.51.100.10:40000"), map[string]string{
		"x-forwarded-for": "203.0.113.9",
		"x-real-ip":       "203.0.113.8",
	})).IPAddress
	assert.Equal(t, "198.51.100.10", got, "forged metadata must be ignored from an untrusted peer")

	// The exploit shape: rotate the forged value, the bucket key must not move.
	first := ""
	for i := 1; i <= 5; i++ {
		ip := MetaFromGRPC(grpcCtx(tcp(t, "198.51.100.10:40000"), map[string]string{
			"x-forwarded-for": fmt.Sprintf("203.0.113.%d", i),
		})).IPAddress
		if first == "" {
			first = ip
		}
		assert.Equal(t, first, ip, "rotating forged metadata produced a new lockout bucket")
	}
}

func TestClientIPFromGRPC_DirectCallerNewConnection(t *testing.T) {
	require.NoError(t, utils.SetTrustedProxies(nil))
	// A fresh source port per connection must not rotate the bucket key.
	a := MetaFromGRPC(grpcCtx(tcp(t, "198.51.100.11:40001"), nil)).IPAddress
	b := MetaFromGRPC(grpcCtx(tcp(t, "198.51.100.11:40002"), nil)).IPAddress
	assert.Equal(t, "198.51.100.11", a)
	assert.Equal(t, a, b)
}

func TestClientIPFromGRPC_TrustedPeerIsHonoured(t *testing.T) {
	require.NoError(t, utils.SetTrustedProxies([]string{"10.0.0.0/8"}))
	t.Cleanup(func() { require.NoError(t, utils.SetTrustedProxies(nil)) })

	got := MetaFromGRPC(grpcCtx(tcp(t, "10.0.0.5:443"), map[string]string{
		"x-forwarded-for": "1.1.1.1",
	})).IPAddress
	assert.Equal(t, "1.1.1.1", got)
}

// TestClientIPFromGRPC_GatewayUsesRightmostEntry pins the REST path.
// grpc-gateway's AnnotateContext appends the real HTTP RemoteAddr to the RIGHT
// of any client-supplied X-Forwarded-For, so the rightmost entry is the true
// HTTP peer.
func TestClientIPFromGRPC_GatewayUsesRightmostEntry(t *testing.T) {
	require.NoError(t, utils.SetTrustedProxies(nil))

	// A REST client forged "203.0.113.9"; the gateway appended the real peer.
	got := MetaFromGRPC(grpcCtx(bufconnAddr{}, map[string]string{
		"x-forwarded-for": "203.0.113.9, 198.51.100.10",
	})).IPAddress
	assert.Equal(t, "198.51.100.10", got, "forged leftmost entry must not win")

	// No forgery: gateway appended the peer to an empty chain.
	got = MetaFromGRPC(grpcCtx(bufconnAddr{}, map[string]string{
		"x-forwarded-for": "198.51.100.10",
	})).IPAddress
	assert.Equal(t, "198.51.100.10", got)
}

// TestClientIPFromGRPC_GatewayBehindTrustedProxy is the deployment shape the
// project actually ships into (Railway, nginx, a cloud LB): a real reverse
// proxy in front of the HTTP listener, whose address the gateway appends last.
//
// Note this is safe under BOTH proxy behaviours. If the proxy strips the
// client's header and writes a single entry, the chain is
// "<client>, <proxy>" and the walk returns <client>. If it appends without
// stripping, the chain is "<forged>, <client>, <proxy>" and the walk still
// returns <client>, because it stops at the rightmost UNTRUSTED entry. The
// commonly-given "just take the leftmost entry" advice is spoofable in the
// second case; this is not.
func TestClientIPFromGRPC_GatewayBehindTrustedProxy(t *testing.T) {
	require.NoError(t, utils.SetTrustedProxies([]string{"100.64.0.0/10"}))
	t.Cleanup(func() { require.NoError(t, utils.SetTrustedProxies(nil)) })

	t.Run("proxy strips client header", func(t *testing.T) {
		got := MetaFromGRPC(grpcCtx(bufconnAddr{}, map[string]string{
			"x-forwarded-for": "1.1.1.1, 100.64.0.24",
		})).IPAddress
		assert.Equal(t, "1.1.1.1", got)
	})

	t.Run("proxy appends without stripping", func(t *testing.T) {
		got := MetaFromGRPC(grpcCtx(bufconnAddr{}, map[string]string{
			"x-forwarded-for": "203.0.113.9, 1.1.1.1, 100.64.0.24",
		})).IPAddress
		assert.Equal(t, "1.1.1.1", got, "forged entry left of the real client must be ignored")
	})
}

// TestClientIPFromGRPC_GatewayWithNoChainFailsClosed: if the gateway had no
// RemoteAddr to append there is nothing anchoring the metadata to a connection,
// so no caller-supplied value may be promoted.
func TestClientIPFromGRPC_GatewayWithNoChainFailsClosed(t *testing.T) {
	require.NoError(t, utils.SetTrustedProxies(nil))
	got := MetaFromGRPC(grpcCtx(bufconnAddr{}, map[string]string{
		"x-real-ip": "203.0.113.9",
	})).IPAddress
	assert.Empty(t, got)
}

// TestClientIPFromGRPC_MatchesHTTPPath: the same client reaching the same
// deployment must get the same identity whether it went through gin or through
// the REST gateway. Divergence here is how the original bug survived review on
// one surface while being fixed on the other.
func TestClientIPFromGRPC_MatchesHTTPPath(t *testing.T) {
	require.NoError(t, utils.SetTrustedProxies([]string{"100.64.0.0/10"}))
	t.Cleanup(func() { require.NoError(t, utils.SetTrustedProxies(nil)) })

	// gin path: proxy 100.64.0.24 is the peer, client 1.1.1.1 in the header.
	r := &http.Request{Header: http.Header{}, RemoteAddr: "100.64.0.24:443"}
	r.Header.Set("X-Forwarded-For", "1.1.1.1")
	viaHTTP := utils.GetIP(r)
	// REST path: gateway appended the same proxy address to the same chain.
	viaGRPC := MetaFromGRPC(grpcCtx(bufconnAddr{}, map[string]string{
		"x-forwarded-for": "1.1.1.1, 100.64.0.24",
	})).IPAddress

	assert.Equal(t, viaHTTP, viaGRPC)
	assert.Equal(t, "1.1.1.1", viaHTTP)
}

// gatewayMatcher mirrors gateway.Handler's WithIncomingHeaderMatcher. Kept in
// sync deliberately: this test asserts the RESOLVER survives a smuggled entry
// even if the boundary filter were ever removed, so it must be able to build
// the pre-filter metadata shape.
func annotateLikeGateway(t *testing.T, remoteAddr string, headers map[string]string) metadata.MD {
	t.Helper()
	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			if strings.EqualFold(key, "x-authorizer-admin-secret") {
				return key, true
			}
			return runtime.DefaultHeaderMatcher(key)
		}),
	)
	r, err := http.NewRequest(http.MethodPost, "http://localhost:8080/v1/admin/login", nil)
	require.NoError(t, err)
	r.RemoteAddr = remoteAddr
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	ctx, err := runtime.AnnotateContext(context.Background(), mux, r,
		"/authorizer.v1.AuthorizerAdminService/AdminLogin")
	require.NoError(t, err)
	out, _ := metadata.FromOutgoingContext(ctx)
	return out
}

// TestClientIPFromGRPC_GrpcMetadataPrefixCannotSmuggle is the regression test
// for a REST-surface bypass of GHSA-93hc-xq3w-xw87 found in adversarial review,
// not in the original report.
//
// grpc-gateway's DefaultHeaderMatcher strips a "Grpc-Metadata-" prefix and
// forwards what remains, so `Grpc-Metadata-X-Forwarded-For: <attacker>` becomes
// a real `x-forwarded-for` metadata entry — placed BEFORE the authoritative
// chain AnnotateContext appends afterwards. Reading the FIRST value therefore
// handed an unauthenticated REST caller its own choice of client IP, and the
// admin-secret lockout bucketed on it again.
//
// Two independent locks now: gateway.Handler refuses these keys at the
// boundary, and this resolver reads the LAST value. This test builds metadata
// through the REAL runtime.AnnotateContext WITHOUT the boundary filter, so it
// exercises the resolver's lock on its own. (metadata.New cannot express a
// multi-value key, which is exactly why the original tests missed this.)
func TestClientIPFromGRPC_GrpcMetadataPrefixCannotSmuggle(t *testing.T) {
	require.NoError(t, utils.SetTrustedProxies(nil))

	md := annotateLikeGateway(t, "198.51.100.10:40000", map[string]string{
		"Grpc-Metadata-X-Forwarded-For": "203.0.113.9",
	})
	require.Len(t, md.Get("x-forwarded-for"), 2,
		"precondition: the smuggled entry must precede the gateway's own chain")

	ctx := peer.NewContext(metadata.NewIncomingContext(context.Background(), md),
		&peer.Peer{Addr: bufconnAddr{}})
	assert.Equal(t, "198.51.100.10", MetaFromGRPC(ctx).IPAddress,
		"a smuggled Grpc-Metadata-X-Forwarded-For must not become the client IP")

	// The exploit shape: rotate the smuggled value, the bucket key must not move.
	first := ""
	for i := 1; i <= 5; i++ {
		m := annotateLikeGateway(t, "198.51.100.10:40000", map[string]string{
			"Grpc-Metadata-X-Forwarded-For": fmt.Sprintf("203.0.113.%d", i),
		})
		c := peer.NewContext(metadata.NewIncomingContext(context.Background(), m),
			&peer.Peer{Addr: bufconnAddr{}})
		ip := MetaFromGRPC(c).IPAddress
		if first == "" {
			first = ip
		}
		assert.Equal(t, first, ip, "rotating the smuggled header moved the lockout bucket")
	}

	// Same for X-Real-Ip.
	md = annotateLikeGateway(t, "198.51.100.11:40000", map[string]string{
		"Grpc-Metadata-X-Real-Ip": "203.0.113.9",
	})
	ctx = peer.NewContext(metadata.NewIncomingContext(context.Background(), md),
		&peer.Peer{Addr: bufconnAddr{}})
	assert.Equal(t, "198.51.100.11", MetaFromGRPC(ctx).IPAddress,
		"a smuggled Grpc-Metadata-X-Real-Ip must not become the client IP")
}
