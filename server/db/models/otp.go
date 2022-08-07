package models

// OTP model for database
type OTP struct {
	Key       string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty"` // for arangodb
	ID        string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id"`
	Email     string `gorm:"unique" json:"email" bson:"email" cql:"email"`
	Otp       string `json:"otp" bson:"otp" cql:"otp"`
	ExpiresAt int64  `json:"expires_at" bson:"expires_at" cql:"expires_at"`
	CreatedAt int64  `json:"created_at" bson:"created_at" cql:"created_at"`
	UpdatedAt int64  `json:"updated_at" bson:"updated_at" cql:"updated_at"`
}
