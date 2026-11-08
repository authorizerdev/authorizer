package validators

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeOrigin(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"https with port 443", "https://example.com:443", "example.com"},
		{"http with port 80", "http://example.com:80", "example.com"},
		{"https without port", "https://example.com", "example.com"},
		{"http without port", "http://example.com", "example.com"},
		{"with custom port", "http://localhost:3000", "localhost:3000"},
		{"bare domain", "example.com", "example.com"},
		{"bare domain with port", "localhost:8080", "localhost:8080"},
		{"with path", "https://example.com/callback", "example.com"},
		{"with path and port", "http://localhost:3000/app/login", "localhost:3000"},
		{"subdomain", "https://auth.example.com", "auth.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeOrigin(tt.input))
		})
	}
}

func TestIsValidOrigin(t *testing.T) {
	t.Run("wildcard allows everything", func(t *testing.T) {
		assert.True(t, IsValidOrigin("https://anything.com", []string{"*"}))
		assert.True(t, IsValidOrigin("http://localhost:9999", []string{"*"}))
	})

	t.Run("empty config defaults to wildcard", func(t *testing.T) {
		assert.True(t, IsValidOrigin("https://anything.com", []string{}))
		assert.True(t, IsValidOrigin("https://anything.com", nil))
	})

	// --- Exact domain matching ---

	t.Run("exact domain without port", func(t *testing.T) {
		allowed := []string{"https://example.com"}
		assert.True(t, IsValidOrigin("https://example.com", allowed))
		assert.True(t, IsValidOrigin("https://example.com/callback", allowed))
		assert.True(t, IsValidOrigin("http://example.com", allowed))
		assert.False(t, IsValidOrigin("https://evil.com", allowed))
		assert.False(t, IsValidOrigin("https://sub.example.com", allowed))
	})

	t.Run("exact domain with standard port is equivalent to without", func(t *testing.T) {
		allowed := []string{"https://example.com:443"}
		assert.True(t, IsValidOrigin("https://example.com", allowed))
		assert.True(t, IsValidOrigin("https://example.com:443", allowed))
	})

	t.Run("http with port 80 is equivalent to without", func(t *testing.T) {
		allowed := []string{"http://example.com:80"}
		assert.True(t, IsValidOrigin("http://example.com", allowed))
		assert.True(t, IsValidOrigin("http://example.com:80", allowed))
	})

	// --- Custom ports ---

	t.Run("localhost with custom port", func(t *testing.T) {
		allowed := []string{"http://localhost:3000"}
		assert.True(t, IsValidOrigin("http://localhost:3000", allowed))
		assert.True(t, IsValidOrigin("http://localhost:3000/app", allowed))
		assert.False(t, IsValidOrigin("http://localhost:4000", allowed))
		assert.False(t, IsValidOrigin("http://localhost", allowed))
	})

	t.Run("domain with non-standard port", func(t *testing.T) {
		allowed := []string{"https://staging.example.com:8443"}
		assert.True(t, IsValidOrigin("https://staging.example.com:8443", allowed))
		assert.False(t, IsValidOrigin("https://staging.example.com", allowed))
		assert.False(t, IsValidOrigin("https://staging.example.com:9443", allowed))
	})

	// --- Allowed origins without protocol ---

	t.Run("allowed origin without protocol", func(t *testing.T) {
		allowed := []string{"example.com"}
		assert.True(t, IsValidOrigin("https://example.com", allowed))
		assert.True(t, IsValidOrigin("http://example.com", allowed))
		assert.False(t, IsValidOrigin("https://evil.com", allowed))
	})

	t.Run("allowed origin as host:port without protocol", func(t *testing.T) {
		allowed := []string{"localhost:3000"}
		assert.True(t, IsValidOrigin("http://localhost:3000", allowed))
		assert.True(t, IsValidOrigin("http://localhost:3000/callback", allowed))
		assert.False(t, IsValidOrigin("http://localhost:4000", allowed))
	})

	// --- Subdomain matching ---

	t.Run("subdomain is distinct from root domain", func(t *testing.T) {
		allowed := []string{"https://example.com"}
		assert.False(t, IsValidOrigin("https://api.example.com", allowed))
		assert.False(t, IsValidOrigin("https://auth.example.com", allowed))
	})

	t.Run("specific subdomain allowed", func(t *testing.T) {
		allowed := []string{"https://auth.example.com"}
		assert.True(t, IsValidOrigin("https://auth.example.com", allowed))
		assert.False(t, IsValidOrigin("https://example.com", allowed))
		assert.False(t, IsValidOrigin("https://api.example.com", allowed))
	})

	t.Run("deep subdomain", func(t *testing.T) {
		allowed := []string{"https://app.auth.example.com"}
		assert.True(t, IsValidOrigin("https://app.auth.example.com", allowed))
		assert.False(t, IsValidOrigin("https://auth.example.com", allowed))
		assert.False(t, IsValidOrigin("https://other.auth.example.com", allowed))
	})

	// --- Wildcard subdomain matching ---

	t.Run("wildcard subdomain", func(t *testing.T) {
		allowed := []string{"*.example.com"}
		assert.True(t, IsValidOrigin("https://auth.example.com", allowed))
		assert.True(t, IsValidOrigin("https://api.example.com", allowed))
		assert.True(t, IsValidOrigin("https://app.staging.example.com", allowed))
		assert.False(t, IsValidOrigin("https://example.com", allowed))
		assert.False(t, IsValidOrigin("https://evil.com", allowed))
		assert.False(t, IsValidOrigin("https://exampleXcom.evil.com", allowed))
	})

	t.Run("wildcard subdomain with port", func(t *testing.T) {
		allowed := []string{"*.example.com:8080"}
		assert.True(t, IsValidOrigin("https://api.example.com:8080", allowed))
		assert.False(t, IsValidOrigin("https://api.example.com", allowed))
		assert.False(t, IsValidOrigin("https://api.example.com:9090", allowed))
	})

	// --- Multiple allowed origins ---

	t.Run("multiple allowed origins", func(t *testing.T) {
		allowed := []string{
			"https://app.example.com",
			"https://admin.example.com",
			"http://localhost:3000",
		}
		assert.True(t, IsValidOrigin("https://app.example.com", allowed))
		assert.True(t, IsValidOrigin("https://admin.example.com", allowed))
		assert.True(t, IsValidOrigin("http://localhost:3000", allowed))
		assert.False(t, IsValidOrigin("https://evil.com", allowed))
		assert.False(t, IsValidOrigin("https://example.com", allowed))
	})

	// --- Attacker URLs (security cases) ---

	t.Run("attacker URL rejected", func(t *testing.T) {
		allowed := []string{"https://example.com"}
		assert.False(t, IsValidOrigin("https://attacker.com/steal", allowed))
		assert.False(t, IsValidOrigin("https://example.com.attacker.com", allowed))
		assert.False(t, IsValidOrigin("https://attacker.com?ref=example.com", allowed))
	})

	t.Run("attacker URL with similar name rejected", func(t *testing.T) {
		allowed := []string{"https://myapp.com"}
		assert.False(t, IsValidOrigin("https://notmyapp.com", allowed))
		assert.False(t, IsValidOrigin("https://myapp.com.evil.com", allowed))
		assert.False(t, IsValidOrigin("https://myapp.com.evil.com:3000", allowed))
	})

	// --- Live domain scenarios ---

	t.Run("production domain with www", func(t *testing.T) {
		allowed := []string{"https://www.myapp.com"}
		assert.True(t, IsValidOrigin("https://www.myapp.com", allowed))
		assert.False(t, IsValidOrigin("https://myapp.com", allowed))
	})

	t.Run("both www and apex allowed", func(t *testing.T) {
		allowed := []string{"https://myapp.com", "https://www.myapp.com"}
		assert.True(t, IsValidOrigin("https://myapp.com", allowed))
		assert.True(t, IsValidOrigin("https://www.myapp.com", allowed))
		assert.False(t, IsValidOrigin("https://api.myapp.com", allowed))
	})

	t.Run("wildcard with live domain", func(t *testing.T) {
		allowed := []string{"*.myapp.com"}
		assert.True(t, IsValidOrigin("https://www.myapp.com", allowed))
		assert.True(t, IsValidOrigin("https://api.myapp.com", allowed))
		assert.True(t, IsValidOrigin("https://staging.api.myapp.com", allowed))
		assert.False(t, IsValidOrigin("https://myapp.com", allowed))
		assert.False(t, IsValidOrigin("https://evil.com", allowed))
	})
}

func TestIsValidRedirectURI(t *testing.T) {
	hostname := "https://myserver.com"

	t.Run("wildcard config allows self-origin redirect", func(t *testing.T) {
		assert.True(t, IsValidRedirectURI("https://myserver.com/app/reset-password", []string{"*"}, hostname))
		assert.True(t, IsValidRedirectURI("https://myserver.com/callback", []string{"*"}, hostname))
		assert.True(t, IsValidRedirectURI("https://myserver.com", []string{"*"}, hostname))
	})

	t.Run("wildcard config rejects attacker URL", func(t *testing.T) {
		assert.False(t, IsValidRedirectURI("https://attacker.com/capture", []string{"*"}, hostname))
		assert.False(t, IsValidRedirectURI("https://evil.com/steal", []string{"*"}, hostname))
		assert.False(t, IsValidRedirectURI("https://myserver.com.attacker.com/cb", []string{"*"}, hostname))
	})

	t.Run("wildcard config rejects subdomain mismatch", func(t *testing.T) {
		assert.False(t, IsValidRedirectURI("https://sub.myserver.com/cb", []string{"*"}, hostname))
	})

	t.Run("empty config defaults to wildcard behavior", func(t *testing.T) {
		assert.True(t, IsValidRedirectURI("https://myserver.com/cb", []string{}, hostname))
		assert.True(t, IsValidRedirectURI("https://myserver.com/cb", nil, hostname))
		assert.False(t, IsValidRedirectURI("https://attacker.com", []string{}, hostname))
		assert.False(t, IsValidRedirectURI("https://attacker.com", nil, hostname))
	})

	t.Run("explicit origins valid redirect", func(t *testing.T) {
		allowed := []string{"https://app.example.com"}
		assert.True(t, IsValidRedirectURI("https://app.example.com/callback", allowed, hostname))
		assert.True(t, IsValidRedirectURI("https://app.example.com", allowed, hostname))
	})

	t.Run("explicit origins invalid redirect", func(t *testing.T) {
		allowed := []string{"https://app.example.com"}
		assert.False(t, IsValidRedirectURI("https://attacker.com", allowed, hostname))
		assert.False(t, IsValidRedirectURI("https://app.example.com.evil.com", allowed, hostname))
	})

	t.Run("wildcard subdomain origins", func(t *testing.T) {
		allowed := []string{"*.example.com"}
		// Valid subdomains
		assert.True(t, IsValidRedirectURI("https://api.example.com/cb", allowed, hostname))
		assert.True(t, IsValidRedirectURI("https://auth.example.com/login", allowed, hostname))
		assert.True(t, IsValidRedirectURI("https://app.staging.example.com/cb", allowed, hostname))
		// Root domain not matched by wildcard subdomain
		assert.False(t, IsValidRedirectURI("https://example.com", allowed, hostname))
		// Attacker domains
		assert.False(t, IsValidRedirectURI("https://evil.com", allowed, hostname))
		assert.False(t, IsValidRedirectURI("https://exampleXcom.evil.com", allowed, hostname))
		assert.False(t, IsValidRedirectURI("https://example.com.attacker.com", allowed, hostname))
	})

	t.Run("wildcard subdomain with port", func(t *testing.T) {
		allowed := []string{"*.example.com:8080"}
		assert.True(t, IsValidRedirectURI("https://api.example.com:8080/cb", allowed, hostname))
		assert.False(t, IsValidRedirectURI("https://api.example.com/cb", allowed, hostname))
		assert.False(t, IsValidRedirectURI("https://api.example.com:9090/cb", allowed, hostname))
	})

	t.Run("multiple origins with wildcard subdomain", func(t *testing.T) {
		allowed := []string{"*.myapp.com", "http://localhost:3000"}
		assert.True(t, IsValidRedirectURI("https://auth.myapp.com/cb", allowed, hostname))
		assert.True(t, IsValidRedirectURI("https://api.myapp.com/cb", allowed, hostname))
		assert.True(t, IsValidRedirectURI("http://localhost:3000/cb", allowed, hostname))
		assert.False(t, IsValidRedirectURI("https://attacker.com", allowed, hostname))
		assert.False(t, IsValidRedirectURI("https://myapp.com", allowed, hostname))
	})

	t.Run("javascript scheme rejected", func(t *testing.T) {
		assert.False(t, IsValidRedirectURI("javascript:alert(1)", []string{"*"}, hostname))
	})

	t.Run("data scheme rejected", func(t *testing.T) {
		assert.False(t, IsValidRedirectURI("data:text/html,<h1>evil</h1>", []string{"*"}, hostname))
	})

	t.Run("ftp scheme rejected", func(t *testing.T) {
		assert.False(t, IsValidRedirectURI("ftp://evil.com/file", []string{"*"}, hostname))
	})

	t.Run("empty string rejected", func(t *testing.T) {
		assert.False(t, IsValidRedirectURI("", []string{"*"}, hostname))
	})

	t.Run("port matching with localhost", func(t *testing.T) {
		localhostHost := "http://localhost:3000"
		assert.True(t, IsValidRedirectURI("http://localhost:3000/cb", []string{"*"}, localhostHost))
		assert.False(t, IsValidRedirectURI("http://localhost:4000/cb", []string{"*"}, localhostHost))
		assert.False(t, IsValidRedirectURI("http://localhost/cb", []string{"*"}, localhostHost))
	})

	t.Run("standard port normalization", func(t *testing.T) {
		assert.True(t, IsValidRedirectURI("https://myserver.com:443/cb", []string{"*"}, hostname))
	})

	t.Run("path is ignored in origin matching", func(t *testing.T) {
		assert.True(t, IsValidRedirectURI("https://myserver.com/any/path/here", []string{"*"}, hostname))
		assert.True(t, IsValidRedirectURI("https://myserver.com/app/reset-password?foo=bar", []string{"*"}, hostname))
	})

	t.Run("multiple allowed origins", func(t *testing.T) {
		allowed := []string{"https://app.example.com", "http://localhost:3000"}
		assert.True(t, IsValidRedirectURI("https://app.example.com/cb", allowed, hostname))
		assert.True(t, IsValidRedirectURI("http://localhost:3000/cb", allowed, hostname))
		assert.False(t, IsValidRedirectURI("https://attacker.com", allowed, hostname))
	})

	// A bare "*" mixed with explicit origins used to reach originMatches, which
	// re-opens "*" as the regex [^.]* — matching EVERY dotless host and NO dotted
	// one. A browser parses a dotless numeric host as an IPv4 address, so
	// http://3405803783/ is 203.0.113.7: an attacker-controlled server that
	// received the victim's access token from /authorize. The self-host branch
	// only fired for a lone "*", so in a mixed list "*" meant neither "self only"
	// nor "any origin", and the server's own origin was rejected as well.
	t.Run("bare wildcard mixed with explicit origins stays self-only", func(t *testing.T) {
		allowed := []string{"*", "https://app.example.com"}

		assert.False(t, IsValidRedirectURI("http://3405803783/steal", allowed, hostname),
			"decimal IPv4 203.0.113.7 must not pass as a dotless host")
		assert.False(t, IsValidRedirectURI("http://0xcb007107/steal", allowed, hostname),
			"hex IPv4 must not pass as a dotless host")
		assert.False(t, IsValidRedirectURI("http://localhost/steal", allowed, hostname),
			"single-label host must not pass")
		assert.False(t, IsValidRedirectURI("https://attacker.com/steal", allowed, hostname))

		// The explicit entry and the server's own origin still work.
		assert.True(t, IsValidRedirectURI("https://app.example.com/cb", allowed, hostname))
		assert.True(t, IsValidRedirectURI("https://myserver.com/app", allowed, hostname),
			"own origin is allowed by the bare wildcard, as it is for a lone *")
	})

	t.Run("wildcard position in the list does not matter", func(t *testing.T) {
		allowed := []string{"https://app.example.com", "*"}
		assert.False(t, IsValidRedirectURI("http://3405803783/steal", allowed, hostname))
		assert.True(t, IsValidRedirectURI("https://myserver.com/app", allowed, hostname))
	})

	t.Run("subdomain wildcard is unaffected", func(t *testing.T) {
		allowed := []string{"*.acme.com", "*"}
		assert.True(t, IsValidRedirectURI("https://login.acme.com/cb", allowed, hostname))
		assert.False(t, IsValidRedirectURI("https://acme.com/cb", allowed, hostname))
		assert.False(t, IsValidRedirectURI("https://evil-acme.com/cb", allowed, hostname))
		assert.False(t, IsValidRedirectURI("http://3405803783/steal", allowed, hostname))
	})
}

// A bare "*" in a mixed CORS list hit the same originMatches rewrite, so the
// allowlist silently became "dotless hosts only": an operator who wrote "*"
// alongside explicit origins got neither allow-all nor their explicit list, and
// http://localhost / http://3405803783 passed the CORS check.
func TestIsValidOrigin_BareWildcardMixedWithExplicitOrigins(t *testing.T) {
	allowed := []string{"*", "https://app.example.com"}

	t.Run("bare wildcard still means allow any origin", func(t *testing.T) {
		assert.True(t, IsValidOrigin("https://attacker.com", allowed),
			"a bare * is allow-all for CORS, as it is when it is the whole list")
		assert.True(t, IsValidOrigin("https://app.example.com", allowed))
		assert.True(t, IsValidOrigin("http://3405803783", allowed))
	})

	t.Run("without a wildcard the explicit list is still exact", func(t *testing.T) {
		explicit := []string{"https://app.example.com"}
		assert.True(t, IsValidOrigin("https://app.example.com", explicit))
		assert.False(t, IsValidOrigin("http://3405803783", explicit))
		assert.False(t, IsValidOrigin("http://localhost", explicit))
		assert.False(t, IsValidOrigin("https://appXexample.com", explicit))
	})
}

// originMatches must never be handed a bare "*": both callers special-case it
// first, with different meanings. Pin that so a third caller cannot reintroduce
// the [^.]* rewrite.
func TestOriginMatches_RefusesBareWildcard(t *testing.T) {
	assert.False(t, originMatches("*", "3405803783"))
	assert.False(t, originMatches("*", "localhost"))
	assert.False(t, originMatches("*", "app.example.com"))
}

// A configured entry is a wildcard by its NORMALISED value, not its raw text.
// Every spelling below collapses to "*" once the scheme, path, default port and
// surrounding space are stripped. The first fix guarded the raw string, so all
// but the literal "*" reached the regex splice, became [^.]*, and matched any
// dotless host — a browser reads http://3405803783 as 203.0.113.7, so an
// attacker who writes their address as one decimal number received the victim's
// tokens. Same CVE, different spelling.
func TestWildcardSpellingsCannotReachTheRegex(t *testing.T) {
	const evil = "http://3405803783/steal"
	const host = "https://myserver.com"
	for _, cfg := range []string{"*", " *", "* ", "https://*", "http://*", "*/", "*/callback", "*:80", "*:443"} {
		t.Run(cfg, func(t *testing.T) {
			assert.True(t, IsWildcardOrigin(cfg), "%q must be recognised as a wildcard", cfg)
			assert.False(t, originMatches(cfg, normalizeOrigin(evil)),
				"%q must never be spliced into a regex", cfg)
			assert.False(t, IsValidRedirectURI(evil, []string{"https://app.example.com", cfg}, host),
				"%q beside an explicit origin must not permit a dotless host", cfg)
			assert.True(t, IsValidRedirectURI(host, []string{"https://app.example.com", cfg}, host),
				"%q must still permit the server's own hostname", cfg)
		})
	}
}

// A wildcard with no literal dot to anchor it is refused outright. "**" and
// "*:3000" are not spellings of "*" — they normalise to themselves — but they
// splice to [^.]*[^.]* and [^.]*:3000, which match a dotless host just the same.
func TestUnanchoredWildcardIsRefused(t *testing.T) {
	const host = "https://myserver.com"
	for cfg, candidate := range map[string]string{
		"**":     "http://3405803783/steal",
		"*:3000": "http://3405803783:3000/steal",
		"*":      "http://3405803783/steal",
	} {
		assert.False(t, originMatches(cfg, normalizeOrigin(candidate)),
			"%q must not match the dotless host %q", cfg, candidate)
		assert.False(t, IsValidRedirectURI(candidate, []string{"https://app.example.com", cfg}, host),
			"%q must not permit %q", cfg, candidate)
	}
}

// The dot-anchoring rule must not narrow the wildcard forms the allowlist
// documents and the rest of this suite relies on.
func TestAnchoredWildcardsStillWork(t *testing.T) {
	assert.True(t, originMatches("*.example.com", "sub.example.com"))
	assert.True(t, originMatches("*.example.com:8080", "sub.example.com:8080"))
	assert.True(t, originMatches("app*.acme.com", "app42.acme.com"))
	assert.False(t, originMatches("*.example.com", "example.com"), "bare domain is not a subdomain")
	assert.False(t, originMatches("*.example.com", "evil-example.com"), "dots must stay escaped")
}

// The wildcard predicate is shared with the CORS/CSRF middleware and the startup
// warning. When each had its own `o == "*"`, pflag's untrimmed " *" made them
// disagree: validators said allow-all while isWildcardOrigins said explicit
// allowlist, so CORS reflected any origin WITH credentials. Pin that they agree.
func TestWildcardPredicateIsUnambiguous(t *testing.T) {
	for _, s := range []string{"*", " *", "* ", "\t*", "https://*", "*:443"} {
		assert.True(t, IsWildcardOrigin(s), "%q is a wildcard", s)
	}
	for _, s := range []string{"https://app.acme.com", "*.acme.com", "**", "*:3000", ""} {
		assert.False(t, IsWildcardOrigin(s), "%q is NOT a bare wildcard", s)
	}
}

// Backward compatibility. The dot-anchoring rule above refuses a wildcard that
// could match a dotless host, so it has to be shown NOT to narrow the configs
// operators already run. Dotless hosts are legitimate and common — "localhost",
// and in-cluster service names like "frontend" or "authorizer" — and they keep
// working, because an entry with no "*" is compared as a literal string and
// never goes near the regex. Only a WILDCARD that relied on matching a dotless
// host changes, and that match is the vulnerability itself.
func TestBackwardCompatibleConfigs(t *testing.T) {
	const host = "https://auth.example.com"
	for _, tc := range []struct {
		cfg, candidate, why string
	}{
		{"http://localhost:3000", "http://localhost:3000/cb", "explicit localhost with port"},
		{"localhost:3000", "http://localhost:3000/cb", "explicit localhost, schemeless entry"},
		{"http://frontend:3000", "http://frontend:3000/cb", "in-cluster service name"},
		{"http://authorizer", "http://authorizer/cb", "bare in-cluster name, default port"},
		{"*.example.com", "https://app.example.com/cb", "subdomain wildcard"},
		{"*.example.com:8080", "https://app.example.com:8080/cb", "subdomain wildcard with port"},
		{"https://app*.acme.com", "https://app42.acme.com/cb", "mid-label wildcard"},
		{"https://app.example.com", "https://app.example.com/cb", "exact origin"},
	} {
		t.Run(tc.why, func(t *testing.T) {
			assert.True(t, IsValidRedirectURI(tc.candidate, []string{tc.cfg}, host),
				"%q must still permit %q", tc.cfg, tc.candidate)
			assert.True(t, IsValidRedirectURI(tc.candidate, []string{"https://other.example.com", tc.cfg}, host),
				"%q must still permit %q beside another entry", tc.cfg, tc.candidate)
			assert.True(t, IsValidOrigin(tc.candidate, []string{tc.cfg}),
				"%q must still permit CORS from %q", tc.cfg, tc.candidate)
		})
	}
}

// The default config is untouched: "*" alone still means "redirects to my own
// hostname only, CORS open". This is the behaviour the pre-fix one-element
// special case gave, and most deployments run exactly this.
func TestDefaultWildcardConfigUnchanged(t *testing.T) {
	const host = "https://auth.example.com"
	assert.True(t, IsValidRedirectURI("https://auth.example.com/cb", []string{"*"}, host))
	assert.False(t, IsValidRedirectURI("https://evil.com/cb", []string{"*"}, host))
	assert.True(t, IsValidOrigin("https://anything.example.net", []string{"*"}))
	// An empty list is still treated as ["*"].
	assert.True(t, IsValidRedirectURI("https://auth.example.com/cb", nil, host))
	assert.False(t, IsValidRedirectURI("https://evil.com/cb", nil, host))
}
