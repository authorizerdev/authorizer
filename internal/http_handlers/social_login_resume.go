package http_handlers

import (
	"net/url"
	"strings"
)

// isAuthorizeResumeTarget reports whether redirectURI is this server's own
// hosted login page (/app, /app/ or /app/signup), which social login may always
// return to (#855).
//
// Why this exists: a social login started from an /authorize request must not
// deliver the authorization response itself. That response belongs to
// /authorize, the one place that checks redirect_uri against the client's
// registered list, runs the consent gate for self-registered and CIMD clients,
// and binds PKCE. Password, passkey and OTP login already work that way: /app
// establishes a session and then replays the original request to /authorize
// (web/app/src/Root.tsx). Social login used to send the browser straight to the
// client's redirect_uri instead, which is why it was checked against the global
// --allowed-origins list, rejected every client on another origin, and skipped
// consent. Returning to /app routes it through the same resumption as every
// other login method.
//
// Returning here is safe for any caller because nothing is delivered on this
// path: the callback sets the session cookie and redirects to this exact URL
// without appending a code, a token or a nonce.
//
// The check is deliberately narrow:
//   - scheme AND host:port must equal this server's own (hostname comes from
//     --url via parsers.GetHost). This is stricter than the allowed-origins
//     normaliser, which ignores the scheme.
//   - the path must be exactly /app, /app/ or /app/signup — not any path on
//     this origin, so the session-carrying redirect cannot be pointed at an
//     API route or another page. Both spellings of /app are needed: /authorize
//     redirects to /app, and the /app route group 301s that to /app/, which is
//     the URL a browser is actually on when it renders the login page.
//   - no userinfo and no fragment. A resuming client puts the /authorize
//     parameters in the query, where the server can see them.
func isAuthorizeResumeTarget(redirectURI, hostname string) bool {
	target, err := url.Parse(strings.TrimSpace(redirectURI))
	if err != nil || target.User != nil || target.Fragment != "" || target.Opaque != "" {
		return false
	}
	self, err := url.Parse(strings.TrimSpace(hostname))
	if err != nil || self.Host == "" {
		return false
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return false
	}
	if !strings.EqualFold(target.Scheme, self.Scheme) {
		return false
	}
	if canonicalHostPort(target) != canonicalHostPort(self) {
		return false
	}
	return target.Path == baseAppPath || target.Path == baseAppPath+"/" || target.Path == signupPath
}

// canonicalHostPort returns u's host in lower case with the scheme's default
// port made explicit, so https://auth.example.com and
// https://auth.example.com:443 compare equal while :8443 does not.
func canonicalHostPort(u *url.URL) string {
	port := u.Port()
	if port == "" {
		switch strings.ToLower(u.Scheme) {
		case "https":
			port = "443"
		case "http":
			port = "80"
		}
	}
	return strings.ToLower(u.Hostname()) + ":" + port
}
