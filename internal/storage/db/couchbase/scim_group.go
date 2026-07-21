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

const scimGroupColumns = "_id, org_id, display_name, external_id, created_at, updated_at"

// AddScimGroup creates a new SCIM group record.
func (p *provider) AddScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if group.ID == "" {
		group.ID = uuid.New().String()
	}
	group.Key = group.ID
	now := time.Now().Unix()
	group.CreatedAt = now
	group.UpdatedAt = now
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	doc, err := structToDocument(group)
	if err != nil {
		return nil, err
	}
	_, err = p.db.Collection(schemas.Collections.ScimGroup).Insert(group.ID, doc, &insertOpt)
	if err != nil {
		return nil, err
	}
	return group, nil
}

// UpdateScimGroup updates a SCIM group record.
// Callers MUST load the existing record and mutate it before calling this
// method — a partial struct blanks fields it does not carry.
func (p *provider) UpdateScimGroup(ctx context.Context, group *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if group.CreatedAt == 0 {
		return nil, fmt.Errorf("UpdateScimGroup: caller must load record before updating (CreatedAt is zero — partial struct detected)")
	}
	group.UpdatedAt = time.Now().Unix()
	groupMap, err := structToDocument(group)
	if err != nil {
		return nil, err
	}
	updateFields, params := GetSetFields(groupMap)
	params["_id"] = group.ID
	query := fmt.Sprintf(`UPDATE %s.%s SET %s WHERE _id=$_id`, p.scopeName, schemas.Collections.ScimGroup, updateFields)
	_, err = p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	return group, nil
}

// DeleteScimGroup removes a SCIM group record.
func (p *provider) DeleteScimGroup(ctx context.Context, group *schemas.ScimGroup) error {
	removeOpt := gocb.RemoveOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.ScimGroup).Remove(group.ID, &removeOpt)
	if err != nil {
		return err
	}
	return nil
}

// GetScimGroupByID fetches a SCIM group by primary key.
func (p *provider) GetScimGroupByID(ctx context.Context, id string) (*schemas.ScimGroup, error) {
	params := make(map[string]interface{}, 1)
	params["_id"] = id
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE _id=$_id LIMIT 1`, scimGroupColumns, p.scopeName, schemas.Collections.ScimGroup)
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
	group := &schemas.ScimGroup{}
	if err := decodeDocument(raw, group); err != nil {
		return nil, err
	}
	return group, nil
}

// GetScimGroupByOrgAndDisplayName resolves the single group with the given
// displayName within an org.
func (p *provider) GetScimGroupByOrgAndDisplayName(ctx context.Context, orgID, displayName string) (*schemas.ScimGroup, error) {
	params := make(map[string]interface{}, 2)
	params["org_id"] = orgID
	params["display_name"] = displayName
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE org_id=$org_id AND display_name=$display_name LIMIT 1`, scimGroupColumns, p.scopeName, schemas.Collections.ScimGroup)
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
	group := &schemas.ScimGroup{}
	if err := decodeDocument(raw, group); err != nil {
		return nil, err
	}
	return group, nil
}
