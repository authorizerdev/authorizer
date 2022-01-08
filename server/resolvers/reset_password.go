package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

func ResetPassword(ctx context.Context, params model.ResetPasswordInput) (*model.Response, error) {
	var res *model.Response
	if constants.DISABLE_BASIC_AUTHENTICATION {
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

	password, _ := utils.HashPassword(params.Password)
	user.Password = &password

	signupMethod := user.SignupMethods
	if !strings.Contains(signupMethod, enum.BasicAuth.String()) {
		signupMethod = signupMethod + "," + enum.BasicAuth.String()
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
