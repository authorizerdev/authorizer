package sql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddTrustedIssuer creates a new trusted issuer record.
func (p *provider) AddTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	if issuer.ID == "" {
		issuer.ID = uuid.New().String()
	}
	issuer.Key = issuer.ID
	now := time.Now().Unix()
	issuer.CreatedAt = now
	issuer.UpdatedAt = now
	res := p.db.Create(issuer)
	if res.Error != nil {
		return nil, res.Error
	}
	return issuer, nil
}

// UpdateTrustedIssuer updates a trusted issuer record.
// Callers MUST load the existing record and mutate it before calling this
// method — Save writes every column and will blank zero-value fields on a
// partial struct (e.g. IssuerURL, ServiceAccountID, KeySourceType).
func (p *provider) UpdateTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	if issuer.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateTrustedIssuer: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	issuer.UpdatedAt = time.Now().Unix()
	res := p.db.Save(issuer)
	if res.Error != nil {
		return nil, res.Error
	}
	return issuer, nil
}

// DeleteTrustedIssuer removes a trusted issuer record.
func (p *provider) DeleteTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) error {
	return p.db.Delete(issuer).Error
}

// GetTrustedIssuerByID fetches a trusted issuer by primary key.
func (p *provider) GetTrustedIssuerByID(ctx context.Context, id string) (*schemas.TrustedIssuer, error) {
	var issuer schemas.TrustedIssuer
	res := p.db.Where("id = ?", id).First(&issuer)
	if res.Error != nil {
		return nil, res.Error
	}
	return &issuer, nil
}

// GetTrustedIssuerByIssuerURL fetches a trusted issuer by its unique issuer URL.
// Called on every client_assertion validation — kept as a single indexed lookup.
func (p *provider) GetTrustedIssuerByIssuerURL(ctx context.Context, issuerURL string) (*schemas.TrustedIssuer, error) {
	var issuer schemas.TrustedIssuer
	res := p.db.Where("issuer_url = ?", issuerURL).First(&issuer)
	if res.Error != nil {
		return nil, res.Error
	}
	return &issuer, nil
}

// GetTrustedIssuerByOrgIDAndKind fetches a trusted issuer by (orgID, kind).
func (p *provider) GetTrustedIssuerByOrgIDAndKind(ctx context.Context, orgID, kind string) (*schemas.TrustedIssuer, error) {
	var issuer schemas.TrustedIssuer
	res := p.db.Where("org_id = ? AND kind = ?", orgID, kind).First(&issuer)
	if res.Error != nil {
		return nil, res.Error
	}
	return &issuer, nil
}

// ListTrustedIssuers returns paginated trusted issuers, optionally filtered by serviceAccountID.
func (p *provider) ListTrustedIssuers(ctx context.Context, serviceAccountID string, pagination *model.Pagination) ([]*schemas.TrustedIssuer, *model.Pagination, error) {
	var issuers []*schemas.TrustedIssuer
	q := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC")
	if serviceAccountID != "" {
		q = q.Where("client_id = ?", serviceAccountID)
	}
	res := q.Find(&issuers)
	if res.Error != nil {
		return nil, nil, res.Error
	}
	var total int64
	countQ := p.db.Model(&schemas.TrustedIssuer{})
	if serviceAccountID != "" {
		countQ = countQ.Where("client_id = ?", serviceAccountID)
	}
	countRes := countQ.Count(&total)
	if countRes.Error != nil {
		return nil, nil, countRes.Error
	}
	return issuers, &model.Pagination{
		Limit:  pagination.Limit,
		Page:   pagination.Page,
		Offset: pagination.Offset,
		Total:  total,
	}, nil
}
