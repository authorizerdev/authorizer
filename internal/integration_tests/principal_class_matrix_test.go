package integration_tests

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// Principal-class matrix.
//
// Authorizer issues tokens to several kinds of principal, and the identity
// invariant is DIFFERENT for each. Treating them as one class is what produced
// the nOAuth takeover: the social path resolved an account by email, while the
// SSO path — correctly — never did, and nobody noticed the two had drifted.
//
//	Database        identity is the email, proven by clicking a mailed link.
//	                (signup_verification_matrix_test.go)
//	Social / OAuth  identity is the email, so the PROVIDER must attest it.
//	                (oauth_noauth_test.go + the per-provider e2e specs)
//	SSO / SAML      identity is (org, issuer, subject). Email never selects an
//	                account at all. (oauth_sso_jit_test.go)
//	M2M             there is NO user and NO email. A service account must never
//	                resolve to a user subject.                    <- here
//	A2A             `sub` stays the delegating USER; the agent rides in `act`.
//	                No user is created or altered.                <- here
//
// This file covers the two machine classes, where the invariant is an absence:
// nothing about a machine principal may look like a user identity, because
// every email-shaped check downstream would then apply to it.

// TestPrincipalMatrix_MachineTokenIsNotAUser pins that a client_credentials
// token carries no user identity.
//
// The failure this prevents is subtle: if a machine token resolved to a user
// subject, every user-scoped gate (email_verified, MFA, revocation, roles)
// would evaluate against an account that does not exist — and gates that read
// "is this verified?" tend to fail OPEN on a zero value, not closed.
func TestPrincipalMatrix_MachineTokenIsNotAUser(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	tokenRouter := gin.New()
	tokenRouter.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	serviceAccountID, publicClientID, secret := createServiceAccountWithClientID(t, ts, "openid")
	token := mintMachineToken(t, tokenRouter, publicClientID, secret, "openid")
	require.NotEmpty(t, token)

	claims, err := ts.TokenProvider.ParseJWTToken(token)
	require.NoError(t, err)

	// `sub` is the service account's own id (AuthTokenConfig.ServiceAccountID),
	// deliberately the surrogate rather than the public client_id — and, the
	// point here, never a user id.
	assert.Equal(t, serviceAccountID, claims["sub"],
		"a machine token's subject is the service account itself — there is no user behind it")
	assert.Equal(t, constants.AuthRecipeMethodServiceAccount, claims["login_method"],
		"the login method marks this as a machine principal so user-scoped paths can tell them apart")

	// No roles claim: machines have none, and an empty-but-present roles claim
	// would let a role check evaluate against a zero value.
	assert.Nil(t, claims["roles"], "machines carry no roles")

	// And no user-identity claims rode along. An `email` here would be picked up
	// by anything that reads the claim as a user address.
	for _, userClaim := range []string{"email", "email_verified", "phone_number", "phone_number_verified"} {
		assert.Nil(t, claims[userClaim],
			"a machine principal has no %s; emitting one invites a user-scoped check to act on it", userClaim)
	}

	assert.Equal(t, constants.TokenTypeAccessToken, claims["token_type"])

	// And nothing was provisioned on its behalf: a service account is a client
	// registry row, never a user row.
	_, ctx := createContext(ts)
	_, err = ts.StorageProvider.GetUserByEmail(ctx, publicClientID+"@service-account.invalid")
	assert.Error(t, err, "minting a machine token must not create a user account")
}

// TestPrincipalMatrix_DelegatedTokenKeepsUserAsSubject pins the A2A invariant:
// an agent acting for a user does NOT become the user, and does not become a
// principal of its own either. `sub` stays the user so every user-scoped check
// still applies; the agent is recorded in `act` for audit and for the FGA
// intersection.
//
// Getting this backwards in either direction is a real vulnerability: `sub` =
// agent silently drops every user-scoped gate, and dropping `act` loses the
// agent half of perms(agent) ∩ perms(user) — the Confused Deputy.
func TestPrincipalMatrix_DelegatedTokenKeepsUserAsSubject(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	tokenRouter := gin.New()
	tokenRouter.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	// Returns (token, agentClientID, userID) — in that order.
	delegated, agentClientID, userID := mintDelegatedViaEndpoint(t, ts, tokenRouter, testAuthorizerHost(ts))
	require.NotEmpty(t, delegated)

	claims, err := ts.TokenProvider.ParseJWTToken(delegated)
	require.NoError(t, err)

	assert.Equal(t, userID, claims["sub"],
		"the delegating user remains the subject — an agent acting for a user must not become the user")

	act, ok := claims["act"].(map[string]any)
	require.True(t, ok, "the immediate actor must be recorded in `act`, or the agent half of the FGA intersection has nothing to evaluate")
	assert.Equal(t, agentClientID, act["sub"], "the actor is the agent's client id")

	// The user account is untouched: delegation grants authority, it does not
	// provision or modify identities.
	_, ctx := createContext(ts)
	user, err := ts.StorageProvider.GetUserByID(ctx, userID)
	require.NoError(t, err)
	assert.NotEmpty(t, refs.StringValue(user.Email), "the delegating user still exists unchanged")

	// The agent must NOT have become a user of its own along the way.
	_, err = ts.StorageProvider.GetUserByEmail(ctx, agentClientID+"@service-account.invalid")
	assert.Error(t, err, "an agent is a client, never a user")
}
