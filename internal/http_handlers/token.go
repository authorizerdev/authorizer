package http_handlers

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
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
	"github.com/authorizerdev/authorizer/internal/service/clientauth"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
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
	// Scope is the space-delimited OAuth2 scope parameter (RFC 6749 §3.3),
	// used by the client_credentials grant to request a subset of the service
	// account's allowed scopes.
	Scope string `form:"scope" json:"scope"`
	// ClientAssertion / ClientAssertionType carry the RFC 7523 JWT-bearer client
	// credential — the secretless workload-identity path (K8s SA tokens etc.).
	ClientAssertion     string `form:"client_assertion" json:"client_assertion"`
	ClientAssertionType string `form:"client_assertion_type" json:"client_assertion_type"`
	// RFC 8693 token-exchange parameters. SubjectToken carries the authority
	// being exercised (the user's token); ActorToken carries the actor (the
	// agent's token) — its presence selects the delegation profile. The `resource`
	// parameter (RFC 8707) is read separately via PostFormArray so a repeated
	// value can be rejected.
	SubjectToken     string `form:"subject_token" json:"subject_token"`
	SubjectTokenType string `form:"subject_token_type" json:"subject_token_type"`
	ActorToken       string `form:"actor_token" json:"actor_token"`
	ActorTokenType   string `form:"actor_token_type" json:"actor_token_type"`
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
		grantType := strings.TrimSpace(reqBody.GrantType)
		refreshToken := strings.TrimSpace(reqBody.RefreshToken)
		bodyClientSecret := strings.TrimSpace(reqBody.ClientSecret)
		scopeParam := strings.TrimSpace(reqBody.Scope)

		if grantType == "" {
			grantType = "authorization_code"
		}

		isRefreshTokenGrant := grantType == "refresh_token"
		isAuthorizationCodeGrant := grantType == "authorization_code"
		isClientCredentialsGrant := grantType == constants.GrantTypeClientCredentials
		isTokenExchangeGrant := grantType == constants.GrantTypeTokenExchange

		if !isRefreshTokenGrant && !isAuthorizationCodeGrant && !isClientCredentialsGrant && !isTokenExchangeGrant {
			log.Debug().Str("grant_type", grantType).Msg("Invalid grant type")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "unsupported_grant_type",
				"error_description": "grant_type is not supported",
			})
			return
		}

		// Authenticate the client through the shared resolver (RFC 6749 §2.3):
		// it extracts the credential from client_secret_basic (Authorization
		// header) or client_secret_post (body), rejects presenting more than one
		// method, looks the client up by its public client_id, and verifies the
		// secret. client_credentials always requires a secret; authorization_code
		// / refresh_token treat a missing secret as "no secret presented" and let
		// the PKCE checks below gate the request. secretPresented drives the
		// no-PKCE "secret required" rule further down.
		basicClientID, basicClientSecret, hasBasicAuth := gc.Request.BasicAuth()
		secretPresented := bodyClientSecret != "" || (hasBasicAuth && basicClientSecret != "")
		clientAssertion := strings.TrimSpace(reqBody.ClientAssertion)
		clientAssertionType := strings.TrimSpace(reqBody.ClientAssertionType)
		resolvedClient, authErr := h.clientAuthProvider.ResolveClient(gc, clientauth.ResolveParams{
			BodyClientID:  strings.TrimSpace(reqBody.ClientID),
			BodySecret:    bodyClientSecret,
			BasicClientID: basicClientID,
			BasicSecret:   basicClientSecret,
			HasBasicAuth:  hasBasicAuth,
			// RFC 7523 client_assertion: when present the resolver authenticates the
			// client by verifying the JWT against a registered TrustedIssuer instead
			// of a secret. Presenting it alongside a secret is rejected as multiple
			// auth methods (RFC 6749 §2.3).
			ClientAssertion:     clientAssertion,
			ClientAssertionType: clientAssertionType,
			// client_credentials always requires a secret; authorization_code
			// verifies a presented secret (PKCE gates a secret-less request);
			// refresh_token ignores the secret and authenticates the client_id
			// only — preserving the pre-registry behavior of each grant.
			RequireSecret:         isClientCredentialsGrant || isTokenExchangeGrant,
			VerifyPresentedSecret: isAuthorizationCodeGrant,
			// client_credentials and token-exchange are machine-only: the calling
			// client is the agent's service account (design §4.1 / §3). The resolver
			// rejects a non-service_account client before verifying the secret, so an
			// interactive client_id cannot confirm a guessed secret on these grants.
			RequireServiceAccountKind: isClientCredentialsGrant || isTokenExchangeGrant,
		})
		if authErr != nil {
			h.respondClientAuthError(gc, authErr, resolvedClient, isClientCredentialsGrant)
			return
		}

		// RFC 6749 §4.4 client_credentials: machine identity. The resolver has
		// already authenticated the service_account; issue its scoped token here.
		if isClientCredentialsGrant {
			h.handleClientCredentialsGrant(gc, resolvedClient, scopeParam)
			return
		}

		// RFC 8693 token-exchange (delegation): the resolver has already
		// authenticated the calling agent's service_account. Mint the delegated,
		// attenuated, resource-bound token carrying the nested `act` chain.
		if isTokenExchangeGrant {
			h.handleTokenExchangeGrant(gc, resolvedClient, &reqBody, scopeParam)
			return
		}

		var userID string
		var roles, scope []string
		loginMethod := ""
		sessionKey := ""
		oidcNonce := ""      // OIDC nonce from the original /authorize request
		authTime := int64(0) // End-User's actual last authentication (OIDC Core §2 auth_time); 0 = unknown, CreateIDToken falls back to time.Now()

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
			// The secret itself was already verified by the client-auth resolver
			// above; here we only enforce that one was presented.
			if storedChallenge == "" && codeVerifier == "" {
				// No PKCE — a client secret is required.
				if !secretPresented {
					gc.JSON(http.StatusBadRequest, gin.H{
						"error":             "invalid_request",
						"error_description": "Either code_verifier or client_secret is required",
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
			authTime = claims.EffectiveAuthTime()

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

			// auth_time survives token refresh unchanged — a refresh must
			// not reset "when the user last actually authenticated".
			// JWT numeric claims decode as float64.
			if at, ok := claims["auth_time"].(float64); ok {
				authTime = int64(at)
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
		// Defense-in-depth for deprovisioning: a revoked user (SCIM active:false,
		// account deactivation) must never renew a token, even if a session-store
		// delete was missed on some instance. RevokedTimestamp is the reliable
		// signal — it is explicitly stamped on revocation and nil otherwise, so it
		// is correct across every provider. IsActive is NOT used here: on the NoSQL
		// providers a normally-signed-up user is stored with is_active=false (no
		// GORM column default), so gating on it would reject legitimate refreshes.
		if user.RevokedTimestamp != nil {
			log.Debug().Str("user_id", userID).Msg("refresh rejected: user revoked")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "unauthorized",
				"error_description": "User is not active",
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
			AuthTime:    authTime,
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
			cookie.SetSession(gc, authToken.FingerPrintHash, h.Config.AppCookieSecure, cookie.ParseSameSite(h.Config.AppCookieSameSite))
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

// respondClientAuthError maps a clientauth resolver error to the RFC 6749 §5.2
// token-endpoint error response. Dual-method and missing-client_id map to
// invalid_request (400); everything else is invalid_client (401 when the client
// authenticated via HTTP Basic per §5.2, else 400). For a failed
// client_credentials attempt against a *resolved* client it also writes the
// token.client_credentials_failed audit event — mirroring the historical
// behavior, which audited only known-client failures (not unknown client_ids).
func (h *httpProvider) respondClientAuthError(gc *gin.Context, err error, resolved *schemas.Client, isClientCredentials bool) {
	log := h.Log.With().Str("func", "respondClientAuthError").Logger()

	switch {
	case errors.Is(err, clientauth.ErrMultipleAuthMethods):
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "Only one client authentication method may be used per request",
		})
		return
	case errors.Is(err, clientauth.ErrMissingClientID):
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "The client_id parameter is required",
		})
		return
	case errors.Is(err, clientauth.ErrUnsupportedAssertionType):
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "Unsupported client_assertion_type",
		})
		return
	case errors.Is(err, clientauth.ErrUnauthorizedClient):
		// Client is authenticated-or-not but simply not allowed to use this grant
		// (an interactive client on client_credentials). Returned before secret
		// verification, so this response is identical for any secret — no oracle.
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             "unauthorized_client",
			"error_description": "This client is not authorized to use this grant type",
		})
		return
	}

	// invalid_client (clientauth.ErrInvalidClient, or any unexpected error).
	metrics.RecordSecurityEvent("invalid_client", "token_endpoint")

	// Audit only a known-client failure on the client_credentials path — mirrors
	// the historical behavior (login.go audits bad_password but not user-not-
	// found). resolved.ID == "" is the synthesized reserved-client fallback,
	// which is not a service_account and must not be audited as one.
	if isClientCredentials && resolved != nil && resolved.ID != "" {
		h.AuditProvider.LogEvent(audit.Event{
			Action:       constants.AuditTokenClientCredentialsFailedEvent,
			ActorID:      resolved.ID,
			ActorType:    constants.AuditActorTypeServiceAccount,
			ResourceType: constants.AuditResourceTypeToken,
			ResourceID:   resolved.ID,
			Metadata:     constants.GrantTypeClientCredentials,
			IPAddress:    utils.GetIP(gc.Request),
			UserAgent:    utils.GetUserAgent(gc.Request),
		})
	}

	log.Debug().Err(err).Msg("client authentication failed")
	// Status selection reproduces the pre-registry behavior exactly:
	//   - HTTP Basic auth failure → 401 + WWW-Authenticate (RFC 6749 §5.2).
	//   - authorization_code with a wrong secret on a resolved (known) client →
	//     401 without WWW-Authenticate (the old confidential-client path).
	//   - everything else (unknown client_id via body, client_credentials
	//     non-Basic) → 400.
	_, _, hasBasicAuth := gc.Request.BasicAuth()
	status := http.StatusBadRequest
	if hasBasicAuth {
		gc.Header("WWW-Authenticate", "Basic realm=\"authorizer\"")
		status = http.StatusUnauthorized
	} else if !isClientCredentials && resolved != nil {
		status = http.StatusUnauthorized
	}
	gc.JSON(status, gin.H{
		"error":             "invalid_client",
		"error_description": "Client authentication failed",
	})
}

// handleClientCredentialsGrant implements the RFC 6749 §4.4 client_credentials
// grant. The client (a service_account) is already authenticated by the
// clientauth resolver; sa is that resolved, active client. This issues a
// stateful access token scoped to a subset of the account's allowed scopes. No
// id_token and no refresh_token are issued — machines re-authenticate on expiry
// (RFC 6749 §4.4.3).
func (h *httpProvider) handleClientCredentialsGrant(gc *gin.Context, sa *schemas.Client, scope string) {
	log := h.Log.With().Str("func", "handleClientCredentialsGrant").Logger()

	// Scope handling (RFC 6749 §3.3 / §5.2). An empty AllowedScopes is DENY-ALL
	// (schema § AllowedScopes comment) — reject rather than issue a scopeless
	// token. The service layer already forbids creating such accounts; this is
	// defense-in-depth.
	allowedScopes := sa.ParsedAllowedScopes()
	if len(allowedScopes) == 0 {
		log.Debug().Msg("service account has no authorized scopes")
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_scope",
			"error_description": "The service account has no authorized scopes",
		})
		return
	}
	allowedSet := make(map[string]struct{}, len(allowedScopes))
	for _, s := range allowedScopes {
		allowedSet[s] = struct{}{}
	}

	var grantedScopes []string
	requestedScopes := strings.Fields(scope)
	if len(requestedScopes) == 0 {
		// No scope requested — grant the full authorized set. This repo's spec
		// does not mandate a default; granting the full authorized set on an
		// omitted scope param is the common client_credentials convention.
		grantedScopes = allowedScopes
	} else {
		// Every requested scope MUST be authorized; reject the whole request
		// otherwise (RFC 6749 §5.2 invalid_scope) rather than silently
		// downgrading, which would hide a client misconfiguration.
		for _, rs := range requestedScopes {
			if _, ok := allowedSet[rs]; !ok {
				log.Debug().Str("scope", rs).Msg("requested scope exceeds allowed scopes")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_scope",
					"error_description": "The requested scope exceeds the scopes authorized for this service account",
				})
				return
			}
		}
		grantedScopes = requestedScopes
	}

	hostname := parsers.GetHost(gc)
	nonce := uuid.New().String()
	// Namespace the session key so it can never collide with a human user's
	// session key (a bare or login-method-prefixed UUID). ValidateAccessToken
	// reconstructs this exact key from the token's login_method + sub claims.
	sessionKey := constants.AuthRecipeMethodServiceAccount + ":" + sa.ID

	// ponytail: aud is the global ClientID, same as human tokens. RFC 8707
	// resource-bound audience binding (spec S8) is deliberately deferred to
	// the Phase 2 token-exchange work, not silently forgotten.
	authToken, err := h.TokenProvider.CreateMachineAuthToken(&token.AuthTokenConfig{
		ServiceAccountID: sa.ID,
		Scope:            grantedScopes,
		Nonce:            nonce,
		LoginMethod:      constants.AuthRecipeMethodServiceAccount,
		HostName:         hostname,
	})
	if err != nil {
		log.Debug().Err(err).Msg("failed to create machine access token")
		gc.JSON(http.StatusInternalServerError, gin.H{
			"error":             "server_error",
			"error_description": "Could not complete token issuance",
		})
		return
	}

	// Access tokens in this codebase are stateful: register in the memory store
	// or ValidateAccessToken (GraphQL context, gRPC interceptor, profile
	// endpoints) rejects a cryptographically-valid-but-unregistered token.
	if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+nonce, authToken.Token, authToken.ExpiresAt); err != nil {
		log.Debug().Err(err).Msg("failed to persist machine access token")
		gc.JSON(http.StatusServiceUnavailable, gin.H{
			"error":             "temporarily_unavailable",
			"error_description": "Could not complete token issuance",
		})
		return
	}

	expiresIn := authToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	metrics.RecordAuthEvent(metrics.EventTokenIssued, metrics.StatusSuccess)
	h.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditTokenClientCredentialsEvent,
		ActorID:      sa.ID,
		ActorType:    constants.AuditActorTypeServiceAccount,
		ResourceType: constants.AuditResourceTypeToken,
		ResourceID:   sa.ID,
		Metadata:     constants.GrantTypeClientCredentials,
		IPAddress:    utils.GetIP(gc.Request),
		UserAgent:    utils.GetUserAgent(gc.Request),
	})

	// RFC 6749 §5.1: no refresh_token and no id_token for client_credentials.
	gc.JSON(http.StatusOK, gin.H{
		"access_token": authToken.Token,
		"token_type":   "Bearer",
		"expires_in":   expiresIn,
		"scope":        strings.Join(grantedScopes, " "),
	})
}
