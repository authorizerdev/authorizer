package schemas

// Permission is the binding layer of the authorization model.
// It connects a Resource to Scopes (via PermissionScope) and Policies (via PermissionPolicy).
// A permission answers: "WHO can do WHAT on WHICH resource?"
type Permission struct {
	// ID is the unique identifier (UUID v4).
	ID string `json:"id" gorm:"primaryKey;type:char(36)" bson:"_id" cql:"id" dynamo:"id,hash"`
	// Key is an alias for ID used by some NoSQL providers.
	Key string `json:"key" gorm:"type:char(36)" bson:"key" cql:"key" dynamo:"key"`
	// Name is a unique human-readable identifier (e.g., "edit-documents").
	Name string `json:"name" gorm:"type:varchar(100);uniqueIndex" bson:"name" cql:"name" dynamo:"name"`
	// Description provides optional context about this permission.
	Description string `json:"description" gorm:"type:text" bson:"description" cql:"description" dynamo:"description"`
	// ResourceID is the foreign key to the Resource this permission protects.
	ResourceID string `json:"resource_id" gorm:"type:char(36);index" bson:"resource_id" cql:"resource_id" dynamo:"resource_id"`
	// DecisionStrategy controls how multiple policies attached to this permission are evaluated.
	// "affirmative" = any policy grants access (OR), "unanimous" = all must agree (AND).
	DecisionStrategy string `json:"decision_strategy" gorm:"type:varchar(20);default:affirmative" bson:"decision_strategy" cql:"decision_strategy" dynamo:"decision_strategy"`
	// CreatedAt is the unix timestamp of creation.
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	// UpdatedAt is the unix timestamp of last update.
	UpdatedAt int64 `json:"updated_at" gorm:"autoUpdateTime" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

// PermissionScope is the join table linking a Permission to its allowed Scopes.
// A permission can cover multiple scopes (e.g., "read" and "write").
type PermissionScope struct {
	// ID is the unique identifier (UUID v4).
	ID string `json:"id" gorm:"primaryKey;type:char(36)" bson:"_id" cql:"id" dynamo:"id,hash"`
	// Key is an alias for ID used by some NoSQL providers.
	Key string `json:"key" gorm:"type:char(36)" bson:"key" cql:"key" dynamo:"key"`
	// PermissionID is the foreign key to the parent Permission.
	PermissionID string `json:"permission_id" gorm:"type:char(36);index;uniqueIndex:idx_ps_unique" bson:"permission_id" cql:"permission_id" dynamo:"permission_id"`
	// ScopeID is the foreign key to the Scope.
	ScopeID string `json:"scope_id" gorm:"type:char(36);index;uniqueIndex:idx_ps_unique" bson:"scope_id" cql:"scope_id" dynamo:"scope_id"`
	// CreatedAt is the unix timestamp of creation.
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime" bson:"created_at" cql:"created_at" dynamo:"created_at"`
}

// PermissionPolicy is the join table linking a Permission to its governing Policies.
// A permission can be governed by multiple policies, evaluated using the permission's DecisionStrategy.
type PermissionPolicy struct {
	// ID is the unique identifier (UUID v4).
	ID string `json:"id" gorm:"primaryKey;type:char(36)" bson:"_id" cql:"id" dynamo:"id,hash"`
	// Key is an alias for ID used by some NoSQL providers.
	Key string `json:"key" gorm:"type:char(36)" bson:"key" cql:"key" dynamo:"key"`
	// PermissionID is the foreign key to the parent Permission.
	PermissionID string `json:"permission_id" gorm:"type:char(36);index;uniqueIndex:idx_pp_unique" bson:"permission_id" cql:"permission_id" dynamo:"permission_id"`
	// PolicyID is the foreign key to the Policy.
	PolicyID string `json:"policy_id" gorm:"type:char(36);index;uniqueIndex:idx_pp_unique" bson:"policy_id" cql:"policy_id" dynamo:"policy_id"`
	// CreatedAt is the unix timestamp of creation.
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime" bson:"created_at" cql:"created_at" dynamo:"created_at"`
}

// PermissionWithPolicies is a denormalized view used by the evaluation engine.
// It bundles a permission with its resolved policies and targets for efficient
// single-query evaluation. Not a database table -- constructed by
// GetPermissionsForResourceScope().
type PermissionWithPolicies struct {
	// PermissionID is the permission being evaluated.
	PermissionID string
	// PermissionName is for logging and debugging.
	PermissionName string
	// DecisionStrategy is how to combine policy results for this permission.
	DecisionStrategy string
	// Policies contains the resolved policies with their targets.
	Policies []PolicyWithTargets
}

// PolicyWithTargets bundles a policy with its resolved targets.
// Used by the evaluation engine to avoid N+1 queries.
type PolicyWithTargets struct {
	// PolicyID is the policy identifier.
	PolicyID string
	// PolicyName is for logging and debugging.
	PolicyName string
	// Type is the policy type discriminator (role, user, client, agent).
	Type string
	// Logic is positive or negative.
	Logic string
	// DecisionStrategy is how to combine targets within this policy.
	DecisionStrategy string
	// Targets are the resolved policy targets.
	Targets []PolicyTargetView
}

// PolicyTargetView is a read-only view of a policy target for evaluation.
type PolicyTargetView struct {
	// TargetType is "role", "user", "client", or "agent".
	TargetType string
	// TargetValue is the role name, user ID, client ID, or agent ID.
	TargetValue string
}
