package integration_tests

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/authctx"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestDeactivatingAServiceAccountStopsItsLiveTokens pins the revocation control
// for machine identities on Authorizer's OWN surfaces — GraphQL, gRPC and REST,
// all of which resolve the caller through ValidateAccessToken.
//
// The check used to resolve a token's subject as a USER only and treat "not
// found" as "not revoked". A client_credentials token's `sub` is the service
// account's row id (schemas.Client.ID), never a user id, so the lookup missed
// every time and reported the caller live. Deactivating a service account
// therefore blocked new token issuance and did nothing whatsoever to the tokens
// already outstanding: an operator revoking a compromised machine identity got a
// success response and a credential that kept working until it expired.
//
// This is the test that would have caught it. It asserts at
// GetUserIDFromSessionOrAccessToken as well as at ValidateAccessToken, because
// the former is what the gRPC interceptor and the GraphQL resolvers actually
// call, and it falls back to the delegated validator when the first-party check
// fails — a boundary that has to hold at the entry point, not one layer below it.
func TestDeactivatingAServiceAccountStopsItsLiveTokens(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	client, err := ts.StorageProvider.AddClient(ctx, &schemas.Client{
		ClientID:      "svc-" + uuid.NewString(),
		Kind:          constants.ClientKindServiceAccount,
		Name:          "worker",
		AllowedScopes: "openid",
		IsActive:      true,
	})
	require.NoError(t, err)

	bearerContext := func(tok string) *gin.Context { return bearerGinContext(t, ts, tok) }

	t.Run("an active service account's token is accepted", func(t *testing.T) {
		tok := mintMachineAccessToken(t, ts, client.ID, "")
		claims, vErr := ts.TokenProvider.ValidateAccessToken(bearerContext(tok), tok)
		require.NoError(t, vErr, "an active service account must keep working — this is the guard against fixing the bug by breaking the feature")
		assert.Equal(t, client.ID, claims["sub"])

		data, rErr := ts.TokenProvider.GetUserIDFromSessionOrAccessToken(bearerContext(tok))
		require.NoError(t, rErr)
		assert.Equal(t, client.ID, data.UserID)
	})

	t.Run("deactivating the account rejects tokens already issued", func(t *testing.T) {
		tok := mintMachineAccessToken(t, ts, client.ID, "")

		client.IsActive = false
		_, uErr := ts.StorageProvider.UpdateClient(ctx, client)
		require.NoError(t, uErr)

		_, vErr := ts.TokenProvider.ValidateAccessToken(bearerContext(tok), tok)
		require.Error(t, vErr, "deactivation must stop live tokens, not merely block new issuance")

		_, rErr := ts.TokenProvider.GetUserIDFromSessionOrAccessToken(bearerContext(tok))
		require.Error(t, rErr, "the entry point gRPC and GraphQL actually call must reject it too")
	})
}

// TestSubjectLivenessFailsClosedForAnUnknownSubject covers the second half of the
// same change: a subject that resolves to neither a user nor a client.
//
// The old rule failed OPEN here — "no user row, so nothing says they are revoked".
// It matters as a fallback rather than as a live hole: DeleteUser purges the
// caller's sessions, but does so through asyncutil.Go on a best-effort basis
// (admin_users.go), so a failed or racing purge left a token whose subject no
// longer exists in any table still authenticating.
func TestSubjectLivenessFailsClosedForAnUnknownSubject(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:         refs.NewStringRef("liveness_" + uuid.NewString() + "@authorizer.dev"),
		SignupMethods: constants.AuthRecipeMethodBasicAuth,
		Roles:         "user",
	})
	require.NoError(t, err)

	tok := mintStatefulAccessToken(t, ts, user, "")
	gc := bearerGinContext(t, ts, tok)

	// Sanity: the token works while the subject exists, so the rejection below is
	// about the deletion and not about some unrelated defect in the fixture.
	_, vErr := ts.TokenProvider.ValidateAccessToken(gc, tok)
	require.NoError(t, vErr)

	// Delete the row directly, leaving the session entry behind — the state a
	// failed async purge produces.
	require.NoError(t, ts.StorageProvider.DeleteUser(ctx, user))

	_, vErr = ts.TokenProvider.ValidateAccessToken(gc, tok)
	require.Error(t, vErr, "a subject that resolves to neither a user nor a client must not authenticate")
}

// adminSvc exposes the admin half of the service provider. The concrete value
// implements both interfaces; service.Provider only declares the public one.
func adminSvc(t *testing.T, ts *testSetup) service.AdminProvider {
	t.Helper()
	admin, ok := ts.ServiceProvider.(service.AdminProvider)
	require.True(t, ok, "the service provider must also implement AdminProvider")
	return admin
}

// TestDeactivationPurgesServiceAccountSessions pins the PRIMARY revocation
// mechanism for machine identities, and it exists because of a gap the liveness
// work exposed rather than created.
//
// Token validation's subject-liveness check is deliberately defense-in-depth: it
// tolerates an unanswerable lookup, because it is the shared core for GraphQL,
// gRPC and REST and turning a database outage into "subject not active" would
// 401 every authenticated request at once. That tolerance is only safe when
// something else is the primary revocation mechanism — for users it is the
// memory-store session delete, which lives in a different system and survives a
// database outage.
//
// Service accounts had no such mechanism. UpdateClient set IsActive=false and
// returned; nothing touched the memory store. So the DB flag was not
// defense-in-depth for machine tokens, it was the entire revocation story, and a
// storage outage would have re-opened it.
//
// Asserting on the session store directly, not on validation: the point is that
// the credential is destroyed at its source, so revocation no longer depends on
// a lookup being reachable at request time.
func TestDeactivationPurgesServiceAccountSessions(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	client, err := ts.StorageProvider.AddClient(ctx, &schemas.Client{
		ClientID:      "svc-" + uuid.NewString(),
		Kind:          constants.ClientKindServiceAccount,
		Name:          "purge-me",
		AllowedScopes: "openid",
		IsActive:      true,
	})
	require.NoError(t, err)

	nonce := uuid.NewString()
	sessionKey := constants.AuthRecipeMethodServiceAccount + ":" + client.ID
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		sessionKey, constants.TokenTypeAccessToken+"_"+nonce, "live-machine-token", time.Now().Add(time.Hour).Unix()))

	_, gErr := ts.MemoryStoreProvider.GetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+nonce)
	require.NoError(t, gErr, "fixture must start with a live session, or the assertion below proves nothing")

	active := false
	// Super-admin via the principal, not via an admin cookie: requireSuperAdmin
	// takes the principal branch first, and its fallback dereferences
	// meta.Request without a nil guard.
	adminCtx := authctx.WithPrincipal(ctx, &authctx.Principal{IsSuperAdmin: true})
	_, _, uErr := adminSvc(t, ts).UpdateClient(adminCtx, service.RequestMetadata{}, &model.UpdateClientRequest{
		ID:       client.ID,
		IsActive: &active,
	})
	require.NoError(t, uErr)

	_, gErr = ts.MemoryStoreProvider.GetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+nonce)
	require.Error(t, gErr, "deactivating a service account must destroy its live sessions, not merely set a flag a later lookup might not reach")
}
