package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	log "github.com/sirupsen/logrus"
)

// GenerateJWTKeysResolver mutation to generate new jwt keys
func GenerateJWTKeysResolver(ctx context.Context, params model.GenerateJWTKeysInput) (*model.GenerateJWTKeysResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin.")
		return nil, fmt.Errorf("unauthorized")
	}

	clientID := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyClientID)
	if crypto.IsHMACA(params.Type) {
		secret, _, err := crypto.NewHMACKey(params.Type, clientID)
		if err != nil {
			log.Debug("Failed to generate new HMAC key:", err)
			return nil, err
		}
		return &model.GenerateJWTKeysResponse{
			Secret: &secret,
		}, nil
	}

	if crypto.IsRSA(params.Type) {
		_, privateKey, publicKey, _, err := crypto.NewRSAKey(params.Type, clientID)
		if err != nil {
			log.Debug("Failed to generate new RSA key:", err)
			return nil, err
		}
		return &model.GenerateJWTKeysResponse{
			PrivateKey: &privateKey,
			PublicKey:  &publicKey,
		}, nil
	}

	if crypto.IsECDSA(params.Type) {
		_, privateKey, publicKey, _, err := crypto.NewECDSAKey(params.Type, clientID)
		if err != nil {
			log.Debug("Failed to generate new ECDSA key:", err)
			return nil, err
		}
		return &model.GenerateJWTKeysResponse{
			PrivateKey: &privateKey,
			PublicKey:  &publicKey,
		}, nil
	}

	log.Debug("Invalid algorithm:", params.Type)
	return nil, fmt.Errorf("invalid algorithm")
}
