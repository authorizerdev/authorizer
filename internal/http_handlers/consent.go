package http_handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/clientmetadata"
	"github.com/authorizerdev/authorizer/internal/metrics"
)

// Pending consents are held in the state store, which applies its own expiry —
// 10 minutes on Redis (stateTTL), 600 seconds read-side on the DB store. That is
// the right order of magnitude here: long enough for a human to read the page,
// short enough that an abandoned tab is not a standing authorization waiting to
// be submitted later. There is no per-key TTL on the SetState interface, so this
// is inherited rather than chosen; if that ever needs to differ for consent, the
// interface has to grow a TTL argument rather than this file working around it.

// pendingConsent is the authorization request held while the user decides.
//
// It exists because consent splits one logical authorization into two HTTP
// requests, and everything the second request needs must survive the gap
// without being re-derived from parameters the browser could have altered in
// between. Only the consent_id travels through the page; every value below is
// read back from the store, so a tampered form cannot widen scope, redirect
// elsewhere, or swap the client.
type pendingConsent struct {
	ClientID    string `json:"client_id"`
	ClientName  string `json:"client_name"`
	RedirectURI string `json:"redirect_uri"`
	// Scopes is rendered on the page. It is NOT read back on submit — the
	// resumed request carries scope in Query, which is the value actually
	// enforced. Kept so the stored record fully describes what was shown.
	Scopes []string `json:"scopes"`
	// Query is the original /authorize parameter set, re-encoded and replayed on
	// approval so the resumed request carries exactly what was consented to.
	//
	// Built from the parsed FORM, not from URL.RawQuery: /authorize is registered
	// for POST as well as GET (RFC 6749 §3.1 / OIDC Core §3.1.2.1 permit it), and
	// a POST carries its parameters in the body. Storing only the query string
	// left Query empty for those, so approval replayed a parameterless request
	// and the user was dropped on "response_type is required" after having
	// approved — while the client waited on a callback that never came.
	Query string `json:"query"`
	// UserID pins the consent to the session that saw the page. A consent
	// approved in one account must never mint a code for another.
	UserID string `json:"user_id"`
}

// renderConsent stores the pending authorization and shows the consent page.
//
// Consent is required here and NOT for pre-registered clients, and the asymmetry
// is the whole point. A registered client was vouched for by an operator who
// entered its redirect URIs by hand. A Client ID Metadata Document client
// asserted its own identity: it chose its `client_name`, and anyone who can host
// a JSON file can claim any name. The only fact about it this server has
// verified is the redirect host — which is precisely what the page leads with.
//
// The MCP authorization spec requires the authorization server to display the
// redirect URI hostname and to warn about loopback-only clients, because a local
// impostor can bind the same port and present the legitimate client's metadata.
// See internal/clientmetadata for why CIMD is the registration mechanism.
func (h *httpProvider) renderConsent(gc *gin.Context, doc *clientmetadata.Document, redirectURI, userID, userEmail string, scopes []string) {
	log := h.Log.With().Str("func", "renderConsent").Logger()

	consentID := uuid.NewString()
	payload, err := json.Marshal(pendingConsent{
		ClientID:    doc.ClientID,
		ClientName:  doc.ClientName,
		RedirectURI: redirectURI,
		Scopes:      scopes,
		Query:       originalParams(gc).Encode(),
		UserID:      userID,
	})
	if err != nil {
		log.Debug().Err(err).Msg("failed to encode pending consent")
		gc.JSON(http.StatusInternalServerError, gin.H{
			"error":             "server_error",
			"error_description": "could not start the consent flow",
		})
		return
	}
	if err := h.MemoryStoreProvider.SetState(consentKey(consentID), string(payload)); err != nil {
		log.Debug().Err(err).Msg("failed to store pending consent")
		gc.JSON(http.StatusInternalServerError, gin.H{
			"error":             "server_error",
			"error_description": "could not start the consent flow",
		})
		return
	}

	redirectHost := redirectURI
	if u, err := url.Parse(redirectURI); err == nil && u.Host != "" {
		redirectHost = u.Host
	}

	metrics.RecordSecurityEvent("cimd_consent_shown", "authorize")
	// The page carries a single-use consent_id and names the signed-in user, so
	// it must not sit in a shared or back-button cache: a cached copy would show
	// one user's email to the next person on the machine, and re-submitting a
	// stale page is a guaranteed "expired or already used" dead end.
	gc.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	gc.Header("Pragma", "no-cache")
	gc.HTML(http.StatusOK, "consent.tmpl", gin.H{
		"client_name": doc.ClientName,
		"client_id":   doc.ClientID,
		// The host, not the full URI: it is the part that decides where the
		// authorization code actually lands, and the part a person can judge.
		"redirect_host":     redirectHost,
		"user_email":        userEmail,
		"scopes":            scopes,
		"loopback_only":     doc.IsLoopbackOnly(),
		"consent_id":        consentID,
		"organization_name": h.Config.OrganizationName,
	})
}

// ConsentHandler processes the approve/deny decision.
//
// On approval it replays the ORIGINAL /authorize query rather than rebuilding
// one from form fields. That is deliberate: rebuilding would make every
// parameter — scope, redirect_uri, code_challenge, resource — attacker-editable
// between the page render and the submit, so a user could be shown one request
// and made to approve another.
func (h *httpProvider) ConsentHandler() gin.HandlerFunc {
	return func(gc *gin.Context) {
		log := h.Log.With().Str("func", "ConsentHandler").Logger()

		consentID := strings.TrimSpace(gc.PostForm("consent_id"))
		if consentID == "" {
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "missing consent_id",
			})
			return
		}

		// Atomic: a read-then-delete lets two concurrent submissions of the same
		// consent_id both pass the single-use check, and this repo already ships
		// GetAndRemoveState precisely because "returning state on the strength of
		// the read alone would hand the same code to every racer".
		raw, err := h.MemoryStoreProvider.GetAndRemoveState(consentKey(consentID))
		if err != nil || raw == "" {
			// Expired, already used, or never existed — all indistinguishable to
			// the caller on purpose, so this is not an oracle for guessing ids.
			log.Debug().Msg("consent rejected: no pending request for this consent_id")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "this consent request has expired or was already used",
			})
			return
		}
		var pending pendingConsent
		if err := json.Unmarshal([]byte(raw), &pending); err != nil {
			log.Debug().Err(err).Msg("consent rejected: corrupt pending state")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "this consent request is no longer valid",
			})
			return
		}

		// The approving session must be the one that was shown the page.
		// Without this, a consent page rendered for one user could be submitted
		// in another user's browser and mint a code against their account.
		tokenData, err := h.TokenProvider.GetUserIDFromSessionOrAccessToken(gc)
		if err != nil || tokenData == nil || tokenData.UserID != pending.UserID {
			metrics.RecordSecurityEvent("cimd_consent_session_mismatch", "authorize")
			log.Warn().Msg("consent rejected: submitted by a different session than it was issued to")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "access_denied",
				"error_description": "please sign in and try again",
			})
			return
		}

		if gc.PostForm("action") != "approve" {
			// RFC 6749 §4.1.2.1: a refusal is reported to the client at its
			// registered redirect_uri, not rendered here — the client is waiting
			// on that callback and would otherwise hang.
			metrics.RecordSecurityEvent("cimd_consent_denied", "authorize")
			// redirectErrorToRP, not a hand-rolled query-string redirect: the
			// original request may have asked for fragment, form_post or
			// web_message, and delivering a query-string 302 to a client that
			// asked for form_post leaves it waiting on a response it will never
			// parse. The helper is what every other /authorize error path uses.
			orig, _ := url.ParseQuery(pending.Query)
			redirectErrorToRP(gc, orig.Get("response_mode"), pending.RedirectURI,
				orig.Get("state"), "access_denied",
				"the user declined to authorize this application")
			return
		}

		// Approved. Record the grant server-side and REDIRECT the browser back to
		// /authorize with the original query, rather than invoking the handler
		// in-process.
		//
		// Calling it directly does not work: gin caches parsed query parameters
		// on the Context at first access, so rewriting Request.URL.RawQuery
		// afterwards leaves the handler reading the POST's (empty) query and
		// failing with "response_type is required". Rewriting gin's internals to
		// force a re-parse would be a worse dependency than a redirect.
		//
		// A redirect is also the more honest shape: it is exactly what the client
		// would see from any other authorization server, and it re-enters
		// /authorize through the front door with every middleware applied.
		metrics.RecordSecurityEvent("cimd_consent_approved", "authorize")
		if err := h.MemoryStoreProvider.SetState(
			consentGrantKey(pending.UserID, pending.ClientID, pending.Query), "1"); err != nil {
			log.Debug().Err(err).Msg("failed to record consent grant")
			gc.JSON(http.StatusInternalServerError, gin.H{
				"error":             "server_error",
				"error_description": "could not record the consent decision",
			})
			return
		}
		gc.Redirect(http.StatusFound, "/authorize?"+pending.Query)
	}
}

// consentGrantKey names the single-use marker that tells the authorize handler
// this user has consented to this client FOR THIS EXACT REQUEST.
//
// The request is part of the key, not just the user and client. Keying on
// (user, client) alone meant a grant that was never redeemed — the tab closed,
// the browser went back, the network dropped — sat in the store for its whole
// TTL and would then satisfy ANY later /authorize for that pair: a different
// scope, a different redirect_uri from the document's list, a different PKCE
// challenge. The user would have been shown one request and a materially
// different one would execute, which is exactly what storing the full parameter
// set in pendingConsent exists to prevent. Hashing it in closes that.
//
// SHA-256 of the encoded parameters rather than the parameters themselves:
// the key goes into a shared store, and a redirect_uri or login_hint in a key is
// needless exposure.
func consentGrantKey(userID, clientID, query string) string {
	sum := sha256.Sum256([]byte(query))
	return "cimd_consent_granted:" + userID + ":" + clientID + ":" + hex.EncodeToString(sum[:])
}

// originalParams returns the /authorize parameters as sent, from whichever place
// they arrived — query string for GET, body for POST. Mirrors how the authorize
// handler itself reads them (gc.Request.FormValue).
func originalParams(gc *gin.Context) url.Values {
	_ = gc.Request.ParseForm()
	out := url.Values{}
	for k, v := range gc.Request.Form {
		if len(v) > 0 {
			out.Set(k, v[0])
		}
	}
	return out
}

func consentKey(id string) string { return "cimd_consent:" + id }
