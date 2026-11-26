package http_handlers

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/asyncutil"
	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/codestate"
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
		// checkClientRedirectURI rather than IsValidRedirectURI directly: the
		// latter compares ORIGINS, so every path under an allowed host passes,
		// and this endpoint hands the target an authorization code. The
		// exact-match validator already existed for /authorize and /app (OIDC
		// Core §3.1.2.1) and simply was never called here
		// (GHSA-44vr-f829-xfch).
		//
		// A verification request records no client_id, so the deployment's own
		// client is the only identity available. Exact matching therefore
		// applies when the operator has declared --redirect-uris, and falls
		// back to the origin allow-list otherwise. That is why dropping the
		// tokens from the redirect (further down) is the primary fix and this
		// is hardening on top of it, not the other way round.
		//
		// Returns false and has ALREADY written the response when the URI is
		// unacceptable. A lookup FAILURE and a non-matching URI are answered
		// differently on purpose: checkClientRedirectURI reads the client
		// registry, a read this handler never used to do, and collapsing a
		// storage outage into "invalid redirect uri" would tell the user their
		// link is bad during what is really a server fault. Mirrors /authorize.
		validRedirect := func(uri string) bool {
			check, cErr := h.checkClientRedirectURI(c.Request.Context(), h.Config.ClientID, uri, hostname)
			switch {
			case cErr != nil:
				log.Warn().Err(cErr).Msg("client lookup failed; refusing rather than falling back to the origin allow-list")
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "could not verify the client; please retry"})
				return false
			case !check.Valid:
				log.Debug().Msg("Invalid redirect URI")
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid redirect uri"})
				return false
			}
			return true
		}
		redirectURL := strings.TrimSpace(c.Query("redirect_uri"))
		if redirectURL != "" && !validRedirect(redirectURL) {
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
			errorRes["error"] = publicErrorMessage(&log, err)
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
		if !validRedirect(redirectURL) {
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
			errorRes["error"] = publicErrorMessage(&log, gateErr)
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
			// Only roles still configured count (#425).
			userRoles := h.Config.EffectiveUserRoles(user.Roles)
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

		// /authorize detour: when the magic link (or signup verification) was
		// started from an /authorize request, the login UI threads its `state`
		// through, and /authorize stored the pre-generated code plus its PKCE
		// challenge under it. Bind them to the session minted here and persist
		// the code state exactly as login/session/signup do, so /oauth/token
		// can redeem the code (PKCE-verified, client- and redirect-bound,
		// single-use via GetAndRemoveState). The code state used to be left
		// unwritten, so the `code` in the redirect was never redeemable.
		//
		// Reached only after the MFA gate above has let the session through:
		// a withheld session returns earlier and leaves the authorize state
		// untouched, so no code is ever bound to a session that does not exist.
		code := ""
		codeChallenge := ""
		nonce := ""
		oidcNonce := ""
		authorizeRedirectURI := ""
		authorizeResource := ""
		authorizeClientID := ""
		if state != "" {
			// Atomic read-and-delete: the authorize state is single-use, so two
			// clicks racing on the same link cannot both bind it.
			authorizeState, _ := h.MemoryStoreProvider.GetAndRemoveState(state)
			if authorizeState != "" {
				// One owner for this positional format — see internal/codestate.
				if codestate.HasCode(authorizeState) {
					as := codestate.DecodeAuthorize(authorizeState)
					// RFC 6749 §4.1.2: the code goes to the redirect_uri of the
					// authorization request and nowhere else. The target here is
					// validated, but against an allow-list that can be wider than
					// the one URI /authorize recorded, so a mismatch gets the
					// session (cookie) without a code rather than a code
					// delivered somewhere the client never asked for.
					if as.RedirectURI == "" || as.RedirectURI == redirectURL {
						code = as.Code
						codeChallenge = as.Challenge
						oidcNonce = as.Nonce
						authorizeRedirectURI = as.RedirectURI
						authorizeResource = as.Resource
						authorizeClientID = as.ClientID
					} else {
						log.Debug().Msg("redirect_uri differs from the authorize request; not issuing a code")
					}
				} else {
					nonce = authorizeState
				}
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
			OIDCNonce:   oidcNonce,
			Code:        code,
			HostName:    hostname,
		})
		if err != nil {
			log.Debug().Err(err).Msg("Error creating auth token")
			errorRes["error"] = publicErrorMessage(&log, err)
			utils.HandleRedirectORJsonResponse(c, http.StatusInternalServerError, errorRes, generateRedirectURL(redirectURL, errorRes))
			return
		}

		if code != "" {
			if err := h.MemoryStoreProvider.SetState(code, codestate.EncodeCode(codestate.Code{
				Challenge:   codeChallenge,
				Session:     authToken.FingerPrintHash,
				Nonce:       oidcNonce,
				RedirectURI: authorizeRedirectURI,
				Resource:    authorizeResource,
				ClientID:    authorizeClientID,
			})); err != nil {
				log.Debug().Err(err).Msg("Error setting code state")
				errorRes["error"] = publicErrorMessage(&log, err)
				utils.HandleRedirectORJsonResponse(c, http.StatusInternalServerError, errorRes, generateRedirectURL(redirectURL, errorRes))
				return
			}
		}

		// Tokens are NOT placed in the redirect URL. This endpoint is reached
		// from an emailed link, so its URL is unusually long-lived: it sits in
		// mail-client history, in the browser history, in every proxy and web
		// server log between the user and the app, and in the Referer of every
		// subresource the target page loads. Appending access_token, id_token
		// and refresh_token to the QUERY string of a target validated only at
		// origin granularity meant any path under an allowed origin received a
		// full session (GHSA-44vr-f829-xfch).
		//
		// The browser session cookie set below, plus the authorization `code`,
		// are the delivery channel — matching oauth_callback.go and
		// oauth_sso.go, which were migrated to code-only delivery already. This
		// handler was the last one still emitting raw tokens.
		params := "state=" + url.QueryEscape(state) + "&nonce=" + url.QueryEscape(nonce)

		if code != "" {
			params += "&code=" + url.QueryEscape(code)
		}

		sessionKey := loginMethod + ":" + user.ID
		cookie.SetSession(c, authToken.FingerPrintHash, h.Config.AppCookieSecure, cookie.ParseSameSite(h.Config.AppCookieSameSite))
		_ = h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, crypto.HashSessionValue(authToken.FingerPrintHash), authToken.SessionTokenExpiresAt)
		_ = h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, crypto.HashSessionValue(authToken.AccessToken.Token), authToken.AccessToken.ExpiresAt)

		if authToken.RefreshToken != nil {
			// Registered in the session store so the refresh token remains
			// redeemable, but NOT appended to the redirect — see above. A
			// refresh token in an emailed URL is the longest-lived credential
			// in the worst possible place.
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
