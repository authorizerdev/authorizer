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
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
)

func Token(ctx context.Context, role *string) (*model.AuthResponse, error) {
	var res *model.AuthResponse

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		return res, err
	}
	token, err := utils.GetAuthToken(gc)
	if err != nil {
		return res, err
	}

	claim, accessTokenErr := utils.VerifyAuthToken(token)
	expiresAt := claim["exp"].(int64)
	email := fmt.Sprintf("%v", claim["email"])

	claimRole := fmt.Sprintf("%v", claim[constants.JWT_ROLE_CLAIM])
	user, err := db.Mgr.GetUserByEmail(email)
	if err != nil {
		return res, err
	}

	if role != nil && role != &claimRole {
		return res, fmt.Errorf(`unauthorized. invalid role for a given token`)
	}

	userIdStr := fmt.Sprintf("%v", user.ID)

	sessionToken := session.GetToken(userIdStr)

	if sessionToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}
	// TODO check if refresh/session token has expired

	expiresTimeObj := time.Unix(expiresAt, 0)
	currentTimeObj := time.Now()
	if accessTokenErr != nil || expiresTimeObj.Sub(currentTimeObj).Minutes() <= 5 {
		// if access token has expired and refresh/session token is valid
		// generate new accessToken
		token, expiresAt, _ = utils.CreateAuthToken(user, enum.AccessToken, claimRole)
	}
	utils.SetCookie(gc, token)
	res = &model.AuthResponse{
		Message:              `Token verified`,
		AccessToken:          &token,
		AccessTokenExpiresAt: &expiresAt,
		User: &model.User{
			ID:        userIdStr,
			Email:     user.Email,
			Image:     &user.Image,
			FirstName: &user.FirstName,
			LastName:  &user.LastName,
			Roles:     strings.Split(user.Roles, ","),
			CreatedAt: &user.CreatedAt,
			UpdatedAt: &user.UpdatedAt,
		},
	}
	return res, nil
}
