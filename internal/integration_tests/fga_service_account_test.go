package integration_tests

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// fgaServiceAccountModel adds the service_account subject type alongside user,
// which is the opt-in that turns autonomous client_credentials callers into FGA
// subjects (locked decision: works only if the model declares service_account).
const fgaServiceAccountModel = `model
  schema 1.1
type user
type service_account
type document
  relations
    define viewer: [user, service_account]
    define can_view: viewer
`

// createServiceAccountWithClientID inserts an active service_account whose PUBLIC
// client_id is DISTINCT from its surrogate ID, and returns (surrogateID,
// publicClientID, plaintextSecret). The distinct client_id lets the tests prove
// the FGA subject keys on client_id, never the internal surrogate id.
func createServiceAccountWithClientID(t *testing.T, ts *testSetup, allowedScopes string) (string, string, string) {
	t.Helper()
	secret := "cc-secret-" + uuid.New().String()
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), ccClientCost)
	require.NoError(t, err)
	publicClientID := "svc-" + uuid.New().String()
	sa, err := ts.StorageProvider.AddClient(context.Background(), &schemas.Client{
		Name:          "cc-fga-sa-" + uuid.New().String(),
		Kind:          constants.ClientKindServiceAccount,
		ClientID:      publicClientID,
		ClientSecret:  string(hash),
		AllowedScopes: allowedScopes,
		IsActive:      true,
	})
	require.NoError(t, err)
	require.NotEqual(t, sa.ID, sa.ClientID, "test requires a client_id distinct from the surrogate id")
	return sa.ID, publicClientID, secret
}

// mintMachineToken runs the real client_credentials grant against /oauth/token
// and returns the issued access token. The issuer host is pinned to
// ccTestAuthorizerURL via X-Authorizer-URL so validation on the GraphQL path can
// reproduce the same iss.
func mintMachineToken(t *testing.T, tokenRouter http.Handler, publicClientID, secret, scope string) string {
	t.Helper()
	form := url.Values{}
	form.Set("grant_type", constants.GrantTypeClientCredentials)
	if scope != "" {
		form.Set("scope", scope)
	}
	w := postClientCredentials(tokenRouter, form, []string{publicClientID, secret})
	require.Equal(t, http.StatusOK, w.Code, "token issuance failed: %s", w.Body.String())
	body := decodeJSON(t, w)
	token, _ := body["access_token"].(string)
	require.NotEmpty(t, token, "must return an access_token")
	return token
}

// presentMachineToken pins the current gin request to a machine bearer token:
// it drops any session cookie and pins X-Authorizer-URL so the iss check matches
// the token minted through /oauth/token.
func presentMachineToken(ts *testSetup, token string) {
	clearCookies(ts)
	ts.GinContext.Request.Header.Set("Authorization", "Bearer "+token)
	ts.GinContext.Request.Header.Set("X-Authorizer-URL", ccTestAuthorizerURL)
}

// TestFGAServiceAccountSubject exercises machine-token subject resolution in the
// public FGA check/list path: an autonomous client_credentials caller resolves
// to service_account:<client_id> and is checked against the model/tuples, with
// the fail-closed and delegation-guard rules the locked design requires.
func TestFGAServiceAccountSubject(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	tokenRouter := gin.New()
	tokenRouter.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	// Admin installs a model that declares service_account.
	setAdminCookie(t, ts)
	modelRes, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaServiceAccountModel})
	require.NoError(t, err)
	require.NotNil(t, modelRes)

	saID, saClientID, saSecret := createServiceAccountWithClientID(t, ts, "openid")

	// Grant the service account (keyed on its PUBLIC client_id) viewer on one
	// document only. Also grant a decoy tuple keyed on the SURROGATE id to prove
	// the surrogate is never the subject.
	setAdminCookie(t, ts)
	_, err = ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
		Tuples: []*model.FgaTupleInput{
			{User: "service_account:" + saClientID, Relation: "viewer", Object: "document:sa-granted"},
			{User: "service_account:" + saID, Relation: "viewer", Object: "document:sa-surrogate-decoy"},
		},
	})
	require.NoError(t, err)

	// (a) machine token + model WITH service_account type → allow/deny per tuples.
	t.Run("machine token resolves to service_account:<client_id> and is allowed on granted object", func(t *testing.T) {
		token := mintMachineToken(t, tokenRouter, saClientID, saSecret, "")
		presentMachineToken(ts, token)

		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:sa-granted"}},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Results[0].Allowed, "machine caller must be allowed on its granted object")
	})

	t.Run("machine token denied on ungranted object", func(t *testing.T) {
		token := mintMachineToken(t, tokenRouter, saClientID, saSecret, "")
		presentMachineToken(ts, token)

		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:sa-other"}},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Results[0].Allowed, "machine caller must be denied on an ungranted object")
	})

	// Decision: reuse client_id, NEVER the surrogate id.
	t.Run("machine subject is client_id not the surrogate id", func(t *testing.T) {
		token := mintMachineToken(t, tokenRouter, saClientID, saSecret, "")
		presentMachineToken(ts, token)

		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:sa-surrogate-decoy"}},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Results[0].Allowed,
			"a tuple keyed on the surrogate id must NOT grant the machine caller (subject is client_id)")
	})

	// list_permissions for the machine caller enumerates only the service
	// account's own objects.
	t.Run("list_permissions returns only the machine caller's objects", func(t *testing.T) {
		token := mintMachineToken(t, tokenRouter, saClientID, saSecret, "")
		presentMachineToken(ts, token)

		res, err := ts.GraphQLProvider.ListPermissions(ctx, &model.ListPermissionsInput{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Contains(t, res.Objects, "document:sa-granted")
		assert.NotContains(t, res.Objects, "document:sa-surrogate-decoy",
			"the surrogate-keyed decoy must not appear for the client_id subject")
	})

	// A machine caller may self-pin its own subject explicitly, but must be
	// REJECTED when probing any other subject (it is never a super-admin).
	t.Run("machine caller may self-pin but cannot probe another subject", func(t *testing.T) {
		token := mintMachineToken(t, tokenRouter, saClientID, saSecret, "")
		presentMachineToken(ts, token)

		self := "service_account:" + saClientID
		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:sa-granted"}},
			User:   &self,
		})
		require.NoError(t, err, "self-specification must be honored")
		require.NotNil(t, res)
		assert.True(t, res.Results[0].Allowed)

		other := "user:someone-else"
		res, err = ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:sa-granted"}},
			User:   &other,
		})
		assert.Error(t, err, "machine caller must not probe another subject")
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "not authorized to query authorization for another subject")
	})

	// (c)+(d) A user/session token still resolves to user:<sub> and NEVER inherits
	// the service account's grants — even though a service_account with tuples
	// exists. A real RFC 8693 delegated token carries a user sub + an act chain
	// and no service_account login_method, so it is classified here exactly like
	// this user token: the guard is structural (see callerOwnSubject).
	t.Run("user token stays user:<sub> and does not inherit service_account grants", func(t *testing.T) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Del("Authorization")
		ts.GinContext.Request.Header.Del("X-Authorizer-URL")

		email := "fga_sa_user_" + uuid.New().String() + "@authorizer.dev"
		password := "Password@123"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		userID := loginRes.User.ID
		sessionToken := latestAppSessionCookie(ts)
		require.NotEmpty(t, sessionToken)

		// Grant THIS user viewer on a different document.
		setAdminCookie(t, ts)
		_, err = ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
			Tuples: []*model.FgaTupleInput{
				{User: "user:" + userID, Relation: "viewer", Object: "document:user-doc"},
			},
		})
		require.NoError(t, err)

		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))

		// Allowed on the user's own grant.
		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:user-doc"}},
		})
		require.NoError(t, err)
		assert.True(t, res.Results[0].Allowed, "user resolves to user:<sub> and sees its own grant")

		// Denied on the service account's grant — no leakage across subject types.
		res, err = ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:sa-granted"}},
		})
		require.NoError(t, err)
		assert.False(t, res.Results[0].Allowed,
			"a user token must never inherit a service_account's grants")
	})
}

// TestFGAServiceAccountModelWithoutType asserts the fail-closed rule: when the
// active model does NOT declare service_account, a machine caller is DENIED and
// never falls back to a user subject.
func TestFGAServiceAccountModelWithoutType(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	tokenRouter := gin.New()
	tokenRouter.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	// Model declares ONLY user — no service_account type.
	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaTestModel})
	require.NoError(t, err)

	_, saClientID, saSecret := createServiceAccountWithClientID(t, ts, "openid")

	token := mintMachineToken(t, tokenRouter, saClientID, saSecret, "")
	presentMachineToken(ts, token)

	res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
		Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:1"}},
	})
	// Fail-closed: either the engine rejects the unknown subject type (error) or
	// returns a no-match deny. Never allowed, never a user-subject fallback.
	if err != nil {
		assert.Nil(t, res, "a model without service_account must fail closed for machine callers")
	} else {
		require.NotNil(t, res)
		assert.False(t, res.Results[0].Allowed,
			"a model without service_account must deny machine callers (fail closed)")
	}
}

// TestFGAServiceAccountScopeCeilingIndependentOfFGA proves the two layers are
// independent (locked decision: allowed_scopes stays the coarse ceiling at
// issuance; FGA is additive below it):
//   - a scope outside allowed_scopes is rejected at the TOKEN endpoint, with no
//     FGA involvement;
//   - a validly-scoped token still needs a passing FGA check at the resource.
func TestFGAServiceAccountScopeCeilingIndependentOfFGA(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	tokenRouter := gin.New()
	tokenRouter.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaServiceAccountModel})
	require.NoError(t, err)

	// allowed_scopes = "read" only.
	_, saClientID, saSecret := createServiceAccountWithClientID(t, ts, "read")

	// Scope ceiling enforced at issuance — a scope outside allowed_scopes is
	// rejected by the token endpoint, independent of any FGA state.
	t.Run("scope outside allowed_scopes is rejected at the token endpoint", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", constants.GrantTypeClientCredentials)
		form.Set("scope", "write")
		w := postClientCredentials(tokenRouter, form, []string{saClientID, saSecret})
		require.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
		body := decodeJSON(t, w)
		assert.Equal(t, "invalid_scope", body["error"])
	})

	// A validly-scoped token is issued, but FGA still governs the resource: no
	// tuple → denied, even though the token is valid and in-scope.
	t.Run("valid scoped token still needs a passing FGA check", func(t *testing.T) {
		token := mintMachineToken(t, tokenRouter, saClientID, saSecret, "read")
		presentMachineToken(ts, token)

		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:scoped-doc"}},
		})
		require.NoError(t, err)
		assert.False(t, res.Results[0].Allowed, "a valid scoped token still needs an FGA grant")

		// Grant the tuple; now the SAME token passes — both layers satisfied.
		setAdminCookie(t, ts)
		_, err = ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
			Tuples: []*model.FgaTupleInput{
				{User: "service_account:" + saClientID, Relation: "viewer", Object: "document:scoped-doc"},
			},
		})
		require.NoError(t, err)
		presentMachineToken(ts, token)

		res, err = ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:scoped-doc"}},
		})
		require.NoError(t, err)
		assert.True(t, res.Results[0].Allowed, "in-scope token with an FGA grant passes both layers")
	})
}
