package resolvers

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/utils"
)

// LogoutResolver is a resolver for logout mutation
func LogoutResolver(ctx context.Context) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	// get fingerprint hash
	fingerprintHash, err := cookie.GetSession(gc)
	if err != nil {
		log.Debug("Failed to get fingerprint hash: ", err)
		return res, err
	}

	decryptedFingerPrint, err := crypto.DecryptAES(fingerprintHash)
	if err != nil {
		log.Debug("Failed to decrypt fingerprint hash: ", err)
		return res, err
	}

	fingerPrint := string(decryptedFingerPrint)

	memorystore.Provider.RemoveState(fingerPrint)
	cookie.DeleteSession(gc)

	res = &model.Response{
		Message: "Logged out successfully",
	}

	return res, nil
}
