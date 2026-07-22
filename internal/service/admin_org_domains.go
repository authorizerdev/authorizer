package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// errDomainOwnedByAnotherOrg is the uniform, stable API error when a domain is
// already verified by a different org (invariant 2). Kept as one string so the
// three write paths (verify, trusted-assert) return it identically.
var errDomainOwnedByAnotherOrg = AlreadyExists("domain_already_verified_by_another_org")

// RequestOrgDomain mints a DNS TXT challenge proving control of a domain and
// returns the record to publish. It creates NO durable row — the pending token
// lives only in the memory store with a ~24h TTL. Gated on params.OrgID
// (super-admin or that org's org-admin).
func (p *provider) RequestOrgDomain(ctx context.Context, meta RequestMetadata, params *model.RequestOrgDomainRequest) (*model.OrgDomainChallenge, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "RequestOrgDomain").Logger()
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}
	orgID := strings.TrimSpace(params.OrgID)
	if orgID == "" {
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainRequestFailedEvent, orgID)
		return nil, nil, InvalidArgument("org_id is required")
	}
	domain, err := normalizeDomain(params.Domain)
	if err != nil {
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainRequestFailedEvent, orgID)
		return nil, nil, err
	}
	if err := guardVerifiableDomain(domain); err != nil {
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainRequestFailedEvent, orgID)
		return nil, nil, err
	}
	// Rate-limit challenge minting per org, mirroring VerifyOrgDomain — bounds
	// memory-store churn from an authenticated tenant admin.
	if p.RateLimitProvider != nil {
		allowed, rlErr := p.RateLimitProvider.Allow(ctx, "org_domain_request:"+orgID)
		if rlErr == nil && !allowed {
			log.Debug().Msg("domain request rate limit exceeded")
			return nil, nil, TooManyRequests("too many domain requests, please retry later")
		}
	}
	if _, err := p.StorageProvider.GetOrganizationByID(ctx, orgID); err != nil {
		log.Debug().Err(err).Msg("organization not found")
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainRequestFailedEvent, orgID)
		return nil, nil, NotFound("organization not found")
	}

	token, err := generateDomainChallengeToken()
	if err != nil {
		log.Debug().Err(err).Msg("failed to generate domain challenge token")
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainRequestFailedEvent, orgID)
		return nil, nil, err
	}
	if err := p.MemoryStoreProvider.SetCache(challengeKey(orgID, domain), token, int64(domainChallengeTTL.Seconds())); err != nil {
		log.Debug().Err(err).Msg("failed to store domain challenge")
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainRequestFailedEvent, orgID)
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditOrgDomainRequestedEvent,
		Protocol:     meta.Protocol,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgDomain,
		ResourceID:   domain,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	// The token is returned to the caller (they publish it in DNS) but is NEVER
	// written to the audit trail.
	return &model.OrgDomainChallenge{
		Domain:      domain,
		RecordType:  "TXT",
		RecordName:  challengeRecordName(domain),
		RecordValue: challengeRecordValue(token),
	}, nil, nil
}

// VerifyOrgDomain proves control of a domain via its DNS TXT challenge and, on
// success, atomically inserts the verified row (first-writer-wins). Idempotent:
// a domain already verified by the same org returns success; one already
// verified by another org is rejected. Gated on params.OrgID and rate-limited
// per org (the verify drives an outbound DNS lookup).
func (p *provider) VerifyOrgDomain(ctx context.Context, meta RequestMetadata, params *model.VerifyOrgDomainRequest) (*model.OrgDomain, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "VerifyOrgDomain").Logger()
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}
	orgID := strings.TrimSpace(params.OrgID)
	domain, err := normalizeDomain(params.Domain)
	if err != nil {
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainVerifyFailedEvent, orgID)
		return nil, nil, err
	}
	if err := guardVerifiableDomain(domain); err != nil {
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainVerifyFailedEvent, orgID)
		return nil, nil, err
	}
	if p.RateLimitProvider != nil {
		allowed, rlErr := p.RateLimitProvider.Allow(ctx, "org_domain_verify:"+orgID)
		if rlErr == nil && !allowed {
			log.Debug().Msg("domain verify rate limit exceeded")
			return nil, nil, TooManyRequests("too many verification attempts, please retry later")
		}
	}

	// Idempotency + conflict pre-check. AddOrgDomain also enforces this
	// atomically; this fast-path keeps re-verify cheap and returns a clean error.
	if existing, _ := p.StorageProvider.GetOrgDomainByDomain(ctx, domain); existing != nil {
		if existing.OrgID == orgID {
			return existing.AsAPIOrgDomain(), nil, nil
		}
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainVerifyFailedEvent, orgID)
		return nil, nil, errDomainOwnedByAnotherOrg
	}

	token, err := p.MemoryStoreProvider.GetCache(challengeKey(orgID, domain))
	if err != nil || token == "" {
		log.Debug().Err(err).Msg("no pending domain challenge")
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainVerifyFailedEvent, orgID)
		return nil, nil, FailedPrecondition("no pending verification challenge for this domain; request one first")
	}

	matched, dnsErr := p.lookupDomainTXTMatches(ctx, domain, token)
	if dnsErr != nil {
		log.Debug().Err(dnsErr).Msg("domain TXT lookup failed")
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainVerifyFailedEvent, orgID)
		return nil, nil, FailedPrecondition("dns verification failed: could not resolve the challenge TXT record")
	}
	if !matched {
		// Leave the challenge in place so the tenant can retry after DNS propagates.
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainVerifyFailedEvent, orgID)
		return nil, nil, FailedPrecondition("dns verification failed: challenge TXT record not found or does not match")
	}

	row, err := p.StorageProvider.AddOrgDomain(ctx, &schemas.OrgDomain{
		ID:         domain,
		Domain:     domain,
		OrgID:      orgID,
		VerifiedAt: time.Now().Unix(),
	})
	if err != nil {
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainVerifyFailedEvent, orgID)
		if errors.Is(err, schemas.ErrOrgDomainConflict) {
			return nil, nil, errDomainOwnedByAnotherOrg
		}
		log.Debug().Err(err).Msg("failed to add org domain")
		return nil, nil, err
	}
	// Invalidate the consumed challenge. The memory-store interface has no
	// exact-key delete, so overwrite with an empty value + 1s TTL — precise
	// (unlike a prefix delete, which could clobber a sibling domain's challenge).
	_ = p.MemoryStoreProvider.SetCache(challengeKey(orgID, domain), "", 1)

	p.logOrgDomainVerified(meta, domain, "dns_txt")
	return row.AsAPIOrgDomain(), nil, nil
}

// AddVerifiedOrgDomain trusted-asserts a verified domain with NO proof. It is
// SUPER-ADMIN ONLY (the platform operator is already trusted); an org admin can
// never reach it. Subject to the same uniqueness + public-suffix guards.
func (p *provider) AddVerifiedOrgDomain(ctx context.Context, meta RequestMetadata, params *model.AddVerifiedOrgDomainRequest) (*model.OrgDomain, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "AddVerifiedOrgDomain").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}
	orgID := strings.TrimSpace(params.OrgID)
	if orgID == "" {
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainVerifyFailedEvent, orgID)
		return nil, nil, InvalidArgument("org_id is required")
	}
	domain, err := normalizeDomain(params.Domain)
	if err != nil {
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainVerifyFailedEvent, orgID)
		return nil, nil, err
	}
	if err := guardVerifiableDomain(domain); err != nil {
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainVerifyFailedEvent, orgID)
		return nil, nil, err
	}
	if _, err := p.StorageProvider.GetOrganizationByID(ctx, orgID); err != nil {
		log.Debug().Err(err).Msg("organization not found")
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainVerifyFailedEvent, orgID)
		return nil, nil, NotFound("organization not found")
	}

	row, err := p.StorageProvider.AddOrgDomain(ctx, &schemas.OrgDomain{
		ID:         domain,
		Domain:     domain,
		OrgID:      orgID,
		VerifiedAt: time.Now().Unix(),
	})
	if err != nil {
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainVerifyFailedEvent, orgID)
		if errors.Is(err, schemas.ErrOrgDomainConflict) {
			return nil, nil, errDomainOwnedByAnotherOrg
		}
		log.Debug().Err(err).Msg("failed to add org domain")
		return nil, nil, err
	}
	// If it was already verified by THIS org, AddOrgDomain returns the existing
	// row idempotently — no error, correct.
	p.logOrgDomainVerified(meta, domain, "trusted_assert")
	return row.AsAPIOrgDomain(), nil, nil
}

// OrgDomains lists an org's verified domains (never another org's). Gated on
// params.OrgID (super-admin or that org's org-admin).
func (p *provider) OrgDomains(ctx context.Context, meta RequestMetadata, params *model.ListOrgDomainsRequest) (*model.OrgDomains, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "OrgDomains").Logger()
	if err := p.requireOrgAdmin(ctx, meta, params.OrgID); err != nil {
		return nil, nil, err
	}
	orgID := strings.TrimSpace(params.OrgID)
	if orgID == "" {
		return nil, nil, InvalidArgument("org_id is required")
	}
	pagination := utils.GetPagination(params.Pagination)
	domains, pagination, err := p.StorageProvider.ListOrgDomainsByOrg(ctx, orgID, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListOrgDomainsByOrg")
		return nil, nil, err
	}
	res := make([]*model.OrgDomain, len(domains))
	for i, d := range domains {
		res[i] = d.AsAPIOrgDomain()
	}
	return &model.OrgDomains{
		Pagination: pagination,
		OrgDomains: res,
	}, nil, nil
}

// DeleteOrgDomain removes a verified domain, freeing it for re-verification. The
// request carries only the domain; the owning org is loaded FIRST and the
// org-admin check keys on THAT org (design H2 — never on a caller-supplied id).
func (p *provider) DeleteOrgDomain(ctx context.Context, meta RequestMetadata, params *model.DeleteOrgDomainRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "DeleteOrgDomain").Logger()
	domain, err := normalizeDomain(params.Domain)
	if err != nil {
		return nil, nil, err
	}
	row, err := p.StorageProvider.GetOrgDomainByDomain(ctx, domain)
	if err != nil || row == nil {
		log.Debug().Err(err).Msg("org domain not found")
		return nil, nil, p.maskNonSuperAdminError(ctx, meta, NotFound("org domain not found"))
	}
	if err := p.requireOrgAdmin(ctx, meta, row.OrgID); err != nil {
		return nil, nil, err
	}
	if err := p.StorageProvider.DeleteOrgDomain(ctx, domain); err != nil {
		log.Debug().Err(err).Msg("failed to delete org domain")
		p.logOrgDomainFailure(meta, constants.AuditOrgDomainDeleteFailedEvent, row.OrgID)
		return nil, nil, err
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditOrgDomainDeletedEvent,
		Protocol:     meta.Protocol,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgDomain,
		ResourceID:   domain,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return &model.Response{Message: "org domain deleted successfully"}, nil, nil
}

// logOrgDomainVerified records a successful verification, tagging the method
// (dns_txt / trusted_assert) in the audit metadata. The token is never logged.
func (p *provider) logOrgDomainVerified(meta RequestMetadata, domain, method string) {
	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditOrgDomainVerifiedEvent,
		Protocol:     meta.Protocol,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgDomain,
		ResourceID:   domain,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
		Metadata:     fmt.Sprintf(`{"method":%q}`, method),
	})
}

// logOrgDomainFailure records a failed domain admin operation.
func (p *provider) logOrgDomainFailure(meta RequestMetadata, action, orgID string) {
	p.AuditProvider.LogEvent(audit.Event{
		Action:       action,
		Protocol:     meta.Protocol,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeOrgDomain,
		ResourceID:   orgID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
}
