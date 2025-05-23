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
	jwtSecret := h.JWTSecret
	jwtPublicKey := h.JWTPublicKey
	var err error
	// check if jwt secret is provided
	if crypto.IsHMACA(algo) {
		jwk, err = crypto.GetPubJWK(algo, clientID, []byte(jwtSecret))
		if err != nil {
			return "", err
		}
	}

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
		var data map[string]string
		jwk, err := h.generateJWKBasedOnEnv()
		if err != nil {
			log.Debug().Err(err).Msg("Error generating JWK")
			c.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
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
