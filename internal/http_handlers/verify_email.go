package http_handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/asyncutil"
	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// VerifyEmailHandler handles the verify email route.
// It verifies email based on JWT token in query string
func (h *httpProvider) VerifyEmailHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "VerifyEmailHandler").Logger()
	return func(c *gin.Context) {
		hostname := parsers.GetHost(c)
		redirectURL := strings.TrimSpace(c.Query("redirect_uri"))
		if redirectURL != "" && !validators.IsValidRedirectURI(redirectURL, h.Config.AllowedOrigins, hostname) {
			log.Debug().Msg("Invalid redirect URI")
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid redirect uri"})
			return
		}
		errorRes := gin.H{
			"error": "token is required",
		}
		tokenInQuery := c.Query("token")
		if tokenInQuery == "" {
			log.Debug().Msg("Token is missing")
			utils.HandleRedirectORJsonResponse(c, http.StatusBadRequest, errorRes, generateRedirectURL(redirectURL, errorRes))
			return
		}

		// Decision logic is shared with the GraphQL/gRPC mutation via
		// service.ConsumeEmailVerificationToken — token validation, purpose
		// binding, user lookup, the revoked check, and the email_verified_at
		// write, in that order. This handler previously reimplemented all of it
		// and drifted twice; see that function's comment. Only presentation
		// stays here (redirect vs AuthResponse).
		verified, err := h.ServiceProvider.ConsumeEmailVerificationToken(c, hostname, tokenInQuery)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to consume verification token")
			errorRes["error"] = err.Error()
			utils.HandleRedirectORJsonResponse(c, http.StatusBadRequest, errorRes, generateRedirectURL(redirectURL, errorRes))
			return
		}
		user := verified.User
		verificationRequest := verified.Request
		isSignUp := verified.IsSignUp
		loginMethod := verified.LoginMethod

		// Resolved once, early: needed both for the MFA-gate-withheld redirect
		// below and the success redirect further down.
		if redirectURL == "" {
			redirectURL = verified.RedirectURI
		}
		if !validators.IsValidRedirectURI(redirectURL, h.Config.AllowedOrigins, hostname) {
			log.Debug().Msg("Invalid redirect URI in token claim")
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid redirect uri"})
			return
		}

		// MFA gate: this REST endpoint is what the emailed verification/magic
		// link literally points to, so it must enforce the same gate every
		// other login entry point does (login.go/signup.go/oauth_callback.go).
		// Previously this handler issued tokens unconditionally, bypassing MFA
		// entirely for magic-link login and signup email verification - the
		// GraphQL verify_email mutation's own gate fix (service.VerifyEmail)
		// never covered this REST path since it's a separate implementation.
		meta := service.MetaFromGin(c)
		side := &service.ResponseSideEffects{}
		withheld, redirectSuffix, gateErr := h.ServiceProvider.EvaluateMFAGateForOAuth(c, meta, side, user)
		if gateErr != nil {
			log.Debug().Err(gateErr).Msg("MFA gate rejected email verification")
			errorRes["error"] = gateErr.Error()
			utils.HandleRedirectORJsonResponse(c, http.StatusBadRequest, errorRes, generateRedirectURL(redirectURL, errorRes))
			return
		}
		if withheld {
			service.ApplyToGin(c, side)
			target := redirectURL
			if strings.Contains(target, "?") {
				target = target + "&" + redirectSuffix
			} else {
				target = target + "?" + strings.TrimPrefix(redirectSuffix, "&")
			}
			c.Redirect(http.StatusTemporaryRedirect, target)
			return
		}

		// delete from verification table
		if err := h.StorageProvider.DeleteVerificationRequest(c, verificationRequest); err != nil {
			log.Debug().Err(err).Msg("Error deleting verification request")
		}

		state := strings.TrimSpace(c.Query("state"))
		rolesString := strings.TrimSpace(c.Query("roles"))
		var roles []string
		if rolesString == "" {
			roles = strings.Split(user.Roles, ",")
		} else {
			roles = strings.Split(rolesString, ",")
			userRoles := strings.Split(user.Roles, ",")
			if !validators.IsValidRoles(roles, userRoles) {
				log.Debug().Msg("Invalid roles requested")
				errorRes["error"] = "invalid roles"
				utils.HandleRedirectORJsonResponse(c, http.StatusBadRequest, errorRes, generateRedirectURL(redirectURL, errorRes))
				return
			}
		}

		scopeString := strings.TrimSpace(c.Query("scope"))
		var scope []string
		if scopeString == "" {
			scope = []string{"openid", "email", "profile"}
		} else {
			scope = strings.Split(scopeString, " ")
		}

		code := ""
		// Not required as /oauth/token cannot be resumed from other tab
		// codeChallenge := ""
		nonce := ""
		if state != "" {
			// Get state from store
			authorizeState, _ := h.MemoryStoreProvider.GetState(state)
			if authorizeState != "" {
				authorizeStateSplit := strings.Split(authorizeState, "@@")
				if len(authorizeStateSplit) > 1 {
					code = authorizeStateSplit[0]
					// Not required as /oauth/token cannot be resumed from other tab
					// codeChallenge = authorizeStateSplit[1]
				} else {
					nonce = authorizeState
				}
				asyncutil.Go(h.Log, func() { _ = h.MemoryStoreProvider.RemoveState(state) })
			}
		}
		if nonce == "" {
			nonce = uuid.New().String()
		}
		authToken, err := h.TokenProvider.CreateAuthToken(c, &token.AuthTokenConfig{
			User:        user,
			Roles:       roles,
			Scope:       scope,
			LoginMethod: loginMethod,
			Nonce:       nonce,
			Code:        code,
			HostName:    hostname,
		})
		if err != nil {
			log.Debug().Err(err).Msg("Error creating auth token")
			errorRes["error"] = err.Error()
			utils.HandleRedirectORJsonResponse(c, http.StatusInternalServerError, errorRes, generateRedirectURL(redirectURL, errorRes))
			return
		}

		// Code challenge could be optional if PKCE flow is not used
		// Not required as /oauth/token cannot be resumed from other tab
		// if code != "" {
		// 	if err := memorystore.Provider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash); err != nil {
		// 		log.Debug("Error setting code state ", err)
		// 		errorRes["error"] = err.Error()
		// 		c.JSON(500, errorRes)
		// 		return
		// 	}
		// }

		expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
		if expiresIn <= 0 {
			expiresIn = 1
		}

		params := "access_token=" + authToken.AccessToken.Token + "&token_type=bearer&expires_in=" + strconv.FormatInt(expiresIn, 10) + "&state=" + state + "&id_token=" + authToken.IDToken.Token + "&nonce=" + nonce

		if code != "" {
			params += "&code=" + code
		}

		sessionKey := loginMethod + ":" + user.ID
		cookie.SetSession(c, authToken.FingerPrintHash, h.Config.AppCookieSecure, cookie.ParseSameSite(h.Config.AppCookieSameSite))
		_ = h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, crypto.HashSessionValue(authToken.FingerPrintHash), authToken.SessionTokenExpiresAt)
		_ = h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, crypto.HashSessionValue(authToken.AccessToken.Token), authToken.AccessToken.ExpiresAt)

		if authToken.RefreshToken != nil {
			params = params + `&refresh_token=` + authToken.RefreshToken.Token
			_ = h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, crypto.HashSessionValue(authToken.RefreshToken.Token), authToken.RefreshToken.ExpiresAt)
		}

		if strings.Contains(redirectURL, "?") {
			redirectURL = redirectURL + "&" + params
		} else {
			redirectURL = redirectURL + "?" + strings.TrimPrefix(params, "&")
		}

		metrics.RecordAuthEvent(metrics.EventVerifyEmail, metrics.StatusSuccess)
		metrics.ActiveSessions.Inc()
		h.AuditProvider.LogEvent(audit.Event{
			Action:       constants.AuditEmailVerifiedEvent,
			ActorID:      user.ID,
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   refs.StringValue(user.Email),
			ResourceType: constants.AuditResourceTypeUser,
			ResourceID:   user.ID,
			IPAddress:    utils.GetIP(c.Request),
			UserAgent:    utils.GetUserAgent(c.Request),
		})
		bgCtx := context.WithoutCancel(c)
		userAgent := utils.GetUserAgent(c.Request)
		ip := utils.GetIP(c.Request)
		asyncutil.Go(h.Log, func() {
			if isSignUp {
				_ = h.EventsProvider.RegisterEvent(bgCtx, constants.UserSignUpWebhookEvent, loginMethod, user)
				// User is also logged in with signup
				_ = h.EventsProvider.RegisterEvent(bgCtx, constants.UserLoginWebhookEvent, loginMethod, user)
			} else {
				_ = h.EventsProvider.RegisterEvent(bgCtx, constants.UserLoginWebhookEvent, loginMethod, user)
			}
			if err := h.StorageProvider.AddSession(bgCtx, &schemas.Session{
				UserID:    user.ID,
				UserAgent: userAgent,
				IP:        ip,
			}); err != nil {
				log.Debug().Err(err).Msg("Error adding session")
			}
		})

		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
	}
}

func generateRedirectURL(url string, res map[string]interface{}) string {
	redirectURL := url
	if redirectURL == "" {
		return ""
	}
	var paramsArr []string
	for key, value := range res {
		paramsArr = append(paramsArr, key+"="+value.(string))
	}
	params := strings.Join(paramsArr, "&")
	if strings.Contains(redirectURL, "?") {
		redirectURL = redirectURL + "&" + params
	} else {
		redirectURL = redirectURL + "?" + strings.TrimPrefix(params, "&")
	}
	return redirectURL
}
