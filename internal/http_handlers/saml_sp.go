package http_handlers

// Per-organization enterprise SAML 2.0 SSO — Authorizer acting as the Service
// Provider (SP). An org configures its upstream corporate IdP (Okta/Entra/ADFS)
// as an sso_saml TrustedIssuer row; its users log in through that IdP, Authorizer
// validates the signed assertion, JIT-provisions the user (namespaced by
// (org_id, IdP-entity-id, NameID)) and issues a normal Authorizer session.
//
// This is the SAML sibling of the OIDC broker in oauth_sso.go.
//
// SECURITY MODEL (design §4.4 / CR3) — enforced by the vetted crewjam/saml
// ServiceProvider.ParseXMLResponse plus the explicit checks below:
//   - Signature over the consumed assertion: ParseXMLResponse validates the
//     XML-DSIG on the SAME <Assertion> element it unmarshals and returns (or a
//     Response signature that covers it), defeating XML Signature Wrapping (XSW).
//     Signatures are validated ONLY against the org's configured IdP certificate.
//   - Unsigned assertions are rejected (signatureRequired unless the Response is
//     itself validly signed).
//   - Audience == this org's SP entity ID; Recipient == this org's ACS URL;
//     Destination == this org's ACS URL. The ServiceProvider is built per
//     org_slug, so an assertion minted for Org B cannot be consumed at Org A.
//   - NotBefore / NotOnOrAfter are validated with a bounded clock skew
//     (saml.MaxClockSkew / saml.MaxIssueDelay, the library defaults).
//   - Single-use AssertionID: consumed assertion IDs are cached in the shared
//     memory store until expiry; a replay is rejected. (crewjam does NOT dedupe.)
//   - InResponseTo: SP-initiated flows persist the AuthnRequest ID keyed by an
//     opaque RelayState; the ACS binds the response to that pending request.
//     IdP-initiated flow is disabled unless the connection opts in.
//   - RelayState is an opaque, single-use handle into the shared store — never a
//     raw redirect target. The final app redirect is validated against
//     AllowedOrigins (mirror oauth_sso.go).
//
// ponytail: AuthnRequests are emitted UNSIGNED (HTTP-Redirect binding). Request
// signing protects the IdP against forged requests, not the SP against forged
// assertions (all SP-side security is at the ACS), and Authorizer has no SP
// X.509 keypair. Upgrade path: set SignatureMethod + Key + Certificate on the
// ServiceProvider and advertise AuthnRequestsSigned in metadata.

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/crewjam/saml"
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

const (
	// samlFlowPrefix namespaces the pending SP-initiated AuthnRequest in the shared
	// store. SetState applies the store's short OAuth-state TTL, so the pending
	// request expires if the assertion never arrives.
	samlFlowPrefix = "saml_flow:"
	// samlAssertionPrefix namespaces the single-use AssertionID replay cache.
	samlAssertionPrefix = "saml_assertion:"
	// samlReplayFallbackTTL bounds the replay cache when an assertion carries no
	// usable expiry (defence in depth; crewjam already rejects such assertions).
	samlReplayFallbackTTL = int64(600)
)

// samlDefaultAttributeMapping maps Authorizer profile fields to the SAML
// attribute names most IdPs emit by default. Overridable per connection.
var samlDefaultAttributeMapping = map[string]string{
	"email":       "email",
	"given_name":  "firstName",
	"family_name": "lastName",
	"nickname":    "displayName",
	"picture":     "picture",
}

// samlFlowState is the per-request SP-initiated context, stored single-use in the
// shared store between login and ACS under an opaque RelayState. RequestID binds
// the returned assertion's InResponseTo to the AuthnRequest we issued.
type samlFlowState struct {
	RequestID   string `json:"request_id"`
	OrgSlug     string `json:"org_slug"`
	AppRedirect string `json:"app_redirect"`
	AppState    string `json:"app_state"`
}

// SAMLMetadataHandler serves this org's SP SAML metadata XML so the org admin can
// register Authorizer at their IdP (entity ID + ACS URL).
func (h *httpProvider) SAMLMetadataHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "SAMLMetadataHandler").Logger()
	return func(c *gin.Context) {
		slug := strings.TrimSpace(c.Param("org_slug"))
		conn, ok := h.resolveActiveSAMLConnection(c, slug, &log)
		if !ok {
			return
		}
		sp, err := buildSAMLServiceProvider(conn, parsers.GetHost(c), slug)
		if err != nil {
			log.Debug().Err(err).Msg("failed to build SP for metadata")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sso_config_error", "error_description": "connection misconfigured"})
			return
		}
		md := sp.Metadata()
		body, err := xml.MarshalIndent(md, "", "  ")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		out := append([]byte(xml.Header), body...)
		c.Data(http.StatusOK, "application/samlmetadata+xml", out)
	}
}

// SAMLLoginHandler starts an SP-initiated SAML login: it resolves the org's
// sso_saml connection, builds an AuthnRequest, persists its ID under an opaque
// RelayState, and redirects the browser to the IdP (HTTP-Redirect binding).
func (h *httpProvider) SAMLLoginHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "SAMLLoginHandler").Logger()
	return func(c *gin.Context) {
		slug := strings.TrimSpace(c.Param("org_slug"))
		appRedirect := strings.TrimSpace(c.Query("redirect_uri"))
		appState := strings.TrimSpace(c.Query("state"))
		hostname := parsers.GetHost(c)

		if appRedirect == "" || !validators.IsValidRedirectURI(appRedirect, h.Config.AllowedOrigins, hostname) {
			log.Debug().Msg("invalid or missing redirect_uri")
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "invalid redirect_uri"})
			return
		}
		if h.MemoryStoreProvider == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		conn, ok := h.resolveActiveSAMLConnection(c, slug, &log)
		if !ok {
			return
		}
		sp, err := buildSAMLServiceProvider(conn, hostname, slug)
		if err != nil {
			log.Debug().Err(err).Msg("failed to build SP for login")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sso_config_error", "error_description": "connection misconfigured"})
			return
		}

		authnReq, err := sp.MakeAuthenticationRequest(sp.GetSSOBindingLocation(saml.HTTPRedirectBinding), saml.HTTPRedirectBinding, saml.HTTPPostBinding)
		if err != nil {
			log.Debug().Err(err).Msg("failed to build AuthnRequest")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sso_config_error", "error_description": "could not build authentication request"})
			return
		}

		relayState, err := randURLString(32)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		flowJSON, err := json.Marshal(samlFlowState{
			RequestID:   authnReq.ID,
			OrgSlug:     slug,
			AppRedirect: appRedirect,
			AppState:    appState,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		if err := h.MemoryStoreProvider.SetState(samlFlowPrefix+relayState, string(flowJSON)); err != nil {
			log.Debug().Err(err).Msg("failed to persist saml flow state")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		redirectURL, err := authnReq.Redirect(relayState, sp)
		if err != nil {
			log.Debug().Err(err).Msg("failed to encode AuthnRequest redirect")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sso_config_error", "error_description": "could not build authentication request"})
			return
		}

		metrics.RecordAuthEvent(metrics.EventOAuthLogin, metrics.StatusSuccess)
		h.AuditProvider.LogEvent(audit.Event{
			Action:       constants.AuditSAMLLoginInitiatedEvent,
			ActorType:    constants.AuditActorTypeUser,
			ResourceType: constants.AuditResourceTypeSession,
			Metadata:     slug,
			IPAddress:    utils.GetIP(c.Request),
			UserAgent:    utils.GetUserAgent(c.Request),
		})
		c.Redirect(http.StatusFound, redirectURL.String())
	}
}

// SAMLACSHandler is the Assertion Consumer Service — the security-critical
// endpoint. It validates the signed assertion against the org's IdP config,
// enforces single-use (replay) semantics, JIT-provisions, and issues a session.
func (h *httpProvider) SAMLACSHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "SAMLACSHandler").Logger()
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		slug := strings.TrimSpace(c.Param("org_slug"))
		if h.MemoryStoreProvider == nil {
			samlFail(c, &log, slug, "internal_error", "server error")
			return
		}
		if err := c.Request.ParseForm(); err != nil {
			samlFail(c, &log, slug, "invalid_request", "malformed request")
			return
		}
		// Artifact binding performs an outbound resolution call (SSRF surface) and
		// is not supported here — only the HTTP-POST assertion binding is accepted.
		if strings.TrimSpace(c.Request.PostForm.Get("SAMLart")) != "" {
			samlFail(c, &log, slug, "unsupported_binding", "artifact binding is not supported")
			return
		}
		rawResponse := strings.TrimSpace(c.Request.PostForm.Get("SAMLResponse"))
		if rawResponse == "" {
			samlFail(c, &log, slug, "invalid_request", "missing SAMLResponse")
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(rawResponse)
		if err != nil {
			samlFail(c, &log, slug, "invalid_request", "malformed SAMLResponse")
			return
		}

		conn, ok := h.resolveActiveSAMLConnection(c, slug, &log)
		if !ok {
			return
		}
		sp, err := buildSAMLServiceProvider(conn, parsers.GetHost(c), slug)
		if err != nil {
			log.Debug().Err(err).Msg("failed to build SP for acs")
			samlFail(c, &log, slug, "sso_config_error", "connection misconfigured")
			return
		}

		// Resolve the pending SP-initiated request (single-use) via the opaque
		// RelayState, or fall back to the IdP-initiated path when permitted.
		possibleRequestIDs, appRedirect, appState, ok := h.resolveSAMLResponseContext(c, sp, slug, &log)
		if !ok {
			return
		}

		assertion, err := sp.ParseXMLResponse(decoded, possibleRequestIDs, sp.AcsURL)
		if err != nil {
			// crewjam wraps the real cause in InvalidResponseError.PrivateErr.
			log.Debug().Err(err).Msg("saml assertion validation failed")
			metrics.RecordSecurityEvent("saml_assertion_invalid", slug)
			samlFail(c, &log, slug, "saml_assertion_invalid", "assertion validation failed")
			return
		}

		// Single-use AssertionID (replay defence). crewjam does not dedupe.
		if err := h.consumeSAMLAssertionID(conn.OrgID, assertion); err != nil {
			metrics.RecordSecurityEvent("saml_assertion_replay", slug)
			samlFail(c, &log, slug, "saml_assertion_replay", "assertion already used")
			return
		}

		subject := ""
		if assertion.Subject != nil && assertion.Subject.NameID != nil {
			subject = strings.TrimSpace(assertion.Subject.NameID.Value)
		}
		if subject == "" {
			samlFail(c, &log, slug, "saml_assertion_invalid", "assertion has no subject NameID")
			return
		}

		profile := extractSAMLProfile(assertion, conn.SAMLAttributeMapping)
		// JIT provisioning namespaced by (org_id, IdP entity id, NameID). The IdP
		// entity id is stored in conn.IssuerURL.
		user, isSignUp, err := h.jitProvisionFederatedUser(ctx, conn.OrgID, conn.IssuerURL, subject, profile)
		if err != nil {
			log.Debug().Err(err).Msg("saml JIT provisioning rejected")
			samlFail(c, &log, slug, "saml_provisioning_failed", err.Error())
			return
		}

		if err := h.issueSAMLSession(c, slug, appRedirect, appState, user, isSignUp); err != nil {
			log.Debug().Err(err).Msg("failed to issue session")
			samlFail(c, &log, slug, "saml_session_failed", "could not establish session")
			return
		}
	}
}

// resolveSAMLResponseContext binds the ACS response to a pending SP-initiated
// request (via the opaque RelayState) or, when the connection opts in, accepts an
// IdP-initiated response. RelayState is NEVER used as a redirect target — the
// final redirect is always a server-side value validated against AllowedOrigins.
func (h *httpProvider) resolveSAMLResponseContext(c *gin.Context, sp *saml.ServiceProvider, slug string, log *zerolog.Logger) ([]string, string, string, bool) {
	relayState := strings.TrimSpace(c.Request.PostForm.Get("RelayState"))
	if relayState != "" {
		raw, err := h.MemoryStoreProvider.GetAndRemoveState(samlFlowPrefix + relayState)
		if err != nil && err != goredis.Nil {
			log.Debug().Err(err).Msg("failed to read saml flow state")
		}
		if strings.TrimSpace(raw) != "" {
			var flow samlFlowState
			if err := json.Unmarshal([]byte(raw), &flow); err != nil {
				samlFail(c, log, slug, "invalid_state", "corrupt state")
				return nil, "", "", false
			}
			if flow.OrgSlug != slug {
				samlFail(c, log, slug, "invalid_state", "state/route mismatch")
				return nil, "", "", false
			}
			return []string{flow.RequestID}, flow.AppRedirect, flow.AppState, true
		}
	}

	// No pending SP-initiated request — this is an IdP-initiated response.
	if !sp.AllowIDPInitiated {
		metrics.RecordSecurityEvent("saml_idp_initiated_rejected", slug)
		samlFail(c, log, slug, "idp_initiated_disabled", "IdP-initiated SSO is disabled for this connection")
		return nil, "", "", false
	}
	// The redirect target for an IdP-initiated flow must still be an explicit,
	// AllowedOrigins-validated redirect_uri — RelayState is not trusted as a URL.
	appRedirect := strings.TrimSpace(c.Request.PostForm.Get("redirect_uri"))
	if appRedirect == "" || !validators.IsValidRedirectURI(appRedirect, h.Config.AllowedOrigins, parsers.GetHost(c)) {
		samlFail(c, log, slug, "invalid_request", "invalid redirect_uri for IdP-initiated flow")
		return nil, "", "", false
	}
	return nil, appRedirect, "", true
}

// consumeSAMLAssertionID enforces single-use of an AssertionID within the org.
// Returns an error if the assertion was already consumed (replay). The cache TTL
// tracks the assertion's own expiry so the entry cannot outlive the replay window.
//
// ponytail: check-then-set (not atomic) — the memory-store interface exposes no
// SetNX, so two requests replaying the same assertion within the same few
// milliseconds could both pass. The assertion's own short NotOnOrAfter window
// bounds this; upgrade path: add an atomic SetNX to the memory store.
func (h *httpProvider) consumeSAMLAssertionID(orgID string, assertion *saml.Assertion) error {
	id := strings.TrimSpace(assertion.ID)
	if id == "" {
		return fmt.Errorf("assertion has no ID")
	}
	key := samlAssertionPrefix + orgID + ":" + id
	existing, err := h.MemoryStoreProvider.GetCache(key)
	if err == nil && strings.TrimSpace(existing) != "" {
		return fmt.Errorf("assertion replay detected")
	}
	ttl := samlReplayFallbackTTL
	if assertion.Conditions != nil && !assertion.Conditions.NotOnOrAfter.IsZero() {
		if secs := int64(time.Until(assertion.Conditions.NotOnOrAfter.Add(saml.MaxClockSkew)).Seconds()); secs > ttl {
			ttl = secs
		}
	}
	return h.MemoryStoreProvider.SetCache(key, "1", ttl)
}

// resolveActiveSAMLConnection looks up the org by slug and its active sso_saml
// connection, writing an error response and returning ok=false on any failure.
func (h *httpProvider) resolveActiveSAMLConnection(c *gin.Context, slug string, log *zerolog.Logger) (*schemas.TrustedIssuer, bool) {
	ctx := c.Request.Context()
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "missing organization"})
		return nil, false
	}
	org, err := h.StorageProvider.GetOrganizationByName(ctx, slug)
	if err != nil || org == nil {
		log.Debug().Err(err).Str("org", slug).Msg("organization not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "sso_not_configured", "error_description": "unknown organization"})
		return nil, false
	}
	if !org.Enabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "sso_not_configured", "error_description": "organization disabled"})
		return nil, false
	}
	conn, err := h.StorageProvider.GetTrustedIssuerByOrgIDAndKind(ctx, org.ID, constants.TrustKindSSOSAML)
	if err != nil || conn == nil || conn.EffectiveKind() != constants.TrustKindSSOSAML || !conn.IsActive {
		log.Debug().Err(err).Str("org", slug).Msg("no active SAML connection for org")
		c.JSON(http.StatusNotFound, gin.H{"error": "sso_not_configured", "error_description": "SAML SSO is not configured for this organization"})
		return nil, false
	}
	return conn, true
}

// buildSAMLServiceProvider constructs a per-org saml.ServiceProvider from the
// stored connection. The SP's EntityID (audience) and AcsURL (recipient) come
// from THIS org's config, and the IDP metadata trusts ONLY this org's certificate
// — the mechanism that keeps an Org B assertion from being consumed at Org A.
func buildSAMLServiceProvider(conn *schemas.TrustedIssuer, hostname, slug string) (*saml.ServiceProvider, error) {
	if conn.SAMLIDPCertPEM == nil || strings.TrimSpace(*conn.SAMLIDPCertPEM) == "" {
		return nil, fmt.Errorf("connection has no IdP certificate")
	}
	if conn.SAMLSSOURL == nil || strings.TrimSpace(*conn.SAMLSSOURL) == "" {
		return nil, fmt.Errorf("connection has no IdP SSO URL")
	}
	block, _ := pem.Decode([]byte(strings.TrimSpace(*conn.SAMLIDPCertPEM)))
	if block == nil {
		return nil, fmt.Errorf("IdP certificate is not valid PEM")
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return nil, fmt.Errorf("IdP certificate is not a valid X.509 certificate: %w", err)
	}
	certB64 := base64.StdEncoding.EncodeToString(block.Bytes)

	spEntityID := samlDefaultSPEntityID(hostname, slug)
	if conn.SAMLSPEntityID != nil && strings.TrimSpace(*conn.SAMLSPEntityID) != "" {
		spEntityID = strings.TrimSpace(*conn.SAMLSPEntityID)
	}
	acsRaw := samlDefaultACSURL(hostname, slug)
	if conn.SAMLACSURL != nil && strings.TrimSpace(*conn.SAMLACSURL) != "" {
		acsRaw = strings.TrimSpace(*conn.SAMLACSURL)
	}
	acsURL, err := url.Parse(acsRaw)
	if err != nil {
		return nil, fmt.Errorf("invalid ACS URL: %w", err)
	}
	metadataURL, err := url.Parse(spEntityID)
	if err != nil {
		return nil, fmt.Errorf("invalid SP entity ID: %w", err)
	}

	return &saml.ServiceProvider{
		EntityID:    spEntityID,
		MetadataURL: *metadataURL,
		AcsURL:      *acsURL,
		IDPMetadata: &saml.EntityDescriptor{
			EntityID: conn.IssuerURL,
			IDPSSODescriptors: []saml.IDPSSODescriptor{{
				SSODescriptor: saml.SSODescriptor{
					RoleDescriptor: saml.RoleDescriptor{
						KeyDescriptors: []saml.KeyDescriptor{{
							Use: "signing",
							KeyInfo: saml.KeyInfo{
								X509Data: saml.X509Data{
									X509Certificates: []saml.X509Certificate{{Data: certB64}},
								},
							},
						}},
					},
				},
				SingleSignOnServices: []saml.Endpoint{{
					Binding:  saml.HTTPRedirectBinding,
					Location: strings.TrimSpace(*conn.SAMLSSOURL),
				}},
			}},
		},
		AllowIDPInitiated: conn.SAMLAllowIDPInitiated,
	}, nil
}

func samlDefaultSPEntityID(hostname, slug string) string {
	return strings.TrimRight(hostname, "/") + "/oauth/saml/" + url.PathEscape(slug) + "/metadata"
}

func samlDefaultACSURL(hostname, slug string) string {
	return strings.TrimRight(hostname, "/") + "/oauth/saml/" + url.PathEscape(slug) + "/acs"
}

// extractSAMLProfile pulls optional profile attributes out of the assertion using
// the connection's attribute mapping (or the built-in defaults). The NameID is
// the federated subject and is handled by the caller, not here.
func extractSAMLProfile(assertion *saml.Assertion, mappingJSON *string) federatedProfile {
	mapping := samlDefaultAttributeMapping
	if mappingJSON != nil && strings.TrimSpace(*mappingJSON) != "" {
		var m map[string]string
		if err := json.Unmarshal([]byte(*mappingJSON), &m); err == nil && len(m) > 0 {
			mapping = m
		}
	}
	return federatedProfile{
		Email:         samlAttr(assertion, mapping["email"]),
		EmailVerified: true, // an assertion from a trusted corporate IdP asserts the identity
		GivenName:     samlAttr(assertion, mapping["given_name"]),
		FamilyName:    samlAttr(assertion, mapping["family_name"]),
		Nickname:      samlAttr(assertion, mapping["nickname"]),
		Picture:       samlAttr(assertion, mapping["picture"]),
	}
}

// samlAttr returns the first value of the assertion attribute whose Name or
// FriendlyName matches (case-insensitively). Empty name yields empty string.
func samlAttr(assertion *saml.Assertion, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			if strings.EqualFold(attr.Name, name) || strings.EqualFold(attr.FriendlyName, name) {
				for _, v := range attr.Values {
					if s := strings.TrimSpace(v.Value); s != "" {
						return s
					}
				}
			}
		}
	}
	return ""
}

// issueSAMLSession mints the Authorizer session/tokens, sets the session cookie,
// records the session, and redirects to the app's redirect_uri. Mirrors
// issueSSOSession but with SAML audit semantics and no upstream nonce.
func (h *httpProvider) issueSAMLSession(c *gin.Context, slug, appRedirect, appState string, user *schemas.User, isSignUp bool) error {
	hostname := parsers.GetHost(c)
	roles := splitRoles(user.Roles)
	authToken, err := h.TokenProvider.CreateAuthToken(c, &token.AuthTokenConfig{
		User:        user,
		Roles:       roles,
		Scope:       []string{"openid", "profile", "email"},
		LoginMethod: constants.AuthRecipeMethodSSO,
		HostName:    hostname,
	})
	if err != nil {
		return err
	}

	sessionKey := constants.AuthRecipeMethodSSO + ":" + user.ID
	cookie.SetSession(c, authToken.FingerPrintHash, h.Config.AppCookieSecure, cookie.ParseSameSite(h.Config.AppCookieSameSite))
	_ = h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	_ = h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)
	if authToken.RefreshToken != nil {
		_ = h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}

	bgCtx := context.WithoutCancel(c.Request.Context())
	userAgent := utils.GetUserAgent(c.Request)
	ip := utils.GetIP(c.Request)
	go func() {
		if isSignUp {
			_ = h.EventsProvider.RegisterEvent(bgCtx, constants.UserSignUpWebhookEvent, constants.AuthRecipeMethodSSO, user)
		}
		_ = h.EventsProvider.RegisterEvent(bgCtx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodSSO, user)
		if err := h.StorageProvider.AddSession(bgCtx, &schemas.Session{UserID: user.ID, UserAgent: userAgent, IP: ip}); err != nil {
			h.Log.Debug().Err(err).Msg("failed to add session")
		}
	}()

	params := "state=" + url.QueryEscape(appState)
	redirectURL := appRedirect
	if strings.Contains(redirectURL, "?") {
		redirectURL = redirectURL + "&" + params
	} else {
		redirectURL = redirectURL + "?" + params
	}
	metrics.RecordAuthEvent(metrics.EventOAuthCallback, metrics.StatusSuccess)
	metrics.ActiveSessions.Inc()
	h.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditSAMLACSSuccessEvent,
		ActorID:      user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeSession,
		ResourceID:   user.ID,
		Metadata:     slug,
		IPAddress:    ip,
		UserAgent:    userAgent,
	})
	c.Redirect(http.StatusFound, redirectURL)
	return nil
}

// samlFail writes a uniform OAuth-style error response and records the failure.
func samlFail(c *gin.Context, log *zerolog.Logger, slug, code, desc string) {
	metrics.RecordAuthEvent(metrics.EventOAuthCallback, metrics.StatusFailure)
	log.Debug().Str("error", code).Str("org", slug).Msg("saml acs failed")
	c.JSON(http.StatusBadRequest, gin.H{"error": code, "error_description": desc})
}
