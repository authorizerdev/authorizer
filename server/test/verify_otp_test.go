package test

import (
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func verifyOTPTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should verify otp`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "verify_otp." + s.TestInfo.Email
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		otp, err := db.Provider.UpsertOTP(ctx, &models.OTP{
			Otp:       "123456",
			Email:     email,
			ExpiresAt: time.Now().Add(1 * time.Minute).Unix(),
		})
		assert.Equal(t, email, otp.Email)
		assert.Nil(t, res.AccessToken, "access token should be nil")

		verifyRes, err := resolvers.VerifyOtpResolver(ctx, model.VerifyOTPRequest{
			Otp:   "123456",
			Email: email,
		})
		assert.Nil(t, err)
		assert.NotEqual(t, verifyRes.AccessToken, "", "access token should not be empty")
		cleanData(email)
	})
}
