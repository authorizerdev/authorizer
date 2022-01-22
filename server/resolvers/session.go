package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// SessionResolver is a resolver for session query
func SessionResolver(ctx context.Context, roles []string) (*model.AuthResponse, error) {
	var res *model.AuthResponse

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		return res, err
	}

	// get refresh token
	refreshToken, err := token.GetRefreshToken(gc)
	if err != nil {
		return res, err
	}

	// get fingerprint hash
	fingerprintHash, err := token.GetFingerPrint(gc)
	if err != nil {
		return res, err
	}

	decryptedFingerPrint, err := utils.DecryptAES([]byte(fingerprintHash))
	if err != nil {
		return res, err
	}

	fingerPrint := string(decryptedFingerPrint)

	// verify refresh token and fingerprint
	claims, err := token.VerifyJWTToken(refreshToken)
	if err != nil {
		return res, err
	}

	userID := claims["id"].(string)

	persistedRefresh := sessionstore.GetUserSession(userID, fingerPrint)
	if refreshToken != persistedRefresh {
		return res, fmt.Errorf(`unauthorized`)
	}

	user, err := db.Provider.GetUserByID(userID)
	if err != nil {
		return res, err
	}

	// refresh token has "roles" as claim
	claimRoleInterface := claims["roles"].([]interface{})
	claimRoles := []string{}
	for _, v := range claimRoleInterface {
		claimRoles = append(claimRoles, v.(string))
	}

	if len(roles) > 0 {
		for _, v := range roles {
			if !utils.StringSliceContains(claimRoles, v) {
				return res, fmt.Errorf(`unauthorized`)
			}
		}
	}

	// delete older session
	sessionstore.DeleteUserSession(userID, fingerPrint)

	authToken, err := token.CreateAuthToken(user, claimRoles)
	if err != nil {
		return res, err
	}
	sessionstore.SetUserSession(user.ID, authToken.FingerPrint, authToken.RefreshToken.Token)
	cookie.SetCookie(gc, authToken.AccessToken.Token, authToken.RefreshToken.Token, authToken.FingerPrintHash)

	res = &model.AuthResponse{
		Message:     `Session token refreshed`,
		AccessToken: &authToken.AccessToken.Token,
		ExpiresAt:   &authToken.AccessToken.ExpiresAt,
		User:        user.AsAPIUser(),
	}

	return res, nil
}
