package resolvers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
	"golang.org/x/crypto/bcrypt"
)

func UpdateProfile(ctx context.Context, params model.UpdateProfileInput) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Response
	if err != nil {
		return res, err
	}

	token, err := utils.GetAuthToken(gc)
	if err != nil {
		return res, err
	}

	claim, err := utils.VerifyAuthToken(token)
	if err != nil {
		return res, err
	}

	id := fmt.Sprintf("%v", claim["id"])
	sessionToken := session.GetToken(id, token)

	if sessionToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	// validate if all params are not empty
	if params.GivenName == nil && params.FamilyName == nil && params.Picture == nil && params.MiddleName == nil && params.Nickname == nil && params.OldPassword == nil && params.Email == nil && params.Birthdate == nil && params.Gender == nil && params.PhoneNumber == nil {
		return res, fmt.Errorf("please enter atleast one param to update")
	}

	email := fmt.Sprintf("%v", claim["email"])
	user, err := db.Mgr.GetUserByEmail(email)
	if err != nil {
		return res, err
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

	if params.OldPassword != nil {
		if err = bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(*params.OldPassword)); err != nil {
			return res, fmt.Errorf("incorrect old password")
		}

		if params.NewPassword == nil {
			return res, fmt.Errorf("new password is required")
		}

		if params.ConfirmNewPassword == nil {
			return res, fmt.Errorf("confirm password is required")
		}

		if *params.ConfirmNewPassword != *params.NewPassword {
			return res, fmt.Errorf(`password and confirm password does not match`)
		}

		password, _ := utils.HashPassword(*params.NewPassword)

		user.Password = &password
	}

	hasEmailChanged := false

	if params.Email != nil && user.Email != *params.Email {
		// check if valid email
		if !utils.IsValidEmail(*params.Email) {
			return res, fmt.Errorf("invalid email address")
		}
		newEmail := strings.ToLower(*params.Email)
		// check if user with new email exists
		_, err := db.Mgr.GetUserByEmail(newEmail)

		// err = nil means user exists
		if err == nil {
			return res, fmt.Errorf("user with this email address already exists")
		}

		session.DeleteUserSession(fmt.Sprintf("%v", user.ID))
		utils.DeleteCookie(gc)

		user.Email = newEmail
		user.EmailVerifiedAt = nil
		hasEmailChanged = true
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

	_, err = db.Mgr.UpdateUser(user)
	if err != nil {
		log.Println("error updating user:", err)
		return res, err
	}
	message := `Profile details updated successfully.`
	if hasEmailChanged {
		message += `For the email change we have sent new verification email, please verify and continue`
	}
	res = &model.Response{
		Message: message,
	}

	return res, nil
}
