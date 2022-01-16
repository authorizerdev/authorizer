package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

// ResetPasswordResolver is a resolver for reset password mutation
func ResetPasswordResolver(ctx context.Context, params model.ResetPasswordInput) (*model.Response, error) {
	var res *model.Response
	if envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDisableBasicAuthentication).(bool) {
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}

	verificationRequest, err := db.Mgr.GetVerificationByToken(params.Token)
	if err != nil {
		return res, fmt.Errorf(`invalid token`)
	}

	if params.Password != params.ConfirmPassword {
		return res, fmt.Errorf(`passwords don't match`)
	}

	// verify if token exists in db
	claim, err := utils.VerifyVerificationToken(params.Token)
	if err != nil {
		return res, fmt.Errorf(`invalid token`)
	}

	user, err := db.Mgr.GetUserByEmail(claim.Email)
	if err != nil {
		return res, err
	}

	password, _ := utils.EncryptPassword(params.Password)
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
	db.Mgr.DeleteVerificationRequest(verificationRequest)
	db.Mgr.UpdateUser(user)

	res = &model.Response{
		Message: `Password updated successfully.`,
	}

	return res, nil
}
