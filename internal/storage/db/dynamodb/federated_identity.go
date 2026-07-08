package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddFederatedIdentity creates a new federated identity. (org_id, issuer,
// subject) is unique; DynamoDB has no compound unique constraint, so guard with
// a check-then-insert (closes the sequential case).
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
	if err := p.putItem(ctx, schemas.Collections.FederatedIdentity, identity); err != nil {
		return nil, err
	}
	return identity, nil
}

// GetFederatedIdentity fetches the federated identity for a (orgID, issuer,
// subject) triple via the org_id GSI with an (issuer, subject) filter.
func (p *provider) GetFederatedIdentity(ctx context.Context, orgID, issuer, subject string) (*schemas.FederatedIdentity, error) {
	// Must NOT pass a Limit alongside a FilterExpression: DynamoDB applies Limit
	// BEFORE the filter, so a Limit read returns arbitrary rows from the org_id
	// partition and can miss the (issuer, subject) match even when it exists.
	// queryEq paginates and filters server-side across the whole partition; the
	// unique (org_id, issuer, subject) invariant means at most one match.
	f := expression.Name("issuer").Equal(expression.Value(issuer)).
		And(expression.Name("subject").Equal(expression.Value(subject)))
	items, err := p.queryEq(ctx, schemas.Collections.FederatedIdentity, "org_id", "org_id", orgID, &f)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no document found")
	}
	var identity schemas.FederatedIdentity
	if err := unmarshalItem(items[0], &identity); err != nil {
		return nil, err
	}
	return &identity, nil
}
