package interceptors

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/authctx"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/token"
)

// mcpStubProvider records exactly which validation path a request took, so the
// tests below can assert that MCP callers reach ValidateMCPAccessToken and
// nothing else.
type mcpStubProvider struct {
	token.Provider

	claims map[string]interface{}
	err    error

	seenToken    string
	seenResource string
	mcpChecks    int
	defaultCheck int
}

func (s *mcpStubProvider) GetAccessToken(gc *gin.Context) (string, error) {
	auth := gc.Request.Header.Get("Authorization")
	if len(auth) < 8 || auth[:7] != "Bearer " {
		return "", fmt.Errorf("unauthorized")
	}
	return auth[7:], nil
}

func (s *mcpStubProvider) ValidateMCPAccessToken(_ *gin.Context, accessToken, resource string) (map[string]interface{}, error) {
	s.mcpChecks++
	s.seenToken = accessToken
	s.seenResource = resource
	if s.err != nil {
		return nil, s.err
	}
	return s.claims, nil
}

func (s *mcpStubProvider) GetUserIDFromSessionOrAccessToken(_ *gin.Context) (*token.SessionOrAccessTokenData, error) {
	s.defaultCheck++
	return &token.SessionOrAccessTokenData{UserID: "default-rule-user"}, nil
}

func mcpGinContext(header http.Header) *gin.Context {
	req, _ := http.NewRequest(http.MethodPost, "/mcp", nil)
	if header != nil {
		req.Header = header
	}
	return &gin.Context{Request: req}
}

// TestMCPTokenResolver covers the resolver in isolation. Until the transport
// lands nothing else constructs it, so without this the first execution of
// auth-critical code would be in production.
func TestMCPTokenResolver(t *testing.T) {
	const resource = "https://auth.example.com/mcp"

	t.Run("a bearer token is validated against the configured resource", func(t *testing.T) {
		stub := &mcpStubProvider{claims: map[string]interface{}{
			"sub":          "user-1",
			"login_method": "basic_auth",
			"nonce":        "n1",
			"scope":        []interface{}{"openid", "profile"},
		}}
		h := http.Header{}
		h.Set("Authorization", "Bearer tok-123")

		data, err := MCPTokenResolver(stub, resource)(mcpGinContext(h))
		require.NoError(t, err)
		assert.Equal(t, "user-1", data.UserID)
		assert.Equal(t, "basic_auth", data.LoginMethod)
		assert.Equal(t, "n1", data.Nonce)
		assert.Equal(t, []string{"openid", "profile"}, data.Scope)

		assert.Equal(t, "tok-123", stub.seenToken)
		assert.Equal(t, resource, stub.seenResource,
			"the resource must be the one captured at wiring time, never derived from the request")
		assert.Equal(t, 1, stub.mcpChecks)
		assert.Zero(t, stub.defaultCheck, "the MCP resolver must never consult the default rule")
	})

	t.Run("a cookie-only request is refused without touching either validator", func(t *testing.T) {
		// The default resolver would accept a session cookie. MCP is a
		// cookieless, Authorization-header protocol, and /mcp is CSRF-exempt
		// precisely because no cookie can authenticate it.
		stub := &mcpStubProvider{claims: map[string]interface{}{"sub": "user-1"}}
		h := http.Header{}
		h.Set("Cookie", "authorizer_session=whatever")

		_, err := MCPTokenResolver(stub, resource)(mcpGinContext(h))
		require.Error(t, err)
		assert.Zero(t, stub.mcpChecks)
		assert.Zero(t, stub.defaultCheck)
	})

	t.Run("a rejected token does not fall back to the default rule", func(t *testing.T) {
		stub := &mcpStubProvider{err: fmt.Errorf("unauthorized: token audience is not this mcp server")}
		h := http.Header{}
		h.Set("Authorization", "Bearer wrong-audience")

		_, err := MCPTokenResolver(stub, resource)(mcpGinContext(h))
		require.Error(t, err)
		assert.Zero(t, stub.defaultCheck)
	})

	t.Run("claims with no sub are refused", func(t *testing.T) {
		stub := &mcpStubProvider{claims: map[string]interface{}{"login_method": "basic_auth"}}
		h := http.Header{}
		h.Set("Authorization", "Bearer tok")

		_, err := MCPTokenResolver(stub, resource)(mcpGinContext(h))
		require.Error(t, err)
	})
}

// TestAuth_SoleAuthorityDisablesCookieAndAdminSecretPaths pins the property the
// MCP audience boundary depends on: when a server supplies its own resolver,
// that resolver is the ONLY way in.
//
// Two paths in the interceptor authenticate without consulting any resolver —
// tp.IsSuperAdmin (an admin cookie or the x-authorizer-admin-secret header) and
// the Session RPC's cookie-only branch. transport.MetaFromGRPC reconstructs
// cookies from gRPC metadata, so on the MCP server those would be reachable the
// moment the HTTP bridge forwarded request headers. Before this guard the
// boundary held only because neither Session nor any admin RPC happened to be
// mcp_tool-exposed — a proto annotation away from a browser cookie
// authenticating a tool call on an internet-facing, CSRF-exempt endpoint.
func TestAuth_SoleAuthorityDisablesCookieAndAdminSecretPaths(t *testing.T) {
	t.Run("super-admin is not honoured when a resolver is set", func(t *testing.T) {
		stub := &stubTokenProvider{superAdmin: true, tokenErr: status.Error(codes.Unauthenticated, "no")}
		mw := Auth(stub, nil, func(_ *gin.Context) (*token.SessionOrAccessTokenData, error) {
			return nil, status.Error(codes.Unauthenticated, "wrong audience")
		})

		called := false
		_, err := mw(context.Background(), nil, info(authorizerv1.AuthorizerAdminService_OrgMembers_FullMethodName),
			func(context.Context, any) (any, error) { called = true; return nil, nil })

		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
		assert.False(t, called)
		assert.Zero(t, stub.superAdminChecks,
			"an admin cookie / admin secret is not a credential this surface accepts, so it must not even be checked")
	})

	t.Run("super-admin still works on a default server", func(t *testing.T) {
		// The inverse: the guard must not disturb every existing gRPC server.
		stub := &stubTokenProvider{superAdmin: true}
		mw := Auth(stub, nil, nil)

		var isSuper bool
		_, err := mw(context.Background(), nil, info(authorizerv1.AuthorizerAdminService_OrgMembers_FullMethodName),
			func(ctx context.Context, _ any) (any, error) {
				p, ok := authctx.FromContext(ctx)
				require.True(t, ok)
				isSuper = p.IsSuperAdmin
				return nil, nil
			})
		require.NoError(t, err)
		assert.True(t, isSuper)
		assert.Equal(t, 1, stub.superAdminChecks)
	})

	t.Run("the Session cookie branch is disabled when a resolver is set", func(t *testing.T) {
		stub := &stubTokenProvider{sessionData: &token.SessionData{Subject: "cookie-user"}}
		mw := Auth(stub, nil, func(_ *gin.Context) (*token.SessionOrAccessTokenData, error) {
			return nil, status.Error(codes.Unauthenticated, "wrong audience")
		})

		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			"cookie", constants.AppCookieName+"_session=sess-token",
		))
		called := false
		_, err := mw(ctx, &authorizerv1.SessionRequest{}, info(authorizerv1.AuthorizerService_Session_FullMethodName),
			func(context.Context, any) (any, error) { called = true; return nil, nil })

		require.Error(t, err)
		assert.False(t, called)
		assert.Zero(t, stub.sessionChecks, "a browser session must not authenticate a resolver-governed surface")
	})
}
