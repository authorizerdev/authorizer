package test

import (
	"context"
	"os"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/memorystore"
)

func TestResolvers(t *testing.T) {
	databases := map[string]string{
		// constants.DbTypeSqlite:   "../../data.db",
		// constants.DbTypeArangodb: "http://localhost:8529",
		// constants.DbTypeMongodb: "mongodb://localhost:27017",
		constants.DbTypeScyllaDB: "127.0.0.1:9042",
	}

	testDb := "authorizer_test"
	s := testSetup()
	defer s.Server.Close()

	for dbType, dbURL := range databases {
		ctx := context.Background()

		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDatabaseURL, dbURL)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDatabaseType, dbType)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDatabaseName, testDb)
		os.Setenv(constants.EnvKeyDatabaseURL, dbURL)
		os.Setenv(constants.EnvKeyDatabaseType, dbType)
		os.Setenv(constants.EnvKeyDatabaseName, testDb)
		memorystore.InitRequiredEnv()

		err := db.InitDB()
		if err != nil {
			t.Errorf("Error initializing database: %s", err.Error())
		}

		// clean the persisted config for test to use fresh config
		envData, err := db.Provider.GetEnv(ctx)
		if err == nil {
			envData.EnvData = ""
			_, err = db.Provider.UpdateEnv(ctx, envData)
			if err != nil {
				t.Errorf("Error updating env: %s", err.Error())
			}
		}
		err = env.PersistEnv()
		if err != nil {
			t.Errorf("Error persisting env: %s", err.Error())
		}

		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyEnv, "test")
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyIsProd, false)
		t.Run("should pass tests for "+dbType, func(t *testing.T) {
			// admin resolvers tests
			adminSignupTests(t, s)
			addWebhookTest(t, s) // add webhooks for all the system events
			testEndpointTest(t, s)
			verificationRequestsTest(t, s)
			updateWebhookTest(t, s)
			webhookTest(t, s)
			webhooksTest(t, s)
			usersTest(t, s)
			deleteUserTest(t, s)
			updateUserTest(t, s)
			adminLoginTests(t, s)
			adminLogoutTests(t, s)
			adminSessionTests(t, s)
			updateEnvTests(t, s)
			envTests(t, s)
			revokeAccessTest(t, s)
			enableAccessTest(t, s)
			generateJWTkeyTest(t, s)

			// user resolvers tests
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
			inviteUserTest(t, s)
			validateJwtTokenTest(t, s)

			webhookLogsTest(t, s)   // get logs after above resolver tests are done
			deleteWebhookTest(t, s) // delete webhooks (admin resolver)
		})
	}
}
