package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/envstore"
)

func TestResolvers(t *testing.T) {
	databases := map[string]string{
		constants.DbTypeSqlite:   "../../data.db",
		constants.DbTypeArangodb: "http://localhost:8529",
		constants.DbTypeMongodb:  "mongodb://localhost:27017",
	}
	envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyVersion, "test")
	for dbType, dbURL := range databases {
		envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyDatabaseURL, dbURL)
		envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyDatabaseType, dbType)

		env.InitEnv()
		db.InitDB()

		// clean the persisted config for test to use fresh config
		envData, err := db.Provider.GetEnv()
		if err == nil {
			envData.EnvData = ""
			db.Provider.UpdateEnv(envData)
		}
		env.PersistEnv()

		s := testSetup()
		defer s.Server.Close()

		t.Run("should pass tests for "+dbType, func(t *testing.T) {
			// admin tests
			adminSignupTests(t, s)
			verificationRequestsTest(t, s)
			usersTest(t, s)
			deleteUserTest(t, s)
			updateUserTest(t, s)
			adminLoginTests(t, s)
			adminLogoutTests(t, s)
			adminSessionTests(t, s)
			updateEnvTests(t, s)
			envTests(t, s)

			// user tests
			loginTests(t, s)
			signupTests(t, s)
			forgotPasswordTest(t, s)
			resendVerifyEmailTests(t, s)
			resetPasswordTest(t, s)
			verifyEmailTest(t, s)
			sessionTests(t, s)
			profileTests(t, s)
			updateProfileTests(t, s)
			magicLinkLoginTests(t, s)
			logoutTests(t, s)
			metaTests(t, s)
			isValidJWTTests(t, s)
		})
	}
}
