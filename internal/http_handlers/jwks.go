package http_handlers

import (
	"encoding/json"
	"net/http"

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
	return func(c *gin.Context) {
		log := h.Log.With().Str("func", "JWKsHandler").Logger()
		var keys []map[string]string

		// Primary key.
		primaryJWK, err := generateJWKFromKey(h.JWTType, h.JWTPublicKey, "", h.ClientID)
		if err != nil {
			// Server-side fault: full failure to publish the JWK set.
			// Log at Error so production operators (with debug filtered
			// out) can see this. Return a generic OAuth2-style error to
			// avoid leaking parser internals to clients.
			log.Error().Err(err).Msg("failed to generate primary JWK")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":             "server_error",
				"error_description": "failed to publish JWK set",
			})
			return
		}
		if primaryJWK != "" {
			var data map[string]string
			if err := json.Unmarshal([]byte(primaryJWK), &data); err != nil {
				log.Error().Err(err).Msg("failed to unmarshal primary JWK")
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":             "server_error",
					"error_description": "failed to publish JWK set",
				})
				return
			}
			keys = append(keys, data)
		}

		// Secondary key (optional). Only published when both algorithm
		// and public key are configured. HMAC secondary keys are
		// silently dropped by generateJWKFromKey. Failures here are
		// degraded service (primary still served), so log at Warn.
		if h.JWTSecondaryType != "" && h.JWTSecondaryPublicKey != "" {
			// Append "-secondary" to the kid to guarantee uniqueness even
			// when primary and secondary use the same algorithm + client ID.
			secondaryJWK, err := generateJWKFromKey(h.JWTSecondaryType, h.JWTSecondaryPublicKey, "-secondary", h.ClientID)
			if err != nil {
				log.Warn().Err(err).Msg("failed to generate secondary JWK - ignoring secondary")
			} else if secondaryJWK != "" {
				var data map[string]string
				if err := json.Unmarshal([]byte(secondaryJWK), &data); err != nil {
					log.Warn().Err(err).Msg("failed to unmarshal secondary JWK - ignoring secondary")
				} else {
					keys = append(keys, data)
				}
			}
		}

		if keys == nil {
			// Ensure JSON emits [] not null.
			keys = []map[string]string{}
		}
		c.JSON(http.StatusOK, gin.H{"keys": keys})
	}
}
