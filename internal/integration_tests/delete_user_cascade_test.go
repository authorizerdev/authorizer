package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestDeleteUserCascade proves a hard delete (_delete_user) removes every piece
// of state the account owned: the six user-keyed tables and its FGA grants.
//
// Before this, only sessions were cascaded. The federated-identity orphan was
// the worst of it — it points at a dead user id, jitProvisionFederatedUser fails
// closed on that branch, and the unique (org_id, issuer, subject) triple blocks
// re-provisioning, so the SSO principal is locked out permanently.
func TestDeleteUserCascade(t *testing.T) {
	cfg := getTestConfig()
	ts, eng := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "delete_cascade_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email: &email, Password: password, ConfirmPassword: password,
	})
	require.NoError(t, err)
	require.NotNil(t, signupRes.User)
	userID := signupRes.User.ID

	sp := ts.StorageProvider
	orgID := "org_" + uuid.New().String()
	issuer := "https://idp.example.com"
	subject := "upstream_" + uuid.New().String()

	require.NoError(t, sp.AddSession(ctx, &schemas.Session{UserID: userID}))
	_, err = sp.AddFederatedIdentity(ctx, &schemas.FederatedIdentity{
		OrgID: orgID, Issuer: issuer, Subject: subject, UserID: userID,
	})
	require.NoError(t, err)
	_, err = sp.AddOrgMembership(ctx, &schemas.OrgMembership{OrgID: orgID, UserID: userID, Roles: "member"})
	require.NoError(t, err)
	_, err = sp.AddAuthenticator(ctx, &schemas.Authenticator{UserID: userID, Method: "totp", Secret: "s3cret"})
	require.NoError(t, err)
	_, err = sp.AddWebauthnCredential(ctx, &schemas.WebauthnCredential{
		UserID: userID, CredentialID: "cred_" + uuid.New().String(), PublicKey: "pk", Name: "laptop",
	})
	require.NoError(t, err)
	require.NoError(t, sp.AddSessionToken(ctx, &schemas.SessionToken{UserID: userID, KeyName: "access", Token: "t"}))
	require.NoError(t, sp.AddMFASession(ctx, &schemas.MFASession{UserID: userID, KeyName: "mfa"}))

	// An FGA grant held by this user, which lives outside StorageProvider.
	setAdminCookie(t, ts)
	_, err = ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaTestModel})
	require.NoError(t, err)
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:" + userID, Relation: "viewer", Object: "document:secret"},
	}))
	clearCookies(ts)

	setAdminCookie(t, ts)
	deleteRes, err := ts.GraphQLProvider.DeleteUser(ctx, &model.DeleteUserRequest{ID: userID})
	require.NoError(t, err)
	require.NotNil(t, deleteRes)

	t.Run("federated identity is gone and the principal can be re-provisioned", func(t *testing.T) {
		_, err := sp.GetFederatedIdentity(ctx, orgID, issuer, subject)
		require.Error(t, err, "orphaned federated identity is a permanent SSO lockout")

		fresh, err := sp.AddUser(ctx, &schemas.User{
			Email:         &[]string{"reprovision_" + uuid.New().String() + "@authorizer.dev"}[0],
			SignupMethods: "basic_auth",
		})
		require.NoError(t, err)
		_, err = sp.AddFederatedIdentity(ctx, &schemas.FederatedIdentity{
			OrgID: orgID, Issuer: issuer, Subject: subject, UserID: fresh.ID,
		})
		require.NoError(t, err, "the (org, issuer, subject) triple must be free again")
	})

	t.Run("org membership, authenticator and passkey are gone", func(t *testing.T) {
		_, err := sp.GetOrgMembership(ctx, orgID, userID)
		assert.Error(t, err)
		_, err = sp.GetAuthenticatorDetailsByUserId(ctx, userID, "totp")
		assert.Error(t, err)
		creds, err := sp.ListWebauthnCredentialsByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Empty(t, creds)
	})

	t.Run("session tokens and mfa sessions are gone", func(t *testing.T) {
		_, err := sp.GetSessionTokenByUserIDAndKey(ctx, userID, "access")
		assert.Error(t, err)
		sessions, err := sp.GetAllMFASessionsByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Empty(t, sessions)
	})

	t.Run("fga tuples are gone", func(t *testing.T) {
		res, err := eng.ReadTuples(ctx, engine.ReadTuplesFilter{Object: "document:secret"})
		require.NoError(t, err)
		for _, tk := range res.Tuples {
			assert.NotEqual(t, "user:"+userID, tk.User, "deleted user must not keep holding grants")
		}
	})
}
