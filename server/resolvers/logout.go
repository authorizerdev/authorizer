package resolvers

import (
	"context"

	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
)

func Logout(ctx context.Context) (*model.Response, error) {
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

	session.DeleteToken(claim.ID)
	res = &model.Response{
		Message: "Logged out successfully",
	}

	utils.DeleteCookie(gc)
	return res, nil
}
