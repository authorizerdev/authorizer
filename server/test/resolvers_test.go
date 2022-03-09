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
		constants.DbTypeSqlite: "../../data.db",
		// constants.DbTypeArangodb: "http://localhost:8529",
		// constants.DbTypeMongodb:  "mongodb://localhost:27017",
	}
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyVersion, "test")
	for dbType, dbURL := range databases {
		s := testSetup()
		envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyDatabaseURL, dbURL)
		envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyDatabaseType, dbType)
		defer s.Server.Close()
		err := db.InitDB()
		if err != nil {
			t.Errorf("Error initializing database: %s", err)
		}

		// clean the persisted config for test to use fresh config
		envData, err := db.Provider.GetEnv()
		if err == nil {
			envData.EnvData = ""
			db.Provider.UpdateEnv(envData)
		}
		env.PersistEnv()

		envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyEnv, "test")
		envstore.EnvStoreObj.UpdateEnvVariable(constants.BoolStoreIdentifier, constants.EnvKeyIsProd, false)
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
		})
	}
}
