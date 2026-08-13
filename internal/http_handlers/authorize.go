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

	"github.com/authorizerdev/authorizer/internal/clientmetadata"
	"github.com/authorizerdev/authorizer/internal/codestate"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/token"
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
	return func(gc *gin.Context) {
		// log is request-scoped (declared inside the closure, not captured from
		// the enclosing func) so concurrent requests each mutate their own
		// value when reassigning below (e.g. attaching client_id) — a logger
		// declared outside the closure and reassigned here would be shared,
		// mutable state race-read/written by every concurrent request.
		log := h.Log.With().Str("func", "AuthorizeHandler").Logger()
		// RFC 6749 §3.1 / OIDC Core §3.1.2.1: the authorization endpoint
		// MUST support GET and MAY support POST (form-urlencoded body).
		// FormValue reads the POST body first, falling back to the query
		// string, so the same handler serves both methods.
		param := func(key string) string {
			return strings.TrimSpace(gc.Request.FormValue(key))
		}

		redirectURI := param("redirect_uri")
		responseType := param("response_type")
		state := param("state")
		codeChallenge := param("code_challenge")
		scopeString := param("scope")
		clientID := param("client_id")
		responseMode := param("response_mode")
		rawResponseMode := responseMode
		nonce := param("nonce")
		screenHint := param("screen_hint")

		// RFC 8707 resource indicator: optional. When present it binds the
		// issued authorization code (and, at exchange time, the access token's
		// `aud`) to the target resource server. RFC 8707 §2 permits multiple
		// resource values, but this flow issues a single-audience access token
		// (matching token_exchange.go), so exactly one is allowed — reject a
		// repeated parameter rather than silently picking one.
		resource := param("resource")

		// OIDC Core §3.1.2.1 standard authorization request parameters.
		loginHint := param("login_hint")
		uiLocales := param("ui_locales")
		prompt := param("prompt")
		maxAgeStr := param("max_age")
		idTokenHint := param("id_token_hint")

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

		// True when the client asserted its own identity through RFC 7591 rather
		// than being created by an operator. Set during redirect validation below,
		// and read by the consent gate and the PKCE check.
		//
		// It stays false when no redirect_uri was supplied, because that branch
		// does no client lookup. That is deliberate and safe rather than a gap:
		// with no redirect_uri the code is delivered to this server's own /app,
		// so a client that skipped consent this way never receives it.
		var selfRegistered bool

		if redirectURI == "" {
			redirectURI = "/app"
		} else {
			hostname := parsers.GetHost(gc)
			// Shared with /app, which re-renders this same redirect_uri on the
			// login page. Both must apply the identical rule; when they did not,
			// a client whose registered redirect was outside AllowedOrigins
			// passed here and was refused there.
			check, cErr := h.checkClientRedirectURI(gc.Request.Context(), clientID, redirectURI, hostname)
			switch {
			case errors.Is(cErr, errClientUnresolvable):
				log.Debug().Err(cErr).Str("client_id", clientID).Msg("could not resolve client metadata document")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_client",
					"error_description": "could not resolve the client metadata document for this client_id",
				})
				return
			case cErr != nil:
				log.Warn().Err(cErr).Str("client_id", clientID).
					Msg("client lookup failed; refusing rather than falling back to the origin allow-list")
				gc.JSON(http.StatusServiceUnavailable, gin.H{
					"error":             "temporarily_unavailable",
					"error_description": "could not verify the client; please retry",
				})
				return
			}
			// A client that registered ITSELF through RFC 7591 is remembered so
			// the consent gate and PKCE check further down do not repeat the
			// lookup. Operator-created clients are left false: they were vouched
			// for by a human who entered their redirect URIs, which is the whole
			// basis for not interrupting their users.
			selfRegistered = check.SelfRegistered
			if !check.Valid {
				log.Debug().Msg("Invalid redirect URI")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_request",
					"error_description": "invalid redirect_uri",
				})
				return
			}
		}

		// RFC 8707: reject a repeated resource parameter (single audience only).
		// redirect_uri is validated above, so redirectErrorToRP is safe here.
		// Debug only, no security-event metric: a routine malformed request,
		// not evidence of an attack (keeps SecurityEventsTotal signal clean).
		if vals := gc.Request.Form["resource"]; len(vals) > 1 {
			log.Debug().Str("client_id", clientID).Msg("rejected: repeated resource parameter")
			redirectErrorToRP(gc, responseMode, redirectURI, state, "invalid_request", "only one resource parameter is supported")
			return
		}

		// RFC 8707 §2: the resource indicator MUST be an absolute URI and MUST
		// NOT include a fragment component. The RFC-conventional error code for
		// a rejected resource is invalid_target.
		if resource != "" && !isValidResourceIndicator(resource) {
			log.Debug().Str("client_id", clientID).Str("resource", resource).Msg("rejected: invalid resource indicator")
			redirectErrorToRP(gc, responseMode, redirectURI, state, "invalid_target", "resource must be an absolute URI without a fragment")
			return
		}

		// OIDCC-3.1.2.6 (JAR, RFC 9101): the server must either process a
		// `request`/`request_uri` object or reject it with
		// request_not_supported / request_uri_not_supported. We don't parse
		// request objects, so reject explicitly rather than silently
		// ignoring the parameter and falling through to a confusing
		// generic validation error.
		if requestObject, requestURI := param("request"), param("request_uri"); requestObject != "" || requestURI != "" {
			errCode := "request_not_supported"
			errDesc := "the request parameter is not supported"
			if requestURI != "" {
				errCode = "request_uri_not_supported"
				errDesc = "the request_uri parameter is not supported"
			}
			log.Debug().Str("client_id", clientID).Str("error", errCode).Msg("rejected: request/request_uri (JAR) not supported")
			redirectErrorToRP(gc, responseMode, redirectURI, state, errCode, errDesc)
			return
		}

		// RFC 6749 §3.1.1: response_type is REQUIRED. gin's Query() can't
		// distinguish an absent parameter from an empty one, so both land
		// here — reject rather than silently defaulting to an implicit-flow
		// token grant the client never asked for.
		if responseType == "" {
			log.Debug().Str("client_id", clientID).Msg("rejected: response_type is required")
			redirectErrorToRP(gc, responseMode, redirectURI, state, "invalid_request", "response_type is required")
			return
		}

		codeChallengeMethod := param("code_challenge_method")
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
		// OAuth 2.1 strict mode: PKCE "plain" is removed — S256 only. Recorded
		// as a security event (not just logged): a client presenting "plain"
		// while strict mode is on is attempting the exact downgrade this
		// setting exists to block.
		if h.Config.OAuth21Strict && codeChallengeMethod == "plain" && codeChallenge != "" {
			metrics.RecordSecurityEvent("pkce_plain_rejected_strict", "authorize_endpoint")
			log.Warn().Str("client_id", clientID).Msg("rejected: PKCE plain method not allowed under OAuth 2.1 strict mode")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "code_challenge_method=plain is not allowed; use S256",
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

		// PKCE is MANDATORY for a self-asserted client — one that registered
		// itself via RFC 7591 or presented a Client ID Metadata Document. Both
		// are public clients, and for public clients PKCE is not advisory:
		//
		//   RFC 9700 §2.1.1: "Public clients MUST use PKCE".
		//   OAuth 2.1 §4.1.1: clients "MUST use code_challenge and code_verifier
		//   and authorization servers MUST enforce their use".
		//   MCP authorization (2025-11-25): clients MUST implement PKCE and MUST
		//   use S256 when technically capable.
		//
		// Enforced HERE rather than left to the token endpoint. Without a secret,
		// PKCE is the only thing binding the code to the instance that started
		// the flow, so a request that omits it is unauthenticatable from the
		// start; the token endpoint's "code_verifier or client_secret" rule would
		// eventually refuse it, but only after the user had already logged in and
		// approved — and with an error naming client_secret, which a public
		// client can never supply. Failing at /authorize keeps the refusal
		// truthful and free of user interaction.
		//
		// Answered as 400 rather than redirected to the client, matching the
		// neighbouring PKCE validations above and the MCP authorization spec's
		// error table, which maps a malformed authorization request to 400.
		if selfRegistered || (h.ClientMetadataProvider != nil && clientmetadata.IsMetadataClientIDFor(clientID, h.Config.ClientID)) {
			// A self-asserted client may not use an implicit response type at
			// all, so PKCE below is unconditional rather than carved out for
			// flows that issue no code.
			//
			// OAuth 2.1 removes the implicit grant, and MCP mandates OAuth 2.1.
			// Independently of that, implicit hands a bearer token straight to
			// the redirect URI in a URL fragment with nothing binding it to the
			// requester — for a client whose identity nobody verified, that is
			// the worst available combination. Both registration paths declare
			// response_types ["code"] anyway (the registration endpoint refuses
			// anything else, and the CIMD spec's example does the same), so this
			// enforces at /authorize what the client already said it would do.
			if isImplicit {
				metrics.RecordSecurityEvent("self_registered_client_implicit_rejected", "authorize_endpoint")
				log.Debug().Str("client_id", clientID).Str("response_type", responseType).
					Msg("rejected: self-registered client requested an implicit response type")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "unsupported_response_type",
					"error_description": "this client may only use response_type=code",
				})
				return
			}
			if codeChallenge == "" {
				metrics.RecordSecurityEvent("self_registered_client_missing_pkce", "authorize_endpoint")
				log.Debug().Str("client_id", clientID).Msg("rejected: self-registered client omitted code_challenge")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_request",
					"error_description": "code_challenge is required for this client; use PKCE with code_challenge_method=S256",
				})
				return
			}
			// S256 only, independent of OAuth21Strict. "plain" carries the
			// verifier in the same authorization request as the challenge, so it
			// protects nothing against an attacker who can read that request —
			// which is the threat model for a client with no secret.
			if codeChallengeMethod != "S256" {
				metrics.RecordSecurityEvent("self_registered_client_weak_pkce", "authorize_endpoint")
				log.Debug().Str("client_id", clientID).Str("method", codeChallengeMethod).
					Msg("rejected: self-registered client used a non-S256 code_challenge_method")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_request",
					"error_description": "code_challenge_method must be S256 for this client",
				})
				return
			}
		}

		// OAuth 2.1 strict mode: the implicit grant is removed. Reject EVERY
		// response type that delivers a bearer access token into the URL
		// fragment — not just "token" and "id_token token" but also the
		// front-channel-token hybrids "code token" and "code id_token token",
		// which equally return an access_token in the fragment (see the
		// hasAccessToken dispatch below). responseType is already canonical
		// (sorted/deduped/lowercased), so a bearer access token is present iff
		// the space-separated components include the exact component "token".
		// The check is component-exact (not a substring) so "id_token" — which
		// contains "token" only as a substring — is NOT matched.
		if h.Config.OAuth21Strict && responseTypeHasBareToken(responseType) {
			metrics.RecordSecurityEvent("bare_token_response_type_rejected_strict", "authorize_endpoint")
			log.Warn().Str("client_id", clientID).Str("response_type", responseType).Msg("rejected: implicit/bare-token response_type not allowed under OAuth 2.1 strict mode")
			redirectErrorToRP(gc, responseMode, redirectURI, state, "unsupported_response_type", "response_type="+responseType+" is not supported")
			return
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
		// values and strict downstream RPs (e.g. Auth0, Okta, Keycloak) reject the
		// resulting tokens.
		authState += "&nonce=" + url.QueryEscape(nonce)
		if codeChallenge != "" {
			authState += "&code_challenge=" + url.QueryEscape(codeChallenge)
			authState += "&code_challenge_method=" + url.QueryEscape(codeChallengeMethod)
		}
		// The RFC 8707 resource indicator has to survive this round-trip too,
		// and its absence failed silently in the worst way: the flow completed,
		// a token was issued, and only its `aud` was wrong — the client id
		// instead of the resource the client asked for. The resource server then
		// rejected every call with a 401 that looked like a credential problem.
		//
		// It only broke for users who were NOT already signed in, because a live
		// session skips this branch entirely. So the first connection failed and
		// a retry after logging in elsewhere succeeded, which is close to the
		// worst possible symptom to debug.
		//
		// Already validated as an absolute URI without a fragment above; escaped
		// here for the same reason every other value is.
		if resource != "" {
			authState += "&resource=" + url.QueryEscape(resource)
		}

		if hasCodeFlow {
			authState += "&code=" + code
			// Store code_challenge with method so token endpoint can verify.
			// Format: "challenge::method@@session" or "@@session" (no PKCE).
			challengeData := codeChallenge
			if codeChallenge != "" {
				challengeData = codeChallenge + "::" + codeChallengeMethod
			}
			// [4] carries the RFC 8707 resource (url-escaped, empty when absent)
			// so the login/signup/session/auth_response services can rebind it
			// to the code state they persist after a fresh login.
			if err := h.MemoryStoreProvider.SetState(state, codestate.EncodeAuthorize(codestate.Authorize{
				Code:        code,
				Challenge:   challengeData,
				Nonce:       nonce,
				RedirectURI: redirectURI,
				Resource:    resource,
				ClientID:    clientID,
			})); err != nil {
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
					// Measured against AuthTime (the End-User's actual last
					// authentication), not IssuedAt (which refreshes on
					// every silent rollover) — otherwise a client can keep
					// a session alive past max_age indefinitely by polling
					// faster than the max_age window.
					authTime := sd.EffectiveAuthTime()
					if time.Now().Unix()-authTime > int64(maxAge) {
						log.Debug().Int("max_age", maxAge).Int64("session_age", time.Now().Unix()-authTime).Msg("session exceeds max_age — forcing re-auth")
						forceReauth = true
					}
				}
			}
		}

		// OIDC Core §3.1.2.1: prompt=login and an exceeded max_age both
		// require actually re-authenticating the user, not just ignoring
		// the local `sessionToken` variable for this one request — the
		// browser's session cookie stays valid, so the login UI's own
		// session check (getSession) still sees a logged-in user and
		// immediately bounces back to /authorize without ever rendering a
		// login form, and the authorization code state is never stored.
		// Revoke the session for real (memory store + cookie) so the login
		// UI genuinely sees a logged-out user and forces fresh credentials.
		if forceReauth && err == nil && sessionToken != "" {
			if decryptedFingerPrint, decErr := crypto.DecryptAES(h.ClientSecret, sessionToken); decErr == nil {
				var sd token.SessionData
				if jsonErr := json.Unmarshal([]byte(decryptedFingerPrint), &sd); jsonErr == nil {
					revokeKey := sd.Subject
					if sd.LoginMethod != "" {
						revokeKey = sd.LoginMethod + ":" + sd.Subject
					}
					_ = h.MemoryStoreProvider.DeleteUserSession(revokeKey, sd.Nonce)
				}
			}
			cookie.DeleteSession(gc, h.Config.AppCookieSecure, cookie.ParseSameSite(h.Config.AppCookieSameSite))
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

		// Consent gate for self-asserted (CIMD) clients. Placed here, after the
		// session is validated and the user resolved, for two reasons: the page
		// names the account being granted, and an unauthenticated visitor must
		// reach the login UI first rather than being asked to approve something
		// on behalf of nobody.
		//
		// Skipped once ConsentHandler has replayed the request — it sets the
		// marker only after its own store lookup and session check pass, so this
		// cannot be short-circuited by anything the client sends.
		isCIMDClient := h.ClientMetadataProvider != nil && clientmetadata.IsMetadataClientIDFor(clientID, h.Config.ClientID)
		if isCIMDClient || selfRegistered {
			// Consume the grant recorded by ConsentHandler. Single-use and keyed
			// to (user, client), so a consent authorizes one authorization
			// request — not every subsequent one until the store expires it.
			// Keyed to THIS request's parameters and consumed atomically, so a
			// grant authorizes the one request it was given for and cannot be
			// redeemed twice by concurrent callers.
			grantKey := consentGrantKey(user.ID, clientID, originalParams(gc).Encode())
			granted, gErr := h.MemoryStoreProvider.GetAndRemoveState(grantKey)
			if gErr != nil || granted == "" {
				// Both kinds of self-asserted client reach the same page, because
				// the user-facing fact is the same either way: nobody at this
				// deployment vouched for the name being displayed. RFC 7591 §5
				// asks for exactly this — it warns that "a rogue client might use
				// the name and logo of a legitimate client" and tells servers to
				// "present warning messages to end-users about dynamically
				// registered clients".
				//
				// Where the name and redirect list come from is the only
				// difference: a document fetched from the client's own URL, or
				// the row it wrote at registration.
				var clientName string
				var redirectURIs []string
				if isCIMDClient {
					doc, dErr := h.ClientMetadataProvider.Resolve(gc.Request.Context(), clientID)
					if dErr != nil {
						log.Debug().Err(dErr).Msg("could not resolve client metadata document for consent")
						gc.JSON(http.StatusBadRequest, gin.H{
							"error":             "invalid_client",
							"error_description": "could not resolve the client metadata document for this client_id",
						})
						return
					}
					clientName, redirectURIs = doc.ClientName, doc.RedirectURIs
				} else {
					client, cErr := h.StorageProvider.GetClientByClientID(gc.Request.Context(), clientID)
					// GetClientByClientID is the documented (nil, nil)-on-absent
					// exception, so absent and unavailable arrive differently and
					// must be answered differently: invalid_client tells a caller
					// their client_id is permanently wrong, which during an
					// outage is both false and non-retryable. Neither outcome
					// skips consent — reaching this point already established the
					// client is self-asserted.
					switch {
					case cErr != nil:
						log.Warn().Err(cErr).Msg("client lookup failed while preparing consent")
						gc.JSON(http.StatusServiceUnavailable, gin.H{
							"error":             "temporarily_unavailable",
							"error_description": "could not load the client; please retry",
						})
						return
					case client == nil:
						// The row was there during redirect validation and is
						// gone now — deleted mid-flight.
						log.Debug().Msg("registered client disappeared between redirect validation and consent")
						gc.JSON(http.StatusBadRequest, gin.H{
							"error":             "invalid_client",
							"error_description": "could not load the client for this client_id",
						})
						return
					}
					clientName, redirectURIs = client.Name, client.ParsedRedirectURIs()
				}
				// OIDC Core §3.1.2.1: with prompt=none the authorization server
				// "MUST NOT display any authentication or consent user interface
				// pages". A self-asserted client still requires consent, and the
				// two cannot both be satisfied — so the request fails with
				// consent_required and the client decides whether to retry
				// interactively.
				//
				// The two prompt=none guards above only cover the UNAUTHENTICATED
				// case. Without this, a caller with a live session would be shown
				// a consent page in response to a request that forbids one.
				if prompt == "none" {
					redirectErrorToRP(gc, responseMode, redirectURI, state, "consent_required",
						"prompt=none was requested but this client requires consent")
					return
				}
				h.renderConsent(gc, consentClient{
					ClientID:     clientID,
					ClientName:   clientName,
					RedirectURIs: redirectURIs,
				}, redirectURI, user.ID, refs.StringValue(user.Email), scope)
				return
			}
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
				AuthTime:    claims.EffectiveAuthTime(),
				ClientID:    clientID,
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
			if err := h.MemoryStoreProvider.SetState(code, codestate.EncodeCode(codestate.Code{
				Challenge:   hybridChallengeData,
				Session:     authToken.FingerPrintHash,
				Nonce:       nonce,
				RedirectURI: redirectURI,
				ClientID:    clientID,
			})); err != nil {
				log.Debug().Err(err).Msg("Error setting temp code for hybrid")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}
			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, crypto.HashSessionValue(authToken.FingerPrintHash), authToken.SessionTokenExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error persisting session for hybrid")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}
			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, crypto.HashSessionValue(authToken.AccessToken.Token), authToken.AccessToken.ExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error persisting access token for hybrid")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}
			cookie.SetSession(gc, authToken.FingerPrintHash, h.Config.AppCookieSecure, cookie.ParseSameSite(h.Config.AppCookieSameSite))
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
				AuthTime:    claims.EffectiveAuthTime(),
				ClientID:    clientID,
			})
			if err != nil {
				log.Debug().Err(err).Msg("Error creating auth token for id_token token response")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, crypto.HashSessionValue(authToken.FingerPrintHash), authToken.SessionTokenExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error persisting session for id_token token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}
			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, crypto.HashSessionValue(authToken.AccessToken.Token), authToken.AccessToken.ExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error persisting access token for id_token token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}
			cookie.SetSession(gc, authToken.FingerPrintHash, h.Config.AppCookieSecure, cookie.ParseSameSite(h.Config.AppCookieSameSite))

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
				AuthTime:    claims.EffectiveAuthTime(),
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
			// [4] binds the RFC 8707 resource to the code (url-escaped, empty
			// when absent); the token endpoint enforces the echoed resource
			// matches this and sets the access token `aud` to it.
			if err := h.MemoryStoreProvider.SetState(code, codestate.EncodeCode(codestate.Code{
				Challenge:   codeChallengeData,
				Session:     newSessionToken,
				Nonce:       nonce,
				RedirectURI: redirectURI,
				Resource:    resource,
				ClientID:    clientID,
			})); err != nil {
				log.Debug().Err(err).Msg("Error setting temp code")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+newSessionTokenData.Nonce, newSessionToken, newSessionExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error setting session token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			cookie.SetSession(gc, newSessionToken, h.Config.AppCookieSecure, cookie.ParseSameSite(h.Config.AppCookieSameSite))

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
			switch responseMode {
			case constants.ResponseModeQuery:
				if strings.Contains(redirectURI, "?") {
					redirectURI = redirectURI + "&" + params
				} else {
					redirectURI = redirectURI + "?" + params
				}
			case constants.ResponseModeFragment:
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
				AuthTime:    claims.EffectiveAuthTime(),
				ClientID:    clientID,
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

			cookie.SetSession(gc, authToken.FingerPrintHash, h.Config.AppCookieSecure, cookie.ParseSameSite(h.Config.AppCookieSameSite))

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
				AuthTime:    claims.EffectiveAuthTime(),
				ClientID:    clientID,
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

			cookie.SetSession(gc, authToken.FingerPrintHash, h.Config.AppCookieSecure, cookie.ParseSameSite(h.Config.AppCookieSameSite))

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

// redirectURIMatches reports whether a presented redirect_uri satisfies a
// registered one.
//
// Exact string comparison, with one carve-out: [RFC 8252 §7.3] requires an
// authorization server to allow a native app to specify ANY port on a loopback
// redirect, because the app binds an ephemeral port at run time and cannot know
// it at registration. Exact matching alone makes loopback redirects unusable —
// a client registering "http://127.0.0.1/callback" then arrives on
// "http://127.0.0.1:53119/callback" and is refused every time.
//
// The carve-out is as narrow as it can be:
//
//   - It applies only when BOTH the registered and the presented URI are
//     loopback. A registered https://app.example.com/cb never matches anything
//     but itself, so nothing about third-party redirect validation changes.
//   - Only the port is ignored. Scheme, host, path and query must still match
//     exactly, so it cannot be used to reach a different path on the same host.
//   - The host is compared literally: a registration for "localhost" does not
//     match "127.0.0.1". They are different names and RFC 8252 §8.3 discourages
//     the former; a client that wants both registers both.
//
// This only ever widens what is accepted, and only for redirects that terminate
// on the user's own machine. The residual risk is the one RFC 8252 §8.3 and the
// MCP authorization spec both name — a local process racing for the port — which
// no server-side URI check can address and which is mitigated by displaying the
// redirect host at consent.
//
// [RFC 8252 §7.3]: https://datatracker.ietf.org/doc/html/rfc8252#section-7.3
func redirectURIMatches(registered, presented string) bool {
	if registered == presented {
		return true
	}
	r, err := url.Parse(registered)
	if err != nil {
		return false
	}
	p, err := url.Parse(presented)
	if err != nil {
		return false
	}
	if !isLoopbackHost(r.Hostname()) || !isLoopbackHost(p.Hostname()) {
		return false
	}
	// Fragment and userinfo are rejected outright rather than compared, because
	// comparing only scheme/host/path/query would silently accept URIs the exact
	// match rejected — everything the comparison omits becomes a free field.
	//
	// A fragment is the worse of the two. RFC 6749 §3.1.2 forbids one on the
	// redirection endpoint, and the response is appended by string
	// concatenation: a presented "http://127.0.0.1:9/cb#x" carries no "?", so
	// the result is ".../cb#x?code=…&state=…" and the entire authorization
	// response lands inside the fragment. The app's loopback listener then
	// receives a request with no code, no state and no error, and the login
	// hangs forever instead of failing cleanly.
	//
	// Userinfo is the phishing shape: "http://evil.com@127.0.0.1/callback" reads
	// as evil.com to a human skimming a consent screen while resolving to
	// loopback.
	if r.Fragment != "" || p.Fragment != "" || r.User != nil || p.User != nil {
		return false
	}
	return r.Scheme == p.Scheme &&
		r.Hostname() == p.Hostname() &&
		r.Path == p.Path &&
		r.RawQuery == p.RawQuery
}

// isLoopbackHost reports whether a hostname names the local machine. "localhost"
// is included because RFC 8252 §7.3's port rule is written for the IP literals,
// while real native clients — Claude Code among them — declare the name form and
// expect the same treatment.
func isLoopbackHost(host string) bool {
	switch host {
	case "127.0.0.1", "::1", "localhost":
		return true
	default:
		return false
	}
}

// isValidResourceIndicator enforces RFC 8707 §2 on a resource indicator: it
// MUST be an absolute URI (scheme + hierarchical part) and MUST NOT contain a
// fragment component. A relative reference, an opaque non-URI string, or any
// value carrying a "#" is rejected. Callers must pass a non-empty value.
func isValidResourceIndicator(resource string) bool {
	// A fragment delimiter is forbidden outright — check the raw string so a
	// trailing "#" (empty fragment) is also rejected, not just a populated one.
	if strings.Contains(resource, "#") {
		return false
	}
	u, err := url.Parse(resource)
	if err != nil {
		return false
	}
	// Absolute URI: has a scheme and is not just an opaque scheme:string. Require
	// a scheme and either a host or a rooted path so bare "mailto:" style opaque
	// forms and relative references are rejected.
	if !u.IsAbs() {
		return false
	}
	if u.Host == "" && !strings.HasPrefix(u.Path, "/") {
		return false
	}
	return true
}

// responseTypeHasBareToken reports whether the (already canonicalized,
// space-separated) response_type includes the exact component "token" — i.e.
// the flow delivers a bearer access token into the front channel (URL
// fragment). It splits on whitespace and compares each component exactly, so
// "id_token" (which contains "token" only as a substring) is NOT matched.
// Matches: "token", "code token", "id_token token", "code id_token token".
func responseTypeHasBareToken(responseType string) bool {
	for _, c := range strings.Fields(responseType) {
		if c == constants.ResponseTypeToken {
			return true
		}
	}
	return false
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
