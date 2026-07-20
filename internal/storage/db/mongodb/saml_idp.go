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

// --- SAMLServiceProvider (registered downstream SPs; Authorizer as IdP) ---

// AddSAMLServiceProvider registers a new downstream SP.
func (p *provider) AddSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) (*schemas.SAMLServiceProvider, error) {
	if sp.ID == "" {
		sp.ID = uuid.New().String()
	}
	sp.Key = sp.ID
	now := time.Now().Unix()
	sp.CreatedAt = now
	sp.UpdatedAt = now
	spCollection := p.db.Collection(schemas.Collections.SAMLServiceProvider, options.Collection())
	_, err := spCollection.InsertOne(ctx, sp)
	if err != nil {
		return nil, err
	}
	return sp, nil
}

// UpdateSAMLServiceProvider writes back a fully-loaded record. The $set write
// replaces every column, so callers MUST load the record and mutate it before
// calling — a partial struct will blank zero-value fields.
func (p *provider) UpdateSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) (*schemas.SAMLServiceProvider, error) {
	if sp.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateSAMLServiceProvider: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	sp.UpdatedAt = time.Now().Unix()
	spCollection := p.db.Collection(schemas.Collections.SAMLServiceProvider, options.Collection())
	_, err := spCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": sp.ID}}, bson.M{"$set": sp}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return sp, nil
}

// DeleteSAMLServiceProvider removes a registered SP.
func (p *provider) DeleteSAMLServiceProvider(ctx context.Context, sp *schemas.SAMLServiceProvider) error {
	spCollection := p.db.Collection(schemas.Collections.SAMLServiceProvider, options.Collection())
	_, err := spCollection.DeleteOne(ctx, bson.M{"_id": sp.ID}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetSAMLServiceProviderByID fetches a registered SP by primary key.
func (p *provider) GetSAMLServiceProviderByID(ctx context.Context, id string) (*schemas.SAMLServiceProvider, error) {
	var sp *schemas.SAMLServiceProvider
	spCollection := p.db.Collection(schemas.Collections.SAMLServiceProvider, options.Collection())
	err := spCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&sp)
	if err != nil {
		return nil, err
	}
	return sp, nil
}

// GetSAMLServiceProviderByOrgAndEntityID resolves the single registered SP for an
// (orgID, entityID) pair — the AuthnRequest-Issuer → trusted-ACS binding.
func (p *provider) GetSAMLServiceProviderByOrgAndEntityID(ctx context.Context, orgID, entityID string) (*schemas.SAMLServiceProvider, error) {
	var sp *schemas.SAMLServiceProvider
	spCollection := p.db.Collection(schemas.Collections.SAMLServiceProvider, options.Collection())
	err := spCollection.FindOne(ctx, bson.M{"org_id": orgID, "entity_id": entityID}).Decode(&sp)
	if err != nil {
		return nil, err
	}
	return sp, nil
}

// ListSAMLServiceProviders returns the registered SPs for an org (paginated).
func (p *provider) ListSAMLServiceProviders(ctx context.Context, orgID string, pagination *model.Pagination) ([]*schemas.SAMLServiceProvider, *model.Pagination, error) {
	sps := []*schemas.SAMLServiceProvider{}
	filter := bson.M{"org_id": orgID}
	opts := options.Find()
	opts.SetLimit(pagination.Limit)
	opts.SetSkip(pagination.Offset)
	opts.SetSort(bson.M{"created_at": -1})
	paginationClone := pagination
	spCollection := p.db.Collection(schemas.Collections.SAMLServiceProvider, options.Collection())
	count, err := spCollection.CountDocuments(ctx, filter, options.Count())
	if err != nil {
		return nil, nil, err
	}
	paginationClone.Total = count
	cursor, err := spCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	for cursor.Next(ctx) {
		var sp *schemas.SAMLServiceProvider
		err := cursor.Decode(&sp)
		if err != nil {
			return nil, nil, err
		}
		sps = append(sps, sp)
	}
	return sps, paginationClone, nil
}

// --- SAMLIDPKey (per-org signing keypairs with rotation) ---

// AddSAMLIDPKey persists a newly-generated signing keypair.
func (p *provider) AddSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) (*schemas.SAMLIDPKey, error) {
	if key.ID == "" {
		key.ID = uuid.New().String()
	}
	key.Key = key.ID
	now := time.Now().Unix()
	key.CreatedAt = now
	key.UpdatedAt = now
	keyCollection := p.db.Collection(schemas.Collections.SAMLIDPKey, options.Collection())
	_, err := keyCollection.InsertOne(ctx, key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// UpdateSAMLIDPKey writes back a fully-loaded record (used to flip status). The
// $set write replaces every column, so callers MUST load the record and mutate
// it before calling — a partial struct will blank zero-value fields.
func (p *provider) UpdateSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) (*schemas.SAMLIDPKey, error) {
	if key.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateSAMLIDPKey: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	key.UpdatedAt = time.Now().Unix()
	keyCollection := p.db.Collection(schemas.Collections.SAMLIDPKey, options.Collection())
	_, err := keyCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": key.ID}}, bson.M{"$set": key}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return key, nil
}

// DeleteSAMLIDPKey removes a signing key.
func (p *provider) DeleteSAMLIDPKey(ctx context.Context, key *schemas.SAMLIDPKey) error {
	keyCollection := p.db.Collection(schemas.Collections.SAMLIDPKey, options.Collection())
	_, err := keyCollection.DeleteOne(ctx, bson.M{"_id": key.ID}, options.Delete())
	if err != nil {
		return err
	}
	return nil
}

// GetSAMLIDPKeyByID fetches a signing key by primary key.
func (p *provider) GetSAMLIDPKeyByID(ctx context.Context, id string) (*schemas.SAMLIDPKey, error) {
	var key *schemas.SAMLIDPKey
	keyCollection := p.db.Collection(schemas.Collections.SAMLIDPKey, options.Collection())
	err := keyCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// ListSAMLIDPKeys returns every signing key for an org (newest first).
func (p *provider) ListSAMLIDPKeys(ctx context.Context, orgID string) ([]*schemas.SAMLIDPKey, error) {
	keys := []*schemas.SAMLIDPKey{}
	keyCollection := p.db.Collection(schemas.Collections.SAMLIDPKey, options.Collection())
	opts := options.Find()
	opts.SetSort(bson.M{"created_at": -1})
	cursor, err := keyCollection.Find(ctx, bson.M{"org_id": orgID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	for cursor.Next(ctx) {
		var key *schemas.SAMLIDPKey
		err := cursor.Decode(&key)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}
