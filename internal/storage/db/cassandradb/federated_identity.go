package cassandradb

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const federatedIdentityColumns = "id, org_id, issuer, subject, user_id, created_at, updated_at"

// scanFederatedIdentity maps the federatedIdentityColumns projection onto a struct.
func scanFederatedIdentity(scan func(...interface{}) error, identity *schemas.FederatedIdentity) error {
	return scan(&identity.ID, &identity.OrgID, &identity.Issuer, &identity.Subject, &identity.UserID, &identity.CreatedAt, &identity.UpdatedAt)
}

// AddFederatedIdentity creates a new federated identity. (org_id, issuer, subject) is unique.
// Cassandra has no cross-attribute unique constraint, so guard with a
// check-then-insert mirroring AddOrgMembership's pre-check.
// ponytail: inherent TOCTOU race — closes the sequential case only.
func (p *provider) AddFederatedIdentity(ctx context.Context, identity *schemas.FederatedIdentity) (*schemas.FederatedIdentity, error) {
	if identity.ID == "" {
		identity.ID = uuid.New().String()
	}
	identity.Key = identity.ID
	now := time.Now().Unix()
	identity.CreatedAt = now
	identity.UpdatedAt = now
	if existing, _ := p.GetFederatedIdentity(ctx, identity.OrgID, identity.Issuer, identity.Subject); existing != nil {
		return nil, fmt.Errorf("federated identity for org_id %s, issuer %s and subject %s already exists", identity.OrgID, identity.Issuer, identity.Subject)
	}
	insertQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?, ?)", KeySpace+"."+schemas.Collections.FederatedIdentity, federatedIdentityColumns)
	err := p.db.Query(insertQuery, identity.ID, identity.OrgID, identity.Issuer, identity.Subject, identity.UserID, identity.CreatedAt, identity.UpdatedAt).Exec()
	if err != nil {
		return nil, err
	}
	return identity, nil
}

// GetFederatedIdentity fetches the federated identity for a (orgID, issuer, subject) triple.
func (p *provider) GetFederatedIdentity(ctx context.Context, orgID, issuer, subject string) (*schemas.FederatedIdentity, error) {
	var identity schemas.FederatedIdentity
	query := fmt.Sprintf("SELECT %s FROM %s WHERE org_id = ? AND issuer = ? AND subject = ? LIMIT 1 ALLOW FILTERING", federatedIdentityColumns, KeySpace+"."+schemas.Collections.FederatedIdentity)
	if err := scanFederatedIdentity(p.db.Query(query, orgID, issuer, subject).Consistency(gocql.One).Scan, &identity); err != nil {
		return nil, err
	}
	return &identity, nil
}
