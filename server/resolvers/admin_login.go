package resolvers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
)

func AdminLoginResolver(ctx context.Context, params model.AdminLoginInput) (*model.AdminLoginResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.AdminLoginResponse
	if err != nil {
		log.Println("=> error:", err)
		return res, err
	}
	if params.AdminSecret != constants.ADMIN_SECRET {
		return nil, fmt.Errorf(`invalid admin secret`)
	}

	refreshToken, _, _ := utils.CreateAdminAuthToken(enum.RefreshToken, gc)
	accessToken, expiresAt, _ := utils.CreateAdminAuthToken(enum.AccessToken, gc)

	currentTime := time.Now().Unix()
	tokenId := fmt.Sprintf("authorizer_admin_%d", currentTime)
	session.SetToken(tokenId, accessToken, refreshToken)
	utils.SetAdminCookie(gc, accessToken)

	res = &model.AdminLoginResponse{
		AccessToken: accessToken,
		ExpiresAt:   expiresAt,
		Message:     "admin logged in successfully",
	}
	return res, nil
}
