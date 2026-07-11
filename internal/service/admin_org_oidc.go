package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// defaultSSOScopes is requested at the upstream IdP when the admin omits scopes.
const defaultSSOScopes = "openid profile email"

// asAPIOrgOIDCConnection projects a sso_oidc TrustedIssuer row onto the GraphQL
// model. The upstream client secret (SSOClientSecretEnc) is intentionally NEVER
// surfaced — the model has no secret field.
func asAPIOrgOIDCConnection(t *schemas.TrustedIssuer) *model.OrgOIDCConnection {
	id := t.ID
	if strings.Contains(id, schemas.Collections.TrustedIssuer+"/") {
		id = strings.TrimPrefix(id, schemas.Collections.TrustedIssuer+"/")
	}
	return &model.OrgOIDCConnection{
		ID:          id,
		OrgID:       t.OrgID,
		Name:        t.Name,
		IssuerURL:   t.IssuerURL,
		SsoClientID: t.SSOClientID,
		Scopes:      refs.NewStringRef(t.SSOScopes),
		RedirectURI: refs.NewStringRef(t.SSORedirectURI),
		IsActive:    t.IsActive,
		CreatedAt:   refs.NewInt64Ref(t.CreatedAt),
		UpdatedAt:   refs.NewInt64Ref(t.UpdatedAt),
	}
}

// validateSSOIssuerURL rejects an obviously invalid upstream issuer URL. The
// actual network fetches (discovery/JWKS/token) are SSRF-hardened at fetch time
// via validators.SafeHTTPClient; this only rejects non-https/opaque input early.
func validateSSOIssuerURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("issuer_url must be a valid https URL")
	}
	return nil
}

// CreateOrgOIDCConnection registers a per-org upstream OIDC IdP (kind=sso_oidc).
// Gated on the org being written (params.OrgID): a super-admin or an org-admin
// of that org. See constants.OrgRoleAdmin.
func (p *provider) CreateOrgOIDCConnection(ctx context.Context, meta RequestMetadata, params *model.CreateOrgOIDCConnectionRequest) (*model.OrgOIDCConnection, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "CreateOrgOIDCConnection").Logger()
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}

	orgID := strings.TrimSpace(params.OrgID)
	name := strings.TrimSpace(params.Name)
	issuerURL := strings.TrimSpace(params.IssuerURL)
	clientID := strings.TrimSpace(params.ClientID)
	clientSecret := params.ClientSecret // preserve verbatim (secret may be non-trimmable)
	if orgID == "" || name == "" || issuerURL == "" || clientID == "" || strings.TrimSpace(clientSecret) == "" {
		return nil, nil, fmt.Errorf("org_id, name, issuer_url, client_id and client_secret are required")
	}
	if err := validateSSOIssuerURL(issuerURL); err != nil {
		return nil, nil, err
	}

	// The organization must exist.
	if _, err := p.StorageProvider.GetOrganizationByID(ctx, orgID); err != nil {
		log.Debug().Err(err).Str("org_id", orgID).Msg("organization not found")
		return nil, nil, fmt.Errorf("organization not found: %s", orgID)
	}
	// At most one OIDC connection per org.
	if existing, _ := p.StorageProvider.GetTrustedIssuerByOrgIDAndKind(ctx, orgID, constants.TrustKindSSOOIDC); existing != nil {
		return nil, nil, fmt.Errorf("an OIDC connection already exists for this organization")
	}
	// issuer_url is globally unique (DB unique index + service guard) — this also
	// prevents an SSO row from shadowing a client_assertion_trust row at the same
	// URL, and vice-versa.
	if existing, _ := p.StorageProvider.GetTrustedIssuerByIssuerURL(ctx, issuerURL); existing != nil {
		return nil, nil, fmt.Errorf("issuer_url already registered: %s", issuerURL)
	}

	// The upstream secret is stored AES-encrypted (reversible: replayed to the
	// upstream token endpoint) keyed on Config.ClientSecret, and carries json:"-".
	encSecret, err := crypto.EncryptAES(p.Config.ClientSecret, clientSecret)
	if err != nil {
		log.Debug().Err(err).Msg("failed to encrypt upstream client secret")
		return nil, nil, fmt.Errorf("failed to store connection")
	}

	scopes := defaultSSOScopes
	if params.Scopes != nil && strings.TrimSpace(*params.Scopes) != "" {
		scopes = strings.TrimSpace(*params.Scopes)
	}
	redirectURI := ""
	if params.RedirectURI != nil {
		redirectURI = strings.TrimSpace(*params.RedirectURI)
	}

	issuer, err := p.StorageProvider.AddTrustedIssuer(ctx, &schemas.TrustedIssuer{
		Kind:               constants.TrustKindSSOOIDC,
		OrgID:              orgID,
		Name:               name,
		IssuerURL:          issuerURL,
		KeySourceType:      constants.KeySourceOIDCDiscovery,
		IssuerType:         "oidc",
		AuthMethod:         "jwt_assertion",
		SSOClientID:        clientID,
		SSOClientSecretEnc: encSecret,
		SSOScopes:          scopes,
		SSORedirectURI:     redirectURI,
		IsActive:           true,
	})
	if err != nil {
		log.Debug().Err(err).Msg("failed AddTrustedIssuer (sso_oidc)")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditOrgOIDCConnectionCreatedEvent,
		Protocol:     meta.Protocol,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgOIDCConnection,
		ResourceID:   issuer.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return asAPIOrgOIDCConnection(issuer), nil, nil
}

// UpdateOrgOIDCConnection mutates only the fields present in params (load-then-
// mutate). Kind and OrgID are immutable. Supplying client_secret rotates it.
// Gated on the loaded row's OrgID (super-admin or that org's org-admin): the
// connection is loaded by id FIRST so authorization keys on its real OrgID, not
// on any caller-supplied org id (design H2, confused-deputy fix).
func (p *provider) UpdateOrgOIDCConnection(ctx context.Context, meta RequestMetadata, params *model.UpdateOrgOIDCConnectionRequest) (*model.OrgOIDCConnection, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "UpdateOrgOIDCConnection").Logger()
	issuer, err := p.StorageProvider.GetTrustedIssuerByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetTrustedIssuerByID")
		return nil, nil, err
	}
	// Guard: this op only edits sso_oidc rows — never a client_assertion row.
	if issuer.EffectiveKind() != constants.TrustKindSSOOIDC {
		return nil, nil, fmt.Errorf("not an OIDC connection")
	}
	if err := p.requireOrgAdmin(ctx, meta, issuer.OrgID); err != nil {
		return nil, nil, err
	}

	if params.Name != nil {
		issuer.Name = strings.TrimSpace(*params.Name)
	}
	if params.IssuerURL != nil {
		u := strings.TrimSpace(*params.IssuerURL)
		if err := validateSSOIssuerURL(u); err != nil {
			return nil, nil, err
		}
		// Preserve global issuer_url uniqueness on change.
		if u != issuer.IssuerURL {
			if existing, _ := p.StorageProvider.GetTrustedIssuerByIssuerURL(ctx, u); existing != nil {
				return nil, nil, fmt.Errorf("issuer_url already registered: %s", u)
			}
		}
		issuer.IssuerURL = u
	}
	if params.ClientID != nil {
		issuer.SSOClientID = strings.TrimSpace(*params.ClientID)
	}
	if params.ClientSecret != nil && strings.TrimSpace(*params.ClientSecret) != "" {
		enc, err := crypto.EncryptAES(p.Config.ClientSecret, *params.ClientSecret)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to store connection")
		}
		issuer.SSOClientSecretEnc = enc
	}
	if params.Scopes != nil && strings.TrimSpace(*params.Scopes) != "" {
		issuer.SSOScopes = strings.TrimSpace(*params.Scopes)
	}
	if params.RedirectURI != nil {
		issuer.SSORedirectURI = strings.TrimSpace(*params.RedirectURI)
	}
	if params.IsActive != nil {
		issuer.IsActive = *params.IsActive
	}

	updated, err := p.StorageProvider.UpdateTrustedIssuer(ctx, issuer)
	if err != nil {
		log.Debug().Err(err).Msg("failed UpdateTrustedIssuer (sso_oidc)")
		return nil, nil, err
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditOrgOIDCConnectionUpdatedEvent,
		Protocol:     meta.Protocol,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgOIDCConnection,
		ResourceID:   updated.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return asAPIOrgOIDCConnection(updated), nil, nil
}

// resolveOrgOIDCConnection loads the connection by id or org_id (exactly one).
func (p *provider) resolveOrgOIDCConnection(ctx context.Context, id, orgID *string) (*schemas.TrustedIssuer, error) {
	switch {
	case id != nil && strings.TrimSpace(*id) != "":
		issuer, err := p.StorageProvider.GetTrustedIssuerByID(ctx, strings.TrimSpace(*id))
		if err != nil {
			return nil, err
		}
		if issuer.EffectiveKind() != constants.TrustKindSSOOIDC {
			return nil, fmt.Errorf("not an OIDC connection")
		}
		return issuer, nil
	case orgID != nil && strings.TrimSpace(*orgID) != "":
		return p.StorageProvider.GetTrustedIssuerByOrgIDAndKind(ctx, strings.TrimSpace(*orgID), constants.TrustKindSSOOIDC)
	default:
		return nil, fmt.Errorf("supply either id or org_id")
	}
}

// DeleteOrgOIDCConnection removes an org's OIDC connection. Gated on the loaded
// row's OrgID (super-admin or that org's org-admin): resolved FIRST so
// authorization keys on its real OrgID, then a caller-supplied org_id that names
// a different org is rejected (design H2, confused-deputy fix).
func (p *provider) DeleteOrgOIDCConnection(ctx context.Context, meta RequestMetadata, params *model.OrgOIDCConnectionRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "DeleteOrgOIDCConnection").Logger()
	issuer, err := p.resolveOrgOIDCConnection(ctx, params.ID, params.OrgID)
	if err != nil {
		log.Debug().Err(err).Msg("failed to resolve OIDC connection")
		return nil, nil, err
	}
	if err := p.requireOrgAdmin(ctx, meta, issuer.OrgID); err != nil {
		return nil, nil, err
	}
	if err := rejectOrgIDMismatch(params.OrgID, issuer.OrgID); err != nil {
		return nil, nil, err
	}
	if err := p.StorageProvider.DeleteTrustedIssuer(ctx, issuer); err != nil {
		log.Debug().Err(err).Msg("failed DeleteTrustedIssuer (sso_oidc)")
		return nil, nil, err
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditOrgOIDCConnectionDeletedEvent,
		Protocol:     meta.Protocol,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgOIDCConnection,
		ResourceID:   issuer.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return &model.Response{Message: "OIDC connection deleted"}, nil, nil
}

// OrgOIDCConnection fetches an org's OIDC connection by id or org_id. Gated on
// the loaded row's OrgID (super-admin or that org's org-admin, design H2). The
// secret is never projected.
func (p *provider) OrgOIDCConnection(ctx context.Context, meta RequestMetadata, params *model.OrgOIDCConnectionRequest) (*model.OrgOIDCConnection, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "OrgOIDCConnection").Logger()
	issuer, err := p.resolveOrgOIDCConnection(ctx, params.ID, params.OrgID)
	if err != nil {
		log.Debug().Err(err).Msg("failed to resolve OIDC connection")
		return nil, nil, err
	}
	if err := p.requireOrgAdmin(ctx, meta, issuer.OrgID); err != nil {
		return nil, nil, err
	}
	if err := rejectOrgIDMismatch(params.OrgID, issuer.OrgID); err != nil {
		return nil, nil, err
	}
	return asAPIOrgOIDCConnection(issuer), nil, nil
}
