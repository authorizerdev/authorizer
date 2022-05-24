package resolvers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// UpdateEnvResolver is a resolver for update config mutation
// This is admin only mutation
func UpdateEnvResolver(ctx context.Context, params model.UpdateEnvInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext", err)
		return res, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin.")
		return res, fmt.Errorf("unauthorized")
	}

	updatedData := envstore.EnvStoreObj.GetEnvStoreClone()

	isJWTUpdated := false
	algo := updatedData.StringEnv[constants.EnvKeyJwtType]
	if params.JwtType != nil {
		algo = *params.JwtType
		if !crypto.IsHMACA(algo) && !crypto.IsECDSA(algo) && !crypto.IsRSA(algo) {
			log.Debug("Invalid JWT type", algo)
			return res, fmt.Errorf("invalid jwt type")
		}

		updatedData.StringEnv[constants.EnvKeyJwtType] = algo
		isJWTUpdated = true
	}

	if params.JwtSecret != nil || params.JwtPublicKey != nil || params.JwtPrivateKey != nil {
		isJWTUpdated = true
	}

	if isJWTUpdated {
		// use to reset when type is changed from rsa, edsa -> hmac or vice a versa
		defaultSecret := ""
		defaultPublicKey := ""
		defaultPrivateKey := ""
		// check if jwt secret is provided
		if crypto.IsHMACA(algo) {
			if params.JwtSecret == nil {
				log.Debug("JWT secret is required for HMAC")
				return res, fmt.Errorf("jwt secret is required for HMAC algorithm")
			}

			// reset public key and private key
			params.JwtPrivateKey = &defaultPrivateKey
			params.JwtPublicKey = &defaultPublicKey
		}

		if crypto.IsRSA(algo) {
			if params.JwtPrivateKey == nil || params.JwtPublicKey == nil {
				log.Debug("JWT private key and public key are required for RSA", params.JwtPrivateKey, params.JwtPublicKey)
				return res, fmt.Errorf("jwt private and public key is required for RSA (PKCS1) / ECDSA algorithm")
			}

			// reset the jwt secret
			params.JwtSecret = &defaultSecret
			_, err = crypto.ParseRsaPrivateKeyFromPemStr(*params.JwtPrivateKey)
			if err != nil {
				log.Debug("Invalid JWT private key", err)
				return res, err
			}

			_, err := crypto.ParseRsaPublicKeyFromPemStr(*params.JwtPublicKey)
			if err != nil {
				log.Debug("Invalid JWT public key", err)
				return res, err
			}
		}

		if crypto.IsECDSA(algo) {
			if params.JwtPrivateKey == nil || params.JwtPublicKey == nil {
				log.Debug("JWT private key and public key are required for ECDSA", params.JwtPrivateKey, params.JwtPublicKey)
				return res, fmt.Errorf("jwt private and public key is required for RSA (PKCS1) / ECDSA algorithm")
			}

			// reset the jwt secret
			params.JwtSecret = &defaultSecret
			_, err = crypto.ParseEcdsaPrivateKeyFromPemStr(*params.JwtPrivateKey)
			if err != nil {
				log.Debug("Invalid JWT private key", err)
				return res, err
			}

			_, err := crypto.ParseEcdsaPublicKeyFromPemStr(*params.JwtPublicKey)
			if err != nil {
				log.Debug("Invalid JWT public key", err)
				return res, err
			}
		}

	}

	var data map[string]interface{}
	byteData, err := json.Marshal(params)
	if err != nil {
		log.Debug("Failed to marshal update env input", err)
		return res, fmt.Errorf("error marshalling params: %t", err)
	}

	err = json.Unmarshal(byteData, &data)
	if err != nil {
		log.Debug("Failed to unmarshal update env input", err)
		return res, fmt.Errorf("error un-marshalling params: %t", err)
	}

	// in case of admin secret change update the cookie with new hash
	if params.AdminSecret != nil {
		if params.OldAdminSecret == nil {
			log.Debug("Old admin secret is required for admin secret update")
			return res, errors.New("admin secret and old admin secret are required for secret change")
		}

		if *params.OldAdminSecret != envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret) {
			log.Debug("Old admin secret is invalid")
			return res, errors.New("old admin secret is not correct")
		}

		if len(*params.AdminSecret) < 6 {
			log.Debug("Admin secret is too short")
			err = fmt.Errorf("admin secret must be at least 6 characters")
			return res, err
		}

	}

	for key, value := range data {
		if value != nil {
			fieldType := reflect.TypeOf(value).String()

			if fieldType == "string" {
				updatedData.StringEnv[key] = value.(string)
			}

			if fieldType == "bool" {
				updatedData.BoolEnv[key] = value.(bool)
			}
			if fieldType == "[]interface {}" {
				stringArr := []string{}
				for _, v := range value.([]interface{}) {
					stringArr = append(stringArr, v.(string))
				}
				updatedData.SliceEnv[key] = stringArr
			}
		}
	}

	// handle derivative cases like disabling email verification & magic login
	// in case SMTP is off but env is set to true
	if updatedData.StringEnv[constants.EnvKeySmtpHost] == "" || updatedData.StringEnv[constants.EnvKeySmtpUsername] == "" || updatedData.StringEnv[constants.EnvKeySmtpPassword] == "" || updatedData.StringEnv[constants.EnvKeySenderEmail] == "" && updatedData.StringEnv[constants.EnvKeySmtpPort] == "" {
		if !updatedData.BoolEnv[constants.EnvKeyDisableEmailVerification] {
			updatedData.BoolEnv[constants.EnvKeyDisableEmailVerification] = true
		}

		if !updatedData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] {
			updatedData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] = true
		}
	}

	// check the roles change
	if len(params.Roles) > 0 {
		if len(params.DefaultRoles) > 0 {
			// should be subset of roles
			for _, role := range params.DefaultRoles {
				if !utils.StringSliceContains(params.Roles, role) {
					log.Debug("Default roles should be subset of roles")
					return res, fmt.Errorf("default role %s is not in roles", role)
				}
			}
		}
	}

	if len(params.ProtectedRoles) > 0 {
		for _, role := range params.ProtectedRoles {
			if utils.StringSliceContains(params.Roles, role) || utils.StringSliceContains(params.DefaultRoles, role) {
				log.Debug("Protected roles should not be in roles or default roles")
				return res, fmt.Errorf("protected role %s found roles or default roles", role)
			}
		}
	}

	// Update local store
	envstore.EnvStoreObj.UpdateEnvStore(updatedData)
	jwk, err := crypto.GenerateJWKBasedOnEnv()
	if err != nil {
		log.Debug("Failed to generate JWK", err)
		return res, err
	}
	// updating jwk
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJWK, jwk)
	err = sessionstore.InitSession()
	if err != nil {
		log.Debug("Failed to init session store", err)
		return res, err
	}
	err = oauth.InitOAuth()
	if err != nil {
		return res, err
	}

	// Fetch the current db store and update it
	env, err := db.Provider.GetEnv()
	if err != nil {
		log.Debug("Failed to get env", err)
		return res, err
	}

	if params.AdminSecret != nil {
		hashedKey, err := crypto.EncryptPassword(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
		if err != nil {
			log.Debug("Failed to encrypt admin secret", err)
			return res, err
		}
		cookie.SetAdminCookie(gc, hashedKey)
	}

	encryptedConfig, err := crypto.EncryptEnvData(updatedData)
	if err != nil {
		log.Debug("Failed to encrypt env data", err)
		return res, err
	}

	env.EnvData = encryptedConfig
	_, err = db.Provider.UpdateEnv(env)
	if err != nil {
		log.Debug("Failed to update env", err)
		return res, err
	}

	res = &model.Response{
		Message: "configurations updated successfully",
	}
	return res, nil
}
