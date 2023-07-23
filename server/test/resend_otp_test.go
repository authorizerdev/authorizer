package test

import (
	"context"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func resendOTPTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should resend otp`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "resend_otp." + s.TestInfo.Email
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)

		// Login should fail as email is not verified
		loginRes, err := resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    email,
			Password: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, loginRes)
		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		assert.Nil(t, err)
		assert.Equal(t, email, verificationRequest.Email)
		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})
		assert.Nil(t, err)
		assert.NotEqual(t, verifyRes.AccessToken, "", "access token should not be empty")

		// Using access token update profile
		s.GinContext.Request.Header.Set("Authorization", "Bearer "+refs.StringValue(verifyRes.AccessToken))
		ctx = context.WithValue(req.Context(), "GinContextKey", s.GinContext)
		updateRes, err := resolvers.UpdateProfileResolver(ctx, model.UpdateProfileInput{
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		assert.NoError(t, err)
		assert.NotNil(t, updateRes)
		// Resend otp should return error as no initial opt is being sent
		resendOtpRes, err := resolvers.ResendOTPResolver(ctx, model.ResendOTPRequest{
			Email: refs.NewStringRef(email),
		})
		assert.Error(t, err)
		assert.Nil(t, resendOtpRes)

		// Login should not return error but access token should be empty as otp should have been sent
		loginRes, err = resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    email,
			Password: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, loginRes)
		assert.Nil(t, loginRes.AccessToken)

		// Get otp from db
		otp, err := db.Provider.GetOTPByEmail(ctx, email)
		assert.NoError(t, err)
		assert.NotEmpty(t, otp.Otp)

		// resend otp
		resendOtpRes, err = resolvers.ResendOTPResolver(ctx, model.ResendOTPRequest{
			Email: refs.NewStringRef(email),
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, resendOtpRes.Message)

		newOtp, err := db.Provider.GetOTPByEmail(ctx, email)
		assert.NoError(t, err)
		assert.NotEmpty(t, newOtp.Otp)
		assert.NotEqual(t, otp.Otp, newOtp)

		// Should return error for older otp
		verifyOtpRes, err := resolvers.VerifyOtpResolver(ctx, model.VerifyOTPRequest{
			Email: &email,
			Otp:   otp.Otp,
		})
		assert.Error(t, err)
		assert.Nil(t, verifyOtpRes)
		verifyOtpRes, err = resolvers.VerifyOtpResolver(ctx, model.VerifyOTPRequest{
			Email: &email,
			Otp:   newOtp.Otp,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, verifyOtpRes.AccessToken, "", "access token should not be empty")
		cleanData(email)
	})
}
