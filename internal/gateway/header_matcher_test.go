package gateway

import (
	"context"
	"net/http"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// TestIncomingHeaderMatcherRefusesForwardingHeaders is the boundary half of the
// regression for the REST bypass of GHSA-93hc-xq3w-xw87.
//
// runtime.DefaultHeaderMatcher strips a "Grpc-Metadata-" prefix and forwards
// whatever follows, so `Grpc-Metadata-X-Forwarded-For: <attacker>` reached the
// gRPC layer as a genuine `x-forwarded-for` metadata entry ahead of the
// authoritative chain AnnotateContext appends. The admin-secret lockout then
// bucketed on an attacker-chosen value again, unauthenticated, over REST.
//
// The client IP must come from the connection, never from a header a client
// can spell.
func TestIncomingHeaderMatcherRefusesForwardingHeaders(t *testing.T) {
	for _, smuggled := range []string{
		"Grpc-Metadata-X-Forwarded-For",
		"Grpc-Metadata-X-Real-Ip",
		"Grpc-Metadata-X-Forwarded-Host",
		"grpc-metadata-x-forwarded-for",
		"GRPC-METADATA-X-FORWARDED-FOR",
	} {
		t.Run(smuggled, func(t *testing.T) {
			md := annotate(t, map[string]string{smuggled: "203.0.113.9"})
			for _, key := range []string{"x-forwarded-for", "x-real-ip", "x-forwarded-host"} {
				for _, v := range md.Get(key) {
					assert.NotEqual(t, "203.0.113.9", v,
						"%s smuggled an attacker value into %s", smuggled, key)
				}
			}
		})
	}

	t.Run("the admin-secret header is still forwarded", func(t *testing.T) {
		md := annotate(t, map[string]string{"x-authorizer-admin-secret": "s3cret"})
		assert.Equal(t, []string{"s3cret"}, md.Get("x-authorizer-admin-secret"),
			"the filter must not break admin header auth over REST")
	})

	t.Run("the gateway's own chain still reaches the server", func(t *testing.T) {
		md := annotate(t, nil)
		require.Len(t, md.Get("x-forwarded-for"), 1)
		assert.Equal(t, "198.51.100.10", md.Get("x-forwarded-for")[0],
			"the real HTTP peer must still be forwarded")
	})
}

// annotate drives the REAL mux built by Handler's options.
func annotate(t *testing.T, headers map[string]string) metadata.MD {
	t.Helper()
	mux := runtime.NewServeMux(incomingHeaderMatcherOption())
	r, err := http.NewRequest(http.MethodPost, "http://localhost:8080/v1/admin/login", nil)
	require.NoError(t, err)
	r.RemoteAddr = "198.51.100.10:40000"
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	ctx, err := runtime.AnnotateContext(context.Background(), mux, r,
		"/authorizer.v1.AuthorizerAdminService/AdminLogin")
	require.NoError(t, err)
	out, _ := metadata.FromOutgoingContext(ctx)
	return out
}
