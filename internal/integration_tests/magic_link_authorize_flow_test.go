package integration_tests

import (
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
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

const magicLinkCallback = "http://localhost:3000/callback"

func magicLinkTestConfig() *config.Config {
	cfg := getTestConfig()
	cfg.EnableMagicLinkLogin = true
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	return cfg
}

// magicLinkRouter mounts the three endpoints a magic link started from
// /authorize travels through.
func magicLinkRouter(ts *testSetup) *gin.Engine {
	r := gin.New()
	r.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	r.GET("/verify_email", ts.HttpProvider.VerifyEmailHandler())
	r.POST("/oauth/token", ts.HttpProvider.TokenHandler())
	return r
}

func serveGET(ts *testSetup, r *gin.Engine, target string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))
	r.ServeHTTP(w, req)
	return w
}

// clickLink follows the emailed link as a browser would, against the router.
// A browser percent-encodes a literal space before sending the request, so
// that is the only normalisation applied.
func clickLink(t *testing.T, ts *testSetup, r *gin.Engine, link string) *httptest.ResponseRecorder {
	t.Helper()
	u, err := url.Parse(link)
	require.NoError(t, err)
	return serveGET(ts, r, strings.ReplaceAll(u.RequestURI(), " ", "%20"))
}

func redeemCode(ts *testSetup, r *gin.Engine, code, verifier, redirectURI string) (int, map[string]interface{}) {
	f := url.Values{}
	f.Set("grant_type", "authorization_code")
	f.Set("code", code)
	f.Set("client_id", ts.Config.ClientID)
	f.Set("redirect_uri", redirectURI)
	if verifier != "" {
		f.Set("code_verifier", verifier)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(f.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))
	r.ServeHTTP(w, req)
	body := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body
}

// authorizeAndClickMagicLink runs /authorize (PKCE, no session) →
// magic_link_login with the state the login UI received and the given
// redirect → clicks the emailed link. It returns the state and the raw
// /verify_email response.
func authorizeAndClickMagicLink(t *testing.T, ts *testSetup, r *gin.Engine, verifier, magicRedirect string) (string, *httptest.ResponseRecorder) {
	t.Helper()
	_, ctx := createContext(ts)

	state := uuid.NewString()
	qs := url.Values{}
	qs.Set("response_type", "code")
	qs.Set("response_mode", "query")
	qs.Set("client_id", ts.Config.ClientID)
	qs.Set("redirect_uri", magicLinkCallback)
	qs.Set("state", state)
	qs.Set("scope", "openid profile email")
	qs.Set("code_challenge", s256Challenge(verifier))
	qs.Set("code_challenge_method", "S256")
	aw := serveGET(ts, r, "/authorize?"+qs.Encode())
	require.Equal(t, http.StatusFound, aw.Code, "authorize: %s", aw.Body.String())
	loginUI, err := url.Parse(aw.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "/app", loginUI.Path, "an unauthenticated /authorize detours through the login UI")
	require.Equal(t, state, loginUI.Query().Get("state"))

	email := "magic_authz_" + uuid.NewString() + "@authorizer.dev"
	_, err = ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{
		Email:       email,
		State:       refs.NewStringRef(state),
		Scope:       []string{"openid", "profile", "email"},
		RedirectURI: refs.NewStringRef(magicRedirect),
	})
	require.NoError(t, err)

	link := ts.SentEmails.waitForVerificationURL(t, email, constants.VerificationTypeMagicLinkLogin)
	return state, clickLink(t, ts, r, link)
}

// startAuthorizeMagicLink is authorizeAndClickMagicLink for the happy path,
// returning the parsed /verify_email redirect.
func startAuthorizeMagicLink(t *testing.T, ts *testSetup, r *gin.Engine, verifier string) *url.URL {
	t.Helper()
	state, vw := authorizeAndClickMagicLink(t, ts, r, verifier, magicLinkCallback)
	require.Equal(t, http.StatusTemporaryRedirect, vw.Code, "verify_email: %s", vw.Body.String())
	loc, err := url.Parse(vw.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, magicLinkCallback, loc.Scheme+"://"+loc.Host+loc.Path)
	require.Equal(t, state, loc.Query().Get("state"))
	return loc
}

// TestMagicLinkAuthorizeCodeIsRedeemable pins that a magic link started from a
// PKCE /authorize request yields a `code` that /oauth/token accepts — exactly
// like a code minted after password login — and that it is bound to the PKCE
// verifier and single-use.
//
// Previously /verify_email put the pre-generated code in the redirect but never
// stored the code state (the SetState was commented out), so every redemption
// failed with invalid_grant.
func TestMagicLinkAuthorizeCodeIsRedeemable(t *testing.T) {
	ts := initTestSetup(t, magicLinkTestConfig())
	r := magicLinkRouter(ts)
	const verifier = "magic-link-verifier-long-enough-to-be-valid-0000000"

	t.Run("correct verifier redeems once", func(t *testing.T) {
		loc := startAuthorizeMagicLink(t, ts, r, verifier)
		code := loc.Query().Get("code")
		require.NotEmpty(t, code, "an authorize-context magic link must deliver a code")

		status, body := redeemCode(ts, r, code, verifier, magicLinkCallback)
		require.Equal(t, http.StatusOK, status, "token exchange: %v", body)
		assert.NotEmpty(t, body["access_token"])
		assert.NotEmpty(t, body["id_token"])

		status, body = redeemCode(ts, r, code, verifier, magicLinkCallback)
		assert.Equal(t, http.StatusBadRequest, status, "a code must be single-use: %v", body)
		assert.Equal(t, "invalid_grant", body["error"])
	})

	t.Run("wrong verifier is refused", func(t *testing.T) {
		loc := startAuthorizeMagicLink(t, ts, r, verifier)
		code := loc.Query().Get("code")
		require.NotEmpty(t, code)

		status, body := redeemCode(ts, r, code, "an-entirely-different-verifier-000000000000000000", magicLinkCallback)
		assert.Equal(t, http.StatusBadRequest, status, "%v", body)
		assert.Equal(t, "invalid_grant", body["error"])
		assert.Empty(t, body["access_token"])
	})

	t.Run("missing verifier is refused", func(t *testing.T) {
		loc := startAuthorizeMagicLink(t, ts, r, verifier)
		code := loc.Query().Get("code")
		require.NotEmpty(t, code)

		status, body := redeemCode(ts, r, code, "", magicLinkCallback)
		assert.Equal(t, http.StatusBadRequest, status, "%v", body)
		assert.Empty(t, body["access_token"])
	})
}

// TestMagicLinkWithoutAuthorizeContextIssuesNoCode pins that /verify_email
// does not invent a code when the magic link carries no /authorize state.
func TestMagicLinkWithoutAuthorizeContextIssuesNoCode(t *testing.T) {
	ts := initTestSetup(t, magicLinkTestConfig())
	r := magicLinkRouter(ts)
	_, ctx := createContext(ts)

	email := "magic_noauthz_" + uuid.NewString() + "@authorizer.dev"
	_, err := ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{
		Email:       email,
		State:       refs.NewStringRef(uuid.NewString()), // never stored by /authorize
		RedirectURI: refs.NewStringRef(magicLinkCallback),
	})
	require.NoError(t, err)

	vw := clickLink(t, ts, r, ts.SentEmails.waitForVerificationURL(t, email, constants.VerificationTypeMagicLinkLogin))
	require.Equal(t, http.StatusTemporaryRedirect, vw.Code, vw.Body.String())
	loc, err := url.Parse(vw.Header().Get("Location"))
	require.NoError(t, err)
	assert.Empty(t, loc.Query().Get("code"))
}

// TestMagicLinkWithExactRedirectURIs pins that a magic link works when the
// deployment declares --redirect-uris. The link used to concatenate
// redirect_uri raw, and magic_link_login appended ?roles=&state=&scope= to the
// redirect itself, so /verify_email saw "<uri>?roles=..." — no longer an exact
// match — and answered 400 "invalid redirect uri".
func TestMagicLinkWithExactRedirectURIs(t *testing.T) {
	cases := []struct {
		name     string
		redirect string
	}{
		{"plain URI", magicLinkCallback},
		{"URI with its own query string", magicLinkCallback + "?tenant=acme&x=a%26b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := magicLinkTestConfig()
			cfg.RedirectURIs = []string{tc.redirect}
			ts := initTestSetup(t, cfg)
			r := magicLinkRouter(ts)
			_, ctx := createContext(ts)

			email := "magic_exact_" + uuid.NewString() + "@authorizer.dev"
			state := uuid.NewString()
			_, err := ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{
				Email:       email,
				State:       refs.NewStringRef(state),
				Scope:       []string{"openid", "email"},
				RedirectURI: refs.NewStringRef(tc.redirect),
			})
			require.NoError(t, err)

			link := ts.SentEmails.waitForVerificationURL(t, email, constants.VerificationTypeMagicLinkLogin)
			lu, err := url.Parse(link)
			require.NoError(t, err)
			assert.Equal(t, tc.redirect, lu.Query().Get("redirect_uri"),
				"redirect_uri must survive the link's own query parsing intact")

			vw := clickLink(t, ts, r, link)
			require.Equal(t, http.StatusTemporaryRedirect, vw.Code, "verify_email: %s", vw.Body.String())

			loc, err := url.Parse(vw.Header().Get("Location"))
			require.NoError(t, err)
			want, err := url.Parse(tc.redirect)
			require.NoError(t, err)
			assert.Equal(t, want.Scheme+"://"+want.Host+want.Path, loc.Scheme+"://"+loc.Host+loc.Path)
			for k, v := range want.Query() {
				assert.Equal(t, v, loc.Query()[k], "the redirect's own %q param must survive", k)
			}
			assert.Equal(t, state, loc.Query().Get("state"))
			assert.Empty(t, loc.Query().Get("roles"), "roles are not smuggled through the redirect URI")
		})
	}
}

// TestMagicLinkAuthorizeCodeWithheldByMFA pins that no code is issued when the
// MFA gate withholds the session: the code state would otherwise bind a code to
// a session the user has not finished establishing. The authorize state must
// survive so the flow can resume after the second factor.
func TestMagicLinkAuthorizeCodeWithheldByMFA(t *testing.T) {
	cfg := magicLinkTestConfig()
	cfg.EnableMFA = true
	cfg.EnableTOTPLogin = true
	cfg.EnforceMFA = true
	ts := initTestSetup(t, cfg)
	r := magicLinkRouter(ts)

	state, vw := authorizeAndClickMagicLink(t, ts, r, "magic-link-verifier-long-enough-to-be-valid-0000000", magicLinkCallback)
	require.Equal(t, http.StatusTemporaryRedirect, vw.Code, vw.Body.String())
	loc, err := url.Parse(vw.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "1", loc.Query().Get("mfa_required"), "precondition: the MFA gate withheld the session")
	assert.Empty(t, loc.Query().Get("code"), "no code while MFA withholds the session")
	assert.NotContains(t, loc.String(), "access_token")

	remaining, _ := ts.MemoryStoreProvider.GetState(state)
	assert.NotEmpty(t, remaining, "the authorize state must not be consumed by a withheld session")
}

// TestMagicLinkAuthorizeCodeOnlyToAuthorizeRedirect pins that the code goes
// only to the redirect_uri the /authorize request recorded (RFC 6749 §4.1.2).
// A magic link pointed at a different, merely allowed, URI still signs the
// user in (cookie) but carries no code.
func TestMagicLinkAuthorizeCodeOnlyToAuthorizeRedirect(t *testing.T) {
	ts := initTestSetup(t, magicLinkTestConfig())
	r := magicLinkRouter(ts)

	_, vw := authorizeAndClickMagicLink(t, ts, r, "magic-link-verifier-long-enough-to-be-valid-0000000", "http://localhost:3000/elsewhere")
	require.Equal(t, http.StatusTemporaryRedirect, vw.Code, vw.Body.String())
	loc, err := url.Parse(vw.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "/elsewhere", loc.Path)
	assert.Empty(t, loc.Query().Get("code"))
}
