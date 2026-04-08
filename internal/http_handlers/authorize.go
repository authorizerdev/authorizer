package http_handlers

/**
LOGIC TO REMEMBER THE AUTHORIZE FLOW


jargons
`at_hash` -> access_token_hash
`c_hash` -> code_hash


# ResponseType: Code
	with /authorize request
		- set state [state, code@@challenge]
		- add &code to login redirect url
	login resolver has optional param state
		-if state found in store, split with @@
		- if len > 1 -> response type is code and has code + challenge
		- set `nonce, code` for createAuthToken request so that `c_hash` can be generated
		- do not add `nonce` to id_token in code flow, instead set `c_hash` and `at_hash`


# ResponseType: token / id_token
	with /authorize request
		- set state [state, nonce]
		- add &nonce to login redirect url
	login resolver has optional param state
		- if state found in store, split with @@
		- if len < 1 -> response type is token / id_token and value is nonce
		- send received nonce for createAuthToken with empty code value
		- set `nonce` and `at_hash` in `id_token`
**/

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// Check the flow for generating and verifying codes: https://developer.okta.com/blog/2019/08/22/okta-authjs-pkce#:~:text=PKCE%20works%20by%20having%20the,is%20called%20the%20Code%20Challenge.

// Check following docs for understanding request / response params for various types of requests: https://auth0.com/docs/authenticate/login/oidc-conformant-authentication/oidc-adoption-auth-code-flow

const (
	authorizeWebMessageTemplate = "authorize_web_message.tmpl"
	authorizeFormPostTemplate   = "authorize_form_post.tmpl"
	baseAppPath                 = "/app"
	signupPath                  = "/app/signup"
)

// AuthorizeHandler is the handler for the /authorize route
// required params
// ?redirect_uri = redirect url
// ?response_mode = to decide if result should be html or re-direct
// state[recommended] = to prevent CSRF attack (for authorizer its compulsory)
// code_challenge = to prevent CSRF attack
// code_challenge_method = to prevent CSRF attack [only sh256 is supported]
func (h *httpProvider) AuthorizeHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "AuthorizeHandler").Logger()
	return func(gc *gin.Context) {
		redirectURI := strings.TrimSpace(gc.Query("redirect_uri"))
		responseType := strings.TrimSpace(gc.Query("response_type"))
		state := strings.TrimSpace(gc.Query("state"))
		codeChallenge := strings.TrimSpace(gc.Query("code_challenge"))
		scopeString := strings.TrimSpace(gc.Query("scope"))
		clientID := strings.TrimSpace(gc.Query("client_id"))
		responseMode := strings.TrimSpace(gc.Query("response_mode"))
		rawResponseMode := responseMode
		nonce := strings.TrimSpace(gc.Query("nonce"))
		screenHint := strings.TrimSpace(gc.Query("screen_hint"))

		// OIDC Core §3.1.2.1 standard authorization request parameters.
		loginHint := strings.TrimSpace(gc.Query("login_hint"))
		uiLocales := strings.TrimSpace(gc.Query("ui_locales"))
		prompt := strings.TrimSpace(gc.Query("prompt"))
		maxAgeStr := strings.TrimSpace(gc.Query("max_age"))
		idTokenHint := strings.TrimSpace(gc.Query("id_token_hint"))

		// max_age is advisory. Parse per OIDC Core §3.1.2.1:
		//   - negative or non-integer → treat as absent (no constraint)
		//   - max_age=0 → force re-auth (equivalent to prompt=login)
		//   - positive → compare against session age (handled below)
		maxAge := -1 // sentinel: "not supplied"
		maxAgeZero := false
		if maxAgeStr != "" {
			if parsed, err := strconv.Atoi(maxAgeStr); err == nil && parsed >= 0 {
				maxAge = parsed
				if parsed == 0 {
					maxAgeZero = true
				}
			}
		}

		// id_token_hint is advisory per OIDC Core §3.1.2.1. Validate
		// structurally; on failure log at debug and continue.
		hintedSub := h.parseExpiredOrValidIDTokenHintSubject(idTokenHint)
		if idTokenHint != "" && hintedSub == "" {
			log.Debug().Msg("id_token_hint provided but invalid — ignoring per OIDC Core §3.1.2.1")
		}

		// prompt=consent / prompt=select_account are accepted but
		// not yet implemented — proceed normally.
		if prompt == "consent" || prompt == "select_account" {
			log.Debug().Str("prompt", prompt).Msg("prompt value accepted but not implemented — proceeding normally")
		}

		var scope []string
		if scopeString == "" {
			scope = []string{"openid", "profile", "email"}
		} else {
			scope = strings.Split(scopeString, " ")
		}

		if responseMode == "" {
			responseMode = h.Config.DefaultAuthorizeResponseMode
		}

		if redirectURI == "" {
			redirectURI = "/app"
		} else {
			hostname := parsers.GetHost(gc)
			if !validators.IsValidRedirectURI(redirectURI, h.Config.AllowedOrigins, hostname) {
				log.Debug().Msg("Invalid redirect URI")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_request",
					"error_description": "invalid redirect_uri",
				})
				return
			}
		}

		if responseType == "" {
			responseType = h.Config.DefaultAuthorizeResponseType
		}

		codeChallengeMethod := strings.TrimSpace(gc.Query("code_challenge_method"))
		// RFC 7636 §4.3: Default to S256 if code_challenge is present but method is not specified
		// Note: We only support S256 as it is mandatory to implement per RFC 7636
		if codeChallengeMethod == "" && codeChallenge != "" {
			codeChallengeMethod = "S256"
		}
		if codeChallengeMethod != "" && codeChallengeMethod != "S256" {
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "Only S256 code_challenge_method is supported",
			})
			return
		}

		canonical, ok := supportedResponseTypeSet(responseType)
		if !ok {
			log.Debug().Str("response_type", responseType).Msg("unsupported response_type")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "unsupported_response_type",
				"error_description": "response_type is not supported",
			})
			return
		}
		responseType = canonical

		// OIDC Core §3.3 hybrid response_type combinations.
		isHybrid := responseType == "code id_token" ||
			responseType == "code token" ||
			responseType == "code id_token token" ||
			responseType == "id_token token"
		if isHybrid {
			// OIDC Core §3.3.2.5: hybrid flow MUST NOT use query response_mode.
			if rawResponseMode == constants.ResponseModeQuery {
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_request",
					"error_description": "response_mode=query is not allowed for hybrid response_type",
				})
				return
			}
			// Default to fragment when the client did not explicitly
			// specify one (the global default may be query).
			if rawResponseMode == "" {
				responseMode = constants.ResponseModeFragment
			}
		}

		if errCode, errDesc := h.validateAuthorizeRequest(responseType, responseMode, clientID, state, codeChallenge); errCode != "" {
			log.Debug().Str("error", errCode).Str("error_description", errDesc).Msg("Invalid request")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             errCode,
				"error_description": errDesc,
			})
			return
		}

		code := uuid.New().String()
		// Track whether the client supplied a nonce. Per OIDC Core
		// §3.1.2.6, the nonce is only echoed back when the client
		// originally supplied one. We still need a value internally
		// for state/session bookkeeping in implicit/hybrid flows, so
		// auto-generate when absent — but never expose that synthetic
		// nonce to the relying party (it would break RP-side nonce
		// validation).
		nonceFromClient := nonce != ""
		if nonce == "" {
			nonce = uuid.New().String()
		}

		log = log.With().Str("response_type", responseType).Str("response_mode", responseMode).Str("state", state).Str("code_challenge", codeChallenge).Str("scope", scopeString).Str("client_id", clientID).Str("nonce", nonce).Logger()

		// Build the auth-state query string used for the login UI
		// redirect. All values pass through url.Values.Encode() so any
		// user-controlled input (state, scope, redirect_uri, code,
		// nonce, login_hint, ui_locales) is properly percent-escaped
		// and cannot inject extra parameters.
		authStateValues := url.Values{}
		authStateValues.Set("state", state)
		authStateValues.Set("scope", scopeString)
		authStateValues.Set("redirect_uri", redirectURI)
		// OIDC Core §3.1.2.1: login_hint and ui_locales are forwarded
		// to the login UI so it can pre-fill the email field and pick
		// the UI language.
		if loginHint != "" {
			authStateValues.Set("login_hint", loginHint)
		}
		if uiLocales != "" {
			authStateValues.Set("ui_locales", uiLocales)
		}
		if responseType == constants.ResponseTypeCode {
			authStateValues.Set("code", code)
			if err := h.MemoryStoreProvider.SetState(state, code+"@@"+codeChallenge); err != nil {
				log.Debug().Err(err).Msg("Error setting temp code")
				gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
		} else {
			authStateValues.Set("nonce", nonce)
			if err := h.MemoryStoreProvider.SetState(state, nonce); err != nil {
				log.Debug().Err(err).Msg("Error setting temp nonce")
				gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
		}
		authState := authStateValues.Encode()

		authURL := baseAppPath + "?" + authState

		if screenHint == constants.ScreenHintSignUp {
			authURL = signupPath + "?" + authState
		}

		if responseMode == constants.ResponseModeFragment && screenHint == constants.ScreenHintSignUp {
			authURL = signupPath + "#" + authState
		} else if responseMode == constants.ResponseModeFragment {
			authURL = baseAppPath + "#" + authState
		}

		if responseType == constants.ResponseTypeCode && codeChallenge == "" {
			handleResponse(gc, responseMode, authURL, redirectURI, map[string]interface{}{
				"type": "authorization_response",
				"response": map[string]interface{}{
					"error":             "code_challenge_required",
					"error_description": "code challenge is required",
				},
			}, http.StatusOK)
			return
		}

		loginError := map[string]interface{}{
			"type": "authorization_response",
			"response": map[string]interface{}{
				"error":             "login_required",
				"error_description": "Login is required",
			},
		}
		// OIDC Core §3.1.2.1: prompt=login forces re-authentication even
		// if a valid session exists. max_age similarly forces re-auth if
		// the current session is older than the allowed window. We only
		// apply forceReauth when prompt != "none" — prompt=none wants to
		// check the existing session, not bypass it.
		// max_age=0 is equivalent to prompt=login (force re-auth) per
		// OIDC Core §3.1.2.1.
		forceReauth := prompt == "login" || maxAgeZero

		sessionToken, err := cookie.GetSession(gc)
		if err == nil && !forceReauth && maxAge > 0 && prompt != "none" {
			// Check session age against max_age.
			if decryptedFingerPrint, decErr := crypto.DecryptAES(h.ClientSecret, sessionToken); decErr == nil {
				var sd token.SessionData
				if jsonErr := json.Unmarshal([]byte(decryptedFingerPrint), &sd); jsonErr == nil {
					if time.Now().Unix()-sd.IssuedAt > int64(maxAge) {
						log.Debug().Int("max_age", maxAge).Int64("session_age", time.Now().Unix()-sd.IssuedAt).Msg("session exceeds max_age — forcing re-auth")
						forceReauth = true
					}
				}
			}
		}

		if forceReauth {
			err = errors.New("force reauth")
			sessionToken = ""
		}

		// promptNoneLoginRequired dispatches the OIDC Core §3.1.2.1
		// login_required error to the client's redirect_uri via the
		// configured response_mode. Used whenever prompt=none cannot
		// complete silently (missing session, expired session, etc).
		promptNoneLoginRequired := func(reason string) {
			log.Debug().Str("reason", reason).Msg("prompt=none cannot complete silently — returning login_required")
			errParams := "error=login_required" +
				"&error_description=" + url.QueryEscape("prompt=none was requested but the user is not authenticated") +
				"&state=" + url.QueryEscape(state)
			errRedirectURI := redirectURI
			switch responseMode {
			case constants.ResponseModeFragment:
				if strings.Contains(errRedirectURI, "#") {
					errRedirectURI = errRedirectURI + "&" + errParams
				} else {
					errRedirectURI = errRedirectURI + "#" + errParams
				}
			case constants.ResponseModeQuery:
				if strings.Contains(errRedirectURI, "?") {
					errRedirectURI = errRedirectURI + "&" + errParams
				} else {
					errRedirectURI = errRedirectURI + "?" + errParams
				}
			}
			errData := map[string]interface{}{
				"type": "authorization_response",
				"response": map[string]interface{}{
					"error":             "login_required",
					"error_description": "prompt=none was requested but the user is not authenticated",
					"state":             state,
				},
			}
			switch responseMode {
			case constants.ResponseModeWebMessage:
				gc.HTML(http.StatusOK, authorizeWebMessageTemplate, gin.H{
					"target_origin":          redirectURI,
					"authorization_response": errData,
				})
			case constants.ResponseModeFormPost:
				gc.HTML(http.StatusOK, authorizeFormPostTemplate, gin.H{
					"target_origin":          redirectURI,
					"authorization_response": errData["response"],
				})
			default:
				gc.Redirect(http.StatusFound, errRedirectURI)
			}
		}

		if prompt == "none" && (err != nil || sessionToken == "") {
			promptNoneLoginRequired("no session cookie")
			return
		}

		if err != nil {
			log.Debug().Err(err).Msg("Error getting session token")
			handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
			return
		}

		// get session from cookie
		claims, err := h.TokenProvider.ValidateBrowserSession(gc, sessionToken)
		if err != nil {
			log.Debug().Err(err).Msg("Error validating session token")
			// OIDC Core §3.1.2.1: prompt=none with a stale/revoked
			// session must still return login_required to the client,
			// not redirect the user-agent to the login UI.
			if prompt == "none" {
				promptNoneLoginRequired("session validation failed")
				return
			}
			handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
			return
		}

		userID := claims.Subject
		user, err := h.StorageProvider.GetUserByID(gc, userID)
		if err != nil {
			log.Debug().Err(err).Msg("Error getting user")
			handleResponse(gc, responseMode, authURL, redirectURI, map[string]interface{}{
				"type": "authorization_response",
				"response": map[string]interface{}{
					"error":             "signup_required",
					"error_description": "Sign up required",
				},
			}, http.StatusOK)
			return
		}

		sessionKey := user.ID
		if claims.LoginMethod != "" {
			sessionKey = claims.LoginMethod + ":" + user.ID
		}

		// rollover the session for security
		go h.MemoryStoreProvider.DeleteUserSession(sessionKey, claims.Nonce)

		// idTokenNonce is the nonce value to embed in the issued ID
		// token. Per OIDC Core §3.1.2.6 / §3.2.2.11 the `nonce` claim
		// MUST only be present when the RP supplied a nonce in the
		// authorization request — never a server-synthesized value.
		idTokenNonce := ""
		if nonceFromClient {
			idTokenNonce = nonce
		}

		if isHybrid {
			hostname := parsers.GetHost(gc)
			// For hybrid flows we mint tokens AND a code. Setting Code
			// on the AuthTokenConfig causes CreateAuthToken to populate
			// cfg.CodeHash, which in turn causes CreateIDToken to emit
			// the c_hash claim per OIDC Core §3.3.2.11.
			authToken, err := h.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
				User:        user,
				Nonce:       idTokenNonce,
				Code:        code,
				Roles:       claims.Roles,
				Scope:       scope,
				LoginMethod: claims.LoginMethod,
				HostName:    hostname,
			})
			if err != nil {
				log.Debug().Err(err).Msg("Error creating auth token for hybrid response")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			// Stash the code so /oauth/token can later exchange it.
			if err := h.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+authToken.FingerPrint); err != nil {
				log.Debug().Err(err).Msg("Error setting temp code for hybrid")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}
			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error persisting session for hybrid")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}
			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error persisting access token for hybrid")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}
			cookie.SetSession(gc, authToken.FingerPrintHash, h.Config.AppCookieSecure)
			expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
			if expiresIn <= 0 {
				// A token issued already-expired indicates a clock or
				// configuration bug. Surface it as a server_error
				// instead of papering over it with a 1-second TTL,
				// which would mask the underlying issue.
				log.Error().Int64("expires_at", authToken.AccessToken.ExpiresAt).Int64("now", time.Now().Unix()).Msg("hybrid: access token issued already-expired")
				gc.JSON(http.StatusInternalServerError, gin.H{
					"error":             "server_error",
					"error_description": "access token issued already-expired",
				})
				return
			}

			hasAccessToken := responseType == "code token" ||
				responseType == "code id_token token" ||
				responseType == "id_token token"
			hasIDToken := responseType == "code id_token" ||
				responseType == "code id_token token" ||
				responseType == "id_token token"

			// Build the response map. Always include code + state for
			// hybrid combos containing "code"; include id_token /
			// access_token based on the requested set.
			res := map[string]interface{}{
				"state":      state,
				"token_type": "Bearer",
				"scope":      strings.Join(scope, " "),
				"expires_in": expiresIn,
			}
			hasCode := strings.Contains(responseType, "code")
			if hasCode {
				res["code"] = code
			}
			if hasAccessToken {
				res["access_token"] = authToken.AccessToken.Token
			}
			if hasIDToken {
				res["id_token"] = authToken.IDToken.Token
			}
			// OIDC Core §3.1.2.6: nonce is only echoed when the client
			// supplied one in the authorization request.
			if nonceFromClient {
				res["nonce"] = nonce
			}

			// Build the fragment params string for redirect modes via
			// url.Values so user-controlled inputs (state, code,
			// access_token, id_token, nonce) cannot inject extra
			// fragment parameters.
			fragmentValues := url.Values{}
			fragmentValues.Set("state", state)
			fragmentValues.Set("token_type", "Bearer")
			fragmentValues.Set("expires_in", strconv.FormatInt(expiresIn, 10))
			if hasCode {
				fragmentValues.Set("code", code)
			}
			if hasAccessToken {
				fragmentValues.Set("access_token", authToken.AccessToken.Token)
			}
			if hasIDToken {
				fragmentValues.Set("id_token", authToken.IDToken.Token)
			}
			if nonceFromClient {
				fragmentValues.Set("nonce", nonce)
			}
			params := fragmentValues.Encode()

			// Hybrid is fragment-default; the pre-check above ensured
			// responseMode != "query".
			if responseMode == constants.ResponseModeFragment {
				if strings.Contains(redirectURI, "#") {
					redirectURI = redirectURI + "&" + params
				} else {
					redirectURI = redirectURI + "#" + params
				}
			}

			handleResponse(gc, responseMode, authURL, redirectURI, map[string]interface{}{
				"type":     "authorization_response",
				"response": res,
			}, http.StatusOK)
			return
		}

		if responseType == constants.ResponseTypeCode {
			newSessionTokenData, newSessionToken, newSessionExpiresAt, err := h.TokenProvider.CreateSessionToken(&token.AuthTokenConfig{
				User:        user,
				Nonce:       nonce,
				Roles:       claims.Roles,
				Scope:       scope,
				LoginMethod: claims.LoginMethod,
			})
			if err != nil {
				log.Debug().Err(err).Msg("Error creating session token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := h.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+newSessionToken); err != nil {
				log.Debug().Err(err).Msg("Error setting temp code")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+newSessionTokenData.Nonce, newSessionToken, newSessionExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error setting session token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			cookie.SetSession(gc, newSessionToken, h.Config.AppCookieSecure)

			// RFC 6749 §4.1.2: Authorization code response MUST only
			// include code and state. Both pass through url.Values so
			// attacker-controlled state cannot inject extra params.
			codeFlowValues := url.Values{}
			codeFlowValues.Set("code", code)
			codeFlowValues.Set("state", state)
			params := codeFlowValues.Encode()
			if responseMode == constants.ResponseModeQuery {
				if strings.Contains(redirectURI, "?") {
					redirectURI = redirectURI + "&" + params
				} else {
					redirectURI = redirectURI + "?" + params
				}
			} else if responseMode == constants.ResponseModeFragment {
				if strings.Contains(redirectURI, "#") {
					redirectURI = redirectURI + "&" + params
				} else {
					redirectURI = redirectURI + "#" + params
				}
			}

			handleResponse(gc, responseMode, authURL, redirectURI, map[string]interface{}{
				"type": "authorization_response",
				"response": map[string]interface{}{
					"code":  code,
					"state": state,
				},
			}, http.StatusOK)

			return
		}

		if responseType == constants.ResponseTypeToken || responseType == constants.ResponseTypeIDToken {
			hostname := parsers.GetHost(gc)
			// rollover the session for security
			authToken, err := h.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
				User:        user,
				Nonce:       idTokenNonce,
				Roles:       claims.Roles,
				Scope:       scope,
				LoginMethod: claims.LoginMethod,
				HostName:    hostname,
			})
			if err != nil {
				log.Debug().Err(err).Msg("Error creating auth token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+nonce, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error setting session token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+nonce, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error setting access token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			cookie.SetSession(gc, authToken.FingerPrintHash, h.Config.AppCookieSecure)

			// OAuth 2.0: expires_in is lifetime in seconds (RFC 6749 §4.2.2), not an absolute timestamp.
			expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
			if expiresIn <= 0 {
				// Same rationale as the hybrid branch above: a token
				// issued already-expired is a clock/config bug, not
				// something to silently clamp.
				log.Error().Int64("expires_at", authToken.AccessToken.ExpiresAt).Int64("now", time.Now().Unix()).Msg("implicit: access token issued already-expired")
				gc.JSON(http.StatusInternalServerError, gin.H{
					"error":             "server_error",
					"error_description": "access token issued already-expired",
				})
				return
			}

			// Build response params via url.Values so user-controlled
			// inputs (state, tokens, nonce) cannot inject extra
			// fragment / query parameters.
			fragmentValues := url.Values{}
			fragmentValues.Set("access_token", authToken.AccessToken.Token)
			fragmentValues.Set("token_type", "Bearer")
			fragmentValues.Set("expires_in", strconv.FormatInt(expiresIn, 10))
			fragmentValues.Set("state", state)
			fragmentValues.Set("id_token", authToken.IDToken.Token)

			res := map[string]interface{}{
				"access_token": authToken.AccessToken.Token,
				"id_token":     authToken.IDToken.Token,
				"state":        state,
				"scope":        strings.Join(scope, " "),
				"token_type":   "Bearer",
				"expires_in":   expiresIn,
			}

			// OIDC Core §3.1.2.6: nonce is only echoed when supplied.
			if nonceFromClient {
				fragmentValues.Set("nonce", nonce)
				res["nonce"] = nonce
			}

			if authToken.RefreshToken != nil {
				res["refresh_token"] = authToken.RefreshToken.Token
				fragmentValues.Set("refresh_token", authToken.RefreshToken.Token)
				if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt); err != nil {
					log.Debug().Err(err).Msg("Error setting refresh token")
					handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
					return
				}
			}

			params := fragmentValues.Encode()

			// validateAuthorizeRequest already rejects
			// response_mode=query for token / id_token flows, so the
			// only redirect mode that needs param composition here is
			// fragment. form_post and web_message render via the
			// response map below.
			if responseMode == constants.ResponseModeFragment {
				if strings.Contains(redirectURI, "#") {
					redirectURI = redirectURI + "&" + params
				} else {
					redirectURI = redirectURI + "#" + params
				}
			}

			handleResponse(gc, responseMode, authURL, redirectURI, map[string]interface{}{
				"type":     "authorization_response",
				"response": res,
			}, http.StatusOK)
			return
		}

		handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
	}
}

// supportedResponseTypeSet normalizes a space-delimited response_type
// string into a canonical sorted form and returns whether it is one of
// the supported OIDC Core combinations. Returns the canonical form and
// true on success; empty string and false on unsupported.
func supportedResponseTypeSet(raw string) (string, bool) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return "", false
	}
	// Dedupe + sort.
	seen := map[string]bool{}
	for _, f := range fields {
		f = strings.ToLower(strings.TrimSpace(f))
		if f != "" {
			seen[f] = true
		}
	}
	tokens := make([]string, 0, len(seen))
	for k := range seen {
		tokens = append(tokens, k)
	}
	sort.Strings(tokens)
	canonical := strings.Join(tokens, " ")

	switch canonical {
	// Existing single-value types.
	case "code", "token", "id_token":
		return canonical, true
	// Hybrid combinations (OIDC Core §3.3).
	case "code id_token":
		return canonical, true
	case "code token":
		return canonical, true
	case "code id_token token":
		return canonical, true
	// Implicit with both.
	case "id_token token":
		return canonical, true
	}
	return "", false
}

// validateAuthorizeRequest validates the structural inputs of an
// authorization request and returns an OAuth2 / OIDC error code + a
// human-readable description per RFC 6749 §5.2. Returns ("", "") on
// success.
func (h *httpProvider) validateAuthorizeRequest(responseType, responseMode, clientID, state, codeChallenge string) (string, string) {
	if strings.TrimSpace(state) == "" {
		return "invalid_request", "state is required to prevent CSRF attacks"
	}
	if _, ok := supportedResponseTypeSet(responseType); !ok {
		return "unsupported_response_type", fmt.Sprintf("response_type %q is not supported", responseType)
	}

	if responseMode != constants.ResponseModeQuery && responseMode != constants.ResponseModeWebMessage && responseMode != constants.ResponseModeFragment && responseMode != constants.ResponseModeFormPost {
		return "invalid_request", fmt.Sprintf("invalid response_mode %q; valid values are 'query', 'fragment', 'form_post', 'web_message'", responseMode)
	}

	// OAuth 2.0 Multiple Response Type Encoding Practices §3.0:
	// response_mode=query MUST NOT be used with response types that issue
	// tokens directly (implicit and hybrid flows). Tokens in the query
	// string get logged in proxy access logs, server access logs, and the
	// browser history bar — a real-world credential leak path.
	//
	// Permitted combinations:
	//   response_type=code              → query, fragment, form_post (any)
	//   response_type=token / id_token  → fragment (default) or form_post only
	if responseMode == constants.ResponseModeQuery && responseType != constants.ResponseTypeCode {
		return "invalid_request", fmt.Sprintf("response_mode=query is not allowed for response_type=%s; use fragment or form_post", responseType)
	}

	if h.Config.ClientID != clientID {
		return "invalid_client", "client_id does not match the configured client"
	}

	return "", ""
}

// parseExpiredOrValidIDTokenHintSubject parses an id_token_hint JWT
// and verifies its signature using the configured signing key, but
// deliberately skips expiry/iat validation. Per OIDC Core §3.1.2.1 the
// hint MAY be an expired ID token — RPs are encouraged to send the
// most recent token they have, even after it has expired, so the OP
// can still recognize the user. Only the signature must be valid.
//
// Returns the `sub` claim on success or empty string on any failure
// (parse error, invalid signature, missing claim, wrong token_type).
func (h *httpProvider) parseExpiredOrValidIDTokenHintSubject(idTokenHint string) string {
	if idTokenHint == "" {
		return ""
	}

	keyFunc := func(t *jwt.Token) (interface{}, error) {
		expected := jwt.GetSigningMethod(h.Config.JWTType)
		if expected == nil {
			return nil, errors.New("unsupported signing method")
		}
		if t.Method.Alg() != expected.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		switch expected {
		case jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512:
			return []byte(h.Config.JWTSecret), nil
		case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
			return crypto.ParseRsaPublicKeyFromPemStr(h.Config.JWTPublicKey)
		case jwt.SigningMethodES256, jwt.SigningMethodES384, jwt.SigningMethodES512:
			return crypto.ParseEcdsaPublicKeyFromPemStr(h.Config.JWTPublicKey)
		default:
			return nil, errors.New("unsupported signing method")
		}
	}

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	var claims jwt.MapClaims
	if _, err := parser.ParseWithClaims(idTokenHint, &claims, keyFunc); err != nil {
		return ""
	}
	if claims == nil {
		return ""
	}
	// Defensive: ensure the hint is an id_token, not some other JWT
	// the caller may have sent (access_token, refresh_token).
	if tt, ok := claims["token_type"].(string); ok && tt != "" && tt != "id_token" {
		return ""
	}
	sub, _ := claims["sub"].(string)
	return sub
}

func handleResponse(gc *gin.Context, responseMode, authURI, redirectURI string, data map[string]interface{}, httpStatusCode int) {
	isAuthenticationRequired := false
	if resp, ok := data["response"].(map[string]interface{}); ok {
		if _, hasErr := resp["error"]; hasErr {
			isAuthenticationRequired = true
		}
	}

	if isAuthenticationRequired && responseMode != constants.ResponseModeWebMessage {
		gc.Redirect(http.StatusFound, authURI)
		return
	}

	switch responseMode {
	case constants.ResponseModeQuery, constants.ResponseModeFragment:
		gc.Redirect(http.StatusFound, redirectURI)
		return
	case constants.ResponseModeWebMessage:
		gc.HTML(httpStatusCode, authorizeWebMessageTemplate, gin.H{
			"target_origin":          redirectURI,
			"authorization_response": data,
		})
		return
	case constants.ResponseModeFormPost:
		gc.HTML(httpStatusCode, authorizeFormPostTemplate, gin.H{
			"target_origin":          redirectURI,
			"authorization_response": data["response"],
		})
		return
	}
}
