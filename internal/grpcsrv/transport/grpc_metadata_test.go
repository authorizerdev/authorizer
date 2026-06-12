package transport

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/authorizerdev/authorizer/internal/service"
)

func TestMetaFromGRPC_ExtractsAllSignals(t *testing.T) {
	md := metadata.New(map[string]string{
		"grpcgateway-x-authorizer-url": "https://auth.example.com",
		"grpcgateway-x-forwarded-for":  "10.1.2.3",
		"grpcgateway-user-agent":       "browser/1.0",
		"grpcgateway-authorization":    "Bearer abc",
		"grpcgateway-cookie":           "authorizer_session=abc; mfa=xyz",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	meta := MetaFromGRPC(ctx)
	assert.Equal(t, "https://auth.example.com", meta.HostURL)
	assert.Equal(t, "10.1.2.3", meta.IPAddress)
	assert.Equal(t, "browser/1.0", meta.UserAgent)
	assert.Equal(t, "Bearer abc", meta.AuthorizationHeader)
	require.Len(t, meta.Cookies, 2)
	cookieValues := map[string]string{}
	for _, c := range meta.Cookies {
		cookieValues[c.Name] = c.Value
	}
	assert.Equal(t, "abc", cookieValues["authorizer_session"])
	assert.Equal(t, "xyz", cookieValues["mfa"])
}

func TestMetaFromGRPC_FallsBackToAuthority(t *testing.T) {
	md := metadata.New(map[string]string{":authority": "auth.example.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	meta := MetaFromGRPC(ctx)
	assert.Equal(t, "http://auth.example.com", meta.HostURL)
}

func TestMetaFromGRPC_NoMetadata(t *testing.T) {
	meta := MetaFromGRPC(context.Background())
	// All transport-derived signals are empty with no metadata...
	assert.Empty(t, meta.HostURL)
	assert.Empty(t, meta.IPAddress)
	assert.Empty(t, meta.UserAgent)
	assert.Empty(t, meta.AuthorizationHeader)
	assert.Empty(t, meta.Cookies)
	// ...but Request is always synthesized (non-nil) so the gin-shim helpers
	// in the migrated service methods never dereference a nil *http.Request.
	require.NotNil(t, meta.Request)
	assert.Empty(t, meta.Request.Header.Get("Authorization"))
}

func TestMetaFromGRPC_SynthesizesRequestFromSignals(t *testing.T) {
	md := metadata.New(map[string]string{
		"grpcgateway-x-authorizer-url": "https://auth.example.com",
		"grpcgateway-x-forwarded-for":  "10.1.2.3",
		"grpcgateway-user-agent":       "browser/1.0",
		"grpcgateway-authorization":    "Bearer abc",
		"grpcgateway-cookie":           "authorizer_session=abc",
	})
	meta := MetaFromGRPC(metadata.NewIncomingContext(context.Background(), md))
	require.NotNil(t, meta.Request)
	// The synthesized request must carry the bearer + cookie so legacy
	// TokenProvider helpers (which read Request.Header / Request.Cookies())
	// behave identically over gRPC/REST and direct HTTP.
	assert.Equal(t, "Bearer abc", meta.Request.Header.Get("Authorization"))
	assert.Equal(t, "browser/1.0", meta.Request.Header.Get("User-Agent"))
	assert.Equal(t, "auth.example.com", meta.Request.Host)
	c, err := meta.Request.Cookie("authorizer_session")
	require.NoError(t, err)
	assert.Equal(t, "abc", c.Value)
}

func TestCookiesFromMetadata_MultipleHeaders(t *testing.T) {
	md := metadata.MD{}
	md.Append("grpcgateway-cookie", "a=1; b=2")
	md.Append("grpcgateway-cookie", "c=3")
	cookies := cookiesFromMetadata(md)
	require.Len(t, cookies, 3)
	got := map[string]string{}
	for _, c := range cookies {
		got[c.Name] = c.Value
	}
	assert.Equal(t, map[string]string{"a": "1", "b": "2", "c": "3"}, got)
}

func TestCookiesFromMetadata_NoCookies(t *testing.T) {
	assert.Nil(t, cookiesFromMetadata(metadata.MD{}))
}

func TestApplyToGRPC_NilSafe(t *testing.T) {
	// nil receiver / empty cookies must not error.
	assert.NoError(t, ApplyToGRPC(context.Background(), nil))
	assert.NoError(t, ApplyToGRPC(context.Background(), &service.ResponseSideEffects{}))
	assert.NoError(t, ApplyToGRPC(context.Background(), &service.ResponseSideEffects{Cookies: []*http.Cookie{nil}}))
}

// Note: ApplyToGRPC's success path uses grpc.SendHeader which requires a
// real *grpc.ServerStream / handler context. That's covered end-to-end by
// the integration tests in internal/integration_tests where cookies emitted
// by a CreateSession handler land in the REST response.
