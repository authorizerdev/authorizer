package models

import (
	"encoding/json"
	"strings"

	"github.com/authorizerdev/authorizer/server/graph/model"
)

// Note: any change here should be reflected in providers/casandra/provider.go as it does not have model support in collection creation

// Webhook model for db
type Webhook struct {
	Key       string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty"` // for arangodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id"`
	EventName string `gorm:"unique" json:"event_name" bson:"event_name" cql:"event_name"`
	EndPoint  string `gorm:"type:text" json:"endpoint" bson:"endpoint" cql:"endpoint"`
	Headers   string `gorm:"type:text" json:"headers" bson:"headers" cql:"headers"`
	Enabled   bool   `json:"enabled" bson:"enabled" cql:"enabled"`
	CreatedAt int64  `json:"created_at" bson:"created_at" cql:"created_at"`
	UpdatedAt int64  `json:"updated_at" bson:"updated_at" cql:"updated_at"`
}

func (w *Webhook) AsAPIWebhook() *model.Webhook {
	headersMap := make(map[string]interface{})
	json.Unmarshal([]byte(w.Headers), &headersMap)

	id := w.ID
	if strings.Contains(id, Collections.Webhook+"/") {
		id = strings.TrimPrefix(id, Collections.Webhook+"/")
	}
	return &model.Webhook{
		ID:        id,
		EventName: &w.EventName,
		Endpoint:  &w.EndPoint,
		Headers:   headersMap,
		Enabled:   &w.Enabled,
		CreatedAt: &w.CreatedAt,
		UpdatedAt: &w.UpdatedAt,
	}
}
