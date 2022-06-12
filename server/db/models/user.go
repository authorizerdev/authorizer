package models

import (
	"strings"

	"github.com/authorizerdev/authorizer/server/graph/model"
)

// Note: any change here should be reflected in providers/casandra/provider.go as it does not have model support in collection creation

// User model for db
type User struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty"` // for arangodb
	ID  string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id"`

	Email                 string  `gorm:"unique" json:"email" bson:"email" cql:"email"`
	EmailVerifiedAt       *int64  `json:"email_verified_at" bson:"email_verified_at" cql:"email_verified_at"`
	Password              *string `gorm:"type:text" json:"password" bson:"password" cql:"password"`
	SignupMethods         string  `json:"signup_methods" bson:"signup_methods" cql:"signup_methods"`
	GivenName             *string `json:"given_name" bson:"given_name" cql:"given_name"`
	FamilyName            *string `json:"family_name" bson:"family_name" cql:"family_name"`
	MiddleName            *string `json:"middle_name" bson:"middle_name" cql:"middle_name"`
	Nickname              *string `json:"nickname" bson:"nickname" cql:"nickname"`
	Gender                *string `json:"gender" bson:"gender" cql:"gender"`
	Birthdate             *string `json:"birthdate" bson:"birthdate" cql:"birthdate"`
	PhoneNumber           *string `gorm:"unique" json:"phone_number" bson:"phone_number" cql:"phone_number"`
	PhoneNumberVerifiedAt *int64  `json:"phone_number_verified_at" bson:"phone_number_verified_at" cql:"phone_number_verified_at"`
	Picture               *string `gorm:"type:text" json:"picture" bson:"picture" cql:"picture"`
	Roles                 string  `json:"roles" bson:"roles" cql:"roles"`
	RevokedTimestamp      *int64  `json:"revoked_timestamp" bson:"revoked_timestamp" cql:"revoked_timestamp"`
	UpdatedAt             int64   `json:"updated_at" bson:"updated_at" cql:"updated_at"`
	CreatedAt             int64   `json:"created_at" bson:"created_at" cql:"created_at"`
}

func (user *User) AsAPIUser() *model.User {
	isEmailVerified := user.EmailVerifiedAt != nil
	isPhoneVerified := user.PhoneNumberVerifiedAt != nil
	email := user.Email
	createdAt := user.CreatedAt
	updatedAt := user.UpdatedAt
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
		RevokedTimestamp:    user.RevokedTimestamp,
		CreatedAt:           &createdAt,
		UpdatedAt:           &updatedAt,
	}
}
