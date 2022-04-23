package env

import (
	"errors"
	"log"
	"os"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// InitRequiredEnv to initialize EnvData and through error if required env are not present
func InitRequiredEnv() error {
	envPath := os.Getenv(constants.EnvKeyEnvPath)

	if envPath == "" {
		envPath = envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyEnvPath)
		if envPath == "" {
			envPath = `.env`
		}
	}

	if envstore.ARG_ENV_FILE != nil && *envstore.ARG_ENV_FILE != "" {
		envPath = *envstore.ARG_ENV_FILE
	}

	err := godotenv.Load(envPath)
	if err != nil {
		log.Printf("using OS env instead of %s file", envPath)
	}

	dbURL := os.Getenv(constants.EnvKeyDatabaseURL)
	dbType := os.Getenv(constants.EnvKeyDatabaseType)
	dbName := os.Getenv(constants.EnvKeyDatabaseName)
	dbPort := os.Getenv(constants.EnvKeyDatabasePort)
	dbHost := os.Getenv(constants.EnvKeyDatabaseHost)
	dbUsername := os.Getenv(constants.EnvKeyDatabaseUsername)
	dbPassword := os.Getenv(constants.EnvKeyDatabasePassword)
	dbCert := os.Getenv(constants.EnvKeyDatabaseCert)
	dbCertKey := os.Getenv(constants.EnvKeyDatabaseCertKey)
	dbCACert := os.Getenv(constants.EnvKeyDatabaseCACert)

	if strings.TrimSpace(dbType) == "" {
		if envstore.ARG_DB_TYPE != nil && *envstore.ARG_DB_TYPE != "" {
			dbType = strings.TrimSpace(*envstore.ARG_DB_TYPE)
		}

		if dbType == "" {
			return errors.New("invalid database type. DATABASE_TYPE is empty")
		}
	}

	if strings.TrimSpace(dbURL) == "" && envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL) == "" {
		if envstore.ARG_DB_URL != nil && *envstore.ARG_DB_URL != "" {
			dbURL = strings.TrimSpace(*envstore.ARG_DB_URL)
		}

		if dbURL == "" && dbPort == "" && dbHost == "" && dbUsername == "" && dbPassword == "" {
			return errors.New("invalid database url. DATABASE_URL is required")
		}
	}

	if dbName == "" {
		if dbName == "" {
			dbName = "authorizer"
		}
	}

	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyEnvPath, envPath)
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyDatabaseURL, dbURL)
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyDatabaseType, dbType)
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyDatabaseName, dbName)
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyDatabaseHost, dbHost)
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyDatabasePort, dbPort)
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyDatabaseUsername, dbUsername)
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyDatabasePassword, dbPassword)
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyDatabaseCert, dbCert)
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyDatabaseCertKey, dbCertKey)
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyDatabaseCACert, dbCACert)

	return nil
}

// InitEnv to initialize EnvData and through error if required env are not present
func InitAllEnv() error {
	envData, err := GetEnvData()
	if err != nil {
		log.Println("No env data found in db, using local clone of env data")
		// get clone of current store
		envData = envstore.EnvStoreObj.GetEnvStoreClone()
	}

	clientID := envData.StringEnv[constants.EnvKeyClientID]
	// unique client id for each instance
	if clientID == "" {
		clientID = uuid.New().String()
		envData.StringEnv[constants.EnvKeyClientID] = clientID
	}

	clientSecret := envData.StringEnv[constants.EnvKeyClientSecret]
	// unique client id for each instance
	if clientSecret == "" {
		clientSecret = uuid.New().String()
		envData.StringEnv[constants.EnvKeyClientSecret] = clientSecret
	}

	if envData.StringEnv[constants.EnvKeyEnv] == "" {
		envData.StringEnv[constants.EnvKeyEnv] = os.Getenv(constants.EnvKeyEnv)
		if envData.StringEnv[constants.EnvKeyEnv] == "" {
			envData.StringEnv[constants.EnvKeyEnv] = "production"
		}

		if envData.StringEnv[constants.EnvKeyEnv] == "production" {
			envData.BoolEnv[constants.EnvKeyIsProd] = true
			gin.SetMode(gin.ReleaseMode)
		} else {
			envData.BoolEnv[constants.EnvKeyIsProd] = false
		}
	}

	if envData.StringEnv[constants.EnvKeyAppURL] == "" {
		envData.StringEnv[constants.EnvKeyAppURL] = os.Getenv(constants.EnvKeyAppURL)
	}

	if envData.StringEnv[constants.EnvKeyAuthorizerURL] == "" {
		envData.StringEnv[constants.EnvKeyAuthorizerURL] = os.Getenv(constants.EnvKeyAuthorizerURL)
	}

	if envData.StringEnv[constants.EnvKeyPort] == "" {
		envData.StringEnv[constants.EnvKeyPort] = os.Getenv(constants.EnvKeyPort)
		if envData.StringEnv[constants.EnvKeyPort] == "" {
			envData.StringEnv[constants.EnvKeyPort] = "8080"
		}
	}

	if envData.StringEnv[constants.EnvKeyAccessTokenExpiryTime] == "" {
		envData.StringEnv[constants.EnvKeyAccessTokenExpiryTime] = os.Getenv(constants.EnvKeyAccessTokenExpiryTime)
		if envData.StringEnv[constants.EnvKeyAccessTokenExpiryTime] == "" {
			envData.StringEnv[constants.EnvKeyAccessTokenExpiryTime] = "30m"
		}
	}

	if envData.StringEnv[constants.EnvKeyAdminSecret] == "" {
		envData.StringEnv[constants.EnvKeyAdminSecret] = os.Getenv(constants.EnvKeyAdminSecret)
	}

	if envData.StringEnv[constants.EnvKeySmtpHost] == "" {
		envData.StringEnv[constants.EnvKeySmtpHost] = os.Getenv(constants.EnvKeySmtpHost)
	}

	if envData.StringEnv[constants.EnvKeySmtpPort] == "" {
		envData.StringEnv[constants.EnvKeySmtpPort] = os.Getenv(constants.EnvKeySmtpPort)
	}

	if envData.StringEnv[constants.EnvKeySmtpUsername] == "" {
		envData.StringEnv[constants.EnvKeySmtpUsername] = os.Getenv(constants.EnvKeySmtpUsername)
	}

	if envData.StringEnv[constants.EnvKeySmtpPassword] == "" {
		envData.StringEnv[constants.EnvKeySmtpPassword] = os.Getenv(constants.EnvKeySmtpPassword)
	}

	if envData.StringEnv[constants.EnvKeySenderEmail] == "" {
		envData.StringEnv[constants.EnvKeySenderEmail] = os.Getenv(constants.EnvKeySenderEmail)
	}

	algo := envData.StringEnv[constants.EnvKeyJwtType]
	if algo == "" {
		envData.StringEnv[constants.EnvKeyJwtType] = os.Getenv(constants.EnvKeyJwtType)
		if envData.StringEnv[constants.EnvKeyJwtType] == "" {
			envData.StringEnv[constants.EnvKeyJwtType] = "RS256"
			algo = envData.StringEnv[constants.EnvKeyJwtType]
		} else {
			algo = envData.StringEnv[constants.EnvKeyJwtType]
			if !crypto.IsHMACA(algo) && !crypto.IsRSA(algo) && !crypto.IsECDSA(algo) {
				return errors.New("invalid JWT_TYPE")
			}
		}
	}

	if crypto.IsHMACA(algo) {
		if envData.StringEnv[constants.EnvKeyJwtSecret] == "" {
			envData.StringEnv[constants.EnvKeyJwtSecret] = os.Getenv(constants.EnvKeyJwtSecret)
			if envData.StringEnv[constants.EnvKeyJwtSecret] == "" {
				envData.StringEnv[constants.EnvKeyJwtSecret], _, err = crypto.NewHMACKey(algo, clientID)
				if err != nil {
					return err
				}
			}
		}
	}

	if crypto.IsRSA(algo) || crypto.IsECDSA(algo) {
		privateKey, publicKey := "", ""

		if envData.StringEnv[constants.EnvKeyJwtPrivateKey] == "" {
			privateKey = os.Getenv(constants.EnvKeyJwtPrivateKey)
		}

		if envData.StringEnv[constants.EnvKeyJwtPublicKey] == "" {
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

		envData.StringEnv[constants.EnvKeyJwtPrivateKey] = privateKey
		envData.StringEnv[constants.EnvKeyJwtPublicKey] = publicKey

	}

	if envData.StringEnv[constants.EnvKeyJwtRoleClaim] == "" {
		envData.StringEnv[constants.EnvKeyJwtRoleClaim] = os.Getenv(constants.EnvKeyJwtRoleClaim)

		if envData.StringEnv[constants.EnvKeyJwtRoleClaim] == "" {
			envData.StringEnv[constants.EnvKeyJwtRoleClaim] = "role"
		}
	}

	if envData.StringEnv[constants.EnvKeyCustomAccessTokenScript] == "" {
		envData.StringEnv[constants.EnvKeyCustomAccessTokenScript] = os.Getenv(constants.EnvKeyCustomAccessTokenScript)
	}

	if envData.StringEnv[constants.EnvKeyRedisURL] == "" {
		envData.StringEnv[constants.EnvKeyRedisURL] = os.Getenv(constants.EnvKeyRedisURL)
	}

	if envData.StringEnv[constants.EnvKeyCookieName] == "" {
		envData.StringEnv[constants.EnvKeyCookieName] = os.Getenv(constants.EnvKeyCookieName)
		if envData.StringEnv[constants.EnvKeyCookieName] == "" {
			envData.StringEnv[constants.EnvKeyCookieName] = "authorizer"
		}
	}

	if envData.StringEnv[constants.EnvKeyGoogleClientID] == "" {
		envData.StringEnv[constants.EnvKeyGoogleClientID] = os.Getenv(constants.EnvKeyGoogleClientID)
	}

	if envData.StringEnv[constants.EnvKeyGoogleClientSecret] == "" {
		envData.StringEnv[constants.EnvKeyGoogleClientSecret] = os.Getenv(constants.EnvKeyGoogleClientSecret)
	}

	if envData.StringEnv[constants.EnvKeyGithubClientID] == "" {
		envData.StringEnv[constants.EnvKeyGithubClientID] = os.Getenv(constants.EnvKeyGithubClientID)
	}

	if envData.StringEnv[constants.EnvKeyGithubClientSecret] == "" {
		envData.StringEnv[constants.EnvKeyGithubClientSecret] = os.Getenv(constants.EnvKeyGithubClientSecret)
	}

	if envData.StringEnv[constants.EnvKeyFacebookClientID] == "" {
		envData.StringEnv[constants.EnvKeyFacebookClientID] = os.Getenv(constants.EnvKeyFacebookClientID)
	}

	if envData.StringEnv[constants.EnvKeyFacebookClientSecret] == "" {
		envData.StringEnv[constants.EnvKeyFacebookClientSecret] = os.Getenv(constants.EnvKeyFacebookClientSecret)
	}

	if envData.StringEnv[constants.EnvKeyResetPasswordURL] == "" {
		envData.StringEnv[constants.EnvKeyResetPasswordURL] = strings.TrimPrefix(os.Getenv(constants.EnvKeyResetPasswordURL), "/")
	}

	envData.BoolEnv[constants.EnvKeyDisableBasicAuthentication] = os.Getenv(constants.EnvKeyDisableBasicAuthentication) == "true"
	envData.BoolEnv[constants.EnvKeyDisableEmailVerification] = os.Getenv(constants.EnvKeyDisableEmailVerification) == "true"
	envData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] = os.Getenv(constants.EnvKeyDisableMagicLinkLogin) == "true"
	envData.BoolEnv[constants.EnvKeyDisableLoginPage] = os.Getenv(constants.EnvKeyDisableLoginPage) == "true"
	envData.BoolEnv[constants.EnvKeyDisableSignUp] = os.Getenv(constants.EnvKeyDisableSignUp) == "true"

	// no need to add nil check as its already done above
	if envData.StringEnv[constants.EnvKeySmtpHost] == "" || envData.StringEnv[constants.EnvKeySmtpUsername] == "" || envData.StringEnv[constants.EnvKeySmtpPassword] == "" || envData.StringEnv[constants.EnvKeySenderEmail] == "" && envData.StringEnv[constants.EnvKeySmtpPort] == "" {
		envData.BoolEnv[constants.EnvKeyDisableEmailVerification] = true
		envData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] = true
	}

	if envData.BoolEnv[constants.EnvKeyDisableEmailVerification] {
		envData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] = true
	}

	allowedOriginsSplit := strings.Split(os.Getenv(constants.EnvKeyAllowedOrigins), ",")
	allowedOrigins := []string{}
	hasWildCard := false

	for _, val := range allowedOriginsSplit {
		trimVal := strings.TrimSpace(val)
		if trimVal != "" {
			if trimVal != "*" {
				host, port := utils.GetHostParts(trimVal)
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

	envData.SliceEnv[constants.EnvKeyAllowedOrigins] = allowedOrigins

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
		return errors.New(`invalid DEFAULT_ROLE environment variable. It can be one from give ROLES environment variable value`)
	}

	envData.SliceEnv[constants.EnvKeyRoles] = roles
	envData.SliceEnv[constants.EnvKeyDefaultRoles] = defaultRoles
	envData.SliceEnv[constants.EnvKeyProtectedRoles] = protectedRoles

	if os.Getenv(constants.EnvKeyOrganizationName) != "" {
		envData.StringEnv[constants.EnvKeyOrganizationName] = os.Getenv(constants.EnvKeyOrganizationName)
	}

	if os.Getenv(constants.EnvKeyOrganizationLogo) != "" {
		envData.StringEnv[constants.EnvKeyOrganizationLogo] = os.Getenv(constants.EnvKeyOrganizationLogo)
	}

	envstore.EnvStoreObj.UpdateEnvStore(envData)
	return nil
}
