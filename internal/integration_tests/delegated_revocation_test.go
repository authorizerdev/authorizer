package integration_tests

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
)

// TestDelegatedTokenEmptyAudienceIsRejected pins that an empty `aud` can never
// authenticate.
//
// The original gate compared the audience against Config.ClientID. An unset
// --client-id defaults to "", so a token carrying aud:"" compared equal and was
// accepted — ValidateAccessToken guarded `aud != "" &&` while the delegated
// path did not. Startup already refuses an empty --client-id
// (cmd/root.go: "client secret missing in rootArgs" / client ID missing), so
// that exact configuration cannot run, but the audience gate must not depend on
// that for its safety. sameAudience now requires BOTH sides non-empty.
func TestDelegatedTokenEmptyAudienceIsRejected(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	createContext(ts)
	gc := &gin.Context{Request: ts.GinContext.Request}

	user, err := ts.StorageProvider.AddUser(context.Background(), &schemas.User{
		Email:         refsString("empty_aud_" + uuid.NewString() + "@authorizer.dev"),
		SignupMethods: constants.AuthRecipeMethodBasicAuth,
		Roles:         "user",
	})
	require.NoError(t, err)

	tok, err := ts.TokenProvider.CreateDelegatedAccessToken(&token.DelegationTokenConfig{
		Subject:  user.ID,
		Actor:    map[string]interface{}{"sub": "agent-x"},
		Audience: "", // no audience at all
		Scope:    []string{"openid"},
		ClientID: "agent-x",
		HostName: testAuthorizerHost(ts),
	})
	require.NoError(t, err)

	_, vErr := ts.TokenProvider.ValidateDelegatedAccessToken(gc, tok.Token)
	require.Error(t, vErr,
		"an empty audience must never authenticate, even when --client-id is unset")
}

// TestDelegatedTokenForDeactivatedServiceAccountIsRejected pins liveness for
// MULTI-HOP chains.
//
// Token exchange permits a service account to be the SUBJECT (agent A delegating
// to agent B). The liveness check only ever looked the subject up as a USER, so
// for a service-account subject it found nothing, returned "not revoked", and
// the delegation kept working for the token's full lifetime after the service
// account had been deactivated.
func TestDelegatedTokenForDeactivatedServiceAccountIsRejected(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	createContext(ts)
	gc := &gin.Context{Request: ts.GinContext.Request}
	ctx := context.Background()

	secret := "sa-secret-" + uuid.NewString()
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	require.NoError(t, err)
	sa, err := ts.StorageProvider.AddClient(ctx, &schemas.Client{
		Name:          "hop-sa-" + uuid.NewString(),
		Kind:          constants.ClientKindServiceAccount,
		ClientSecret:  string(hash),
		AllowedScopes: "openid",
		IsActive:      true,
	})
	require.NoError(t, err)

	mint := func() string {
		tok, mErr := ts.TokenProvider.CreateDelegatedAccessToken(&token.DelegationTokenConfig{
			Subject:  sa.ID, // the SUBJECT is a service account, not a user
			Actor:    map[string]interface{}{"sub": "downstream-agent"},
			Audience: testAuthorizerHost(ts),
			Scope:    []string{"openid"},
			ClientID: "downstream-agent",
			HostName: testAuthorizerHost(ts),
		})
		require.NoError(t, mErr)
		return tok.Token
	}

	t.Run("active service-account subject is accepted", func(t *testing.T) {
		_, vErr := ts.TokenProvider.ValidateDelegatedAccessToken(gc, mint())
		require.NoError(t, vErr, "an active service-account subject must still work")
	})

	t.Run("deactivated service-account subject is rejected", func(t *testing.T) {
		sa.IsActive = false
		_, uErr := ts.StorageProvider.UpdateClient(ctx, sa)
		require.NoError(t, uErr)

		_, vErr := ts.TokenProvider.ValidateDelegatedAccessToken(gc, mint())
		assert.Error(t, vErr,
			"deactivating a service account must stop delegations that name it as the subject; "+
				"otherwise the chain outlives the deactivation for the token's full TTL")
	})
}

func refsString(s string) *string { return &s }
