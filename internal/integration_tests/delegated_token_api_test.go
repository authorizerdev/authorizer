package integration_tests

import (
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

// mintDelegated builds an RFC 8693 delegated access token with the given
// audience, mirroring what /oauth/token issues for the delegation grant.
func mintDelegated(t *testing.T, ts *testSetup, subject, agentID, aud string) string {
	t.Helper()
	tok, err := ts.TokenProvider.CreateDelegatedAccessToken(&token.DelegationTokenConfig{
		Subject:  subject,
		Actor:    map[string]interface{}{"sub": agentID},
		Audience: aud,
		Scope:    []string{"openid"},
		ClientID: agentID,
		HostName: testAuthorizerHost(ts),
	})
	require.NoError(t, err)
	require.NotNil(t, tok)
	return tok.Token
}

// NOTE: these are unit-level checks of the validator's negative cases. The
// AUTHORITATIVE reachability proof is TestAgentDelegatedTokenReachesAuthorizerAPI,
// which mints through the real /oauth/token endpoint. An earlier version of this
// file passed `cfg.ClientID` as the audience — a value no resource indicator can
// ever be — and so asserted a contract the system could not produce, hiding the
// fact that the feature was unreachable.
//
// TestDelegatedTokenAtAuthorizerAPI pins the security envelope of the delegated
// validation path added so an agent can ask Authorizer about its own authority.
//
// The path is weaker than first-party validation by exactly one property — it
// skips the session lookup, because delegated tokens are stateless by design —
// and stricter by one: the audience must be this server. These tests pin both
// halves, because a mistake in either direction is a real vulnerability:
// too strict and the agent feature is dead code, too loose and a token minted
// for someone else's resource server authenticates here.
func TestDelegatedTokenAtAuthorizerAPI(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)
	_ = ctx

	mkUser := func() *schemas.User {
		now := time.Now().Unix()
		u, uErr := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:           refs.NewStringRef("agent_api_" + uuid.NewString() + "@authorizer.dev"),
			EmailVerifiedAt: &now,
			SignupMethods:   constants.AuthRecipeMethodBasicAuth,
			Roles:           "user",
		})
		require.NoError(t, uErr)
		return u
	}
	user := mkUser()

	gc := &gin.Context{Request: ts.GinContext.Request}
	agentID := "agent-" + uuid.NewString()

	t.Run("a token bound to THIS server is accepted", func(t *testing.T) {
		tok := mintDelegated(t, ts, user.ID, agentID, testAuthorizerHost(ts))
		claims, err := ts.TokenProvider.ValidateDelegatedAccessToken(gc, tok)
		require.NoError(t, err, "an agent must be able to reach Authorizer with a correctly-audienced delegated token")
		assert.Equal(t, user.ID, claims["sub"])
		assert.Equal(t, agentID, token.ImmediateActor(claims),
			"the immediate actor must survive validation — it is what distinguishes agent from user")
	})

	t.Run("a RESOURCE-BOUND token is rejected", func(t *testing.T) {
		// This is the audience-confusion test. A token minted for a downstream
		// resource server must never authenticate at Authorizer's own API, or
		// the RFC 8707 resource binding is decorative.
		tok := mintDelegated(t, ts, user.ID, agentID, "https://mcp.example.com")
		_, err := ts.TokenProvider.ValidateDelegatedAccessToken(gc, tok)
		require.Error(t, err, "a token bound to another resource server must not authenticate here")
		assert.Contains(t, err.Error(), "audience")
	})

	t.Run("a NON-delegated token is rejected on this path", func(t *testing.T) {
		// The weaker path must only ever accept tokens that actually carry an
		// act chain; it must not become a way to bypass session validation for
		// ordinary access tokens.
		authToken, err := ts.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
			User:        user,
			Roles:       []string{"user"},
			Scope:       []string{"openid"},
			LoginMethod: "basic_auth",
			HostName:    testAuthorizerHost(ts),
		})
		require.NoError(t, err)
		_, err = ts.TokenProvider.ValidateDelegatedAccessToken(gc, authToken.AccessToken.Token)
		require.Error(t, err, "a first-party token must not be accepted by the delegated path")
		assert.Contains(t, err.Error(), "not a delegated token")
	})

	t.Run("a revoked user's agent is rejected", func(t *testing.T) {
		revoked := mkUser()
		now := time.Now().Unix()
		revoked.RevokedTimestamp = &now
		_, err := ts.StorageProvider.UpdateUser(ctx, revoked)
		require.NoError(t, err)

		tok := mintDelegated(t, ts, revoked.ID, agentID, testAuthorizerHost(ts))
		_, err = ts.TokenProvider.ValidateDelegatedAccessToken(gc, tok)
		require.Error(t, err, "revoking the user must stop their agents — revocation is a DB lookup, not session-based")
	})
}
