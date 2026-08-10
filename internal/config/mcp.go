package config

import (
	"net/url"
	"strings"
)

// MCPResourcePath is the single path the MCP transport is served on. Fixed, not
// configurable: it is baked into the canonical resource URI that clients name in
// their RFC 8707 `resource` parameter and that tokens carry as `aud`, so making
// it an operator knob would only create ways for the three places that must
// agree to disagree.
const MCPResourcePath = "/mcp"

// CanonicalURL returns the operator-configured --url reduced to scheme+host,
// with path, query, fragment, userinfo and any trailing slash stripped. Empty
// when --url is unset or is not a usable http(s) origin.
//
// Normalization matches parsers.SetTrustedURL exactly, and that is the whole
// point: parsers.GetHost returns the sanitized value, and it is the sanitized
// value that gets stamped as every token's `iss` claim and used to build every
// self-referential URL the discovery documents publish. Anything derived from
// the RAW --url instead lands one normalization behind and disagrees with the
// tokens the same server mints — an operator running `--url
// https://auth.example.com/auth` would publish an authorization server and a
// jwks_uri that both 404, and an issuer that no token matches, while startup
// reported everything fine.
func (c *Config) CanonicalURL() string {
	if c == nil {
		return ""
	}
	u, err := url.Parse(strings.TrimSpace(c.AuthorizerURL))
	if err != nil || u.User != nil || u.Host == "" {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return strings.TrimSuffix(u.Scheme+"://"+u.Host, "/")
}

// MCPResource returns the canonical resource identifier of this deployment's MCP
// server: "<canonical --url>/mcp". Empty when --url is unset.
//
// This is the ONE place the canonical form is computed. Three things must agree
// on it or the surface is either unreachable or unsafe:
//
//   - the RFC 9728 protected resource metadata `resource` field, which is what
//     clients read and then send as the RFC 8707 `resource` parameter;
//   - the `aud` comparison in token.ValidateMCPAccessToken;
//   - the documentation operators copy into their client config.
//
// It is derived from the operator-configured --url and never from a request.
// parsers.GetHost falls back to request headers when --url is unset, and an
// audience check against a header the caller controls authenticates anyone: an
// attacker would simply send X-Authorizer-URL naming whatever audience their
// token already has. That is why this returns empty rather than guessing, and
// why startup refuses --mcp-enabled without --url.
func (c *Config) MCPResource() string {
	base := c.CanonicalURL()
	if base == "" {
		return ""
	}
	return base + MCPResourcePath
}
