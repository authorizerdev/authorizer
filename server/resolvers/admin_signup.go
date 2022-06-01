package resolvers

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/utils"
)

// AdminSignupResolver is a resolver for admin signup mutation
func AdminSignupResolver(ctx context.Context, params model.AdminSignupInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	if strings.TrimSpace(params.AdminSecret) == "" {
		log.Debug("Admin secret is empty")
		err = fmt.Errorf("please select secure admin secret")
		return res, err
	}

	if len(params.AdminSecret) < 6 {
		log.Debug("Admin secret is too short")
		err = fmt.Errorf("admin secret must be at least 6 characters")
		return res, err
	}

	adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
	if err != nil {
		log.Debug("Error getting admin secret: ", err)
		adminSecret = ""
	}

	if adminSecret != "" {
		log.Debug("Admin secret is already set")
		err = fmt.Errorf("admin sign up already completed")
		return res, err
	}

	memorystore.Provider.UpdateEnvVariable(constants.EnvKeyAdminSecret, params.AdminSecret)
	// consvert EnvData to JSON
	storeData, err := memorystore.Provider.GetEnvStore()
	if err != nil {
		log.Debug("Error getting env store: ", err)
		return res, err
	}

	env, err := db.Provider.GetEnv()
	if err != nil {
		log.Debug("Failed to get env: ", err)
		return res, err
	}

	envData, err := crypto.EncryptEnvData(storeData)
	if err != nil {
		log.Debug("Failed to encrypt envstore: ", err)
		return res, err
	}

	env.EnvData = envData
	if _, err := db.Provider.UpdateEnv(env); err != nil {
		log.Debug("Failed to update env: ", err)
		return res, err
	}

	hashedKey, err := crypto.EncryptPassword(params.AdminSecret)
	if err != nil {
		log.Debug("Failed to encrypt admin session key: ", err)
		return res, err
	}
	cookie.SetAdminCookie(gc, hashedKey)

	res = &model.Response{
		Message: "admin signed up successfully",
	}
	return res, nil
}
