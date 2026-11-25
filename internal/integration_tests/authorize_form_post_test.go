package integration_tests

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/token"
)

// formPostSessionRouter returns an /authorize router with the real response
// templates loaded plus a live browser session cookie value, so the
// authorization response is exercised end to end. Production loads every
// template via LoadHTMLGlob after registering a "json" func
// (internal/server/http_routes.go); the func is registered here too so these
// templates keep parsing if they start using it.
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
	router.SetFuncMap(template.FuncMap{
		"json": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
	})
	router.LoadHTMLFiles(
		"../../web/templates/authorize_form_post.tmpl",
		"../../web/templates/authorize_web_message.tmpl",
		"../../web/templates/au_shell.tmpl",
	)
	router.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	return router, sessionToken
}

// formPostHiddenValue returns the value of the hidden input named name in the
// rendered authorize_form_post template, or "" if it is absent.
func formPostHiddenValue(body, name string) string {
	m := regexp.MustCompile(`<input type="hidden" key="` + regexp.QuoteMeta(name) + `" value="([^"]*)"`).FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return m[1]
}

var scriptNonceRE = regexp.MustCompile(`<script nonce="([^"]+)">`)

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
			assert.NotEmpty(t, formPostHiddenValue(body, "code"),
				"the authorization code must be a non-empty hidden form field")
			assert.Equal(t, "formpost-state", formPostHiddenValue(body, "state"),
				"the state must be echoed unchanged as a hidden form field")

			// The auto-submit script only runs if its nonce is the one the
			// CSP header allows; a mismatch renders the form but browsers
			// block the submit, which is the user-visible failure in #469.
			m := scriptNonceRE.FindStringSubmatch(body)
			require.NotNil(t, m, "the auto-submit script must carry a CSP nonce")
			assert.Contains(t, w.Header().Get("Content-Security-Policy"), "'nonce-"+m[1]+"'",
				"the CSP header must allow the auto-submit script's nonce")
		})
	}
}
