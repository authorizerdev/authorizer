package resolvers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
)

func UpdateUser(ctx context.Context, params model.UpdateUserInput) (*model.User, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.User
	if err != nil {
		return res, err
	}

	if !utils.IsSuperAdmin(gc) {
		return res, fmt.Errorf("unauthorized")
	}

	if params.GivenName == nil && params.FamilyName == nil && params.Picture == nil && params.MiddleName == nil && params.Nickname == nil && params.Email == nil && params.Birthdate == nil && params.Gender == nil && params.PhoneNumber == nil && params.Roles == nil {
		return res, fmt.Errorf("please enter atleast one param to update")
	}

	user, err := db.Mgr.GetUserByID(params.ID)
	if err != nil {
		return res, fmt.Errorf(`User not found`)
	}

	if params.GivenName != nil && user.GivenName != params.GivenName {
		user.GivenName = params.GivenName
	}

	if params.FamilyName != nil && user.FamilyName != params.FamilyName {
		user.FamilyName = params.FamilyName
	}

	if params.MiddleName != nil && user.MiddleName != params.MiddleName {
		user.MiddleName = params.MiddleName
	}

	if params.Nickname != nil && user.Nickname != params.Nickname {
		user.Nickname = params.Nickname
	}

	if params.Birthdate != nil && user.Birthdate != params.Birthdate {
		user.Birthdate = params.Birthdate
	}

	if params.Gender != nil && user.Gender != params.Gender {
		user.Gender = params.Gender
	}

	if params.PhoneNumber != nil && user.PhoneNumber != params.PhoneNumber {
		user.PhoneNumber = params.PhoneNumber
	}

	if params.Picture != nil && user.Picture != params.Picture {
		user.Picture = params.Picture
	}

	if params.Email != nil && user.Email != *params.Email {
		// check if valid email
		if !utils.IsValidEmail(*params.Email) {
			return res, fmt.Errorf("invalid email address")
		}
		newEmail := strings.ToLower(*params.Email)
		// check if user with new email exists
		_, err = db.Mgr.GetUserByEmail(newEmail)
		// err = nil means user exists
		if err == nil {
			return res, fmt.Errorf("user with this email address already exists")
		}

		session.DeleteUserSession(fmt.Sprintf("%v", user.ID))
		utils.DeleteCookie(gc)

		user.Email = newEmail
		user.EmailVerifiedAt = nil
		// insert verification request
		verificationType := enum.UpdateEmail.String()
		token, err := utils.CreateVerificationToken(newEmail, verificationType)
		if err != nil {
			log.Println(`error generating token`, err)
		}
		db.Mgr.AddVerification(db.VerificationRequest{
			Token:      token,
			Identifier: verificationType,
			ExpiresAt:  time.Now().Add(time.Minute * 30).Unix(),
			Email:      newEmail,
		})

		// exec it as go routin so that we can reduce the api latency
		go func() {
			utils.SendVerificationMail(newEmail, token)
		}()
	}

	rolesToSave := ""
	if params.Roles != nil && len(params.Roles) > 0 {
		currentRoles := strings.Split(user.Roles, ",")
		inputRoles := []string{}
		for _, item := range params.Roles {
			inputRoles = append(inputRoles, *item)
		}

		if !utils.IsValidRoles(append([]string{}, append(constants.EnvData.ROLES, constants.EnvData.PROTECTED_ROLES...)...), inputRoles) {
			return res, fmt.Errorf("invalid list of roles")
		}

		if !utils.IsStringArrayEqual(inputRoles, currentRoles) {
			rolesToSave = strings.Join(inputRoles, ",")
		}

		session.DeleteUserSession(fmt.Sprintf("%v", user.ID))
		utils.DeleteCookie(gc)
	}

	if rolesToSave != "" {
		user.Roles = rolesToSave
	}

	user, err = db.Mgr.UpdateUser(user)
	if err != nil {
		log.Println("error updating user:", err)
		return res, err
	}

	res = &model.User{
		ID:         params.ID,
		Email:      user.Email,
		Picture:    user.Picture,
		GivenName:  user.GivenName,
		FamilyName: user.FamilyName,
		Roles:      strings.Split(user.Roles, ","),
		CreatedAt:  &user.CreatedAt,
		UpdatedAt:  &user.UpdatedAt,
	}
	return res, nil
}
