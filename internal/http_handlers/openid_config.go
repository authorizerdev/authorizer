package http_handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/parsers"
)

// The following package-level variables are the single source of truth for
// the static portions of the OpenID Connect Discovery 1.0 metadata document
// served from /.well-known/openid-configuration. Promoting them out of the
// handler body makes the metadata trivially testable and gives operators a
// single, well-documented place to add a new claim, scope, response mode, or
// auth method.

// discoveryResponseTypesSupported lists every response_type value the
// authorize endpoint accepts, including the OIDC Core §3.3 hybrid flows.
var discoveryResponseTypesSupported = []string{
	"code", "token", "id_token",
	"code id_token", "code token", "code id_token token",
	"id_token token",
}

// discoveryGrantTypesSupported is kept consistent with
// discoveryResponseTypesSupported. The "implicit" grant covers
// response_type=token / id_token / id_token token.
var discoveryGrantTypesSupported = []string{
	"authorization_code", "refresh_token", "implicit",
}

// discoverySubjectTypesSupported advertises only the "public" subject type:
// the same `sub` value is returned to every relying party.
var discoverySubjectTypesSupported = []string{"public"}

// discoveryScopesSupported lists every scope honoured at /authorize and
// /token. Adding a new scope to the issuer requires updating this list.
var discoveryScopesSupported = []string{
	"openid", "email", "profile", "offline_access",
}

// discoveryClaimsSupported lists every claim that may appear in an ID token
// or /userinfo response. This MUST stay aligned with what the token issuer
// actually emits — advertising a claim that is never produced violates OIDC
// Discovery §3 and breaks spec-compliant relying parties that branch on
// claims_supported. The token issuer emits "roles" (plural); "role"
// (singular) is intentionally NOT included.
var discoveryClaimsSupported = []string{
	"aud", "exp", "iss", "iat", "sub",
	"given_name", "family_name", "middle_name", "nickname", "preferred_username",
	"picture", "email", "email_verified", "roles",
	"gender", "birthdate", "phone_number", "phone_number_verified",
	"nonce", "updated_at", "created_at", "auth_time", "amr", "acr",
	"at_hash", "c_hash",
}

// discoveryResponseModesSupported lists only IANA-registered OAuth 2.0
// Authorization Endpoint Response Modes. Vendor extensions such as
// "web_message" are intentionally excluded for spec compliance.
var discoveryResponseModesSupported = []string{
	"query", "fragment", "form_post",
}

// discoveryCodeChallengeMethodsSupported advertises the PKCE methods
// accepted at /authorize. Authorizer requires S256 and rejects "plain".
var discoveryCodeChallengeMethodsSupported = []string{"S256"}

// discoveryTokenEndpointAuthMethodsSupported lists the client authentication
// methods accepted at /oauth/token, /oauth/revoke, and /oauth/introspect.
var discoveryTokenEndpointAuthMethodsSupported = []string{
	"client_secret_basic", "client_secret_post",
}

// OpenIDConfigurationHandler handler for open-id configurations
// Implements OpenID Connect Discovery 1.0
func (h *httpProvider) OpenIDConfigurationHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Per-function logger kept for parity with other HTTP handlers and
		// to support future structured diagnostics; the discovery document
		// itself is static enough that no per-request logging is currently
		// emitted.
		_ = h.Log.With().Str("func", "OpenIDConfigurationHandler").Logger()

		issuer := parsers.GetHost(c)

		// id_token_signing_alg_values_supported MUST advertise every alg
		// the issuer can produce. During key rotation operators may run a
		// primary + secondary pair (e.g. RS256 primary, ES256 secondary)
		// where JWKS publishes both keys. Advertising only the primary
		// causes spec-compliant RPs to reject tokens minted with the
		// secondary key.
		signingAlgs := buildSigningAlgs(h.Config.JWTType, h.Config.JWTSecondaryType)

		// Back-channel logout is advertised only when the operator has
		// configured a backchannel_logout_uri — avoids lying to RPs.
		backchannelSupported := h.Config.BackchannelLogoutURI != ""

		resp := gin.H{
			// REQUIRED fields (OIDC Discovery §3)
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"jwks_uri":                              issuer + "/.well-known/jwks.json",
			"response_types_supported":              discoveryResponseTypesSupported,
			"subject_types_supported":               discoverySubjectTypesSupported,
			"id_token_signing_alg_values_supported": signingAlgs,

			// RECOMMENDED fields
			"token_endpoint":                                issuer + "/oauth/token",
			"userinfo_endpoint":                             issuer + "/userinfo",
			"scopes_supported":                              discoveryScopesSupported,
			"claims_supported":                              discoveryClaimsSupported,
			"response_modes_supported":                      discoveryResponseModesSupported,
			"grant_types_supported":                         discoveryGrantTypesSupported,
			"token_endpoint_auth_methods_supported":         discoveryTokenEndpointAuthMethodsSupported,
			"code_challenge_methods_supported":              discoveryCodeChallengeMethodsSupported,
			"revocation_endpoint":                           issuer + "/oauth/revoke",
			"revocation_endpoint_auth_methods_supported":    discoveryTokenEndpointAuthMethodsSupported,
			"introspection_endpoint":                        issuer + "/oauth/introspect",
			"introspection_endpoint_auth_methods_supported": discoveryTokenEndpointAuthMethodsSupported,
			"end_session_endpoint":                          issuer + "/logout",
			"backchannel_logout_supported":                  backchannelSupported,
			"backchannel_logout_session_supported":          backchannelSupported,
			"claims_parameter_supported":                    false,
			"request_parameter_supported":                   false,
			"request_uri_parameter_supported":               false,
		}

		c.JSON(200, resp)
	}
}

// buildSigningAlgs returns the deduplicated list of JWT signing algorithms
// advertised in id_token_signing_alg_values_supported. It always includes
// the configured primary alg, optionally the secondary alg when set, and
// guarantees RS256 is present (OIDC Discovery 1.0 §3 MUST).
func buildSigningAlgs(primary, secondary string) []string {
	out := make([]string, 0, 3)
	seen := make(map[string]struct{}, 3)
	add := func(alg string) {
		if alg == "" {
			return
		}
		if _, ok := seen[alg]; ok {
			return
		}
		seen[alg] = struct{}{}
		out = append(out, alg)
	}
	add(primary)
	add(secondary)
	add("RS256")
	return out
}
