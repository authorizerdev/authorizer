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

	sessionToken := session.GetToken(claim.ID)

	if sessionToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	// validate if all params are not empty
	if params.FirstName == nil && params.LastName == nil && params.Image == nil && params.OldPassword == nil && params.Email == nil {
		return res, fmt.Errorf("please enter atleast one param to update")
	}

	user, err := db.Mgr.GetUserByEmail(claim.Email)
	if err != nil {
		return res, err
	}

	if params.FirstName != nil && user.FirstName != *params.FirstName {
		user.FirstName = *params.FirstName
	}

	if params.LastName != nil && user.LastName != *params.LastName {
		user.LastName = *params.LastName
	}

	if params.Image != nil && user.Image != *params.Image {
		user.Image = *params.Image
	}

	if params.OldPassword != nil {
		if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(*params.OldPassword)); err != nil {
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

		user.Password = password
	}

	hasEmailChanged := false

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

		session.DeleteToken(fmt.Sprintf("%d", user.ID))
		utils.DeleteCookie(gc)

		user.Email = newEmail
		user.EmailVerifiedAt = 0
		hasEmailChanged = true
		// insert verification request
		verificationType := enum.UpdateEmail.String()
		token, err := utils.CreateVerificationToken(newEmail, verificationType, gc.Request.Host)
		if err != nil {
			log.Println(`Error generating token`, err)
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
		log.Println("Error updating user:", err)
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
