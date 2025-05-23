package schemas

const (
	// FieldName email is the field name for email
	FieldNameEmail = "email"
	// FieldNamePhoneNumber is the field name for phone number
	FieldNamePhoneNumber = "phone_number"
)

// OTP model for database
type OTP struct {
	Key         string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // for arangodb
	ID          string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`
	Email       string `gorm:"index" json:"email" bson:"email" cql:"email" dynamo:"email" index:"email,hash"`
	PhoneNumber string `gorm:"index" json:"phone_number" bson:"phone_number" cql:"phone_number" dynamo:"phone_number"`
	Otp         string `json:"otp" bson:"otp" cql:"otp" dynamo:"otp"`
	ExpiresAt   int64  `json:"expires_at" bson:"expires_at" cql:"expires_at" dynamo:"expires_at"`
	CreatedAt   int64  `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt   int64  `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

type Paging struct {
	ID string `json:"id,omitempty" dynamo:"id,hash"`
}
