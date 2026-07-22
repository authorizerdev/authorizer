package service

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
	"strings"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// asAPISAMLIDPKey projects a signing-key row onto the GraphQL model. The private
// key is NEVER surfaced.
func asAPISAMLIDPKey(k *schemas.SAMLIDPKey) *model.SAMLIDPKey {
	id := k.ID
	if strings.Contains(id, schemas.Collections.SAMLIDPKey+"/") {
		id = strings.TrimPrefix(id, schemas.Collections.SAMLIDPKey+"/")
	}
	return &model.SAMLIDPKey{
		ID:        id,
		OrgID:     k.OrgID,
		CertPem:   k.CertPEM,
		Algorithm: k.Algorithm,
		Status:    k.Status,
		CreatedAt: refs.NewInt64Ref(k.CreatedAt),
		UpdatedAt: refs.NewInt64Ref(k.UpdatedAt),
	}
}

// validateSAMLACSURL rejects a non-http(s) or hostless ACS URL. http is permitted
// (local/testing SPs); https is expected in production.
func validateSAMLACSURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return InvalidArgument("acs_url must be a valid http(s) URL")
	}
	return nil
}

// CreateSAMLServiceProvider registers a downstream SP (Authorizer as IdP).
//
// Permissions: super-admin or org-admin of params.OrgID.
func (p *provider) CreateSAMLServiceProvider(ctx context.Context, meta RequestMetadata, params *model.CreateSAMLServiceProviderRequest) (*model.SAMLServiceProvider, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "CreateSAMLServiceProvider").Logger()
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}
	orgID := strings.TrimSpace(params.OrgID)
	name := strings.TrimSpace(params.Name)
	entityID := strings.TrimSpace(params.EntityID)
	acsURL := strings.TrimSpace(params.AcsURL)
	if orgID == "" || name == "" || entityID == "" || acsURL == "" {
		return nil, nil, InvalidArgument("org_id, name, entity_id and acs_url are required")
	}
	if err := validateSAMLACSURL(acsURL); err != nil {
		return nil, nil, err
	}
	if params.SpCertPem != nil && strings.TrimSpace(*params.SpCertPem) != "" {
		if err := validateSAMLCertPEM(*params.SpCertPem); err != nil {
			return nil, nil, fmt.Errorf("sp_cert_pem is not a valid X.509 certificate: %w", err)
		}
	}
	if params.MappedAttributes != nil {
		if err := validateSAMLAttributeMapping(*params.MappedAttributes); err != nil {
			return nil, nil, err
		}
	}
	if _, err := p.StorageProvider.GetOrganizationByID(ctx, orgID); err != nil {
		return nil, nil, NotFound(fmt.Sprintf("organization not found: %s", orgID))
	}
	if existing, _ := p.StorageProvider.GetSAMLServiceProviderByOrgAndEntityID(ctx, orgID, entityID); existing != nil {
		return nil, nil, AlreadyExists(fmt.Sprintf("a service provider with entity_id %q is already registered for this organization", entityID))
	}

	sp := &schemas.SAMLServiceProvider{
		OrgID:             orgID,
		Name:              name,
		EntityID:          entityID,
		ACSURL:            acsURL,
		SPCertPEM:         trimmedRef(params.SpCertPem),
		NameIDFormat:      strings.TrimSpace(refs.StringValue(params.NameIDFormat)),
		MappedAttributes:  trimmedRef(params.MappedAttributes),
		AllowIDPInitiated: params.AllowIdpInitiated != nil && *params.AllowIdpInitiated,
		IsActive:          true,
	}
	saved, err := p.StorageProvider.AddSAMLServiceProvider(ctx, sp)
	if err != nil {
		log.Debug().Err(err).Msg("failed AddSAMLServiceProvider")
		return nil, nil, err
	}
	p.auditSAMLIDP(meta, constants.AuditSAMLIDPServiceProviderChangedEvent, saved.ID, "created")
	return saved.AsAPISAMLServiceProvider(), nil, nil
}

// UpdateSAMLServiceProvider mutates the provided fields of a registered SP.
//
// Permissions: super-admin or org-admin of the SP's org.
func (p *provider) UpdateSAMLServiceProvider(ctx context.Context, meta RequestMetadata, params *model.UpdateSAMLServiceProviderRequest) (*model.SAMLServiceProvider, *ResponseSideEffects, error) {
	id := strings.TrimSpace(params.ID)
	if id == "" {
		return nil, nil, InvalidArgument("id is required")
	}
	existing, err := p.StorageProvider.GetSAMLServiceProviderByID(ctx, id)
	if err != nil || existing == nil {
		return nil, nil, NotFound(fmt.Sprintf("service provider not found: %s", id))
	}
	if err := p.requireOrgAdmin(ctx, meta, existing.OrgID); err != nil {
		return nil, nil, err
	}

	if params.Name != nil {
		if v := strings.TrimSpace(*params.Name); v != "" {
			existing.Name = v
		}
	}
	if params.EntityID != nil {
		v := strings.TrimSpace(*params.EntityID)
		if v != "" && v != existing.EntityID {
			if conflict, _ := p.StorageProvider.GetSAMLServiceProviderByOrgAndEntityID(ctx, existing.OrgID, v); conflict != nil && conflict.ID != existing.ID {
				return nil, nil, AlreadyExists(fmt.Sprintf("a service provider with entity_id %q is already registered for this organization", v))
			}
			existing.EntityID = v
		}
	}
	if params.AcsURL != nil {
		v := strings.TrimSpace(*params.AcsURL)
		if v != "" {
			if err := validateSAMLACSURL(v); err != nil {
				return nil, nil, err
			}
			existing.ACSURL = v
		}
	}
	if params.SpCertPem != nil {
		v := strings.TrimSpace(*params.SpCertPem)
		if v != "" {
			if err := validateSAMLCertPEM(v); err != nil {
				return nil, nil, fmt.Errorf("sp_cert_pem is not a valid X.509 certificate: %w", err)
			}
			existing.SPCertPEM = refs.NewStringRef(v)
		} else {
			existing.SPCertPEM = nil
		}
	}
	if params.NameIDFormat != nil {
		existing.NameIDFormat = strings.TrimSpace(*params.NameIDFormat)
	}
	if params.MappedAttributes != nil {
		v := strings.TrimSpace(*params.MappedAttributes)
		if v != "" {
			if err := validateSAMLAttributeMapping(v); err != nil {
				return nil, nil, err
			}
			existing.MappedAttributes = refs.NewStringRef(v)
		} else {
			existing.MappedAttributes = nil
		}
	}
	if params.AllowIdpInitiated != nil {
		existing.AllowIDPInitiated = *params.AllowIdpInitiated
	}
	if params.IsActive != nil {
		existing.IsActive = *params.IsActive
	}

	saved, err := p.StorageProvider.UpdateSAMLServiceProvider(ctx, existing)
	if err != nil {
		return nil, nil, err
	}
	p.auditSAMLIDP(meta, constants.AuditSAMLIDPServiceProviderChangedEvent, saved.ID, "updated")
	return saved.AsAPISAMLServiceProvider(), nil, nil
}

// DeleteSAMLServiceProvider removes a registered SP.
//
// Permissions: super-admin or org-admin of the SP's org.
func (p *provider) DeleteSAMLServiceProvider(ctx context.Context, meta RequestMetadata, params *model.SAMLServiceProviderRequest) (*model.Response, *ResponseSideEffects, error) {
	existing, err := p.StorageProvider.GetSAMLServiceProviderByID(ctx, strings.TrimSpace(params.ID))
	if err != nil || existing == nil {
		return nil, nil, NotFound(fmt.Sprintf("service provider not found: %s", params.ID))
	}
	if err := p.requireOrgAdmin(ctx, meta, existing.OrgID); err != nil {
		return nil, nil, err
	}
	if err := p.StorageProvider.DeleteSAMLServiceProvider(ctx, existing); err != nil {
		return nil, nil, err
	}
	p.auditSAMLIDP(meta, constants.AuditSAMLIDPServiceProviderChangedEvent, existing.ID, "deleted")
	return &model.Response{Message: "service provider deleted"}, nil, nil
}

// SAMLServiceProvider returns a registered SP by id.
//
// Permissions: super-admin or org-admin of the SP's org.
func (p *provider) SAMLServiceProvider(ctx context.Context, meta RequestMetadata, params *model.SAMLServiceProviderRequest) (*model.SAMLServiceProvider, *ResponseSideEffects, error) {
	existing, err := p.StorageProvider.GetSAMLServiceProviderByID(ctx, strings.TrimSpace(params.ID))
	if err != nil || existing == nil {
		return nil, nil, NotFound(fmt.Sprintf("service provider not found: %s", params.ID))
	}
	if err := p.requireOrgAdmin(ctx, meta, existing.OrgID); err != nil {
		return nil, nil, err
	}
	return existing.AsAPISAMLServiceProvider(), nil, nil
}

// ListSAMLServiceProviders returns an org's registered SPs (paginated).
//
// Permissions: super-admin or org-admin of params.OrgID.
func (p *provider) ListSAMLServiceProviders(ctx context.Context, meta RequestMetadata, params *model.ListSAMLServiceProvidersRequest) (*model.SAMLServiceProviders, *ResponseSideEffects, error) {
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}
	pagination := utils.GetPagination(params.Pagination)
	rows, page, err := p.StorageProvider.ListSAMLServiceProviders(ctx, strings.TrimSpace(params.OrgID), pagination)
	if err != nil {
		return nil, nil, err
	}
	out := make([]*model.SAMLServiceProvider, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.AsAPISAMLServiceProvider())
	}
	return &model.SAMLServiceProviders{Pagination: page, SamlServiceProviders: out}, nil, nil
}

// RotateSAMLIDPCert generates a new "current" signing keypair; the previously
// current key becomes "active" (still published in metadata) until retired.
//
// Permissions: super-admin or org-admin of params.OrgID.
func (p *provider) RotateSAMLIDPCert(ctx context.Context, meta RequestMetadata, params *model.RotateSAMLIDPCertRequest) (*model.SAMLIDPKey, *ResponseSideEffects, error) {
	orgID := strings.TrimSpace(params.OrgID)
	if err := p.requireOrgAdmin(ctx, meta, orgID); err != nil {
		return nil, nil, err
	}
	if _, err := p.StorageProvider.GetOrganizationByID(ctx, orgID); err != nil {
		return nil, nil, NotFound(fmt.Sprintf("organization not found: %s", orgID))
	}
	keys, err := p.StorageProvider.ListSAMLIDPKeys(ctx, orgID)
	if err != nil {
		return nil, nil, err
	}
	privPEM, certPEM, err := crypto.NewSAMLSigningKeypair("Authorizer SAML IdP " + orgID)
	if err != nil {
		return nil, nil, err
	}
	enc, err := crypto.EncryptAES(p.ClientSecret, privPEM)
	if err != nil {
		return nil, nil, err
	}
	// Demote the previously-current key(s) to "active" BEFORE inserting the new
	// current, so there is never more than one "current" key. A demotion failure
	// is fatal (returned) rather than logged-and-ignored. If the insert below then
	// fails the org is momentarily left with no current key — self-healing, since
	// the next issuance lazily regenerates one; strictly safer than two currents.
	for _, k := range keys {
		if k.Status == schemas.SAMLIDPKeyStatusCurrent {
			k.Status = schemas.SAMLIDPKeyStatusActive
			if _, err := p.StorageProvider.UpdateSAMLIDPKey(ctx, k); err != nil {
				return nil, nil, fmt.Errorf("failed to demote previous signing key: %w", err)
			}
		}
	}
	newKey, err := p.StorageProvider.AddSAMLIDPKey(ctx, &schemas.SAMLIDPKey{
		OrgID:         orgID,
		CertPEM:       certPEM,
		PrivateKeyEnc: enc,
		Algorithm:     "RS256",
		Status:        schemas.SAMLIDPKeyStatusCurrent,
	})
	if err != nil {
		return nil, nil, err
	}
	p.auditSAMLIDP(meta, constants.AuditSAMLIDPKeyRotatedEvent, newKey.ID, orgID)
	return asAPISAMLIDPKey(newKey), nil, nil
}

// RetireSAMLIDPKey retires a superseded ("active") key so it stops appearing in
// IdP metadata. The current key cannot be retired.
//
// Permissions: super-admin or org-admin of the key's org.
func (p *provider) RetireSAMLIDPKey(ctx context.Context, meta RequestMetadata, params *model.RetireSAMLIDPKeyRequest) (*model.Response, *ResponseSideEffects, error) {
	key, err := p.StorageProvider.GetSAMLIDPKeyByID(ctx, strings.TrimSpace(params.ID))
	if err != nil || key == nil {
		return nil, nil, NotFound(fmt.Sprintf("signing key not found: %s", params.ID))
	}
	if err := p.requireOrgAdmin(ctx, meta, key.OrgID); err != nil {
		return nil, nil, err
	}
	if key.Status == schemas.SAMLIDPKeyStatusCurrent {
		return nil, nil, FailedPrecondition("cannot retire the current signing key; rotate to a new key first")
	}
	if key.Status == schemas.SAMLIDPKeyStatusRetired {
		return &model.Response{Message: "signing key already retired"}, nil, nil
	}
	key.Status = schemas.SAMLIDPKeyStatusRetired
	if _, err := p.StorageProvider.UpdateSAMLIDPKey(ctx, key); err != nil {
		return nil, nil, err
	}
	p.auditSAMLIDP(meta, constants.AuditSAMLIDPKeyRetiredEvent, key.ID, key.OrgID)
	return &model.Response{Message: "signing key retired"}, nil, nil
}

// ListSAMLIDPKeys returns an org's signing keys (current, active, retired).
//
// Permissions: super-admin or org-admin of params.OrgID.
func (p *provider) ListSAMLIDPKeys(ctx context.Context, meta RequestMetadata, params *model.ListSAMLIDPKeysRequest) ([]*model.SAMLIDPKey, *ResponseSideEffects, error) {
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}
	keys, err := p.StorageProvider.ListSAMLIDPKeys(ctx, strings.TrimSpace(params.OrgID))
	if err != nil {
		return nil, nil, err
	}
	out := make([]*model.SAMLIDPKey, 0, len(keys))
	for _, k := range keys {
		out = append(out, asAPISAMLIDPKey(k))
	}
	return out, nil, nil
}

// ImportSAMLSPMetadata parses pasted SP metadata XML and returns the extracted
// entity_id / acs_url / certificate. It does NOT create a record.
//
// Security: parsing uses crewjam/samlsp.ParseMetadata over Go's encoding/xml,
// which ignores DTDs and does not resolve external entities — no XXE surface.
//
// Permissions: super-admin (no org context).
func (p *provider) ImportSAMLSPMetadata(ctx context.Context, meta RequestMetadata, params *model.ImportSAMLSPMetadataRequest) (*model.SAMLSPMetadataParseResult, *ResponseSideEffects, error) {
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}
	raw := strings.TrimSpace(params.MetadataXML)
	if raw == "" {
		return nil, nil, InvalidArgument("metadata_xml is required")
	}
	ed, err := samlsp.ParseMetadata([]byte(raw))
	if err != nil {
		return nil, nil, InvalidArgument(fmt.Sprintf("could not parse SP metadata XML: %v", err))
	}
	acsURL := ""
	certPEM := ""
	for _, spd := range ed.SPSSODescriptors {
		for _, acs := range spd.AssertionConsumerServices {
			if acs.Binding == saml.HTTPPostBinding && acs.Location != "" {
				acsURL = acs.Location
				break
			}
		}
		if acsURL == "" && len(spd.AssertionConsumerServices) > 0 {
			acsURL = spd.AssertionConsumerServices[0].Location
		}
		if certPEM == "" {
			certPEM = firstSigningCertPEM(spd.KeyDescriptors)
		}
	}
	if ed.EntityID == "" || acsURL == "" {
		return nil, nil, InvalidArgument("metadata did not contain an SP entity ID and Assertion Consumer Service URL")
	}
	res := &model.SAMLSPMetadataParseResult{EntityID: ed.EntityID, AcsURL: acsURL}
	if certPEM != "" {
		res.Certificate = refs.NewStringRef(certPEM)
	}
	return res, nil, nil
}

// firstSigningCertPEM returns the first usable X.509 certificate from an SP's key
// descriptors as PEM, preferring the "signing" use.
func firstSigningCertPEM(kds []saml.KeyDescriptor) string {
	pick := func(uses ...string) string {
		for _, kd := range kds {
			match := len(uses) == 0
			for _, u := range uses {
				if kd.Use == u {
					match = true
				}
			}
			if !match {
				continue
			}
			for _, c := range kd.KeyInfo.X509Data.X509Certificates {
				if data := strings.TrimSpace(c.Data); data != "" {
					der, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(data), ""))
					if err != nil {
						continue
					}
					return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
				}
			}
		}
		return ""
	}
	if pem := pick("signing"); pem != "" {
		return pem
	}
	return pick()
}

// trimmedRef returns a trimmed *string, or nil when the input is nil/blank.
func trimmedRef(s *string) *string {
	if s == nil {
		return nil
	}
	if v := strings.TrimSpace(*s); v != "" {
		return refs.NewStringRef(v)
	}
	return nil
}

// auditSAMLIDP records an admin SAML-IdP configuration change.
func (p *provider) auditSAMLIDP(meta RequestMetadata, action, resourceID, detail string) {
	p.AuditProvider.LogEvent(audit.Event{
		Action:       action,
		Protocol:     meta.Protocol,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgSAMLConnection,
		ResourceID:   resourceID,
		Metadata:     detail,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
}
