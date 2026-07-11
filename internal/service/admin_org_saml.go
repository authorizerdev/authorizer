package service

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/url"
	"strings"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// asAPIOrgSAMLConnection projects a sso_saml TrustedIssuer row onto the GraphQL
// model. The IdP signing certificate (SAMLIDPCertPEM) is intentionally NOT
// surfaced — the model has no certificate field.
func asAPIOrgSAMLConnection(t *schemas.TrustedIssuer) *model.OrgSAMLConnection {
	id := t.ID
	if strings.Contains(id, schemas.Collections.TrustedIssuer+"/") {
		id = strings.TrimPrefix(id, schemas.Collections.TrustedIssuer+"/")
	}
	return &model.OrgSAMLConnection{
		ID:                id,
		OrgID:             t.OrgID,
		Name:              t.Name,
		IdpEntityID:       t.IssuerURL,
		IdpSsoURL:         t.SAMLSSOURL,
		SpEntityID:        t.SAMLSPEntityID,
		AcsURL:            t.SAMLACSURL,
		AttributeMapping:  t.SAMLAttributeMapping,
		AllowIdpInitiated: t.SAMLAllowIDPInitiated,
		IsActive:          t.IsActive,
		CreatedAt:         refs.NewInt64Ref(t.CreatedAt),
		UpdatedAt:         refs.NewInt64Ref(t.UpdatedAt),
	}
}

// validateSAMLHTTPSURL rejects a non-https or opaque URL. Used for the IdP SSO
// endpoint and the optional SP entity ID / ACS URL overrides.
func validateSAMLHTTPSURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("must be a valid https URL")
	}
	return nil
}

// validateSAMLCertPEM ensures the supplied string is a parseable PEM-encoded
// X.509 certificate. This is a trust-boundary check: an unparseable cert would
// mean every assertion for the org fails signature validation, so reject early.
func validateSAMLCertPEM(raw string) error {
	block, _ := pem.Decode([]byte(strings.TrimSpace(raw)))
	if block == nil {
		return fmt.Errorf("idp_certificate must be a PEM-encoded X.509 certificate")
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return fmt.Errorf("idp_certificate is not a valid X.509 certificate: %w", err)
	}
	return nil
}

// validateSAMLAttributeMapping ensures the mapping, when present, is a JSON
// object of string→string. Empty/whitespace is allowed (defaults are used).
func validateSAMLAttributeMapping(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return fmt.Errorf("attribute_mapping must be a JSON object of string values: %w", err)
	}
	return nil
}

// CreateOrgSAMLConnection registers a per-org upstream SAML IdP (kind=sso_saml).
// Gated on the org being written (params.OrgID): a super-admin or an
// org-admin of that org. See constants.OrgRoleAdmin — the bare "admin" role is
// not accepted and existing bare-admin memberships are never auto-promoted.
func (p *provider) CreateOrgSAMLConnection(ctx context.Context, meta RequestMetadata, params *model.CreateOrgSAMLConnectionRequest) (*model.OrgSAMLConnection, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "CreateOrgSAMLConnection").Logger()
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}

	orgID := strings.TrimSpace(params.OrgID)
	name := strings.TrimSpace(params.Name)
	idpEntityID := strings.TrimSpace(params.IdpEntityID)
	idpSSOURL := strings.TrimSpace(params.IdpSsoURL)
	idpCert := strings.TrimSpace(params.IdpCertificate)
	if orgID == "" || name == "" || idpEntityID == "" || idpSSOURL == "" || idpCert == "" {
		return nil, nil, fmt.Errorf("org_id, name, idp_entity_id, idp_sso_url and idp_certificate are required")
	}
	if err := validateSAMLHTTPSURL(idpSSOURL); err != nil {
		return nil, nil, fmt.Errorf("idp_sso_url %w", err)
	}
	if err := validateSAMLCertPEM(idpCert); err != nil {
		return nil, nil, err
	}
	spEntityID, acsURL, attrMap, err := validateOptionalSAMLFields(params.SpEntityID, params.AcsURL, params.AttributeMapping)
	if err != nil {
		return nil, nil, err
	}

	// The organization must exist.
	if _, err := p.StorageProvider.GetOrganizationByID(ctx, orgID); err != nil {
		log.Debug().Err(err).Str("org_id", orgID).Msg("organization not found")
		return nil, nil, fmt.Errorf("organization not found: %s", orgID)
	}
	// At most one SAML connection per org.
	if existing, _ := p.StorageProvider.GetTrustedIssuerByOrgIDAndKind(ctx, orgID, constants.TrustKindSSOSAML); existing != nil {
		return nil, nil, fmt.Errorf("a SAML connection already exists for this organization")
	}
	// idp_entity_id is stored in the globally-unique IssuerURL column — this also
	// prevents a SAML row from shadowing an OIDC/client_assertion_trust row at the
	// same issuer value, and vice-versa.
	if existing, _ := p.StorageProvider.GetTrustedIssuerByIssuerURL(ctx, idpEntityID); existing != nil {
		return nil, nil, fmt.Errorf("idp_entity_id already registered: %s", idpEntityID)
	}

	allowIDPInitiated := params.AllowIdpInitiated != nil && *params.AllowIdpInitiated

	issuer, err := p.StorageProvider.AddTrustedIssuer(ctx, &schemas.TrustedIssuer{
		Kind:                  constants.TrustKindSSOSAML,
		OrgID:                 orgID,
		Name:                  name,
		IssuerURL:             idpEntityID,
		KeySourceType:         "saml_idp_certificate",
		IssuerType:            "saml",
		AuthMethod:            "saml_assertion",
		SAMLSSOURL:            refs.NewStringRef(idpSSOURL),
		SAMLIDPCertPEM:        refs.NewStringRef(idpCert),
		SAMLSPEntityID:        spEntityID,
		SAMLACSURL:            acsURL,
		SAMLAttributeMapping:  attrMap,
		SAMLAllowIDPInitiated: allowIDPInitiated,
		IsActive:              true,
	})
	if err != nil {
		log.Debug().Err(err).Msg("failed AddTrustedIssuer (sso_saml)")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditOrgSAMLConnectionCreatedEvent,
		Protocol:     meta.Protocol,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgSAMLConnection,
		ResourceID:   issuer.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return asAPIOrgSAMLConnection(issuer), nil, nil
}

// validateOptionalSAMLFields validates and normalises the optional SP entity ID,
// ACS URL and attribute mapping. Returns nil pointers for omitted fields.
func validateOptionalSAMLFields(spEntityID, acsURL, attrMap *string) (*string, *string, *string, error) {
	var outSP, outACS, outAttr *string
	if spEntityID != nil {
		v := strings.TrimSpace(*spEntityID)
		if v != "" {
			if err := validateSAMLHTTPSURL(v); err != nil {
				return nil, nil, nil, fmt.Errorf("sp_entity_id %w", err)
			}
			outSP = refs.NewStringRef(v)
		}
	}
	if acsURL != nil {
		v := strings.TrimSpace(*acsURL)
		if v != "" {
			if err := validateSAMLHTTPSURL(v); err != nil {
				return nil, nil, nil, fmt.Errorf("acs_url %w", err)
			}
			outACS = refs.NewStringRef(v)
		}
	}
	if attrMap != nil {
		v := strings.TrimSpace(*attrMap)
		if v != "" {
			if err := validateSAMLAttributeMapping(v); err != nil {
				return nil, nil, nil, err
			}
			outAttr = refs.NewStringRef(v)
		}
	}
	return outSP, outACS, outAttr, nil
}

// UpdateOrgSAMLConnection mutates only the fields present in params (load-then-
// mutate). Kind and OrgID are immutable. Supplying idp_certificate replaces it.
// Gated on the loaded row's OrgID (super-admin or that org's org-admin): the
// connection is loaded by id FIRST so authorization keys on its real OrgID, not
// on any caller-supplied org id (design H2, confused-deputy fix).
func (p *provider) UpdateOrgSAMLConnection(ctx context.Context, meta RequestMetadata, params *model.UpdateOrgSAMLConnectionRequest) (*model.OrgSAMLConnection, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "UpdateOrgSAMLConnection").Logger()
	issuer, err := p.StorageProvider.GetTrustedIssuerByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetTrustedIssuerByID")
		return nil, nil, err
	}
	// Guard: this op only edits sso_saml rows — never a client_assertion row.
	if issuer.EffectiveKind() != constants.TrustKindSSOSAML {
		return nil, nil, fmt.Errorf("not a SAML connection")
	}
	if err := p.requireOrgAdmin(ctx, meta, issuer.OrgID); err != nil {
		return nil, nil, err
	}

	if params.Name != nil {
		issuer.Name = strings.TrimSpace(*params.Name)
	}
	if params.IdpEntityID != nil {
		v := strings.TrimSpace(*params.IdpEntityID)
		if v == "" {
			return nil, nil, fmt.Errorf("idp_entity_id cannot be empty")
		}
		// Preserve global issuer_url (idp_entity_id) uniqueness on change.
		if v != issuer.IssuerURL {
			if existing, _ := p.StorageProvider.GetTrustedIssuerByIssuerURL(ctx, v); existing != nil {
				return nil, nil, fmt.Errorf("idp_entity_id already registered: %s", v)
			}
		}
		issuer.IssuerURL = v
	}
	if params.IdpSsoURL != nil {
		v := strings.TrimSpace(*params.IdpSsoURL)
		if err := validateSAMLHTTPSURL(v); err != nil {
			return nil, nil, fmt.Errorf("idp_sso_url %w", err)
		}
		issuer.SAMLSSOURL = refs.NewStringRef(v)
	}
	if params.IdpCertificate != nil && strings.TrimSpace(*params.IdpCertificate) != "" {
		v := strings.TrimSpace(*params.IdpCertificate)
		if err := validateSAMLCertPEM(v); err != nil {
			return nil, nil, err
		}
		issuer.SAMLIDPCertPEM = refs.NewStringRef(v)
	}
	if params.SpEntityID != nil {
		v := strings.TrimSpace(*params.SpEntityID)
		if v != "" {
			if err := validateSAMLHTTPSURL(v); err != nil {
				return nil, nil, fmt.Errorf("sp_entity_id %w", err)
			}
			issuer.SAMLSPEntityID = refs.NewStringRef(v)
		} else {
			issuer.SAMLSPEntityID = nil
		}
	}
	if params.AcsURL != nil {
		v := strings.TrimSpace(*params.AcsURL)
		if v != "" {
			if err := validateSAMLHTTPSURL(v); err != nil {
				return nil, nil, fmt.Errorf("acs_url %w", err)
			}
			issuer.SAMLACSURL = refs.NewStringRef(v)
		} else {
			issuer.SAMLACSURL = nil
		}
	}
	if params.AttributeMapping != nil {
		v := strings.TrimSpace(*params.AttributeMapping)
		if v != "" {
			if err := validateSAMLAttributeMapping(v); err != nil {
				return nil, nil, err
			}
			issuer.SAMLAttributeMapping = refs.NewStringRef(v)
		} else {
			issuer.SAMLAttributeMapping = nil
		}
	}
	if params.AllowIdpInitiated != nil {
		issuer.SAMLAllowIDPInitiated = *params.AllowIdpInitiated
	}
	if params.IsActive != nil {
		issuer.IsActive = *params.IsActive
	}

	updated, err := p.StorageProvider.UpdateTrustedIssuer(ctx, issuer)
	if err != nil {
		log.Debug().Err(err).Msg("failed UpdateTrustedIssuer (sso_saml)")
		return nil, nil, err
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditOrgSAMLConnectionUpdatedEvent,
		Protocol:     meta.Protocol,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgSAMLConnection,
		ResourceID:   updated.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return asAPIOrgSAMLConnection(updated), nil, nil
}

// resolveOrgSAMLConnection loads the connection by id or org_id (exactly one).
func (p *provider) resolveOrgSAMLConnection(ctx context.Context, id, orgID *string) (*schemas.TrustedIssuer, error) {
	switch {
	case id != nil && strings.TrimSpace(*id) != "":
		issuer, err := p.StorageProvider.GetTrustedIssuerByID(ctx, strings.TrimSpace(*id))
		if err != nil {
			return nil, err
		}
		if issuer.EffectiveKind() != constants.TrustKindSSOSAML {
			return nil, fmt.Errorf("not a SAML connection")
		}
		return issuer, nil
	case orgID != nil && strings.TrimSpace(*orgID) != "":
		return p.StorageProvider.GetTrustedIssuerByOrgIDAndKind(ctx, strings.TrimSpace(*orgID), constants.TrustKindSSOSAML)
	default:
		return nil, fmt.Errorf("supply either id or org_id")
	}
}

// DeleteOrgSAMLConnection removes an org's SAML connection. Gated on the loaded
// row's OrgID (super-admin or that org's org-admin): the connection is resolved
// FIRST so authorization keys on its real OrgID, then a caller-supplied org_id
// that names a different org is rejected (design H2, confused-deputy fix).
func (p *provider) DeleteOrgSAMLConnection(ctx context.Context, meta RequestMetadata, params *model.OrgSAMLConnectionRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "DeleteOrgSAMLConnection").Logger()
	issuer, err := p.resolveOrgSAMLConnection(ctx, params.ID, params.OrgID)
	if err != nil {
		log.Debug().Err(err).Msg("failed to resolve SAML connection")
		return nil, nil, err
	}
	if err := p.requireOrgAdmin(ctx, meta, issuer.OrgID); err != nil {
		return nil, nil, err
	}
	if err := rejectOrgIDMismatch(params.OrgID, issuer.OrgID); err != nil {
		return nil, nil, err
	}
	if err := p.StorageProvider.DeleteTrustedIssuer(ctx, issuer); err != nil {
		log.Debug().Err(err).Msg("failed DeleteTrustedIssuer (sso_saml)")
		return nil, nil, err
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditOrgSAMLConnectionDeletedEvent,
		Protocol:     meta.Protocol,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgSAMLConnection,
		ResourceID:   issuer.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return &model.Response{Message: "SAML connection deleted"}, nil, nil
}

// OrgSAMLConnection fetches an org's SAML connection by id or org_id. Gated on
// the loaded row's OrgID (super-admin or that org's org-admin, design H2). The
// IdP certificate is never projected.
func (p *provider) OrgSAMLConnection(ctx context.Context, meta RequestMetadata, params *model.OrgSAMLConnectionRequest) (*model.OrgSAMLConnection, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "OrgSAMLConnection").Logger()
	issuer, err := p.resolveOrgSAMLConnection(ctx, params.ID, params.OrgID)
	if err != nil {
		log.Debug().Err(err).Msg("failed to resolve SAML connection")
		return nil, nil, err
	}
	if err := p.requireOrgAdmin(ctx, meta, issuer.OrgID); err != nil {
		return nil, nil, err
	}
	if err := rejectOrgIDMismatch(params.OrgID, issuer.OrgID); err != nil {
		return nil, nil, err
	}
	return asAPIOrgSAMLConnection(issuer), nil, nil
}
