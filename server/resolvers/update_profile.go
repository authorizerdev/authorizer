package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/email"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/parsers"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/validators"
	"golang.org/x/crypto/bcrypt"
)

// UpdateProfileResolver is resolver for update profile mutation
func UpdateProfileResolver(ctx context.Context, params model.UpdateProfileInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	accessToken, err := token.GetAccessToken(gc)
	if err != nil {
		log.Debug("Failed to get access token: ", err)
		return res, err
	}
	claims, err := token.ValidateAccessToken(gc, accessToken)
	if err != nil {
		log.Debug("Failed to validate access token: ", err)
		return res, err
	}

	// validate if all params are not empty
	if params.GivenName == nil && params.FamilyName == nil && params.Picture == nil && params.MiddleName == nil && params.Nickname == nil && params.OldPassword == nil && params.Email == nil && params.Birthdate == nil && params.Gender == nil && params.PhoneNumber == nil {
		log.Debug("All params are empty")
		return res, fmt.Errorf("please enter at least one param to update")
	}

	userID := claims["sub"].(string)
	log := log.WithFields(log.Fields{
		"user_id": userID,
	})

	user, err := db.Provider.GetUserByID(ctx, userID)
	if err != nil {
		log.Debug("Failed to get user by id: ", err)
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
			log.Debug("Failed to compare hash and old password: ", err)
			return res, fmt.Errorf("incorrect old password")
		}

		if params.NewPassword == nil {
			log.Debug("Failed to get new password: ")
			return res, fmt.Errorf("new password is required")
		}

		if params.ConfirmNewPassword == nil {
			log.Debug("Failed to get confirm new password: ")
			return res, fmt.Errorf("confirm password is required")
		}

		if *params.ConfirmNewPassword != *params.NewPassword {
			log.Debug("Failed to compare new password and confirm new password")
			return res, fmt.Errorf(`password and confirm password does not match`)
		}

		password, _ := crypto.EncryptPassword(*params.NewPassword)

		user.Password = &password
	}

	hasEmailChanged := false

	if params.Email != nil && user.Email != *params.Email {
		// check if valid email
		if !validators.IsValidEmail(*params.Email) {
			log.Debug("Failed to validate email: ", *params.Email)
			return res, fmt.Errorf("invalid email address")
		}
		newEmail := strings.ToLower(*params.Email)

		// check if valid email
		if !validators.IsValidEmail(newEmail) {
			log.Debug("Failed to validate new email: ", newEmail)
			return res, fmt.Errorf("invalid new email address")
		}
		// check if user with new email exists
		_, err := db.Provider.GetUserByEmail(ctx, newEmail)
		// err = nil means user exists
		if err == nil {
			log.Debug("Failed to get user by email: ", newEmail)
			return res, fmt.Errorf("user with this email address already exists")
		}

		go memorystore.Provider.DeleteAllUserSessions(user.ID)
		go cookie.DeleteSession(gc)

		user.Email = newEmail
		isEmailVerificationDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableEmailVerification)
		if err != nil {
			log.Debug("Failed to get disable email verification env variable: ", err)
			return res, err
		}
		if !isEmailVerificationDisabled {
			hostname := parsers.GetHost(gc)
			user.EmailVerifiedAt = nil
			hasEmailChanged = true
			// insert verification request
			_, nonceHash, err := utils.GenerateNonce()
			if err != nil {
				log.Debug("Failed to generate nonce: ", err)
				return res, err
			}
			verificationType := constants.VerificationTypeUpdateEmail
			redirectURL := parsers.GetAppURL(gc)
			verificationToken, err := token.CreateVerificationToken(newEmail, verificationType, hostname, nonceHash, redirectURL)
			if err != nil {
				log.Debug("Failed to create verification token: ", err)
				return res, err
			}
			_, err = db.Provider.AddVerificationRequest(ctx, models.VerificationRequest{
				Token:       verificationToken,
				Identifier:  verificationType,
				ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
				Email:       newEmail,
				Nonce:       nonceHash,
				RedirectURI: redirectURL,
			})
			if err != nil {
				log.Debug("Failed to add verification request: ", err)
				return res, err
			}

			// exec it as go routin so that we can reduce the api latency
			go email.SendVerificationMail(newEmail, verificationToken, hostname)

		}
	}
	_, err = db.Provider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug("Failed to update user: ", err)
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
