package models

// SMS verification requests model for database
type SMSVerificationRequest struct {
	ID            string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`
	PhoneNumber   string `gorm:"unique" json:"phone_number" bson:"phone_number" cql:"phone_number" dynamo:"phone_number" index:"phone_number,hash"`
	Code          string `json:"code" bson:"code" cql:"code" dynamo:"code"`
	CodeExpiresAt int64  `json:"code_expires_at" bson:"code_expires_at" cql:"code_expires_at" dynamo:"code_expires_at"`
	CreatedAt     int64  `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt     int64  `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}
