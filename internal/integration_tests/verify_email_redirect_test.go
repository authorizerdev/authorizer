package integration_tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/codestate"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// noRedirectClient follows nothing, so the Location header can be inspected.
func noRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// TestVerifyEmailDoesNotLeakTokensInRedirect pins that /verify_email never puts
// session tokens in the URL it redirects to.
//
// It used to append access_token, id_token and refresh_token to the QUERY
// string of a redirect target validated only at origin granularity, so any path
// under an allowed origin received a full session (GHSA-44vr-f829-xfch). An
// unauthenticated attacker could start the flow through magic_link_login with a
// redirect_uri under the app's own origin and read the victim's tokens from the
// target page, the access log, the browser history or the Referer header.
//
// This endpoint is reached from an emailed link, so its URL is unusually
// long-lived: it sits in mail-client history and in every proxy log between the
// user and the app. oauth_callback.go and oauth_sso.go already deliver a `code`
// plus the session cookie for exactly this reason; this handler was the last one
// still emitting raw tokens.
func TestVerifyEmailDoesNotLeakTokensInRedirect(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "verify_email_leak_" + uuid.New().String() + "@authorizer.dev"
	const password = "Password@123"

	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	vreq, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err)
	require.NotNil(t, vreq)

	resp, err := noRedirectClient().Get(
		ts.HttpServer.URL + "/verify_email?token=" + url.QueryEscape(vreq.Token) +
			"&redirect_uri=" + url.QueryEscape("http://localhost:3000/callback"))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	location := resp.Header.Get("Location")
	require.NotEmpty(t, location)

	for _, leaked := range []string{"access_token=", "id_token=", "refresh_token="} {
		assert.NotContains(t, location, leaked,
			"a session token must never appear in the verify_email redirect URL")
	}

	// Asserted on the whole Location, so moving the tokens from the query to
	// the fragment would not pass either — a fragment is still in the URL, and
	// the URL is in the email.
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	assert.NotContains(t, parsed.Fragment, "access_token")
	assert.NotContains(t, parsed.Fragment, "refresh_token")

	// The flow must still complete: the browser session is the delivery
	// channel now, so if this cookie stops being set the fix has broken login
	// rather than secured it.
	var sawSessionCookie bool
	for _, c := range resp.Cookies() {
		if strings.Contains(c.Name, "session") && c.Value != "" {
			sawSessionCookie = true
		}
	}
	assert.True(t, sawSessionCookie, "the session cookie must still be set")
}

// TestVerifyEmailRejectsUnregisteredRedirect pins the second half: when the
// deployment has declared its redirect URIs with --redirect-uris, the target is
// matched EXACTLY (OIDC Core §3.1.2.1) rather than by origin, so a sibling path
// under an allowed host is refused.
//
// Origin-only matching is what let any path through in the first place. The
// exact-match validator already existed for /authorize and /app; this endpoint
// simply never called it.
func TestVerifyEmailRejectsUnregisteredRedirect(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	cfg.RedirectURIs = []string{"http://localhost:3000/callback"}
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "verify_email_exact_" + uuid.New().String() + "@authorizer.dev"
	const password = "Password@123"

	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	vreq, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err)

	t.Run("a sibling path under the same origin is refused", func(t *testing.T) {
		resp, err := noRedirectClient().Get(
			ts.HttpServer.URL + "/verify_email?token=" + url.QueryEscape(vreq.Token) +
				"&redirect_uri=" + url.QueryEscape("http://localhost:3000/attacker-controlled-path"))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
			"an unregistered path must be refused, not redirected to")
		assert.NotContains(t, resp.Header.Get("Location"), "access_token=")
	})

	t.Run("the registered URI still works", func(t *testing.T) {
		resp, err := noRedirectClient().Get(
			ts.HttpServer.URL + "/verify_email?token=" + url.QueryEscape(vreq.Token) +
				"&redirect_uri=" + url.QueryEscape("http://localhost:3000/callback"))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode,
			"the exact registered redirect must still complete the flow")
	})
}

// TestVerifyEmailWildcardOriginUnchanged guards the default configuration. With
// allowed_origins at "*", IsValidRedirectURI restricts redirects to the
// server's own host; that behaviour predates this fix and must survive it.
func TestVerifyEmailWildcardOriginUnchanged(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	cfg.AllowedOrigins = []string{"*"}
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "verify_email_wildcard_" + uuid.New().String() + "@authorizer.dev"
	const password = "Password@123"

	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	vreq, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err)

	resp, err := noRedirectClient().Get(
		ts.HttpServer.URL + "/verify_email?token=" + url.QueryEscape(vreq.Token) +
			"&redirect_uri=" + url.QueryEscape("https://evil.example.com/steal"))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
		"a foreign origin must stay refused under the wildcard default")
}

// TestVerifyEmailStillDeliversCodeForAuthorizeFlow proves the replacement
// delivery channel actually works.
//
// Removing the tokens from the redirect is only safe if the client still has a
// way to obtain a session. For a flow that began at /authorize there is one:
// the authorization `code`, which this handler carries through from the stored
// state and which the client exchanges at /oauth/token. Asserting only that the
// tokens are GONE would pass just as well if the handler had stopped issuing
// anything at all.
func TestVerifyEmailStillDeliversCodeForAuthorizeFlow(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "verify_email_code_" + uuid.New().String() + "@authorizer.dev"
	const password = "Password@123"

	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	vreq, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
	require.NoError(t, err)

	// Seed the state entry exactly as /authorize does when it detours an
	// unauthenticated visitor through the login page.
	state := uuid.New().String()
	expectedCode := uuid.New().String()
	require.NoError(t, ts.MemoryStoreProvider.SetState(state, codestate.EncodeAuthorize(codestate.Authorize{
		Code:        expectedCode,
		Nonce:       uuid.New().String(),
		RedirectURI: "http://localhost:3000/callback",
		ClientID:    ts.Config.ClientID,
	})))

	resp, err := noRedirectClient().Get(
		ts.HttpServer.URL + "/verify_email?token=" + url.QueryEscape(vreq.Token) +
			"&state=" + url.QueryEscape(state) +
			"&redirect_uri=" + url.QueryEscape("http://localhost:3000/callback"))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	location := resp.Header.Get("Location")

	assert.Contains(t, location, "code="+expectedCode,
		"the authorization code must still reach the client — it is how a session is obtained now")
	assert.NotContains(t, location, "access_token=")
	assert.NotContains(t, location, "refresh_token=")
}
