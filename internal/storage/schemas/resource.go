package schemas

import "github.com/authorizerdev/authorizer/internal/graph/model"

// Resource represents a protected resource type in the authorization model.
// Resources are types (e.g., "document", "invoice"), not instances.
// They define WHAT is being protected.
type Resource struct {
	// ID is the unique identifier (UUID v4).
	ID string `json:"id" gorm:"primaryKey;type:char(36)" bson:"_id" cql:"id" dynamo:"id,hash"`
	// Key is an alias for ID used by some NoSQL providers.
	Key string `json:"key" gorm:"type:char(36)" bson:"key" cql:"key" dynamo:"key"`
	// Name is a unique human-readable identifier (e.g., "document", "invoice").
	// Must be alphanumeric with hyphens and underscores, max 100 chars.
	Name string `json:"name" gorm:"type:varchar(100);uniqueIndex" bson:"name" cql:"name" dynamo:"name"`
	// Description provides optional context about what this resource represents.
	Description string `json:"description" gorm:"type:text" bson:"description" cql:"description" dynamo:"description"`
	// CreatedAt is the unix timestamp of creation.
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	// UpdatedAt is the unix timestamp of last update.
	UpdatedAt int64 `json:"updated_at" gorm:"autoUpdateTime" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

// AsAPIResource converts a storage Resource to the GraphQL API model.
func (r *Resource) AsAPIResource() *model.AuthzResource {
	return &model.AuthzResource{
		ID:          r.ID,
		Name:        r.Name,
		Description: &r.Description,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
