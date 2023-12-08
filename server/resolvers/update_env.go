package resolvers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// check if login methods have been disabled
// remove the session tokens for those methods
func clearSessionIfRequired(currentData, updatedData map[string]interface{}) {
	isCurrentBasicAuthEnabled := !currentData[constants.EnvKeyDisableBasicAuthentication].(bool)
	isCurrentMobileBasicAuthEnabled := !currentData[constants.EnvKeyDisableMobileBasicAuthentication].(bool)
	isCurrentMagicLinkLoginEnabled := !currentData[constants.EnvKeyDisableMagicLinkLogin].(bool)
	isCurrentAppleLoginEnabled := currentData[constants.EnvKeyAppleClientID] != nil && currentData[constants.EnvKeyAppleClientSecret] != nil && currentData[constants.EnvKeyAppleClientID].(string) != "" && currentData[constants.EnvKeyAppleClientSecret].(string) != ""
	isCurrentFacebookLoginEnabled := currentData[constants.EnvKeyFacebookClientID] != nil && currentData[constants.EnvKeyFacebookClientSecret] != nil && currentData[constants.EnvKeyFacebookClientID].(string) != "" && currentData[constants.EnvKeyFacebookClientSecret].(string) != ""
	isCurrentGoogleLoginEnabled := currentData[constants.EnvKeyGoogleClientID] != nil && currentData[constants.EnvKeyGoogleClientSecret] != nil && currentData[constants.EnvKeyGoogleClientID].(string) != "" && currentData[constants.EnvKeyGoogleClientSecret].(string) != ""
	isCurrentGithubLoginEnabled := currentData[constants.EnvKeyGithubClientID] != nil && currentData[constants.EnvKeyGithubClientSecret] != nil && currentData[constants.EnvKeyGithubClientID].(string) != "" && currentData[constants.EnvKeyGithubClientSecret].(string) != ""
	isCurrentLinkedInLoginEnabled := currentData[constants.EnvKeyLinkedInClientID] != nil && currentData[constants.EnvKeyLinkedInClientSecret] != nil && currentData[constants.EnvKeyLinkedInClientID].(string) != "" && currentData[constants.EnvKeyLinkedInClientSecret].(string) != ""
	isCurrentTwitterLoginEnabled := currentData[constants.EnvKeyTwitterClientID] != nil && currentData[constants.EnvKeyTwitterClientSecret] != nil && currentData[constants.EnvKeyTwitterClientID].(string) != "" && currentData[constants.EnvKeyTwitterClientSecret].(string) != ""
	isCurrentMicrosoftLoginEnabled := currentData[constants.EnvKeyMicrosoftClientID] != nil && currentData[constants.EnvKeyMicrosoftClientSecret] != nil && currentData[constants.EnvKeyMicrosoftClientID].(string) != "" && currentData[constants.EnvKeyMicrosoftClientSecret].(string) != ""
	isCurrentTwitchLoginEnabled := currentData[constants.EnvKeyTwitchClientID] != nil && currentData[constants.EnvKeyTwitchClientSecret] != nil && currentData[constants.EnvKeyTwitchClientID].(string) != "" && currentData[constants.EnvKeyTwitchClientSecret].(string) != ""

	isUpdatedBasicAuthEnabled := !updatedData[constants.EnvKeyDisableBasicAuthentication].(bool)
	isUpdatedMobileBasicAuthEnabled := !updatedData[constants.EnvKeyDisableMobileBasicAuthentication].(bool)
	isUpdatedMagicLinkLoginEnabled := !updatedData[constants.EnvKeyDisableMagicLinkLogin].(bool)
	isUpdatedAppleLoginEnabled := updatedData[constants.EnvKeyAppleClientID] != nil && updatedData[constants.EnvKeyAppleClientSecret] != nil && updatedData[constants.EnvKeyAppleClientID].(string) != "" && updatedData[constants.EnvKeyAppleClientSecret].(string) != ""
	isUpdatedFacebookLoginEnabled := updatedData[constants.EnvKeyFacebookClientID] != nil && updatedData[constants.EnvKeyFacebookClientSecret] != nil && updatedData[constants.EnvKeyFacebookClientID].(string) != "" && updatedData[constants.EnvKeyFacebookClientSecret].(string) != ""
	isUpdatedGoogleLoginEnabled := updatedData[constants.EnvKeyGoogleClientID] != nil && updatedData[constants.EnvKeyGoogleClientSecret] != nil && updatedData[constants.EnvKeyGoogleClientID].(string) != "" && updatedData[constants.EnvKeyGoogleClientSecret].(string) != ""
	isUpdatedGithubLoginEnabled := updatedData[constants.EnvKeyGithubClientID] != nil && updatedData[constants.EnvKeyGithubClientSecret] != nil && updatedData[constants.EnvKeyGithubClientID].(string) != "" && updatedData[constants.EnvKeyGithubClientSecret].(string) != ""
	isUpdatedLinkedInLoginEnabled := updatedData[constants.EnvKeyLinkedInClientID] != nil && updatedData[constants.EnvKeyLinkedInClientSecret] != nil && updatedData[constants.EnvKeyLinkedInClientID].(string) != "" && updatedData[constants.EnvKeyLinkedInClientSecret].(string) != ""
	isUpdatedTwitterLoginEnabled := updatedData[constants.EnvKeyTwitterClientID] != nil && updatedData[constants.EnvKeyTwitterClientSecret] != nil && updatedData[constants.EnvKeyTwitterClientID].(string) != "" && updatedData[constants.EnvKeyTwitterClientSecret].(string) != ""
	isUpdatedMicrosoftLoginEnabled := updatedData[constants.EnvKeyMicrosoftClientID] != nil && updatedData[constants.EnvKeyMicrosoftClientSecret] != nil && updatedData[constants.EnvKeyMicrosoftClientID].(string) != "" && updatedData[constants.EnvKeyMicrosoftClientSecret].(string) != ""
	isUpdatedTwitchLoginEnabled := updatedData[constants.EnvKeyTwitchClientID] != nil && updatedData[constants.EnvKeyTwitchClientSecret] != nil && updatedData[constants.EnvKeyTwitchClientID].(string) != "" && updatedData[constants.EnvKeyTwitchClientSecret].(string) != ""

	if isCurrentBasicAuthEnabled && !isUpdatedBasicAuthEnabled {
		memorystore.Provider.DeleteSessionForNamespace(constants.AuthRecipeMethodBasicAuth)
	}

	if isCurrentMobileBasicAuthEnabled && !isUpdatedMobileBasicAuthEnabled {
		memorystore.Provider.DeleteSessionForNamespace(constants.AuthRecipeMethodMobileBasicAuth)
	}

	if isCurrentMagicLinkLoginEnabled && !isUpdatedMagicLinkLoginEnabled {
		memorystore.Provider.DeleteSessionForNamespace(constants.AuthRecipeMethodMagicLinkLogin)
	}

	if isCurrentAppleLoginEnabled && !isUpdatedAppleLoginEnabled {
		memorystore.Provider.DeleteSessionForNamespace(constants.AuthRecipeMethodApple)
	}

	if isCurrentFacebookLoginEnabled && !isUpdatedFacebookLoginEnabled {
		memorystore.Provider.DeleteSessionForNamespace(constants.AuthRecipeMethodFacebook)
	}

	if isCurrentGoogleLoginEnabled && !isUpdatedGoogleLoginEnabled {
		memorystore.Provider.DeleteSessionForNamespace(constants.AuthRecipeMethodGoogle)
	}

	if isCurrentGithubLoginEnabled && !isUpdatedGithubLoginEnabled {
		memorystore.Provider.DeleteSessionForNamespace(constants.AuthRecipeMethodGithub)
	}

	if isCurrentLinkedInLoginEnabled && !isUpdatedLinkedInLoginEnabled {
		memorystore.Provider.DeleteSessionForNamespace(constants.AuthRecipeMethodLinkedIn)
	}

	if isCurrentTwitterLoginEnabled && !isUpdatedTwitterLoginEnabled {
		memorystore.Provider.DeleteSessionForNamespace(constants.AuthRecipeMethodTwitter)
	}

	if isCurrentMicrosoftLoginEnabled && !isUpdatedMicrosoftLoginEnabled {
		memorystore.Provider.DeleteSessionForNamespace(constants.AuthRecipeMethodMicrosoft)
	}

	if isCurrentTwitchLoginEnabled && !isUpdatedTwitchLoginEnabled {
		memorystore.Provider.DeleteSessionForNamespace(constants.AuthRecipeMethodTwitch)
	}
}

// updateRoles will update DB for user roles, if a role is deleted by admin
// then this function will those roles from user roles if exists
func updateRoles(ctx context.Context, deletedRoles []string) error {
	data, err := db.Provider.ListUsers(ctx, &model.Pagination{
		Limit:  1,
		Offset: 1,
	})
	if err != nil {
		return err
	}

	allData, err := db.Provider.ListUsers(ctx, &model.Pagination{
		Limit: data.Pagination.Total,
	})
	if err != nil {
		return err
	}

	for i := range allData.Users {
		roles := utils.DeleteFromArray(allData.Users[i].Roles, deletedRoles)
		if len(allData.Users[i].Roles) != len(roles) {
			updatedValues := map[string]interface{}{
				"roles":      strings.Join(roles, ","),
				"updated_at": time.Now().Unix(),
			}
			id := []string{allData.Users[i].ID}
			err = db.Provider.UpdateUsers(ctx, updatedValues, id)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateEnvResolver is a resolver for update config mutation
// This is admin only mutation
func UpdateEnvResolver(ctx context.Context, params model.UpdateEnvInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return res, fmt.Errorf("unauthorized")
	}

	currentData, err := memorystore.Provider.GetEnvStore()
	if err != nil {
		log.Debug("Failed to get env store: ", err)
		return res, err
	}

	// clone currentData in new var
	// that will be updated based on the req
	updatedData := make(map[string]interface{})
	for key, val := range currentData {
		updatedData[key] = val
	}

	isJWTUpdated := false
	algo := updatedData[constants.EnvKeyJwtType].(string)
	if params.JwtType != nil {
		algo = *params.JwtType
		if !crypto.IsHMACA(algo) && !crypto.IsECDSA(algo) && !crypto.IsRSA(algo) {
			log.Debug("Invalid JWT type: ", algo)
			return res, fmt.Errorf("invalid jwt type")
		}

		updatedData[constants.EnvKeyJwtType] = algo
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
				log.Debug("JWT private key and public key are required for RSA: ", *params.JwtPrivateKey, *params.JwtPublicKey)
				return res, fmt.Errorf("jwt private and public key is required for RSA (PKCS1) / ECDSA algorithm")
			}

			// reset the jwt secret
			params.JwtSecret = &defaultSecret
			_, err = crypto.ParseRsaPrivateKeyFromPemStr(*params.JwtPrivateKey)
			if err != nil {
				log.Debug("Invalid JWT private key: ", err)
				return res, err
			}

			_, err := crypto.ParseRsaPublicKeyFromPemStr(*params.JwtPublicKey)
			if err != nil {
				log.Debug("Invalid JWT public key: ", err)
				return res, err
			}
		}

		if crypto.IsECDSA(algo) {
			if params.JwtPrivateKey == nil || params.JwtPublicKey == nil {
				log.Debug("JWT private key and public key are required for ECDSA: ", *params.JwtPrivateKey, *params.JwtPublicKey)
				return res, fmt.Errorf("jwt private and public key is required for RSA (PKCS1) / ECDSA algorithm")
			}

			// reset the jwt secret
			params.JwtSecret = &defaultSecret
			_, err = crypto.ParseEcdsaPrivateKeyFromPemStr(*params.JwtPrivateKey)
			if err != nil {
				log.Debug("Invalid JWT private key: ", err)
				return res, err
			}

			_, err := crypto.ParseEcdsaPublicKeyFromPemStr(*params.JwtPublicKey)
			if err != nil {
				log.Debug("Invalid JWT public key: ", err)
				return res, err
			}
		}

	}

	var data map[string]interface{}
	byteData, err := json.Marshal(params)
	if err != nil {
		log.Debug("Failed to marshal update env input: ", err)
		return res, fmt.Errorf("error marshalling params: %t", err)
	}

	err = json.Unmarshal(byteData, &data)
	if err != nil {
		log.Debug("Failed to unmarshal update env input: ", err)
		return res, fmt.Errorf("error un-marshalling params: %t", err)
	}

	// in case of admin secret change update the cookie with new hash
	if params.AdminSecret != nil {
		if params.OldAdminSecret == nil {
			log.Debug("Old admin secret is required for admin secret update")
			return res, errors.New("admin secret and old admin secret are required for secret change")
		}
		oldAdminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		if err != nil {
			log.Debug("Failed to get old admin secret: ", err)
			return res, err
		}
		if *params.OldAdminSecret != oldAdminSecret {
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
				updatedData[key] = value.(string)
			}

			if fieldType == "bool" {
				updatedData[key] = value.(bool)
			}
			if fieldType == "[]interface {}" {
				stringArr := utils.ConvertInterfaceToStringSlice(value)
				updatedData[key] = strings.Join(stringArr, ",")
			}
		}
	}

	// handle derivative cases like disabling email verification & magic login
	// in case SMTP is off but env is set to true
	if updatedData[constants.EnvKeySmtpHost] == "" || updatedData[constants.EnvKeySmtpUsername] == "" || updatedData[constants.EnvKeySmtpPassword] == "" || updatedData[constants.EnvKeySenderEmail] == "" && updatedData[constants.EnvKeySmtpPort] == "" {
		updatedData[constants.EnvKeyIsEmailServiceEnabled] = false
		if !updatedData[constants.EnvKeyDisableEmailVerification].(bool) {
			updatedData[constants.EnvKeyDisableEmailVerification] = true
		}
		if !updatedData[constants.EnvKeyDisableMailOTPLogin].(bool) {
			updatedData[constants.EnvKeyDisableMailOTPLogin] = true
		}
		if !updatedData[constants.EnvKeyDisableMagicLinkLogin].(bool) {
			updatedData[constants.EnvKeyDisableMailOTPLogin] = true
		}
	}

	if updatedData[constants.EnvKeySmtpHost] != "" || updatedData[constants.EnvKeySmtpUsername] != "" || updatedData[constants.EnvKeySmtpPassword] != "" || updatedData[constants.EnvKeySenderEmail] != "" && updatedData[constants.EnvKeySmtpPort] != "" {
		updatedData[constants.EnvKeyIsEmailServiceEnabled] = true
	}

	if updatedData[constants.EnvKeyTwilioAPIKey] == "" || updatedData[constants.EnvKeyTwilioAPISecret] == "" || updatedData[constants.EnvKeyTwilioAccountSID] == "" || updatedData[constants.EnvKeyTwilioSender] == "" {
		updatedData[constants.EnvKeyIsSMSServiceEnabled] = false
		if !updatedData[constants.EnvKeyIsSMSServiceEnabled].(bool) {
			updatedData[constants.EnvKeyDisablePhoneVerification] = true
		}
	}

	if updatedData[constants.EnvKeyDisableMultiFactorAuthentication].(bool) && updatedData[constants.EnvKeyIsEmailServiceEnabled].(bool) {
		updatedData[constants.EnvKeyDisableMailOTPLogin] = true
	}

	if !currentData[constants.EnvKeyEnforceMultiFactorAuthentication].(bool) && updatedData[constants.EnvKeyEnforceMultiFactorAuthentication].(bool) && !updatedData[constants.EnvKeyDisableMultiFactorAuthentication].(bool) {
		go db.Provider.UpdateUsers(ctx, map[string]interface{}{
			"is_multi_factor_auth_enabled": true,
		}, nil)
	}

	previousRoles := strings.Split(currentData[constants.EnvKeyRoles].(string), ",")
	updatedRoles := strings.Split(updatedData[constants.EnvKeyRoles].(string), ",")
	updatedDefaultRoles := strings.Split(updatedData[constants.EnvKeyDefaultRoles].(string), ",")
	updatedProtectedRoles := strings.Split(updatedData[constants.EnvKeyProtectedRoles].(string), ",")

	// check the roles change
	if len(updatedRoles) > 0 {
		if len(updatedDefaultRoles) > 0 {
			// should be subset of roles
			for _, role := range updatedDefaultRoles {
				if !utils.StringSliceContains(updatedRoles, role) {
					log.Debug("Default roles should be subset of roles")
					return res, fmt.Errorf("default role %s is not in roles", role)
				}
			}
		}
	}

	if len(updatedProtectedRoles) > 0 {
		for _, role := range updatedProtectedRoles {
			if utils.StringSliceContains(updatedRoles, role) || utils.StringSliceContains(updatedDefaultRoles, role) {
				log.Debug("Protected roles should not be in roles or default roles")
				return res, fmt.Errorf("protected role %s found roles or default roles", role)
			}
		}
	}

	deletedRoles := utils.FindDeletedValues(previousRoles, updatedRoles)
	if len(deletedRoles) > 0 {
		go updateRoles(ctx, deletedRoles)
	}

	// Update local store
	memorystore.Provider.UpdateEnvStore(updatedData)
	jwk, err := crypto.GenerateJWKBasedOnEnv()
	if err != nil {
		log.Debug("Failed to generate JWK: ", err)
		return res, err
	}
	// updating jwk
	err = memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJWK, jwk)
	if err != nil {
		log.Debug("Failed to update JWK: ", err)
		return res, err
	}

	err = oauth.InitOAuth()
	if err != nil {
		return res, err
	}

	// Fetch the current db store and update it
	env, err := db.Provider.GetEnv(ctx)
	if err != nil {
		log.Debug("Failed to get env: ", err)
		return res, err
	}

	if params.AdminSecret != nil {
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		if err != nil {
			log.Debug("Failed to get admin secret: ", err)
			return res, err
		}
		hashedKey, err := crypto.EncryptPassword(adminSecret)
		if err != nil {
			log.Debug("Failed to encrypt admin secret: ", err)
			return res, err
		}
		cookie.SetAdminCookie(gc, hashedKey)
	}

	encryptedConfig, err := crypto.EncryptEnvData(updatedData)
	if err != nil {
		log.Debug("Failed to encrypt env data: ", err)
		return res, err
	}

	env.EnvData = encryptedConfig
	_, err = db.Provider.UpdateEnv(ctx, env)
	if err != nil {
		log.Debug("Failed to update env: ", err)
		return res, err
	}

	go clearSessionIfRequired(currentData, updatedData)

	res = &model.Response{
		Message: "configurations updated successfully",
	}
	return res, nil
}
