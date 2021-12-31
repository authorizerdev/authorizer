package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
)

func Session(ctx context.Context, roles []string) (*model.AuthResponse, error) {
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

	user, err := db.Mgr.GetUserByEmail(email)
	if err != nil {
		return res, err
	}

	userIdStr := fmt.Sprintf("%v", user.ID)

	sessionToken := session.GetToken(userIdStr, token)

	if sessionToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	expiresTimeObj := time.Unix(expiresAt, 0)
	currentTimeObj := time.Now()

	claimRoleInterface := claim[constants.EnvData.JWT_ROLE_CLAIM].([]interface{})
	claimRoles := make([]string, len(claimRoleInterface))
	for i, v := range claimRoleInterface {
		claimRoles[i] = v.(string)
	}

	if len(roles) > 0 {
		for _, v := range roles {
			if !utils.StringSliceContains(claimRoles, v) {
				return res, fmt.Errorf(`unauthorized`)
			}
		}
	}

	if accessTokenErr != nil || expiresTimeObj.Sub(currentTimeObj).Minutes() <= 5 {
		// if access token has expired and refresh/session token is valid
		// generate new accessToken
		currentRefreshToken := session.GetToken(userIdStr, token)
		session.DeleteVerificationRequest(userIdStr, token)
		token, expiresAt, _ = utils.CreateAuthToken(user, enum.AccessToken, claimRoles)
		session.SetToken(userIdStr, token, currentRefreshToken)
		utils.CreateSession(user.ID, gc)
	}

	utils.SetCookie(gc, token)
	res = &model.AuthResponse{
		Message:     `Token verified`,
		AccessToken: &token,
		ExpiresAt:   &expiresAt,
		User:        utils.GetResponseUserData(user),
	}
	return res, nil
}
