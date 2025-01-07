package test

import (
	"context"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func profileTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should get profile only access_token token`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "profile." + s.TestInfo.Email

		resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		_, err := resolvers.ProfileResolver(ctx)
		assert.NotNil(t, err, "unauthorized")

		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		assert.NoError(t, err)
		assert.NotNil(t, verificationRequest)
		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})
		assert.NoError(t, err)
		assert.NotNil(t, verifyRes.AccessToken)

		s.GinContext.Request.Header.Set("Authorization", "Bearer "+*verifyRes.AccessToken)
		ctx = context.WithValue(req.Context(), "GinContextKey", s.GinContext)
		profileRes, err := resolvers.ProfileResolver(ctx)
		assert.Nil(t, err)
		assert.NotNil(t, profileRes)
		s.GinContext.Request.Header.Set("Authorization", "")
		newEmail := profileRes.Email
		assert.Equal(t, email, refs.StringValue(newEmail), "emails should be equal")
		cleanData(email)
	})
}
