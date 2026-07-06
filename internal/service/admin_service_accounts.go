package service

import (
	"context"
	"crypto/rand"
	b64 "encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// serviceAccountSecretBytes is the entropy of a generated client secret.
// 32 bytes (256 bits) base64url-encoded — well above bcrypt's 72-byte input cap.
const serviceAccountSecretBytes = 32

// serviceAccountSecretCost is the bcrypt cost used for hashing service account
// client secrets. The schema doc comment on ClientSecret commits to cost 12 —
// this MUST stay 12, not bcrypt.DefaultCost (10).
const serviceAccountSecretCost = 12

// generateServiceAccountSecret returns a cryptographically random, URL-safe
// plaintext client secret. Mirrors the PKCE verifier generation in
// internal/utils/pkce.go.
func generateServiceAccountSecret() (string, error) {
	b := make([]byte, serviceAccountSecretBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate client secret: %w", err)
	}
	return strings.Trim(b64.URLEncoding.EncodeToString(b), "="), nil
}

// normalizeScopes trims whitespace, drops empty segments, and dedupes the
// requested scopes, returning the canonical comma-separated form for storage.
// An all-empty/whitespace input yields "" — callers MUST reject that, since an
// empty AllowedScopes is DENY-ALL and must never be persisted (schema §
// AllowedScopes comment).
func normalizeScopes(scopes []string) string {
	seen := make(map[string]struct{}, len(scopes))
	out := make([]string, 0, len(scopes))
	for _, s := range scopes {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return strings.Join(out, ",")
}

// CreateServiceAccount provisions a new machine/workload identity. It generates
// a random client secret, stores only its bcrypt hash (cost 12), and returns
// the plaintext exactly once. Requires super-admin auth.
func (p *provider) CreateServiceAccount(ctx context.Context, meta RequestMetadata, params *model.CreateServiceAccountRequest) (*model.CreateServiceAccountResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "CreateServiceAccount").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(params.Name) == "" {
		log.Debug().Msg("name is required")
		return nil, nil, fmt.Errorf("name is required")
	}

	scopes := normalizeScopes(params.AllowedScopes)
	if scopes == "" {
		log.Debug().Msg("empty allowed_scopes rejected")
		return nil, nil, fmt.Errorf("at least one allowed scope is required")
	}

	secret, err := generateServiceAccountSecret()
	if err != nil {
		log.Debug().Err(err).Msg("failed to generate client secret")
		return nil, nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), serviceAccountSecretCost)
	if err != nil {
		log.Debug().Err(err).Msg("failed to hash client secret")
		return nil, nil, err
	}

	sa, err := p.StorageProvider.AddServiceAccount(ctx, &schemas.ServiceAccount{
		Name:          strings.TrimSpace(params.Name),
		Description:   params.Description,
		ClientSecret:  string(hash),
		AllowedScopes: scopes,
		// Set IsActive explicitly — never rely on the GORM `default:true` column
		// default (a future pre-disabled create path would silently come back
		// active). There is no create-as-disabled feature in Phase 1.
		IsActive: true,
	})
	if err != nil {
		log.Debug().Err(err).Msg("failed to add service account")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditServiceAccountCreatedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeServiceAccount,
		ResourceID:   sa.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.CreateServiceAccountResponse{
		ServiceAccount: sa.AsAPIServiceAccount(),
		ClientSecret:   secret,
	}, nil, nil
}

// UpdateServiceAccount mutates only the fields present in params (load-then-
// mutate, so the storage Save does not blank untouched columns). It never
// touches the client secret. Requires super-admin auth.
func (p *provider) UpdateServiceAccount(ctx context.Context, meta RequestMetadata, params *model.UpdateServiceAccountRequest) (*model.ServiceAccount, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "UpdateServiceAccount").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	sa, err := p.StorageProvider.GetServiceAccountByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetServiceAccountByID")
		return nil, nil, err
	}

	if params.Name != nil {
		sa.Name = strings.TrimSpace(*params.Name)
	}
	if params.Description != nil {
		sa.Description = params.Description
	}
	// A non-nil slice means the client sent allowed_scopes; normalize and reject
	// if it collapses to empty (defense-in-depth against a zero-scope account).
	if params.AllowedScopes != nil {
		scopes := normalizeScopes(params.AllowedScopes)
		if scopes == "" {
			log.Debug().Msg("empty allowed_scopes rejected")
			return nil, nil, fmt.Errorf("at least one allowed scope is required")
		}
		sa.AllowedScopes = scopes
	}
	if params.IsActive != nil {
		sa.IsActive = *params.IsActive
	}

	updated, err := p.StorageProvider.UpdateServiceAccount(ctx, sa)
	if err != nil {
		log.Debug().Err(err).Msg("failed UpdateServiceAccount")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditServiceAccountUpdatedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeServiceAccount,
		ResourceID:   updated.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return updated.AsAPIServiceAccount(), nil, nil
}

// DeleteServiceAccount removes a service account. The storage layer cascades to
// the account's TrustedIssuers. Requires super-admin auth.
func (p *provider) DeleteServiceAccount(ctx context.Context, meta RequestMetadata, params *model.ServiceAccountRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "DeleteServiceAccount").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	if params.ID == "" {
		log.Debug().Msg("service account ID required")
		return nil, nil, fmt.Errorf("service account ID required")
	}

	sa, err := p.StorageProvider.GetServiceAccountByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetServiceAccountByID")
		return nil, nil, err
	}

	if err := p.StorageProvider.DeleteServiceAccount(ctx, sa); err != nil {
		log.Debug().Err(err).Msg("failed DeleteServiceAccount")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditServiceAccountDeletedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeServiceAccount,
		ResourceID:   params.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{
		Message: "Service account deleted successfully",
	}, nil, nil
}

// RotateServiceAccountSecret generates a fresh client secret, replaces the
// stored bcrypt hash (cost 12), and returns the new plaintext exactly once.
// The old secret stops validating immediately. Requires super-admin auth.
func (p *provider) RotateServiceAccountSecret(ctx context.Context, meta RequestMetadata, params *model.ServiceAccountRequest) (*model.CreateServiceAccountResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "RotateServiceAccountSecret").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	if params.ID == "" {
		log.Debug().Msg("service account ID required")
		return nil, nil, fmt.Errorf("service account ID required")
	}

	sa, err := p.StorageProvider.GetServiceAccountByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetServiceAccountByID")
		return nil, nil, err
	}

	secret, err := generateServiceAccountSecret()
	if err != nil {
		log.Debug().Err(err).Msg("failed to generate client secret")
		return nil, nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), serviceAccountSecretCost)
	if err != nil {
		log.Debug().Err(err).Msg("failed to hash client secret")
		return nil, nil, err
	}
	sa.ClientSecret = string(hash)

	updated, err := p.StorageProvider.UpdateServiceAccount(ctx, sa)
	if err != nil {
		log.Debug().Err(err).Msg("failed UpdateServiceAccount")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditServiceAccountSecretRotatedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeServiceAccount,
		ResourceID:   updated.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.CreateServiceAccountResponse{
		ServiceAccount: updated.AsAPIServiceAccount(),
		ClientSecret:   secret,
	}, nil, nil
}

// ServiceAccount returns a single service account by id. The client secret is
// never surfaced. Requires super-admin auth.
func (p *provider) ServiceAccount(ctx context.Context, meta RequestMetadata, params *model.ServiceAccountRequest) (*model.ServiceAccount, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "ServiceAccount").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	sa, err := p.StorageProvider.GetServiceAccountByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetServiceAccountByID")
		return nil, nil, err
	}
	return sa.AsAPIServiceAccount(), nil, nil
}

// ServiceAccounts returns a paginated list of service accounts. Client secrets
// are never surfaced. Requires super-admin auth.
func (p *provider) ServiceAccounts(ctx context.Context, meta RequestMetadata, params *model.ListServiceAccountsRequest) (*model.ServiceAccounts, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "ServiceAccounts").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	var paginatedReq *model.PaginatedRequest
	if params != nil {
		paginatedReq = params.Pagination
	}
	pagination := utils.GetPagination(paginatedReq)

	sas, pagination, err := p.StorageProvider.ListServiceAccounts(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListServiceAccounts")
		return nil, nil, err
	}
	res := make([]*model.ServiceAccount, len(sas))
	for i, sa := range sas {
		res[i] = sa.AsAPIServiceAccount()
	}
	return &model.ServiceAccounts{
		Pagination:      pagination,
		ServiceAccounts: res,
	}, nil, nil
}
