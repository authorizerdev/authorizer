package arangodb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddScimGroup creates a new SCIM group record.
func (p *provider) AddScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if group.ID == "" {
		group.ID = uuid.New().String()
	}
	group.Key = group.ID
	now := time.Now().Unix()
	group.CreatedAt = now
	group.UpdatedAt = now
	groupCollection, _ := p.db.Collection(ctx, schemas.Collections.ScimGroup)
	doc, err := structToDocument(group)
	if err != nil {
		return nil, err
	}
	meta, err := groupCollection.CreateDocument(ctx, doc)
	if err != nil {
		return nil, err
	}
	group.Key = meta.Key
	// ID is the portable bare identifier — matching every other provider's
	// contract (ID == Key == bare uuid) and the _key GetScimGroupByID filters on.
	// It is what URLs and FGA group-object ids are built from, so it must NOT be
	// arango's full "collection/key" handle (meta.ID.String()).
	group.ID = meta.Key
	return group, nil
}

// UpdateScimGroup updates a SCIM group record.
// Callers MUST load the existing record and mutate it before calling this
// method — this is a partial update via UpdateDocument (ArangoDB PATCH
// semantics), safe here because callers pass a fully-loaded struct.
func (p *provider) UpdateScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if group.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateScimGroup: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	group.UpdatedAt = time.Now().Unix()
	groupCollection, _ := p.db.Collection(ctx, schemas.Collections.ScimGroup)
	doc, err := structToDocument(group)
	if err != nil {
		return nil, err
	}
	meta, err := groupCollection.UpdateDocument(ctx, group.Key, doc)
	if err != nil {
		return nil, err
	}
	group.Key = meta.Key
	// Keep ID as the bare portable identifier (see AddScimGroup).
	group.ID = meta.Key
	return group, nil
}

// DeleteScimGroup removes a SCIM group record.
func (p *provider) DeleteScimGroup(ctx context.Context, group *schemas.ScimGroup) error {
	groupCollection, _ := p.db.Collection(ctx, schemas.Collections.ScimGroup)
	_, err := groupCollection.RemoveDocument(ctx, group.Key)
	if err != nil {
		return err
	}
	return nil
}

// GetScimGroupByID fetches a SCIM group by primary key. Filters on _key, not
// _id: every real caller holds the bare id, never the full "collection/key" handle.
func (p *provider) GetScimGroupByID(ctx context.Context, id string) (*schemas.ScimGroup, error) {
	var group *schemas.ScimGroup
	query := fmt.Sprintf("FOR d in %s FILTER d._key == @id LIMIT 1 RETURN d", schemas.Collections.ScimGroup)
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
			if group == nil {
				return nil, fmt.Errorf("scim group not found")
			}
			break
		}
		g := &schemas.ScimGroup{}
		if _, err := readDocument(ctx, cursor, g); err != nil {
			return nil, err
		}
		group = g
	}
	return group, nil
}

// GetScimGroupByOrgAndDisplayName resolves the single group with the given
// displayName within an org.
func (p *provider) GetScimGroupByOrgAndDisplayName(ctx context.Context, orgID, displayName string) (*schemas.ScimGroup, error) {
	var group *schemas.ScimGroup
	query := fmt.Sprintf("FOR d in %s FILTER d.org_id == @org_id AND d.display_name == @display_name LIMIT 1 RETURN d", schemas.Collections.ScimGroup)
	bindVars := map[string]interface{}{
		"org_id":       orgID,
		"display_name": displayName,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if group == nil {
				return nil, fmt.Errorf("scim group not found")
			}
			break
		}
		g := &schemas.ScimGroup{}
		if _, err := readDocument(ctx, cursor, g); err != nil {
			return nil, err
		}
		group = g
	}
	return group, nil
}

// GetScimGroupByOrgAndExternalID resolves the single group with the given
// externalId within an org. externalId is stored org-namespaced ("<orgID>:<raw>")
// exactly like User.ExternalID, so this can never resolve another org's group.
func (p *provider) GetScimGroupByOrgAndExternalID(ctx context.Context, orgID, externalID string) (*schemas.ScimGroup, error) {
	var group *schemas.ScimGroup
	query := fmt.Sprintf("FOR d in %s FILTER d.org_id == @org_id AND d.external_id == @external_id LIMIT 1 RETURN d", schemas.Collections.ScimGroup)
	bindVars := map[string]interface{}{
		"org_id":      orgID,
		"external_id": orgID + ":" + externalID,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close() }()
	for {
		if !cursor.HasMore() {
			if group == nil {
				return nil, fmt.Errorf("scim group not found")
			}
			break
		}
		g := &schemas.ScimGroup{}
		if _, err := readDocument(ctx, cursor, g); err != nil {
			return nil, err
		}
		group = g
	}
	return group, nil
}
