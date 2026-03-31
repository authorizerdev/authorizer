package schemas

// Note: any change here should be reflected in providers/casandra/provider.go as it does not have model support in collection creation

// LoginAttempt tracks individual login attempts for sliding window rate limiting
type LoginAttempt struct {
	Key         string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // for arangodb
	ID          string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`
	Email       string `gorm:"type:varchar(256);index" json:"email" bson:"email" cql:"email" dynamo:"email"`
	IPAddress   string `gorm:"type:varchar(45)" json:"ip_address" bson:"ip_address" cql:"ip_address" dynamo:"ip_address"`
	Successful  bool   `json:"successful" bson:"successful" cql:"successful" dynamo:"successful"`
	AttemptedAt int64  `gorm:"index" json:"attempted_at" bson:"attempted_at" cql:"attempted_at" dynamo:"attempted_at"`
	CreatedAt   int64  `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
}
