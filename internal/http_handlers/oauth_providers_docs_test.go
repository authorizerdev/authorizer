package http_handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/oauth"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// This file pins each provider handler to the payload its provider's own
// documentation says it returns - specifically the parts the handlers used to
// get wrong: mixed-type JSON, optional/nullable fields, and array-typed
// claims.

// newOAuthTestServer mocks a provider's /token plus one userinfo-style route.
func newOAuthTestServer(t *testing.T, routes map[string]interface{}) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "mock-access-token",
			"token_type":   "bearer",
		})
	})
	for path, body := range routes {
		body := body
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(body)
		})
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func newOAuthTestProvider(t *testing.T, mockBase string, apply func(*config.Config)) *httpProvider {
	t.Helper()
	config.TestOAuthMockBaseOverride = mockBase
	t.Cleanup(func() { config.TestOAuthMockBaseOverride = "" })
	logger := zerolog.Nop()
	cfg := &config.Config{Env: constants.E2EEnv}
	apply(cfg)
	oauthProvider, err := oauth.New(cfg, &oauth.Dependencies{Log: &logger})
	require.NoError(t, err)
	return &httpProvider{
		Config: cfg,
		Dependencies: Dependencies{
			Log:           &logger,
			OAuthProvider: oauthProvider,
		},
	}
}

// --- Microsoft / Google / Twitch: OIDC ID token claims -----------------------

// TestOIDCClaims_ArrayValuedClaimsDoNotBreakDecode is the regression guard for
// decoding ID tokens straight into schemas.User: Microsoft Entra emits `roles`
// as an "Array of strings" (ID token claims reference) while
// schemas.User.Roles is a string, so every login for a tenant that assigns app
// roles failed with "unable to extract claims".
func TestOIDCClaims_ArrayValuedClaimsDoNotBreakDecode(t *testing.T) {
	// A realistic Entra v2.0 ID token payload.
	payload := []byte(`{
		"aud": "6731de76-14a6-49ae-97bc-6eba6914391e",
		"iss": "https://login.microsoftonline.com/tenant/v2.0",
		"iat": 1735689600,
		"name": "Ada Lovelace",
		"given_name": "Ada",
		"family_name": "Lovelace",
		"email": "ada@contoso.com",
		"preferred_username": "ada@contoso.com",
		"roles": ["Admin", "Reader"],
		"groups": ["7d0f7fbd-1cbe-4f2b-9d5c-4a0f1b1a3f9c"],
		"wids": ["62e90394-69f5-4237-9190-012177145e10"],
		"hasgroups": true,
		"ver": "2.0"
	}`)

	claims := &oidcClaims{}
	require.NoError(t, json.Unmarshal(payload, claims), "array-valued claims must not fail the decode")

	user := claims.toUser()
	assert.Equal(t, "ada@contoso.com", refs.StringValue(user.Email))
	assert.Equal(t, "Ada", refs.StringValue(user.GivenName))
	assert.Equal(t, "Lovelace", refs.StringValue(user.FamilyName))
	// Storage-only columns are never fed from claims, whatever the IdP sends.
	assert.Empty(t, user.Roles)
	assert.Empty(t, user.ID)
	assert.Empty(t, user.SignupMethods)
}

// TestOIDCClaims_AbsentClaimsStayNil keeps absent claims from overwriting
// stored values with empty strings.
func TestOIDCClaims_AbsentClaimsStayNil(t *testing.T) {
	claims := &oidcClaims{}
	require.NoError(t, json.Unmarshal([]byte(`{"email":"a@b.com"}`), claims))

	user := claims.toUser()
	require.NotNil(t, user.Email)
	assert.Nil(t, user.GivenName)
	assert.Nil(t, user.Picture)
	assert.Nil(t, user.PhoneNumber)
}

// --- Facebook ---------------------------------------------------------------

func facebookProfile(withEmail bool) map[string]interface{} {
	profile := map[string]interface{}{
		"id":         "10224444555566666",
		"first_name": "Ada",
		"last_name":  "Lovelace",
		"name":       "Ada Lovelace",
		"picture": map[string]interface{}{
			"data": map[string]interface{}{
				"height":        50,
				"is_silhouette": false,
				"url":           "https://scontent.xx.fbcdn.net/v/ada.jpg",
				"width":         50,
			},
		},
	}
	if withEmail {
		profile["email"] = "ada@example.com"
	}
	return profile
}

func TestProcessFacebookUserInfo_RealProfile(t *testing.T) {
	server := newOAuthTestServer(t, map[string]interface{}{"/userinfo": facebookProfile(true)})
	h := newOAuthTestProvider(t, server.URL, func(c *config.Config) {
		c.FacebookClientID = "test-client"
		c.FacebookClientSecret = "test-secret"
	})

	user, _, err := h.processFacebookUserInfo(testGinContext(), "code")
	require.NoError(t, err)

	assert.Equal(t, "ada@example.com", refs.StringValue(user.Email))
	assert.Equal(t, "Ada", refs.StringValue(user.GivenName))
	assert.Equal(t, "Lovelace", refs.StringValue(user.FamilyName))
	assert.Equal(t, "https://scontent.xx.fbcdn.net/v/ada.jpg", refs.StringValue(user.Picture))
}

// TestProcessFacebookUserInfo_MissingEmailIsAnError is the regression guard:
// Graph API omits `email` when "no valid email address is available", and the
// handler used to fmt.Sprintf the missing key into the literal string "<nil>"
// and store that as the user's email address.
func TestProcessFacebookUserInfo_MissingEmailIsAnError(t *testing.T) {
	server := newOAuthTestServer(t, map[string]interface{}{"/userinfo": facebookProfile(false)})
	h := newOAuthTestProvider(t, server.URL, func(c *config.Config) {
		c.FacebookClientID = "test-client"
		c.FacebookClientSecret = "test-secret"
	})

	user, _, err := h.processFacebookUserInfo(testGinContext(), "code")
	require.Error(t, err)
	assert.Nil(t, user)
	assert.NotContains(t, err.Error(), "<nil>")
}

// --- LinkedIn ---------------------------------------------------------------

// TestProcessLinkedInUserInfo_OIDCUserinfo covers the migration off the legacy
// /v2/me + /v2/emailAddress pair (r_liteprofile/r_emailaddress, not
// provisioned for apps onboarded to "Sign In with LinkedIn using OpenID
// Connect") onto the discovery-published /v2/userinfo endpoint. Payload is
// LinkedIn's own documented sample response.
func TestProcessLinkedInUserInfo_OIDCUserinfo(t *testing.T) {
	server := newOAuthTestServer(t, map[string]interface{}{
		"/userinfo": map[string]interface{}{
			"sub":            "782bbtaQ",
			"name":           "John Doe",
			"given_name":     "John",
			"family_name":    "Doe",
			"picture":        "https://media.licdn-ei.com/dms/image/ada",
			"locale":         "en-US",
			"email":          "doe@email.com",
			"email_verified": true,
		},
	})
	h := newOAuthTestProvider(t, server.URL, func(c *config.Config) {
		c.LinkedinClientID = "test-client"
		c.LinkedinClientSecret = "test-secret"
	})

	user, _, err := h.processLinkedInUserInfo(testGinContext(), "code")
	require.NoError(t, err)

	assert.Equal(t, "doe@email.com", refs.StringValue(user.Email))
	assert.Equal(t, "John", refs.StringValue(user.GivenName))
	assert.Equal(t, "Doe", refs.StringValue(user.FamilyName))
	assert.Equal(t, "https://media.licdn-ei.com/dms/image/ada", refs.StringValue(user.Picture))
}

// TestProcessLinkedInUserInfo_OptionalEmailAbsent: LinkedIn documents `email`
// as optional, and `sub` is pairwise per-app so it is no usable identity key.
func TestProcessLinkedInUserInfo_OptionalEmailAbsent(t *testing.T) {
	server := newOAuthTestServer(t, map[string]interface{}{
		"/userinfo": map[string]interface{}{
			"sub":         "782bbtaQ",
			"given_name":  "John",
			"family_name": "Doe",
		},
	})
	h := newOAuthTestProvider(t, server.URL, func(c *config.Config) {
		c.LinkedinClientID = "test-client"
		c.LinkedinClientSecret = "test-secret"
	})

	user, _, err := h.processLinkedInUserInfo(testGinContext(), "code")
	require.Error(t, err)
	assert.Nil(t, user)
}

// --- GitHub: unverified email addresses --------------------------------------

// TestProcessGithubUserInfo_UnverifiedEmailsRejected guards the account-linking
// hole: GET /user/emails lists unverified addresses too, and the caller looks
// an existing account up by email - so accepting an unverified address would
// let a GitHub account that merely typed someone else's address sign in as
// them.
func TestProcessGithubUserInfo_UnverifiedEmailsRejected(t *testing.T) {
	server := newGithubTestServer(t, githubProfile(nil), []map[string]interface{}{
		{"email": "victim@example.com", "primary": true, "verified": false},
	})
	h := newGithubTestHTTPProvider(t, server.URL)

	user, _, err := h.processGithubUserInfo(testGinContext(), "code")
	require.Error(t, err)
	assert.Nil(t, user)
}

// TestProcessGithubUserInfo_PrefersVerifiedPrimary: an unverified address must
// not win over the verified primary one.
func TestProcessGithubUserInfo_PrefersVerifiedPrimary(t *testing.T) {
	server := newGithubTestServer(t, githubProfile(nil), []map[string]interface{}{
		{"email": "unverified@example.com", "primary": false, "verified": false},
		{"email": "primary@example.com", "primary": true, "verified": true},
	})
	h := newGithubTestHTTPProvider(t, server.URL)

	user, _, err := h.processGithubUserInfo(testGinContext(), "code")
	require.NoError(t, err)
	assert.Equal(t, "primary@example.com", refs.StringValue(user.Email))
}

// --- Discord: nullable avatar ------------------------------------------------

// TestProcessDiscordUserInfo_NullAvatar: `avatar` is ?string in Discord's user
// object; building a CDN URL from an empty hash produced a dead link.
func TestProcessDiscordUserInfo_NullAvatar(t *testing.T) {
	server := newOAuthTestServer(t, map[string]interface{}{
		"/userinfo": map[string]interface{}{
			"id":          "80351110224678912",
			"username":    "ada",
			"global_name": nil,
			"avatar":      nil,
			"email":       "ada@example.com",
			"verified":    true,
		},
	})
	h := newOAuthTestProvider(t, server.URL, func(c *config.Config) {
		c.DiscordClientID = "test-client"
		c.DiscordClientSecret = "test-secret"
	})

	user, _, err := h.processDiscordUserInfo(testGinContext(), "code")
	require.NoError(t, err)

	assert.Equal(t, "ada@example.com", refs.StringValue(user.Email))
	assert.Empty(t, refs.StringValue(user.Picture), "no avatar hash must not yield a dead CDN URL")
}

func TestProcessDiscordUserInfo_WithAvatar(t *testing.T) {
	server := newOAuthTestServer(t, map[string]interface{}{
		"/userinfo": map[string]interface{}{
			"id":       "80351110224678912",
			"username": "ada",
			"avatar":   "8342729096ea3675442027381ff50dfe",
			"email":    "ada@example.com",
		},
	})
	h := newOAuthTestProvider(t, server.URL, func(c *config.Config) {
		c.DiscordClientID = "test-client"
		c.DiscordClientSecret = "test-secret"
	})

	user, _, err := h.processDiscordUserInfo(testGinContext(), "code")
	require.NoError(t, err)
	assert.Equal(t,
		"https://cdn.discordapp.com/avatars/80351110224678912/8342729096ea3675442027381ff50dfe.png",
		refs.StringValue(user.Picture))
}
