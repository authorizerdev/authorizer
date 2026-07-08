package http_handlers

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/service/clientauth"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/gin-gonic/gin"
)

// RevokeRefreshTokenHandler handler to revoke refresh token
// Implements RFC 7009 - OAuth 2.0 Token Revocation
// Accepts both application/x-www-form-urlencoded and application/json
func (h *httpProvider) RevokeRefreshTokenHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "RevokeRefreshTokenHandler").Logger()
	return func(gc *gin.Context) {
		// RFC 7009 §2.1: Accept both form-encoded and JSON for backward compatibility
		var tokenValue, clientID, tokenTypeHint string

		contentType := gc.ContentType()
		if strings.Contains(contentType, "application/json") {
			var reqBody map[string]string
			if err := gc.BindJSON(&reqBody); err != nil {
				log.Debug().Err(err).Msg("failed to bind json")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_request",
					"error_description": "Unable to parse request body",
				})
				return
			}
			// Support both "token" (RFC 7009 standard) and "refresh_token" (backward compat)
			tokenValue = strings.TrimSpace(reqBody["token"])
			if tokenValue == "" {
				tokenValue = strings.TrimSpace(reqBody["refresh_token"])
			}
			tokenTypeHint = strings.TrimSpace(reqBody["token_type_hint"])
			clientID = strings.TrimSpace(reqBody["client_id"])
		} else {
			// application/x-www-form-urlencoded (RFC 7009 §2.1 standard)
			tokenValue = strings.TrimSpace(gc.PostForm("token"))
			if tokenValue == "" {
				tokenValue = strings.TrimSpace(gc.PostForm("refresh_token"))
			}
			tokenTypeHint = strings.TrimSpace(gc.PostForm("token_type_hint"))
			clientID = strings.TrimSpace(gc.PostForm("client_id"))
		}

		// Fall back to header for client_id (backward compatibility)
		if clientID == "" {
			clientID = gc.Request.Header.Get("x-authorizer-client-id")
		}

		// Also support HTTP Basic Auth for client authentication
		if clientID == "" {
			clientID, _, _ = gc.Request.BasicAuth()
		}

		if clientID == "" {
			log.Debug().Msg("Client ID is missing")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "The client_id parameter is required",
			})
			return
		}

		// RFC 7009 §2.1: authenticate the calling client through the shared registry
		// resolver (RFC 6749 §2.3) with refresh_token-grant semantics — the client_id
		// is authenticated against the authoritative registry (the reserved client via
		// the Config fallback), but the secret is not required. This preserves the
		// pre-registry behavior where revocation clients present only a client_id and
		// prove possession of the token; token-ownership (checked after parsing) is the
		// real protection.
		_, _, hasBasicAuth := gc.Request.BasicAuth()
		resolvedClient, authErr := h.clientAuthProvider.ResolveClient(gc, clientauth.ResolveParams{
			BodyClientID: clientID,
		})
		if authErr != nil {
			log.Debug().Err(authErr).Str("client_id", clientID).Msg("client authentication failed")
			// Non-basic invalid_client maps to 401 on the revocation endpoint,
			// preserving the pre-registry response shape.
			respondResourceClientAuthError(gc, authErr, hasBasicAuth, http.StatusUnauthorized)
			return
		}

		// Validate token_type_hint if provided
		if tokenTypeHint != "" && tokenTypeHint != "refresh_token" && tokenTypeHint != "access_token" {
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "unsupported_token_type",
				"error_description": "The given token_type_hint is not supported",
			})
			return
		}

		// RFC 7009 §2.2: Invalid tokens do NOT cause error responses.
		// The server responds with HTTP 200 for both valid and invalid tokens.
		if tokenValue == "" {
			log.Debug().Msg("Token is empty, returning 200 per RFC 7009")
			gc.JSON(http.StatusOK, gin.H{})
			return
		}

		claims, err := h.TokenProvider.ParseJWTToken(tokenValue)
		if err != nil {
			// RFC 7009 §2.2: Invalid token - return 200
			log.Debug().Err(err).Msg("Failed to parse token, returning 200 per RFC 7009")
			gc.JSON(http.StatusOK, gin.H{})
			return
		}

		// RFC 7009 token-ownership: only revoke a token issued to the authenticated
		// client. A token's aud is its client's client_id; when it does not match the
		// resolved caller, respond 200 {} (RFC 7009 §2.2, no oracle) without touching
		// the session — one client must not be able to revoke another client's token.
		if !audienceMatchesIntrospect(claims["aud"], resolvedClient.ClientID) {
			log.Debug().Msg("Token not issued to authenticated client, returning 200 per RFC 7009")
			gc.JSON(http.StatusOK, gin.H{})
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			// RFC 7009 §2.2: Invalid token - return 200
			gc.JSON(http.StatusOK, gin.H{})
			return
		}

		loginMethod := claims["login_method"]
		sessionToken := userID
		if lm, ok := loginMethod.(string); ok && lm != "" {
			sessionToken = lm + ":" + userID
		}

		nonce, ok := claims["nonce"].(string)
		if !ok || nonce == "" {
			// RFC 7009 §2.2: Invalid token - return 200
			gc.JSON(http.StatusOK, gin.H{})
			return
		}

		existingToken, err := h.MemoryStoreProvider.GetUserSession(sessionToken, constants.TokenTypeRefreshToken+"_"+nonce)
		// RFC 7009 §2.1: use constant-time comparison to prevent timing attacks
		if err != nil || existingToken == "" || subtle.ConstantTimeCompare([]byte(existingToken), []byte(tokenValue)) != 1 {
			// RFC 7009 §2.2: Token not found or mismatch - return 200
			log.Debug().Msg("Token not found or mismatch, returning 200 per RFC 7009")
			gc.JSON(http.StatusOK, gin.H{})
			return
		}

		if err := h.MemoryStoreProvider.DeleteUserSession(sessionToken, nonce); err != nil {
			log.Debug().Err(err).Msg("failed to delete user session")
			// RFC 7009 §2.2.1: Use 503 if server cannot handle request
			gc.JSON(http.StatusServiceUnavailable, gin.H{
				"error":             "server_error",
				"error_description": "The server encountered an unexpected error",
			})
			return
		}
		metrics.RecordAuthEvent(metrics.EventTokenRevoke, metrics.StatusSuccess)
		h.AuditProvider.LogEvent(audit.Event{
			Action:       constants.AuditTokenRevokedEvent,
			ActorID:      userID,
			ActorType:    constants.AuditActorTypeUser,
			ResourceType: constants.AuditResourceTypeToken,
			ResourceID:   userID,
			IPAddress:    utils.GetIP(gc.Request),
			UserAgent:    utils.GetUserAgent(gc.Request),
		})

		gc.JSON(http.StatusOK, gin.H{})
	}
}
