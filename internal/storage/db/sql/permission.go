package sql

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddPermission creates a new authorization permission.
func (p *provider) AddPermission(ctx context.Context, permission *schemas.Permission) (*schemas.Permission, error) {
	if permission.ID == "" {
		permission.ID = uuid.New().String()
	}
	permission.Key = permission.ID
	permission.CreatedAt = time.Now().Unix()
	permission.UpdatedAt = time.Now().Unix()
	res := p.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&permission)
	if res.Error != nil {
		return nil, res.Error
	}
	return permission, nil
}

// UpdatePermission updates an existing authorization permission.
func (p *provider) UpdatePermission(ctx context.Context, permission *schemas.Permission) (*schemas.Permission, error) {
	permission.UpdatedAt = time.Now().Unix()
	result := p.db.Save(&permission)
	if result.Error != nil {
		return nil, result.Error
	}
	return permission, nil
}

// DeletePermission deletes an authorization permission by ID.
// Cascade-deletes associated permission_scopes and permission_policies.
func (p *provider) DeletePermission(ctx context.Context, id string) error {
	result := p.db.Where("permission_id = ?", id).Delete(&schemas.PermissionScope{})
	if result.Error != nil {
		return result.Error
	}
	result = p.db.Where("permission_id = ?", id).Delete(&schemas.PermissionPolicy{})
	if result.Error != nil {
		return result.Error
	}
	result = p.db.Where("id = ?", id).Delete(&schemas.Permission{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetPermissionByID returns an authorization permission by its ID.
func (p *provider) GetPermissionByID(ctx context.Context, id string) (*schemas.Permission, error) {
	var permission schemas.Permission
	result := p.db.Where("id = ?", id).First(&permission)
	if result.Error != nil {
		return nil, result.Error
	}
	return &permission, nil
}

// ListPermissions returns a paginated list of authorization permissions.
func (p *provider) ListPermissions(ctx context.Context, pagination *model.Pagination) ([]*schemas.Permission, *model.Pagination, error) {
	var permissions []*schemas.Permission
	result := p.db.Limit(int(pagination.Limit)).Offset(int(pagination.Offset)).Order("created_at DESC").Find(&permissions)
	if result.Error != nil {
		return nil, nil, result.Error
	}
	var total int64
	totalRes := p.db.Model(&schemas.Permission{}).Count(&total)
	if totalRes.Error != nil {
		return nil, nil, totalRes.Error
	}
	paginationClone := pagination
	paginationClone.Total = total
	return permissions, paginationClone, nil
}

// AddPermissionScope links a scope to a permission.
func (p *provider) AddPermissionScope(ctx context.Context, ps *schemas.PermissionScope) (*schemas.PermissionScope, error) {
	if ps.ID == "" {
		ps.ID = uuid.New().String()
	}
	ps.Key = ps.ID
	ps.CreatedAt = time.Now().Unix()
	res := p.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&ps)
	if res.Error != nil {
		return nil, res.Error
	}
	return ps, nil
}

// DeletePermissionScopesByPermissionID removes all scope links for a permission.
func (p *provider) DeletePermissionScopesByPermissionID(ctx context.Context, permissionID string) error {
	result := p.db.Where("permission_id = ?", permissionID).Delete(&schemas.PermissionScope{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetPermissionScopes returns all scope links for a permission.
func (p *provider) GetPermissionScopes(ctx context.Context, permissionID string) ([]*schemas.PermissionScope, error) {
	var scopes []*schemas.PermissionScope
	result := p.db.Where("permission_id = ?", permissionID).Find(&scopes)
	if result.Error != nil {
		return nil, result.Error
	}
	return scopes, nil
}

// AddPermissionPolicy links a policy to a permission.
func (p *provider) AddPermissionPolicy(ctx context.Context, pp *schemas.PermissionPolicy) (*schemas.PermissionPolicy, error) {
	if pp.ID == "" {
		pp.ID = uuid.New().String()
	}
	pp.Key = pp.ID
	pp.CreatedAt = time.Now().Unix()
	res := p.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&pp)
	if res.Error != nil {
		return nil, res.Error
	}
	return pp, nil
}

// DeletePermissionPoliciesByPermissionID removes all policy links for a permission.
func (p *provider) DeletePermissionPoliciesByPermissionID(ctx context.Context, permissionID string) error {
	result := p.db.Where("permission_id = ?", permissionID).Delete(&schemas.PermissionPolicy{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetPermissionPolicies returns all policy links for a permission.
func (p *provider) GetPermissionPolicies(ctx context.Context, permissionID string) ([]*schemas.PermissionPolicy, error) {
	var policies []*schemas.PermissionPolicy
	result := p.db.Where("permission_id = ?", permissionID).Find(&policies)
	if result.Error != nil {
		return nil, result.Error
	}
	return policies, nil
}

// permissionRow is an intermediate struct for scanning the multi-JOIN query result
// in GetPermissionsForResourceScope.
type permissionRow struct {
	PermissionID           string `gorm:"column:permission_id"`
	PermissionName         string `gorm:"column:permission_name"`
	DecisionStrategy       string `gorm:"column:decision_strategy"`
	PolicyID               string `gorm:"column:policy_id"`
	PolicyName             string `gorm:"column:policy_name"`
	PolicyType             string `gorm:"column:policy_type"`
	PolicyLogic            string `gorm:"column:policy_logic"`
	PolicyDecisionStrategy string `gorm:"column:policy_decision_strategy"`
	TargetType             string `gorm:"column:target_type"`
	TargetValue            string `gorm:"column:target_value"`
}

// GetPermissionsForResourceScope returns all permissions (with their policies and targets)
// that match a given resource name and scope name. This is the hot-path query used by
// the evaluation engine.
func (p *provider) GetPermissionsForResourceScope(ctx context.Context, resourceName string, scopeName string) ([]*schemas.PermissionWithPolicies, error) {
	query := `SELECT p.id AS permission_id, p.name AS permission_name, p.decision_strategy,
       pol.id AS policy_id, pol.name AS policy_name, pol.type AS policy_type, pol.logic AS policy_logic, pol.decision_strategy AS policy_decision_strategy,
       pt.target_type, pt.target_value
FROM ` + schemas.Prefix + `permissions p
JOIN ` + schemas.Prefix + `resources r ON r.id = p.resource_id
JOIN ` + schemas.Prefix + `permission_scopes ps ON ps.permission_id = p.id
JOIN ` + schemas.Prefix + `scopes s ON s.id = ps.scope_id
JOIN ` + schemas.Prefix + `permission_policies pp ON pp.permission_id = p.id
JOIN ` + schemas.Prefix + `policies pol ON pol.id = pp.policy_id
JOIN ` + schemas.Prefix + `policy_targets pt ON pt.policy_id = pol.id
WHERE r.name = ? AND s.name = ?`

	var rows []permissionRow
	result := p.db.Raw(query, resourceName, scopeName).Scan(&rows)
	if result.Error != nil {
		return nil, result.Error
	}

	return groupPermissionRows(rows), nil
}

// groupPermissionRows groups flat permissionRow results into nested PermissionWithPolicies structs.
func groupPermissionRows(rows []permissionRow) []*schemas.PermissionWithPolicies {
	// Track insertion order for permissions and policies
	permOrder := make([]string, 0)
	permMap := make(map[string]*schemas.PermissionWithPolicies)
	policyOrderMap := make(map[string][]string)              // permissionID -> ordered policy IDs
	policyMap := make(map[string]*schemas.PolicyWithTargets) // "permID:polID" -> policy

	for _, row := range rows {
		// Ensure permission exists
		perm, ok := permMap[row.PermissionID]
		if !ok {
			perm = &schemas.PermissionWithPolicies{
				PermissionID:     row.PermissionID,
				PermissionName:   row.PermissionName,
				DecisionStrategy: row.DecisionStrategy,
				Policies:         nil,
			}
			permMap[row.PermissionID] = perm
			permOrder = append(permOrder, row.PermissionID)
		}

		// Ensure policy exists within this permission
		policyKey := row.PermissionID + ":" + row.PolicyID
		pol, ok := policyMap[policyKey]
		if !ok {
			pol = &schemas.PolicyWithTargets{
				PolicyID:         row.PolicyID,
				PolicyName:       row.PolicyName,
				Type:             row.PolicyType,
				Logic:            row.PolicyLogic,
				DecisionStrategy: row.PolicyDecisionStrategy,
				Targets:          nil,
			}
			policyMap[policyKey] = pol
			policyOrderMap[row.PermissionID] = append(policyOrderMap[row.PermissionID], policyKey)
		}

		// Add target
		pol.Targets = append(pol.Targets, schemas.PolicyTargetView{
			TargetType:  row.TargetType,
			TargetValue: row.TargetValue,
		})
	}

	// Assemble in order
	result := make([]*schemas.PermissionWithPolicies, 0, len(permOrder))
	for _, permID := range permOrder {
		perm := permMap[permID]
		for _, policyKey := range policyOrderMap[permID] {
			pol := policyMap[policyKey]
			perm.Policies = append(perm.Policies, *pol)
		}
		result = append(result, perm)
	}
	return result
}
