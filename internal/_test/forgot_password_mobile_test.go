package test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func forgotPasswordMobileTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should run forgot password for mobile`, func(t *testing.T) {
		req, ctx := createContext(s)
		phoneNumber := "6240345678"
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			PhoneNumber:     refs.NewStringRef(phoneNumber),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		forgotPasswordRes, err := resolvers.ForgotPasswordResolver(ctx, model.ForgotPasswordInput{
			PhoneNumber: refs.NewStringRef(phoneNumber),
		})
		assert.Nil(t, err, "no errors for forgot password")
		assert.NotNil(t, forgotPasswordRes)
		assert.True(t, *forgotPasswordRes.ShouldShowMobileOtpScreen)
		otpReq, err := db.Provider.GetOTPByPhoneNumber(ctx, phoneNumber)
		assert.Nil(t, err)
		mfaSession := uuid.NewString()
		memorystore.Provider.SetMfaSession(res.User.ID, mfaSession, time.Now().Add(1*time.Minute).Unix())
		cookie := fmt.Sprintf("%s=%s;", constants.MfaCookieName+"_session", mfaSession)
		cookie = strings.TrimSuffix(cookie, ";")
		req.Header.Set("Cookie", cookie)
		// Reset password
		resetPasswordRes, err := resolvers.ResetPasswordResolver(ctx, model.ResetPasswordInput{
			PhoneNumber:     refs.NewStringRef(phoneNumber),
			Otp:             refs.NewStringRef(otpReq.Otp),
			Password:        s.TestInfo.Password + "test",
			ConfirmPassword: s.TestInfo.Password + "test",
		})
		assert.Nil(t, err)
		assert.NotNil(t, resetPasswordRes)
		// Test login
		loginRes, err := resolvers.LoginResolver(ctx, model.LoginInput{
			PhoneNumber: refs.NewStringRef(phoneNumber),
			Password:    s.TestInfo.Password + "test",
		})
		assert.Nil(t, err)
		assert.NotNil(t, loginRes)
	})
}
