package integration_tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
)

// bearerGinContext builds a request carrying `tok` as a bearer token, addressed
// to the test server's REAL host.
//
// The host is not incidental. ValidateJWTClaims compares the token's `iss`
// against parsers.GetHost(gc), and httptest.NewRequest defaults the host to
// "example.com" — so a context built that way rejects every token this suite
// mints with an issuer mismatch. A negative assertion would then pass for a
// reason unrelated to the rule under test, and would keep passing after that
// rule was removed.
func bearerGinContext(t *testing.T, ts *testSetup, tok string) *gin.Context {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, testAuthorizerHost(ts)+"/graphql", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+tok)
	return &gin.Context{Request: req}
}

// mintStatefulAccessToken issues an access token bound to `resource` (RFC 8707)
// and registers it in the memory store exactly as /oauth/token does, so it is a
// genuine first-party token and not a hand-rolled JWT. Passing resource="" is
// how a test produces an ordinary login token, whose audience is the client id.
func mintStatefulAccessToken(t *testing.T, ts *testSetup, user *schemas.User, resource string) string {
	t.Helper()
	nonce := uuid.NewString()
	tok, _, err := ts.TokenProvider.CreateAccessToken(&token.AuthTokenConfig{
		User:        user,
		Nonce:       nonce,
		Roles:       []string{"user"},
		Scope:       []string{"openid"},
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
		HostName:    testAuthorizerHost(ts),
		Resource:    resource,
		ExpireTime:  "30m",
	})
	require.NoError(t, err)
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		constants.AuthRecipeMethodBasicAuth+":"+user.ID,
		constants.TokenTypeAccessToken+"_"+nonce,
		tok,
		time.Now().Add(time.Hour).Unix(),
	))
	return tok
}

// mintMachineAccessToken is mintStatefulAccessToken for a service account: `sub`
// is the client's surrogate id and there is no user. Same stateful registration,
// because the token endpoint registers machine tokens the same way.
func mintMachineAccessToken(t *testing.T, ts *testSetup, clientRowID, resource string) string {
	t.Helper()
	nonce := uuid.NewString()
	tok, _, err := ts.TokenProvider.CreateAccessToken(&token.AuthTokenConfig{
		ServiceAccountID: clientRowID,
		Nonce:            nonce,
		Scope:            []string{"openid"},
		LoginMethod:      constants.AuthRecipeMethodServiceAccount,
		HostName:         testAuthorizerHost(ts),
		Resource:         resource,
		ExpireTime:       "30m",
	})
	require.NoError(t, err)
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		constants.AuthRecipeMethodServiceAccount+":"+clientRowID,
		constants.TokenTypeAccessToken+"_"+nonce,
		tok,
		time.Now().Add(time.Hour).Unix(),
	))
	return tok
}

// TestMCPAccessTokenAudienceBoundary pins BOTH directions of the audience
// boundary between the MCP surface and Authorizer's first-party surfaces.
//
// This matters more than either half alone. The MCP specification requires that
// a server accept only tokens naming it as the audience; Authorizer additionally
// requires that a token naming the MCP server is NOT accepted anywhere else. One
// rule without the other is not a partial implementation, it is a vulnerability:
//
//   - too loose at /mcp and a token minted for any other resource server (or an
//     ordinary login token) authenticates MCP tool calls;
//   - too loose on the first-party path and an MCP token — which a client may
//     hand to a semi-trusted agent — becomes a full GraphQL/gRPC credential.
//
// These are also the regression tests for the decision-core extraction: the
// first-party rule is asserted here in its own right, so a future edit to the
// shared core that quietly widened it would fail.
func TestMCPAccessTokenAudienceBoundary(t *testing.T) {
	cfg := getTestConfig()
	// The canonical resource is derived from --url and nothing else. Setting it
	// here is what the operator does at startup; without it MCPResource() is
	// empty and the server refuses to enable MCP at all.
	cfg.AuthorizerURL = "https://auth.example.com"
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	resource := cfg.MCPResource()
	require.Equal(t, "https://auth.example.com/mcp", resource,
		"the canonical resource form is what clients send as `resource` and what tokens carry as `aud` — it must not drift")

	now := time.Now().Unix()
	user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:           refs.NewStringRef("mcp_" + uuid.NewString() + "@authorizer.dev"),
		EmailVerifiedAt: &now,
		SignupMethods:   constants.AuthRecipeMethodBasicAuth,
		Roles:           "user",
	})
	require.NoError(t, err)

	gc := &gin.Context{Request: ts.GinContext.Request}

	t.Run("a token bound to this MCP server is accepted at /mcp", func(t *testing.T) {
		claims, vErr := ts.TokenProvider.ValidateMCPAccessToken(gc, mintStatefulAccessToken(t, ts, user, resource), resource)
		require.NoError(t, vErr, "a correctly-audienced token must reach MCP, or the surface is dead code")
		assert.Equal(t, user.ID, claims["sub"])
	})

	t.Run("the same token is rejected at authorizer's own surfaces", func(t *testing.T) {
		tok := mintStatefulAccessToken(t, ts, user, resource)
		_, vErr := ts.TokenProvider.ValidateAccessToken(gc, tok)
		require.Error(t, vErr, "an MCP-bound token must not double as a GraphQL/gRPC credential")

		// Assert at the ENTRY POINT gRPC, REST and GraphQL actually call, not
		// just at ValidateAccessToken. GetUserIDFromSessionOrAccessToken falls
		// back to ValidateDelegatedAccessToken whenever the first-party check
		// fails, so the real boundary is the pair. Today the fallback also
		// rejects (no `act` claim, and aud=<url>/mcp is not the bare host), but
		// asserting only the inner call would let a future relaxation of either
		// delegated rule reopen MCP-token-authenticates-GraphQL with this test
		// still green — the exact failure this test exists to prevent.
		_, vErr = ts.TokenProvider.GetUserIDFromSessionOrAccessToken(bearerGinContext(t, ts, tok))
		require.Error(t, vErr, "an MCP-bound token must not resolve to an identity on the shared entry point")
	})

	t.Run("an ordinary login token is rejected at /mcp", func(t *testing.T) {
		loginToken := mintStatefulAccessToken(t, ts, user, "")
		// Sanity: it really is a working first-party token, so the rejection
		// below is about audience and not about some unrelated defect.
		_, vErr := ts.TokenProvider.ValidateAccessToken(gc, loginToken)
		require.NoError(t, vErr)

		_, vErr = ts.TokenProvider.ValidateMCPAccessToken(gc, loginToken, resource)
		require.Error(t, vErr, "the client_id audience every login produces must not authenticate MCP")
	})

	t.Run("a token bound to a different resource server is rejected at /mcp", func(t *testing.T) {
		_, vErr := ts.TokenProvider.ValidateMCPAccessToken(gc,
			mintStatefulAccessToken(t, ts, user, "https://evil.example.com/mcp"), resource)
		require.Error(t, vErr)
	})

	t.Run("a token bound to the bare host is rejected at /mcp", func(t *testing.T) {
		// The path is part of the identifier. Accepting the bare origin would
		// mean any token minted for "this server" reached the tool surface.
		_, vErr := ts.TokenProvider.ValidateMCPAccessToken(gc,
			mintStatefulAccessToken(t, ts, user, "https://auth.example.com"), resource)
		require.Error(t, vErr)
	})

	t.Run("an empty configured resource authenticates nobody", func(t *testing.T) {
		_, vErr := ts.TokenProvider.ValidateMCPAccessToken(gc, mintStatefulAccessToken(t, ts, user, resource), "")
		require.Error(t, vErr, "a misconfigured deployment must fail closed, not accept every audience")
	})

	t.Run("a token with no live session is rejected at /mcp", func(t *testing.T) {
		// Not registered in the memory store: logout, password reset and admin
		// revoke all work by removing that entry, so this is the revocation lever.
		tok, _, mErr := ts.TokenProvider.CreateAccessToken(&token.AuthTokenConfig{
			User:        user,
			Nonce:       uuid.NewString(),
			Roles:       []string{"user"},
			LoginMethod: constants.AuthRecipeMethodBasicAuth,
			HostName:    testAuthorizerHost(ts),
			Resource:    resource,
			ExpireTime:  "30m",
		})
		require.NoError(t, mErr)
		_, vErr := ts.TokenProvider.ValidateMCPAccessToken(gc, tok, resource)
		require.Error(t, vErr)
	})

	t.Run("a garbage token is rejected at /mcp", func(t *testing.T) {
		_, vErr := ts.TokenProvider.ValidateMCPAccessToken(gc, "not-a-jwt", resource)
		require.Error(t, vErr)
	})
}

// TestMCPAccessTokenServiceAccountLiveness pins the one place the MCP validator
// is deliberately STRICTER than the first-party path.
//
// userIsRevoked resolves a token's subject as a user only, and returns "not
// revoked" when it finds nothing — so a machine token, whose `sub` is a client
// row id, survives deactivation of the service account until it expires. That is
// inherited behaviour on existing surfaces. MCP's headline callers are agents and
// service accounts, so shipping it there knowingly would be worse: the validator
// uses subjectIsLive, which resolves user-then-client and fails closed.
func TestMCPAccessTokenServiceAccountLiveness(t *testing.T) {
	cfg := getTestConfig()
	cfg.AuthorizerURL = "https://auth.example.com"
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	resource := cfg.MCPResource()
	gc := &gin.Context{Request: ts.GinContext.Request}

	client, err := ts.StorageProvider.AddClient(ctx, &schemas.Client{
		ClientID:      "svc-" + uuid.NewString(),
		Kind:          constants.ClientKindServiceAccount,
		Name:          "mcp-agent",
		AllowedScopes: "openid",
		IsActive:      true,
	})
	require.NoError(t, err)

	t.Run("an active service account reaches /mcp", func(t *testing.T) {
		_, vErr := ts.TokenProvider.ValidateMCPAccessToken(gc, mintMachineAccessToken(t, ts, client.ID, resource), resource)
		require.NoError(t, vErr)
	})

	t.Run("a deactivated service account does not", func(t *testing.T) {
		tok := mintMachineAccessToken(t, ts, client.ID, resource)
		client.IsActive = false
		_, uErr := ts.StorageProvider.UpdateClient(ctx, client)
		require.NoError(t, uErr)

		_, vErr := ts.TokenProvider.ValidateMCPAccessToken(gc, tok, resource)
		require.Error(t, vErr, "deactivating a service account must stop its live MCP tokens, not just block new ones")
	})
}
