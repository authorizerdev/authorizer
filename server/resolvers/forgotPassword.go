package resolvers

import (
	"context"
	"fmt"

	"github.com/yauthdev/yauth/server/db"
	"github.com/yauthdev/yauth/server/graph/model"
	"github.com/yauthdev/yauth/server/utils"
)

func ForgotPassword(ctx context.Context, params model.ForgotPasswordInput) (*model.Response, error) {
	var res *model.Response

	if params.NewPassword != params.ConfirmPassword {
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

	password, _ := utils.HashPassword(params.NewPassword)
	user.Password = password

	// delete from verification table
	db.Mgr.DeleteToken(claim.Email)
	db.Mgr.UpdateUser(user)

	res = &model.Response{
		Message: `Password updated successfully.`,
	}

	return res, nil
}
