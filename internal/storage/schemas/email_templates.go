package schemas

import (
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// EmailTemplate model for database
type EmailTemplate struct {
	Key       string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // for arangodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`
	EventName string `gorm:"unique" json:"event_name" bson:"event_name" cql:"event_name" dynamo:"event_name" index:"event_name,hash"`
	Subject   string `json:"subject" bson:"subject" cql:"subject" dynamo:"subject"`
	Template  string `json:"template" bson:"template" cql:"template" dynamo:"template"`
	Design    string `json:"design" bson:"design" cql:"design" dynamo:"design"`
	CreatedAt int64  `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt int64  `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
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
		Subject:   e.Subject,
		Template:  e.Template,
		Design:    e.Design,
		CreatedAt: refs.NewInt64Ref(e.CreatedAt),
		UpdatedAt: refs.NewInt64Ref(e.UpdatedAt),
	}
}
