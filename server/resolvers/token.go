package resolvers

import (
	"context"
	"fmt"

	"github.com/yauthdev/yauth/server/db"
	"github.com/yauthdev/yauth/server/enum"
	"github.com/yauthdev/yauth/server/graph/model"
	"github.com/yauthdev/yauth/server/session"
	"github.com/yauthdev/yauth/server/utils"
)

func Token(ctx context.Context) (*model.LoginResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.LoginResponse
	if err != nil {
		return res, err
	}
	token, err := utils.GetAuthToken(gc)
	if err != nil {
		return res, err
	}

	claim, accessTokenErr := utils.VerifyAuthToken(token)
	expiresAt := claim.ExpiresAt

	user, err := db.Mgr.GetUserByEmail(claim.Email)
	if err != nil {
		return res, err
	}

	userIdStr := fmt.Sprintf("%d", user.ID)

	sessionToken := session.GetToken(userIdStr)

	if sessionToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}
	// TODO check if session token has expired

	if accessTokenErr != nil {
		// if access token has expired and refresh/session token is valid
		// generate new accessToken
		token, expiresAt, _ = utils.CreateAuthToken(utils.UserAuthInfo{
			ID:    userIdStr,
			Email: user.Email,
		}, enum.AccessToken)
	}
	utils.SetCookie(gc, token)
	res = &model.LoginResponse{
		Message:              `Email verified successfully.`,
		AccessToken:          &token,
		AccessTokenExpiresAt: &expiresAt,
		User: &model.User{
			ID:        userIdStr,
			Email:     user.Email,
			Image:     &user.Image,
			FirstName: &user.FirstName,
			LastName:  &user.LastName,
			CreatedAt: &user.CreatedAt,
			UpdatedAt: &user.UpdatedAt,
		},
	}
	return res, nil
}
