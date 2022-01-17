package test

import (
	"log"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/env"
)

func TestResolvers(t *testing.T) {
	databases := map[string]string{
		enum.Sqlite.String():   "../../data.db",
		enum.Arangodb.String(): "http://localhost:8529",
		enum.Mongodb.String():  "mongodb://localhost:27017",
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
			adminSignupTests(s, t)
			verificationRequestsTest(s, t)
			usersTest(s, t)
			deleteUserTest(s, t)
			updateUserTest(s, t)
			adminLoginTests(s, t)
			adminLogoutTests(s, t)
			adminSessionTests(s, t)
			updateConfigTests(s, t)
			configTests(s, t)

			// user tests
			loginTests(s, t)
			signupTests(s, t)
			forgotPasswordTest(s, t)
			resendVerifyEmailTests(s, t)
			resetPasswordTest(s, t)
			verifyEmailTest(s, t)
			sessionTests(s, t)
			profileTests(s, t)
			updateProfileTests(s, t)
			magicLinkLoginTests(s, t)
			logoutTests(s, t)
			metaTests(s, t)
		})
	}
}
