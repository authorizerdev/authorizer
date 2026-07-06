package arangodb

import (
	"context"
	"fmt"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
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
	issuerCollection, _ := p.db.Collection(ctx, schemas.Collections.TrustedIssuer)
	meta, err := issuerCollection.CreateDocument(ctx, issuer)
	if err != nil {
		return nil, err
	}
	issuer.Key = meta.Key
	issuer.ID = meta.ID.String()
	return issuer, nil
}

// UpdateTrustedIssuer updates a trusted issuer record.
// Callers MUST load the existing record and mutate it before calling this
// method — the document replace writes every field.
func (p *provider) UpdateTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	if issuer.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateTrustedIssuer: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	issuer.UpdatedAt = time.Now().Unix()
	issuerCollection, _ := p.db.Collection(ctx, schemas.Collections.TrustedIssuer)
	meta, err := issuerCollection.UpdateDocument(ctx, issuer.Key, issuer)
	if err != nil {
		return nil, err
	}
	issuer.Key = meta.Key
	issuer.ID = meta.ID.String()
	return issuer, nil
}

// DeleteTrustedIssuer removes a trusted issuer record.
func (p *provider) DeleteTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) error {
	issuerCollection, _ := p.db.Collection(ctx, schemas.Collections.TrustedIssuer)
	_, err := issuerCollection.RemoveDocument(ctx, issuer.Key)
	if err != nil {
		return err
	}
	return nil
}

// GetTrustedIssuerByID fetches a trusted issuer by primary key.
func (p *provider) GetTrustedIssuerByID(ctx context.Context, id string) (*schemas.TrustedIssuer, error) {
	var issuer *schemas.TrustedIssuer
	query := fmt.Sprintf("FOR d in %s FILTER d._id == @id LIMIT 1 RETURN d", schemas.Collections.TrustedIssuer)
	bindVars := map[string]interface{}{
		"id": id,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if issuer == nil {
				return nil, fmt.Errorf("trusted issuer not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &issuer)
		if err != nil {
			return nil, err
		}
	}
	return issuer, nil
}

// GetTrustedIssuerByIssuerURL fetches a trusted issuer by its unique issuer URL.
// Called on every client_assertion validation — kept as a single indexed lookup.
func (p *provider) GetTrustedIssuerByIssuerURL(ctx context.Context, issuerURL string) (*schemas.TrustedIssuer, error) {
	var issuer *schemas.TrustedIssuer
	query := fmt.Sprintf("FOR d in %s FILTER d.issuer_url == @issuer_url LIMIT 1 RETURN d", schemas.Collections.TrustedIssuer)
	bindVars := map[string]interface{}{
		"issuer_url": issuerURL,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if issuer == nil {
				return nil, fmt.Errorf("trusted issuer not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &issuer)
		if err != nil {
			return nil, err
		}
	}
	return issuer, nil
}

// ListTrustedIssuers returns paginated trusted issuers, optionally filtered by serviceAccountID.
func (p *provider) ListTrustedIssuers(ctx context.Context, serviceAccountID string, pagination *model.Pagination) ([]*schemas.TrustedIssuer, *model.Pagination, error) {
	issuers := []*schemas.TrustedIssuer{}
	filter := ""
	bindVars := map[string]interface{}{}
	if serviceAccountID != "" {
		filter = "FILTER d.service_account_id == @service_account_id "
		bindVars["service_account_id"] = serviceAccountID
	}
	query := fmt.Sprintf("FOR d in %s %sSORT d.created_at DESC LIMIT %d, %d RETURN d", schemas.Collections.TrustedIssuer, filter, pagination.Offset, pagination.Limit)
	sctx := arangoDriver.WithQueryFullCount(ctx)
	cursor, err := p.db.Query(sctx, query, bindVars)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close() }()
	paginationClone := pagination
	paginationClone.Total = cursor.Statistics().FullCount()
	for {
		var issuer *schemas.TrustedIssuer
		meta, err := cursor.ReadDocument(ctx, &issuer)
		if arangoDriver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		if meta.Key != "" {
			issuers = append(issuers, issuer)
		}
	}
	return issuers, paginationClone, nil
}
