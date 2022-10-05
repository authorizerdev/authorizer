package models

import (
	"strings"

	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/refs"
)

// Note: any change here should be reflected in providers/casandra/provider.go as it does not have model support in collection creation

// WebhookLog model for db
type WebhookLog struct {
	Key        string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // for arangodb
	ID         string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`
	HttpStatus int64  `json:"http_status" bson:"http_status" cql:"http_status" dynamo:"http_status"`
	Response   string `gorm:"type:text" json:"response" bson:"response" cql:"response" dynamo:"response"`
	Request    string `gorm:"type:text" json:"request" bson:"request" cql:"request" dynamo:"request"`
	WebhookID  string `gorm:"type:char(36)" json:"webhook_id" bson:"webhook_id" cql:"webhook_id" dynamo:"webhook_id" index:"webhook_id,hash"`
	CreatedAt  int64  `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt  int64  `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

// AsAPIWebhookLog to return webhook log as graphql response object
func (w *WebhookLog) AsAPIWebhookLog() *model.WebhookLog {
	id := w.ID
	if strings.Contains(id, Collections.WebhookLog+"/") {
		id = strings.TrimPrefix(id, Collections.WebhookLog+"/")
	}
	return &model.WebhookLog{
		ID:         id,
		HTTPStatus: refs.NewInt64Ref(w.HttpStatus),
		Response:   refs.NewStringRef(w.Response),
		Request:    refs.NewStringRef(w.Request),
		WebhookID:  refs.NewStringRef(w.WebhookID),
		CreatedAt:  refs.NewInt64Ref(w.CreatedAt),
		UpdatedAt:  refs.NewInt64Ref(w.UpdatedAt),
	}
}
