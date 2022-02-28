package resolvers

import (
	"context"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// LogoutResolver is a resolver for logout mutation
func LogoutResolver(ctx context.Context) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Response
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

	decryptedFingerPrint, err := crypto.DecryptAES([]byte(fingerprintHash))
	if err != nil {
		return res, err
	}

	fingerPrint := string(decryptedFingerPrint)

	// verify refresh token and fingerprint
	claims, err := token.ParseJWTToken(refreshToken)
	if err != nil {
		return res, err
	}

	userID := claims["id"].(string)
	sessionstore.DeleteUserSession(userID, fingerPrint)
	cookie.DeleteCookie(gc)

	res = &model.Response{
		Message: "Logged out successfully",
	}

	return res, nil
}
