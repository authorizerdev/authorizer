package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/parsers"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/validators"
)

// ResetPasswordResolver is a resolver for reset password mutation
func ResetPasswordResolver(ctx context.Context, params model.ResetPasswordInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	isBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication)
	if err != nil {
		log.Debug("Error getting basic auth disabled: ", err)
		isBasicAuthDisabled = true
	}
	if isBasicAuthDisabled {
		log.Debug("Basic authentication is disabled")
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}

	verificationRequest, err := db.Provider.GetVerificationRequestByToken(params.Token)
	if err != nil {
		log.Debug("Failed to get verification request: ", err)
		return res, fmt.Errorf(`invalid token`)
	}

	if params.Password != params.ConfirmPassword {
		log.Debug("Passwords do not match")
		return res, fmt.Errorf(`passwords don't match`)
	}

	if err := validators.IsValidPassword(params.Password); err != nil {
		log.Debug("Invalid password")
		return res, err
	}

	// verify if token exists in db
	hostname := parsers.GetHost(gc)
	claim, err := token.ParseJWTToken(params.Token)
	if err != nil {
		log.Debug("Failed to parse token: ", err)
		return res, fmt.Errorf(`invalid token`)
	}

	if ok, err := token.ValidateJWTClaims(claim, hostname, verificationRequest.Nonce, verificationRequest.Email); !ok || err != nil {
		log.Debug("Failed to validate jwt claims: ", err)
		return res, fmt.Errorf(`invalid token`)
	}

	email := claim["sub"].(string)
	log := log.WithFields(log.Fields{
		"email": email,
	})
	user, err := db.Provider.GetUserByEmail(email)
	if err != nil {
		log.Debug("Failed to get user: ", err)
		return res, err
	}

	password, _ := crypto.EncryptPassword(params.Password)
	user.Password = &password

	signupMethod := user.SignupMethods
	if !strings.Contains(signupMethod, constants.AuthRecipeMethodBasicAuth) {
		signupMethod = signupMethod + "," + constants.AuthRecipeMethodBasicAuth
	}
	user.SignupMethods = signupMethod

	// helpful if user has not signed up with basic auth
	if user.EmailVerifiedAt == nil {
		now := time.Now().Unix()
		user.EmailVerifiedAt = &now
	}

	// delete from verification table
	err = db.Provider.DeleteVerificationRequest(verificationRequest)
	if err != nil {
		log.Debug("Failed to delete verification request: ", err)
		return res, err
	}

	_, err = db.Provider.UpdateUser(user)
	if err != nil {
		log.Debug("Failed to update user: ", err)
		return res, err
	}

	res = &model.Response{
		Message: `Password updated successfully.`,
	}

	return res, nil
}
