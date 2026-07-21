package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

// groupScanSafetyCap bounds the org-scoped scan-and-compare-in-app
// displayName lookup below. It is NOT a realistic ceiling on an org's group
// count, purely a circuit-breaker against unbounded work on a pathologically
// large org, mirroring the equivalent cap in the SQL/Cassandra/DynamoDB/
// ArangoDB providers.
const groupScanSafetyCap = 100_000

// GetScimGroupByOrgAndDisplayName resolves the single group with the given
// displayName within an org. displayName is compared case-insensitively:
// SCIM Group.displayName is caseExact:false (RFC 7644 §3.4.2.2).
//
// This deliberately does NOT use N1QL's LOWER() for the comparison, for the
// same reason the SQL provider doesn't: engine-native case folding is not
// guaranteed to agree with Go's strings.ToLower/EqualFold for non-ASCII
// display names, which would make this provider disagree with the others on
// a "same" displayName. Fetching by org (indexed, cheap) and comparing with
// strings.EqualFold in Go instead makes every storage backend fold
// identically, matching the SQL/Cassandra/DynamoDB/ArangoDB fetch-then-filter
// shape.
func (p *provider) GetScimGroupByOrgAndDisplayName(ctx context.Context, orgID, displayName string) (*schemas.ScimGroup, error) {
	params := make(map[string]interface{}, 2)
	params["org_id"] = orgID
	params["cap"] = groupScanSafetyCap
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE org_id=$org_id LIMIT $cap`, scimGroupColumns, p.scopeName, schemas.Collections.ScimGroup)
	queryResult, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
		NamedParameters: params,
	})
	if err != nil {
		return nil, err
	}
	examined := 0
	for queryResult.Next() {
		var raw json.RawMessage
		if err := queryResult.Row(&raw); err != nil {
			return nil, err
		}
		group := &schemas.ScimGroup{}
		if err := decodeDocument(raw, group); err != nil {
			return nil, err
		}
		if strings.EqualFold(group.DisplayName, displayName) {
			return group, nil
		}
		examined++
	}
	if err := queryResult.Err(); err != nil {
		return nil, err
	}
	if examined >= groupScanSafetyCap {
		p.dependencies.Log.Warn().Str("org_id", orgID).Int("examined", examined).
			Msg("GetScimGroupByOrgAndDisplayName: hit the scan safety cap without a match")
	}
	return nil, fmt.Errorf("scim group not found")
}

// GetScimGroupByOrgAndExternalID resolves the single group with the given
// externalId within an org. externalId is stored org-namespaced ("<orgID>:<raw>")
// exactly like User.ExternalID, so this can never resolve another org's group.
func (p *provider) GetScimGroupByOrgAndExternalID(ctx context.Context, orgID, externalID string) (*schemas.ScimGroup, error) {
	params := make(map[string]interface{}, 2)
	params["org_id"] = orgID
	params["external_id"] = orgID + ":" + externalID
	query := fmt.Sprintf(`SELECT %s FROM %s.%s WHERE org_id=$org_id AND external_id=$external_id LIMIT 1`, scimGroupColumns, p.scopeName, schemas.Collections.ScimGroup)
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
