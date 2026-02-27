package schemas

// SessionToken model for storing user session tokens in database
// This replaces the in-memory storage for session tokens when Redis is not configured
type SessionToken struct {
	Key       string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // for arangodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`
	UserID    string `gorm:"type:varchar(255);index:idx_session_token_user_id_key" json:"user_id" bson:"user_id" cql:"user_id" dynamo:"user_id" index:"user_id,hash"`
	KeyName   string `gorm:"type:varchar(255);index:idx_session_token_user_id_key" json:"key_name" bson:"key_name" cql:"key_name" dynamo:"key_name"`
	Token     string `gorm:"type:text" json:"token" bson:"token" cql:"token" dynamo:"token"`
	ExpiresAt int64  `gorm:"index" json:"expires_at" bson:"expires_at" cql:"expires_at" dynamo:"expires_at"`
	CreatedAt int64  `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt int64  `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}
