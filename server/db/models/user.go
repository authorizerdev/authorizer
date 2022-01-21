package models

// User model for db
type User struct {
	Key string `json:"_key,omitempty" bson:"_key"` // for arangodb
	ID  string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id"`

	Email                 string  `gorm:"unique" json:"email" bson:"email"`
	EmailVerifiedAt       *int64  `json:"email_verified_at" bson:"email_verified_at"`
	Password              *string `gorm:"type:text" json:"password" bson:"password"`
	SignupMethods         string  `json:"signup_methods" bson:"signup_methods"`
	GivenName             *string `json:"given_name" bson:"given_name"`
	FamilyName            *string `json:"family_name" bson:"family_name"`
	MiddleName            *string `json:"middle_name" bson:"middle_name"`
	Nickname              *string `json:"nickname" bson:"nickname"`
	Gender                *string `json:"gender" bson:"gender"`
	Birthdate             *string `json:"birthdate" bson:"birthdate"`
	PhoneNumber           *string `gorm:"unique" json:"phone_number" bson:"phone_number"`
	PhoneNumberVerifiedAt *int64  `json:"phone_number_verified_at" bson:"phone_number_verified_at"`
	Picture               *string `gorm:"type:text" json:"picture" bson:"picture"`
	Roles                 string  `json:"roles" bson:"roles"`
	UpdatedAt             int64   `gorm:"autoUpdateTime" json:"updated_at" bson:"updated_at"`
	CreatedAt             int64   `gorm:"autoCreateTime" json:"created_at" bson:"created_at"`
}
