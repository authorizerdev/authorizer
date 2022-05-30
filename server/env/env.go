package env

import (
	"errors"
	"os"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/parsers"
	"github.com/authorizerdev/authorizer/server/utils"
)

// InitEnv to initialize EnvData and through error if required env are not present
func InitAllEnv() error {
	envData, err := GetEnvData()
	if err != nil {
		log.Info("No env data found in db, using local clone of env data")
		// get clone of current store
		envData, err = memorystore.Provider.GetEnvStore()
		if err != nil {
			log.Debug("Error while getting env data from memorystore: ", err)
			return err
		}
	}

	clientID := envData[constants.EnvKeyClientID].(string)
	// unique client id for each instance
	if clientID == "" {
		clientID = uuid.New().String()
		envData[constants.EnvKeyClientID] = clientID
	}

	clientSecret := envData[constants.EnvKeyClientSecret]
	// unique client id for each instance
	if clientSecret == "" {
		clientSecret = uuid.New().String()
		envData[constants.EnvKeyClientSecret] = clientSecret
	}

	if envData[constants.EnvKeyEnv] == "" {
		envData[constants.EnvKeyEnv] = os.Getenv(constants.EnvKeyEnv)
		if envData[constants.EnvKeyEnv] == "" {
			envData[constants.EnvKeyEnv] = "production"
		}

		if envData[constants.EnvKeyEnv] == "production" {
			envData[constants.EnvKeyIsProd] = true
		} else {
			envData[constants.EnvKeyIsProd] = false
		}
	}

	if envData[constants.EnvKeyAppURL] == "" {
		envData[constants.EnvKeyAppURL] = os.Getenv(constants.EnvKeyAppURL)
	}

	if envData[constants.EnvKeyAuthorizerURL] == "" {
		envData[constants.EnvKeyAuthorizerURL] = os.Getenv(constants.EnvKeyAuthorizerURL)
	}

	if envData[constants.EnvKeyPort] == "" {
		envData[constants.EnvKeyPort] = os.Getenv(constants.EnvKeyPort)
		if envData[constants.EnvKeyPort] == "" {
			envData[constants.EnvKeyPort] = "8080"
		}
	}

	if envData[constants.EnvKeyAccessTokenExpiryTime] == "" {
		envData[constants.EnvKeyAccessTokenExpiryTime] = os.Getenv(constants.EnvKeyAccessTokenExpiryTime)
		if envData[constants.EnvKeyAccessTokenExpiryTime] == "" {
			envData[constants.EnvKeyAccessTokenExpiryTime] = "30m"
		}
	}

	if envData[constants.EnvKeyAdminSecret] == "" {
		envData[constants.EnvKeyAdminSecret] = os.Getenv(constants.EnvKeyAdminSecret)
	}

	if envData[constants.EnvKeySmtpHost] == "" {
		envData[constants.EnvKeySmtpHost] = os.Getenv(constants.EnvKeySmtpHost)
	}

	if envData[constants.EnvKeySmtpPort] == "" {
		envData[constants.EnvKeySmtpPort] = os.Getenv(constants.EnvKeySmtpPort)
	}

	if envData[constants.EnvKeySmtpUsername] == "" {
		envData[constants.EnvKeySmtpUsername] = os.Getenv(constants.EnvKeySmtpUsername)
	}

	if envData[constants.EnvKeySmtpPassword] == "" {
		envData[constants.EnvKeySmtpPassword] = os.Getenv(constants.EnvKeySmtpPassword)
	}

	if envData[constants.EnvKeySenderEmail] == "" {
		envData[constants.EnvKeySenderEmail] = os.Getenv(constants.EnvKeySenderEmail)
	}

	algo := envData[constants.EnvKeyJwtType].(string)
	if algo == "" {
		envData[constants.EnvKeyJwtType] = os.Getenv(constants.EnvKeyJwtType)
		if envData[constants.EnvKeyJwtType] == "" {
			envData[constants.EnvKeyJwtType] = "RS256"
			algo = envData[constants.EnvKeyJwtType].(string)
		} else {
			algo = envData[constants.EnvKeyJwtType].(string)
			if !crypto.IsHMACA(algo) && !crypto.IsRSA(algo) && !crypto.IsECDSA(algo) {
				log.Debug("Invalid JWT Algorithm")
				return errors.New("invalid JWT_TYPE")
			}
		}
	}

	if crypto.IsHMACA(algo) {
		if envData[constants.EnvKeyJwtSecret] == "" {
			envData[constants.EnvKeyJwtSecret] = os.Getenv(constants.EnvKeyJwtSecret)
			if envData[constants.EnvKeyJwtSecret] == "" {
				envData[constants.EnvKeyJwtSecret], _, err = crypto.NewHMACKey(algo, clientID)
				if err != nil {
					return err
				}
			}
		}
	}

	if crypto.IsRSA(algo) || crypto.IsECDSA(algo) {
		privateKey, publicKey := "", ""

		if envData[constants.EnvKeyJwtPrivateKey] == "" {
			privateKey = os.Getenv(constants.EnvKeyJwtPrivateKey)
		}

		if envData[constants.EnvKeyJwtPublicKey] == "" {
			publicKey = os.Getenv(constants.EnvKeyJwtPublicKey)
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

	if envData[constants.EnvKeyJwtRoleClaim] == "" {
		envData[constants.EnvKeyJwtRoleClaim] = os.Getenv(constants.EnvKeyJwtRoleClaim)

		if envData[constants.EnvKeyJwtRoleClaim] == "" {
			envData[constants.EnvKeyJwtRoleClaim] = "role"
		}
	}

	if envData[constants.EnvKeyCustomAccessTokenScript] == "" {
		envData[constants.EnvKeyCustomAccessTokenScript] = os.Getenv(constants.EnvKeyCustomAccessTokenScript)
	}

	if envData[constants.EnvKeyRedisURL] == "" {
		envData[constants.EnvKeyRedisURL] = os.Getenv(constants.EnvKeyRedisURL)
	}

	if envData[constants.EnvKeyGoogleClientID] == "" {
		envData[constants.EnvKeyGoogleClientID] = os.Getenv(constants.EnvKeyGoogleClientID)
	}

	if envData[constants.EnvKeyGoogleClientSecret] == "" {
		envData[constants.EnvKeyGoogleClientSecret] = os.Getenv(constants.EnvKeyGoogleClientSecret)
	}

	if envData[constants.EnvKeyGithubClientID] == "" {
		envData[constants.EnvKeyGithubClientID] = os.Getenv(constants.EnvKeyGithubClientID)
	}

	if envData[constants.EnvKeyGithubClientSecret] == "" {
		envData[constants.EnvKeyGithubClientSecret] = os.Getenv(constants.EnvKeyGithubClientSecret)
	}

	if envData[constants.EnvKeyFacebookClientID] == "" {
		envData[constants.EnvKeyFacebookClientID] = os.Getenv(constants.EnvKeyFacebookClientID)
	}

	if envData[constants.EnvKeyFacebookClientSecret] == "" {
		envData[constants.EnvKeyFacebookClientSecret] = os.Getenv(constants.EnvKeyFacebookClientSecret)
	}

	if envData[constants.EnvKeyResetPasswordURL] == "" {
		envData[constants.EnvKeyResetPasswordURL] = strings.TrimPrefix(os.Getenv(constants.EnvKeyResetPasswordURL), "/")
	}

	envData[constants.EnvKeyDisableBasicAuthentication] = os.Getenv(constants.EnvKeyDisableBasicAuthentication) == "true"
	envData[constants.EnvKeyDisableEmailVerification] = os.Getenv(constants.EnvKeyDisableEmailVerification) == "true"
	envData[constants.EnvKeyDisableMagicLinkLogin] = os.Getenv(constants.EnvKeyDisableMagicLinkLogin) == "true"
	envData[constants.EnvKeyDisableLoginPage] = os.Getenv(constants.EnvKeyDisableLoginPage) == "true"
	envData[constants.EnvKeyDisableSignUp] = os.Getenv(constants.EnvKeyDisableSignUp) == "true"

	// no need to add nil check as its already done above
	if envData[constants.EnvKeySmtpHost] == "" || envData[constants.EnvKeySmtpUsername] == "" || envData[constants.EnvKeySmtpPassword] == "" || envData[constants.EnvKeySenderEmail] == "" && envData[constants.EnvKeySmtpPort] == "" {
		envData[constants.EnvKeyDisableEmailVerification] = true
		envData[constants.EnvKeyDisableMagicLinkLogin] = true
	}

	if envData[constants.EnvKeyDisableEmailVerification].(bool) {
		envData[constants.EnvKeyDisableMagicLinkLogin] = true
	}

	allowedOriginsSplit := strings.Split(os.Getenv(constants.EnvKeyAllowedOrigins), ",")
	allowedOrigins := []string{}
	hasWildCard := false

	for _, val := range allowedOriginsSplit {
		trimVal := strings.TrimSpace(val)
		if trimVal != "" {
			if trimVal != "*" {
				host, port := parsers.GetHostParts(trimVal)
				allowedOrigins = append(allowedOrigins, host+":"+port)
			} else {
				hasWildCard = true
				allowedOrigins = append(allowedOrigins, trimVal)
				break
			}
		}
	}

	if len(allowedOrigins) > 1 && hasWildCard {
		allowedOrigins = []string{"*"}
	}

	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	envData[constants.EnvKeyAllowedOrigins] = allowedOrigins

	rolesEnv := strings.TrimSpace(os.Getenv(constants.EnvKeyRoles))
	rolesSplit := strings.Split(rolesEnv, ",")
	roles := []string{}
	if len(rolesEnv) == 0 {
		roles = []string{"user"}
	}

	defaultRolesEnv := strings.TrimSpace(os.Getenv(constants.EnvKeyDefaultRoles))
	defaultRoleSplit := strings.Split(defaultRolesEnv, ",")
	defaultRoles := []string{}

	if len(defaultRolesEnv) == 0 {
		defaultRoles = []string{"user"}
	}

	protectedRolesEnv := strings.TrimSpace(os.Getenv(constants.EnvKeyProtectedRoles))
	protectedRolesSplit := strings.Split(protectedRolesEnv, ",")
	protectedRoles := []string{}

	if len(protectedRolesEnv) > 0 {
		for _, val := range protectedRolesSplit {
			trimVal := strings.TrimSpace(val)
			protectedRoles = append(protectedRoles, trimVal)
		}
	}

	for _, val := range rolesSplit {
		trimVal := strings.TrimSpace(val)
		if trimVal != "" {
			roles = append(roles, trimVal)
			if utils.StringSliceContains(defaultRoleSplit, trimVal) {
				defaultRoles = append(defaultRoles, trimVal)
			}
		}
	}

	if len(roles) > 0 && len(defaultRoles) == 0 && len(defaultRolesEnv) > 0 {
		log.Debug("Default roles not found in roles list. It can be one from ROLES only")
		return errors.New(`invalid DEFAULT_ROLE environment variable. It can be one from give ROLES environment variable value`)
	}

	envData[constants.EnvKeyRoles] = roles
	envData[constants.EnvKeyDefaultRoles] = defaultRoles
	envData[constants.EnvKeyProtectedRoles] = protectedRoles

	if os.Getenv(constants.EnvKeyOrganizationName) != "" {
		envData[constants.EnvKeyOrganizationName] = os.Getenv(constants.EnvKeyOrganizationName)
	}

	if os.Getenv(constants.EnvKeyOrganizationLogo) != "" {
		envData[constants.EnvKeyOrganizationLogo] = os.Getenv(constants.EnvKeyOrganizationLogo)
	}

	memorystore.Provider.UpdateEnvStore(envData)
	return nil
}
