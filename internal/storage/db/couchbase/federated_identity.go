package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const federatedIdentityColumns = "_id, org_id, issuer, subject, user_id, created_at, updated_at"

// AddFederatedIdentity records a JIT-provisioned federated identity.
// (org_id, issuer, subject) is unique; Couchbase has no compound unique
// constraint, so guard with a check-then-insert (closes the sequential case).
func (p *provider) AddFederatedIdentity(ctx context.Context, identity *schemas.FederatedIdentity) (*schemas.FederatedIdentity, error) {
	if identity.ID == "" {
		identity.ID = uuid.New().String()
	}
	identity.Key = identity.ID
	now := time.Now().Unix()
	identity.CreatedAt = now
	identity.UpdatedAt = now
	if existing, _ := p.GetFederatedIdentity(ctx, identity.OrgID, identity.Issuer, identity.Subject); existing != nil {
		return nil, fmt.Errorf("federated identity for org_id %s issuer %s subject %s already exists", identity.OrgID, identity.Issuer, identity.Subject)
	}
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	doc, err := structToDocument(identity)
	if err != nil {
		return nil, err
	}
	_, err = p.db.Collection(schemas.Collections.FederatedIdentity).Insert(identity.ID, doc, &insertOpt)
	if err != nil {
		return nil, err
	}
	return identity, nil
}

// GetFederatedIdentity fetches the federated identity for an (orgID, issuer, subject) triple.
func (p *provider) GetFederatedIdentity(ctx context.Context, orgID, issuer, subject string) (*schemas.FederatedIdentity, error) {
	params := make(map[string]interface{}, 3)
	params["org_id"] = orgID
	params["issuer"] = issuer
	params["subject"] = subject
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE org_id=$org_id AND issuer=$issuer AND subject=$subject LIMIT 1`, federatedIdentityColumns, p.scopeName, schemas.Collections.FederatedIdentity)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := q.One(&raw); err != nil {
		return nil, err
	}
	identity := &schemas.FederatedIdentity{}
	if err := decodeDocument(raw, identity); err != nil {
		return nil, err
	}
	return identity, nil
}
