package http_handlers

import (
	"encoding/json"

	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/gin-gonic/gin"
)

// generateJWKBasedOnEnv generates JWK based on root args config
// make sure clientID, jwtType, jwtSecret / public & private key pair is set
// this is called while initializing app / when env is updated
func (h *httpProvider) generateJWKBasedOnEnv() (string, error) {
	jwk := ""
	algo := h.JWTType
	clientID := h.ClientID
	jwtPublicKey := h.JWTPublicKey
	// HMAC (symmetric) keys must never be exposed via the public JWKS endpoint.
	// Publishing the secret would allow anyone to forge tokens.
	// Only asymmetric public keys (RSA, ECDSA) are included.

	if crypto.IsRSA(algo) {
		publicKeyInstance, err := crypto.ParseRsaPublicKeyFromPemStr(jwtPublicKey)
		if err != nil {
			return "", err
		}

		jwk, err = crypto.GetPubJWK(algo, clientID, publicKeyInstance)
		if err != nil {
			return "", err
		}
	}

	if crypto.IsECDSA(algo) {
		publicKeyInstance, err := crypto.ParseEcdsaPublicKeyFromPemStr(jwtPublicKey)
		if err != nil {
			return "", err
		}

		jwk, err = crypto.GetPubJWK(algo, clientID, publicKeyInstance)
		if err != nil {
			return "", err
		}
	}

	return jwk, nil
}

func (h *httpProvider) JWKsHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "JWKsHandler").Logger()
	return func(c *gin.Context) {
		jwk, err := h.generateJWKBasedOnEnv()
		if err != nil {
			log.Debug().Err(err).Msg("Error generating JWK")
			c.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
		// HMAC-only configurations have no public key to expose;
		// return an empty key set per RFC 7517.
		if jwk == "" {
			c.JSON(200, gin.H{
				"keys": []map[string]string{},
			})
			return
		}
		var data map[string]string
		err = json.Unmarshal([]byte(jwk), &data)
		if err != nil {
			log.Debug().Err(err).Msg("Error unmarshalling JWK")
			c.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(200, gin.H{
			"keys": []map[string]string{
				data,
			},
		})
	}
}
