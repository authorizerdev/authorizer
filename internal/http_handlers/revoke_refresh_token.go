package http_handlers

import (
	"net/http"
	"strings"

	"github.com/authorizerdev/authorizer/internal/constants"
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

		if h.Config.ClientID != clientID {
			log.Debug().Str("client_id", clientID).Msg("Client ID is invalid")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_client",
				"error_description": "Client authentication failed",
			})
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
		if err != nil || existingToken == "" || existingToken != tokenValue {
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
		h.logAuditEvent(gc, constants.AuditTokenRevokedEvent, AuditLogOpts{
			ActorID:      userID,
			ActorType:    "user",
			ResourceType: "token",
			ResourceID:   userID,
		})

		gc.JSON(http.StatusOK, gin.H{})
	}
}
