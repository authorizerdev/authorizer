package http_handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/token"
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
		var reqBody RequestBody
		if err := gc.Bind(&reqBody); err != nil {
			log.Debug().Err(err).Msg("failed to bind json")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "error_binding_json",
				"error_description": err.Error(),
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
				"error":             "invalid_grant_type",
				"error_description": "grant_type is invalid",
			})
		}

		// check if clientID & clientSecret are present as part of
		// authorization header with basic auth
		if clientID == "" && clientSecret == "" {
			clientID, clientSecret, _ = gc.Request.BasicAuth()
		}

		if clientID == "" {
			log.Debug().Msg("Client ID is missing")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "client_id_required",
				"error_description": "The client id is missing",
			})
			return
		}

		if h.Config.ClientID != clientID {
			log.Debug().Str("client_id", clientID).Msg("Client ID is invalid")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_client_id",
				"error_description": "The client id is invalid",
			})
			return
		}

		var userID string
		var roles, scope []string
		loginMethod := ""
		sessionKey := ""

		if isAuthorizationCodeGrant {
			if code == "" {
				log.Debug().Msg("Code is missing")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_code",
					"error_description": "The code is required",
				})
				return
			}

			if codeVerifier == "" && clientSecret == "" {
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_data",
					"error_description": "The code verifier or client secret is required",
				})
				return
			}
			// Get state
			sessionData, err := h.MemoryStoreProvider.GetState(code)
			if sessionData == "" || err != nil {
				log.Debug().Err(err).Msg("Error getting session data")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_code",
					"error_description": "The code is invalid",
				})
				return
			}

			// [0] -> code_challenge
			// [1] -> session cookie
			sessionDataSplit := strings.Split(sessionData, "@@")

			go h.MemoryStoreProvider.RemoveState(code)

			if codeVerifier != "" {
				hash := sha256.New()
				hash.Write([]byte(codeVerifier))
				encryptedCode := strings.ReplaceAll(base64.RawURLEncoding.EncodeToString(hash.Sum(nil)), "+", "-")
				encryptedCode = strings.ReplaceAll(encryptedCode, "/", "_")
				encryptedCode = strings.ReplaceAll(encryptedCode, "=", "")
				if encryptedCode != sessionDataSplit[0] {
					gc.JSON(http.StatusBadRequest, gin.H{
						"error":             "invalid_code_verifier",
						"error_description": "The code verifier is invalid",
					})
					return
				}

			} else {
				if clientSecret != h.Config.ClientSecret {
					log.Debug().Err(err).Msg("Error getting client secret")
					gc.JSON(http.StatusBadRequest, gin.H{
						"error":             "invalid_client_secret",
						"error_description": "The client secret is invalid",
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

			// rollover the session for security
			sessionKey = userID
			if loginMethod != "" {
				sessionKey = loginMethod + ":" + userID
			}

			go h.MemoryStoreProvider.DeleteUserSession(sessionKey, claims.Nonce)

		} else {
			// validate refresh token
			if refreshToken == "" {
				log.Debug().Msg("Refresh token is missing")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_refresh_token",
					"error_description": "The refresh token is invalid",
				})
				return
			}

			claims, err := h.TokenProvider.ValidateRefreshToken(gc, refreshToken)
			if err != nil {
				log.Debug().Err(err).Msg("Error validating refresh token")
				gc.JSON(http.StatusUnauthorized, gin.H{
					"error":             "unauthorized",
					"error_description": err.Error(),
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
			go h.MemoryStoreProvider.DeleteUserSession(sessionKey, nonce)
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
		nonce := uuid.New().String() + "@@" + code
		authToken, err := h.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
			User:        user,
			Roles:       roles,
			Scope:       scope,
			LoginMethod: loginMethod,
			Nonce:       nonce,
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

		h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
		h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)
		cookie.SetSession(gc, authToken.FingerPrintHash, h.Config.AppCookieSecure)

		expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
		if expiresIn <= 0 {
			expiresIn = 1
		}

		res := map[string]interface{}{
			"access_token": authToken.AccessToken.Token,
			"id_token":     authToken.IDToken.Token,
			"scope":        strings.Join(scope, " "),
			"roles":        roles,
			"expires_in":   expiresIn,
		}
		if authToken.RefreshToken != nil {
			res["refresh_token"] = authToken.RefreshToken.Token
			h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
		}
		gc.JSON(http.StatusOK, res)
	}
}
