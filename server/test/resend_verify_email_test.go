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

func commonResendVerifyEmailTest(s TestSetup, t *testing.T) {
	email := "resend_verify_email." + s.TestInfo.Email
	_, err := resolvers.Signup(s.Ctx, model.SignUpInput{
		Email:           email,
		Password:        s.TestInfo.Password,
		ConfirmPassword: s.TestInfo.Password,
	})

	_, err = resolvers.ResendVerifyEmail(s.Ctx, model.ResendVerifyEmailInput{
		Email:      email,
		Identifier: enum.BasicAuthSignup.String(),
	})

	assert.Nil(t, err)

	cleanData(email)
}

func TestResendVerifyEmail(t *testing.T) {
	s := testSetup()
	defer s.Server.Close()

	if s.TestInfo.ShouldExecuteForSQL {
		t.Run("resend verify email for sql dbs should pass", func(t *testing.T) {
			constants.DATABASE_URL = s.TestInfo.SQL
			constants.DATABASE_TYPE = enum.Sqlite.String()
			db.InitDB()
			commonResendVerifyEmailTest(s, t)
		})
	}

	if s.TestInfo.ShouldExecuteForArango {
		t.Run("resend verify email for arangodb should pass", func(t *testing.T) {
			constants.DATABASE_URL = s.TestInfo.ArangoDB
			constants.DATABASE_TYPE = enum.Arangodb.String()
			db.InitDB()
			commonResendVerifyEmailTest(s, t)
		})
	}

	if s.TestInfo.ShouldExecuteForMongo {
		t.Run("resend verify email for mongodb should pass", func(t *testing.T) {
			constants.DATABASE_URL = s.TestInfo.MongoDB
			constants.DATABASE_TYPE = enum.Mongodb.String()
			db.InitDB()
			commonResendVerifyEmailTest(s, t)
		})
	}
}
