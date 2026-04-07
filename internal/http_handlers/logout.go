package http_handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

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
			if idTokenHint == "" || !h.isValidIDTokenHint(idTokenHint) {
				log.Debug().Bool("had_hint", idTokenHint != "").Msg("serving logout confirmation page")
				gc.Header("Cache-Control", "no-store")
				gc.HTML(http.StatusOK, logoutConfirmTemplate, gin.H{
					"redirect_uri":             gc.Query("redirect_uri"),
					"post_logout_redirect_uri": gc.Query("post_logout_redirect_uri"),
					"state":                    gc.Query("state"),
				})
				return
			}
			// Valid id_token_hint — fall through to the normal logout flow.
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
		// goroutine with a 5-second HTTP timeout so the user-facing
		// logout response is never blocked by a slow receiver.
		if strings.TrimSpace(h.Config.BackchannelLogoutURI) != "" {
			hostname := parsers.GetHost(gc)
			go func(uri, host, sub, sid string) {
				if err := h.TokenProvider.NotifyBackchannelLogout(context.Background(), uri, &token.BackchannelLogoutConfig{
					HostName:  host,
					Subject:   sub,
					SessionID: sid,
				}); err != nil {
					log.Debug().Err(err).Msg("backchannel logout notification failed")
				}
			}(h.Config.BackchannelLogoutURI, hostname, userID, sessionData.Nonce)
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

// isValidIDTokenHint verifies a logout id_token_hint by parsing the JWT
// against the server's own signing key. The token does not need to be
// unexpired (the OIDC spec explicitly allows expired ID tokens as logout
// hints) — only that the signature is valid and the token claims to have
// been issued by this server. This is enough to defeat the
// <img src="/logout"> CSRF vector because an attacker on a third-party
// page cannot synthesise a valid signature.
func (h *httpProvider) isValidIDTokenHint(idTokenHint string) bool {
	if idTokenHint == "" {
		return false
	}
	claims, err := h.TokenProvider.ParseJWTToken(idTokenHint)
	if err != nil || claims == nil {
		return false
	}
	// Sanity-check that this looks like an ID token (not, say, a refresh
	// token someone tried to slip through).
	if tt, ok := claims["token_type"].(string); ok && tt != "" && tt != "id_token" {
		return false
	}
	return true
}
