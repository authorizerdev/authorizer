package integration_tests

import (
	"context"
	"encoding/base64"
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

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestSocialLoginResumesThroughAuthorize is the regression suite for #855.
//
// A client registered a callback on an origin that is NOT in --allowed-origins.
// /authorize and /app accept it (they check the client's registered list), but
// social login used to refuse it, because /oauth_login and /oauth_callback held
// redirect_uri to --allowed-origins and delivered the code themselves.
//
// The fix routes social login the way password login already goes: back to this
// server's /app, which replays the request to /authorize. These tests drive the
// real handler chain against a mock GitHub:
//
//	/authorize → /app → /oauth_login/github → (mock IdP) → /oauth_callback/github
//	           → /app → /authorize → client callback → /oauth/token
func TestSocialLoginResumesThroughAuthorize(t *testing.T) {
	cfg := getTestConfig()
	cfg.Env = constants.E2EEnv // enables the test-only OAuth mock override
	cfg.GithubClientID = "mock-github-client"
	cfg.GithubClientSecret = "mock-github-secret"
	// Explicit list WITHOUT the client's origin, and without this server.
	cfg.AllowedOrigins = []string{"http://localhost:3000"}

	email := "social_resume_" + uuid.NewString() + "@authorizer.dev"
	mock := newMockGithub(t, email)
	config.TestOAuthMockBaseOverride = mock.URL
	t.Cleanup(func() { config.TestOAuthMockBaseOverride = "" })

	ts := initTestSetup(t, cfg)
	host := testAuthorizerHost(ts)

	router := gin.New()
	router.LoadHTMLFiles(
		"../../web/templates/consent.tmpl",
		"../../web/templates/consent_error.tmpl",
		"../../web/templates/au_shell.tmpl",
	)
	router.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())
	router.GET("/oauth_login/:oauth_provider", ts.HttpProvider.OAuthLoginHandler())
	router.GET("/oauth_callback/:oauth_provider", ts.HttpProvider.OAuthCallbackHandler())

	do := func(t *testing.T, method, target string, body url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		var req *http.Request
		if body != nil {
			req = httptest.NewRequest(method, target, strings.NewReader(body.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		} else {
			req = httptest.NewRequest(method, target, nil)
		}
		req.Header.Set("X-Authorizer-URL", host)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	cookieNamed := func(w *httptest.ResponseRecorder, name string) *http.Cookie {
		for _, c := range w.Result().Cookies() {
			if c.Name == name && c.Value != "" {
				return c
			}
		}
		return nil
	}

	const verifier = "social-resume-verifier-long-enough-to-be-valid-0000"
	authorizeQuery := func(clientID, redirectURI, state string) string {
		q := url.Values{}
		q.Set("response_type", "code")
		q.Set("response_mode", "query")
		q.Set("client_id", clientID)
		q.Set("redirect_uri", redirectURI)
		q.Set("state", state)
		q.Set("nonce", "client-nonce-"+state)
		q.Set("scope", "openid profile email")
		q.Set("code_challenge", s256Challenge(verifier))
		q.Set("code_challenge_method", "S256")
		return q.Encode()
	}

	// socialLogin runs /authorize (no session) → /oauth_login → mock IdP →
	// /oauth_callback with redirect_uri set to the /app URL that /authorize
	// produced, which is what an updated login UI sends. It returns that /app
	// URL, the callback response, and the session cookie it set.
	socialLogin := func(t *testing.T, query string) (appURL string, cb *httptest.ResponseRecorder, session *http.Cookie) {
		t.Helper()
		w := do(t, http.MethodGet, "/authorize?"+query, nil)
		require.Equal(t, http.StatusFound, w.Code, "unauthenticated /authorize sends the user to /app: %s", w.Body.String())
		loc, err := url.Parse(w.Header().Get("Location"))
		require.NoError(t, err)
		require.Equal(t, "/app", loc.Path)
		appURL = host + loc.String()

		login := do(t, http.MethodGet, "/oauth_login/github?"+url.Values{
			"redirect_uri": {appURL},
			"state":        {loc.Query().Get("state")},
			"scope":        {"openid profile email"},
		}.Encode(), nil)
		require.Equal(t, http.StatusTemporaryRedirect, login.Code,
			"/oauth_login must accept this server's own /app as redirect_uri: %s", login.Body.String())
		idp, err := url.Parse(login.Header().Get("Location"))
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(idp.String(), mock.URL), "redirected to the (mock) provider: %s", idp)
		stateCookie := cookieNamed(login, constants.OAuthStateCookieName)
		require.NotNil(t, stateCookie, "oauth state cookie binds the flow to this browser")

		cb = do(t, http.MethodGet, "/oauth_callback/github?"+url.Values{
			"state": {idp.Query().Get("state")},
			"code":  {"mock-provider-code"},
		}.Encode(), nil, stateCookie)
		session = cookieNamed(cb, constants.AppCookieName+"_session")
		return appURL, cb, session
	}

	t.Run("#855: a client on another origin completes social login", func(t *testing.T) {
		const clientRedirect = "https://app.example.org/auth/callback"
		client, err := ts.StorageProvider.AddClient(context.Background(), &schemas.Client{
			Name:          "first-party-" + uuid.NewString(),
			Kind:          constants.ClientKindInteractive,
			RedirectURIs:  clientRedirect,
			AllowedScopes: "openid,profile,email",
			GrantTypes:    "authorization_code,refresh_token",
			IsActive:      true,
		})
		require.NoError(t, err)
		query := authorizeQuery(client.ClientID, clientRedirect, "s-855-"+uuid.NewString())

		appURL, cb, session := socialLogin(t, query)
		require.Equal(t, http.StatusFound, cb.Code, "callback: %s", cb.Body.String())
		require.NotNil(t, session, "the callback establishes the browser session")

		// The callback delivers NOTHING on the resume hop: it returns to the
		// exact /app URL, with no code and no replacement nonce appended (/app
		// forwards every parameter to /authorize, later duplicates winning).
		assert.Equal(t, appURL, cb.Header().Get("Location"),
			"the resume redirect must be the /app URL unchanged")

		// /app replays the original request; /authorize now has a session,
		// validates the client's registered redirect_uri and issues the code.
		w := do(t, http.MethodGet, "/authorize?"+query, nil, session)
		require.Equal(t, http.StatusFound, w.Code, "resumed /authorize: %s", w.Body.String())
		final, err := url.Parse(w.Header().Get("Location"))
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(final.String(), clientRedirect+"?"),
			"the code goes to the client's registered callback: %s", final)
		code := final.Query().Get("code")
		require.NotEmpty(t, code)

		tok := do(t, http.MethodPost, "/oauth/token", url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {clientRedirect},
			"client_id":     {client.ClientID},
			"client_secret": {client.ClientSecret},
			"code_verifier": {verifier},
		})
		require.Equal(t, http.StatusOK, tok.Code, "token exchange: %s", tok.Body.String())
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(tok.Body.Bytes(), &body))
		assert.NotEmpty(t, body["access_token"], "the code is redeemable end to end")

		// The client's own nonce must survive the round trip into the ID token.
		// Had the callback appended its random nonce to the /app URL, /app would
		// have forwarded that one instead and the client would reject the token.
		idToken, _ := body["id_token"].(string)
		parts := strings.Split(idToken, ".")
		require.Len(t, parts, 3, "an id_token is issued for scope openid")
		claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
		require.NoError(t, err)
		var claims map[string]interface{}
		require.NoError(t, json.Unmarshal(claimsJSON, &claims))
		assert.Equal(t, mustParseQuery(t, query).Get("nonce"), claims["nonce"],
			"id_token nonce is the one the client sent to /authorize")
	})

	t.Run("a self-registered client gets consent, never a code", func(t *testing.T) {
		const clientRedirect = "https://attacker.example.net/cb"
		client, err := ts.StorageProvider.AddClient(context.Background(), &schemas.Client{
			Name:          "self-registered-" + uuid.NewString(),
			Kind:          constants.ClientKindDynamic,
			RedirectURIs:  clientRedirect,
			AllowedScopes: "openid,profile,email",
			GrantTypes:    "authorization_code,refresh_token",
			IsActive:      true,
		})
		require.NoError(t, err)
		query := authorizeQuery(client.ClientID, clientRedirect, "s-dcr-"+uuid.NewString())

		_, cb, session := socialLogin(t, query)
		require.Equal(t, http.StatusFound, cb.Code, "callback: %s", cb.Body.String())
		require.NotNil(t, session)
		cbLoc := cb.Header().Get("Location")
		assert.False(t, strings.HasPrefix(cbLoc, clientRedirect), "the callback must never redirect to the client: %s", cbLoc)
		assert.NotContains(t, cbLoc, "code=mock", "no provider or authorization code is appended")

		// Resuming lands on the consent page, not on the client's callback.
		w := do(t, http.MethodGet, "/authorize?"+query, nil, session)
		assert.Equal(t, http.StatusOK, w.Code, "consent page is rendered: %s", w.Header().Get("Location"))
		assert.Empty(t, w.Header().Get("Location"), "no redirect to the client before consent")
		assert.Contains(t, w.Body.String(), "consent_id", "the consent form is shown")
	})

	t.Run("only /app on this server's own origin is a resume target", func(t *testing.T) {
		for _, target := range []string{
			host + "/graphql",                 // own origin, wrong path
			host + "/app/../graphql",          // path traversal spelling
			host + "/app#state=x",             // fragment
			"https://app.example.org/app",     // other origin, right path
			"https://app.example.org/auth/cb", // a client's callback: still not allowed here
		} {
			w := do(t, http.MethodGet, "/oauth_login/github?"+url.Values{
				"redirect_uri": {target}, "state": {"x"},
			}.Encode(), nil)
			assert.Equal(t, http.StatusBadRequest, w.Code, "must refuse %q", target)
		}
		// The canonical spelling a browser is actually on: the /app route
		// group 301s /app to /app/, so the login page renders at /app/?….
		for _, target := range []string{host + "/app/?client_id=c&response_type=code", host + "/app/signup?client_id=c"} {
			w := do(t, http.MethodGet, "/oauth_login/github?"+url.Values{
				"redirect_uri": {target}, "state": {"x"},
			}.Encode(), nil)
			assert.Equal(t, http.StatusTemporaryRedirect, w.Code, "must accept %q: %s", target, w.Body.String())
		}
	})

	t.Run("backward compatible: an allowed origin still gets the code directly", func(t *testing.T) {
		// What a login UI built on the current @authorizerdev/authorizer-react
		// sends: the client's own callback, on an origin in --allowed-origins.
		const legacyRedirect = "http://localhost:3000/cb"
		w := do(t, http.MethodGet, "/oauth_login/github?"+url.Values{
			"redirect_uri": {legacyRedirect}, "state": {"legacy-" + uuid.NewString()},
		}.Encode(), nil)
		require.Equal(t, http.StatusTemporaryRedirect, w.Code, w.Body.String())
		idp, err := url.Parse(w.Header().Get("Location"))
		require.NoError(t, err)
		cb := do(t, http.MethodGet, "/oauth_callback/github?"+url.Values{
			"state": {idp.Query().Get("state")}, "code": {"mock-provider-code"},
		}.Encode(), nil, cookieNamed(w, constants.OAuthStateCookieName))
		require.Equal(t, http.StatusFound, cb.Code, cb.Body.String())
		loc := cb.Header().Get("Location")
		assert.True(t, strings.HasPrefix(loc, legacyRedirect+"?"), "legacy direct redirect unchanged: %s", loc)
		assert.Contains(t, loc, "state=", "legacy parameters are still appended")
	})

	t.Run("social login from the hosted /app with no /authorize request", func(t *testing.T) {
		// A user signing in on /app directly (no client). Before this change the
		// server's own /app was accepted only under a bare "*" and came back as
		// /app?state=..&nonce=..; nothing on /app reads either. It is now
		// accepted under any allowed-origins list and comes back unchanged,
		// with the session established.
		appURL := host + "/app"
		w := do(t, http.MethodGet, "/oauth_login/github?"+url.Values{
			"redirect_uri": {appURL}, "state": {"standalone-" + uuid.NewString()},
		}.Encode(), nil)
		require.Equal(t, http.StatusTemporaryRedirect, w.Code, w.Body.String())
		idp, err := url.Parse(w.Header().Get("Location"))
		require.NoError(t, err)
		cb := do(t, http.MethodGet, "/oauth_callback/github?"+url.Values{
			"state": {idp.Query().Get("state")}, "code": {"mock-provider-code"},
		}.Encode(), nil, cookieNamed(w, constants.OAuthStateCookieName))
		require.Equal(t, http.StatusFound, cb.Code, cb.Body.String())
		assert.Equal(t, appURL, cb.Header().Get("Location"))
		assert.NotNil(t, cookieNamed(cb, constants.AppCookieName+"_session"), "the user is signed in on /app")
	})
}

func mustParseQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	v, err := url.ParseQuery(raw)
	require.NoError(t, err)
	return v
}

// newMockGithub serves the three GitHub endpoints the callback calls, at the
// root of an httptest server (config.TestOAuthMockBaseOverride replaces the
// whole base URL), returning one verified primary email.
func newMockGithub(t *testing.T, email string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, v interface{}) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"access_token": "mock-access-token", "token_type": "bearer"})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]interface{}{
			"login": "resume-user", "id": 4242, "name": "Resume User",
			"email": email, "avatar_url": "https://avatars.example.com/u/4242",
		})
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []map[string]interface{}{{"email": email, "primary": true, "verified": true}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}
