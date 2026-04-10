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
	"crypto/rand"
	"encoding/base64"
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
		hintedSub := h.parseIDTokenHintSubject(idTokenHint)
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
		// RFC 7636 §4.2: "If the client is capable of using
		// "S256", it MUST use "S256" [...] If the server does not
		// support the transformation, [...] it MUST return [...].
		// If no code_challenge_method is present, the server MUST
		// use "plain" as the default."
		if codeChallengeMethod == "" && codeChallenge != "" {
			codeChallengeMethod = "plain"
		}
		if codeChallengeMethod == "plain" && codeChallenge != "" {
			log.Debug().Msg("PKCE plain method in use — code_verifier will be visible in URL parameters; S256 is recommended")
		}
		// RFC 7636 §4.2: servers MUST support plain and S256
		if codeChallengeMethod != "" && codeChallengeMethod != "S256" && codeChallengeMethod != "plain" {
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "Supported code_challenge_method values are S256 and plain",
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

		// OIDC Core §3.3 hybrid response_type combinations (contain "code" plus tokens).
		isHybrid := responseType == "code id_token" ||
			responseType == "code token" ||
			responseType == "code id_token token"

		// Implicit flows: tokens returned directly, no code exchange.
		// "id_token token" is implicit per OIDC Core §3.2, NOT hybrid.
		isImplicit := responseType == "token" ||
			responseType == "id_token" ||
			responseType == "id_token token"

		if isHybrid || isImplicit {
			// Tokens MUST NOT appear in query strings (OAuth 2.0 Multiple
			// Response Type Encoding Practices §3.0).
			if rawResponseMode == constants.ResponseModeQuery {
				// redirect_uri is validated at this point; redirect the
				// error to the RP per RFC 6749 §4.1.2.1.
				redirectErrorToRP(gc, constants.ResponseModeFragment, redirectURI, state, "invalid_request", "response_mode=query is not allowed for response_type="+responseType)
				return
			}
			// Default to fragment when the client did not explicitly
			// specify one (the global default may be query).
			if rawResponseMode == "" {
				responseMode = constants.ResponseModeFragment
			}
		}

		if errCode, errDesc := h.validateAuthorizeRequest(responseType, responseMode, state); errCode != "" {
			log.Debug().Str("error", errCode).Str("error_description", errDesc).Msg("Invalid request")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             errCode,
				"error_description": errDesc,
			})
			return
		}

		// OIDC Core §3.2.2.1 / §3.3.2.1: nonce is REQUIRED when id_token
		// appears in the response_type (implicit and hybrid flows that
		// return id_token directly from the authorization endpoint).
		requiresNonce := strings.Contains(responseType, "id_token")
		if requiresNonce && nonce == "" {
			redirectErrorToRP(gc, responseMode, redirectURI, state, "invalid_request", "nonce is required for response_type="+responseType)
			return
		}

		// OIDC Core §3.2.2.1: when response_type=token and scope includes
		// openid, the RP should use id_token or id_token token instead.
		// We log a warning but don't reject to avoid breaking existing flows.
		if responseType == "token" {
			for _, s := range scope {
				if s == "openid" {
					log.Debug().Msg("response_type=token with openid scope — consider using id_token or id_token token instead")
					break
				}
			}
		}

		// Generate code only for flows that include "code" in response_type.
		hasCodeFlow := strings.Contains(responseType, "code")
		code := ""
		if hasCodeFlow {
			code = uuid.New().String()
		}
		if nonce == "" {
			nonce = uuid.New().String()
		}

		log = log.With().Str("response_type", responseType).Str("response_mode", responseMode).Str("scope", scopeString).Str("client_id", clientID).Bool("has_code_challenge", codeChallenge != "").Logger()

		// TODO add state with timeout
		// used for response mode query or fragment
		authState := "state=" + url.QueryEscape(state) + "&scope=" + url.QueryEscape(scopeString) + "&redirect_uri=" + url.QueryEscape(redirectURI) + "&response_mode=" + url.QueryEscape(responseMode) + "&response_type=" + url.QueryEscape(responseType) + "&client_id=" + url.QueryEscape(clientID)
		// OIDC Core §3.1.2.1: login_hint and ui_locales are forwarded
		// to the login UI so it can pre-fill the email field and pick
		// the UI language.
		if loginHint != "" {
			authState += "&login_hint=" + url.QueryEscape(loginHint)
		}
		if uiLocales != "" {
			authState += "&ui_locales=" + url.QueryEscape(uiLocales)
		}
		// Forward all RP-provided OIDC parameters through the login UI so
		// the React app can send them back on the second /authorize
		// round-trip. Without this, the code flow loses the RP-provided
		// values and Auth0/Okta/Keycloak reject the resulting tokens.
		authState += "&nonce=" + url.QueryEscape(nonce)
		if codeChallenge != "" {
			authState += "&code_challenge=" + url.QueryEscape(codeChallenge)
			authState += "&code_challenge_method=" + url.QueryEscape(codeChallengeMethod)
		}

		if hasCodeFlow {
			authState += "&code=" + code
			// Store code_challenge with method so token endpoint can verify.
			// Format: "challenge::method@@session" or "@@session" (no PKCE).
			challengeData := codeChallenge
			if codeChallenge != "" {
				challengeData = codeChallenge + "::" + codeChallengeMethod
			}
			if err := h.MemoryStoreProvider.SetState(state, code+"@@"+challengeData+"@@"+nonce+"@@"+url.QueryEscape(redirectURI)); err != nil {
				log.Debug().Err(err).Msg("Error setting temp code")
				gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
		} else {
			if err := h.MemoryStoreProvider.SetState(state, nonce); err != nil {
				log.Debug().Err(err).Msg("Error setting temp nonce")
				gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
		}

		authURL := baseAppPath + "?" + authState

		if screenHint == constants.ScreenHintSignUp {
			authURL = signupPath + "?" + authState
		}

		if responseMode == constants.ResponseModeFragment && screenHint == constants.ScreenHintSignUp {
			authURL = signupPath + "#" + authState
		} else if responseMode == constants.ResponseModeFragment {
			authURL = baseAppPath + "#" + authState
		}

		// Reject if code_challenge_method is set without code_challenge
		if responseType == constants.ResponseTypeCode && codeChallenge == "" && codeChallengeMethod != "" {
			redirectErrorToRP(gc, responseMode, redirectURI, state, "invalid_request", "code_challenge is required when code_challenge_method is specified")
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

		// When prompt=login and a valid session cookie exists, don't discard
		// it. The normal flow below validates it, performs a session rollover,
		// stores the authorization code state, and redirects to the RP.
		// Discarding the session here would send the user to the login UI
		// where the React SDK auto-detects the still-valid cookie, redirects
		// immediately, but the authorization code state is never stored
		// because the login mutation is never called.
		//
		// For max_age=0 or max_age-exceeded, we DO discard the session
		// because the spec requires actual re-authentication based on time.
		if forceReauth && !(prompt == "login" && err == nil && sessionToken != "") {
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
				cspNonce := setFormPostCSP(gc)
				gc.HTML(http.StatusOK, authorizeFormPostTemplate, gin.H{
					"target_origin":          redirectURI,
					"authorization_response": errData["response"],
					"csp_nonce":              cspNonce,
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
		if err := h.MemoryStoreProvider.DeleteUserSession(sessionKey, claims.Nonce); err != nil {
			log.Debug().Err(err).Str("session_key", sessionKey).Msg("Failed to delete old session during rollover")
		}

		if isHybrid {
			hostname := parsers.GetHost(gc)
			// For hybrid flows we mint tokens AND a code. Setting Code
			// on the AuthTokenConfig causes CreateAuthToken to populate
			// cfg.CodeHash, which in turn causes CreateIDToken to emit
			// the c_hash claim per OIDC Core §3.3.2.11.
			authToken, err := h.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
				User:        user,
				Nonce:       nonce,
				Code:        code,
				Roles:       claims.Roles,
				Scope:       scope,
				LoginMethod: claims.LoginMethod,
				HostName:    hostname,
				AuthTime:    claims.IssuedAt,
			})
			if err != nil {
				log.Debug().Err(err).Msg("Error creating auth token for hybrid response")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			// OIDC Core §3.3: hybrid flow codes are exchanged at /oauth/token
			// which calls ValidateBrowserSession — store AES-encrypted session
			// (FingerPrintHash), not the raw nonce (FingerPrint).
			hybridChallengeData := codeChallenge
			if codeChallenge != "" {
				hybridChallengeData = codeChallenge + "::" + codeChallengeMethod
			}
			if err := h.MemoryStoreProvider.SetState(code, hybridChallengeData+"@@"+authToken.FingerPrintHash+"@@"+nonce+"@@"+url.QueryEscape(redirectURI)); err != nil {
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
				expiresIn = 1
			}

			hasAccessToken := responseType == "code token" ||
				responseType == "code id_token token"
			hasIDToken := responseType == "code id_token" ||
				responseType == "code id_token token"

			// Build the response map. Hybrid always includes code + state.
			res := map[string]interface{}{
				"code":       code,
				"state":      state,
				"token_type": "Bearer",
				"expires_in": expiresIn,
			}
			if hasAccessToken {
				res["access_token"] = authToken.AccessToken.Token
			}
			if hasIDToken {
				res["id_token"] = authToken.IDToken.Token
			}

			// Build the fragment params string for redirect modes.
			params := "code=" + code + "&state=" + state + "&token_type=Bearer&expires_in=" + strconv.FormatInt(expiresIn, 10)
			if hasAccessToken {
				params += "&access_token=" + authToken.AccessToken.Token
			}
			if hasIDToken {
				params += "&id_token=" + authToken.IDToken.Token
			}

			// Hybrid defaults to fragment; the pre-check above rejected query.
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

		// OIDC Core §3.2.2.5: response_type="id_token token" is an implicit
		// flow returning both id_token and access_token directly. No code, no
		// refresh_token. Nonce is required (enforced above).
		if responseType == "id_token token" {
			hostname := parsers.GetHost(gc)
			authToken, err := h.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
				User:        user,
				Nonce:       nonce,
				Roles:       claims.Roles,
				Scope:       scope,
				LoginMethod: claims.LoginMethod,
				HostName:    hostname,
				AuthTime:    claims.IssuedAt,
			})
			if err != nil {
				log.Debug().Err(err).Msg("Error creating auth token for id_token token response")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error persisting session for id_token token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}
			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error persisting access token for id_token token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}
			cookie.SetSession(gc, authToken.FingerPrintHash, h.Config.AppCookieSecure)

			expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
			if expiresIn <= 0 {
				expiresIn = 1
			}

			res := map[string]interface{}{
				"access_token": authToken.AccessToken.Token,
				"id_token":     authToken.IDToken.Token,
				"state":        state,
				"token_type":   "Bearer",
				"expires_in":   expiresIn,
			}

			params := "access_token=" + authToken.AccessToken.Token +
				"&id_token=" + authToken.IDToken.Token +
				"&token_type=Bearer&expires_in=" + strconv.FormatInt(expiresIn, 10) +
				"&state=" + state

			// Fragment-only: tokens MUST NOT appear in query strings.
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
				AuthTime:    claims.IssuedAt,
			})
			if err != nil {
				log.Debug().Err(err).Msg("Error creating session token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			// TODO: add state with timeout
			// if err := memorystore.Provider.SetState(codeChallenge, code+"@"+newSessionToken); err != nil {
			// 	log.Debug("SetState failed: ", err)
			// 	handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
			// 	return
			// }

			// TODO: add state with timeout
			codeChallengeData := codeChallenge
			if codeChallenge != "" {
				codeChallengeData = codeChallenge + "::" + codeChallengeMethod
			}
			if err := h.MemoryStoreProvider.SetState(code, codeChallengeData+"@@"+newSessionToken+"@@"+nonce+"@@"+url.QueryEscape(redirectURI)); err != nil {
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

			// in case, response type is code and user is already logged in send the code and state
			// and cookie session will already be rolled over and set
			// gc.HTML(http.StatusOK, authorizeWebMessageTemplate, gin.H{
			// 	"target_origin": redirectURI,
			// 	"authorization_response": map[string]interface{}{
			// 		"type": "authorization_response",
			// 		"response": map[string]string{
			// 			"code":  code,
			// 			"state": state,
			// 		},
			// 	},
			// })

			// RFC 6749 §4.1.2: Authorization code response MUST only include code and state
			params := "code=" + code + "&state=" + state
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

		// OIDC Core §3.2.2.5: response_type=id_token returns ONLY id_token
		// and state. No access_token, token_type, or expires_in.
		if responseType == constants.ResponseTypeIDToken {
			hostname := parsers.GetHost(gc)
			authToken, err := h.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
				User:        user,
				Nonce:       nonce,
				Roles:       claims.Roles,
				Scope:       scope,
				LoginMethod: claims.LoginMethod,
				HostName:    hostname,
				AuthTime:    claims.IssuedAt,
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

			cookie.SetSession(gc, authToken.FingerPrintHash, h.Config.AppCookieSecure)

			// OIDC Core §3.2.2.5: response params are id_token + state only.
			// The nonce is carried inside the id_token JWT claims (not as a
			// separate response parameter).
			res := map[string]interface{}{
				"id_token": authToken.IDToken.Token,
				"state":    state,
			}
			params := "id_token=" + authToken.IDToken.Token + "&state=" + state

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

		// RFC 6749 §4.2.2: response_type=token is a pure OAuth 2.0 implicit
		// flow. Return ONLY access_token, token_type, expires_in, state.
		// No id_token (not OIDC). No refresh_token (implicit MUST NOT).
		if responseType == constants.ResponseTypeToken {
			hostname := parsers.GetHost(gc)
			authToken, err := h.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
				User:        user,
				Nonce:       nonce,
				Roles:       claims.Roles,
				Scope:       scope,
				LoginMethod: claims.LoginMethod,
				HostName:    hostname,
				AuthTime:    claims.IssuedAt,
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

			expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
			if expiresIn <= 0 {
				expiresIn = 1
			}

			// RFC 6749 §4.2.2: implicit token response params.
			params := "access_token=" + authToken.AccessToken.Token +
				"&token_type=Bearer&expires_in=" + strconv.FormatInt(expiresIn, 10) +
				"&state=" + state

			res := map[string]interface{}{
				"access_token": authToken.AccessToken.Token,
				"state":        state,
				"token_type":   "Bearer",
				"expires_in":   expiresIn,
			}

			// Fragment-only: tokens MUST NOT appear in query strings.
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

// validateAuthorizeRequest validates the authorize request parameters and
// returns RFC 6749 §4.1.2.1 compliant error code and description on failure.
// Returns empty strings when validation passes.
func (h *httpProvider) validateAuthorizeRequest(responseType, responseMode, state string) (string, string) {
	if strings.TrimSpace(state) == "" {
		return "invalid_request", "state parameter is required"
	}

	if responseMode != constants.ResponseModeQuery && responseMode != constants.ResponseModeWebMessage && responseMode != constants.ResponseModeFragment && responseMode != constants.ResponseModeFormPost {
		return "invalid_request", "response_mode must be one of: query, fragment, form_post, web_message"
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

	return "", ""
}

// parseIDTokenHintSubject parses and verifies an id_token_hint JWT
// against the server's own signing key. Per OIDC Core §3.1.2.1 the hint
// need not be unexpired — only structurally valid. Returns the subject
// claim on success so callers can use it for logging / future
// user-selection enforcement. Returns empty string on any failure.
func (h *httpProvider) parseIDTokenHintSubject(idTokenHint string) string {
	if idTokenHint == "" {
		return ""
	}
	claims, err := h.TokenProvider.ParseJWTToken(idTokenHint)
	if err != nil || claims == nil {
		return ""
	}
	if tt, ok := claims["token_type"].(string); ok && tt != "" && tt != "id_token" {
		return ""
	}
	sub, _ := claims["sub"].(string)
	return sub
}

// setFormPostCSP overrides the Content-Security-Policy header to allow
// OIDC Form Post Response Mode (OAuth 2.0 Form Post Response Mode §1).
// Returns a cryptographic nonce for use in script tags.
func setFormPostCSP(gc *gin.Context) string {
	// Generate a cryptographic nonce for CSP script-src.
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		// Fallback: allow unsafe-inline if crypto/rand fails (should never happen).
		nonceBytes = []byte("fallback-nonce-value")
	}
	cspNonce := base64.RawURLEncoding.EncodeToString(nonceBytes)

	gc.Writer.Header().Set("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self' 'nonce-"+cspNonce+"'; "+
			"style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data: https:; "+
			"font-src 'self' data:; "+
			"connect-src 'self'; "+
			"frame-ancestors 'none'; "+
			"base-uri 'self'; "+
			"form-action *;")
	return cspNonce
}

// redirectErrorToRP sends an OAuth2/OIDC error to the RP's redirect_uri
// using the configured response_mode. Per RFC 6749 §4.1.2.1 this MUST only
// be called after redirect_uri has been validated. If redirect_uri is the
// default "/app" fallback (unvalidated), falls back to a JSON error response.
func redirectErrorToRP(gc *gin.Context, responseMode, redirectURI, state, errCode, errDesc string) {
	// If redirect_uri is the default "/app" fallback, we have no validated
	// RP endpoint to redirect to — return JSON error instead.
	if redirectURI == "/app" {
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             errCode,
			"error_description": errDesc,
		})
		return
	}

	errParams := "error=" + url.QueryEscape(errCode) +
		"&error_description=" + url.QueryEscape(errDesc)
	if state != "" {
		errParams += "&state=" + url.QueryEscape(state)
	}

	errData := map[string]interface{}{
		"type": "authorization_response",
		"response": map[string]interface{}{
			"error":             errCode,
			"error_description": errDesc,
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
		cspNonce := setFormPostCSP(gc)
		gc.HTML(http.StatusOK, authorizeFormPostTemplate, gin.H{
			"target_origin":          redirectURI,
			"authorization_response": errData["response"],
			"csp_nonce":              cspNonce,
		})
	default:
		// query or fragment
		errRedirectURI := redirectURI
		if responseMode == constants.ResponseModeFragment {
			if strings.Contains(errRedirectURI, "#") {
				errRedirectURI += "&" + errParams
			} else {
				errRedirectURI += "#" + errParams
			}
		} else {
			if strings.Contains(errRedirectURI, "?") {
				errRedirectURI += "&" + errParams
			} else {
				errRedirectURI += "?" + errParams
			}
		}
		gc.Redirect(http.StatusFound, errRedirectURI)
	}
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
		cspNonce := setFormPostCSP(gc)
		gc.HTML(httpStatusCode, authorizeFormPostTemplate, gin.H{
			"target_origin":          redirectURI,
			"authorization_response": data["response"],
			"csp_nonce":              cspNonce,
		})
		return
	}
}
