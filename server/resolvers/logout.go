package resolvers

import (
	"context"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/utils"
)

// LogoutResolver is a resolver for logout mutation
func LogoutResolver(ctx context.Context) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Response
	if err != nil {
		return res, err
	}

	// get fingerprint hash
	fingerprintHash, err := cookie.GetSession(gc)
	if err != nil {
		return res, err
	}

	decryptedFingerPrint, err := crypto.DecryptAES(fingerprintHash)
	if err != nil {
		return res, err
	}

	fingerPrint := string(decryptedFingerPrint)

	sessionstore.RemoveState(fingerPrint)
	cookie.DeleteSession(gc)

	res = &model.Response{
		Message: "Logged out successfully",
	}

	return res, nil
}
