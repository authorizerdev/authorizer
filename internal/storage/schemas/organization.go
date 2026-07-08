package schemas

import (
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// Organization is a tenant grouping that owns per-org memberships (and, in
// later phases, SSO/SCIM connections and (org, client) grants). It is managed
// by platform admins through the `_organization`/`_organizations` admin API.
//
// Name is a unique, URL-safe slug (the stable external identifier); DisplayName
// is the human-readable label. Enabled gates whether the organization is
// active; disabling it does not delete its memberships.
//
// Note: any field addition must also be reflected in the cassandradb provider;
// it does not use GORM AutoMigrate for collection creation.
type Organization struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // ArangoDB document key

	ID string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`

	// Name is the unique organization slug (e.g. "acme-corp").
	Name string `gorm:"uniqueIndex" json:"name" bson:"name" cql:"name" dynamo:"name" index:"name,hash"`

	// DisplayName is the human-readable organization name.
	DisplayName *string `json:"display_name" bson:"display_name" cql:"display_name" dynamo:"display_name"`

	// Enabled controls whether the organization is active.
	//
	// GORM NOTE: gorm:"default:true" means db.Create with Enabled=false will
	// persist as true (Go zero-value triggers the column default). The service
	// layer must always set Enabled explicitly and use Save (not Create-only)
	// when creating a disabled organization.
	Enabled bool `json:"enabled" bson:"enabled" cql:"enabled" dynamo:"enabled" gorm:"default:true"`

	CreatedAt int64 `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt int64 `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

// AsAPIOrganization converts the storage record into the GraphQL model.
func (o *Organization) AsAPIOrganization() *model.Organization {
	id := o.ID
	if strings.Contains(id, Collections.Organization+"/") {
		id = strings.TrimPrefix(id, Collections.Organization+"/")
	}
	return &model.Organization{
		ID:          id,
		Name:        o.Name,
		DisplayName: o.DisplayName,
		Enabled:     o.Enabled,
		CreatedAt:   refs.NewInt64Ref(o.CreatedAt),
		UpdatedAt:   refs.NewInt64Ref(o.UpdatedAt),
	}
}
