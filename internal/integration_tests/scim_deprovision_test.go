package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestDeprovisionedUserRevocation asserts the deprovision (revoked) invariant at
// the token endpoints: once a user's RevokedTimestamp is stamped (what SCIM
// active:false / account deactivation do), that user can neither renew via the
// refresh_token grant nor have a still-held access token introspect as active.
func TestDeprovisionedUserRevocation(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "scim_deprov_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)
	require.NotNil(t, signupRes)

	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
		Email:    &email,
		Password: password,
		Scope:    []string{"openid", "email", "profile", "offline_access"},
	})
	require.NoError(t, err)
	require.NotNil(t, loginRes)
	require.NotNil(t, loginRes.RefreshToken)
	require.NotNil(t, loginRes.AccessToken)

	issuer := "http://" + ts.HttpServer.Listener.Addr().String()

	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())
	router.POST("/oauth/introspect", ts.HttpProvider.IntrospectHandler())

	introspect := func(token string) bool {
		form := url.Values{}
		form.Set("token", token)
		form.Set("client_id", cfg.ClientID)
		form.Set("client_secret", cfg.ClientSecret)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/introspect", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Authorizer-URL", issuer)
		router.ServeHTTP(w, req)
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		active, _ := body["active"].(bool)
		return active
	}

	refresh := func() int {
		form := url.Values{}
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", *loginRes.RefreshToken)
		form.Set("client_id", cfg.ClientID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Authorizer-URL", issuer)
		router.ServeHTTP(w, req)
		return w.Code
	}

	// Baseline: before deprovision the token is active and refresh works.
	assert.True(t, introspect(*loginRes.AccessToken), "access token should be active before deprovision")
	require.Equal(t, http.StatusOK, refresh(), "refresh should succeed before deprovision")

	// Deprovision: stamp RevokedTimestamp (the provider-agnostic signal SCIM
	// active:false sets) and drop live sessions, mirroring the SCIM service.
	user, err := ts.StorageProvider.GetUserByID(ctx, signupRes.User.ID)
	require.NoError(t, err)
	now := time.Now().Unix()
	user.RevokedTimestamp = &now
	_, err = ts.StorageProvider.UpdateUser(ctx, user)
	require.NoError(t, err)
	require.NoError(t, ts.MemoryStoreProvider.DeleteAllUserSessions(user.ID))

	// After deprovision: refresh is rejected and introspection reports inactive.
	assert.Equal(t, http.StatusUnauthorized, refresh(), "refresh must be rejected for a revoked user")
	assert.False(t, introspect(*loginRes.AccessToken), "a revoked user's access token must introspect as inactive")
}
