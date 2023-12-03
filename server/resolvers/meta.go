package resolvers

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
)

// MetaResolver is a resolver for meta query
func MetaResolver(ctx context.Context) (*model.Meta, error) {
	clientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID)
	if err != nil {
		return nil, err
	}

	googleClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientID)
	if err != nil {
		log.Debug("Failed to get Google Client ID from environment variable", err)
		googleClientID = ""
	}

	googleClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientSecret)
	if err != nil {
		log.Debug("Failed to get Google Client Secret from environment variable", err)
		googleClientSecret = ""
	}

	facebookClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientID)
	if err != nil {
		log.Debug("Failed to get Facebook Client ID from environment variable", err)
		facebookClientID = ""
	}

	facebookClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientSecret)
	if err != nil {
		log.Debug("Failed to get Facebook Client Secret from environment variable", err)
		facebookClientSecret = ""
	}

	linkedClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyLinkedInClientID)
	if err != nil {
		log.Debug("Failed to get LinkedIn Client ID from environment variable", err)
		linkedClientID = ""
	}

	linkedInClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyLinkedInClientSecret)
	if err != nil {
		log.Debug("Failed to get LinkedIn Client Secret from environment variable", err)
		linkedInClientSecret = ""
	}

	appleClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAppleClientID)
	if err != nil {
		log.Debug("Failed to get Apple Client ID from environment variable", err)
		appleClientID = ""
	}

	appleClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAppleClientSecret)
	if err != nil {
		log.Debug("Failed to get Apple Client Secret from environment variable", err)
		appleClientSecret = ""
	}

	githubClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyGithubClientID)
	if err != nil {
		log.Debug("Failed to get Github Client ID from environment variable", err)
		githubClientID = ""
	}

	githubClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyGithubClientSecret)
	if err != nil {
		log.Debug("Failed to get Github Client Secret from environment variable", err)
		githubClientSecret = ""
	}

	twitterClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyTwitterClientID)
	if err != nil {
		log.Debug("Failed to get Twitter Client ID from environment variable", err)
		twitterClientID = ""
	}

	twitterClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyTwitterClientSecret)
	if err != nil {
		log.Debug("Failed to get Twitter Client Secret from environment variable", err)
		twitterClientSecret = ""
	}

	microsoftClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyMicrosoftClientID)
	if err != nil {
		log.Debug("Failed to get Microsoft Client ID from environment variable", err)
		microsoftClientID = ""
	}

	microsoftClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyMicrosoftClientSecret)
	if err != nil {
		log.Debug("Failed to get Microsoft Client Secret from environment variable", err)
		microsoftClientSecret = ""
	}

	isBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication)
	if err != nil {
		log.Debug("Failed to get Disable Basic Authentication from environment variable", err)
		isBasicAuthDisabled = true
	}
	isMobileBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMobileBasicAuthentication)
	if err != nil {
		log.Debug("Failed to get Disable Basic Authentication from environment variable", err)
		isMobileBasicAuthDisabled = true
	}
	isMobileVerificationDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisablePhoneVerification)
	if err != nil {
		log.Debug("Failed to get Disable Basic Authentication from environment variable", err)
		isMobileVerificationDisabled = true
	}

	isEmailVerificationDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableEmailVerification)
	if err != nil {
		log.Debug("Failed to get Disable Email Verification from environment variable", err)
		isEmailVerificationDisabled = true
	}

	isMagicLinkLoginDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMagicLinkLogin)
	if err != nil {
		log.Debug("Failed to get Disable Magic Link Login from environment variable", err)
		isMagicLinkLoginDisabled = true
	}

	isSignUpDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableSignUp)
	if err != nil {
		log.Debug("Failed to get Disable Signup from environment variable", err)
		isSignUpDisabled = true
	}

	isStrongPasswordDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableStrongPassword)
	if err != nil {
		log.Debug("Failed to get Disable Signup from environment variable", err)
		isSignUpDisabled = true
	}

	isMultiFactorAuthenticationEnabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMultiFactorAuthentication)
	if err != nil {
		log.Debug("Failed to get Disable Multi Factor Authentication from environment variable", err)
		isSignUpDisabled = true
	}

	metaInfo := model.Meta{
		Version:                            constants.VERSION,
		ClientID:                           clientID,
		IsGoogleLoginEnabled:               googleClientID != "" && googleClientSecret != "",
		IsGithubLoginEnabled:               githubClientID != "" && githubClientSecret != "",
		IsFacebookLoginEnabled:             facebookClientID != "" && facebookClientSecret != "",
		IsLinkedinLoginEnabled:             linkedClientID != "" && linkedInClientSecret != "",
		IsAppleLoginEnabled:                appleClientID != "" && appleClientSecret != "",
		IsTwitterLoginEnabled:              twitterClientID != "" && twitterClientSecret != "",
		IsMicrosoftLoginEnabled:            microsoftClientID != "" && microsoftClientSecret != "",
		IsBasicAuthenticationEnabled:       !isBasicAuthDisabled,
		IsEmailVerificationEnabled:         !isEmailVerificationDisabled,
		IsMagicLinkLoginEnabled:            !isMagicLinkLoginDisabled,
		IsSignUpEnabled:                    !isSignUpDisabled,
		IsStrongPasswordEnabled:            !isStrongPasswordDisabled,
		IsMultiFactorAuthEnabled:           !isMultiFactorAuthenticationEnabled,
		IsMobileBasicAuthenticationEnabled: !isMobileBasicAuthDisabled,
		IsPhoneVerificationEnabled:         !isMobileVerificationDisabled,
	}
	return &metaInfo, nil
}
