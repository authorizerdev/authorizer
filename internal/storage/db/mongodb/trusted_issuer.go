package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

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
	issuerCollection := p.db.Collection(schemas.Collections.TrustedIssuer, options.Collection())
	_, err := issuerCollection.InsertOne(ctx, issuer)
	if err != nil {
		return nil, err
	}
	return issuer, nil
}

// UpdateTrustedIssuer updates a trusted issuer record.
// Callers MUST load the existing record and mutate it before calling this
// method — the $set write replaces every column and will blank zero-value
// fields on a partial struct (e.g. IssuerURL, ServiceAccountID, KeySourceType).
func (p *provider) UpdateTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) (*schemas.TrustedIssuer, error) {
	if issuer.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateTrustedIssuer: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	issuer.UpdatedAt = time.Now().Unix()
	issuerCollection := p.db.Collection(schemas.Collections.TrustedIssuer, options.Collection())
	_, err := issuerCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": issuer.ID}}, bson.M{"$set": issuer}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return issuer, nil
}

// DeleteTrustedIssuer removes a trusted issuer record.
func (p *provider) DeleteTrustedIssuer(ctx context.Context, issuer *schemas.TrustedIssuer) error {
	issuerCollection := p.db.Collection(schemas.Collections.TrustedIssuer, options.Collection())
	_, err := issuerCollection.DeleteOne(ctx, bson.M{"_id": issuer.ID}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetTrustedIssuerByID fetches a trusted issuer by primary key.
func (p *provider) GetTrustedIssuerByID(ctx context.Context, id string) (*schemas.TrustedIssuer, error) {
	var issuer *schemas.TrustedIssuer
	issuerCollection := p.db.Collection(schemas.Collections.TrustedIssuer, options.Collection())
	err := issuerCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&issuer)
	if err != nil {
		return nil, err
	}
	return issuer, nil
}

// GetTrustedIssuerByIssuerURL fetches a trusted issuer by its unique issuer URL.
// Called on every client_assertion validation — kept as a single indexed lookup.
func (p *provider) GetTrustedIssuerByIssuerURL(ctx context.Context, issuerURL string) (*schemas.TrustedIssuer, error) {
	var issuer *schemas.TrustedIssuer
	issuerCollection := p.db.Collection(schemas.Collections.TrustedIssuer, options.Collection())
	err := issuerCollection.FindOne(ctx, bson.M{"issuer_url": issuerURL}).Decode(&issuer)
	if err != nil {
		return nil, err
	}
	return issuer, nil
}

// GetTrustedIssuerByOrgIDAndKind fetches a trusted issuer by its (org_id, kind) pair.
func (p *provider) GetTrustedIssuerByOrgIDAndKind(ctx context.Context, orgID, kind string) (*schemas.TrustedIssuer, error) {
	var issuer *schemas.TrustedIssuer
	issuerCollection := p.db.Collection(schemas.Collections.TrustedIssuer, options.Collection())
	err := issuerCollection.FindOne(ctx, bson.M{"org_id": orgID, "kind": kind}).Decode(&issuer)
	if err != nil {
		return nil, err
	}
	return issuer, nil
}

// ListTrustedIssuers returns paginated trusted issuers, optionally filtered by serviceAccountID.
func (p *provider) ListTrustedIssuers(ctx context.Context, serviceAccountID string, pagination *model.Pagination) ([]*schemas.TrustedIssuer, *model.Pagination, error) {
	issuers := []*schemas.TrustedIssuer{}
	filter := bson.M{}
	if serviceAccountID != "" {
		filter["client_id"] = serviceAccountID
	}
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})
	paginationClone := pagination
	issuerCollection := p.db.Collection(schemas.Collections.TrustedIssuer, options.Collection())
	count, err := issuerCollection.CountDocuments(ctx, filter, options.Count())
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = count
	cursor, err := issuerCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	for cursor.Next(ctx) {
		var issuer *schemas.TrustedIssuer
		err := cursor.Decode(&issuer)
		if err != nil {
			return nil, nil, err
		}
		issuers = append(issuers, issuer)
	}
	return issuers, paginationClone, nil
}
