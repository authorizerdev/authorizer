package utils

import (
	"strings"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
)

// TODO move this to provider
// rename it to AsAPIUser

func GetResponseUserData(user models.User) *model.User {
	isEmailVerified := user.EmailVerifiedAt != nil
	isPhoneVerified := user.PhoneNumberVerifiedAt != nil
	return &model.User{
		ID:                  user.ID,
		Email:               user.Email,
		EmailVerified:       isEmailVerified,
		SignupMethods:       user.SignupMethods,
		GivenName:           user.GivenName,
		FamilyName:          user.FamilyName,
		MiddleName:          user.MiddleName,
		Nickname:            user.Nickname,
		PreferredUsername:   &user.Email,
		Gender:              user.Gender,
		Birthdate:           user.Birthdate,
		PhoneNumber:         user.PhoneNumber,
		PhoneNumberVerified: &isPhoneVerified,
		Picture:             user.Picture,
		Roles:               strings.Split(user.Roles, ","),
		CreatedAt:           &user.CreatedAt,
		UpdatedAt:           &user.UpdatedAt,
	}
}
