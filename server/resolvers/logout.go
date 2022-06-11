package resolvers

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// LogoutResolver is a resolver for logout mutation
func LogoutResolver(ctx context.Context) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	// get fingerprint hash
	fingerprintHash, err := cookie.GetSession(gc)
	if err != nil {
		log.Debug("Failed to get fingerprint hash: ", err)
		return nil, err
	}

	decryptedFingerPrint, err := crypto.DecryptAES(fingerprintHash)
	if err != nil {
		log.Debug("Failed to decrypt fingerprint hash: ", err)
		return nil, err
	}

	var sessionData token.SessionData
	err = json.Unmarshal([]byte(decryptedFingerPrint), &sessionData)
	if err != nil {
		return nil, err
	}

	memorystore.Provider.DeleteUserSession(sessionData.Subject, fingerprintHash)
	cookie.DeleteSession(gc)

	res := &model.Response{
		Message: "Logged out successfully",
	}

	return res, nil
}
