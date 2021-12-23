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

func commonVerifyEmailTest(s TestSetup, t *testing.T) {
	email := "verify_email." + s.TestInfo.Email
	res, err := resolvers.Signup(s.Ctx, model.SignUpInput{
		Email:           email,
		Password:        s.TestInfo.Password,
		ConfirmPassword: s.TestInfo.Password,
	})

	user := *res.User
	assert.Equal(t, email, user.Email)
	assert.Nil(t, res.AccessToken, "access token should be nil")
	verificationRequest, err := db.Mgr.GetVerificationByEmail(email, enum.BasicAuthSignup.String())
	assert.Nil(t, err)
	assert.Equal(t, email, verificationRequest.Email)

	verifyRes, err := resolvers.VerifyEmail(s.Ctx, model.VerifyEmailInput{
		Token: verificationRequest.Token,
	})
	assert.Nil(t, err)
	assert.NotEqual(t, verifyRes.AccessToken, "", "access token should not be empty")

	cleanData(email)
}

func TestVerifyEmail(t *testing.T) {
	s := testSetup()
	defer s.Server.Close()

	if s.TestInfo.ShouldExecuteForSQL {
		t.Run("verify email for sql dbs should pass", func(t *testing.T) {
			constants.DATABASE_URL = s.TestInfo.SQL
			constants.DATABASE_TYPE = enum.Sqlite.String()
			db.InitDB()
			commonVerifyEmailTest(s, t)
		})
	}

	if s.TestInfo.ShouldExecuteForArango {
		t.Run("verify email for arangodb should pass", func(t *testing.T) {
			constants.DATABASE_URL = s.TestInfo.ArangoDB
			constants.DATABASE_TYPE = enum.Arangodb.String()
			db.InitDB()
			commonVerifyEmailTest(s, t)
		})
	}

	if s.TestInfo.ShouldExecuteForMongo {
		t.Run("verify email for mongodb should pass", func(t *testing.T) {
			constants.DATABASE_URL = s.TestInfo.MongoDB
			constants.DATABASE_TYPE = enum.Mongodb.String()
			db.InitDB()
			commonVerifyEmailTest(s, t)
		})
	}
}
