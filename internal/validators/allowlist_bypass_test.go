package validators

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExactOriginIsNotARegex is the regression guard for the allowlist bypass.
//
// The configured origin used to be spliced into `^`+pattern+`$` with dots left
// unescaped on the exact-origin branch, so `https://login.acme.com` compiled to
// `^login.acme.com$` where every "." matches ANY character. An attacker
// registers a lookalike that differs only where a dot was, passes the
// allowlist, and OAuth access tokens, ID tokens and password-reset tokens are
// delivered to their domain.
//
// This is the branch operators are told to use in production (an explicit
// ALLOWED_ORIGINS list), and it was the branch without the escaping.
func TestExactOriginIsNotARegex(t *testing.T) {
	allowed := []string{"https://login.acme.com"}

	for _, lookalike := range []string{
		"https://loginxacme.com",  // "." matched "x"
		"https://login-acme.com",  // "." matched "-"
		"https://loginqacme.com",  // "." matched any letter
		"https://login.acme-com",  // the second "." matched "-"
		"https://xlogin.acme.com", // not a subdomain — a different registrable domain
	} {
		assert.False(t, IsValidOrigin(lookalike, allowed),
			"IsValidOrigin must not accept %s against %v", lookalike, allowed)
		assert.False(t, IsValidRedirectURI(lookalike+"/callback", allowed, "auth.local"),
			"IsValidRedirectURI must not accept %s against %v", lookalike, allowed)
	}

	// The legitimate origin still works — the fix tightens, it does not break.
	assert.True(t, IsValidOrigin("https://login.acme.com", allowed))
	assert.True(t, IsValidOrigin("https://login.acme.com:443", allowed), "default port still normalises away")
	assert.True(t, IsValidRedirectURI("https://login.acme.com/callback", allowed, "auth.local"))
}

// TestWildcardOriginStillMatchesOnlyRealSubdomains pins that routing both
// callers through one matcher did not loosen the wildcard branch, which was
// already correct. A regression here would be worse than the bug being fixed.
func TestWildcardOriginStillMatchesOnlyRealSubdomains(t *testing.T) {
	allowed := []string{"https://*.acme.com"}

	for _, ok := range []string{"https://app.acme.com", "https://a.b.acme.com"} {
		assert.True(t, IsValidOrigin(ok, allowed), "%s is a proper subdomain", ok)
	}
	for _, bad := range []string{
		"https://acme.com",      // bare domain is not a subdomain
		"https://evil-acme.com", // no dot boundary
		"https://app.acmeXcom",  // the surviving dot must stay literal
		"https://app.acme.com.evil.test",
	} {
		assert.False(t, IsValidOrigin(bad, allowed), "%s must not match %v", bad, allowed)
	}

	// A prefix wildcard stays confined to one label.
	prefixed := []string{"https://api-*.acme.com"}
	assert.True(t, IsValidOrigin("https://api-eu.acme.com", prefixed))
	assert.False(t, IsValidOrigin("https://api-eu.sub.acme.com", prefixed), "[^.]* must not cross a dot")
	assert.False(t, IsValidOrigin("https://api-eu.acmeXcom", prefixed), "dots outside the wildcard stay literal")
}

// TestRegexMetacharactersInConfigAreLiteral pins that nothing an operator can
// put in the allowlist is interpreted as a pattern any more, beyond "*". A
// config value is not attacker-controlled, but treating it as a regex is how
// the bug above happened, and a stray metacharacter should never silently widen
// an allowlist.
func TestRegexMetacharactersInConfigAreLiteral(t *testing.T) {
	assert.True(t, IsValidOrigin("https://a+b.acme.com", []string{"https://a+b.acme.com"}),
		"a literal '+' in the host must match itself")
	assert.False(t, IsValidOrigin("https://ab.acme.com", []string{"https://a+b.acme.com"}),
		"'+' must not act as a quantifier")
	assert.False(t, IsValidOrigin("https://anything.acme.com", []string{"https://.*.acme.com"}),
		"a config value that looks like a regex must not behave as one")
}

// TestSSRFGuardBlocksIPv6TransitionRanges is the regression guard for the SSRF
// bypass. 6to4, Teredo and NAT64 addresses EMBED an arbitrary IPv4 address, so
// a v6 literal in those ranges reaches a v4 destination that every private
// range in the blocklist would otherwise have refused.
func TestSSRFGuardBlocksIPv6TransitionRanges(t *testing.T) {
	for _, tc := range []struct{ addr, why string }{
		{"2002:7f00:0001::", "6to4 wrapping 127.0.0.1"},
		{"2002:a9fe:a9fe::", "6to4 wrapping 169.254.169.254 (cloud metadata)"},
		{"2002:c0a8:0001::", "6to4 wrapping 192.168.0.1"},
		{"2002:0a00:0001::", "6to4 wrapping 10.0.0.1"},
		{"64:ff9b::7f00:1", "NAT64 wrapping 127.0.0.1"},
		{"64:ff9b::a9fe:a9fe", "NAT64 wrapping 169.254.169.254 (cloud metadata)"},
		{"64:ff9b:1::a9fe:a9fe", "RFC 8215 local-use NAT64"},
		{"2001:0:4136:e378:8000:63bf:3fff:fdd2", "Teredo"},
		{"::", "unspecified address"},
		{"100::1", "discard-only prefix"},
	} {
		ip := net.ParseIP(tc.addr)
		require.NotNil(t, ip, "test address %q must parse", tc.addr)
		assert.True(t, isPrivateIP(ip), "%s (%s) must be treated as private", tc.addr, tc.why)
	}
}

// TestSSRFGuardStillAllowsOrdinaryPublicAddresses pins that the widened
// blocklist did not start refusing real webhook or OIDC destinations.
func TestSSRFGuardStillAllowsOrdinaryPublicAddresses(t *testing.T) {
	for _, addr := range []string{
		"8.8.8.8",
		"1.1.1.1",
		"93.184.216.34",
		"2606:4700:4700::1111", // Cloudflare DNS over IPv6
		"2a00:1450:4001:800::200e",
	} {
		ip := net.ParseIP(addr)
		require.NotNil(t, ip)
		assert.False(t, isPrivateIP(ip), "%s is a legitimate public destination", addr)
	}
}
