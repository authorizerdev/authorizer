package http_handlers

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/internalerr"
)

// generateJWKFromKey generates the JWK for the given algorithm and public
// key. HMAC (symmetric) keys are never exposed; returns an empty string in
// that case. The kid is the key's RFC 7638 thumbprint (crypto.KeyID), the
// same value the signer stamps into the token header, so kid-strict JWKS
// consumers can select the key. Kept as a pure function so both the primary
// and secondary keys are processed uniformly.
func generateJWKFromKey(algo, jwtPublicKey string) (string, error) {
	var publicKey any
	switch {
	case crypto.IsRSA(algo):
		key, err := crypto.ParseRsaPublicKeyFromPemStr(jwtPublicKey)
		if err != nil {
			return "", err
		}
		publicKey = key
	case crypto.IsECDSA(algo):
		key, err := crypto.ParseEcdsaPublicKeyFromPemStr(jwtPublicKey)
		if err != nil {
			return "", err
		}
		publicKey = key
	default:
		// HMAC keys must never be exposed via JWKS.
		return "", nil
	}
	kid, err := crypto.KeyID(publicKey)
	if err != nil {
		return "", err
	}
	return crypto.GetPubJWK(algo, kid, publicKey)
}

func (h *httpProvider) JWKsHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "JWKsHandler").Logger()
	return func(c *gin.Context) {
		var keys []map[string]string

		// Primary key.
		primaryJWK, err := generateJWKFromKey(h.JWTType, h.JWTPublicKey)
		if err != nil {
			log.Debug().Err(err).Msg("Error generating primary JWK")
			// A key that cannot be rendered is server misconfiguration; its
			// parse error must not be published on an unauthenticated endpoint.
			c.JSON(500, gin.H{"error": publicErrorMessage(&log, internalerr.Wrap("jwks", err))})
			return
		}
		if primaryJWK != "" {
			var data map[string]string
			if err := json.Unmarshal([]byte(primaryJWK), &data); err != nil {
				log.Debug().Err(err).Msg("Error unmarshalling primary JWK")
				c.JSON(500, gin.H{"error": publicErrorMessage(&log, internalerr.Wrap("jwks", err))})
				return
			}
			keys = append(keys, data)
		}

		// Secondary key (optional). Only published when both algorithm
		// and public key are configured. HMAC secondary keys are
		// silently dropped by generateJWKFromKey.
		if h.JWTSecondaryType != "" && h.JWTSecondaryPublicKey != "" {
			// The kid is the key's thumbprint, so primary and secondary are
			// distinct whenever they are different keys.
			secondaryJWK, err := generateJWKFromKey(h.JWTSecondaryType, h.JWTSecondaryPublicKey)
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
