package http_handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
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

// logoutConfirmTmpl is the OIDC RP-initiated logout confirmation page.
// Served on GET /logout when no id_token_hint is supplied; clicking the
// button POSTs back to /logout to actually terminate the session. This
// defeats the <img src="/logout"> CSRF vector while remaining compliant
// with the OIDC RP-Initiated Logout 1.0 spec, which requires GET to be
// supported on the end_session_endpoint.
var logoutConfirmTmpl = template.Must(template.New("logoutConfirm").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>Sign out</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>body{font-family:system-ui,sans-serif;max-width:32rem;margin:4rem auto;padding:0 1rem;color:#222}button{font-size:1rem;padding:.6rem 1.2rem;cursor:pointer}</style>
</head><body>
<h1>Sign out?</h1>
<p>Click below to confirm signing out of your account.</p>
<form method="POST" action="/logout">
<input type="hidden" name="redirect_uri" value="{{.RedirectURI}}">
<input type="hidden" name="post_logout_redirect_uri" value="{{.PostLogoutRedirectURI}}">
<input type="hidden" name="state" value="{{.State}}">
<button type="submit">Sign out</button>
</form>
</body></html>`))

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
				gc.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
				gc.Writer.Header().Set("Cache-Control", "no-store")
				gc.Status(http.StatusOK)
				_ = logoutConfirmTmpl.Execute(gc.Writer, map[string]string{
					"RedirectURI":           gc.Query("redirect_uri"),
					"PostLogoutRedirectURI": gc.Query("post_logout_redirect_uri"),
					"State":                 gc.Query("state"),
				})
				return
			}
			// Valid id_token_hint — fall through to the normal logout flow.
		}

		redirectURL := strings.TrimSpace(gc.Query("redirect_uri"))
		// Allow redirect_uri to come from POST form body too.
		if redirectURL == "" {
			redirectURL = strings.TrimSpace(gc.PostForm("redirect_uri"))
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

		if redirectURL != "" {
			hostname := parsers.GetHost(gc)
			if !validators.IsValidRedirectURI(redirectURL, h.Config.AllowedOrigins, hostname) {
				log.Debug().Msg("Invalid redirect URI")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid redirect uri",
				})
				return
			}
			gc.Redirect(http.StatusFound, redirectURL)
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
