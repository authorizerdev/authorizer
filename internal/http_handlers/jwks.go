package http_handlers

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/crypto"
)

// generateJWKFromKey generates a JWK for the given algorithm + public
// key + client ID. HMAC (symmetric) keys are never exposed; returns an
// empty string in that case. Kept as a pure function so both the primary
// and secondary keys can be processed uniformly.
func generateJWKFromKey(algo, jwtPublicKey, kidSuffix, clientID string) (string, error) {
	// HMAC keys must never be exposed via JWKS.
	if crypto.IsRSA(algo) {
		publicKeyInstance, err := crypto.ParseRsaPublicKeyFromPemStr(jwtPublicKey)
		if err != nil {
			return "", err
		}
		return crypto.GetPubJWK(algo, clientID+kidSuffix, publicKeyInstance)
	}
	if crypto.IsECDSA(algo) {
		publicKeyInstance, err := crypto.ParseEcdsaPublicKeyFromPemStr(jwtPublicKey)
		if err != nil {
			return "", err
		}
		return crypto.GetPubJWK(algo, clientID+kidSuffix, publicKeyInstance)
	}
	return "", nil
}

// generateJWKBasedOnEnv generates the JWK for the primary key.
// Retained for backward-compat with any callers that only want the
// primary; the JWKsHandler below handles both keys.
func (h *httpProvider) generateJWKBasedOnEnv() (string, error) {
	return generateJWKFromKey(h.JWTType, h.JWTPublicKey, "", h.ClientID)
}

func (h *httpProvider) JWKsHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "JWKsHandler").Logger()
	return func(c *gin.Context) {
		var keys []map[string]string

		// Primary key.
		primaryJWK, err := generateJWKFromKey(h.JWTType, h.JWTPublicKey, "", h.ClientID)
		if err != nil {
			log.Debug().Err(err).Msg("Error generating primary JWK")
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if primaryJWK != "" {
			var data map[string]string
			if err := json.Unmarshal([]byte(primaryJWK), &data); err != nil {
				log.Debug().Err(err).Msg("Error unmarshalling primary JWK")
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			keys = append(keys, data)
		}

		// Secondary key (optional). Only published when both algorithm
		// and public key are configured. HMAC secondary keys are
		// silently dropped by generateJWKFromKey.
		if h.JWTSecondaryType != "" && h.JWTSecondaryPublicKey != "" {
			// Append "-secondary" to the kid to guarantee uniqueness even
			// when primary and secondary use the same algorithm + client ID.
			secondaryJWK, err := generateJWKFromKey(h.JWTSecondaryType, h.JWTSecondaryPublicKey, "-secondary", h.ClientID)
			if err != nil {
				log.Debug().Err(err).Msg("Error generating secondary JWK - ignoring secondary")
			} else if secondaryJWK != "" {
				var data map[string]string
				if err := json.Unmarshal([]byte(secondaryJWK), &data); err != nil {
					log.Debug().Err(err).Msg("Error unmarshalling secondary JWK - ignoring secondary")
				} else {
					keys = append(keys, data)
				}
			}
		}

		if keys == nil {
			// Ensure JSON emits [] not null.
			keys = []map[string]string{}
		}
		c.JSON(200, gin.H{"keys": keys})
	}
}
