package env

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/utils"
)

// InitEnv to initialize EnvData and through error if required env are not present
func InitAllEnv() error {
	envData, err := GetEnvData()
	if err != nil || envData == nil {
		log.Info("No env data found in db, using local clone of env data")
		// get clone of current store
		envData, err = memorystore.Provider.GetEnvStore()
		if err != nil {
			log.Debug("Error while getting env data from memorystore: ", err)
			return err
		}
	}

	// unique client id for each instance
	cid, ok := envData[constants.EnvKeyClientID]
	clientID := ""
	if !ok || cid == "" {
		clientID = uuid.New().String()
		envData[constants.EnvKeyClientID] = clientID
	} else {
		clientID = cid.(string)
	}

	// unique client secret for each instance
	if val, ok := envData[constants.EnvKeyClientSecret]; !ok || val != "" {
		envData[constants.EnvKeyClientSecret] = uuid.New().String()
	}

	// os string envs
	osEnv := os.Getenv(constants.EnvKeyEnv)
	osAppURL := os.Getenv(constants.EnvKeyAppURL)
	osAuthorizerURL := os.Getenv(constants.EnvKeyAuthorizerURL)
	osPort := os.Getenv(constants.EnvKeyPort)
	osAccessTokenExpiryTime := os.Getenv(constants.EnvKeyAccessTokenExpiryTime)
	osAdminSecret := os.Getenv(constants.EnvKeyAdminSecret)
	osSmtpHost := os.Getenv(constants.EnvKeySmtpHost)
	osSmtpPort := os.Getenv(constants.EnvKeySmtpPort)
	osSmtpUsername := os.Getenv(constants.EnvKeySmtpUsername)
	osSmtpPassword := os.Getenv(constants.EnvKeySmtpPassword)
	osSmtpLocalName := os.Getenv(constants.EnvKeySmtpLocalName)
	osSenderEmail := os.Getenv(constants.EnvKeySenderEmail)
	osSenderName := os.Getenv(constants.EnvKeySenderName)
	osJwtType := os.Getenv(constants.EnvKeyJwtType)
	osJwtSecret := os.Getenv(constants.EnvKeyJwtSecret)
	osJwtPrivateKey := os.Getenv(constants.EnvKeyJwtPrivateKey)
	osJwtPublicKey := os.Getenv(constants.EnvKeyJwtPublicKey)
	osJwtRoleClaim := os.Getenv(constants.EnvKeyJwtRoleClaim)
	osCustomAccessTokenScript := os.Getenv(constants.EnvKeyCustomAccessTokenScript)
	osGoogleClientID := os.Getenv(constants.EnvKeyGoogleClientID)
	osGoogleClientSecret := os.Getenv(constants.EnvKeyGoogleClientSecret)
	osGithubClientID := os.Getenv(constants.EnvKeyGithubClientID)
	osGithubClientSecret := os.Getenv(constants.EnvKeyGithubClientSecret)
	osFacebookClientID := os.Getenv(constants.EnvKeyFacebookClientID)
	osFacebookClientSecret := os.Getenv(constants.EnvKeyFacebookClientSecret)
	osLinkedInClientID := os.Getenv(constants.EnvKeyLinkedInClientID)
	osLinkedInClientSecret := os.Getenv(constants.EnvKeyLinkedInClientSecret)
	osAppleClientID := os.Getenv(constants.EnvKeyAppleClientID)
	osAppleClientSecret := os.Getenv(constants.EnvKeyAppleClientSecret)
	osTwitterClientID := os.Getenv(constants.EnvKeyTwitterClientID)
	osTwitterClientSecret := os.Getenv(constants.EnvKeyTwitterClientSecret)
	osMicrosoftClientID := os.Getenv(constants.EnvKeyMicrosoftClientID)
	osMicrosoftClientSecret := os.Getenv(constants.EnvKeyMicrosoftClientSecret)
	osMicrosoftActiveDirectoryTenantID := os.Getenv(constants.EnvKeyMicrosoftActiveDirectoryTenantID)
	osResetPasswordURL := os.Getenv(constants.EnvKeyResetPasswordURL)
	osOrganizationName := os.Getenv(constants.EnvKeyOrganizationName)
	osOrganizationLogo := os.Getenv(constants.EnvKeyOrganizationLogo)
	osAwsRegion := os.Getenv(constants.EnvAwsRegion)
	osAwsAccessKey := os.Getenv(constants.EnvAwsAccessKeyID)
	osAwsSecretKey := os.Getenv(constants.EnvAwsSecretAccessKey)
	osCouchbaseBucket := os.Getenv(constants.EnvCouchbaseBucket)
	osCouchbaseScope := os.Getenv(constants.EnvCouchbaseScope)
	osCouchbaseBucketRAMQuotaMB := os.Getenv(constants.EnvCouchbaseBucketRAMQuotaMB)
	osAuthorizeResponseType := os.Getenv(constants.EnvKeyDefaultAuthorizeResponseType)
	osAuthorizeResponseMode := os.Getenv(constants.EnvKeyDefaultAuthorizeResponseMode)

	// os bool vars
	osAppCookieSecure := os.Getenv(constants.EnvKeyAppCookieSecure)
	osAdminCookieSecure := os.Getenv(constants.EnvKeyAdminCookieSecure)
	osDisableBasicAuthentication := os.Getenv(constants.EnvKeyDisableBasicAuthentication)
	osDisableMobileBasicAuthentication := os.Getenv(constants.AuthRecipeMethodMobileBasicAuth)
	osDisableEmailVerification := os.Getenv(constants.EnvKeyDisableEmailVerification)
	osDisableMagicLinkLogin := os.Getenv(constants.EnvKeyDisableMagicLinkLogin)
	osDisableLoginPage := os.Getenv(constants.EnvKeyDisableLoginPage)
	osDisableSignUp := os.Getenv(constants.EnvKeyDisableSignUp)
	osDisableRedisForEnv := os.Getenv(constants.EnvKeyDisableRedisForEnv)
	osDisableStrongPassword := os.Getenv(constants.EnvKeyDisableStrongPassword)
	osEnforceMultiFactorAuthentication := os.Getenv(constants.EnvKeyEnforceMultiFactorAuthentication)
	osDisableMultiFactorAuthentication := os.Getenv(constants.EnvKeyDisableMultiFactorAuthentication)
	// phone verification var
	osDisablePhoneVerification := os.Getenv(constants.EnvKeyDisablePhoneVerification)
	// twilio vars
	osTwilioApiKey := os.Getenv(constants.EnvKeyTwilioAPIKey)
	osTwilioApiSecret := os.Getenv(constants.EnvKeyTwilioAPISecret)
	osTwilioAccountSid := os.Getenv(constants.EnvKeyTwilioAccountSID)
	osTwilioSender := os.Getenv(constants.EnvKeyTwilioSender)

	// os slice vars
	osAllowedOrigins := os.Getenv(constants.EnvKeyAllowedOrigins)
	osRoles := os.Getenv(constants.EnvKeyRoles)
	osDefaultRoles := os.Getenv(constants.EnvKeyDefaultRoles)
	osProtectedRoles := os.Getenv(constants.EnvKeyProtectedRoles)

	ienv, ok := envData[constants.EnvKeyEnv]
	if !ok || ienv == "" {
		envData[constants.EnvKeyEnv] = osEnv
		if envData[constants.EnvKeyEnv] == "" {
			envData[constants.EnvKeyEnv] = "production"
		}

		if envData[constants.EnvKeyEnv] == "production" {
			envData[constants.EnvKeyIsProd] = true
		} else {
			envData[constants.EnvKeyIsProd] = false
		}
	}
	if osEnv != "" && osEnv != envData[constants.EnvKeyEnv] {
		envData[constants.EnvKeyEnv] = osEnv
		if envData[constants.EnvKeyEnv] == "production" {
			envData[constants.EnvKeyIsProd] = true
		} else {
			envData[constants.EnvKeyIsProd] = false
		}
	}

	if val, ok := envData[constants.EnvAwsRegion]; !ok || val == "" {
		envData[constants.EnvAwsRegion] = osAwsRegion
	}

	if osAwsRegion != "" && envData[constants.EnvAwsRegion] != osAwsRegion {
		envData[constants.EnvAwsRegion] = osAwsRegion
	}

	if val, ok := envData[constants.EnvAwsAccessKeyID]; !ok || val == "" {
		envData[constants.EnvAwsAccessKeyID] = osAwsAccessKey
	}
	if osAwsAccessKey != "" && envData[constants.EnvAwsAccessKeyID] != osAwsAccessKey {
		envData[constants.EnvAwsAccessKeyID] = osAwsAccessKey
	}

	if val, ok := envData[constants.EnvAwsSecretAccessKey]; !ok || val == "" {
		envData[constants.EnvAwsSecretAccessKey] = osAwsSecretKey
	}
	if osAwsSecretKey != "" && envData[constants.EnvAwsSecretAccessKey] != osAwsSecretKey {
		envData[constants.EnvAwsSecretAccessKey] = osAwsSecretKey
	}

	if val, ok := envData[constants.EnvCouchbaseBucket]; !ok || val == "" {
		envData[constants.EnvCouchbaseBucket] = osCouchbaseBucket
	}
	if osCouchbaseBucket != "" && envData[constants.EnvCouchbaseBucket] != osCouchbaseBucket {
		envData[constants.EnvCouchbaseBucket] = osCouchbaseBucket
	}

	if val, ok := envData[constants.EnvCouchbaseBucketRAMQuotaMB]; !ok || val == "" {
		envData[constants.EnvCouchbaseBucketRAMQuotaMB] = osCouchbaseBucketRAMQuotaMB
	}
	if osCouchbaseBucketRAMQuotaMB != "" && envData[constants.EnvCouchbaseBucketRAMQuotaMB] != osCouchbaseBucketRAMQuotaMB {
		envData[constants.EnvCouchbaseBucketRAMQuotaMB] = osCouchbaseBucketRAMQuotaMB
	}

	if val, ok := envData[constants.EnvCouchbaseScope]; !ok || val == "" {
		envData[constants.EnvCouchbaseScope] = osCouchbaseScope
	}
	if osCouchbaseScope != "" && envData[constants.EnvCouchbaseScope] != osCouchbaseScope {
		envData[constants.EnvCouchbaseScope] = osCouchbaseScope
	}

	if val, ok := envData[constants.EnvKeyAppURL]; !ok || val == "" {
		envData[constants.EnvKeyAppURL] = osAppURL
	}
	if osAppURL != "" && envData[constants.EnvKeyAppURL] != osAppURL {
		envData[constants.EnvKeyAppURL] = osAppURL
	}

	if val, ok := envData[constants.EnvKeyAuthorizerURL]; !ok || val == "" {
		envData[constants.EnvKeyAuthorizerURL] = osAuthorizerURL
	}
	if osAuthorizerURL != "" && envData[constants.EnvKeyAuthorizerURL] != osAuthorizerURL {
		envData[constants.EnvKeyAuthorizerURL] = osAuthorizerURL
	}

	if val, ok := envData[constants.EnvKeyPort]; !ok || val == "" {
		envData[constants.EnvKeyPort] = osPort
		if envData[constants.EnvKeyPort] == "" {
			envData[constants.EnvKeyPort] = "8080"
		}
	}
	if osPort != "" && envData[constants.EnvKeyPort] != osPort {
		envData[constants.EnvKeyPort] = osPort
	}

	if val, ok := envData[constants.EnvKeyAccessTokenExpiryTime]; !ok || val == "" {
		envData[constants.EnvKeyAccessTokenExpiryTime] = osAccessTokenExpiryTime
		if envData[constants.EnvKeyAccessTokenExpiryTime] == "" {
			envData[constants.EnvKeyAccessTokenExpiryTime] = "30m"
		}
	}
	if osAccessTokenExpiryTime != "" && envData[constants.EnvKeyAccessTokenExpiryTime] != osAccessTokenExpiryTime {
		envData[constants.EnvKeyAccessTokenExpiryTime] = osAccessTokenExpiryTime
	}

	if val, ok := envData[constants.EnvKeyAdminSecret]; !ok || val == "" {
		envData[constants.EnvKeyAdminSecret] = osAdminSecret
	}
	if osAdminSecret != "" && envData[constants.EnvKeyAdminSecret] != osAdminSecret {
		envData[constants.EnvKeyAdminSecret] = osAdminSecret
	}

	if val, ok := envData[constants.EnvKeySmtpHost]; !ok || val == "" {
		envData[constants.EnvKeySmtpHost] = osSmtpHost
	}
	if osSmtpHost != "" && envData[constants.EnvKeySmtpHost] != osSmtpHost {
		envData[constants.EnvKeySmtpHost] = osSmtpHost
	}

	if val, ok := envData[constants.EnvKeySmtpPort]; !ok || val == "" {
		envData[constants.EnvKeySmtpPort] = osSmtpPort
	}
	if osSmtpPort != "" && envData[constants.EnvKeySmtpPort] != osSmtpPort {
		envData[constants.EnvKeySmtpPort] = osSmtpPort
	}

	if val, ok := envData[constants.EnvKeySmtpUsername]; !ok || val == "" {
		envData[constants.EnvKeySmtpUsername] = osSmtpUsername
	}
	if osSmtpUsername != "" && envData[constants.EnvKeySmtpUsername] != osSmtpUsername {
		envData[constants.EnvKeySmtpUsername] = osSmtpUsername
	}

	if val, ok := envData[constants.EnvKeySmtpLocalName]; !ok || val == "" {
		envData[constants.EnvKeySmtpLocalName] = osSmtpLocalName
	}
	if osSmtpLocalName != "" && envData[constants.EnvKeySmtpLocalName] != osSmtpLocalName {
		envData[constants.EnvKeySmtpLocalName] = osSmtpLocalName
	}

	if val, ok := envData[constants.EnvKeySmtpPassword]; !ok || val == "" {
		envData[constants.EnvKeySmtpPassword] = osSmtpPassword
	}
	if osSmtpPassword != "" && envData[constants.EnvKeySmtpPassword] != osSmtpPassword {
		envData[constants.EnvKeySmtpPassword] = osSmtpPassword
	}

	if val, ok := envData[constants.EnvKeySenderEmail]; !ok || val == "" {
		envData[constants.EnvKeySenderEmail] = osSenderEmail
	}
	if osSenderEmail != "" && envData[constants.EnvKeySenderEmail] != osSenderEmail {
		envData[constants.EnvKeySenderEmail] = osSenderEmail
	}

	if val, ok := envData[constants.EnvKeySenderName]; !ok || val == "" {
		envData[constants.EnvKeySenderName] = osSenderName
	}
	if osSenderName != "" && envData[constants.EnvKeySenderName] != osSenderName {
		envData[constants.EnvKeySenderName] = osSenderName
	}

	algoVal, ok := envData[constants.EnvKeyJwtType]
	algo := ""
	if !ok || algoVal == "" {
		envData[constants.EnvKeyJwtType] = osJwtType
		if envData[constants.EnvKeyJwtType] == "" {
			envData[constants.EnvKeyJwtType] = "RS256"
			algo = envData[constants.EnvKeyJwtType].(string)
		}
	} else {
		algo = algoVal.(string)
		if !crypto.IsHMACA(algo) && !crypto.IsRSA(algo) && !crypto.IsECDSA(algo) {
			log.Debug("Invalid JWT Algorithm")
			return errors.New("invalid JWT_TYPE")
		}
	}
	if osJwtType != "" && osJwtType != algo {
		if !crypto.IsHMACA(osJwtType) && !crypto.IsRSA(osJwtType) && !crypto.IsECDSA(osJwtType) {
			log.Debug("Invalid JWT Algorithm")
			return errors.New("invalid JWT_TYPE")
		}
		algo = osJwtType
		envData[constants.EnvKeyJwtType] = osJwtType
	}

	if crypto.IsHMACA(algo) {
		if val, ok := envData[constants.EnvKeyJwtSecret]; !ok || val == "" {
			envData[constants.EnvKeyJwtSecret] = osJwtSecret
			if envData[constants.EnvKeyJwtSecret] == "" {
				envData[constants.EnvKeyJwtSecret], _, err = crypto.NewHMACKey(algo, clientID)
				if err != nil {
					return err
				}
			}
		}
		if osJwtSecret != "" && envData[constants.EnvKeyJwtSecret] != osJwtSecret {
			envData[constants.EnvKeyJwtSecret] = osJwtSecret
		}
	}

	if crypto.IsRSA(algo) || crypto.IsECDSA(algo) {
		privateKey, publicKey := "", ""

		if val, ok := envData[constants.EnvKeyJwtPrivateKey]; !ok || val == "" {
			privateKey = osJwtPrivateKey
		}
		if osJwtPrivateKey != "" && privateKey != osJwtPrivateKey {
			privateKey = osJwtPrivateKey
		}

		if val, ok := envData[constants.EnvKeyJwtPublicKey]; !ok || val == "" {
			publicKey = osJwtPublicKey
		}
		if osJwtPublicKey != "" && publicKey != osJwtPublicKey {
			publicKey = osJwtPublicKey
		}

		// if algo is RSA / ECDSA, then we need to have both private and public key
		// if either of them is not present generate new keys
		if privateKey == "" || publicKey == "" {
			if crypto.IsRSA(algo) {
				_, privateKey, publicKey, _, err = crypto.NewRSAKey(algo, clientID)
				if err != nil {
					return err
				}
			} else if crypto.IsECDSA(algo) {
				_, privateKey, publicKey, _, err = crypto.NewECDSAKey(algo, clientID)
				if err != nil {
					return err
				}
			}
		} else {
			// parse keys to make sure they are valid
			if crypto.IsRSA(algo) {
				_, err = crypto.ParseRsaPrivateKeyFromPemStr(privateKey)
				if err != nil {
					return err
				}

				_, err := crypto.ParseRsaPublicKeyFromPemStr(publicKey)
				if err != nil {
					return err
				}

			} else if crypto.IsECDSA(algo) {
				_, err = crypto.ParseEcdsaPrivateKeyFromPemStr(privateKey)
				if err != nil {
					return err
				}

				_, err := crypto.ParseEcdsaPublicKeyFromPemStr(publicKey)
				if err != nil {
					return err
				}
			}
		}

		envData[constants.EnvKeyJwtPrivateKey] = privateKey
		envData[constants.EnvKeyJwtPublicKey] = publicKey

	}

	if val, ok := envData[constants.EnvKeyJwtRoleClaim]; !ok || val == "" {
		envData[constants.EnvKeyJwtRoleClaim] = osJwtRoleClaim

		if envData[constants.EnvKeyJwtRoleClaim] == "" {
			envData[constants.EnvKeyJwtRoleClaim] = "roles"
		}
	}
	if osJwtRoleClaim != "" && envData[constants.EnvKeyJwtRoleClaim] != osJwtRoleClaim {
		envData[constants.EnvKeyJwtRoleClaim] = osJwtRoleClaim
	}

	if val, ok := envData[constants.EnvKeyCustomAccessTokenScript]; !ok || val == "" {
		envData[constants.EnvKeyCustomAccessTokenScript] = osCustomAccessTokenScript
	}
	if osCustomAccessTokenScript != "" && envData[constants.EnvKeyCustomAccessTokenScript] != osCustomAccessTokenScript {
		envData[constants.EnvKeyCustomAccessTokenScript] = osCustomAccessTokenScript
	}

	if val, ok := envData[constants.EnvKeyGoogleClientID]; !ok || val == "" {
		envData[constants.EnvKeyGoogleClientID] = osGoogleClientID
	}
	if osGoogleClientID != "" && envData[constants.EnvKeyGoogleClientID] != osGoogleClientID {
		envData[constants.EnvKeyGoogleClientID] = osGoogleClientID
	}

	if val, ok := envData[constants.EnvKeyGoogleClientSecret]; !ok || val == "" {
		envData[constants.EnvKeyGoogleClientSecret] = osGoogleClientSecret
	}
	if osGoogleClientSecret != "" && envData[constants.EnvKeyGoogleClientSecret] != osGoogleClientSecret {
		envData[constants.EnvKeyGoogleClientSecret] = osGoogleClientSecret
	}

	if val, ok := envData[constants.EnvKeyGithubClientID]; !ok || val == "" {
		envData[constants.EnvKeyGithubClientID] = osGithubClientID
	}
	if osGithubClientID != "" && envData[constants.EnvKeyGithubClientID] != osGithubClientID {
		envData[constants.EnvKeyGithubClientID] = osGithubClientID
	}

	if val, ok := envData[constants.EnvKeyGithubClientSecret]; !ok || val == "" {
		envData[constants.EnvKeyGithubClientSecret] = osGithubClientSecret
	}
	if osGithubClientSecret != "" && envData[constants.EnvKeyGithubClientSecret] != osGithubClientSecret {
		envData[constants.EnvKeyGithubClientSecret] = osGithubClientSecret
	}

	if val, ok := envData[constants.EnvKeyFacebookClientID]; !ok || val == "" {
		envData[constants.EnvKeyFacebookClientID] = osFacebookClientID
	}
	if osFacebookClientID != "" && envData[constants.EnvKeyFacebookClientID] != osFacebookClientID {
		envData[constants.EnvKeyFacebookClientID] = osFacebookClientID
	}

	if val, ok := envData[constants.EnvKeyFacebookClientSecret]; !ok || val == "" {
		envData[constants.EnvKeyFacebookClientSecret] = osFacebookClientSecret
	}
	if osFacebookClientSecret != "" && envData[constants.EnvKeyFacebookClientSecret] != osFacebookClientSecret {
		envData[constants.EnvKeyFacebookClientSecret] = osFacebookClientSecret
	}

	if val, ok := envData[constants.EnvKeyLinkedInClientID]; !ok || val == "" {
		envData[constants.EnvKeyLinkedInClientID] = osLinkedInClientID
	}
	if osLinkedInClientID != "" && envData[constants.EnvKeyLinkedInClientID] != osLinkedInClientID {
		envData[constants.EnvKeyLinkedInClientID] = osLinkedInClientID
	}

	if val, ok := envData[constants.EnvKeyLinkedInClientSecret]; !ok || val == "" {
		envData[constants.EnvKeyLinkedInClientSecret] = osLinkedInClientSecret
	}
	if osLinkedInClientSecret != "" && envData[constants.EnvKeyLinkedInClientSecret] != osLinkedInClientSecret {
		envData[constants.EnvKeyLinkedInClientSecret] = osLinkedInClientSecret
	}

	if val, ok := envData[constants.EnvKeyAppleClientID]; !ok || val == "" {
		envData[constants.EnvKeyAppleClientID] = osAppleClientID
	}
	if osAppleClientID != "" && envData[constants.EnvKeyAppleClientID] != osAppleClientID {
		envData[constants.EnvKeyAppleClientID] = osAppleClientID
	}

	if val, ok := envData[constants.EnvKeyAppleClientSecret]; !ok || val == "" {
		envData[constants.EnvKeyAppleClientSecret] = osAppleClientSecret
	}
	if osAppleClientSecret != "" && envData[constants.EnvKeyAppleClientSecret] != osAppleClientSecret {
		envData[constants.EnvKeyAppleClientSecret] = osAppleClientSecret
	}

	if val, ok := envData[constants.EnvKeyTwitterClientID]; !ok || val == "" {
		envData[constants.EnvKeyTwitterClientID] = osTwitterClientID
	}
	if osTwitterClientID != "" && envData[constants.EnvKeyTwitterClientID] != osTwitterClientID {
		envData[constants.EnvKeyTwitterClientID] = osTwitterClientID
	}

	if val, ok := envData[constants.EnvKeyTwitterClientSecret]; !ok || val == "" {
		envData[constants.EnvKeyTwitterClientSecret] = osTwitterClientSecret
	}
	if osTwitterClientSecret != "" && envData[constants.EnvKeyTwitterClientSecret] != osTwitterClientSecret {
		envData[constants.EnvKeyTwitterClientSecret] = osTwitterClientSecret
	}

	if val, ok := envData[constants.EnvKeyMicrosoftClientID]; !ok || val == "" {
		envData[constants.EnvKeyMicrosoftClientID] = osMicrosoftClientID
	}
	if osMicrosoftClientID != "" && envData[constants.EnvKeyMicrosoftClientID] != osMicrosoftClientID {
		envData[constants.EnvKeyMicrosoftClientID] = osMicrosoftClientID
	}

	if val, ok := envData[constants.EnvKeyMicrosoftClientSecret]; !ok || val == "" {
		envData[constants.EnvKeyMicrosoftClientSecret] = osMicrosoftClientSecret
	}
	if osMicrosoftClientSecret != "" && envData[constants.EnvKeyMicrosoftClientSecret] != osMicrosoftClientSecret {
		envData[constants.EnvKeyMicrosoftClientSecret] = osMicrosoftClientSecret
	}

	if val, ok := envData[constants.EnvKeyMicrosoftActiveDirectoryTenantID]; !ok || val == "" {
		envData[constants.EnvKeyMicrosoftActiveDirectoryTenantID] = osMicrosoftActiveDirectoryTenantID
	}
	if osMicrosoftActiveDirectoryTenantID != "" && envData[constants.EnvKeyMicrosoftActiveDirectoryTenantID] != osMicrosoftActiveDirectoryTenantID {
		envData[constants.EnvKeyMicrosoftActiveDirectoryTenantID] = osMicrosoftActiveDirectoryTenantID
	}

	if val, ok := envData[constants.EnvKeyResetPasswordURL]; !ok || val == "" {
		envData[constants.EnvKeyResetPasswordURL] = strings.TrimPrefix(osResetPasswordURL, "/")
	}
	if osResetPasswordURL != "" && envData[constants.EnvKeyResetPasswordURL] != osResetPasswordURL {
		envData[constants.EnvKeyResetPasswordURL] = osResetPasswordURL
	}

	if val, ok := envData[constants.EnvKeyOrganizationName]; !ok || val == "" {
		envData[constants.EnvKeyOrganizationName] = osOrganizationName
	}
	if osOrganizationName != "" && envData[constants.EnvKeyOrganizationName] != osOrganizationName {
		envData[constants.EnvKeyOrganizationName] = osOrganizationName
	}

	if val, ok := envData[constants.EnvKeyOrganizationLogo]; !ok || val == "" {
		envData[constants.EnvKeyOrganizationLogo] = osOrganizationLogo
	}
	if osOrganizationLogo != "" && envData[constants.EnvKeyOrganizationLogo] != osOrganizationLogo {
		envData[constants.EnvKeyOrganizationLogo] = osOrganizationLogo
	}

	if _, ok := envData[constants.EnvKeyAppCookieSecure]; !ok {
		if osAppCookieSecure == "" {
			envData[constants.EnvKeyAppCookieSecure] = true
		} else {
			envData[constants.EnvKeyAppCookieSecure] = osAppCookieSecure == "true"
		}
	}
	if osAppCookieSecure != "" {
		boolValue, err := strconv.ParseBool(osAppCookieSecure)
		if err != nil {
			return err
		}
		if boolValue != envData[constants.EnvKeyAppCookieSecure].(bool) {
			envData[constants.EnvKeyAppCookieSecure] = boolValue
		}
	}

	if _, ok := envData[constants.EnvKeyAdminCookieSecure]; !ok {
		if osAdminCookieSecure == "" {
			envData[constants.EnvKeyAdminCookieSecure] = true
		} else {
			envData[constants.EnvKeyAdminCookieSecure] = osAdminCookieSecure == "true"
		}
	}
	if osAdminCookieSecure != "" {
		boolValue, err := strconv.ParseBool(osAdminCookieSecure)
		if err != nil {
			return err
		}
		if boolValue != envData[constants.EnvKeyAdminCookieSecure].(bool) {
			envData[constants.EnvKeyAdminCookieSecure] = boolValue
		}
	}

	if _, ok := envData[constants.EnvKeyDisableBasicAuthentication]; !ok {
		envData[constants.EnvKeyDisableBasicAuthentication] = osDisableBasicAuthentication == "true"
	}
	if osDisableBasicAuthentication != "" {
		boolValue, err := strconv.ParseBool(osDisableBasicAuthentication)
		if err != nil {
			return err
		}
		if boolValue != envData[constants.EnvKeyDisableBasicAuthentication].(bool) {
			envData[constants.EnvKeyDisableBasicAuthentication] = boolValue
		}
	}

	if _, ok := envData[constants.EnvKeyDisableMobileBasicAuthentication]; !ok {
		envData[constants.EnvKeyDisableMobileBasicAuthentication] = osDisableBasicAuthentication == "true"
	}
	if osDisableMobileBasicAuthentication != "" {
		boolValue, err := strconv.ParseBool(osDisableMobileBasicAuthentication)
		if err != nil {
			return err
		}
		if boolValue != envData[constants.EnvKeyDisableMobileBasicAuthentication].(bool) {
			envData[constants.EnvKeyDisableMobileBasicAuthentication] = boolValue
		}
	}

	if _, ok := envData[constants.EnvKeyDisableEmailVerification]; !ok {
		envData[constants.EnvKeyDisableEmailVerification] = osDisableEmailVerification == "true"
	}
	if osDisableEmailVerification != "" {
		boolValue, err := strconv.ParseBool(osDisableEmailVerification)
		if err != nil {
			return err
		}
		if boolValue != envData[constants.EnvKeyDisableEmailVerification].(bool) {
			envData[constants.EnvKeyDisableEmailVerification] = boolValue
		}
	}

	if _, ok := envData[constants.EnvKeyDisableMagicLinkLogin]; !ok {
		envData[constants.EnvKeyDisableMagicLinkLogin] = osDisableMagicLinkLogin == "true"
	}
	if osDisableMagicLinkLogin != "" {
		boolValue, err := strconv.ParseBool(osDisableMagicLinkLogin)
		if err != nil {
			return err
		}
		if boolValue != envData[constants.EnvKeyDisableMagicLinkLogin] {
			envData[constants.EnvKeyDisableMagicLinkLogin] = boolValue
		}
	}

	if _, ok := envData[constants.EnvKeyDisableLoginPage]; !ok {
		envData[constants.EnvKeyDisableLoginPage] = osDisableLoginPage == "true"
	}
	if osDisableLoginPage != "" {
		boolValue, err := strconv.ParseBool(osDisableLoginPage)
		if err != nil {
			return err
		}
		if boolValue != envData[constants.EnvKeyDisableLoginPage].(bool) {
			envData[constants.EnvKeyDisableLoginPage] = boolValue
		}
	}

	if _, ok := envData[constants.EnvKeyDisableSignUp]; !ok {
		envData[constants.EnvKeyDisableSignUp] = osDisableSignUp == "true"
	}
	if osDisableSignUp != "" {
		boolValue, err := strconv.ParseBool(osDisableSignUp)
		if err != nil {
			return err
		}
		if boolValue != envData[constants.EnvKeyDisableSignUp].(bool) {
			envData[constants.EnvKeyDisableSignUp] = boolValue
		}
	}

	if _, ok := envData[constants.EnvKeyDisableRedisForEnv]; !ok {
		envData[constants.EnvKeyDisableRedisForEnv] = osDisableRedisForEnv == "true"
	}
	if osDisableRedisForEnv != "" {
		boolValue, err := strconv.ParseBool(osDisableRedisForEnv)
		if err != nil {
			return err
		}
		if boolValue != envData[constants.EnvKeyDisableRedisForEnv].(bool) {
			envData[constants.EnvKeyDisableRedisForEnv] = boolValue
		}
	}

	if _, ok := envData[constants.EnvKeyDisableStrongPassword]; !ok {
		envData[constants.EnvKeyDisableStrongPassword] = osDisableStrongPassword == "true"
	}
	if osDisableStrongPassword != "" {
		boolValue, err := strconv.ParseBool(osDisableStrongPassword)
		if err != nil {
			return err
		}
		if boolValue != envData[constants.EnvKeyDisableStrongPassword].(bool) {
			envData[constants.EnvKeyDisableStrongPassword] = boolValue
		}
	}

	if _, ok := envData[constants.EnvKeyEnforceMultiFactorAuthentication]; !ok {
		envData[constants.EnvKeyEnforceMultiFactorAuthentication] = osEnforceMultiFactorAuthentication == "true"
	}
	if osEnforceMultiFactorAuthentication != "" {
		boolValue, err := strconv.ParseBool(osEnforceMultiFactorAuthentication)
		if err != nil {
			return err
		}
		if boolValue != envData[constants.EnvKeyEnforceMultiFactorAuthentication].(bool) {
			envData[constants.EnvKeyEnforceMultiFactorAuthentication] = boolValue
		}
	}

	if _, ok := envData[constants.EnvKeyDisableMultiFactorAuthentication]; !ok {
		envData[constants.EnvKeyDisableMultiFactorAuthentication] = osDisableMultiFactorAuthentication == "true"
	}
	if osDisableMultiFactorAuthentication != "" {
		boolValue, err := strconv.ParseBool(osDisableMultiFactorAuthentication)
		if err != nil {
			return err
		}
		if boolValue != envData[constants.EnvKeyDisableMultiFactorAuthentication].(bool) {
			envData[constants.EnvKeyDisableMultiFactorAuthentication] = boolValue
		}
	}

	// no need to add nil check as its already done above
	if envData[constants.EnvKeySmtpHost] == "" || envData[constants.EnvKeySmtpUsername] == "" || envData[constants.EnvKeySmtpPassword] == "" || envData[constants.EnvKeySenderEmail] == "" && envData[constants.EnvKeySmtpPort] == "" {
		envData[constants.EnvKeyDisableEmailVerification] = true
		envData[constants.EnvKeyDisableMagicLinkLogin] = true
		envData[constants.EnvKeyIsEmailServiceEnabled] = false
	}

	if envData[constants.EnvKeySmtpHost] != "" && envData[constants.EnvKeySmtpUsername] != "" && envData[constants.EnvKeySmtpPassword] != "" && envData[constants.EnvKeySenderEmail] != "" && envData[constants.EnvKeySmtpPort] != "" {
		envData[constants.EnvKeyIsEmailServiceEnabled] = true
	}

	if envData[constants.EnvKeyEnforceMultiFactorAuthentication].(bool) && !envData[constants.EnvKeyIsEmailServiceEnabled].(bool) && !envData[constants.EnvKeyIsSMSServiceEnabled].(bool) {
		return errors.New("to enable multi factor authentication, please enable email service")
	}

	if !envData[constants.EnvKeyIsEmailServiceEnabled].(bool) {
		envData[constants.EnvKeyDisableMultiFactorAuthentication] = true
	}

	if envData[constants.EnvKeyDisableEmailVerification].(bool) {
		envData[constants.EnvKeyDisableMagicLinkLogin] = true
	}

	if val, ok := envData[constants.EnvKeyAllowedOrigins]; !ok || val == "" {
		envData[constants.EnvKeyAllowedOrigins] = osAllowedOrigins
		if envData[constants.EnvKeyAllowedOrigins] == "" {
			envData[constants.EnvKeyAllowedOrigins] = "*"
		}
	}
	if osAllowedOrigins != "" && envData[constants.EnvKeyAllowedOrigins] != osAllowedOrigins {
		envData[constants.EnvKeyAllowedOrigins] = osAllowedOrigins
	}

	if val, ok := envData[constants.EnvKeyRoles]; !ok || val == "" {
		envData[constants.EnvKeyRoles] = osRoles
		if envData[constants.EnvKeyRoles] == "" {
			envData[constants.EnvKeyRoles] = "user"
		}
	}
	if osRoles != "" && envData[constants.EnvKeyRoles] != osRoles {
		envData[constants.EnvKeyRoles] = osRoles
	}
	roles := strings.Split(envData[constants.EnvKeyRoles].(string), ",")

	if val, ok := envData[constants.EnvKeyDefaultRoles]; !ok || val == "" {
		envData[constants.EnvKeyDefaultRoles] = osDefaultRoles
		if envData[constants.EnvKeyDefaultRoles] == "" {
			envData[constants.EnvKeyDefaultRoles] = "user"
		}
	}
	if osDefaultRoles != "" && envData[constants.EnvKeyDefaultRoles] != osDefaultRoles {
		envData[constants.EnvKeyDefaultRoles] = osDefaultRoles
	}
	defaultRoles := strings.Split(envData[constants.EnvKeyDefaultRoles].(string), ",")
	if len(defaultRoles) == 0 {
		defaultRoles = []string{roles[0]}
	}

	for _, role := range defaultRoles {
		if !utils.StringSliceContains(roles, role) {
			return fmt.Errorf("Default role %s is not defined in roles", role)
		}
	}

	if val, ok := envData[constants.EnvKeyProtectedRoles]; !ok || val == "" {
		envData[constants.EnvKeyProtectedRoles] = osProtectedRoles
	}
	if osProtectedRoles != "" && envData[constants.EnvKeyProtectedRoles] != osProtectedRoles {
		envData[constants.EnvKeyProtectedRoles] = osProtectedRoles
	}

	if val, ok := envData[constants.EnvKeyDefaultAuthorizeResponseType]; !ok || val == "" {
		envData[constants.EnvKeyDefaultAuthorizeResponseType] = osAuthorizeResponseType
		// Set the default value to token type
		if envData[constants.EnvKeyDefaultAuthorizeResponseType] == "" {
			envData[constants.EnvKeyDefaultAuthorizeResponseType] = constants.ResponseTypeToken
		}
	}
	if osAuthorizeResponseType != "" && envData[constants.EnvKeyDefaultAuthorizeResponseType] != osAuthorizeResponseType {
		envData[constants.EnvKeyDefaultAuthorizeResponseType] = osAuthorizeResponseType
	}

	if val, ok := envData[constants.EnvKeyDefaultAuthorizeResponseMode]; !ok || val == "" {
		envData[constants.EnvKeyDefaultAuthorizeResponseMode] = osAuthorizeResponseMode
		// Set the default value to token type
		if envData[constants.EnvKeyDefaultAuthorizeResponseMode] == "" {
			envData[constants.EnvKeyDefaultAuthorizeResponseMode] = constants.ResponseModeQuery
		}
	}
	if osAuthorizeResponseMode != "" && envData[constants.EnvKeyDefaultAuthorizeResponseMode] != osAuthorizeResponseMode {
		envData[constants.EnvKeyDefaultAuthorizeResponseMode] = osAuthorizeResponseMode
	}

	if val, ok := envData[constants.EnvKeyTwilioAPISecret]; !ok || val == "" {
		envData[constants.EnvKeyTwilioAPISecret] = osTwilioApiSecret
	}
	if osTwilioApiSecret != "" && envData[constants.EnvKeyTwilioAPISecret] != osTwilioApiSecret {
		envData[constants.EnvKeyTwilioAPISecret] = osTwilioApiSecret
	}

	if val, ok := envData[constants.EnvKeyTwilioAPIKey]; !ok || val == "" {
		envData[constants.EnvKeyTwilioAPIKey] = osTwilioApiKey
	}
	if osTwilioApiKey != "" && envData[constants.EnvKeyTwilioAPIKey] != osTwilioApiKey {
		envData[constants.EnvKeyTwilioAPIKey] = osTwilioApiKey
	}

	if val, ok := envData[constants.EnvKeyTwilioAccountSID]; !ok || val == "" {
		envData[constants.EnvKeyTwilioAccountSID] = osTwilioAccountSid
	}
	if osTwilioAccountSid != "" && envData[constants.EnvKeyTwilioAccountSID] != osTwilioAccountSid {
		envData[constants.EnvKeyTwilioAccountSID] = osTwilioAccountSid
	}

	if val, ok := envData[constants.EnvKeyTwilioSender]; !ok || val == "" {
		envData[constants.EnvKeyTwilioSender] = osTwilioSender
	}
	if osTwilioSender != "" && envData[constants.EnvKeyTwilioSender] != osTwilioSender {
		envData[constants.EnvKeyTwilioSender] = osTwilioSender
	}

	if _, ok := envData[constants.EnvKeyDisablePhoneVerification]; !ok {
		envData[constants.EnvKeyDisablePhoneVerification] = osDisablePhoneVerification == "false"
	}
	if osDisablePhoneVerification != "" {
		boolValue, err := strconv.ParseBool(osDisablePhoneVerification)
		if err != nil {
			return err
		}
		if boolValue != envData[constants.EnvKeyDisablePhoneVerification] {
			envData[constants.EnvKeyDisablePhoneVerification] = boolValue
		}
	}

	if envData[constants.EnvKeyTwilioAPIKey] == "" || envData[constants.EnvKeyTwilioAPISecret] == "" || envData[constants.EnvKeyTwilioAccountSID] == "" || envData[constants.EnvKeyTwilioSender] == "" {
		envData[constants.EnvKeyDisablePhoneVerification] = true
		envData[constants.EnvKeyIsSMSServiceEnabled] = false
	}
	if envData[constants.EnvKeyTwilioAPIKey] != "" && envData[constants.EnvKeyTwilioAPISecret] != "" && envData[constants.EnvKeyTwilioAccountSID] != "" && envData[constants.EnvKeyTwilioSender] != "" {
		envData[constants.EnvKeyDisablePhoneVerification] = false
		envData[constants.EnvKeyIsSMSServiceEnabled] = true
	}

	err = memorystore.Provider.UpdateEnvStore(envData)
	if err != nil {
		log.Debug("Error while updating env store: ", err)
		return err
	}
	return nil
}
