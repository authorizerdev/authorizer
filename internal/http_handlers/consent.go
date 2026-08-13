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

// consentClient is the minimum a self-asserted client must supply to be
// described on the consent page.
//
// It exists so the page does not depend on WHERE the client's claims came from.
// The two sources — a Client ID Metadata Document fetched from the client's own
// URL, and a row written by the RFC 7591 registration endpoint — differ in
// plumbing and not in trust: in both cases the name is chosen by the client and
// verified by nobody. Passing a *clientmetadata.Document here instead would have
// forced the registration path to fabricate one, which would have read as if a
// document had been fetched when none had.
type consentClient struct {
	ClientID   string
	ClientName string
	// RedirectURIs is the client's full registered list, used only to decide
	// whether the loopback warning applies. The URI actually being authorized is
	// passed separately and is what the page displays.
	RedirectURIs []string
}

// isLoopbackOnly reports whether every registered redirect URI is a loopback
// address.
//
// Such a client cannot be distinguished from a local impostor: any process on
// the user's machine can bind a port and claim the same metadata document, or
// register itself under the same name. Both specs call this out and say the
// authorization server SHOULD warn about it, because it is not solvable
// server-side. The consent screen uses this to say so out loud.
//
// Shares isLoopbackHost with the RFC 8252 redirect matcher and the registration
// validator, so "is this loopback?" has exactly one answer in this codebase.
func (c consentClient) isLoopbackOnly() bool {
	if len(c.RedirectURIs) == 0 {
		return false
	}
	for _, raw := range c.RedirectURIs {
		u, err := url.Parse(raw)
		if err != nil || !isLoopbackHost(u.Hostname()) {
			return false
		}
	}
	return true
}

// renderConsent stores the pending authorization and shows the consent page.
//
// Consent is required for self-asserted clients and NOT for operator-registered
// ones, and the asymmetry is the whole point. A registered client was vouched
// for by an operator who entered its redirect URIs by hand. A self-asserted
// client — one presenting a Client ID Metadata Document, or one that registered
// itself through RFC 7591 — chose its own `client_name`, and anyone who can host
// a JSON file or POST to /oauth/register can claim any name. The only fact about
// it this server has verified is the redirect host, which is precisely what the
// page leads with.
//
// The MCP authorization spec requires the authorization server to display the
// redirect URI hostname and to warn about loopback-only clients, because a local
// impostor can bind the same port and present the legitimate client's metadata.
// RFC 7591 §5 asks for the same warning for dynamically registered clients.
func (h *httpProvider) renderConsent(gc *gin.Context, client consentClient, redirectURI, userID, userEmail string, scopes []string) {
	log := h.Log.With().Str("func", "renderConsent").Logger()

	consentID := uuid.NewString()
	payload, err := json.Marshal(pendingConsent{
		ClientID:    client.ClientID,
		ClientName:  client.ClientName,
		RedirectURI: redirectURI,
		Scopes:      scopes,
		Query:       originalParams(gc).Encode(),
		UserID:      userID,
	})
	if err != nil {
		log.Debug().Err(err).Msg("failed to encode pending consent")
		h.consentError(gc, http.StatusInternalServerError,
			"Something went wrong",
			"We could not start the approval for this application.",
			"Please return to the application and try connecting again.")
		return
	}
	if err := h.MemoryStoreProvider.SetState(consentKey(consentID), string(payload)); err != nil {
		log.Debug().Err(err).Msg("failed to store pending consent")
		h.consentError(gc, http.StatusInternalServerError,
			"Something went wrong",
			"We could not start the approval for this application.",
			"Please return to the application and try connecting again.")
		return
	}

	redirectHost := redirectURI
	if u, err := url.Parse(redirectURI); err == nil && u.Host != "" {
		redirectHost = u.Host
	}

	// The consent form's submission legitimately ends at the client's
	// redirect_uri, so the CSP must permit that destination — otherwise the
	// button does nothing.
	//
	// A browser enforces `form-action` across the ENTIRE redirect chain of a
	// form submission, not just its immediate target. Approving POSTs to
	// /authorize/consent → 302 /authorize → 302 the client's redirect_uri, so
	// the default `form-action 'self'` aborts the navigation at the last hop.
	// The user is left staring at the consent page with no error, assumes the
	// click missed, clicks again — and the second POST fails with "expired or
	// already used", because the first one already consumed the consent.
	//
	// This server already hit the same rule twice: setFormPostCSP relaxes
	// form-action for OIDC form_post, and samlIDPSSOCSP omits it because
	// "form-action 'self' would silently break every SAML IdP login". The
	// consent page is the third instance of the same shape.
	//
	// Scoped to the ONE origin being approved rather than the `form-action *`
	// that form_post uses: the redirect_uri has already been validated against
	// this client's registered list, so the exact destination is known here and
	// nothing wider needs allowing.
	setConsentCSP(gc, redirectURI)

	metrics.RecordSecurityEvent("cimd_consent_shown", "authorize")
	// The page carries a single-use consent_id and names the signed-in user, so
	// it must not sit in a shared or back-button cache: a cached copy would show
	// one user's email to the next person on the machine, and re-submitting a
	// stale page is a guaranteed "expired or already used" dead end.
	gc.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	gc.Header("Pragma", "no-cache")
	gc.HTML(http.StatusOK, "consent.tmpl", gin.H{
		"client_name": client.ClientName,
		"client_id":   client.ClientID,
		// The host, not the full URI: it is the part that decides where the
		// authorization code actually lands, and the part a person can judge.
		"redirect_host":     redirectHost,
		"user_email":        userEmail,
		"scopes":            scopes,
		"loopback_only":     client.isLoopbackOnly(),
		"consent_id":        consentID,
		"organization_name": h.Config.OrganizationName,
		"organization_logo": h.Config.OrganizationLogo,
	})
}

// consentError renders a consent failure as a page rather than a JSON body.
//
// Everything that reaches this handler is a browser following a form the server
// itself rendered, so the response is read by a person, not a program. It used
// to return raw JSON — which is what a user actually saw after clicking
// "Allow access" twice: `{"error":"invalid_request","error_description":"this
// consent request has expired or was already used"}`, with no indication of
// what to do next.
//
// The OAuth error code stays in the logs and metrics; it is not something the
// person in front of the screen can act on. What they can act on is the hint.
func (h *httpProvider) consentError(gc *gin.Context, status int, title, message, hint string) {
	gc.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	gc.Header("Pragma", "no-cache")
	gc.HTML(status, "consent_error.tmpl", gin.H{
		"title":             title,
		"message":           message,
		"hint":              hint,
		"organization_name": h.Config.OrganizationName,
		"organization_logo": h.Config.OrganizationLogo,
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
			h.consentError(gc, http.StatusBadRequest,
				"This approval link is not valid",
				"The request is missing the identifier that ties it to an approval.",
				"Please return to the application and try connecting again.")
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
			h.consentError(gc, http.StatusBadRequest,
				"This approval has already been used",
				"Approvals can only be used once, and they expire after a few minutes.",
				"Nothing has been shared. Return to the application and connect again to get a fresh approval.")
			return
		}
		var pending pendingConsent
		if err := json.Unmarshal([]byte(raw), &pending); err != nil {
			log.Debug().Err(err).Msg("consent rejected: corrupt pending state")
			h.consentError(gc, http.StatusBadRequest,
				"This approval is no longer valid",
				"The saved approval could not be read.",
				"Please return to the application and try connecting again.")
			return
		}

		// The approving session must be the one that was shown the page.
		// Without this, a consent page rendered for one user could be submitted
		// in another user's browser and mint a code against their account.
		tokenData, err := h.TokenProvider.GetUserIDFromSessionOrAccessToken(gc)
		if err != nil || tokenData == nil || tokenData.UserID != pending.UserID {
			metrics.RecordSecurityEvent("cimd_consent_session_mismatch", "authorize")
			log.Warn().Msg("consent rejected: submitted by a different session than it was issued to")
			h.consentError(gc, http.StatusUnauthorized,
				"Please sign in again",
				"This approval was created for a different sign-in session, so it was not applied.",
				"Return to the application and connect again to approve as the account you are signed in with.")
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
			h.consentError(gc, http.StatusInternalServerError,
				"Something went wrong",
				"Your approval could not be recorded.",
				"Please return to the application and try connecting again.")
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

// setConsentCSP writes the consent page's Content-Security-Policy, widening
// form-action to include the redirect_uri's origin and nothing else.
//
// It mirrors the default policy (internal/http_handlers/security_headers.go)
// rather than deriving from it, for the same reason setFormPostCSP does: the
// header is replaced wholesale, so it has to be complete. Keep the two in step.
//
// The origin is appended, never substituted — 'self' must stay, because the
// form posts to /authorize/consent on this origin before it redirects anywhere.
func setConsentCSP(gc *gin.Context, redirectURI string) {
	formAction := "'self'"
	// A relative redirect (the "/app" default, reachable when a metadata-document
	// client omits redirect_uri) has no origin to add and needs none.
	if u, err := url.Parse(redirectURI); err == nil && u.Scheme != "" && u.Host != "" {
		formAction += " " + u.Scheme + "://" + u.Host
	}
	gc.Writer.Header().Set("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self'; "+
			"style-src 'self' 'unsafe-inline'; "+
			// `https:` matches the default policy, and it is load-bearing here:
			// --organization-logo is usually hosted somewhere else entirely, so
			// restricting to 'self' renders the operator's own branding as a
			// broken image on the one page where branding is the trust signal.
			"img-src 'self' data: https:; "+
			"font-src 'self' data:; "+
			"connect-src 'self'; "+
			"frame-ancestors 'none'; "+
			"base-uri 'self'; "+
			"form-action "+formAction+";")
}
