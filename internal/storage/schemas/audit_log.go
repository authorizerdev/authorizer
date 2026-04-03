package schemas

import (
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// Note: any change here should be reflected in providers/casandra/provider.go as it does not have model support in collection creation

// AuditLog model for db
type AuditLog struct {
	Key            string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // for arangodb
	ID             string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`
	Timestamp      int64  `json:"timestamp" bson:"timestamp" cql:"timestamp" dynamo:"timestamp"`
	ActorID        string `gorm:"type:char(36)" json:"actor_id" bson:"actor_id" cql:"actor_id" dynamo:"actor_id" index:"actor_id,hash"`
	ActorType      string `gorm:"type:varchar(30)" json:"actor_type" bson:"actor_type" cql:"actor_type" dynamo:"actor_type"`
	ActorEmail     string `gorm:"type:varchar(256)" json:"actor_email" bson:"actor_email" cql:"actor_email" dynamo:"actor_email"`
	Action         string `gorm:"type:varchar(100)" json:"action" bson:"action" cql:"action" dynamo:"action" index:"action,hash"`
	ResourceType   string `gorm:"type:varchar(50)" json:"resource_type" bson:"resource_type" cql:"resource_type" dynamo:"resource_type"`
	ResourceID     string `gorm:"type:char(36)" json:"resource_id" bson:"resource_id" cql:"resource_id" dynamo:"resource_id"`
	IPAddress      string `gorm:"type:varchar(45)" json:"ip_address" bson:"ip_address" cql:"ip_address" dynamo:"ip_address"`
	UserAgent      string `gorm:"type:text" json:"user_agent" bson:"user_agent" cql:"user_agent" dynamo:"user_agent"`
	Metadata       string `gorm:"type:text" json:"metadata" bson:"metadata" cql:"metadata" dynamo:"metadata"`
	OrganizationID string `gorm:"type:char(36)" json:"organization_id" bson:"organization_id" cql:"organization_id" dynamo:"organization_id"`
	CreatedAt      int64  `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt      int64  `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

// AsAPIAuditLog converts the database audit log to a GraphQL response object.
func (a *AuditLog) AsAPIAuditLog() *model.AuditLog {
	id := a.ID
	if strings.Contains(id, Collections.AuditLog+"/") {
		id = strings.TrimPrefix(id, Collections.AuditLog+"/")
	}
	return &model.AuditLog{
		ID:             id,
		Timestamp:      refs.NewInt64Ref(a.Timestamp),
		ActorID:        refs.NewStringRef(a.ActorID),
		ActorType:      refs.NewStringRef(a.ActorType),
		ActorEmail:     refs.NewStringRef(a.ActorEmail),
		Action:         refs.NewStringRef(a.Action),
		ResourceType:   refs.NewStringRef(a.ResourceType),
		ResourceID:     refs.NewStringRef(a.ResourceID),
		IPAddress:      refs.NewStringRef(a.IPAddress),
		UserAgent:      refs.NewStringRef(a.UserAgent),
		Metadata:       refs.NewStringRef(a.Metadata),
		OrganizationID: refs.NewStringRef(a.OrganizationID),
		CreatedAt:      refs.NewInt64Ref(a.CreatedAt),
		UpdatedAt:      refs.NewInt64Ref(a.UpdatedAt),
	}
}
