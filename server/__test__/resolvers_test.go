package test

import (
	"log"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/env"
)

func TestResolvers(t *testing.T) {
	databases := map[string]string{
		constants.DbTypeSqlite: "../../data.db",
		// constants.DbTypeArangodb: "http://localhost:8529",
		// constants.DbTypeMongodb:  "mongodb://localhost:27017",
	}

	for dbType, dbURL := range databases {
		constants.EnvData.DATABASE_URL = dbURL
		constants.EnvData.DATABASE_TYPE = dbType
		db.InitDB()

		// clean the persisted config for test to use fresh config
		config, err := db.Mgr.GetConfig()
		if err == nil {
			config.Config = []byte{}
			db.Mgr.UpdateConfig(config)
		}
		env.PersistEnv()

		s := testSetup()
		defer s.Server.Close()

		log.Println("EnvData:", constants.EnvData)
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
			updateConfigTests(t, s)
			configTests(t, s)

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
