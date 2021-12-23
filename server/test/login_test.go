package test

import (
	"log"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func commonLoginTest(s TestSetup, t *testing.T) {
	email := "login." + s.TestInfo.Email
	_, err := resolvers.Signup(s.Ctx, model.SignUpInput{
		Email:           email,
		Password:        s.TestInfo.Password,
		ConfirmPassword: s.TestInfo.Password,
	})

	_, err = resolvers.Login(s.Ctx, model.LoginInput{
		Email:    email,
		Password: s.TestInfo.Password,
	})

	assert.NotNil(t, err, "should fail because email is not verified")

	verificationRequest, err := db.Mgr.GetVerificationByEmail(email, enum.BasicAuthSignup.String())
	resolvers.VerifyEmail(s.Ctx, model.VerifyEmailInput{
		Token: verificationRequest.Token,
	})

	_, err = resolvers.Login(s.Ctx, model.LoginInput{
		Email:    email,
		Password: s.TestInfo.Password,
		Roles:    []string{"test"},
	})
	assert.NotNil(t, err, "invalid roles")

	_, err = resolvers.Login(s.Ctx, model.LoginInput{
		Email:    email,
		Password: s.TestInfo.Password + "s",
	})
	assert.NotNil(t, err, "invalid password")

	loginRes, err := resolvers.Login(s.Ctx, model.LoginInput{
		Email:    email,
		Password: s.TestInfo.Password,
	})

	log.Println("=> access token:", loginRes.AccessToken)
	assert.Nil(t, err, "login successful")
	assert.NotNil(t, loginRes.AccessToken, "access token should not be empty")

	cleanData(email)
}

func TestLogin(t *testing.T) {
	s := testSetup()
	defer s.Server.Close()

	if s.TestInfo.ShouldExecuteForSQL {
		t.Run("login for sql dbs should pass", func(t *testing.T) {
			constants.DATABASE_URL = s.TestInfo.SQL
			constants.DATABASE_TYPE = enum.Sqlite.String()
			db.InitDB()
			commonLoginTest(s, t)
		})
	}

	if s.TestInfo.ShouldExecuteForArango {
		t.Run("login for arangodb should pass", func(t *testing.T) {
			constants.DATABASE_URL = s.TestInfo.ArangoDB
			constants.DATABASE_TYPE = enum.Arangodb.String()
			db.InitDB()
			commonLoginTest(s, t)
		})
	}

	if s.TestInfo.ShouldExecuteForMongo {
		t.Run("login for mongodb should pass", func(t *testing.T) {
			constants.DATABASE_URL = s.TestInfo.MongoDB
			constants.DATABASE_TYPE = enum.Mongodb.String()
			db.InitDB()
			commonLoginTest(s, t)
		})
	}
}
