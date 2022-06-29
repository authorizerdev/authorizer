package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func profileTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should get profile only access_token token`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "profile." + s.TestInfo.Email

		resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		_, err := resolvers.ProfileResolver(ctx)
		assert.NotNil(t, err, "unauthorized")

		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(email, constants.VerificationTypeBasicAuthSignup)
		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})
		assert.NoError(t, err)
		assert.NotNil(t, verifyRes.AccessToken)

		s.GinContext.Request.Header.Set("Authorization", "Bearer "+*verifyRes.AccessToken)
		ctx = context.WithValue(req.Context(), "GinContextKey", s.GinContext)
		profileRes, err := resolvers.ProfileResolver(ctx)
		fmt.Println("=> err:", err)
		assert.Nil(t, err)
		assert.NotNil(t, profileRes)
		s.GinContext.Request.Header.Set("Authorization", "")
		fmt.Println("=> res:", profileRes.Email, email)
		newEmail := profileRes.Email
		assert.Equal(t, email, newEmail, "emails should be equal")

		cleanData(email)
	})
}
