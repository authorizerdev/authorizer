package arangodb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddScimEndpoint creates a new SCIM endpoint record.
func (p *provider) AddScimEndpoint(ctx context.Context, scimEndpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	if scimEndpoint.ID == "" {
		scimEndpoint.ID = uuid.New().String()
	}
	scimEndpoint.Key = scimEndpoint.ID
	now := time.Now().Unix()
	scimEndpoint.CreatedAt = now
	scimEndpoint.UpdatedAt = now
	scimEndpointCollection, _ := p.db.Collection(ctx, schemas.Collections.ScimEndpoint)
	doc, err := structToDocument(scimEndpoint)
	if err != nil {
		return nil, err
	}
	meta, err := scimEndpointCollection.CreateDocument(ctx, doc)
	if err != nil {
		return nil, err
	}
	scimEndpoint.Key = meta.Key
	scimEndpoint.ID = meta.ID.String()
	return scimEndpoint, nil
}

// UpdateScimEndpoint updates a SCIM endpoint record.
// Callers MUST load the existing record and mutate it before calling this
// method — this is a partial update via UpdateDocument (ArangoDB PATCH
// semantics), safe here because callers pass a fully-loaded struct.
func (p *provider) UpdateScimEndpoint(ctx context.Context, scimEndpoint *schemas.ScimEndpoint) (*schemas.ScimEndpoint, error) {
	if scimEndpoint.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateScimEndpoint: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	scimEndpoint.UpdatedAt = time.Now().Unix()
	scimEndpointCollection, _ := p.db.Collection(ctx, schemas.Collections.ScimEndpoint)
	doc, err := structToDocument(scimEndpoint)
	if err != nil {
		return nil, err
	}
	meta, err := scimEndpointCollection.UpdateDocument(ctx, scimEndpoint.Key, doc)
	if err != nil {
		return nil, err
	}
	scimEndpoint.Key = meta.Key
	scimEndpoint.ID = meta.ID.String()
	return scimEndpoint, nil
}

// DeleteScimEndpoint removes a SCIM endpoint record.
func (p *provider) DeleteScimEndpoint(ctx context.Context, scimEndpoint *schemas.ScimEndpoint) error {
	scimEndpointCollection, _ := p.db.Collection(ctx, schemas.Collections.ScimEndpoint)
	_, err := scimEndpointCollection.RemoveDocument(ctx, scimEndpoint.Key)
	if err != nil {
		return err
	}
	return nil
}

// GetScimEndpointByID fetches a SCIM endpoint by primary key.
// Filters on _key, not _id: every real caller holds the bare id
// AsAPIScimEndpoint exposes, never the full "collection/key" handle.
func (p *provider) GetScimEndpointByID(ctx context.Context, id string) (*schemas.ScimEndpoint, error) {
	var scimEndpoint *schemas.ScimEndpoint
	query := fmt.Sprintf("FOR d in %s FILTER d._key == @id LIMIT 1 RETURN d", schemas.Collections.ScimEndpoint)
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
			if scimEndpoint == nil {
				return nil, fmt.Errorf("scim endpoint not found")
			}
			break
		}
		s := &schemas.ScimEndpoint{}
		if _, err := readDocument(ctx, cursor, s); err != nil {
			return nil, err
		}
		scimEndpoint = s
	}
	return scimEndpoint, nil
}

// GetScimEndpointByOrgID fetches the SCIM endpoint for an organization.
// OrgID is unique: one SCIM endpoint per org.
func (p *provider) GetScimEndpointByOrgID(ctx context.Context, orgID string) (*schemas.ScimEndpoint, error) {
	var scimEndpoint *schemas.ScimEndpoint
	query := fmt.Sprintf("FOR d in %s FILTER d.org_id == @org_id LIMIT 1 RETURN d", schemas.Collections.ScimEndpoint)
	bindVars := map[string]interface{}{
		"org_id": orgID,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if scimEndpoint == nil {
				return nil, fmt.Errorf("scim endpoint not found")
			}
			break
		}
		s := &schemas.ScimEndpoint{}
		if _, err := readDocument(ctx, cursor, s); err != nil {
			return nil, err
		}
		scimEndpoint = s
	}
	return scimEndpoint, nil
}
