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

// clientSecretBytes is the entropy of a generated client secret.
// 32 bytes (256 bits) base64url-encoded — well above bcrypt's 72-byte input cap.
const clientSecretBytes = 32

// clientSecretCost is the bcrypt cost used for hashing service account
// client secrets. The schema doc comment on ClientSecret commits to cost 12 —
// this MUST stay 12, not bcrypt.DefaultCost (10).
const clientSecretCost = 12

// generateClientSecret returns a cryptographically random, URL-safe
// plaintext client secret. Mirrors the PKCE verifier generation in
// internal/utils/pkce.go.
func generateClientSecret() (string, error) {
	b := make([]byte, clientSecretBytes)
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

// CreateClient provisions a new machine/workload identity. It generates
// a random client secret, stores only its bcrypt hash (cost 12), and returns
// the plaintext exactly once. Requires super-admin auth.
func (p *provider) CreateClient(ctx context.Context, meta RequestMetadata, params *model.CreateClientRequest) (*model.CreateClientResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "CreateClient").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(params.Name) == "" {
		log.Debug().Msg("name is required")
		return nil, nil, InvalidArgument("name is required")
	}

	scopes := normalizeScopes(params.AllowedScopes)
	if scopes == "" {
		log.Debug().Msg("empty allowed_scopes rejected")
		return nil, nil, InvalidArgument("at least one allowed scope is required")
	}

	secret, err := generateClientSecret()
	if err != nil {
		log.Debug().Err(err).Msg("failed to generate client secret")
		return nil, nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), clientSecretCost)
	if err != nil {
		log.Debug().Err(err).Msg("failed to hash client secret")
		return nil, nil, err
	}

	sa, err := p.StorageProvider.AddClient(ctx, &schemas.Client{
		Name:        strings.TrimSpace(params.Name),
		Description: params.Description,
		// Kind is immutable and defaults to service_account in this rename step;
		// the interactive kind lands in a later Phase A step.
		Kind:          "service_account",
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
		Action:   constants.AuditClientCreatedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeServiceAccount,
		ResourceID:   sa.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.CreateClientResponse{
		Client:       sa.AsAPIClient(),
		ClientSecret: secret,
	}, nil, nil
}

// UpdateClient mutates only the fields present in params (load-then-
// mutate, so the storage Save does not blank untouched columns). It never
// touches the client secret. Requires super-admin auth.
func (p *provider) UpdateClient(ctx context.Context, meta RequestMetadata, params *model.UpdateClientRequest) (*model.Client, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "UpdateClient").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	sa, err := p.StorageProvider.GetClientByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetClientByID")
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
			return nil, nil, InvalidArgument("at least one allowed scope is required")
		}
		sa.AllowedScopes = scopes
	}
	if params.IsActive != nil {
		sa.IsActive = *params.IsActive
	}

	updated, err := p.StorageProvider.UpdateClient(ctx, sa)
	if err != nil {
		log.Debug().Err(err).Msg("failed UpdateClient")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditClientUpdatedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeServiceAccount,
		ResourceID:   updated.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return updated.AsAPIClient(), nil, nil
}

// DeleteClient removes a service account. The storage layer cascades to
// the account's TrustedIssuers. Requires super-admin auth.
func (p *provider) DeleteClient(ctx context.Context, meta RequestMetadata, params *model.ClientRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "DeleteClient").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	if params.ID == "" {
		log.Debug().Msg("service account ID required")
		return nil, nil, InvalidArgument("service account ID required")
	}

	sa, err := p.StorageProvider.GetClientByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetClientByID")
		return nil, nil, err
	}

	if err := p.StorageProvider.DeleteClient(ctx, sa); err != nil {
		log.Debug().Err(err).Msg("failed DeleteClient")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditClientDeletedEvent,
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

// RotateClientSecret generates a fresh client secret, replaces the
// stored bcrypt hash (cost 12), and returns the new plaintext exactly once.
// The old secret stops validating immediately. Requires super-admin auth.
func (p *provider) RotateClientSecret(ctx context.Context, meta RequestMetadata, params *model.ClientRequest) (*model.CreateClientResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "RotateClientSecret").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	if params.ID == "" {
		log.Debug().Msg("service account ID required")
		return nil, nil, InvalidArgument("service account ID required")
	}

	sa, err := p.StorageProvider.GetClientByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetClientByID")
		return nil, nil, err
	}

	secret, err := generateClientSecret()
	if err != nil {
		log.Debug().Err(err).Msg("failed to generate client secret")
		return nil, nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), clientSecretCost)
	if err != nil {
		log.Debug().Err(err).Msg("failed to hash client secret")
		return nil, nil, err
	}
	sa.ClientSecret = string(hash)

	updated, err := p.StorageProvider.UpdateClient(ctx, sa)
	if err != nil {
		log.Debug().Err(err).Msg("failed UpdateClient")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditClientSecretRotatedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeServiceAccount,
		ResourceID:   updated.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.CreateClientResponse{
		Client:       updated.AsAPIClient(),
		ClientSecret: secret,
	}, nil, nil
}

// Client returns a single service account by id. The client secret is
// never surfaced. Requires super-admin auth.
func (p *provider) Client(ctx context.Context, meta RequestMetadata, params *model.ClientRequest) (*model.Client, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Client").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	sa, err := p.StorageProvider.GetClientByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetClientByID")
		return nil, nil, err
	}
	return sa.AsAPIClient(), nil, nil
}

// Clients returns a paginated list of service accounts. Client secrets
// are never surfaced. Requires super-admin auth.
func (p *provider) Clients(ctx context.Context, meta RequestMetadata, params *model.ListClientsRequest) (*model.Clients, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Clients").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}

	var paginatedReq *model.PaginationRequest
	if params != nil {
		paginatedReq = params.Pagination
	}
	pagination := utils.GetPagination(paginatedReq)

	sas, pagination, err := p.StorageProvider.ListClients(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListClients")
		return nil, nil, err
	}
	res := make([]*model.Client, len(sas))
	for i, sa := range sas {
		res[i] = sa.AsAPIClient()
	}
	return &model.Clients{
		Pagination: pagination,
		Clients:    res,
	}, nil, nil
}
