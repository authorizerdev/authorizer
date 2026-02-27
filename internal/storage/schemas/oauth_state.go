package schemas

// OAuthState model for storing OAuth state in database
// This replaces the in-memory storage for OAuth state when Redis is not configured
type OAuthState struct {
	Key       string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // for arangodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`
	StateKey  string `gorm:"type:varchar(255);uniqueIndex" json:"state_key" bson:"state_key" cql:"state_key" dynamo:"state_key" index:"state_key,hash"`
	State     string `gorm:"type:text" json:"state" bson:"state" cql:"state" dynamo:"state"`
	CreatedAt int64  `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt int64  `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}
