package arangodb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddFederatedIdentity records a JIT-provisioned federated identity. The unique
// hash index on (org_id, issuer, subject) rejects duplicates at the database
// layer.
func (p *provider) AddFederatedIdentity(ctx context.Context, identity *schemas.FederatedIdentity) (*schemas.FederatedIdentity, error) {
	if identity.ID == "" {
		identity.ID = uuid.New().String()
	}
	identity.Key = identity.ID
	now := time.Now().Unix()
	identity.CreatedAt = now
	identity.UpdatedAt = now
	identityCollection, _ := p.db.Collection(ctx, schemas.Collections.FederatedIdentity)
	doc, err := structToDocument(identity)
	if err != nil {
		return nil, err
	}
	meta, err := identityCollection.CreateDocument(ctx, doc)
	if err != nil {
		return nil, err
	}
	identity.Key = meta.Key
	identity.ID = meta.ID.String()
	return identity, nil
}

// GetFederatedIdentity fetches the federated identity for a (orgID, issuer, subject) triple.
func (p *provider) GetFederatedIdentity(ctx context.Context, orgID, issuer, subject string) (*schemas.FederatedIdentity, error) {
	var identity *schemas.FederatedIdentity
	query := fmt.Sprintf("FOR d in %s FILTER d.org_id == @org_id AND d.issuer == @issuer AND d.subject == @subject LIMIT 1 RETURN d", schemas.Collections.FederatedIdentity)
	bindVars := map[string]interface{}{
		"org_id":  orgID,
		"issuer":  issuer,
		"subject": subject,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if identity == nil {
				return nil, fmt.Errorf("federated identity not found")
			}
			break
		}
		fi := &schemas.FederatedIdentity{}
		if _, err := readDocument(ctx, cursor, fi); err != nil {
			return nil, err
		}
		identity = fi
	}
	return identity, nil
}
