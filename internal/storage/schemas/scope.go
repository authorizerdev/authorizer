package schemas

// Scope represents an action that can be performed on a resource.
// Scopes are global verbs (e.g., "read", "write", "delete", "approve").
// They define WHAT ACTIONS are allowed.
type Scope struct {
	// ID is the unique identifier (UUID v4).
	ID string `json:"id" gorm:"primaryKey;type:char(36)" bson:"_id" cql:"id" dynamo:"id,hash"`
	// Key is an alias for ID used by some NoSQL providers.
	Key string `json:"key" gorm:"type:char(36)" bson:"key" cql:"key" dynamo:"key"`
	// Name is a unique human-readable identifier (e.g., "read", "write").
	// Must be alphanumeric with hyphens and underscores, max 100 chars.
	Name string `json:"name" gorm:"type:varchar(100);uniqueIndex" bson:"name" cql:"name" dynamo:"name"`
	// Description provides optional context about what this scope represents.
	Description string `json:"description" gorm:"type:text" bson:"description" cql:"description" dynamo:"description"`
	// CreatedAt is the unix timestamp of creation.
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	// UpdatedAt is the unix timestamp of last update.
	UpdatedAt int64 `json:"updated_at" gorm:"autoUpdateTime" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}
