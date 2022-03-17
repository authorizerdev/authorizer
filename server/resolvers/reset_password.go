package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// ResetPasswordResolver is a resolver for reset password mutation
func ResetPasswordResolver(ctx context.Context, params model.ResetPasswordInput) (*model.Response, error) {
	var res *model.Response
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		return res, err
	}
	if envstore.EnvStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication) {
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}

	verificationRequest, err := db.Provider.GetVerificationRequestByToken(params.Token)
	if err != nil {
		return res, fmt.Errorf(`invalid token`)
	}

	if params.Password != params.ConfirmPassword {
		return res, fmt.Errorf(`passwords don't match`)
	}

	if !utils.IsValidPassword(params.Password) {
		return res, fmt.Errorf(`password is not valid. It needs to be at least 6 characters long and contain at least one number, one uppercase letter, one lowercase letter and one special character`)
	}

	// verify if token exists in db
	hostname := utils.GetHost(gc)
	claim, err := token.ParseJWTToken(params.Token, hostname, verificationRequest.Nonce, verificationRequest.Email)
	if err != nil {
		return res, fmt.Errorf(`invalid token`)
	}

	user, err := db.Provider.GetUserByEmail(claim["sub"].(string))
	if err != nil {
		return res, err
	}

	password, _ := crypto.EncryptPassword(params.Password)
	user.Password = &password

	signupMethod := user.SignupMethods
	if !strings.Contains(signupMethod, constants.SignupMethodBasicAuth) {
		signupMethod = signupMethod + "," + constants.SignupMethodBasicAuth
	}
	user.SignupMethods = signupMethod

	// helpful if user has not signed up with basic auth
	if user.EmailVerifiedAt == nil {
		now := time.Now().Unix()
		user.EmailVerifiedAt = &now
	}

	// delete from verification table
	db.Provider.DeleteVerificationRequest(verificationRequest)
	db.Provider.UpdateUser(user)

	res = &model.Response{
		Message: `Password updated successfully.`,
	}

	return res, nil
}
