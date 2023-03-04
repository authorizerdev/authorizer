package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/parsers"
)

// OpenIDConfigurationHandler handler for open-id configurations
func OpenIDConfigurationHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		issuer := parsers.GetHost(c)
		jwtType, _ := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJwtType)

		c.JSON(200, gin.H{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"token_endpoint":                        issuer + "/oauth/token",
			"userinfo_endpoint":                     issuer + "/userinfo",
			"jwks_uri":                              issuer + "/.well-known/jwks.json",
			"registration_endpoint":                 issuer + "/app",
			"response_types_supported":              []string{"code", "token", "id_token"},
			"scopes_supported":                      []string{"openid", "email", "profile"},
			"response_modes_supported":              []string{"query", "fragment", "form_post", "web_message"},
			"subject_types_supported":               "public",
			"id_token_signing_alg_values_supported": []string{jwtType},
			"claims_supported":                      []string{"aud", "exp", "iss", "iat", "sub", "given_name", "family_name", "middle_name", "nickname", "preferred_username", "picture", "email", "email_verified", "roles", "role", "gender", "birthdate", "phone_number", "phone_number_verified", "nonce", "updated_at", "created_at", "revoked_timestamp", "login_method", "signup_methods", "token_type"},
		})
	}
}
