package models

import (
	"strings"

	"github.com/authorizerdev/authorizer/server/graph/model"
)

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
	UpdatedAt             int64   `json:"updated_at" bson:"updated_at"`
	CreatedAt             int64   `json:"created_at" bson:"created_at"`
	RevokedTimestamp      int64   `json:"revoked_timestamp" bson:"revoked_timestamp"`
}

func (user *User) AsAPIUser() *model.User {
	isEmailVerified := user.EmailVerifiedAt != nil
	isPhoneVerified := user.PhoneNumberVerifiedAt != nil
	email := user.Email
	createdAt := user.CreatedAt
	updatedAt := user.UpdatedAt
	revokedTimestamp := user.RevokedTimestamp
	return &model.User{
		ID:                  user.ID,
		Email:               user.Email,
		EmailVerified:       isEmailVerified,
		SignupMethods:       user.SignupMethods,
		GivenName:           user.GivenName,
		FamilyName:          user.FamilyName,
		MiddleName:          user.MiddleName,
		Nickname:            user.Nickname,
		PreferredUsername:   &email,
		Gender:              user.Gender,
		Birthdate:           user.Birthdate,
		PhoneNumber:         user.PhoneNumber,
		PhoneNumberVerified: &isPhoneVerified,
		Picture:             user.Picture,
		Roles:               strings.Split(user.Roles, ","),
		CreatedAt:           &createdAt,
		UpdatedAt:           &updatedAt,
		RevokedTimestamp:    &revokedTimestamp,
	}
}
