package test

import (
	"log"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
)

func TestResolvers(t *testing.T) {
	databases := map[string]string{
		enum.Sqlite.String():   "../../data.db",
		enum.Arangodb.String(): "http://root:root@localhost:8529",
		enum.Mongodb.String():  "mongodb://localhost:27017",
	}

	log.Println("==== Testing resolvers =====")

	for dbType, dbURL := range databases {
		constants.DATABASE_URL = dbURL
		constants.DATABASE_TYPE = dbType
		db.InitDB()
		s := testSetup()
		defer s.Server.Close()
		t.Run("running test cases for "+dbType, func(t *testing.T) {
			loginTests(s, t)
			signupTests(s, t)
			forgotPasswordTest(s, t)
			resendVerifyEmailTests(s, t)
			resetPasswordTest(s, t)
			verifyEmailTest(s, t)
		})
	}
}
