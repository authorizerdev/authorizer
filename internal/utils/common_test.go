package utils

import (
	"net/url"
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

	t.Run("caller supplied token param cannot shadow the real token", func(t *testing.T) {
		got := GetForgotPasswordURL(tok, "http://localhost:8080/app/reset-password?token=attacker")
		u, err := url.Parse(got)
		require.NoError(t, err)
		assert.Equal(t, []string{tok}, u.Query()["token"])
	})
}
