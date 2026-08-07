package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/cookie"
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

// TestAppCookieSameSiteIsValidated pins that a mistyped value stops the process
// instead of silently becoming lax.
//
// cookie.ParseSameSite falls back to Lax for anything unrecognised. That is a
// safe default but a silent one: an operator who asks for `strict` and mistypes
// it gets Lax — a real downgrade from what they requested, with nothing
// anywhere to say so — and a mistyped `none` withholds the session cookie from
// cross-site apps, which presents as "login randomly doesn't stick".
func TestAppCookieSameSiteIsValidated(t *testing.T) {
	t.Parallel()

	for _, valid := range []string{"lax", "strict", "none", "STRICT", " none ", ""} {
		cfg := config.Config{AppCookieSameSite: valid}
		assert.NoError(t, cfg.ValidateAppCookieSameSite(), "%q is a supported value", valid)
	}

	for _, invalid := range []string{"strct", "nonw", "same-site", "true", "0"} {
		cfg := config.Config{AppCookieSameSite: invalid}
		err := cfg.ValidateAppCookieSameSite()
		require.Error(t, err, "%q must be rejected, not silently downgraded to lax", invalid)
		assert.Contains(t, err.Error(), "app-cookie-same-site")
	}
}

// TestEverySameSiteValueRoundTrips pins that each accepted CLI value maps to the
// SameSite mode it names — the validator and the parser must agree, or a value
// passes validation and then means something else.
func TestEverySameSiteValueRoundTrips(t *testing.T) {
	t.Parallel()
	want := map[string]http.SameSite{
		"lax":    http.SameSiteLaxMode,
		"strict": http.SameSiteStrictMode,
		"none":   http.SameSiteNoneMode,
	}
	for _, v := range config.ValidAppCookieSameSiteValues {
		cfg := config.Config{AppCookieSameSite: v}
		require.NoError(t, cfg.ValidateAppCookieSameSite())
		assert.Equal(t, want[v], cookie.ParseSameSite(v),
			"%q passes validation, so it must parse to the mode it names", v)
	}
}
