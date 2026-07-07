package integration_tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

// ccTestAuthorizerURL pins the issuer host for both token issuance and
// validation so the round-trip iss check is deterministic (parsers.GetHost
// honours X-Authorizer-URL). Real service-account secrets are hashed at bcrypt
// cost 12 (see admin_service_accounts.go serviceAccountSecretCost).
const (
	ccTestAuthorizerURL   = "http://localhost:8080"
	ccServiceAccountCost  = 12
	ccServiceAccountScope = "read,write"
)

// createTestServiceAccount inserts a service account with a known plaintext
// secret and returns (id, plaintextSecret). Accounts default to active; pass
// active=false to persist an inactive one (via Save, since GORM's default:true
// column default would otherwise flip a Create-time false back to true).
func createTestServiceAccount(t *testing.T, ts *testSetup, allowedScopes string, active bool) (string, string) {
	t.Helper()
	secret := "cc-secret-" + uuid.New().String()
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), ccServiceAccountCost)
	require.NoError(t, err)
	sa, err := ts.StorageProvider.AddServiceAccount(context.Background(), &schemas.ServiceAccount{
		Name:          "cc-sa-" + uuid.New().String(),
		ClientSecret:  string(hash),
		AllowedScopes: allowedScopes,
		IsActive:      true,
	})
	require.NoError(t, err)
	if !active {
		sa.IsActive = false
		_, err = ts.StorageProvider.UpdateServiceAccount(context.Background(), sa)
		require.NoError(t, err)
	}
	return sa.ID, secret
}

// postClientCredentials POSTs a form to /oauth/token. If basicAuth is non-nil
// ({clientID, clientSecret}) the credentials go in an HTTP Basic header instead
// of the form body.
func postClientCredentials(router http.Handler, form url.Values, basicAuth []string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Authorizer-URL", ccTestAuthorizerURL)
	if basicAuth != nil {
		req.SetBasicAuth(basicAuth[0], basicAuth[1])
	}
	router.ServeHTTP(w, req)
	return w
}

func decodeJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestClientCredentialsGrant(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	t.Run("happy_path_basic_auth", func(t *testing.T) {
		id, secret := createTestServiceAccount(t, ts, ccServiceAccountScope, true)
		form := url.Values{}
		form.Set("grant_type", constants.GrantTypeClientCredentials)

		w := postClientCredentials(router, form, []string{id, secret})
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		body := decodeJSON(t, w)
		assert.NotEmpty(t, body["access_token"], "must return an access_token")
		assert.Equal(t, "Bearer", body["token_type"])
		assert.EqualValues(t, "read write", body["scope"], "omitted scope grants full allowed set")
		assert.NotZero(t, body["expires_in"])
		// RFC 6749 §5.1: machine tokens carry no refresh_token / id_token.
		assert.Nil(t, body["refresh_token"], "client_credentials MUST NOT return refresh_token")
		assert.Nil(t, body["id_token"], "client_credentials MUST NOT return id_token")
		assert.Nil(t, body["roles"], "machine tokens have no roles")
	})

	t.Run("happy_path_form_body_credentials", func(t *testing.T) {
		id, secret := createTestServiceAccount(t, ts, ccServiceAccountScope, true)
		form := url.Values{}
		form.Set("grant_type", constants.GrantTypeClientCredentials)
		form.Set("client_id", id)
		form.Set("client_secret", secret)

		w := postClientCredentials(router, form, nil)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		body := decodeJSON(t, w)
		assert.NotEmpty(t, body["access_token"])
		assert.Equal(t, "Bearer", body["token_type"])
	})

	t.Run("requested_scope_subset_is_granted", func(t *testing.T) {
		id, secret := createTestServiceAccount(t, ts, "read,write,admin", true)
		form := url.Values{}
		form.Set("grant_type", constants.GrantTypeClientCredentials)
		form.Set("scope", "read write")

		w := postClientCredentials(router, form, []string{id, secret})
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
		body := decodeJSON(t, w)
		assert.Equal(t, "read write", body["scope"], "granted scope echoes the requested subset")
	})

	t.Run("scope_exceeding_allowed_is_rejected", func(t *testing.T) {
		id, secret := createTestServiceAccount(t, ts, "read", true)
		form := url.Values{}
		form.Set("grant_type", constants.GrantTypeClientCredentials)
		form.Set("scope", "read write")

		w := postClientCredentials(router, form, []string{id, secret})
		require.Equal(t, http.StatusBadRequest, w.Code)
		body := decodeJSON(t, w)
		assert.Equal(t, "invalid_scope", body["error"],
			"RFC 6749 §5.2: unauthorized scope MUST return invalid_scope")
	})

	t.Run("wrong_secret_returns_invalid_client", func(t *testing.T) {
		id, _ := createTestServiceAccount(t, ts, ccServiceAccountScope, true)
		form := url.Values{}
		form.Set("grant_type", constants.GrantTypeClientCredentials)
		form.Set("client_id", id)
		form.Set("client_secret", "totally-wrong-secret")

		w := postClientCredentials(router, form, nil)
		require.Equal(t, http.StatusBadRequest, w.Code)
		body := decodeJSON(t, w)
		assert.Equal(t, "invalid_client", body["error"])
	})

	t.Run("unknown_client_indistinguishable_from_wrong_secret", func(t *testing.T) {
		// Wrong secret against a real account.
		id, _ := createTestServiceAccount(t, ts, ccServiceAccountScope, true)
		wrongSecretForm := url.Values{}
		wrongSecretForm.Set("grant_type", constants.GrantTypeClientCredentials)
		wrongSecretForm.Set("client_id", id)
		wrongSecretForm.Set("client_secret", "totally-wrong-secret")
		wrongSecretResp := postClientCredentials(router, wrongSecretForm, nil)

		// Unknown client_id entirely.
		unknownForm := url.Values{}
		unknownForm.Set("grant_type", constants.GrantTypeClientCredentials)
		unknownForm.Set("client_id", uuid.New().String())
		unknownForm.Set("client_secret", "totally-wrong-secret")
		unknownResp := postClientCredentials(router, unknownForm, nil)

		assert.Equal(t, wrongSecretResp.Code, unknownResp.Code,
			"unknown client and wrong secret MUST return the same status")
		assert.JSONEq(t, wrongSecretResp.Body.String(), unknownResp.Body.String(),
			"unknown client and wrong secret MUST return an identical error body")
	})

	t.Run("inactive_service_account_returns_invalid_client", func(t *testing.T) {
		id, secret := createTestServiceAccount(t, ts, ccServiceAccountScope, false)
		form := url.Values{}
		form.Set("grant_type", constants.GrantTypeClientCredentials)
		form.Set("client_id", id)
		form.Set("client_secret", secret)

		w := postClientCredentials(router, form, nil)
		require.Equal(t, http.StatusBadRequest, w.Code)
		body := decodeJSON(t, w)
		assert.Equal(t, "invalid_client", body["error"],
			"an inactive account (even with the correct secret) MUST return invalid_client")
	})

	t.Run("invalid_client_via_basic_auth_returns_401", func(t *testing.T) {
		id, _ := createTestServiceAccount(t, ts, ccServiceAccountScope, true)
		form := url.Values{}
		form.Set("grant_type", constants.GrantTypeClientCredentials)

		w := postClientCredentials(router, form, []string{id, "wrong-secret"})
		require.Equal(t, http.StatusUnauthorized, w.Code,
			"RFC 6749 §5.2: client auth failure via Basic MUST return 401")
		assert.NotEmpty(t, w.Header().Get("WWW-Authenticate"))
	})

	t.Run("issued_token_validates_and_round_trips", func(t *testing.T) {
		id, secret := createTestServiceAccount(t, ts, "read,write", true)
		form := url.Values{}
		form.Set("grant_type", constants.GrantTypeClientCredentials)
		form.Set("scope", "read")

		w := postClientCredentials(router, form, []string{id, secret})
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
		accessToken, _ := decodeJSON(t, w)["access_token"].(string)
		require.NotEmpty(t, accessToken)

		// Prove the memory-store registration is correct: a token that was never
		// registered would be rejected here even with a valid signature.
		validateReq, _ := http.NewRequest(http.MethodGet, "/", nil)
		validateReq.Header.Set("X-Authorizer-URL", ccTestAuthorizerURL)
		gctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		gctx.Request = validateReq

		claims, err := ts.TokenProvider.ValidateAccessToken(gctx, accessToken)
		require.NoError(t, err, "issued machine token must validate downstream")
		assert.Equal(t, id, claims["sub"], "sub must be the service account id")
		assert.Equal(t, constants.TokenTypeAccessToken, claims["token_type"])
		assert.Equal(t, constants.AuthRecipeMethodServiceAccount, claims["login_method"])
		// Machines carry no roles/allowed_roles.
		assert.Nil(t, claims["roles"])
		assert.Nil(t, claims["allowed_roles"])
	})

	t.Run("missing_client_id_returns_invalid_request", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", constants.GrantTypeClientCredentials)

		w := postClientCredentials(router, form, nil)
		require.Equal(t, http.StatusBadRequest, w.Code)
		body := decodeJSON(t, w)
		assert.Equal(t, "invalid_request", body["error"])
	})

	// True end-to-end: unlike every subtest above (which fabricates a
	// ServiceAccount + bcrypt hash directly via storage), this goes through
	// the real admin API — the same path an operator actually uses — and
	// proves the client_id/client_secret it hands back are genuinely usable
	// at /oauth/token, not just internally-consistent test fixtures.
	t.Run("admin_created_service_account_authenticates_end_to_end", func(t *testing.T) {
		_, adminCtx := createContext(ts)
		setAdminCookie(t, ts)

		created, err := ts.GraphQLProvider.CreateServiceAccount(adminCtx, &model.CreateServiceAccountRequest{
			Name:          "e2e-worker-" + uuid.New().String(),
			AllowedScopes: []string{"read", "write"},
		})
		require.NoError(t, err)
		require.NotNil(t, created)
		require.NotEmpty(t, created.ClientSecret, "create must return the plaintext secret exactly once")

		clientID := created.ServiceAccount.ID
		clientSecret := created.ClientSecret

		form := url.Values{}
		form.Set("grant_type", constants.GrantTypeClientCredentials)
		form.Set("scope", "read")

		w := postClientCredentials(router, form, []string{clientID, clientSecret})
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		body := decodeJSON(t, w)
		accessToken, _ := body["access_token"].(string)
		require.NotEmpty(t, accessToken)
		assert.Equal(t, "read", body["scope"])

		// The issued token must actually validate downstream, and the fetched
		// ServiceAccount (via the same admin API) must never expose the secret.
		validateReq, _ := http.NewRequest(http.MethodGet, "/", nil)
		validateReq.Header.Set("X-Authorizer-URL", ccTestAuthorizerURL)
		gctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		gctx.Request = validateReq
		claims, err := ts.TokenProvider.ValidateAccessToken(gctx, accessToken)
		require.NoError(t, err)
		assert.Equal(t, clientID, claims["sub"])

		fetched, err := ts.GraphQLProvider.ServiceAccount(adminCtx, &model.ServiceAccountRequest{ID: clientID})
		require.NoError(t, err)
		assert.Equal(t, created.ServiceAccount.Name, fetched.Name)
	})
}
