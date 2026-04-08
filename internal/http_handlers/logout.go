package http_handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// backchannelLogoutGoroutineTimeout bounds the fire-and-forget BCL
// notification spawned from the logout handler. The token provider
// itself enforces a separate per-HTTP timeout; this outer bound exists
// so the goroutine cannot leak indefinitely if the inner timeout is
// ever lifted or extended.
const backchannelLogoutGoroutineTimeout = 10 * time.Second

// logoutConfirmTemplate is the gin template name for the OIDC RP-initiated
// logout confirmation page. The template lives at
// web/templates/logout_confirm.tmpl and is loaded by NewRouter() via
// LoadHTMLGlob, the same way the other authorize_* templates are.
const logoutConfirmTemplate = "logout_confirm.tmpl"

// Handler to logout user
func (h *httpProvider) LogoutHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "LogoutHandler").Logger()
	return func(gc *gin.Context) {
		// OIDC RP-initiated logout uses GET on the end_session_endpoint.
		// Without protection, an attacker can place <img src="/logout">
		// on a page they control and the victim's browser will silently
		// terminate their session via the cookie. Defence-in-depth:
		//
		//   - GET with id_token_hint  → proceed (the ID token proves the
		//     request originates from a real authenticated session — an
		//     <img> tag cannot forge an ID token)
		//   - GET without id_token_hint → render an HTML confirmation
		//     page; only the POST that follows actually deletes the
		//     session
		//   - POST → proceed unconditionally (existing behaviour; CSRF
		//     middleware enforces Origin/Referer for all POSTs)
		if gc.Request.Method == http.MethodGet {
			idTokenHint := strings.TrimSpace(gc.Query("id_token_hint"))
			hintBoundToSession := false
			if idTokenHint != "" {
				// Resolve the current session subject (if any) so we can
				// bind the hint to the actual logged-in user. An attacker
				// who obtains any valid ID token (browser history, leaked
				// log) must NOT be able to use it to log a different
				// victim out — that would turn /logout into a CSRF
				// primitive. We therefore require sub(hint) == sub(session).
				currentSubject := h.currentSessionSubject(gc)
				if currentSubject != "" {
					hintBoundToSession = h.isValidIDTokenHintForSubject(idTokenHint, currentSubject)
				}
			}
			if !hintBoundToSession {
				log.Debug().Bool("had_hint", idTokenHint != "").Msg("serving logout confirmation page")
				gc.Header("Cache-Control", "no-store")
				gc.HTML(http.StatusOK, logoutConfirmTemplate, gin.H{
					"redirect_uri":             gc.Query("redirect_uri"),
					"post_logout_redirect_uri": gc.Query("post_logout_redirect_uri"),
					"state":                    gc.Query("state"),
				})
				return
			}
			// Valid id_token_hint bound to the current session — fall
			// through to the normal logout flow.
		}

		// OIDC RP-Initiated Logout 1.0 §3 uses post_logout_redirect_uri.
		// Fall back to the legacy redirect_uri for backward compatibility.
		redirectURL := strings.TrimSpace(gc.Query("post_logout_redirect_uri"))
		if redirectURL == "" {
			redirectURL = strings.TrimSpace(gc.PostForm("post_logout_redirect_uri"))
		}
		if redirectURL == "" {
			redirectURL = strings.TrimSpace(gc.Query("redirect_uri"))
		}
		if redirectURL == "" {
			redirectURL = strings.TrimSpace(gc.PostForm("redirect_uri"))
		}

		// state, when present, MUST be echoed on the final redirect per
		// OIDC RP-Initiated Logout §3.
		state := strings.TrimSpace(gc.Query("state"))
		if state == "" {
			state = strings.TrimSpace(gc.PostForm("state"))
		}
		// get fingerprint hash
		fingerprintHash, err := cookie.GetSession(gc)
		if err != nil {
			log.Debug().Err(err).Msg("failed GetSession")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		decryptedFingerPrint, err := crypto.DecryptAES(h.ClientSecret, fingerprintHash)
		if err != nil {
			log.Debug().Err(err).Msg("failed to decrypt fingerprint")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		var sessionData token.SessionData
		err = json.Unmarshal([]byte(decryptedFingerPrint), &sessionData)
		if err != nil {
			log.Debug().Err(err).Msg("failed to unmarshal session data")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		userID := sessionData.Subject
		loginMethod := sessionData.LoginMethod
		sessionToken := userID
		if loginMethod != "" {
			sessionToken = loginMethod + ":" + userID
		}

		h.MemoryStoreProvider.DeleteUserSession(sessionToken, sessionData.Nonce)
		cookie.DeleteSession(gc, h.Config.AppCookieSecure)
		metrics.RecordAuthEvent(metrics.EventLogout, metrics.StatusSuccess)
		metrics.ActiveSessions.Dec()
		h.AuditProvider.LogEvent(audit.Event{
			Action:       constants.AuditSessionTerminatedEvent,
			ActorID:      userID,
			ActorType:    constants.AuditActorTypeUser,
			ResourceType: constants.AuditResourceTypeSession,
			ResourceID:   userID,
			IPAddress:    utils.GetIP(gc.Request),
			UserAgent:    utils.GetUserAgent(gc.Request),
		})

		// OIDC Back-Channel Logout 1.0: when configured, fire a signed
		// logout_token POST to the operator-supplied URI. Done in a
		// goroutine so the user-facing logout response is never blocked
		// by a slow receiver. The goroutine also bounds itself with an
		// outer context deadline so it cannot leak.
		if strings.TrimSpace(h.Config.BackchannelLogoutURI) != "" {
			hostname := parsers.GetHost(gc)
			// Capture a per-request logger value (zerolog.Logger is a
			// value type and safe to copy across goroutines).
			bclLog := log.With().Str("subsystem", "backchannel_logout").Logger()
			// Pass empty SessionID: the previous implementation passed
			// sessionData.Nonce, but the Nonce is the in-memory session
			// store key — leaking it to the relying party would expose
			// internal state. Branch 2 omits the sid claim when empty;
			// receivers fall back to sub-based session matching, which
			// is explicitly allowed by OIDC BCL 1.0 §2.4.
			go h.notifyBackchannelLogoutAsync(bclLog, h.Config.BackchannelLogoutURI, hostname, userID)
		}

		if redirectURL != "" {
			hostname := parsers.GetHost(gc)
			if !validators.IsValidRedirectURI(redirectURL, h.Config.AllowedOrigins, hostname) {
				log.Debug().Msg("Invalid redirect URI")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid redirect uri",
				})
				return
			}
			// Append state if supplied (OIDC RP-Initiated Logout §3).
			finalURL := redirectURL
			if state != "" {
				if strings.Contains(finalURL, "?") {
					finalURL = finalURL + "&state=" + url.QueryEscape(state)
				} else {
					finalURL = finalURL + "?state=" + url.QueryEscape(state)
				}
			}
			gc.Redirect(http.StatusFound, finalURL)
		} else {
			gc.JSON(http.StatusOK, gin.H{
				"message": "Logged out successfully",
			})
		}
	}
}

// notifyBackchannelLogoutAsync runs the OIDC Back-Channel Logout
// notification in its own goroutine with an outer deadline. The
// SessionID parameter is intentionally always empty — see the call
// site for the rationale.
func (h *httpProvider) notifyBackchannelLogoutAsync(log zerolog.Logger, uri, hostname, subject string) {
	ctx, cancel := context.WithTimeout(context.Background(), backchannelLogoutGoroutineTimeout)
	defer cancel()
	if err := h.TokenProvider.NotifyBackchannelLogout(ctx, uri, &token.BackchannelLogoutConfig{
		HostName: hostname,
		Subject:  subject,
		// SessionID is intentionally empty — the in-memory nonce MUST
		// NOT leak to relying parties as the sid claim. Branch 2's
		// NotifyBackchannelLogout omits the claim when empty.
		SessionID: "",
	}); err != nil {
		// Warn (not Debug) so operators can see misconfigured BCL
		// receivers without enabling verbose logging.
		log.Warn().Err(err).Msg("backchannel logout notification failed")
	}
}

// currentSessionSubject returns the subject of the currently
// authenticated browser session, or "" if there is no session. Used to
// bind id_token_hint to the session that is actually being terminated.
// Errors (no cookie, invalid cookie) are intentionally swallowed: the
// caller treats an empty subject the same as an unbound hint and
// renders the confirmation page.
func (h *httpProvider) currentSessionSubject(gc *gin.Context) string {
	fingerprintHash, err := cookie.GetSession(gc)
	if err != nil || fingerprintHash == "" {
		return ""
	}
	decryptedFingerPrint, err := crypto.DecryptAES(h.ClientSecret, fingerprintHash)
	if err != nil {
		return ""
	}
	var sessionData token.SessionData
	if err := json.Unmarshal([]byte(decryptedFingerPrint), &sessionData); err != nil {
		return ""
	}
	return sessionData.Subject
}

// isValidIDTokenHintForSubject verifies a logout id_token_hint by
// parsing the JWT against the server's own signing key AND requiring
// that its `sub` claim matches `expectedSubject` — the subject of the
// currently authenticated session. Without the sub binding the hint
// would be a logout-CSRF primitive: any valid ID token (browser
// history, leaked log) would suffice to log a different victim out.
//
// The token does NOT need to be unexpired: OIDC Core §3.1.2.1
// explicitly allows expired ID tokens as logout hints. We therefore
// parse with claim validation disabled but still enforce signature
// verification.
func (h *httpProvider) isValidIDTokenHintForSubject(idTokenHint, expectedSubject string) bool {
	if idTokenHint == "" || expectedSubject == "" {
		return false
	}
	claims, err := h.parseExpiredOrValidIDTokenHint(idTokenHint)
	if err != nil || claims == nil {
		return false
	}
	// Sanity-check token_type if present (not all flows set it).
	if tt, ok := claims["token_type"].(string); ok && tt != "" && tt != "id_token" {
		return false
	}
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return false
	}
	// Constant-time comparison: although `sub` is not strictly secret,
	// using subtle here removes any timing oracle that could let an
	// attacker probe valid subjects against an unknown session.
	if subtle.ConstantTimeCompare([]byte(sub), []byte(expectedSubject)) != 1 {
		return false
	}
	return true
}

// parseExpiredOrValidIDTokenHint parses an id_token_hint accepting
// expired tokens per OIDC Core §3.1.2.1. The signature MUST be valid:
// it is verified against the primary signing key first, then the
// optional secondary key (manual rotation window). Claim expiry checks
// are skipped. The function is deliberately self-contained — it does
// not delegate to token.Provider.ParseJWTToken because that helper
// enforces `exp`.
func (h *httpProvider) parseExpiredOrValidIDTokenHint(tokenString string) (jwt.MapClaims, error) {
	if tokenString == "" {
		return nil, errors.New("empty token")
	}
	claims, err := h.parseHintWithKey(tokenString, h.Config.JWTType, h.Config.JWTSecret, h.Config.JWTPublicKey)
	if err == nil {
		return claims, nil
	}
	if strings.TrimSpace(h.Config.JWTSecondaryType) != "" {
		secondaryClaims, secondaryErr := h.parseHintWithKey(tokenString,
			h.Config.JWTSecondaryType,
			h.Config.JWTSecondarySecret,
			h.Config.JWTSecondaryPublicKey,
		)
		if secondaryErr == nil {
			return secondaryClaims, nil
		}
	}
	return nil, err
}

// parseHintWithKey verifies a JWT signature against a single key
// (primary or secondary) without enforcing claim expiry. Mirrors the
// algorithm dispatch in token.parseJWTWithKey but stays local to this
// file per the worktree contract (do not modify token/jwt.go).
func (h *httpProvider) parseHintWithKey(tokenString, algo, secret, publicKey string) (jwt.MapClaims, error) {
	signingMethod := jwt.GetSigningMethod(algo)
	if signingMethod == nil {
		return nil, errors.New("unsupported signing method")
	}
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	var claims jwt.MapClaims
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != signingMethod.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		switch signingMethod {
		case jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512:
			return []byte(secret), nil
		case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
			return crypto.ParseRsaPublicKeyFromPemStr(publicKey)
		case jwt.SigningMethodES256, jwt.SigningMethodES384, jwt.SigningMethodES512:
			return crypto.ParseEcdsaPublicKeyFromPemStr(publicKey)
		default:
			return nil, errors.New("unsupported signing method")
		}
	}
	if _, err := parser.ParseWithClaims(tokenString, &claims, keyFunc); err != nil {
		return nil, err
	}
	return claims, nil
}
