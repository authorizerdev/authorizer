package test

import (
	"context"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func updateProfileTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should update the profile with access token only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "update_profile." + s.TestInfo.Email

		resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		fName := "samani"
		_, err := resolvers.UpdateProfileResolver(ctx, model.UpdateProfileInput{
			FamilyName: &fName,
		})
		assert.NotNil(t, err, "unauthorized")

		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		assert.NoError(t, err)
		assert.NotNil(t, verificationRequest)
		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})
		assert.NoError(t, err)
		assert.NotNil(t, verifyRes)
		s.GinContext.Request.Header.Set("Authorization", "Bearer "+*verifyRes.AccessToken)
		ctx = context.WithValue(req.Context(), "GinContextKey", s.GinContext)

		newEmail := "new_" + email
		_, err = resolvers.UpdateProfileResolver(ctx, model.UpdateProfileInput{
			Email: &newEmail,
		})
		s.GinContext.Request.Header.Set("Authorization", "")
		assert.Nil(t, err)
		_, err = resolvers.ProfileResolver(ctx)
		assert.NotNil(t, err, "unauthorized")

		cleanData(newEmail)
		cleanData(email)
	})
}
