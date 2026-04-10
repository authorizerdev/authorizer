package http_handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/parsers"
)

// OpenIDConfigurationHandler handler for open-id configurations
// Implements OpenID Connect Discovery 1.0
func (h *httpProvider) OpenIDConfigurationHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		issuer := parsers.GetHost(c)
		jwtType := h.Config.JWTType

		// OIDC Discovery 1.0: id_token_signing_alg_values_supported MUST include RS256
		signingAlgs := []string{jwtType}
		if jwtType != "RS256" {
			signingAlgs = append(signingAlgs, "RS256")
		}

		// Hybrid flow response_types per OIDC Core §3.3.
		responseTypes := []string{
			"code", "token", "id_token",
			"code id_token", "code token", "code id_token token",
			"id_token token",
		}

		// grant_types_supported stays consistent with response_types_supported:
		// "implicit" corresponds to response_type=token / id_token / id_token token.
		grantTypes := []string{"authorization_code", "refresh_token", "implicit"}

		// Back-channel logout is advertised only when the operator has
		// configured a backchannel_logout_uri — avoids lying to RPs.
		backchannelSupported := h.Config.BackchannelLogoutURI != ""

		resp := gin.H{
			// REQUIRED fields (OIDC Discovery §3)
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"jwks_uri":                              issuer + "/.well-known/jwks.json",
			"response_types_supported":              responseTypes,
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": signingAlgs,

			// RECOMMENDED fields
			"token_endpoint":                                issuer + "/oauth/token",
			"userinfo_endpoint":                             issuer + "/userinfo",
			"scopes_supported":                              []string{"openid", "email", "profile", "offline_access"},
			"claims_supported":                              []string{"aud", "exp", "iss", "iat", "sub", "given_name", "family_name", "middle_name", "nickname", "preferred_username", "picture", "email", "email_verified", "roles", "role", "gender", "birthdate", "phone_number", "phone_number_verified", "nonce", "updated_at", "created_at", "auth_time", "amr", "acr", "at_hash", "c_hash"},
			"response_modes_supported":                      []string{"query", "fragment", "form_post", "web_message"},
			"grant_types_supported":                         grantTypes,
			"token_endpoint_auth_methods_supported":         []string{"client_secret_basic", "client_secret_post"},
			"code_challenge_methods_supported":              []string{"S256", "plain"},
			"revocation_endpoint":                           issuer + "/oauth/revoke",
			"revocation_endpoint_auth_methods_supported":    []string{"client_secret_basic", "client_secret_post"},
			"introspection_endpoint":                        issuer + "/oauth/introspect",
			"introspection_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
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
