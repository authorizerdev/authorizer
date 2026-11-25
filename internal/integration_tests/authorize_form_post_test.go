package integration_tests

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/token"
)

// formPostSessionRouter returns an /authorize router with the real templates
// loaded (as the embedded production router has) plus a live browser session
// cookie value, so the authorization response is exercised end to end.
func formPostSessionRouter(t *testing.T, ts *testSetup) (http.Handler, string) {
	t.Helper()
	_, ctx := createContext(ts)

	email := "fp_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)

	nonce := uuid.New().String()
	sessionData, sessionToken, sessionExpiresAt, err := ts.TokenProvider.CreateSessionToken(&token.AuthTokenConfig{
		User:        user,
		Nonce:       nonce,
		Roles:       ts.Config.DefaultRoles,
		Scope:       []string{"openid"},
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)

	sessionKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		sessionKey, constants.TokenTypeSessionToken+"_"+sessionData.Nonce, sessionToken, sessionExpiresAt))

	router := gin.New()
	router.LoadHTMLFiles(
		"../../web/templates/authorize_form_post.tmpl",
		"../../web/templates/authorize_web_message.tmpl",
		"../../web/templates/au_shell.tmpl",
	)
	router.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	return router, sessionToken
}

func formPostAuthorizeGet(router http.Handler, qs url.Values, sessionToken string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/authorize?"+qs.Encode(), nil)
	req.AddCookie(&http.Cookie{Name: constants.AppCookieName + "_session", Value: sessionToken})
	router.ServeHTTP(w, req)
	return w
}

// Issue #469 reported the authorization response ignoring form_post and
// falling back to a query-string redirect. The flow on main honors it; this
// test locks the behavior in for both ways the mode can be selected: the
// response_mode request parameter and the default-authorize-response-mode
// server setting.
func TestAuthorizeFormPostDeliversAutoSubmitForm(t *testing.T) {
	for _, tc := range []struct {
		name          string
		serverDefault string
		requestMode   string
	}{
		{name: "request parameter", requestMode: "form_post"},
		{name: "server default", serverDefault: "form_post"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := getTestConfig()
			cfg.DefaultAuthorizeResponseMode = tc.serverDefault
			ts := initTestSetup(t, cfg)
			registerTestClient(t, ts, "formpost-client", "formpost-secret")

			router, sessionToken := formPostSessionRouter(t, ts)

			verifier := uuid.New().String() + uuid.New().String()
			sum := sha256.Sum256([]byte(verifier))
			challenge := base64.RawURLEncoding.EncodeToString(sum[:])

			qs := url.Values{}
			qs.Set("client_id", "formpost-client")
			qs.Set("response_type", "code")
			qs.Set("redirect_uri", "http://localhost:3000/callback")
			qs.Set("code_challenge", challenge)
			qs.Set("code_challenge_method", "S256")
			qs.Set("state", "formpost-state")
			qs.Set("scope", "openid")
			if tc.requestMode != "" {
				qs.Set("response_mode", tc.requestMode)
			}

			w := formPostAuthorizeGet(router, qs, sessionToken)

			require.Equal(t, http.StatusOK, w.Code,
				"form_post must render the auto-submit form, not a query redirect to %q",
				w.Header().Get("Location"))
			body := w.Body.String()
			assert.Contains(t, body, `name="authorize_form_post"`,
				"the form_post template must be rendered")
			assert.Contains(t, body, `action="http://localhost:3000/callback"`,
				"the form must post to the client redirect_uri")
			assert.Contains(t, body, `name="code"`,
				"the authorization code must be a hidden form field")
			assert.Contains(t, body, `name="state"`,
				"the state must be a hidden form field")
		})
	}
}
