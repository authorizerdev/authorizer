package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// AdminSignup is a resolver for admin signup
// Permissions: none,
// Deprecated and admin secret can only be set via cli args
func (g *graphqlProvider) AdminSignup(ctx context.Context, params *model.AdminSignupInput) (*model.Response, error) {
	// var res *model.Response

	// gc, err := utils.GinContextFromContext(ctx)
	// if err != nil {
	// 	log.Debug("Failed to get GinContext: ", err)
	// 	return res, err
	// }

	// if strings.TrimSpace(params.AdminSecret) == "" {
	// 	log.Debug("Admin secret is empty")
	// 	err = fmt.Errorf("please select secure admin secret")
	// 	return res, err
	// }

	// if len(params.AdminSecret) < 6 {
	// 	log.Debug("Admin secret is too short")
	// 	err = fmt.Errorf("admin secret must be at least 6 characters")
	// 	return res, err
	// }

	// adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
	// if err != nil {
	// 	log.Debug("Error getting admin secret: ", err)
	// 	adminSecret = ""
	// }

	// if adminSecret != "" {
	// 	log.Debug("Admin secret is already set")
	// 	err = fmt.Errorf("admin sign up already completed")
	// 	return res, err
	// }

	// memorystore.Provider.UpdateEnvVariable(constants.EnvKeyAdminSecret, params.AdminSecret)
	// // consvert EnvData to JSON
	// storeData, err := memorystore.Provider.GetEnvStore()
	// if err != nil {
	// 	log.Debug("Error getting env store: ", err)
	// 	return res, err
	// }

	// env, err := db.Provider.GetEnv(ctx)
	// if err != nil {
	// 	log.Debug("Failed to get env: ", err)
	// 	return res, err
	// }

	// envData, err := crypto.EncryptEnvData(storeData)
	// if err != nil {
	// 	log.Debug("Failed to encrypt envstore: ", err)
	// 	return res, err
	// }

	// env.EnvData = envData
	// if _, err := db.Provider.UpdateEnv(ctx, env); err != nil {
	// 	log.Debug("Failed to update env: ", err)
	// 	return res, err
	// }

	// hashedKey, err := crypto.EncryptPassword(params.AdminSecret)
	// if err != nil {
	// 	log.Debug("Failed to encrypt admin session key: ", err)
	// 	return res, err
	// }
	// cookie.SetAdminCookie(gc, hashedKey)

	// res = &model.Response{
	// 	Message: "admin signed up successfully",
	// }
	return nil, fmt.Errorf("deprecated. please configure admin secret via cli args")
}
