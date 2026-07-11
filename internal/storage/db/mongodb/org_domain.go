package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOrgDomain atomically inserts a verified domain row. _id is the normalized
// domain, so a duplicate insert is rejected by MongoDB (E11000) — first-writer
// wins with no check-then-insert race. On conflict we classify by owning org.
func (p *provider) AddOrgDomain(ctx context.Context, domain *schemas.OrgDomain) (*schemas.OrgDomain, error) {
	domain.Key = domain.ID
	now := time.Now().Unix()
	domain.CreatedAt = now
	domain.UpdatedAt = now
	if domain.VerifiedAt == 0 {
		domain.VerifiedAt = now
	}
	orgDomainCollection := p.db.Collection(schemas.Collections.OrgDomain, options.Collection())
	_, err := orgDomainCollection.InsertOne(ctx, domain)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			existing, getErr := p.GetOrgDomainByDomain(ctx, domain.ID)
			if getErr == nil && existing != nil {
				if existing.OrgID == domain.OrgID {
					return existing, nil
				}
				return nil, schemas.ErrOrgDomainConflict
			}
		}
		return nil, err
	}
	return domain, nil
}

// GetOrgDomainByDomain fetches a verified domain by its normalized value (_id).
func (p *provider) GetOrgDomainByDomain(ctx context.Context, domain string) (*schemas.OrgDomain, error) {
	var d *schemas.OrgDomain
	orgDomainCollection := p.db.Collection(schemas.Collections.OrgDomain, options.Collection())
	err := orgDomainCollection.FindOne(ctx, bson.M{"_id": domain}).Decode(&d)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// ListOrgDomainsByOrg returns an org's verified domains, paginated.
func (p *provider) ListOrgDomainsByOrg(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.OrgDomain, *model.Pagination, error) {
	orgDomainCollection := p.db.Collection(schemas.Collections.OrgDomain, options.Collection())
	total, err := orgDomainCollection.CountDocuments(ctx, bson.M{"org_id": orgID})
	if err != nil {
		return nil, nil, err
	}
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})
	cursor, err := orgDomainCollection.Find(ctx, bson.M{"org_id": orgID}, opts)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	domains := []*schemas.OrgDomain{}
	for cursor.Next(ctx) {
		var d schemas.OrgDomain
		if err := cursor.Decode(&d); err != nil {
			return nil, nil, err
		}
		domains = append(domains, &d)
	}
	return domains, &model.Pagination{
		Limit:  pagination.Limit,
		Page:   pagination.Page,
		Offset: pagination.Offset,
		Total:  total,
	}, nil
}

// DeleteOrgDomain removes a verified domain mapping by normalized domain.
func (p *provider) DeleteOrgDomain(ctx context.Context, domain string) error {
	orgDomainCollection := p.db.Collection(schemas.Collections.OrgDomain, options.Collection())
	_, err := orgDomainCollection.DeleteOne(ctx, bson.M{"_id": domain}, options.Delete())
	return err
}

// DeleteOrgDomainsByOrg removes all of an org's verified domains (cascade).
func (p *provider) DeleteOrgDomainsByOrg(ctx context.Context, orgID string) error {
	orgDomainCollection := p.db.Collection(schemas.Collections.OrgDomain, options.Collection())
	_, err := orgDomainCollection.DeleteMany(ctx, bson.M{"org_id": orgID}, options.Delete())
	return err
}
