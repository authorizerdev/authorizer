package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func commonForgotPasswordTest(s TestSetup, t *testing.T) {
	email := "forgot_password." + s.TestInfo.Email
	_, err := resolvers.Signup(s.Ctx, model.SignUpInput{
		Email:           email,
		Password:        s.TestInfo.Password,
		ConfirmPassword: s.TestInfo.Password,
	})

	_, err = resolvers.ForgotPassword(s.Ctx, model.ForgotPasswordInput{
		Email: email,
	})
	assert.Nil(t, err, "no errors for forgot password")

	verificationRequest, err := db.Mgr.GetVerificationByEmail(email, enum.ForgotPassword.String())
	assert.Nil(t, err)

	assert.Equal(t, verificationRequest.Identifier, enum.ForgotPassword.String())

	cleanData(email)
}

func TestForgotPassword(t *testing.T) {
	s := testSetup()
	defer s.Server.Close()

	if s.TestInfo.ShouldExecuteForSQL {
		t.Run("forgot password for sql dbs should pass", func(t *testing.T) {
			constants.DATABASE_URL = s.TestInfo.SQL
			constants.DATABASE_TYPE = enum.Sqlite.String()
			db.InitDB()
			commonForgotPasswordTest(s, t)
		})
	}

	if s.TestInfo.ShouldExecuteForArango {
		t.Run("forgot password for arangodb should pass", func(t *testing.T) {
			constants.DATABASE_URL = s.TestInfo.ArangoDB
			constants.DATABASE_TYPE = enum.Arangodb.String()
			db.InitDB()
			commonForgotPasswordTest(s, t)
		})
	}

	if s.TestInfo.ShouldExecuteForMongo {
		t.Run("forgot password for mongodb should pass", func(t *testing.T) {
			constants.DATABASE_URL = s.TestInfo.MongoDB
			constants.DATABASE_TYPE = enum.Mongodb.String()
			db.InitDB()
			commonForgotPasswordTest(s, t)
		})
	}
}
