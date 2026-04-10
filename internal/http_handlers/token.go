package http_handlers

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

type RequestBody struct {
	CodeVerifier string `form:"code_verifier" json:"code_verifier"`
	Code         string `form:"code" json:"code"`
	ClientID     string `form:"client_id" json:"client_id"`
	ClientSecret string `form:"client_secret" json:"client_secret"`
	GrantType    string `form:"grant_type" json:"grant_type"`
	RefreshToken string `form:"refresh_token" json:"refresh_token"`
	RedirectURI  string `form:"redirect_uri" json:"redirect_uri"`
}

// TokenHandler to handle /oauth/token requests
// grant type required
func (h *httpProvider) TokenHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "TokenHandler").Logger()
	return func(gc *gin.Context) {
		// RFC 6749 §5.1: token endpoint responses must not be cached.
		gc.Writer.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		gc.Writer.Header().Set("Pragma", "no-cache")

		var reqBody RequestBody
		if err := gc.Bind(&reqBody); err != nil {
			log.Debug().Err(err).Msg("failed to bind request body")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "Failed to parse request body",
			})
			return
		}

		codeVerifier := strings.TrimSpace(reqBody.CodeVerifier)
		code := strings.TrimSpace(reqBody.Code)
		clientID := strings.TrimSpace(reqBody.ClientID)
		grantType := strings.TrimSpace(reqBody.GrantType)
		refreshToken := strings.TrimSpace(reqBody.RefreshToken)
		clientSecret := strings.TrimSpace(reqBody.ClientSecret)

		if grantType == "" {
			grantType = "authorization_code"
		}

		isRefreshTokenGrant := grantType == "refresh_token"
		isAuthorizationCodeGrant := grantType == "authorization_code"

		if !isRefreshTokenGrant && !isAuthorizationCodeGrant {
			log.Debug().Str("grant_type", grantType).Msg("Invalid grant type")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "unsupported_grant_type",
				"error_description": "grant_type is not supported",
			})
			return
		}

		// check if clientID & clientSecret are present as part of
		// authorization header with basic auth
		if clientID == "" && clientSecret == "" {
			clientID, clientSecret, _ = gc.Request.BasicAuth()
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
			metrics.RecordSecurityEvent("invalid_client", "token_endpoint")
			// RFC 6749 §5.2: If client auth fails via HTTP Basic, return 401
			if _, _, hasBasicAuth := gc.Request.BasicAuth(); hasBasicAuth {
				gc.Header("WWW-Authenticate", "Basic realm=\"authorizer\"")
				gc.JSON(http.StatusUnauthorized, gin.H{
					"error":             "invalid_client",
					"error_description": "Client authentication failed",
				})
			} else {
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_client",
					"error_description": "The client_id is invalid",
				})
			}
			return
		}

		var userID string
		var roles, scope []string
		loginMethod := ""
		sessionKey := ""
		oidcNonce := "" // OIDC nonce from the original /authorize request

		if isAuthorizationCodeGrant {
			if code == "" {
				log.Debug().Msg("Code is missing")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_request",
					"error_description": "The code parameter is required for authorization_code grant type",
				})
				return
			}

			// RFC 6749 §4.1.2: Authorization codes MUST be single-use.
			// GetAndRemoveState atomically retrieves and deletes the code
			// to prevent replay via TOCTOU race.
			sessionData, err := h.MemoryStoreProvider.GetAndRemoveState(code)
			if sessionData == "" || err != nil {
				log.Debug().Err(err).Msg("Error getting session data")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_grant",
					"error_description": "The authorization code is invalid or has expired",
				})
				return
			}

			// [0] -> code_challenge (may contain "::method" suffix) or empty
			// [1] -> session cookie
			// [2] -> OIDC nonce from /authorize request (optional)
			// [3] -> redirect_uri from /authorize request (optional, for RFC 6749 §4.1.3)
			sessionDataSplit := strings.Split(sessionData, "@@")

			// RFC 6749 §4.1.3: If redirect_uri was included in the authorization
			// request, the token request MUST include the identical redirect_uri.
			storedRedirectURI := ""
			if len(sessionDataSplit) > 3 {
				storedRedirectURI, _ = url.QueryUnescape(sessionDataSplit[3])
			}
			requestRedirectURI := strings.TrimSpace(reqBody.RedirectURI)
			if storedRedirectURI != "" {
				if requestRedirectURI == "" {
					gc.JSON(http.StatusBadRequest, gin.H{
						"error":             "invalid_request",
						"error_description": "redirect_uri is required when it was included in the authorization request",
					})
					return
				}
				if subtle.ConstantTimeCompare([]byte(requestRedirectURI), []byte(storedRedirectURI)) != 1 {
					gc.JSON(http.StatusBadRequest, gin.H{
						"error":             "invalid_grant",
						"error_description": "redirect_uri does not match the one used in the authorization request",
					})
					return
				}
			}

			// Parse code_challenge and method from stored state.
			// Format: "challenge::method" or just "challenge" (legacy, defaults to plain per RFC 7636 §4.2)
			// or empty string (no PKCE — confidential client).
			storedChallenge := sessionDataSplit[0]
			storedMethod := "plain"
			if idx := strings.LastIndex(storedChallenge, "::"); idx >= 0 {
				storedMethod = storedChallenge[idx+2:]
				storedChallenge = storedChallenge[:idx]
			}

			// RFC 7636 §4.5: If PKCE was used at /authorize, the token request
			// MUST include code_verifier. This is orthogonal to client authentication.
			if storedChallenge != "" && codeVerifier != "" {
				// PKCE was used — verify code_verifier against stored challenge
				switch storedMethod {
				case "S256":
					hash := sha256.New()
					hash.Write([]byte(codeVerifier))
					computed := base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
					// RFC 7636 Appendix B: code_challenge uses BASE64URL without padding.
					// Tolerate clients that send padding ('=') by stripping it before comparison.
					normalizedChallenge := strings.TrimRight(storedChallenge, "=")
					if subtle.ConstantTimeCompare([]byte(computed), []byte(normalizedChallenge)) != 1 {
						gc.JSON(http.StatusBadRequest, gin.H{
							"error":             "invalid_grant",
							"error_description": "The code_verifier does not match the code_challenge",
						})
						return
					}
				case "plain":
					// RFC 7636 §4.6: plain method — code_verifier == code_challenge
					if subtle.ConstantTimeCompare([]byte(codeVerifier), []byte(storedChallenge)) != 1 {
						gc.JSON(http.StatusBadRequest, gin.H{
							"error":             "invalid_grant",
							"error_description": "The code_verifier does not match the code_challenge",
						})
						return
					}
				default:
					gc.JSON(http.StatusBadRequest, gin.H{
						"error":             "invalid_request",
						"error_description": "Unsupported code_challenge_method",
					})
					return
				}
			} else if storedChallenge != "" && codeVerifier == "" {
				// PKCE was used at /authorize but client didn't send code_verifier
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_request",
					"error_description": "code_verifier is required when code_challenge was used",
				})
				return
			} else if storedChallenge == "" && codeVerifier != "" {
				// code_verifier sent but no code_challenge was registered at /authorize.
				// Reject to prevent an attacker from bypassing client_secret by
				// supplying an arbitrary code_verifier.
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_request",
					"error_description": "code_verifier was provided but no code_challenge was registered",
				})
				return
			}

			// RFC 6749 §4.1.3: Confidential clients MUST authenticate regardless
			// of whether PKCE was used. PKCE protects against authorization code
			// interception; client authentication is a separate concern.
			// When no PKCE was used, client_secret is the sole proof of identity.
			if storedChallenge == "" && codeVerifier == "" {
				// No PKCE — client_secret is required
				if clientSecret == "" {
					gc.JSON(http.StatusBadRequest, gin.H{
						"error":             "invalid_request",
						"error_description": "Either code_verifier or client_secret is required",
					})
					return
				}
			}
			// Always validate client_secret when provided (confidential client).
			if clientSecret != "" {
				if subtle.ConstantTimeCompare([]byte(clientSecret), []byte(h.Config.ClientSecret)) != 1 {
					log.Debug().Msg("Client secret is invalid")
					gc.JSON(http.StatusUnauthorized, gin.H{
						"error":             "invalid_client",
						"error_description": "Client authentication failed",
					})
					return
				}
			}

			// validate session
			claims, err := h.TokenProvider.ValidateBrowserSession(gc, sessionDataSplit[1])
			if err != nil {
				log.Debug().Err(err).Msg("Error validating session")
				gc.JSON(http.StatusUnauthorized, gin.H{
					"error":             "unauthorized",
					"error_description": "Invalid session data",
				})
				return
			}

			userID = claims.Subject
			roles = claims.Roles
			scope = claims.Scope
			loginMethod = claims.LoginMethod

			// Extract OIDC nonce from stored code data (third @@-separated part).
			if len(sessionDataSplit) > 2 {
				oidcNonce = sessionDataSplit[2]
			}

			sessionKey = userID
			if loginMethod != "" {
				sessionKey = loginMethod + ":" + userID
			}

			// NOTE: Do NOT delete the user's browser session here. The
			// /authorize endpoint already performed session rollover when it
			// created the authorization code. The /oauth/token endpoint is
			// called server-to-server by the RP (Auth0/Okta/Keycloak), not
			// by the user's browser. Deleting the session here would
			// invalidate the cookie the user's browser holds, breaking
			// subsequent session lookups (e.g., GraphQL session query).

		} else {
			// validate refresh token
			if refreshToken == "" {
				log.Debug().Msg("Refresh token is missing")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_request",
					"error_description": "The refresh_token parameter is required for refresh_token grant type",
				})
				return
			}

			claims, err := h.TokenProvider.ValidateRefreshToken(gc, refreshToken)
			if err != nil {
				log.Debug().Err(err).Msg("Error validating refresh token")
				gc.JSON(http.StatusUnauthorized, gin.H{
					"error":             "invalid_grant",
					"error_description": "The refresh token is invalid or has expired",
				})
				return
			}

			sub, ok := claims["sub"].(string)
			if !ok || sub == "" {
				log.Debug().Msg("Invalid subject in refresh token")
				gc.JSON(http.StatusUnauthorized, gin.H{
					"error":             "unauthorized",
					"error_description": "Invalid refresh token",
				})
				return
			}
			userID = sub

			claimLoginMethod := claims["login_method"]
			if rolesVal, ok := claims["roles"].([]interface{}); ok {
				for _, v := range rolesVal {
					roleStr, ok := v.(string)
					if !ok || roleStr == "" {
						log.Debug().Msg("Invalid role claim in refresh token")
						gc.JSON(http.StatusUnauthorized, gin.H{
							"error":             "unauthorized",
							"error_description": "Invalid refresh token",
						})
						return
					}
					roles = append(roles, roleStr)
				}
			} else {
				log.Debug().Msg("Missing roles claim in refresh token")
				gc.JSON(http.StatusUnauthorized, gin.H{
					"error":             "unauthorized",
					"error_description": "Invalid refresh token",
				})
				return
			}

			if scopeVal, ok := claims["scope"].([]interface{}); ok {
				for _, v := range scopeVal {
					scopeStr, ok := v.(string)
					if !ok || scopeStr == "" {
						log.Debug().Msg("Invalid scope claim in refresh token")
						gc.JSON(http.StatusUnauthorized, gin.H{
							"error":             "unauthorized",
							"error_description": "Invalid refresh token",
						})
						return
					}
					scope = append(scope, scopeStr)
				}
			} else {
				log.Debug().Msg("Missing scope claim in refresh token")
				gc.JSON(http.StatusUnauthorized, gin.H{
					"error":             "unauthorized",
					"error_description": "Invalid refresh token",
				})
				return
			}

			sessionKey = userID
			if lm, ok := claimLoginMethod.(string); ok && lm != "" {
				sessionKey = lm + ":" + sessionKey
				loginMethod = lm
			}

			nonce, ok := claims["nonce"].(string)
			if !ok || nonce == "" {
				log.Debug().Msg("Invalid nonce in refresh token")
				gc.JSON(http.StatusUnauthorized, gin.H{
					"error":             "unauthorized",
					"error_description": "Invalid refresh token",
				})
				return
			}

			// remove older refresh token and rotate it for security
			if err := h.MemoryStoreProvider.DeleteUserSession(sessionKey, nonce); err != nil {
				log.Debug().Err(err).Str("session_key", sessionKey).Msg("Failed to delete old session during token refresh")
			}
		}

		if sessionKey == "" {
			log.Debug().Str("session_key", sessionKey).Str("login_method", loginMethod).Msg("Session key not found")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "unauthorized",
				"error_description": "User not found",
			})
			return
		}

		user, err := h.StorageProvider.GetUserByID(gc, userID)
		if err != nil {
			log.Debug().Err(err).Str("user_id", userID).Msg("Error getting user")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "unauthorized",
				"error_description": "User not found",
			})
			return
		}
		hostname := parsers.GetHost(gc)
		nonce := uuid.New().String()
		authToken, err := h.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
			User:        user,
			Roles:       roles,
			Scope:       scope,
			LoginMethod: loginMethod,
			Nonce:       nonce,
			OIDCNonce:   oidcNonce,
			HostName:    hostname,
		})
		if err != nil {
			log.Debug().Err(err).Msg("Error creating auth token")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "unauthorized",
				"error_description": "User not found",
			})
			return
		}

		// For authorization_code grant the user's browser session was already
		// created by /authorize. The token endpoint is called server-to-server
		// by the RP — creating a new browser session here would be orphaned
		// (the cookie goes to the RP, not the user's browser) and deleting /
		// replacing the existing one would invalidate the user's cookie.
		//
		// For refresh_token grant the caller IS the user's browser (or an app
		// holding the refresh token), so we do a full session rollover.
		if isRefreshTokenGrant {
			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error persisting session token")
				gc.JSON(http.StatusServiceUnavailable, gin.H{
					"error":             "temporarily_unavailable",
					"error_description": "Could not complete token issuance",
				})
				return
			}
			cookie.SetSession(gc, authToken.FingerPrintHash, h.Config.AppCookieSecure)
		}

		if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt); err != nil {
			log.Debug().Err(err).Msg("Error persisting access token")
			gc.JSON(http.StatusServiceUnavailable, gin.H{
				"error":             "temporarily_unavailable",
				"error_description": "Could not complete token issuance",
			})
			return
		}

		expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
		if expiresIn <= 0 {
			expiresIn = 1
		}

		res := map[string]interface{}{
			"access_token": authToken.AccessToken.Token,
			"token_type":   "Bearer",
			"id_token":     authToken.IDToken.Token,
			"scope":        strings.Join(scope, " "),
			"roles":        roles,
			"expires_in":   expiresIn,
		}
		if authToken.RefreshToken != nil {
			res["refresh_token"] = authToken.RefreshToken.Token
			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error persisting refresh token")
				gc.JSON(http.StatusServiceUnavailable, gin.H{
					"error":             "temporarily_unavailable",
					"error_description": "Could not complete token issuance",
				})
				return
			}
		}
		if isRefreshTokenGrant {
			metrics.RecordAuthEvent(metrics.EventTokenRefresh, metrics.StatusSuccess)
		} else {
			metrics.RecordAuthEvent(metrics.EventTokenIssued, metrics.StatusSuccess)
		}
		auditAction := constants.AuditTokenIssuedEvent
		if isRefreshTokenGrant {
			auditAction = constants.AuditTokenRefreshedEvent
		}
		h.AuditProvider.LogEvent(audit.Event{
			Action:       auditAction,
			ActorID:      user.ID,
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   refs.StringValue(user.Email),
			ResourceType: constants.AuditResourceTypeToken,
			ResourceID:   user.ID,
			Metadata:     grantType,
			IPAddress:    utils.GetIP(gc.Request),
			UserAgent:    utils.GetUserAgent(gc.Request),
		})
		gc.JSON(http.StatusOK, res)
	}
}
