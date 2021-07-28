package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

func ResetPassword(ctx context.Context, params model.ResetPassowrdInput) (*model.Response, error) {
	var res *model.Response
	if constants.DISABLE_BASIC_AUTHENTICATION == "true" {
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}

	if params.Password != params.ConfirmPassword {
		return res, fmt.Errorf(`passwords don't match`)
	}

	_, err := db.Mgr.GetVerificationByToken(params.Token)
	if err != nil {
		return res, fmt.Errorf(`invalid token`)
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
	user.Password = password

	// delete from verification table
	db.Mgr.DeleteToken(claim.Email)
	db.Mgr.UpdateUser(user)

	res = &model.Response{
		Message: `Password updated successfully.`,
	}

	return res, nil
}
