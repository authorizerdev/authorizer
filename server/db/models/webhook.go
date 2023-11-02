package models

import (
	"encoding/json"
	"strings"

	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/refs"
)

// Note: any change here should be reflected in providers/casandra/provider.go as it does not have model support in collection creation

// Event name has been kept unique as per initial design. But later on decided that we can have
// multiple hooks for same event so will be in a pattern `event_name-TIMESTAMP`

// Webhook model for db
type Webhook struct {
	Key              string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // for arangodb
	ID               string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`
	EventName        string `gorm:"unique" json:"event_name" bson:"event_name" cql:"event_name" dynamo:"event_name" index:"event_name,hash"`
	EventDescription string `json:"event_description" bson:"event_description" cql:"event_description" dynamo:"event_description"`
	EndPoint         string `json:"endpoint" bson:"endpoint" cql:"endpoint" dynamo:"endpoint"`
	Headers          string `json:"headers" bson:"headers" cql:"headers" dynamo:"headers"`
	Enabled          bool   `json:"enabled" bson:"enabled" cql:"enabled" dynamo:"enabled"`
	CreatedAt        int64  `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt        int64  `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

// AsAPIWebhook to return webhook as graphql response object
func (w *Webhook) AsAPIWebhook() *model.Webhook {
	headersMap := make(map[string]interface{})
	json.Unmarshal([]byte(w.Headers), &headersMap)
	id := w.ID
	if strings.Contains(id, Collections.Webhook+"/") {
		id = strings.TrimPrefix(id, Collections.Webhook+"/")
	}
	// set default title to event name without dot(.)
	if w.EventDescription == "" {
		w.EventDescription = strings.Join(strings.Split(w.EventName, "."), " ")
	}
	return &model.Webhook{
		ID:               id,
		EventName:        refs.NewStringRef(w.EventName),
		EventDescription: refs.NewStringRef(w.EventDescription),
		Endpoint:         refs.NewStringRef(w.EndPoint),
		Headers:          headersMap,
		Enabled:          refs.NewBoolRef(w.Enabled),
		CreatedAt:        refs.NewInt64Ref(w.CreatedAt),
		UpdatedAt:        refs.NewInt64Ref(w.UpdatedAt),
	}
}
