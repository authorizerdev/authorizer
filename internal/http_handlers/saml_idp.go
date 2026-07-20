package http_handlers

// Per-organization SAML 2.0 Identity Provider — Authorizer acting as the IdP that
// issues signed assertions to downstream Service Providers (Zendesk, Notion, …).
// This is the architectural inverse of saml_sp.go (Authorizer as SP): there we
// CONSUME a signed assertion; here we PRODUCE one.
//
// SECURITY MODEL — the assertions this file emits must satisfy every check the SP
// side documents (saml_sp.go header): a valid XML-DSIG signature over the
// assertion, a tight Audience/Recipient/Destination/NotBefore/NotOnOrAfter, and
// (SP-initiated) an InResponseTo bound to the SP's AuthnRequest.
//
//   - ACS/EntityID strict binding (open-redirect / exfiltration guard): the SP is
//     resolved by the AuthnRequest Issuer against the org's REGISTERED
//     SAMLServiceProvider rows via samlSPRegistry.GetServiceProvider. The ACS URL
//     is taken ONLY from that record; crewjam's IdpAuthnRequest.Validate rejects a
//     request-supplied AssertionConsumerServiceURL that does not match it. A
//     request for an unregistered SP is refused (os.ErrNotExist).
//   - Audience isolation: the assertion Audience is set to the resolved SP's
//     EntityID, so an assertion minted for SP-A cannot validate at SP-B.
//   - No unauthenticated issuance (SP-initiated): an assertion is only produced
//     once the browser presents a valid Authorizer session; otherwise the flow
//     bounces through the normal login UI and resumes.
//   - IdP-initiated SSO is refused unless the registered SP opts in
//     (AllowIDPInitiated). There is NO issuance-side replay guard for the
//     unsolicited path: replay defence for an assertion is the CONSUMING SP's job
//     (single-use AssertionID), and Authorizer is the issuer here — a tight
//     NotBefore/NotOnOrAfter window bounds the assertion instead. (See PR notes.)
//   - Signing uses ONLY the org's "current" key; metadata additionally publishes
//     every "active" (superseded-but-not-retired) cert so SPs with cached metadata
//     keep validating during a rotation overlap.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/crewjam/saml"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

const (
	// samlIDPFlowPrefix namespaces a pending IdP SSO request stashed in the shared
	// store while the browser completes login. Single-use, short OAuth-state TTL.
	samlIDPFlowPrefix = "saml_idp_flow:"
	// samlIDPContinueParam carries the opaque flow key back to the SSO endpoint
	// after the login bounce.
	samlIDPContinueParam = "saml_continue"
	// defaultSAMLNameIDFormat is used when a registered SP specifies none.
	defaultSAMLNameIDFormat = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
	// rsaSHA256SigMethod is dsig.RSASHA256SignatureMethod, inlined to avoid a
	// direct goxmldsig import for a single constant. Assertions are signed
	// RSA-SHA256 (stronger than the crewjam RSA-SHA1 default).
	rsaSHA256SigMethod = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
)

// samlIDPFlowState is the pending IdP SSO request preserved across the login
// bounce, stored single-use under an opaque key. It never contains a redirect
// target — the ACS URL is always re-resolved from the registered SP record.
type samlIDPFlowState struct {
	OrgSlug      string `json:"org_slug"`
	IdPInitiated bool   `json:"idp_initiated"`
	SPEntityID   string `json:"sp_entity_id"` // IdP-initiated: the target SP
	RelayState   string `json:"relay_state"`
	RequestB64   string `json:"request_b64"` // SP-initiated: base64 of the raw AuthnRequest
}

// SAMLIDPMetadataHandler serves this org's IdP metadata (EntityDescriptor /
// IDPSSODescriptor) so a downstream SP can register Authorizer as its IdP. Every
// currently-published (current + active) signing certificate is emitted as a
// separate <KeyDescriptor use="signing">.
func (h *httpProvider) SAMLIDPMetadataHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "SAMLIDPMetadataHandler").Logger()
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		slug := strings.TrimSpace(c.Param("org_slug"))
		orgID, ok := h.resolveSAMLIDPOrg(c, slug, &log)
		if !ok {
			return
		}
		if _, err := h.getOrCreateCurrentSAMLKey(ctx, orgID, slug); err != nil {
			log.Debug().Err(err).Msg("failed to ensure signing key")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		keys, err := h.StorageProvider.ListSAMLIDPKeys(ctx, orgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		hostname := parsers.GetHost(c)
		md := buildIDPMetadata(samlIDPEntityID(hostname, slug), samlIDPSSOURL(hostname, slug), publishedSAMLKeys(keys))
		body, err := xml.MarshalIndent(md, "", "  ")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		out := append([]byte(xml.Header), body...)
		c.Data(http.StatusOK, "application/samlmetadata+xml", out)
	}
}

// SAMLIDPSSOHandler is the SP-initiated SSO endpoint (GET HTTP-Redirect binding,
// POST HTTP-POST binding). It also handles the post-login resume (?saml_continue).
func (h *httpProvider) SAMLIDPSSOHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "SAMLIDPSSOHandler").Logger()
	return func(c *gin.Context) {
		slug := strings.TrimSpace(c.Param("org_slug"))
		orgID, ok := h.resolveSAMLIDPOrg(c, slug, &log)
		if !ok {
			return
		}

		// Resume path after a login bounce.
		if key := strings.TrimSpace(c.Query(samlIDPContinueParam)); key != "" {
			h.resumeSAMLIDPFlow(c, orgID, slug, key, &log)
			return
		}

		idp, err := h.buildIdentityProvider(c, orgID, slug)
		if err != nil {
			h.samlIDPFail(c, &log, slug, "sso_config_error", "IdP not configured")
			return
		}

		req, err := saml.NewIdpAuthnRequest(idp, c.Request)
		if err != nil {
			h.samlIDPFail(c, &log, slug, "invalid_request", "malformed AuthnRequest")
			return
		}
		if err := req.Validate(); err != nil {
			// Validate binds the SP + ACS URL against the registry; failure here is
			// an unknown SP, an ACS/Destination mismatch, or an expired request.
			log.Debug().Err(err).Msg("AuthnRequest validation failed")
			metrics.RecordSecurityEvent("saml_idp_request_invalid", slug)
			h.samlIDPFail(c, &log, slug, "invalid_authn_request", "authentication request rejected")
			return
		}

		user, authed := h.currentSAMLUser(c)
		if !authed {
			flow := samlIDPFlowState{
				OrgSlug:    slug,
				RelayState: req.RelayState,
				RequestB64: base64.StdEncoding.EncodeToString(req.RequestBuffer),
			}
			h.bounceSAMLIDPToLogin(c, slug, flow, &log)
			return
		}

		h.emitSAMLAssertion(c, idp, req, orgID, slug, user, &log)
	}
}

// SAMLIDPInitiatedHandler builds and POSTs an unsolicited assertion to a
// registered SP's ACS. Gated by the SP's AllowIDPInitiated flag.
// Route: GET /saml/idp/:org_slug/sso/:sp_id
func (h *httpProvider) SAMLIDPInitiatedHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "SAMLIDPInitiatedHandler").Logger()
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		slug := strings.TrimSpace(c.Param("org_slug"))
		spID := strings.TrimSpace(c.Param("sp_id"))
		relayState := strings.TrimSpace(c.Query("RelayState"))
		orgID, ok := h.resolveSAMLIDPOrg(c, slug, &log)
		if !ok {
			return
		}

		sp, err := h.StorageProvider.GetSAMLServiceProviderByID(ctx, spID)
		if err != nil || sp == nil || sp.OrgID != orgID || !sp.IsActive {
			h.samlIDPFail(c, &log, slug, "unknown_service_provider", "unknown service provider")
			return
		}
		if !sp.AllowIDPInitiated {
			metrics.RecordSecurityEvent("saml_idp_initiated_rejected", slug)
			h.samlIDPFail(c, &log, slug, "idp_initiated_disabled", "IdP-initiated SSO is disabled for this service provider")
			return
		}

		user, authed := h.currentSAMLUser(c)
		if !authed {
			flow := samlIDPFlowState{
				OrgSlug:      slug,
				IdPInitiated: true,
				SPEntityID:   sp.EntityID,
				RelayState:   relayState,
			}
			h.bounceSAMLIDPToLogin(c, slug, flow, &log)
			return
		}

		idp, err := h.buildIdentityProvider(c, orgID, slug)
		if err != nil {
			h.samlIDPFail(c, &log, slug, "sso_config_error", "IdP not configured")
			return
		}
		h.emitIDPInitiatedAssertion(c, idp, sp, orgID, slug, relayState, user, &log)
	}
}

// resumeSAMLIDPFlow reloads a pending SSO request after the login bounce and, now
// that a session exists, completes it.
func (h *httpProvider) resumeSAMLIDPFlow(c *gin.Context, orgID, slug, key string, log *zerolog.Logger) {
	ctx := c.Request.Context()
	raw, err := h.MemoryStoreProvider.GetAndRemoveState(samlIDPFlowPrefix + key)
	if err != nil || strings.TrimSpace(raw) == "" {
		h.samlIDPFail(c, log, slug, "invalid_state", "the login request expired, please retry")
		return
	}
	var flow samlIDPFlowState
	if err := json.Unmarshal([]byte(raw), &flow); err != nil || flow.OrgSlug != slug {
		h.samlIDPFail(c, log, slug, "invalid_state", "corrupt login state")
		return
	}
	user, authed := h.currentSAMLUser(c)
	if !authed {
		// Login did not establish a session — do not loop back to login.
		h.samlIDPFail(c, log, slug, "login_required", "authentication required")
		return
	}
	idp, err := h.buildIdentityProvider(c, orgID, slug)
	if err != nil {
		h.samlIDPFail(c, log, slug, "sso_config_error", "IdP not configured")
		return
	}

	if flow.IdPInitiated {
		sp, err := h.StorageProvider.GetSAMLServiceProviderByOrgAndEntityID(ctx, orgID, flow.SPEntityID)
		if err != nil || sp == nil || !sp.IsActive || !sp.AllowIDPInitiated {
			h.samlIDPFail(c, log, slug, "unknown_service_provider", "service provider no longer available")
			return
		}
		h.emitIDPInitiatedAssertion(c, idp, sp, orgID, slug, flow.RelayState, user, log)
		return
	}

	buf, err := base64.StdEncoding.DecodeString(flow.RequestB64)
	if err != nil {
		h.samlIDPFail(c, log, slug, "invalid_state", "corrupt request")
		return
	}
	req := &saml.IdpAuthnRequest{
		IDP:           idp,
		HTTPRequest:   c.Request,
		RequestBuffer: buf,
		RelayState:    flow.RelayState,
		Now:           saml.TimeNow(),
	}
	if err := req.Validate(); err != nil {
		metrics.RecordSecurityEvent("saml_idp_request_invalid", slug)
		h.samlIDPFail(c, log, slug, "invalid_authn_request", "authentication request rejected")
		return
	}
	h.emitSAMLAssertion(c, idp, req, orgID, slug, user, log)
}

// emitSAMLAssertion completes an SP-initiated flow: it maps attributes from the
// registered SP record, builds the assertion, and writes the auto-POST form.
func (h *httpProvider) emitSAMLAssertion(c *gin.Context, idp *saml.IdentityProvider, req *saml.IdpAuthnRequest, orgID, slug string, user *schemas.User, log *zerolog.Logger) {
	ctx := c.Request.Context()
	spEntityID := req.ServiceProviderMetadata.EntityID
	sp, err := h.StorageProvider.GetSAMLServiceProviderByOrgAndEntityID(ctx, orgID, spEntityID)
	if err != nil || sp == nil || !sp.IsActive {
		h.samlIDPFail(c, log, slug, "unknown_service_provider", "unknown service provider")
		return
	}
	session := buildSAMLSession(user, sp)
	maker := mappedAssertionMaker{attributes: buildMappedAttributes(user, sp)}
	if err := maker.MakeAssertion(req, session); err != nil {
		log.Debug().Err(err).Msg("failed to build assertion")
		h.samlIDPFail(c, log, slug, "assertion_error", "could not build assertion")
		return
	}
	if err := req.WriteResponse(c.Writer); err != nil {
		log.Debug().Err(err).Msg("failed to write assertion response")
		h.samlIDPFail(c, log, slug, "assertion_error", "could not deliver assertion")
		return
	}
	h.auditSAMLIDPIssued(c, slug, sp, user)
}

// emitIDPInitiatedAssertion produces an unsolicited assertion and POSTs it to the
// SP's ACS. Mirrors crewjam ServeIDPInitiated, but with our session + mapped
// attributes. No InResponseTo (there is no request to bind to).
func (h *httpProvider) emitIDPInitiatedAssertion(c *gin.Context, idp *saml.IdentityProvider, sp *schemas.SAMLServiceProvider, orgID, slug, relayState string, user *schemas.User, log *zerolog.Logger) {
	spMetadata := buildSPEntityDescriptor(sp)
	req := &saml.IdpAuthnRequest{
		IDP:                     idp,
		HTTPRequest:             c.Request,
		RelayState:              relayState,
		ServiceProviderMetadata: spMetadata,
		SPSSODescriptor:         &spMetadata.SPSSODescriptors[0],
		ACSEndpoint:             &spMetadata.SPSSODescriptors[0].AssertionConsumerServices[0],
		Now:                     saml.TimeNow(),
	}
	session := buildSAMLSession(user, sp)
	maker := mappedAssertionMaker{attributes: buildMappedAttributes(user, sp)}
	if err := maker.MakeAssertion(req, session); err != nil {
		log.Debug().Err(err).Msg("failed to build idp-initiated assertion")
		h.samlIDPFail(c, log, slug, "assertion_error", "could not build assertion")
		return
	}
	if err := req.WriteResponse(c.Writer); err != nil {
		log.Debug().Err(err).Msg("failed to write idp-initiated response")
		h.samlIDPFail(c, log, slug, "assertion_error", "could not deliver assertion")
		return
	}
	h.auditSAMLIDPIssued(c, slug, sp, user)
}

// mappedAssertionMaker delegates to crewjam's DefaultAssertionMaker for the
// security-critical Subject/Conditions/InResponseTo/Audience/NotBefore
// construction, then replaces the attribute statement with exactly the attributes
// the registered SP was configured to receive.
type mappedAssertionMaker struct {
	attributes []saml.Attribute
}

func (m mappedAssertionMaker) MakeAssertion(req *saml.IdpAuthnRequest, session *saml.Session) error {
	if err := (saml.DefaultAssertionMaker{}).MakeAssertion(req, session); err != nil {
		return err
	}
	req.Assertion.AttributeStatements = []saml.AttributeStatement{{Attributes: m.attributes}}
	return nil
}

// buildSAMLSession assembles the crewjam session that drives the assertion
// Subject/NameID and AuthnStatement. Attributes are handled by
// mappedAssertionMaker, not here.
func buildSAMLSession(user *schemas.User, sp *schemas.SAMLServiceProvider) *saml.Session {
	format := strings.TrimSpace(sp.NameIDFormat)
	if format == "" {
		format = defaultSAMLNameIDFormat
	}
	email := refs.StringValue(user.Email)
	// NameID: the email address for the emailAddress format, otherwise the stable
	// user id (persistent/unspecified/transient SPs key off an opaque identifier).
	nameID := user.ID
	if format == defaultSAMLNameIDFormat && email != "" {
		nameID = email
	}
	now := saml.TimeNow()
	return &saml.Session{
		ID:            user.ID,
		CreateTime:    now,
		ExpireTime:    now.Add(time.Hour),
		Index:         user.ID,
		NameID:        nameID,
		NameIDFormat:  format,
		UserEmail:     email,
		UserName:      nameID,
		UserGivenName: refs.StringValue(user.GivenName),
		UserSurname:   refs.StringValue(user.FamilyName),
	}
}

// buildMappedAttributes emits the SAML attributes for the registered SP. The map
// keys are Authorizer profile fields; the map VALUES are the SAML attribute names
// this SP expects (the inverse of the SP-side attribute mapping). Empty/omitted
// values are skipped.
func buildMappedAttributes(user *schemas.User, sp *schemas.SAMLServiceProvider) []saml.Attribute {
	mapping := samlDefaultAttributeMapping
	if sp.MappedAttributes != nil && strings.TrimSpace(*sp.MappedAttributes) != "" {
		var m map[string]string
		if err := json.Unmarshal([]byte(*sp.MappedAttributes), &m); err == nil && len(m) > 0 {
			mapping = m
		}
	}
	attrs := []saml.Attribute{}
	add := func(field string, value string) {
		name := strings.TrimSpace(mapping[field])
		if name == "" || strings.TrimSpace(value) == "" {
			return
		}
		attrs = append(attrs, saml.Attribute{
			Name:       name,
			NameFormat: "urn:oasis:names:tc:SAML:2.0:attrname-format:basic",
			Values:     []saml.AttributeValue{{Type: "xs:string", Value: value}},
		})
	}
	add("email", refs.StringValue(user.Email))
	add("given_name", refs.StringValue(user.GivenName))
	add("family_name", refs.StringValue(user.FamilyName))
	add("nickname", refs.StringValue(user.Nickname))
	add("picture", refs.StringValue(user.Picture))
	return attrs
}

// samlSPRegistry implements crewjam's ServiceProviderProvider against the org's
// registered SAMLServiceProvider rows. This is the strict binding: the returned
// EntityDescriptor carries the SP's ACS URL from the DB, never from the request,
// so Validate refuses a request that names a different ACS.
type samlSPRegistry struct {
	storage interface {
		GetSAMLServiceProviderByOrgAndEntityID(ctx context.Context, orgID, entityID string) (*schemas.SAMLServiceProvider, error)
	}
	orgID string
	ctx   context.Context
}

func (r *samlSPRegistry) GetServiceProvider(_ *http.Request, serviceProviderID string) (*saml.EntityDescriptor, error) {
	sp, err := r.storage.GetSAMLServiceProviderByOrgAndEntityID(r.ctx, r.orgID, serviceProviderID)
	if err != nil || sp == nil || !sp.IsActive {
		return nil, os.ErrNotExist
	}
	return buildSPEntityDescriptor(sp), nil
}

// buildSPEntityDescriptor builds the minimal SP metadata crewjam needs to bind an
// assertion: the EntityID (assertion Audience) and the single trusted ACS URL. The
// SP's optional signing cert is intentionally NOT wired as an encryption key here,
// so assertions are signed but not encrypted (the compatible default); it is
// retained on the record for future AuthnRequest-signature validation.
func buildSPEntityDescriptor(sp *schemas.SAMLServiceProvider) *saml.EntityDescriptor {
	isDefault := true
	return &saml.EntityDescriptor{
		EntityID: sp.EntityID,
		SPSSODescriptors: []saml.SPSSODescriptor{{
			SSODescriptor: saml.SSODescriptor{
				RoleDescriptor: saml.RoleDescriptor{
					ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
				},
			},
			AssertionConsumerServices: []saml.IndexedEndpoint{{
				Binding:   saml.HTTPPostBinding,
				Location:  sp.ACSURL,
				Index:     0,
				IsDefault: &isDefault,
			}},
		}},
	}
}

// buildIDPMetadata renders the IdP EntityDescriptor with one signing
// <KeyDescriptor> per published (current + active) certificate.
func buildIDPMetadata(entityID, ssoURL string, keys []*schemas.SAMLIDPKey) *saml.EntityDescriptor {
	keyDescriptors := make([]saml.KeyDescriptor, 0, len(keys))
	for _, k := range keys {
		block := pemCertB64(k.CertPEM)
		if block == "" {
			continue
		}
		keyDescriptors = append(keyDescriptors, saml.KeyDescriptor{
			Use: "signing",
			KeyInfo: saml.KeyInfo{
				X509Data: saml.X509Data{
					X509Certificates: []saml.X509Certificate{{Data: block}},
				},
			},
		})
	}
	return &saml.EntityDescriptor{
		EntityID: entityID,
		IDPSSODescriptors: []saml.IDPSSODescriptor{{
			SSODescriptor: saml.SSODescriptor{
				RoleDescriptor: saml.RoleDescriptor{
					ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
					KeyDescriptors:             keyDescriptors,
				},
				NameIDFormats: []saml.NameIDFormat{
					saml.NameIDFormat(defaultSAMLNameIDFormat),
					saml.NameIDFormat("urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"),
				},
			},
			SingleSignOnServices: []saml.Endpoint{
				{Binding: saml.HTTPRedirectBinding, Location: ssoURL},
				{Binding: saml.HTTPPostBinding, Location: ssoURL},
			},
		}},
	}
}

// currentSAMLUser resolves the browser's authenticated Authorizer user from the
// session cookie, or (nil,false) when there is no valid session.
func (h *httpProvider) currentSAMLUser(c *gin.Context) (*schemas.User, bool) {
	sessionToken, err := cookie.GetSession(c)
	if err != nil || strings.TrimSpace(sessionToken) == "" {
		return nil, false
	}
	claims, err := h.TokenProvider.ValidateBrowserSession(c, sessionToken)
	if err != nil {
		return nil, false
	}
	user, err := h.StorageProvider.GetUserByID(c, claims.Subject)
	if err != nil || user == nil {
		return nil, false
	}
	return user, true
}

// bounceSAMLIDPToLogin stashes the pending SSO request and redirects the browser
// to the login UI, which returns to the SSO endpoint (?saml_continue) once a
// session exists. The continue URL is always Authorizer's own SSO endpoint — never
// a request-supplied redirect target.
func (h *httpProvider) bounceSAMLIDPToLogin(c *gin.Context, slug string, flow samlIDPFlowState, log *zerolog.Logger) {
	if h.MemoryStoreProvider == nil {
		h.samlIDPFail(c, log, slug, "internal_error", "server error")
		return
	}
	key, err := randURLString(32)
	if err != nil {
		h.samlIDPFail(c, log, slug, "internal_error", "server error")
		return
	}
	payload, err := json.Marshal(flow)
	if err != nil {
		h.samlIDPFail(c, log, slug, "internal_error", "server error")
		return
	}
	if err := h.MemoryStoreProvider.SetState(samlIDPFlowPrefix+key, string(payload)); err != nil {
		h.samlIDPFail(c, log, slug, "internal_error", "server error")
		return
	}
	hostname := parsers.GetHost(c)
	continueURL := samlIDPSSOURL(hostname, slug) + "?" + samlIDPContinueParam + "=" + url.QueryEscape(key)
	loginURL := "/app?redirect_uri=" + url.QueryEscape(continueURL)
	c.Redirect(http.StatusFound, loginURL)
}

// buildIdentityProvider constructs a per-org crewjam IdentityProvider signed by
// the org's current key and backed by the registry-based SP lookup.
func (h *httpProvider) buildIdentityProvider(c *gin.Context, orgID, slug string) (*saml.IdentityProvider, error) {
	ctx := c.Request.Context()
	current, err := h.getOrCreateCurrentSAMLKey(ctx, orgID, slug)
	if err != nil {
		return nil, err
	}
	priv, cert, err := h.parseSigningKey(current)
	if err != nil {
		return nil, err
	}
	hostname := parsers.GetHost(c)
	metadataURL, err := url.Parse(samlIDPEntityID(hostname, slug))
	if err != nil {
		return nil, err
	}
	ssoURL, err := url.Parse(samlIDPSSOURL(hostname, slug))
	if err != nil {
		return nil, err
	}
	return &saml.IdentityProvider{
		Key:             priv,
		Certificate:     cert,
		MetadataURL:     *metadataURL,
		SSOURL:          *ssoURL,
		SignatureMethod: rsaSHA256SigMethod,
		ServiceProviderProvider: &samlSPRegistry{
			storage: h.StorageProvider,
			orgID:   orgID,
			ctx:     ctx,
		},
	}, nil
}

// getOrCreateCurrentSAMLKey returns the org's current signing key, lazily
// generating the first keypair when none exists yet.
func (h *httpProvider) getOrCreateCurrentSAMLKey(ctx context.Context, orgID, slug string) (*schemas.SAMLIDPKey, error) {
	keys, err := h.StorageProvider.ListSAMLIDPKeys(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		if k.Status == schemas.SAMLIDPKeyStatusCurrent {
			return k, nil
		}
	}
	priv, certPEM, err := crypto.NewSAMLSigningKeypair("Authorizer SAML IdP " + slug)
	if err != nil {
		return nil, err
	}
	enc, err := crypto.EncryptAES(h.ClientSecret, priv)
	if err != nil {
		return nil, err
	}
	return h.StorageProvider.AddSAMLIDPKey(ctx, &schemas.SAMLIDPKey{
		OrgID:         orgID,
		CertPEM:       certPEM,
		PrivateKeyEnc: enc,
		Algorithm:     "RS256",
		Status:        schemas.SAMLIDPKeyStatusCurrent,
	})
}

// parseSigningKey decrypts and parses the RSA private key and X.509 cert of a key.
func (h *httpProvider) parseSigningKey(k *schemas.SAMLIDPKey) (*rsa.PrivateKey, *x509.Certificate, error) {
	privPEM, err := crypto.DecryptAES(h.ClientSecret, k.PrivateKeyEnc)
	if err != nil {
		return nil, nil, err
	}
	priv, err := crypto.ParseRsaPrivateKeyFromPemStr(privPEM)
	if err != nil {
		return nil, nil, err
	}
	cert, err := crypto.ParseCertificateFromPemStr(k.CertPEM)
	if err != nil {
		return nil, nil, err
	}
	return priv, cert, nil
}

// resolveSAMLIDPOrg looks up an enabled org by slug and returns its ID.
func (h *httpProvider) resolveSAMLIDPOrg(c *gin.Context, slug string, log *zerolog.Logger) (string, bool) {
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "missing organization"})
		return "", false
	}
	org, err := h.StorageProvider.GetOrganizationByName(c.Request.Context(), slug)
	if err != nil || org == nil {
		log.Debug().Err(err).Str("org", slug).Msg("organization not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "sso_not_configured", "error_description": "unknown organization"})
		return "", false
	}
	if !org.Enabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "sso_not_configured", "error_description": "organization disabled"})
		return "", false
	}
	return org.ID, true
}

// samlIDPFail writes a uniform error response and audits the rejection so
// assertion-forgery attempts (unknown SP, ACS mismatch, IdP-initiated-disabled)
// leave a trail.
func (h *httpProvider) samlIDPFail(c *gin.Context, log *zerolog.Logger, slug, code, desc string) {
	metrics.RecordAuthEvent(metrics.EventOAuthCallback, metrics.StatusFailure)
	log.Debug().Str("error", code).Str("org", slug).Msg("saml idp sso failed")
	h.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditSAMLIDPAssertionFailedEvent,
		ActorType:    constants.AuditActorTypeUser,
		ResourceType: constants.AuditResourceTypeSession,
		Metadata:     slug + ":" + code,
		IPAddress:    utils.GetIP(c.Request),
		UserAgent:    utils.GetUserAgent(c.Request),
	})
	c.JSON(http.StatusBadRequest, gin.H{"error": code, "error_description": desc})
}

func (h *httpProvider) auditSAMLIDPIssued(c *gin.Context, slug string, sp *schemas.SAMLServiceProvider, user *schemas.User) {
	metrics.RecordAuthEvent(metrics.EventOAuthCallback, metrics.StatusSuccess)
	h.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditSAMLIDPAssertionIssuedEvent,
		ActorID:      user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeSession,
		ResourceID:   sp.ID,
		Metadata:     slug + ":" + sp.EntityID,
		IPAddress:    utils.GetIP(c.Request),
		UserAgent:    utils.GetUserAgent(c.Request),
	})
}

// pemCertB64 extracts the base64 DER (single line) from a PEM certificate for
// embedding in <X509Certificate>. Returns "" if the PEM cannot be decoded.
func pemCertB64(certPEM string) string {
	cert, err := crypto.ParseCertificateFromPemStr(strings.TrimSpace(certPEM))
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(cert.Raw)
}

// publishedSAMLKeys returns the keys published in IdP metadata: current + active
// (never retired).
func publishedSAMLKeys(keys []*schemas.SAMLIDPKey) []*schemas.SAMLIDPKey {
	out := make([]*schemas.SAMLIDPKey, 0, len(keys))
	for _, k := range keys {
		if k.Status == schemas.SAMLIDPKeyStatusCurrent || k.Status == schemas.SAMLIDPKeyStatusActive {
			out = append(out, k)
		}
	}
	return out
}

func samlIDPEntityID(hostname, slug string) string {
	return strings.TrimRight(hostname, "/") + "/saml/idp/" + url.PathEscape(slug) + "/metadata"
}

func samlIDPSSOURL(hostname, slug string) string {
	return strings.TrimRight(hostname, "/") + "/saml/idp/" + url.PathEscape(slug) + "/sso"
}
