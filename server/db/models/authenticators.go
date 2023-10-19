package models

// Note: any change here should be reflected in providers/casandra/provider.go as it does not have model support in collection creation

// Authenticators model for db
type Authenticators struct {
	Key          string  `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // for arangodb
	ID           string  `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`
	UserID       string  `gorm:"type:char(36)" json:"user_id" bson:"user_id" cql:"user_id" dynamo:"user_id" index:"user_id,hash"`
	Method       string  `json:"method" bson:"method" cql:"method" dynamo:"method"`
	Secret       string  `json:"secret" bson:"secret" cql:"secret" dynamo:"secret"`
	RecoveryCode *string `json:"recovery_code" bson:"recovery_code" cql:"recovery_code" dynamo:"recovery_code"`
	VerifiedAt   *int64  `json:"verified_at" bson:"verified_at" cql:"verified_at" dynamo:"verified_at"`
	CreatedAt    int64   `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt    int64   `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}
