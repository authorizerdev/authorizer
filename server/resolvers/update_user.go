package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/email"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// UpdateUserResolver is a resolver for update user mutation
// This is admin only mutation
func UpdateUserResolver(ctx context.Context, params model.UpdateUserInput) (*model.User, error) {
	var res *model.User

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return res, fmt.Errorf("unauthorized")
	}

	if params.ID == "" {
		log.Debug("UserID is empty")
		return res, fmt.Errorf("User ID is required")
	}

	log := log.WithFields(log.Fields{
		"user_id": params.ID,
	})

	if params.GivenName == nil && params.FamilyName == nil && params.Picture == nil && params.MiddleName == nil && params.Nickname == nil && params.Email == nil && params.Birthdate == nil && params.Gender == nil && params.PhoneNumber == nil && params.Roles == nil {
		log.Debug("No params to update")
		return res, fmt.Errorf("please enter atleast one param to update")
	}

	user, err := db.Provider.GetUserByID(params.ID)
	if err != nil {
		log.Debug("Failed to get user by id: ", err)
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

	if params.EmailVerified != nil {
		if *params.EmailVerified {
			now := time.Now().Unix()
			user.EmailVerifiedAt = &now
		} else {
			user.EmailVerifiedAt = nil
		}
	}

	if params.Email != nil && user.Email != *params.Email {
		// check if valid email
		if !utils.IsValidEmail(*params.Email) {
			log.Debug("Invalid email: ", *params.Email)
			return res, fmt.Errorf("invalid email address")
		}
		newEmail := strings.ToLower(*params.Email)
		// check if user with new email exists
		_, err = db.Provider.GetUserByEmail(newEmail)
		// err = nil means user exists
		if err == nil {
			log.Debug("User with email already exists: ", newEmail)
			return res, fmt.Errorf("user with this email address already exists")
		}

		// TODO figure out how to do this
		go memorystore.Provider.DeleteAllUserSession(user.ID)

		hostname := utils.GetHost(gc)
		user.Email = newEmail
		user.EmailVerifiedAt = nil
		// insert verification request
		_, nonceHash, err := utils.GenerateNonce()
		if err != nil {
			log.Debug("Failed to generate nonce: ", err)
			return res, err
		}
		verificationType := constants.VerificationTypeUpdateEmail
		redirectURL := utils.GetAppURL(gc)
		verificationToken, err := token.CreateVerificationToken(newEmail, verificationType, hostname, nonceHash, redirectURL)
		if err != nil {
			log.Debug("Failed to create verification token: ", err)
		}
		_, err = db.Provider.AddVerificationRequest(models.VerificationRequest{
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

	rolesToSave := ""
	if params.Roles != nil && len(params.Roles) > 0 {
		currentRoles := strings.Split(user.Roles, ",")
		inputRoles := []string{}
		for _, item := range params.Roles {
			inputRoles = append(inputRoles, *item)
		}

		if !utils.IsValidRoles(inputRoles, append([]string{}, append(envstore.EnvStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyRoles), envstore.EnvStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyProtectedRoles)...)...)) {
			log.Debug("Invalid roles: ", params.Roles)
			return res, fmt.Errorf("invalid list of roles")
		}

		if !utils.IsStringArrayEqual(inputRoles, currentRoles) {
			rolesToSave = strings.Join(inputRoles, ",")
		}

		go memorystore.Provider.DeleteAllUserSession(user.ID)
	}

	if rolesToSave != "" {
		user.Roles = rolesToSave
	}

	user, err = db.Provider.UpdateUser(user)
	if err != nil {
		log.Debug("Failed to update user: ", err)
		return res, err
	}

	createdAt := user.CreatedAt
	updatedAt := user.UpdatedAt
	res = &model.User{
		ID:         params.ID,
		Email:      user.Email,
		Picture:    user.Picture,
		GivenName:  user.GivenName,
		FamilyName: user.FamilyName,
		Roles:      strings.Split(user.Roles, ","),
		CreatedAt:  &createdAt,
		UpdatedAt:  &updatedAt,
	}
	return res, nil
}
