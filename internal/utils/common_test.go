package utils

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetForgotPasswordURL(t *testing.T) {
	const tok = "header.payload.sig"

	t.Run("plain redirect uri gets token query", func(t *testing.T) {
		assert.Equal(t, "http://localhost:8080/app/reset-password?token="+tok,
			GetForgotPasswordURL(tok, "http://localhost:8080/app/reset-password"))
	})

	// Issue #344: the login UI forwards the original /authorize parameters on
	// the reset-password redirect_uri so the OAuth flow can resume after the
	// reset. The token must be merged into that query, not appended as a
	// second "?".
	t.Run("existing query is preserved and token merged", func(t *testing.T) {
		redirect := "http://localhost:8080/app/reset-password?client_id=abc&state=s%26t&redirect_uri=" +
			url.QueryEscape("http://localhost:3000/cb") + "&response_type=code&code_challenge=xyz"
		got := GetForgotPasswordURL(tok, redirect)
		u, err := url.Parse(got)
		require.NoError(t, err)
		assert.Equal(t, "/app/reset-password", u.Path)
		q := u.Query()
		assert.Equal(t, tok, q.Get("token"))
		assert.Equal(t, "abc", q.Get("client_id"))
		assert.Equal(t, "s&t", q.Get("state"))
		assert.Equal(t, "http://localhost:3000/cb", q.Get("redirect_uri"))
		assert.Equal(t, "code", q.Get("response_type"))
		assert.Equal(t, "xyz", q.Get("code_challenge"))
	})

	// Hash-routed SPAs read the token from the fragment route; it must stay
	// there (pre-#344 behaviour), not move to the real query string.
	t.Run("hash routed redirect uri keeps token in fragment", func(t *testing.T) {
		assert.Equal(t, "https://app.example.com/#/reset-password?token="+tok,
			GetForgotPasswordURL(tok, "https://app.example.com/#/reset-password"))
	})

	t.Run("hash routed redirect uri with fragment query merges token", func(t *testing.T) {
		got := GetForgotPasswordURL(tok, "https://app.example.com/base?x=1#/reset-password?state=s%26t&token=attacker")
		u, err := url.Parse(got)
		require.NoError(t, err)
		assert.Equal(t, "/base", u.Path)
		assert.Equal(t, "x=1", u.RawQuery)
		route, rawQ, ok := strings.Cut(u.EscapedFragment(), "?")
		require.True(t, ok)
		assert.Equal(t, "/reset-password", route)
		fq, err := url.ParseQuery(rawQ)
		require.NoError(t, err)
		assert.Equal(t, []string{tok}, fq["token"])
		assert.Equal(t, "s&t", fq.Get("state"))
	})

	t.Run("caller supplied token param cannot shadow the real token", func(t *testing.T) {
		got := GetForgotPasswordURL(tok, "http://localhost:8080/app/reset-password?token=attacker")
		u, err := url.Parse(got)
		require.NoError(t, err)
		assert.Equal(t, []string{tok}, u.Query()["token"])
	})
}

// A redirect_uri carrying its own query string must round-trip intact through
// the emailed link; it used to be concatenated raw, so its parameters were
// parsed as the link's own and /verify_email received a truncated URI.
func TestGetEmailVerificationURL(t *testing.T) {
	const tok = "header.payload.sig"
	redirect := "https://app.example.com/cb?tenant=acme&x=a%26b"

	link := GetEmailVerificationURL(tok, "https://auth.example.com", redirect, url.Values{
		"state": {"s&1"},
		"scope": {"openid email"},
	})
	require.True(t, strings.HasPrefix(link, "https://auth.example.com/verify_email?"), link)
	u, err := url.Parse(link)
	require.NoError(t, err)
	q := u.Query()
	assert.Equal(t, tok, q.Get("token"))
	assert.Equal(t, redirect, q.Get("redirect_uri"))
	assert.Equal(t, "s&1", q.Get("state"))
	assert.Equal(t, "openid email", q.Get("scope"))
	assert.Empty(t, q.Get("tenant"), "the redirect's params must not leak into the link")

	plain, err := url.Parse(GetEmailVerificationURL(tok, "https://auth.example.com", "https://app.example.com/cb", nil))
	require.NoError(t, err)
	assert.Equal(t, "https://app.example.com/cb", plain.Query().Get("redirect_uri"))
}

func TestGetInviteVerificationURL(t *testing.T) {
	redirect := "https://app.example.com/cb?tenant=acme"
	u, err := url.Parse(GetInviteVerificationURL("https://app.example.com/setup-password", "tok", redirect))
	require.NoError(t, err)
	assert.Equal(t, redirect, u.Query().Get("redirect_uri"))
	assert.Equal(t, "tok", u.Query().Get("token"))
}
