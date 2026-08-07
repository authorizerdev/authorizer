package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAppCookieSameSiteDefaultIsNone is a decision guard for the other half of
// the session-cookie topology (the first half is pinned in
// internal/cookie.TestSessionCookieTopologyIsDeliberate).
//
// "none" looks like an obviously-wrong default to a scanner or a security
// review, and the 2.4.0 pre-release audit duly flagged it. It was consciously
// kept: Authorizer targets an auth server on a subdomain serving apps on other
// sites, and Lax withholds the session cookie on exactly those cross-site
// requests, breaking the browser-session half of the SDK. Auth0 takes the same
// position — it recommends SameSite=None for cross-origin authentication and
// ships fallback cookies for browsers that cannot do it.
//
// CSRF middleware, HttpOnly, and Authorization-header auth are what actually
// carry the security here; SameSite is defense-in-depth. See
// cookie.BuildSessionCookies before changing this.
func TestAppCookieSameSiteDefaultIsNone(t *testing.T) {
	f := RootCmd.PersistentFlags().Lookup("app-cookie-same-site")
	require.NotNil(t, f, "the --app-cookie-same-site flag must exist")
	assert.Equal(t, "none", f.DefValue,
		"changing this default breaks cross-site apps; read cookie.BuildSessionCookies first")
}
