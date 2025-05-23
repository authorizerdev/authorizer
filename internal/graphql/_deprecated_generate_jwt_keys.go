package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// GenerateJWTKeysResolver mutation to generate new jwt keys
// Permissions: authorizer:admin
// Deprecated for
func (g *graphqlProvider) GenerateJWTKeysResolver(ctx context.Context, params model.GenerateJWTKeysInput) (*model.GenerateJWTKeysResponse, error) {
	// gc, err := utils.GinContextFromContext(ctx)
	// if err != nil {
	// 	log.Debug("Failed to get GinContext: ", err)
	// 	return nil, err
	// }

	// if !token.IsSuperAdmin(gc) {
	// 	log.Debug("Not logged in as super admin")
	// 	return nil, fmt.Errorf("unauthorized")
	// }

	// clientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID)
	// if err != nil {
	// 	log.Debug("Error getting client id: ", err)
	// 	return nil, err
	// }
	// if crypto.IsHMACA(params.Type) {
	// 	secret, _, err := crypto.NewHMACKey(params.Type, clientID)
	// 	if err != nil {
	// 		log.Debug("Failed to generate new HMAC key: ", err)
	// 		return nil, err
	// 	}
	// 	return &model.GenerateJWTKeysResponse{
	// 		Secret: &secret,
	// 	}, nil
	// }

	// if crypto.IsRSA(params.Type) {
	// 	_, privateKey, publicKey, _, err := crypto.NewRSAKey(params.Type, clientID)
	// 	if err != nil {
	// 		log.Debug("Failed to generate new RSA key: ", err)
	// 		return nil, err
	// 	}
	// 	return &model.GenerateJWTKeysResponse{
	// 		PrivateKey: &privateKey,
	// 		PublicKey:  &publicKey,
	// 	}, nil
	// }

	// if crypto.IsECDSA(params.Type) {
	// 	_, privateKey, publicKey, _, err := crypto.NewECDSAKey(params.Type, clientID)
	// 	if err != nil {
	// 		log.Debug("Failed to generate new ECDSA key: ", err)
	// 		return nil, err
	// 	}
	// 	return &model.GenerateJWTKeysResponse{
	// 		PrivateKey: &privateKey,
	// 		PublicKey:  &publicKey,
	// 	}, nil
	// }

	// log.Debug("Invalid algorithm: ", params.Type)
	return nil, fmt.Errorf("deprecated")
}
