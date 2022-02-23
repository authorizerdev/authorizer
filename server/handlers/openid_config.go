package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/utils"
)

// OpenIDConfigurationHandler handler for open-id configurations
func OpenIDConfigurationHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.Contains(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtType), "HS") {
			c.JSON(400, gin.H{"error": "openid not supported for HSA algorithm"})
			return
		}

		issuer := utils.GetHost(c)
		jwtType := envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtType)

		c.JSON(200, gin.H{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"token_endpoint":                        issuer + "/oauth/token",
			"userinfo_endpoint":                     issuer + "/userinfo",
			"jwks_uri":                              issuer + "/jwks.json",
			"response_types_supported":              []string{"code", "token", "id_token", "code token", "code id_token", "token id_token", "code token id_token"},
			"scopes_supported":                      []string{"openid", "email", "profile", "email_verified", "given_name", "family_name", "nick_name", "picture"},
			"response_modes_supported":              []string{"query", "fragment", "form_post"},
			"id_token_signing_alg_values_supported": []string{jwtType},
			"claims_supported":                      []string{"aud", "exp", "iss", "iat", "sub", "given_name", "family_name", "middle_name", "nickname", "preferred_username", "picture", "email", "email_verified", "roles", "gender", "birthdate", "phone_number", "phone_number_verified"},
		})
	}
}
