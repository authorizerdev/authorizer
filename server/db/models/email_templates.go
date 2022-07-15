package models

import (
	"strings"

	"github.com/authorizerdev/authorizer/server/graph/model"
)

// EmailTemplate model for database
type EmailTemplate struct {
	Key       string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty"` // for arangodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id"`
	EventName string `gorm:"unique" json:"event_name" bson:"event_name" cql:"event_name"`
	Template  string `gorm:"type:text" json:"endpoint" bson:"endpoint" cql:"endpoint"`
	CreatedAt int64  `json:"created_at" bson:"created_at" cql:"created_at"`
	UpdatedAt int64  `json:"updated_at" bson:"updated_at" cql:"updated_at"`
}

// AsAPIEmailTemplate to return email template as graphql response object
func (e *EmailTemplate) AsAPIEmailTemplate() *model.EmailTemplate {
	id := e.ID
	if strings.Contains(id, Collections.EmailTemplate+"/") {
		id = strings.TrimPrefix(id, Collections.EmailTemplate+"/")
	}
	return &model.EmailTemplate{
		ID:        id,
		EventName: e.EventName,
		Template:  e.Template,
		CreatedAt: &e.CreatedAt,
		UpdatedAt: &e.UpdatedAt,
	}
}
